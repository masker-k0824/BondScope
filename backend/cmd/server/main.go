package main

import (
	"bondscope/database"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// 1. DB初期化（環境変数 DATABASE_URL を使用）
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	// 2. APIエンドポイント (JSONを返す)
	http.HandleFunc("/api/yield", func(w http.ResponseWriter, r *http.Request) {
		// CORS対応（フロントエンドをRenderの別サービス等にする場合にも備えて）
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		rates, err := database.GetYieldRates(db, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(rates)
	})

	// 3. フロントエンド配信 (HTML/JavaScriptを直接返す)
	// Renderの Root Directory が "backend" なので、
	// index.html が backend 直下にある前提です。
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// 4. ポート設定（Render対応の動的ポート取得）
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル実行時のデフォルト
	}

	fmt.Printf("BondScope Server starting at port %s ...\n", port)

	// 5. サーバー起動
	// 文字列 ":" と port を結合して ":8080" や ":10000" にします
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
