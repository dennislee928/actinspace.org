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
	ID          uint      `gorm:"primaryKey" json:"id"`
	Component   string    `gorm:"not null;index" json:"component"`
	EventType   string    `gorm:"not null;index" json:"eventType"`
	Command     string    `gorm:"index" json:"command,omitempty"`
	OperatorRole string   `gorm:"index" json:"operatorRole,omitempty"`
	Decision    string    `json:"decision,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Status      string    `json:"status,omitempty"`
	Message     string    `json:"message,omitempty"`
	Metadata    string    `gorm:"type:text" json:"metadata,omitempty"` // JSON string
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
}

// IngestRequest 定義從外部組件接收的事件格式。
type IngestRequest struct {
	Component   string                 `json:"component" binding:"required"`
	EventType   string                 `json:"eventType" binding:"required"`
	Command     string                 `json:"command,omitempty"`
	OperatorRole string                `json:"operatorRole,omitempty"`
	Decision    string                 `json:"decision,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	if err := db.AutoMigrate(&Event{}); err != nil {
		log.Fatalf("資料庫遷移失敗: %v", err)
	}

	log.Println("資料庫初始化完成")
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
			EventType:   req.EventType,
			Command:     req.Command,
			OperatorRole: req.OperatorRole,
			Decision:    req.Decision,
			Reason:      req.Reason,
			Status:      req.Status,
			Message:     req.Message,
			Metadata:    metadataJSON,
			CreatedAt:   time.Now().UTC(),
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("space-soc backend server failed: %v", err)
	}
}

