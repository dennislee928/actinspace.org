package model

import "time"

// DatasetRef CLMS 用資料集參考
type DatasetRef struct {
	ID                uint      `gorm:"primaryKey"`
	DatasetUID        string    `gorm:"column:dataset_uid;type:text;not null"`
	DownloadInfoID     string    `gorm:"column:download_info_id;type:text;not null"`
	Title             string    `gorm:"type:text"`
	ProductFamily     string    `gorm:"column:product_family;type:text"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (DatasetRef) TableName() string { return "dataset_ref" }
