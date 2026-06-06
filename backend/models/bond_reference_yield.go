package models

import "time"

type BondType struct {
	TypeID   int16  `gorm:"primaryKey"`
	TypeName string `gorm:"not null"`
}

func (BondType) TableName() string { return "bond_types" }

type BondMaster struct {
	BondID       int32      `gorm:"primaryKey;autoIncrement"`
	BondCode     string     `gorm:"size:12;uniqueIndex;not null"`
	BondName     string     `gorm:"size:100;not null"`
	TypeID       int16      `gorm:"not null"`
	IssueDate    *time.Time `gorm:"type:date"`
	MaturityDate *time.Time `gorm:"type:date"`
}

func (BondMaster) TableName() string { return "bond_master" }

// BondPrice は bond_prices テーブルに対応。
// float32(REAL/4バイト) を使い float64(DOUBLE/8バイト) の半分に抑える。
// 複利利回りは有効桁数6桁で十分なため単精度で問題なし。
type BondPrice struct {
	Date          time.Time `gorm:"type:date;primaryKey"`
	BondID        int32     `gorm:"primaryKey"`
	RemainingTerm float32   `gorm:"type:real;not null"`
	CompoundYield *float32  `gorm:"type:real"`
	AvgPrice      *float32  `gorm:"type:real"`
}

func (BondPrice) TableName() string { return "bond_prices" }

// BondPriceWithMeta はAPIレスポンス用（DB直接マッピングではない）
type BondPriceWithMeta struct {
	Date          string  `json:"date"`
	BondCode      string  `json:"bond_code"`
	BondName      string  `json:"bond_name"`
	TypeID        int16   `json:"type_id"`
	RemainingTerm float32 `json:"remaining_tenor"`
	CompoundYield float32 `json:"compound_yield"`
	JGBYield      float64 `json:"jgb_yield"`
	TSpread       float64 `json:"t_spread"`
}
