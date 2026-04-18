package parser

import (
	"bondscope/models" // go mod init で決めた名前
	"encoding/csv"
	"io"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// NewSJISReader は既存のまま
func NewSJISReader(r io.Reader) *csv.Reader {
	decoder := japanese.ShiftJIS.NewDecoder()
	return csv.NewReader(transform.NewReader(r, decoder))
}

// ParseJgbCSV はCSV全体をパースして YieldRate のスライスを返します
func ParseJgbCSV(r *csv.Reader) ([]models.YieldRate, error) {
	var results []models.YieldRate

	// 1行目: "国債金利情報" 等のタイトル行をスキップ
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	// 2行目: ヘッダー行 ("基準日", "1年", "2年"...) を取得
	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	// ヘッダーから年限(Tenor)の数値を抽出
	// tenors[1] = 1.0 (1年), tenors[2] = 2.0 (2年) ...
	tenors := make(map[int]float64)
	for i, col := range header {
		if col == "基準日" {
			continue
		}
		// "10年" -> "10" -> 10.0
		tStr := strings.TrimSuffix(col, "年")
		tVal, err := strconv.ParseFloat(tStr, 64)
		if err == nil {
			tenors[i] = tVal
		}
	}

	// 3行目以降: 実際のデータ行
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// 日付のパース
		date, err := ParseJapaneseDate(record[0])
		if err != nil {
			continue // 日付が不正な行はスキップ
		}

		// 各列を縦持ちデータに変換
		for i, valStr := range record {
			tenor, ok := tenors[i]
			if !ok {
				continue
			}

			// ハイフン "-" は欠損値としてスキップ
			if valStr == "-" || valStr == "" {
				continue
			}

			val, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				continue
			}

			results = append(results, models.YieldRate{
				Date:  date,
				Tenor: tenor,
				Value: val,
			})
		}
	}

	return results, nil
}
