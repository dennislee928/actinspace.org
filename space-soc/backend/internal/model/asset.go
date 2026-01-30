package model

import "time"

// Asset 產出檔案（對應 DB，bands 存 JSON 字串）
type Asset struct {
	ID          uint      `gorm:"primaryKey"`
	JobID       uint      `gorm:"not null;index"`
	Filename    string    `gorm:"type:text;not null"`
	Mime        string    `gorm:"type:text"`
	CRS         string    `gorm:"column:crs;type:text"`
	Resolution  string    `gorm:"type:text"`
	Bands       string    `gorm:"type:text"` // JSON string (jsonb in PG)
	StorageURL  string    `gorm:"column:storage_url;type:text;not null"`
	Checksum    string    `gorm:"type:text"`
	SizeBytes   *int64    `gorm:"column:size_bytes"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Asset) TableName() string { return "asset" }
