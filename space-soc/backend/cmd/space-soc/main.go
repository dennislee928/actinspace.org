package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	ScenarioID   string    `gorm:"index" json:"scenarioID,omitempty"`   // 關聯的威脅場景
	IncidentID   *uint     `gorm:"index" json:"incidentID,omitempty"`   // 關聯的 incident
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
		// 預設使用 SQLite（開發環境）
		dialector = sqlite.Open("space-soc.db")
	} else {
		// 使用 PostgreSQL（生產環境）
		dialector = postgres.Open(dbURL)
	}

	db, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("無法連接到資料庫: %v", err)
	}

	// 自動遷移
	if err := db.AutoMigrate(&Event{}, &Incident{}); err != nil {
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

func main() {
	initDB()

	r := gin.Default()

	// CORS 設定（允許 frontend 存取）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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

	// 查詢事件（依場景）
	r.GET("/api/v1/events/scenario/:scenarioId", func(c *gin.Context) {
		scenarioID := c.Param("scenarioId")
		var events []Event

		if err := db.Where("scenario_id = ?", scenarioID).Order("created_at DESC").Find(&events).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法查詢事件"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"events": events, "count": len(events), "scenarioId": scenarioID})
	})

	// Incident API
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("space-soc backend server failed: %v", err)
	}
}
