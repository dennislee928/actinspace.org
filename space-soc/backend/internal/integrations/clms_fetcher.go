// Package integrations：CLMS 全自動 Fetcher（search → datarequest_post → 輪詢 → 下載 → Object Storage + DB）。

package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"actinspace.org/space-soc/backend/internal/model"
	"actinspace.org/space-soc/backend/internal/storage"
	"gorm.io/gorm"
)

const (
	clmsPollInterval   = 30 * time.Second
	clmsPollMaxWait    = 2 * time.Hour
	clmsStatusFinished = "Finished_ok"
	clmsStatusError    = "Finished_with_errors"
)

// CLMSFetcherParams 單次 CLMS 下載請求參數（對應 @datarequest_post 的單一 Dataset 項）。
type CLMSFetcherParams struct {
	DatasetID                   string    `json:"DatasetID"`                   // UID
	DatasetDownloadInformationID string    `json:"DatasetDownloadInformationID"`
	NUTS                        string    `json:"NUTS,omitempty"`             // 例如 "ITC11"
	BoundingBox                 []float64 `json:"BoundingBox,omitempty"`      // [maxLat, maxLon, minLat, minLon] EPSG:4326
	TemporalFilter              *struct {
		StartDate int64 `json:"StartDate"` // ms since epoch
		EndDate   int64 `json:"EndDate"`
	} `json:"TemporalFilter,omitempty"`
	OutputFormat string `json:"OutputFormat"` // 如 "Geotiff"
	OutputGCS    string `json:"OutputGCS"`    // 如 "EPSG:4326"
}

// clmsDatarequestPostBody 對應 POST @datarequest_post 的 request body。
type clmsDatarequestPostBody struct {
	Datasets []clmsDatasetItem `json:"Datasets"`
}

type clmsDatasetItem struct {
	DatasetID                   string    `json:"DatasetID"`
	DatasetDownloadInformationID string    `json:"DatasetDownloadInformationID,omitempty"`
	NUTS                        string    `json:"NUTS,omitempty"`
	BoundingBox                 []float64 `json:"BoundingBox,omitempty"`
	TemporalFilter              *struct {
		StartDate int64 `json:"StartDate"`
		EndDate   int64 `json:"EndDate"`
	} `json:"TemporalFilter,omitempty"`
	OutputFormat string `json:"OutputFormat"`
	OutputGCS    string `json:"OutputGCS"`
}

// clmsDatarequestPostResponse 對應 POST @datarequest_post 的 response。
type clmsDatarequestPostResponse struct {
	TaskIds      []struct{ TaskID string } `json:"TaskIds"`
	ErrorTaskIds []interface{}             `json:"ErrorTaskIds"`
}

// clmsDatarequestStatus 對應 GET @datarequest_status_get 的 response。
type clmsDatarequestStatus struct {
	Status       string `json:"Status"`
	DownloadURL  string `json:"DownloadURL"`
	FileSize     int64  `json:"FileSize"`
	Datasets     []struct {
		DatasetID     string `json:"DatasetID"`
		DatasetTitle  string `json:"DatasetTitle"`
		OutputFormat  string `json:"OutputFormat"`
		OutputGCS     string `json:"OutputGCS"`
	} `json:"Datasets"`
}

// RunCLMSFetcher 執行一筆 CLMS 下載流程並回寫 DB。
// 使用 auth 取得 Bearer token；將下載檔案寫入 objStore；更新 job 狀態與 asset、dataset_ref。
func RunCLMSFetcher(ctx context.Context, db *gorm.DB, auth *CLMSAuth, objStore storage.ObjectStorage, jobID uint, params CLMSFetcherParams) error {
	if auth == nil || !auth.Valid() {
		return fmt.Errorf("CLMS auth not configured")
	}
	if objStore == nil {
		return fmt.Errorf("object storage not configured")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	token, err := auth.BearerToken(client)
	if err != nil {
		return fmt.Errorf("get CLMS token: %w", err)
	}

	var job model.SourceJob
	if err := db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return fmt.Errorf("load source_job: %w", err)
	}
	if job.Source != "clms" {
		return fmt.Errorf("job %d is not clms", jobID)
	}

	// 1. POST @datarequest_post
	body := clmsDatarequestPostBody{
		Datasets: []clmsDatasetItem{{
			DatasetID:                    params.DatasetID,
			DatasetDownloadInformationID: params.DatasetDownloadInformationID,
			NUTS:                         params.NUTS,
			BoundingBox:                  params.BoundingBox,
			TemporalFilter:               params.TemporalFilter,
			OutputFormat:                 params.OutputFormat,
			OutputGCS:                    params.OutputGCS,
		}},
	}
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, clmsBaseURL+"/@datarequest_post", bytes.NewReader(bodyBytes))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		updateJobStatus(ctx, db, jobID, "failed", "", nil)
		return fmt.Errorf("datarequest_post: %w", err)
	}
	defer resp.Body.Close()
	var postResp clmsDatarequestPostResponse
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		updateJobStatus(ctx, db, jobID, "failed", "", nil)
		return fmt.Errorf("decode datarequest_post response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		updateJobStatus(ctx, db, jobID, "failed", "", nil)
		return fmt.Errorf("datarequest_post returned %d", resp.StatusCode)
	}
	if len(postResp.TaskIds) == 0 {
		updateJobStatus(ctx, db, jobID, "failed", "", nil)
		return fmt.Errorf("no TaskID in response")
	}
	taskID := postResp.TaskIds[0].TaskID
	if err := updateJobStatus(ctx, db, jobID, "running", taskID, nil); err != nil {
		return err
	}

	// 2. 輪詢 @datarequest_status_get
	deadline := time.Now().Add(clmsPollMaxWait)
	var statusResp clmsDatarequestStatus
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
			return ctx.Err()
		default:
		}
		req, _ = http.NewRequestWithContext(ctx, http.MethodGet, clmsBaseURL+"/@datarequest_status_get?TaskID="+taskID, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			time.Sleep(clmsPollInterval)
			continue
		}
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			resp.Body.Close()
			time.Sleep(clmsPollInterval)
			continue
		}
		resp.Body.Close()
		if statusResp.Status == clmsStatusFinished {
			break
		}
		if statusResp.Status == clmsStatusError || strings.HasPrefix(statusResp.Status, "Finished_") {
			updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
			return fmt.Errorf("CLMS task finished with status: %s", statusResp.Status)
		}
		time.Sleep(clmsPollInterval)
	}
	if statusResp.Status != clmsStatusFinished || statusResp.DownloadURL == "" {
		updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
		return fmt.Errorf("download not ready within %v", clmsPollMaxWait)
	}

	// 3. 下載檔案
	dlReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, statusResp.DownloadURL, nil)
	dlReq.Header.Set("Authorization", "Bearer "+token)
	dlResp, err := client.Do(dlReq)
	if err != nil {
		updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
		return fmt.Errorf("download file: %w", err)
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
		return fmt.Errorf("download returned %d", dlResp.StatusCode)
	}
	contentType := dlResp.Header.Get("Content-Type")
	disposition := dlResp.Header.Get("Content-Disposition")
	filename := "clms_" + taskID
	if idx := strings.Index(disposition, "filename="); idx >= 0 {
		name := strings.Trim(disposition[idx+9:], "\" ")
		if name != "" {
			filename = path.Base(name)
		}
	}

	// 4. 上傳至 Object Storage
	bucket := "" // 使用預設
	key := fmt.Sprintf("clms/%d/%s", jobID, filename)
	putRes, err := objStore.Put(ctx, bucket, key, dlResp.Body, contentType)
	if err != nil {
		updateJobStatus(ctx, db, jobID, "failed", taskID, nil)
		return fmt.Errorf("upload to storage: %w", err)
	}

	// 5. 寫入 asset 與 dataset_ref，更新 job
	crs := params.OutputGCS
	if len(statusResp.Datasets) > 0 {
		crs = statusResp.Datasets[0].OutputGCS
	}
	size := putRes.SizeBytes
	if size == 0 && statusResp.FileSize > 0 {
		size = statusResp.FileSize
	}
	asset := model.Asset{
		JobID:      jobID,
		Filename:   filename,
		Mime:       contentType,
		CRS:        crs,
		StorageURL: putRes.StorageURL,
		Checksum:   putRes.Checksum,
		SizeBytes:  &size,
	}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		return fmt.Errorf("create asset: %w", err)
	}
	ref := model.DatasetRef{
		DatasetUID:        params.DatasetID,
		DownloadInfoID:     params.DatasetDownloadInformationID,
		ProductFamily:     params.OutputFormat,
	}
	if len(statusResp.Datasets) > 0 {
		ref.Title = statusResp.Datasets[0].DatasetTitle
	}
	if err := db.WithContext(ctx).Create(&ref).Error; err != nil {
		return fmt.Errorf("create dataset_ref: %w", err)
	}
	return updateJobStatus(ctx, db, jobID, "completed", taskID, nil)
}

func updateJobStatus(ctx context.Context, db *gorm.DB, jobID uint, status, externalTaskID string, t *time.Time) error {
	upd := map[string]interface{}{"status": status}
	if externalTaskID != "" {
		upd["external_task_id"] = externalTaskID
	}
	return db.WithContext(ctx).Model(&model.SourceJob{}).Where("id = ?", jobID).Updates(upd).Error
}

// CLMSFetcherParamsFromJob 從 source_job 的 aoi_geom / time 等推導出 CLMSFetcherParams（需呼叫端提供 dataset_uid 與 download_info_id）。
// 若 job 無額外欄位存參數，可從 external_task_id 或 metadata 解析；此處僅回傳必填，其餘由呼叫端設定。
func CLMSFetcherParamsFromJob(datasetUID, downloadInfoID, outputFormat, outputGCS string, job *model.SourceJob) CLMSFetcherParams {
	p := CLMSFetcherParams{
		DatasetID:                    datasetUID,
		DatasetDownloadInformationID: downloadInfoID,
		OutputFormat:                 outputFormat,
		OutputGCS:                    outputGCS,
	}
	if job == nil {
		return p
	}
	// 可依 job.AoiGeom 解析 bbox 或 NUTS；job.TimeStart/TimeEnd 填 TemporalFilter
	if job.TimeStart != nil && job.TimeEnd != nil {
		p.TemporalFilter = &struct {
			StartDate int64 `json:"StartDate"`
			EndDate   int64 `json:"EndDate"`
		}{
			StartDate: job.TimeStart.UnixMilli(),
			EndDate:   job.TimeEnd.UnixMilli(),
		}
	}
	return p
}
