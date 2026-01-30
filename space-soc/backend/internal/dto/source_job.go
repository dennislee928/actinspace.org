// Package dto：API 請求結構（Request），含 binding/json tag。
package dto

// CreateSourceJobRequest 建立 source_job 的請求體。
// source 必填（clms|s2gm）；CLMS 任務需填 dataset_uid、download_info_id。
type CreateSourceJobRequest struct {
	Source          string  `json:"source" binding:"required,oneof=clms s2gm"`
	AoiGeom         string  `json:"aoi_geom"`
	TimeStart       *string `json:"time_start"` // ISO8601，如 "2006-01-02T15:04:05Z07:00"
	TimeEnd         *string `json:"time_end"`
	DatasetUID      string  `json:"dataset_uid"`      // CLMS 用
	DownloadInfoID  string  `json:"download_info_id"` // CLMS 用
	OutputFormat    string  `json:"output_format"`    // 如 "Geotiff"，預設 Geotiff
	OutputGCS       string  `json:"output_gcs"`       // 如 "EPSG:4326"，預設 EPSG:4326
}
