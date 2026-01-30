package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"actinspace.org/space-soc/backend/internal/integrations"
	"actinspace.org/space-soc/backend/internal/integrations/s2gm"
	"actinspace.org/space-soc/backend/internal/dto"
	"actinspace.org/space-soc/backend/internal/model"
	"actinspace.org/space-soc/backend/internal/scheduler"
	"actinspace.org/space-soc/backend/internal/storage"
	"actinspace.org/space-soc/backend/internal/vo"
)

// Event 定義 Space-SOC 儲存的事件格式。
type Event struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Component    string    `gorm:"not null;index" json:"component"`
	EventType    string    `gorm:"not null;index" json:"eventType"`
	Command      string    `gorm:"index" json:"command,omitempty"`
	OperatorRole string    `gorm:"index" json:"operatorRole,omitempty"`
	Decision     string    `json:"decision,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Status       string    `json:"status,omitempty"`
	Message      string    `json:"message,omitempty"`
	Severity     string    `gorm:"index" json:"severity,omitempty"` // "low", "medium", "high", "critical"
	RuleID       string    `json:"ruleID,omitempty"`
	AnomalyType  string    `json:"anomalyType,omitempty"`
	ScenarioID   string    `gorm:"index" json:"scenarioID,omitempty"` // 關聯的威脅場景
	IncidentID   *uint     `gorm:"index" json:"incidentID,omitempty"` // 關聯的 incident
	Metadata     string    `gorm:"type:text" json:"metadata,omitempty"` // JSON string
	CreatedAt    time.Time `gorm:"index" json:"createdAt"`
}

// Incident 定義安全事件。
type Incident struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Severity    string    `gorm:"not null;index" json:"severity"`            // "low", "medium", "high", "critical"
	Status      string    `gorm:"not null;index;default:open" json:"status"` // "open", "investigating", "resolved", "closed"
	ScenarioID  string    `gorm:"index" json:"scenarioID,omitempty"`         // 關聯的威脅場景
	Events      []Event   `gorm:"foreignKey:IncidentID" json:"events,omitempty"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SoftwarePosture 定義組件的軟體姿態。
type SoftwarePosture struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Component       string    `gorm:"not null;uniqueIndex" json:"component"` // satellite-sim, ttc-gateway, etc.
	CurrentVersion  string    `gorm:"not null" json:"currentVersion"`
	LatestVersion   string    `json:"latestVersion,omitempty"`
	ImageDigest     string    `json:"imageDigest,omitempty"`
	SBOMURL         string    `json:"sbomUrl,omitempty"`
	VulnCount       int       `json:"vulnCount"` // 已知漏洞數量
	LastScanTime    time.Time `json:"lastScanTime,omitempty"`
	LastUpdateTime  time.Time `json:"lastUpdateTime,omitempty"`
	UpdateAvailable bool      `json:"updateAvailable"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// IngestRequest 定義從外部組件接收的事件格式。
type IngestRequest struct {
	Component    string                 `json:"component" binding:"required"`
	EventType    string                 `json:"eventType" binding:"required"`
	Command      string                 `json:"command,omitempty"`
	OperatorRole string                 `json:"operatorRole,omitempty"`
	Decision     string                 `json:"decision,omitempty"`
	Reason       string                 `json:"reason,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Severity     string                 `json:"severity,omitempty"`
	RuleID       string                 `json:"ruleID,omitempty"`
	AnomalyType  string                 `json:"anomalyType,omitempty"`
	ScenarioID   string                 `json:"scenarioID,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

var db *gorm.DB

func initDB() {
	var err error
	var dialector gorm.Dialector

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("SUPABASE_DATABASE_URL")
	}
	if dbURL != "" {
		dialector = postgres.Open(dbURL)
	} else {
		// 預設使用 SQLite（開發環境）
		dialector = sqlite.Open("space-soc.db")
	}

	db, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("無法連接到資料庫: %v", err)
	}

	// 自動遷移（含 CLMS/S2GM 表；正式環境可改用 database/migrations 執行 SQL）
	if err := db.AutoMigrate(&Event{}, &Incident{}, &SoftwarePosture{},
		&model.SourceJob{}, &model.Asset{}, &model.DatasetRef{}); err != nil {
		log.Fatalf("資料庫遷移失敗: %v", err)
	}

	log.Println("資料庫初始化完成")
}

// createOrUpdateIncident 根據事件創建或更新 incident。
func createOrUpdateIncident(req IngestRequest, db *gorm.DB) *Incident {
	// 查找是否有相關的開放 incident
	var existingIncident Incident
	query := db.Where("status IN ?", []string{"open", "investigating"})

	if req.ScenarioID != "" {
		query = query.Where("scenario_id = ?", req.ScenarioID)
	} else if req.Severity == "critical" || req.Severity == "high" {
		// 查找相同嚴重性的開放 incident
		query = query.Where("severity = ?", req.Severity)
	}

	query.First(&existingIncident)

	now := time.Now().UTC()

	if existingIncident.ID == 0 {
		// 創建新 incident
		title := fmt.Sprintf("Security Incident: %s", req.EventType)
		if req.Severity == "critical" {
			title = fmt.Sprintf("CRITICAL: %s", req.EventType)
		}

		incident := Incident{
			Title:       title,
			Description: fmt.Sprintf("Detected %s event from %s. %s", req.EventType, req.Component, req.Message),
			Severity:    req.Severity,
			Status:      "open",
			ScenarioID:  req.ScenarioID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := db.Create(&incident).Error; err != nil {
			log.Printf("無法創建 incident: %v", err)
			return nil
		}

		return &incident
	} else {
		// 更新現有 incident
		existingIncident.UpdatedAt = now
		if existingIncident.Status == "open" && req.Severity == "critical" {
			existingIncident.Status = "investigating"
		}
		db.Save(&existingIncident)
		return &existingIncident
	}
}

// ResourceItem 單一資源連結（專利／EO／ACRI-ST）。
type ResourceItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Note        string `json:"note,omitempty"`
}

// ResourceCategory 資源分類（專利資料庫、地球觀測、ACRI-ST）。
type ResourceCategory struct {
	ID    string         `json:"id"`
	Label string         `json:"label"`
	Items []ResourceItem `json:"items"`
}

// getResourcesHandler 回傳 GET /api/v1/resources 的 handler；ACRI-ST 連結由 ACRI_ST_FOLDER_URL 填入。
func getResourcesHandler() gin.HandlerFunc {
	acriURL := os.Getenv("ACRI_ST_FOLDER_URL")
	if acriURL == "" {
		acriURL = "#"
	}
	return func(c *gin.Context) {
		categories := []ResourceCategory{
			{
				ID:    "patents",
				Label: "專利資料庫",
				Items: []ResourceItem{
					{
						Title:       "CNES",
						URL:         "https://www.connectbycnes.fr/ressources-valorisation",
						Description: "Connect by CNES 專利與資源",
					},
					{
						Title:       "ESA",
						URL:         "https://commercialisation.esa.int/patents/",
						Description: "ESA 專利資料庫",
					},
					{
						Title:       "Airbus",
						URL:         "https://worldwide.espacenet.com/advancedSearch?locale=en_EP",
						Description: "Espacenet 進階搜尋",
						Note:        "Applicant 選「Airbus Defence」以檢視清單",
					},
				},
			},
			{
				ID:    "earth_observation",
				Label: "地球觀測／Copernicus",
				Items: []ResourceItem{
					{
						Title:       "S2GM (Sentinel-2 Global Mosaic)",
						URL:         "https://s2gm.land.copernicus.eu/",
						Description: "全球 mosaic／時間序列；不需 VPN（2026.01.27 起）",
					},
					{
						Title:       "CLMS",
						URL:         "https://land.copernicus.eu/en",
						Description: "Copernicus Land Monitoring Service；陸域產品（土地覆蓋、地表形變、植被等）",
					},
					{
						Title:       "Copernicus Data Space Ecosystem",
						URL:         "https://dataspace.copernicus.eu/",
						Description: "登入後可使用 OGC WMTS/WMS、STAC 等",
					},
				},
			},
			{
				ID:    "acri_st",
				Label: "ACRI-ST／挑戰題",
				Items: []ResourceItem{
					{
						Title:       "ACRI-ST #1",
						URL:         acriURL,
						Description: "挑戰題 ACRI ST #1、雲端資料夾、中文翻譯與專利 PDF 連結",
						Note:        "主辦方註明需 VPN 連法國，依說明使用",
					},
					{
						Title:       "Copernicus Marine",
						URL:         "https://marine.copernicus.eu/",
						Description: "海洋數據；註冊後可搜尋關鍵字 OCEANCOLOUR 取得相關產品",
					},
					{
						Title:       "OCDB",
						URL:         "https://ocdb.eumetsat.int/",
						Description: "Ocean Colour 資料；可透過 ocdb-cli 或 Python API 存取，詳見官方文件",
					},
				},
			},
		}
		c.JSON(http.StatusOK, gin.H{"categories": categories})
	}
}

// parseOptionalTime 解析 ISO8601 字串為 *time.Time；空字串或 nil 回傳 nil, nil。
func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// createSourceJobHandler 建立一筆 source_job（status=pending），CLMS 任務由 Scheduler 撿取執行。
func createSourceJobHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateSourceJobRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var timeStart, timeEnd *time.Time
		if req.TimeStart != nil {
			t, err := parseOptionalTime(req.TimeStart)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_start: " + err.Error()})
				return
			}
			timeStart = t
		}
		if req.TimeEnd != nil {
			t, err := parseOptionalTime(req.TimeEnd)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_end: " + err.Error()})
				return
			}
			timeEnd = t
		}
		job := model.SourceJob{
			Source:          req.Source,
			AoiGeom:         req.AoiGeom,
			TimeStart:       timeStart,
			TimeEnd:         timeEnd,
			Status:          "pending",
			DatasetUID:      req.DatasetUID,
			DownloadInfoID:  req.DownloadInfoID,
			OutputFormat:   req.OutputFormat,
			OutputGCS:       req.OutputGCS,
		}
		if err := db.Create(&job).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
			return
		}
		c.JSON(http.StatusCreated, sourceJobToVO(&job))
	}
}

// listSourceJobsHandler 查詢 source_job，支援 query 篩選 source、status。
func listSourceJobsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := db.Model(&model.SourceJob{})
		if source := c.Query("source"); source != "" {
			query = query.Where("source = ?", source)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		var jobs []model.SourceJob
		if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
			return
		}
		list := make([]vo.SourceJobVO, len(jobs))
		for i := range jobs {
			list[i] = sourceJobToVO(&jobs[i])
		}
		c.JSON(http.StatusOK, gin.H{"jobs": list, "count": len(list)})
	}
}

// getSourceJobHandler 回傳單一 source_job。
func getSourceJobHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var job model.SourceJob
		if err := db.First(&job, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusOK, sourceJobToVO(&job))
	}
}

// listSourceJobAssetsHandler 回傳該 job 的資產列表（VO：filename, storage_url, mime 等）。
func listSourceJobAssetsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		jobID, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var job model.SourceJob
		if err := db.First(&job, jobID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		var assets []model.Asset
		if err := db.Where("job_id = ?", jobID).Find(&assets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assets"})
			return
		}
		list := make([]vo.AssetVO, len(assets))
		for i := range assets {
			list[i] = vo.AssetVO{
				ID:         assets[i].ID,
				JobID:      assets[i].JobID,
				Filename:   assets[i].Filename,
				Mime:       assets[i].Mime,
				CRS:        assets[i].CRS,
				Resolution: assets[i].Resolution,
				StorageURL: assets[i].StorageURL,
				SizeBytes:  assets[i].SizeBytes,
				CreatedAt:  assets[i].CreatedAt,
			}
		}
		c.JSON(http.StatusOK, gin.H{"assets": list, "count": len(list)})
	}
}

func sourceJobToVO(j *model.SourceJob) vo.SourceJobVO {
	return vo.SourceJobVO{
		ID:             j.ID,
		Source:         j.Source,
		AoiGeom:        j.AoiGeom,
		TimeStart:      j.TimeStart,
		TimeEnd:        j.TimeEnd,
		Status:         j.Status,
		ExternalTaskID: j.ExternalTaskID,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
}

// updateSoftwarePosture 更新組件的軟體姿態。
func updateSoftwarePosture(component, version, imageDigest string, db *gorm.DB) {
	var posture SoftwarePosture

	err := db.Where("component = ?", component).First(&posture).Error
	if err != nil {
		// 創建新記錄
		posture = SoftwarePosture{
			Component:      component,
			CurrentVersion: version,
			ImageDigest:    imageDigest,
			LastUpdateTime: time.Now().UTC(),
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		db.Create(&posture)
	} else {
		// 更新現有記錄
		posture.CurrentVersion = version
		posture.ImageDigest = imageDigest
		posture.LastUpdateTime = time.Now().UTC()
		posture.UpdatedAt = time.Now().UTC()
		db.Save(&posture)
	}
}

func main() {
	initDB()

	// CLMS Scheduler：週期掃 pending CLMS job 並執行 Fetcher（需 CLMS JWT + Object Storage）
	auth, _ := integrations.NewCLMSAuthFromEnv()
	objStore, _ := storage.FromEnv()
	go scheduler.RunCLMSScheduler(context.Background(), db, auth, objStore)

	// S2GM Watcher：監控 S2GM_WATCH_DIR，完成產品上傳至 Object Storage 並寫入 source_job/asset
	go s2gm.RunS2GMWatcher(context.Background(), db, objStore, os.Getenv("S2GM_WATCH_DIR"))

	r := gin.Default()

	// CORS 設定（允許 frontend 存取）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Resources API：專利／地球觀測／ACRI-ST 連結與說明（前端參考資源頁資料來源）
	r.GET("/api/v1/resources", getResourcesHandler())

	// Copernicus WMTS/WMS 設定（含 S2GM 底圖）；前端地圖可選顯示 S2GM
	r.GET("/api/v1/copernicus/wmts-config", integrations.CopernicusWMTSConfig())

	// CLMS：代理 land.copernicus.eu 資料集列表與下載請求（下載需 CLMS_BEARER_TOKEN）
	r.GET("/api/v1/clms/datasets", integrations.CLMSDatasets())
	r.POST("/api/v1/clms/datarequest", integrations.CLMSDataRequest())

	// Source jobs：建立/查詢 CLMS/S2GM 任務；GET :id/assets 回傳該 job 的資產列表
	r.POST("/api/v1/jobs/source", createSourceJobHandler(db))
	r.GET("/api/v1/jobs/source", listSourceJobsHandler(db))
	r.GET("/api/v1/jobs/source/:id", getSourceJobHandler(db))
	r.GET("/api/v1/jobs/source/:id/assets", listSourceJobAssetsHandler(db))

	// EPO OPS：代理專利搜尋（需 EPO_OPS_CONSUMER_KEY、EPO_OPS_CONSUMER_SECRET）
	r.GET("/api/v1/patents/search", integrations.PatentsSearch())

	// [ECONOMIC MODEL] - Enterprise Feature
	// This is a placeholder for a licensing middleware.
	// In a real application, `middleware.CheckLicenseMiddleware()` would verify
	// if the user/organization has the necessary license for enterprise features.
	// For this example, we'll just add a dummy endpoint.
	r.GET("/api/v1/enterprise/threat-replay", func(c *gin.Context) {
		// In a real scenario, CheckLicenseMiddleware would handle access.
		// For this example, we'll just grant access.
		c.JSON(http.StatusOK, gin.H{
			"status": "access_granted",
			"msg":    "Welcome to the Enterprise Threat Replay Engine",
		})
	})

	// 事件接收端點
	r.POST("/api/v1/events", func(c *gin.Context) {
		var req IngestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 將 metadata 轉換為 JSON 字串
		var metadataJSON string
		if req.Metadata != nil {
			metadataBytes, _ := json.Marshal(req.Metadata)
			metadataJSON = string(metadataBytes)
		}

		event := Event{
			Component:    req.Component,
			EventType:    req.EventType,
			Command:      req.Command,
			OperatorRole: req.OperatorRole,
			Decision:     req.Decision,
			Reason:       req.Reason,
			Status:       req.Status,
			Message:      req.Message,
			Severity:     req.Severity,
			RuleID:       req.RuleID,
			AnomalyType:  req.AnomalyType,
			ScenarioID:   req.ScenarioID,
			Metadata:     metadataJSON,
			CreatedAt:    time.Now().UTC(),
		}

		// 如果是高嚴重性事件，自動創建或更新 incident
		if req.Severity == "high" || req.Severity == "critical" {
			incident := createOrUpdateIncident(req, db)
			if incident != nil {
				event.IncidentID = &incident.ID
			}
		}

		// 如果是 OTA 相關事件，更新軟體姿態
		if req.EventType == "release_approved" || req.EventType == "update_applied" {
			if component, ok := req.Metadata["component"].(string); ok {
				if version, ok := req.Metadata["version"].(string); ok {
					imageDigest := ""
					if digest, ok := req.Metadata["imageDigest"].(string); ok {
						imageDigest = digest
					}
					updateSoftwarePosture(component, version, imageDigest, db)
				}
			}
		}

		if err := db.Create(&event).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法儲存事件"})
			return
		}

		c.JSON(http.StatusCreated, event)
	})

	// 查詢事件端點
	r.GET("/api/v1/events", func(c *gin.Context) {
		var events []Event
		query := db.Model(&Event{})

		// 可選的篩選參數
		if component := c.Query("component"); component != "" {
			query = query.Where("component = ?", component)
		}
		if eventType := c.Query("eventType"); eventType != "" {
			query = query.Where("event_type = ?", eventType)
		}
		if command := c.Query("command"); command != "" {
			query = query.Where("command = ?", command)
		}

		// 限制結果數量（預設 100）
		limit := 100
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
				limit = parsedLimit
			}
		}
		query = query.Limit(limit).Order("created_at DESC")

		if err := query.Find(&events).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢事件"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"events": events, "count": len(events)})
	})

	// Incident API（必須在 events/scenario 之前註冊，避免路由衝突）
	// 創建 incident
	r.POST("/api/v1/incidents", func(c *gin.Context) {
		var req struct {
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
			Severity    string `json:"severity" binding:"required"`
			ScenarioID  string `json:"scenarioID,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		incident := Incident{
			Title:       req.Title,
			Description: req.Description,
			Severity:    req.Severity,
			Status:      "open",
			ScenarioID:  req.ScenarioID,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err := db.Create(&incident).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法創建 incident"})
			return
		}

		c.JSON(http.StatusCreated, incident)
	})

	// 查詢所有 incidents
	r.GET("/api/v1/incidents", func(c *gin.Context) {
		var incidents []Incident
		query := db.Model(&Incident{})

		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if severity := c.Query("severity"); severity != "" {
			query = query.Where("severity = ?", severity)
		}
		if scenarioID := c.Query("scenarioId"); scenarioID != "" {
			query = query.Where("scenario_id = ?", scenarioID)
		}

		query = query.Preload("Events").Order("created_at DESC").Limit(100)

		if err := query.Find(&incidents).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢 incidents"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"incidents": incidents, "count": len(incidents)})
	})

	// 查詢單一 incident
	r.GET("/api/v1/incidents/:id", func(c *gin.Context) {
		var incident Incident
		idStr := c.Param("id")

		// 驗證 ID 是有效的數字（防止 SQL injection）
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident ID"})
			return
		}

		if err := db.Preload("Events").First(&incident, uint(id)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
			return
		}

		c.JSON(http.StatusOK, incident)
	})

	// 更新 incident 狀態
	r.PATCH("/api/v1/incidents/:id", func(c *gin.Context) {
		var incident Incident
		idStr := c.Param("id")

		// 驗證 ID 是有效的數字（防止 SQL injection）
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident ID"})
			return
		}

		if err := db.First(&incident, uint(id)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
			return
		}

		var req struct {
			Status string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Status != "" {
			incident.Status = req.Status
		}
		incident.UpdatedAt = time.Now().UTC()

		if err := db.Save(&incident).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法更新 incident"})
			return
		}

		c.JSON(http.StatusOK, incident)
	})

	// Software Posture API
	// 查詢所有組件的軟體姿態
	r.GET("/api/v1/posture", func(c *gin.Context) {
		var postures []SoftwarePosture

		if err := db.Order("component ASC").Find(&postures).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢軟體姿態"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"postures": postures, "count": len(postures)})
	})

	// 查詢單一組件的軟體姿態
	r.GET("/api/v1/posture/:component", func(c *gin.Context) {
		component := c.Param("component")
		var posture SoftwarePosture

		if err := db.Where("component = ?", component).First(&posture).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "component not found"})
			return
		}

		c.JSON(http.StatusOK, posture)
	})

	// 更新組件軟體姿態（由 OTA controller 或 CI 調用）
	r.POST("/api/v1/posture", func(c *gin.Context) {
		var req struct {
			Component       string    `json:"component" binding:"required"`
			CurrentVersion  string    `json:"currentVersion" binding:"required"`
			ImageDigest     string    `json:"imageDigest,omitempty"`
			SBOMURL         string    `json:"sbomUrl,omitempty"`
			VulnCount       int       `json:"vulnCount"`
			LastScanTime    time.Time `json:"lastScanTime,omitempty"`
			UpdateAvailable bool      `json:"updateAvailable"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var posture SoftwarePosture
		err := db.Where("component = ?", req.Component).First(&posture).Error

		now := time.Now().UTC()

		if err != nil {
			// 創建新記錄
			posture = SoftwarePosture{
				Component:       req.Component,
				CurrentVersion:  req.CurrentVersion,
				ImageDigest:     req.ImageDigest,
				SBOMURL:         req.SBOMURL,
				VulnCount:       req.VulnCount,
				LastScanTime:    req.LastScanTime,
				UpdateAvailable: req.UpdateAvailable,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := db.Create(&posture).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "無法創建軟體姿態"})
				return
			}
		} else {
			// 更新現有記錄
			posture.CurrentVersion = req.CurrentVersion
			posture.ImageDigest = req.ImageDigest
			posture.SBOMURL = req.SBOMURL
			posture.VulnCount = req.VulnCount
			posture.LastScanTime = req.LastScanTime
			posture.UpdateAvailable = req.UpdateAvailable
			posture.UpdatedAt = now
			db.Save(&posture)
		}

		c.JSON(http.StatusOK, posture)
	})

	// 查詢事件（依場景）- 放在 incidents 路由之後，避免路由衝突
	r.GET("/api/v1/events/scenario/:scenarioId", func(c *gin.Context) {
		scenarioID := c.Param("scenarioId")
		var events []Event

		if err := db.Where("scenario_id = ?", scenarioID).Order("created_at DESC").Find(&events).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢事件"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"events": events, "count": len(events), "scenarioId": scenarioID})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("space-soc backend server failed: %v", err)
	}
}
