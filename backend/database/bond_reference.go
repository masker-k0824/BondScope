package database

import (
	"bondscope/models"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const bondPriceBatchSize = 5000

// UpsertBondMasters は銘柄マスタをupsertし、bond_code→bond_idのマップを返す。
// 一括インポート時はこの関数を先に呼び、idMapをBondPriceにセットしてからUpsertBondPricesへ。
func UpsertBondMasters(db *gorm.DB, masters []models.BondMaster) (map[string]int32, error) {
	if len(masters) == 0 {
		return map[string]int32{}, nil
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bond_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"bond_name", "type_id", "maturity_date"}),
	}).CreateInBatches(masters, 1000).Error
	if err != nil {
		return nil, fmt.Errorf("bond_master upsert failed: %w", err)
	}

	codes := make([]string, len(masters))
	for i, m := range masters {
		codes[i] = m.BondCode
	}

	var results []models.BondMaster
	if err := db.Select("bond_id, bond_code").Where("bond_code IN ?", codes).Find(&results).Error; err != nil {
		return nil, fmt.Errorf("bond_id fetch failed: %w", err)
	}

	idMap := make(map[string]int32, len(results))
	for _, r := range results {
		idMap[r.BondCode] = r.BondID
	}
	return idMap, nil
}

// UpsertBondPrices は価格・利回りデータをバッチupsertする。
// バッチサイズ5000はPgBouncerのmax_prepared_statements制限を考慮した値。
func UpsertBondPrices(db *gorm.DB, prices []models.BondPrice) error {
	if len(prices) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}, {Name: "bond_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"remaining_term", "compound_yield", "avg_price"}),
	}).CreateInBatches(prices, bondPriceBatchSize).Error
}

// GetBondsForTSpread はTスプレッド計算用データを返す。
// bond_codesが空の場合は全銘柄を対象とする。
func GetBondsForTSpread(db *gorm.DB, bondCodes []string, start, end string) ([]models.BondPriceWithMeta, error) {
	query := db.Table("bond_prices bp").
		Select(`bp.date, bm.bond_code, bm.bond_name, bm.type_id,
		        bp.remaining_term, bp.compound_yield`).
		Joins("JOIN bond_master bm ON bm.bond_id = bp.bond_id").
		Where("bp.compound_yield IS NOT NULL").
		Where("bp.date BETWEEN ? AND ?", start, end).
		Order("bp.date ASC, bm.bond_code ASC")

	if len(bondCodes) > 0 {
		query = query.Where("bm.bond_code IN ?", bondCodes)
	}

	var rows []struct {
		Date          time.Time
		BondCode      string
		BondName      string
		TypeID        int16
		RemainingTerm float32
		CompoundYield float32
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]models.BondPriceWithMeta, len(rows))
	for i, r := range rows {
		result[i] = models.BondPriceWithMeta{
			Date:          r.Date.Format("2006-01-02"),
			BondCode:      r.BondCode,
			BondName:      r.BondName,
			TypeID:        r.TypeID,
			RemainingTerm: r.RemainingTerm,
			CompoundYield: r.CompoundYield,
		}
	}
	return result, nil
}

// GetBondList は銘柄一覧を返す（フロント銘柄選択UI用）。
// typeID=0 は全種別。
func GetBondList(db *gorm.DB, typeID int16) ([]models.BondMaster, error) {
	var bonds []models.BondMaster
	q := db.Order("bond_code ASC")
	if typeID != 0 {
		q = q.Where("type_id = ?", typeID)
	}
	err := q.Find(&bonds).Error
	return bonds, err
}

// InterpolateJGB はJGB利回り曲線を線形補間してtarget年限の金利を返す。
// tenors と yields は昇順で揃っていること。
// target がレンジ外の場合は端点の値を返す（外挿なし）。
func InterpolateJGB(tenors []float64, yields []float64, target float64) float64 {
	if len(tenors) == 0 {
		return 0
	}
	if target <= tenors[0] {
		return yields[0]
	}
	if target >= tenors[len(tenors)-1] {
		return yields[len(tenors)-1]
	}

	// target を挟む区間を二分探索
	i := sort.SearchFloat64s(tenors, target)
	// i は tenors[i] >= target となる最小インデックス
	if tenors[i] == target {
		return yields[i]
	}
	// 線形補間: tenors[i-1] ≤ target < tenors[i]
	lo, hi := tenors[i-1], tenors[i]
	t := (target - lo) / (hi - lo)
	return yields[i-1] + t*(yields[i]-yields[i-1])
}

// DownsampleToMonthly は指定日より古いデータを月末の1点に間引く。
// SQLite の strftime を使用（PostgreSQL の DATE_TRUNC は不使用）。
func DownsampleToMonthly(db *gorm.DB, olderThan time.Time) (int64, error) {
	result := db.Exec(`
		DELETE FROM bond_prices
		WHERE date < ?
		  AND rowid NOT IN (
		      SELECT MAX(rowid)
		      FROM bond_prices
		      WHERE date < ?
		      GROUP BY strftime('%Y-%m', date), bond_id
		  )
	`, olderThan, olderThan)
	return result.RowsAffected, result.Error
}
