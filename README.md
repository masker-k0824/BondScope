# BondScope

公社債の利回りと T スプレッドをブラウザで手軽に確認するための個人ツール。
JSDA データの著作権制約を考慮した設計が特徴です。

**デモ（Render）:** `https://<your-render-url>` 

## 機能

| 機能 | 環境 | データソース |
|---|---|---|
| JGB イールドカーブ表示 | ローカル・本番 | 財務省（MOF） |
| T スプレッド表示 | **ローカルのみ** | JSDA 公社債売買参考統計値 |

> **T スプレッドがローカル専用な理由**
> JSDA の公社債売買参考統計値は著作権により再配布が制限されています。
> そのため取得データをクラウド DB に保存せず、ローカルの SQLite に蓄積する設計にしています。

## 技術スタック

| レイヤー | 技術 |
|---|---|
| バックエンド | Go（標準 `net/http`、GORM） |
| フロントエンド | Vanilla HTML/CSS/JS + Chart.js |
| クラウド DB | Supabase（PostgreSQL）— JGB 金利データ |
| ローカル DB | SQLite（`data/jsda.db`）— JSDA データ |
| デプロイ | Render |

## アーキテクチャ

```
MOF CSV（国債金利）
  └── /api/update  →  Supabase（yield_rates）
        └── /api/yield  →  フロントエンド（イールドカーブ）

JSDA CSV（公社債売買参考統計値）
  └── /api/update/jsda  →  ローカル SQLite（bond_master / bond_prices）
        └── /api/tspread  →  フロントエンド（Tスプレッド）
              ↑ JGB 補間利回りと合算して T スプレッドをサーバー側で計算
```

T スプレッドは「社債の複利利回り − 残存年限に対応する JGB 補間利回り」を線形補間でサーバー側計算しています。

## ローカルで動かす（自分用メモ）

> **前提:** JGB 金利の保存先として Supabase を使用しているため、接続文字列が必要です。

```powershell
# 1. 環境変数を設定（.env は .gitignore 済みなので git には乗らない）
$env:DATABASE_URL = "postgresql://..."   # Supabase の接続文字列

# 2. サーバー起動
cd backend
go run ./cmd/server

# 3. JSDA データを取得してローカル DB に保存
# ブラウザ or curl で以下を叩く（営業日のみ有効）
curl http://localhost:8080/api/update/jsda
curl "http://localhost:8080/api/update/jsda?date=2026-06-13"
# → {"status":"ok","date":"2026-06-13","records":1234}

# 過去分をまとめて取得したい場合は日付を変えて繰り返し実行
```

### ローカル CSV から手動インポートする場合

```powershell
go run ./cmd/jsda_import -file data/S260529.csv -date 2026-05-29
```

## API エンドポイント

| パス | 説明 | 環境 |
|---|---|---|
| `GET /api/yield?start=&end=` | JGB 金利取得 | 共通 |
| `GET /api/update` | JGB 金利更新（MOF から取得） | 共通 |
| `GET /api/update/jsda?date=` | JSDA データ更新（ローカル SQLite へ保存） | ローカルのみ |
| `GET /api/bonds?type_id=` | 銘柄一覧 | ローカルのみ |
| `GET /api/tspread?codes=&start=&end=` | T スプレッド取得 | ローカルのみ |
