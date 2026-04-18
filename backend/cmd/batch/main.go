package main

import (
	"bondscope/database"
	"bondscope/parser"
	"fmt"
	"log"
	"os"
)

func main() {
	// 1. DB初期化
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	// 2. CSVファイルを開く (全データの方で試してもOK)
	file, err := os.Open("../data/jgbcm.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 3. パース
	sjisReader := parser.NewSJISReader(file)
	yieldRates, err := parser.ParseJgbCSV(sjisReader)
	if err != nil {
		log.Fatal(err)
	}

	// 4. DB保存 (Upsert)
	fmt.Printf("Saving %d records to database...\n", len(yieldRates))
	err = database.SaveYieldRates(db, yieldRates)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully saved all records!")
}
