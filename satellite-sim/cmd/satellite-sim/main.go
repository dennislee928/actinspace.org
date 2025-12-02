package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"actinspace.org/satellite-sim/internal/ota"
)

// CommandRequest 定義從 TT&C gateway 接收到的指令格式。
type CommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// CommandResponse 是衛星模擬節點回應的基本格式。
type CommandResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func main() {
	r := gin.Default()

	// 啟動 OTA client（如果配置了 OTA controller URL）
	otaControllerURL := os.Getenv("OTA_CONTROLLER_URL")
	if otaControllerURL != "" {
		version := os.Getenv("VERSION")
		if version == "" {
			version = "v1.0.0"
		}

		otaClient := ota.NewClient(otaControllerURL, "satellite-sim", version)
		go otaClient.StartUpdateLoop(30 * time.Second) // 每 30 秒檢查一次
		log.Printf("OTA client 已啟動，連接到: %s", otaControllerURL)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/command", func(c *gin.Context) {
		var req CommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf(`{"component":"satellite-sim","event":"command_received","command":"%s"}`, req.Command)

		resp := CommandResponse{
			Status:     "accepted",
			Message:    "command queued for execution (simulated)",
			ReceivedAt: time.Now().UTC(),
		}
		c.JSON(http.StatusOK, resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("satellite-sim server failed: %v", err)
	}
}


