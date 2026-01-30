// Package vo：API 回應結構（Response）。
package vo

import "time"

// AssetVO asset 的 API 回應（不暴露內部 path，僅 storage_url）。
// swagger:model AssetVO
type AssetVO struct {
	ID         uint      `json:"id"`
	JobID      uint      `json:"job_id"`
	Filename   string    `json:"filename"`
	Mime       string    `json:"mime,omitempty"`
	CRS        string    `json:"crs,omitempty"`
	Resolution string    `json:"resolution,omitempty"`
	StorageURL string    `json:"storage_url"`
	SizeBytes  *int64    `json:"size_bytes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
