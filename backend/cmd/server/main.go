package main

import (
    "bondscope/database"
    "bondscope/updater"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
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

	// 3. バッチ更新エンドポイント
	http.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("UPDATE_SECRET")
		if secret != "" && r.URL.Query().Get("secret") != secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		n, err := updater.UpdateJGBData(updater.MOFSURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","records":%d}`, n)
	})

	// JSDA 公社債売買参考統計値の更新
	// ?date=YYYY-MM-DD で日付指定、省略時は当日
	http.HandleFunc("/api/update/jsda", func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv("UPDATE_SECRET")
		if secret != "" && r.URL.Query().Get("secret") != secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		date := time.Now()
		if d := r.URL.Query().Get("date"); d != "" {
			parsed, err := time.Parse("2006-01-02", d)
			if err != nil {
				http.Error(w, "invalid date (use YYYY-MM-DD)", http.StatusBadRequest)
				return
			}
			date = parsed
		}

		n, err := updater.UpdateJSDAData(date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","date":"%s","records":%d}`, date.Format("2006-01-02"), n)
	})

	// GET /api/bonds?type_id=4  銘柄一覧（省略時全件）
	http.HandleFunc("/api/bonds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		var typeID int16
		if s := r.URL.Query().Get("type_id"); s != "" {
			var id int64
			if _, err := fmt.Sscan(s, &id); err != nil {
				http.Error(w, "invalid type_id", http.StatusBadRequest)
				return
			}
			typeID = int16(id)
		}

		bonds, err := database.GetBondList(db, typeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(bonds)
	})

	// 4. フロントエンド配信 (HTML/JavaScriptを直接返す)
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
