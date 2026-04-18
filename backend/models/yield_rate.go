package models

import "time"

type YieldRate struct {
	ID        uint      `gorm:"primaryKey"`
	Date      time.Time `gorm:"index:idx_date_tenor,unique"` // 日付と年限でユニーク制約
	Tenor     float64   `gorm:"index:idx_date_tenor,unique"` // 1, 2, 5, 10, 40 などの年数
	Value     float64   // 金利 (%)
}