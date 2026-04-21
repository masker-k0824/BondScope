package main

import (
    "bondscope/database"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time" // 1. timeパッケージを追加
)

func main() {
	// 1. DB初期化（環境変数 DATABASE_URL を使用）
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	// 2. APIエンドポイント (JSONを返す)
	http.HandleFunc("/api/yield", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Content-Type", "application/json")

        start := r.URL.Query().Get("start")
        end := r.URL.Query().Get("end")

        // 2. 日付が指定されていない場合の処理
        now := time.Now()
        if end == "" {
            // 今日の日付を "YYYY-MM-DD" 形式にする
            end = now.Format("2006-01-02")
        }
        if start == "" {
            // startがなければ、1ヶ月前の日付をデフォルトにする（例）
            start = now.AddDate(0, -1, 0).Format("2006-01-02")
        }

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
		http.ServeFile(w, r, "cmd/server/index.html")
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
