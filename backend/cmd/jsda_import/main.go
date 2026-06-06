package main

import (
	"bondscope/updater"
	"flag"
	"fmt"
	"log"
	"time"
)

func main() {
	file := flag.String("file", "", "../data/S260529.csv")
	dateStr := flag.String("date", "", "2026-05-29")
	flag.Parse()

	if *file == "" || *dateStr == "" {
		log.Fatal("使い方: jsda_import -file <path> -date <YYYY-MM-DD>")
	}

	date, err := time.Parse("2006-01-02", *dateStr)
	if err != nil {
		log.Fatalf("日付フォーマットエラー: %v", err)
	}

	fmt.Printf("インポート開始: %s (%s)\n", *file, date.Format("2006-01-02"))

	n, err := updater.UpdateJSDADataFromFile(*file, date)
	if err != nil {
		log.Fatalf("インポート失敗: %v", err)
	}

	fmt.Printf("完了: %d 件を登録しました\n", n)
}
