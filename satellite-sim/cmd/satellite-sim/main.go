package main

import (
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"actinspace.org/satellite-sim/internal/ota"
	"github.com/gin-gonic/gin"
)

// CommandRequest 定義從 TT&C gateway 接收到的指令格式。
type CommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// Telemetry 定義衛星遙測數據格式。
type Telemetry struct {
	SatelliteID string    `json:"satelliteId"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // "nominal", "warning", "critical"
	Battery     float64   `json:"batteryLevel"`
	Memory      float64   `json:"memoryUsage"`
	CPU         float64   `json:"cpuUsage"`
	// [SPACE DATA] Enhanced Telemetry
	SolarCurrent   float64    `json:"solarCurrent"`   // Amps
	BatteryVoltage float64    `json:"batteryVoltage"` // Volts
	Temperature    float64    `json:"temperature"`    // Celsius
	Attitude       [4]float64 `json:"attitude"`      // Quaternions [q1, q2, q3, q4]
}

// CommandResponse 是衛星模擬節點回應的基本格式。
type CommandResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func generateTelemetry(satelliteID string) Telemetry {
	// [SPACE DATA] Simulation Logic
	// Temperature follows a sine wave to simulate orbit (sunlight/eclipse)
	t := float64(time.Now().Unix()%6000) / 6000.0 * 2 * math.Pi
	temp := 20.0 + 50.0*math.Sin(t) // -30C to +70C

	// Battery voltage depends on solar current (simplified)
	solarCurrent := 0.0
	if temp > 0 { // In sunlight
		solarCurrent = 10.0 + rand.Float64()*2.0
	}
	battVolts := 24.0 + (solarCurrent * 0.1)

	return Telemetry{
		SatelliteID:    satelliteID,
		Timestamp:      time.Now().UTC(),
		Status:         "nominal",
		Battery:        80.0 + rand.Float64()*20.0,
		Memory:         30.0 + rand.Float64()*10.0,
		CPU:            10.0 + rand.Float64()*40.0,
		// Enhanced fields
		SolarCurrent:   solarCurrent,
		BatteryVoltage: battVolts,
		Temperature:    temp,
		Attitude:       [4]float64{0.707, 0.0, 0.707, 0.0}, // Stable
	}
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


