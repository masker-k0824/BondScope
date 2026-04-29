package updater

import (
	"bondscope/database"
	"bondscope/parser"
	"encoding/csv"
	"fmt"
	"net/http"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const MOFSURL = "https://www.mof.go.jp/jgbs/reference/interest_rate/jgbcm.csv"

func UpdateJGBData(csvURL string) (int, error) {
	db, err := database.InitDB()
	if err != nil {
		return 0, fmt.Errorf("DB initialization failed: %w", err)
	}

	fmt.Printf("Fetching CSV from: %s\n", csvURL)
	resp, err := http.Get(csvURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	sjisReader := transform.NewReader(resp.Body, japanese.ShiftJIS.NewDecoder())
	csvReader := csv.NewReader(sjisReader)
	yieldRates, err := parser.ParseJgbCSV(csvReader)
	if err != nil {
		return 0, fmt.Errorf("parsing failed: %w", err)
	}

	fmt.Printf("Saving %d records to database...\n", len(yieldRates))
	if err := database.SaveYieldRates(db, yieldRates); err != nil {
		return 0, fmt.Errorf("DB save failed: %w", err)
	}

	return len(yieldRates), nil
}
