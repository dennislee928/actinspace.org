// Package scheduler：週期掃描 pending CLMS job，呼叫 CLMS Fetcher。
// MVP 採同進程內 cron + in-memory，無額外 Job Queue 依賴。

package scheduler

import (
	"context"
	"log"
	"os"
	"time"

	"actinspace.org/space-soc/backend/internal/integrations"
	"actinspace.org/space-soc/backend/internal/model"
	"actinspace.org/space-soc/backend/internal/storage"
	"gorm.io/gorm"
)

const (
	defaultCLMSInterval = 2 * time.Minute
)

// RunCLMSScheduler 在背景週期掃描 source_job 中 status=pending、source=clms 的任務，
// 依 job 的 dataset_uid、download_info_id 等建 CLMSFetcherParams 並執行 RunCLMSFetcher。
// 若 auth 或 objStore 為 nil，僅記錄並跳過（不中斷迴圈）。
func RunCLMSScheduler(ctx context.Context, db *gorm.DB, auth *integrations.CLMSAuth, objStore storage.ObjectStorage) {
	if auth == nil || !auth.Valid() {
		log.Println("[scheduler] CLMS auth not configured, CLMS scheduler disabled")
		return
	}
	if objStore == nil {
		log.Println("[scheduler] object storage not configured, CLMS scheduler disabled")
		return
	}

	interval := defaultCLMSInterval
	if v := os.Getenv("CLMS_SCHEDULER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCLMSPoll(ctx, db, auth, objStore)
		}
	}
}

func runCLMSPoll(ctx context.Context, db *gorm.DB, auth *integrations.CLMSAuth, objStore storage.ObjectStorage) {
	var jobs []model.SourceJob
	if err := db.WithContext(ctx).Where("source = ? AND status = ?", "clms", "pending").Find(&jobs).Error; err != nil {
		log.Printf("[scheduler] list pending CLMS jobs: %v", err)
		return
	}
	for i := range jobs {
		job := &jobs[i]
		if job.DatasetUID == "" || job.DownloadInfoID == "" {
			log.Printf("[scheduler] job %d missing dataset_uid or download_info_id, skipping", job.ID)
			continue
		}
		outputFormat := job.OutputFormat
		if outputFormat == "" {
			outputFormat = "Geotiff"
		}
		outputGCS := job.OutputGCS
		if outputGCS == "" {
			outputGCS = "EPSG:4326"
		}
		params := integrations.CLMSFetcherParamsFromJob(job.DatasetUID, job.DownloadInfoID, outputFormat, outputGCS, job)
		if err := integrations.RunCLMSFetcher(ctx, db, auth, objStore, job.ID, params); err != nil {
			log.Printf("[scheduler] CLMS fetcher job %d: %v", job.ID, err)
		}
	}
}
