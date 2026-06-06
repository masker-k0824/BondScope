package database

import (
	"bondscope/models"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitLocalDB はローカルSQLiteファイルを開き、JSDAデータ用テーブルを作成する。
// path が存在しない場合は自動作成する。
func InitLocalDB(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("local DB dir: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("local DB open: %w", err)
	}

	if err := db.AutoMigrate(&models.BondMaster{}, &models.BondPrice{}); err != nil {
		return nil, fmt.Errorf("local DB migrate: %w", err)
	}

	return db, nil
}
