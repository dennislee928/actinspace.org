# Migrations

CLMS/S2GM 相關表由 `000001_create_clms_s2gm_tables.up.sql` 定義。

- **開發環境**：使用 SQLite 時，`main` 的 GORM AutoMigrate 會自動建立對應表，無需手動執行 SQL。
- **PostgreSQL**：可選用 [golang-migrate](https://github.com/golang-migrate/migrate) 或 [goose](https://github.com/pressly/goose) 執行：
  - `migrate -path database/migrations -database "postgres://..." up`
  - 或 `goose -dir database/migrations postgres "..." up`
