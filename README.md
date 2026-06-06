# BondScope

日本国内の公社債データを可視化するツールです。

## 機能概要

| 機能 | 環境 | データソース |
|---|---|---|
| JGB金利カーブ表示 | ローカル・本番 | 財務省（MOF） |
| Tスプレッド表示 | **ローカルのみ** | JSDA 公社債売買参考統計値 |

### Tスプレッドについて

JSDAの公社債売買参考統計値は著作権により再配布が制限されているため、**ローカル環境でのみ**Tスプレッドの閲覧が可能です。

ローカルで `data/jsda.db`（SQLite）にデータを保存し、JGB金利との差分をリアルタイムで計算・表示します。本番環境（Render）では `/api/bonds` と `/api/tspread` エンドポイントは 503 を返します。

## ローカルセットアップ

```bash
# 1. 環境変数を設定
export DATABASE_URL="<Supabase接続文字列>"

# 2. サーバー起動
cd backend
go run ./cmd/server

# 3. JSDAデータのインポート（初回・手動）
go run ./cmd/jsda_import -file data/S260529.csv -date 2026-05-29
```

## APIエンドポイント

| メソッド | パス | 説明 | 環境 |
|---|---|---|---|
| GET | `/api/yield?start=&end=` | JGB金利取得 | 共通 |
| POST/GET | `/api/update?secret=` | JGB金利更新（MOF） | 共通 |
| POST/GET | `/api/update/jsda?date=&secret=` | JSDA売買参考統計値更新 | ローカルのみ |
| GET | `/api/bonds?type_id=` | 銘柄一覧 | ローカルのみ |
| GET | `/api/tspread?codes=&start=&end=` | Tスプレッド取得 | ローカルのみ |

## データ管理

- `yield_rates`（JGB金利）: Supabase / GORM AutoMigrate
- `bond_master` / `bond_prices`（JSDA）: ローカル SQLite（`data/jsda.db`）
