package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Release 定義一個軟體發布版本。
type Release struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Component   string    `gorm:"not null;index" json:"component"` // satellite-sim, ttc-gateway, etc.
	Version     string    `gorm:"not null" json:"version"`
	ImageDigest string    `gorm:"not null" json:"imageDigest"`
	SBOMURL     string    `json:"sbomUrl,omitempty"`
	Attestation string    `gorm:"type:text" json:"attestation"` // JSON string
	Status      string    `gorm:"not null;index" json:"status"` // "pending", "approved", "rejected"
	ApprovedBy  string    `json:"approvedBy,omitempty"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UpdateRequest 定義衛星請求更新的格式。
type UpdateRequest struct {
	Component      string `json:"component" binding:"required"`
	CurrentVersion string `json:"currentVersion"`
	SatelliteID    string `json:"satelliteId,omitempty"`
}

// UpdateResponse 定義 OTA controller 的回應。
type UpdateResponse struct {
	Available      bool      `json:"available"`
	Version        string    `json:"version,omitempty"`
	ImageDigest    string    `json:"imageDigest,omitempty"`
	SBOMURL        string    `json:"sbomUrl,omitempty"`
	Attestation    string    `json:"attestation,omitempty"`
	Message        string    `json:"message"`
	UpdateAllowed  bool      `json:"updateAllowed"`
	DenialReason   string    `json:"denialReason,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

var db *gorm.DB

func initDB() {
	var err error
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "ota-controller.db"
	}

	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("無法連接到資料庫: %v", err)
	}

	// 自動遷移
	if err := db.AutoMigrate(&Release{}); err != nil {
		log.Fatalf("資料庫遷移失敗: %v", err)
	}

	log.Println("OTA Controller 資料庫初始化完成")
}

func main() {
	initDB()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 查詢可用更新
	r.POST("/api/v1/updates/check", func(c *gin.Context) {
		var req UpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 查找最新的已批准版本
		var latestRelease Release
		err := db.Where("component = ? AND status = ?", req.Component, "approved").
			Order("created_at DESC").
			First(&latestRelease).Error

		if err != nil {
			// 沒有可用更新
			c.JSON(http.StatusOK, UpdateResponse{
				Available:     false,
				Message:       "no approved updates available",
				UpdateAllowed: false,
				Timestamp:     time.Now().UTC(),
			})
			return
		}

		// 檢查是否為新版本
		if latestRelease.Version == req.CurrentVersion {
			c.JSON(http.StatusOK, UpdateResponse{
				Available:     false,
				Message:       "already on latest version",
				UpdateAllowed: false,
				Timestamp:     time.Now().UTC(),
			})
			return
		}

		// 檢查任務政策（例如：關鍵階段禁止更新）
		missionPhase := os.Getenv("MISSION_PHASE")
		if missionPhase == "critical" {
			c.JSON(http.StatusOK, UpdateResponse{
				Available:     true,
				Version:       latestRelease.Version,
				UpdateAllowed: false,
				DenialReason:  "updates blocked during critical mission phase",
				Timestamp:     time.Now().UTC(),
			})
			return
		}

		// 允許更新
		c.JSON(http.StatusOK, UpdateResponse{
			Available:     true,
			Version:       latestRelease.Version,
			ImageDigest:   latestRelease.ImageDigest,
			SBOMURL:       latestRelease.SBOMURL,
			Attestation:   latestRelease.Attestation,
			Message:       "update available",
			UpdateAllowed: true,
			Timestamp:     time.Now().UTC(),
		})

		// 記錄更新檢查事件
		logEvent("update_check", map[string]interface{}{
			"component":      req.Component,
			"currentVersion": req.CurrentVersion,
			"latestVersion":  latestRelease.Version,
			"satelliteId":    req.SatelliteID,
			"updateAllowed":  true,
		})
	})

	// 註冊新版本（由 CI pipeline 調用）
	r.POST("/api/v1/releases", func(c *gin.Context) {
		var req struct {
			Component   string `json:"component" binding:"required"`
			Version     string `json:"version" binding:"required"`
			ImageDigest string `json:"imageDigest" binding:"required"`
			SBOMURL     string `json:"sbomUrl,omitempty"`
			Attestation string `json:"attestation,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		release := Release{
			Component:   req.Component,
			Version:     req.Version,
			ImageDigest: req.ImageDigest,
			SBOMURL:     req.SBOMURL,
			Attestation: req.Attestation,
			Status:      "pending", // 需要人工批准
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		if err := db.Create(&release).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法創建 release"})
			return
		}

		logEvent("release_registered", map[string]interface{}{
			"component":   req.Component,
			"version":     req.Version,
			"imageDigest": req.ImageDigest,
			"status":      "pending",
		})

		c.JSON(http.StatusCreated, release)
	})

	// 批准版本
	r.POST("/api/v1/releases/:id/approve", func(c *gin.Context) {
		var release Release
		idStr := c.Param("id")

		// 驗證 ID 是有效的數字（防止 SQL injection）
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid release ID"})
			return
		}

		if err := db.First(&release, uint(id)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
			return
		}

		release.Status = "approved"
		release.ApprovedBy = "admin" // 實際應從認證 token 取得
		release.UpdatedAt = time.Now().UTC()

		if err := db.Save(&release).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法批准 release"})
			return
		}

		logEvent("release_approved", map[string]interface{}{
			"component":  release.Component,
			"version":    release.Version,
			"approvedBy": release.ApprovedBy,
		})

		c.JSON(http.StatusOK, release)
	})

	// 查詢所有 releases
	r.GET("/api/v1/releases", func(c *gin.Context) {
		var releases []Release
		query := db.Model(&Release{})

		if component := c.Query("component"); component != "" {
			query = query.Where("component = ?", component)
		}
		if status := c.Query("status"); status != "" {
			query = query.Where("status = ?", status)
		}

		query = query.Order("created_at DESC").Limit(100)

		if err := query.Find(&releases).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢 releases"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"releases": releases, "count": len(releases)})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("ota-controller server failed: %v", err)
	}
}

// logEvent 記錄結構化日誌。
func logEvent(eventType string, data map[string]interface{}) {
	logData := map[string]interface{}{
		"component": "ota-controller",
		"event":     eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range data {
		logData[k] = v
	}
	jsonData, _ := json.Marshal(logData)
	log.Println(string(jsonData))

	// 發送到 Space-SOC（如果配置）
	socURL := os.Getenv("SPACE_SOC_URL")
	if socURL != "" {
		sendEventToSOC(socURL, logData)
	}
}

// sendEventToSOC 發送事件到 Space-SOC。
func sendEventToSOC(socURL string, event map[string]interface{}) {
	// 轉換為 Space-SOC 格式
	socEvent := map[string]interface{}{
		"component": event["component"],
		"eventType": event["event"],
	}
	for k, v := range event {
		if k != "component" && k != "event" && k != "timestamp" {
			socEvent[k] = v
		}
	}

	eventData, _ := json.Marshal(socEvent)
	resp, err := http.Post(socURL+"/api/v1/events", "application/json", bytes.NewBuffer(eventData))
	if err != nil {
		log.Printf("無法發送事件到 Space-SOC: %v", err)
		return
	}
	defer resp.Body.Close()
}

