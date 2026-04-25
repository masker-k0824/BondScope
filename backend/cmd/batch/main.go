package main

import (
	"bondscope/database"
	"bondscope/parser"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// 他のパッケージ（server等）から呼び出せるように関数化
func UpdateJGBData(csvURL string) error {
	// 1. DB初期化
	db, err := database.InitDB()
	if err != nil {
		return fmt.Errorf("DB initialization failed: %w", err)
	}

	// 2. 財務省のURLからCSVを直接取得
	fmt.Printf("Fetching CSV from: %s\n", csvURL)
	resp, err := http.Get(csvURL)
	if err != nil {
		return fmt.Errorf("failed to fetch CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// 3. パース (Shift-JIS 変換を噛ませる)
	// parser.ParseJgbCSV が io.Reader を受け取る前提
	sjisReader := transform.NewReader(resp.Body, japanese.ShiftJIS.NewDecoder())
	csvReader := csv.NewReader(sjisReader)
	yieldRates, err := parser.ParseJgbCSV(csvReader)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	// 4. DB保存 (Upsert)
	fmt.Printf("Saving %d records to database...\n", len(yieldRates))
	err = database.SaveYieldRates(db, yieldRates)
	if err != nil {
		return fmt.Errorf("DB save failed: %w", err)
	}

	return nil
}

func main() {
	// 財務省の最新CSVのURL
	const targetURL = "https://www.mof.go.jp/jgbs/reference/interest_rate/jgbcm.csv"

	err := UpdateJGBData(targetURL)
	if err != nil {
		log.Fatalf("Error updating JGB data: %v", err)
	}

	fmt.Println("Successfully updated all records from MOF!")
}
