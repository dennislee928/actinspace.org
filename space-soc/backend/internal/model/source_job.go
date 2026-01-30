package model

import "time"

// SourceJob 下單/任務（CLMS 或 S2GM）
// 與 DB schema 對應，不混用 DTO/VO。
// CLMS 任務需填 DatasetUID、DownloadInfoID、OutputFormat、OutputGCS 供 Scheduler/Fetcher 使用。
type SourceJob struct {
	ID              uint       `gorm:"primaryKey"`
	Source          string     `gorm:"type:text;not null;index"` // "clms" | "s2gm"
	AoiGeom         string     `gorm:"column:aoi_geom;type:text"`
	TimeStart       *time.Time `gorm:"column:time_start;type:datetime"`
	TimeEnd         *time.Time `gorm:"column:time_end;type:datetime"`
	Status          string     `gorm:"type:text;not null;index;default:pending"`
	ExternalTaskID  string     `gorm:"column:external_task_id;type:text;index"`
	// CLMS 專用（Scheduler 依此建 CLMSFetcherParams）
	DatasetUID       string `gorm:"column:dataset_uid;type:text"`
	DownloadInfoID   string `gorm:"column:download_info_id;type:text"`
	OutputFormat     string `gorm:"column:output_format;type:text"` // 如 "Geotiff"
	OutputGCS        string `gorm:"column:output_gcs;type:text"`    // 如 "EPSG:4326"
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Assets []Asset `gorm:"foreignKey:JobID"`
}

func (SourceJob) TableName() string { return "source_job" }
