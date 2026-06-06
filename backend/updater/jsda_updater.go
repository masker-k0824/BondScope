package updater

import (
	"bondscope/database"
	"bondscope/models"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
)

// col2ToTypeID maps JSDA 銘柄種別コード → bond_types.type_id
// 変動利付・短期は除外（複利=999.999 でフィルタされる）
var col2ToTypeID = map[int]int16{
	2: 1, 3: 1, 5: 1, // 国債（中期・分離元本・物価連動）
	10: 3,             // 地方債
	20: 2, 22: 6,      // 政府保証債 / 財投機関債
	31: 5,             // 金融債（利付）
	40: 4, 43: 4, 44: 4, // 社債・特定社債・円建外債
}

// JSDADataURL は指定日の JSDA 公社債売買参考統計値 CSV の URL を返す。
// ファイル名パターン: S{YYMMDD}.csv（例: S260529.csv = 2026-05-29）
func JSDADataURL(date time.Time) string {
	return fmt.Sprintf("https://www.jsda.or.jp/shiryoshitsu/toukei/baisan/data/S%s.csv",
		date.Format("060102"))
}

// UpdateJSDADataFromFile はローカルファイルから JSDA データをインポートする（テスト・バックフィル用）。
func UpdateJSDADataFromFile(path string, date time.Time) (int, error) {
	db, err := database.InitDB()
	if err != nil {
		return 0, fmt.Errorf("DB init: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return upsertJSDA(db, f, date)
}

// UpdateJSDAData は指定日の JSDA CSV を取得し bond_master + bond_prices を upsert する。
func UpdateJSDAData(date time.Time) (int, error) {
	db, err := database.InitDB()
	if err != nil {
		return 0, fmt.Errorf("DB init: %w", err)
	}

	url := JSDADataURL(date)
	fmt.Printf("Fetching JSDA CSV: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("JSDA fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("JSDA HTTP %s", resp.Status)
	}

	return upsertJSDA(db, resp.Body, date)
}

func upsertJSDA(db *gorm.DB, r io.Reader, date time.Time) (int, error) {
	// 時刻部分を切り捨てて純粋な日付にする
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	masters, priceMap, err := parseJSDACSV(r, date)
	if err != nil {
		return 0, err
	}

	idMap, err := database.UpsertBondMasters(db, masters)
	if err != nil {
		return 0, err
	}

	prices := make([]models.BondPrice, 0, len(priceMap))
	for code, p := range priceMap {
		id, ok := idMap[code]
		if !ok {
			continue
		}
		p.BondID = id
		prices = append(prices, p)
	}

	if err := database.UpsertBondPrices(db, prices); err != nil {
		return 0, err
	}
	return len(prices), nil
}

// parseJSDACSV は Shift-JIS の JSDA CSV を読んで bond_master / bond_prices 用データを返す。
// 複利 = 999.999 の行（変動利付・短期）はスキップする。
func parseJSDACSV(r io.Reader, date time.Time) ([]models.BondMaster, map[string]models.BondPrice, error) {
	dec := transform.NewReader(r, japanese.ShiftJIS.NewDecoder())
	cr := csv.NewReader(dec)
	cr.FieldsPerRecord = -1

	masters := make([]models.BondMaster, 0, 5000)
	prices := make(map[string]models.BondPrice, 5000)

	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) < 15 {
			continue
		}

		// col7 (index 6): 平均値複利 — 999.999 = データなし → スキップ
		cy, err := strconv.ParseFloat(row[6], 32)
		if err != nil || cy >= 999.0 {
			continue
		}

		// col2 (index 1): 銘柄種別コード → type_id に変換
		typeCode, err := strconv.Atoi(row[1])
		if err != nil {
			continue
		}
		typeID, ok := col2ToTypeID[typeCode]
		if !ok {
			continue
		}

		code := row[2] // col3: 銘柄コード
		name := row[3] // col4: 銘柄名

		// col5 (index 4): 償還期日 YYYYMMDD
		matDate, err := time.Parse("20060102", row[4])
		if err != nil {
			continue
		}

		// 残存年限 (Actual/365)
		remainingTerm := float32(matDate.Sub(date).Hours() / 24.0 / 365.0)
		if remainingTerm <= 0 {
			continue
		}

		// bond_master: 初出の銘柄コードのみ追加
		if _, exists := prices[code]; !exists {
			mat := matDate // ループ変数のコピーを取ってポインタを安全に取る
			masters = append(masters, models.BondMaster{
				BondCode:     code,
				BondName:     name,
				TypeID:       typeID,
				MaturityDate: &mat,
			})
		}

		// float32 ポインタ（欠損は nil）
		cyF := float32(cy)

		var avgPrice *float32
		if p, err := strconv.ParseFloat(row[7], 32); err == nil && p < 999.0 {
			pf := float32(p)
			avgPrice = &pf
		}

		prices[code] = models.BondPrice{
			Date:          date,
			RemainingTerm: remainingTerm,
			CompoundYield: &cyF,
			AvgPrice:      avgPrice,
		}
	}

	return masters, prices, nil
}
