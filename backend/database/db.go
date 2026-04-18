package database

import (
	"bondscope/models"
	"fmt"
	"os"

	"gorm.io/driver/postgres" // PostgreSQL用のドライバーに変更
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InitDB() (*gorm.DB, error) {
	// 1. まずは環境変数 "DATABASE_URL" を見に行く
	dsn := os.Getenv("DATABASE_URL")

	// 2. もし環境変数が空だったら（設定し忘れや、ローカルでの予備）
	if dsn == "" {
		// ここでエラーを出すか、一時的なテスト用URIを出す
		return nil, fmt.Errorf("DATABASE_URL が設定されていません")
	}

	// 接続先をコンソールに表示（パスワードは隠れませんので、公開時は注意）
	fmt.Println("🐘 Connecting to PostgreSQL (Supabase)...")

	// 2. Postgresに接続（sqlite.Open ではなく postgres.Open を使う）
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("DB接続失敗: %v", err)
	}

	// 3. テーブルの自動作成
	// PostgreSQL側に yield_rates テーブルが自動的に作成されます
	err = db.AutoMigrate(&models.YieldRate{})
	if err != nil {
		return nil, fmt.Errorf("マイグレーション失敗: %v", err)
	}

	return db, nil
}

// SaveYieldRates は PostgreSQL の UPSERT 構文にも対応しています
func SaveYieldRates(db *gorm.DB, rates []models.YieldRate) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}, {Name: "tenor"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).CreateInBatches(rates, 2000).Error
}

// GetYieldRates もそのままの SQL ロジックで PostgreSQL でも動作します
func GetYieldRates(db *gorm.DB, start, end string) ([]models.YieldRate, error) {
	var rates []models.YieldRate

	fullEnd := end + " 23:59:59"

	err := db.Where("date BETWEEN ? AND ?", start, fullEnd).
		Order("date ASC, tenor ASC").
		Find(&rates).Error

	return rates, err
}
