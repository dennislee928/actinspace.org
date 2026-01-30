# Space-SOC Backend

此服務將提供：

- 日誌與事件的 HTTP ingestion 端點
- 指令事件與 incident 的資料庫 schema/API
- ACRI-ST / S2GM / CLMS 集中式 schema 與 API（source_job、asset、dataset_ref）
- （未來）簡單關聯與規則型偵測邏輯

## Environment variables

複製 [.env.example](.env.example) 為 `.env` 並依環境調整。重要變數：

| 變數 | 必填 | 說明 |
|------|------|------|
| `DATABASE_URL` | 否 | PostgreSQL 連線字串。留空則使用 `SUPABASE_DATABASE_URL`（Supabase）或 SQLite（開發） |
| `SUPABASE_DATABASE_URL` | 否 | Supabase 提供的 Postgres 連線字串（留空 DATABASE_URL 時使用） |
| `PORT` | 否 | 預設 8080 |
| `ACRI_ST_FOLDER_URL` | 否 | ACRI-ST 資源連結（GET /api/v1/resources） |
| `CLMS_CLIENT_ID`, `CLMS_PRIVATE_KEY`, `CLMS_TOKEN_URI`, `CLMS_USER_ID` | CLMS 全自動時 | CLMS JWT（Scheduler/Fetcher） |
| `CLMS_BEARER_TOKEN` | 否 | CLMS proxy 手動 token（選用） |
| `CLMS_SCHEDULER_INTERVAL` | 否 | 例：2m |
| `S2GM_WATCH_DIR` | S2GM Watcher 時 | 監控目錄 |
| `S2GM_WATCH_INTERVAL` | 否 | 例：1m |
| `S3_*` | CLMS/S2GM 產出時 | Object Storage：Cloudflare R2 或 S3-compatible（MinIO/S3）。R2 使用 S3 API 端點 |
| `EPO_OPS_*` | 否 | 專利 API（選用） |
| `ACTINSPACE_LICENSE_KEY` | 否 | 企業功能（選用） |


