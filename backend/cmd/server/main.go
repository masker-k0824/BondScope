package main

import (
	"bondscope/database"
	"bondscope/updater"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

func main() {
	// 1. Supabase接続（JGB金利用）
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	// 2. ローカルSQLite接続（JSDAデータ用・著作権上ローカル保存）
	// Render環境では無効化（RENDER=true が自動セットされる）
	var localDB *gorm.DB
	if os.Getenv("RENDER") != "" {
		log.Println("INFO: Render環境を検出 - JSDA機能を無効化")
	} else {
		localDB, err = database.InitLocalDB(updater.LocalDBPath)
		if err != nil {
			log.Printf("WARNING: ローカルDB初期化失敗（Tスプレッド機能無効）: %v", err)
		}
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

	// JSDA 公社債売買参考統計値の更新（ローカルSQLiteに保存）
	// ?date=YYYY-MM-DD で日付指定、省略時は当日
	http.HandleFunc("/api/update/jsda", func(w http.ResponseWriter, r *http.Request) {
		if localDB == nil {
			http.Error(w, "JSDA機能はローカル環境でのみ利用可能です", http.StatusServiceUnavailable)
			return
		}

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

		n, err := updater.UpdateJSDAData(localDB, date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","date":"%s","records":%d}`, date.Format("2006-01-02"), n)
	})

	// GET /api/bonds?type_id=4  銘柄一覧（ローカル専用）
	http.HandleFunc("/api/bonds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		if localDB == nil {
			http.Error(w, "JSDA機能はローカル環境でのみ利用可能です", http.StatusServiceUnavailable)
			return
		}

		var typeID int16
		if s := r.URL.Query().Get("type_id"); s != "" {
			var id int64
			if _, err := fmt.Sscan(s, &id); err != nil {
				http.Error(w, "invalid type_id", http.StatusBadRequest)
				return
			}
			typeID = int16(id)
		}

		bonds, err := database.GetBondList(localDB, typeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(bonds)
	})

	// GET /api/tspread?codes=040001234,040002345&start=2025-01-01&end=2026-05-29
	// Tスプレッド（複利利回り - JGB補間利回り）を返す。単位は%ポイント。ローカル専用。
	http.HandleFunc("/api/tspread", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		if localDB == nil {
			http.Error(w, "JSDA機能はローカル環境でのみ利用可能です", http.StatusServiceUnavailable)
			return
		}

		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		if start == "" || end == "" {
			http.Error(w, "start and end are required", http.StatusBadRequest)
			return
		}

		var codes []string
		if s := r.URL.Query().Get("codes"); s != "" {
			codes = strings.Split(s, ",")
		}

		bondPrices, err := database.GetBondsForTSpread(localDB, codes, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		yieldRates, err := database.GetYieldRates(db, start, end)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 日付ごとに tenor→yield のスライスを構築（GetYieldRatesはtenor昇順で返す）
		type yp struct{ tenor, yield float64 }
		yieldByDate := make(map[string][]yp, len(yieldRates))
		for _, yr := range yieldRates {
			d := yr.Date.Format("2006-01-02")
			yieldByDate[d] = append(yieldByDate[d], yp{yr.Tenor, yr.Value})
		}

		for i, bp := range bondPrices {
			pts := yieldByDate[bp.Date]
			if len(pts) == 0 {
				continue
			}
			tenors := make([]float64, len(pts))
			yields := make([]float64, len(pts))
			for j, p := range pts {
				tenors[j] = p.tenor
				yields[j] = p.yield
			}
			jgb := database.InterpolateJGB(tenors, yields, float64(bp.RemainingTerm))
			bondPrices[i].JGBYield = jgb
			bondPrices[i].TSpread = float64(bp.CompoundYield) - jgb
		}

		json.NewEncoder(w).Encode(bondPrices)
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
