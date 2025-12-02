package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// CommandRequest 定義從 ground-station 接收到的指令格式。
type CommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params,omitempty"`
	SatelliteID string             `json:"satelliteId,omitempty"`
}

// CommandResponse 是 gateway 回應的格式。
type CommandResponse struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Decision    string    `json:"decision"` // "allowed" or "denied"
	Reason      string    `json:"reason,omitempty"`
	ProcessedAt time.Time `json:"processedAt"`
}

// PolicyDecision 定義 policy 引擎的決策結果。
type PolicyDecision struct {
	Allowed bool
	Reason  string
}

// 硬編碼的 policy 規則（Phase 1 MVP）
func evaluatePolicy(cmd string, operatorRole string) PolicyDecision {
	// 危險指令列表
	dangerousCommands := map[string]bool{
		"deorbit":      true,
		"disable_power": true,
		"format_memory": true,
	}

	// 檢查是否為危險指令
	if dangerousCommands[cmd] {
		if operatorRole != "admin" {
			return PolicyDecision{
				Allowed: false,
				Reason:  fmt.Sprintf("command '%s' requires admin role, got '%s'", cmd, operatorRole),
			}
		}
	}

	// 預設允許
	return PolicyDecision{
		Allowed: true,
		Reason:  "command allowed by policy",
	}
}

// 轉發指令到 satellite-sim
func forwardToSatellite(satelliteURL string, req CommandRequest) (*CommandResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(satelliteURL+"/command", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cmdResp CommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return nil, err
	}

	return &cmdResp, nil
}

// 記錄結構化日誌
func logCommandEvent(eventType string, data map[string]interface{}) {
	logData := map[string]interface{}{
		"component": "ttc-gateway",
		"event":     eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range data {
		logData[k] = v
	}
	jsonData, _ := json.Marshal(logData)
	log.Println(string(jsonData))
}

// 發送事件到 Space-SOC
func sendEventToSOC(socURL string, event map[string]interface{}) {
	if socURL == "" {
		return // 如果未設定 SOC URL，跳過
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		log.Printf("無法序列化事件: %v", err)
		return
	}

	resp, err := http.Post(socURL+"/api/v1/events", "application/json", bytes.NewBuffer(eventData))
	if err != nil {
		log.Printf("無法發送事件到 Space-SOC: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Space-SOC 回應錯誤狀態碼: %d", resp.StatusCode)
	}
}

func main() {
	r := gin.Default()

	// 從環境變數讀取配置
	satelliteURL := os.Getenv("SATELLITE_SIM_URL")
	if satelliteURL == "" {
		satelliteURL = "http://satellite-sim:8082"
	}

	// Token 驗證中間件（簡化版，Phase 1 MVP）
	authMiddleware := func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
			c.Abort()
			return
		}

		// 簡化的 token 驗證（實際應使用 JWT 或 OIDC）
		// 這裡假設 token 格式為 "Bearer <role>"
		role := "operator" // 預設角色
		if len(token) > 7 && token[:7] == "Bearer " {
			roleToken := token[7:]
			// 簡單的角色映射（實際應從 token 解析）
			if roleToken == "admin-token" {
				role = "admin"
			} else if roleToken == "engineer-token" {
				role = "engineer"
			}
		}

		c.Set("operatorRole", role)
		c.Set("token", token)
		c.Next()
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/command", authMiddleware, func(c *gin.Context) {
		var req CommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		operatorRole, _ := c.Get("operatorRole")
		roleStr := operatorRole.(string)

		// Policy 評估
		decision := evaluatePolicy(req.Command, roleStr)

		// 記錄決策
		decisionStr := "denied"
		if decision.Allowed {
			decisionStr = "allowed"
		}
		logCommandEvent("policy_decision", map[string]interface{}{
			"command":      req.Command,
			"operatorRole": roleStr,
			"decision":     decisionStr,
			"reason":       decision.Reason,
		})

		// 發送到 Space-SOC
		socURL := os.Getenv("SPACE_SOC_URL")
		sendEventToSOC(socURL, map[string]interface{}{
			"component":    "ttc-gateway",
			"eventType":    "policy_decision",
			"command":      req.Command,
			"operatorRole": roleStr,
			"decision":     decisionStr,
			"reason":       decision.Reason,
		})

		if !decision.Allowed {
			resp := CommandResponse{
				Status:      "denied",
				Message:     "command rejected by policy",
				Decision:    "denied",
				Reason:      decision.Reason,
				ProcessedAt: time.Now().UTC(),
			}
			c.JSON(http.StatusForbidden, resp)
			return
		}

		// 轉發到 satellite-sim
		satResp, err := forwardToSatellite(satelliteURL, req)
		if err != nil {
			logCommandEvent("forward_error", map[string]interface{}{
				"command": req.Command,
				"error":   err.Error(),
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward command to satellite"})
			return
		}

		// 記錄成功
		logCommandEvent("command_forwarded", map[string]interface{}{
			"command":      req.Command,
			"operatorRole": roleStr,
			"satelliteResponse": satResp.Status,
		})

		// 發送到 Space-SOC
		sendEventToSOC(socURL, map[string]interface{}{
			"component":    "ttc-gateway",
			"eventType":    "command_forwarded",
			"command":      req.Command,
			"operatorRole": roleStr,
			"status":       satResp.Status,
			"message":      satResp.Message,
		})

		resp := CommandResponse{
			Status:      "success",
			Message:     "command forwarded to satellite",
			Decision:    "allowed",
			Reason:      decision.Reason,
			ProcessedAt: time.Now().UTC(),
		}
		c.JSON(http.StatusOK, resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("ttc-gateway server failed: %v", err)
	}
}

