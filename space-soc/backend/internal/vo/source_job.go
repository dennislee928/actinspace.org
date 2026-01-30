// Package vo：API 回應結構（Response），含 json tag 與 Swagger 註解。
package vo

import "time"

// SourceJobVO source_job 的 API 回應。
// swagger:model SourceJobVO
type SourceJobVO struct {
	ID              uint       `json:"id"`
	Source          string     `json:"source"`
	AoiGeom         string     `json:"aoi_geom,omitempty"`
	TimeStart       *time.Time `json:"time_start,omitempty"`
	TimeEnd         *time.Time `json:"time_end,omitempty"`
	Status          string     `json:"status"`
	ExternalTaskID  string     `json:"external_task_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
