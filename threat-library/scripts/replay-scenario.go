package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario 定義威脅場景的結構。
type Scenario struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Objectives  []string               `yaml:"objectives"`
	Playbook    Playbook               `yaml:"playbook_steps"`
	Severity    string                 `yaml:"severity"`
	Metadata    map[string]interface{} `yaml:",inline"`
}

// Playbook 定義場景的執行步驟。
type Playbook struct {
	Steps []string `yaml:"steps"`
}

func main() {
	scenarioFile := flag.String("scenario", "", "威脅場景 YAML 檔案路徑（必填）")
	gatewayURL := flag.String("gateway", "http://localhost:8081", "TT&C Gateway URL")
	token := flag.String("token", "operator-token", "認證 token")
	delay := flag.Duration("delay", 2*time.Second, "步驟之間的延遲時間")
	flag.Parse()

	if *scenarioFile == "" {
		fmt.Fprintf(os.Stderr, "錯誤: 必須指定場景檔案 (-scenario)\n")
		flag.Usage()
		os.Exit(1)
	}

	// 驗證檔案路徑（防止 Path Traversal）
	scenarioPath := strings.TrimSpace(*scenarioFile)
	if strings.Contains(scenarioPath, "..") || strings.HasPrefix(scenarioPath, "/") {
		fmt.Fprintf(os.Stderr, "錯誤: 無效的場景檔案路徑\n")
		os.Exit(1)
	}
	// 確保路徑在 threat-library/scenarios/ 目錄內
	if !strings.HasPrefix(scenarioPath, "threat-library/scenarios/") {
		scenarioPath = "threat-library/scenarios/" + scenarioPath
	}

	// 讀取場景檔案
	data, err := os.ReadFile(scenarioPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法讀取場景檔案: %v\n", err)
		os.Exit(1)
	}

	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法解析場景檔案: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("開始重演場景: %s\n", scenario.Name)
	fmt.Printf("描述: %s\n\n", scenario.Description)

	// 根據場景 ID 執行對應的攻擊流程
	switch scenario.ID {
	case "unauthorized-dangerous-command":
		replayUnauthorizedCommand(*gatewayURL, *token, *delay)
	case "uplink-spoofing-flood":
		replayUplinkFlood(*gatewayURL, *delay)
	case "critical-phase-violation":
		replayCriticalPhaseViolation(*gatewayURL, *token, *delay)
	default:
		fmt.Printf("場景 '%s' 的重演腳本尚未實作\n", scenario.ID)
		fmt.Printf("請手動執行場景步驟\n")
	}

	fmt.Println("\n場景重演完成")
}

// validateGatewayURL 驗證 gateway URL（防止 SSRF）。
func validateGatewayURL(gatewayURL string) error {
	parsedURL, err := url.Parse(gatewayURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("無效的 gateway URL (必須是 http:// 或 https://)")
	}
	host := strings.ToLower(parsedURL.Hostname())
	allowedHosts := []string{"localhost", "127.0.0.1", "::1"}
	isPrivateIP := strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.16.") ||
		strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") ||
		strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.20.") ||
		strings.HasPrefix(host, "172.21.") ||
		strings.HasPrefix(host, "172.22.") ||
		strings.HasPrefix(host, "172.23.") ||
		strings.HasPrefix(host, "172.24.") ||
		strings.HasPrefix(host, "172.25.") ||
		strings.HasPrefix(host, "172.26.") ||
		strings.HasPrefix(host, "172.27.") ||
		strings.HasPrefix(host, "172.28.") ||
		strings.HasPrefix(host, "172.29.") ||
		strings.HasPrefix(host, "172.30.") ||
		strings.HasPrefix(host, "172.31.")

	isAllowed := false
	for _, allowed := range allowedHosts {
		if host == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed && !isPrivateIP {
		return fmt.Errorf("gateway URL 必須指向 localhost 或私有網路 (目前: %s)", host)
	}
	return nil
}

// replayUnauthorizedCommand 重演未授權危險指令場景。
func replayUnauthorizedCommand(gatewayURL, token string, delay time.Duration) {
	if err := validateGatewayURL(gatewayURL); err != nil {
		fmt.Printf("警告: %v\n", err)
		return
	}
	fmt.Println("步驟 1: 使用 operator 角色嘗試發送 deorbit 指令...")
	time.Sleep(delay)

	resp, err := sendCommand(gatewayURL, token, "deorbit", nil)
	if err != nil {
		fmt.Printf("錯誤: %v\n", err)
		return
	}

	fmt.Printf("回應: %s - %s\n", resp.Status, resp.Message)
	fmt.Printf("決策: %s\n", resp.Decision)
	if resp.Reason != "" {
		fmt.Printf("原因: %s\n", resp.Reason)
	}

	fmt.Println("\n步驟 2: 嘗試發送多個危險指令...")
	time.Sleep(delay)

	commands := []string{"disable_power", "format_memory", "orbit_change"}
	for _, cmd := range commands {
		resp, err := sendCommand(gatewayURL, token, cmd, nil)
		if err != nil {
			fmt.Printf("錯誤發送 %s: %v\n", cmd, err)
			continue
		}
		fmt.Printf("  %s: %s\n", cmd, resp.Decision)
		time.Sleep(delay / 2)
	}
}

// replayUplinkFlood 重演 uplink flood 場景。
func replayUplinkFlood(gatewayURL string, delay time.Duration) {
	if err := validateGatewayURL(gatewayURL); err != nil {
		fmt.Printf("警告: %v\n", err)
		return
	}
	fmt.Println("步驟 1: 發送未認證的請求...")
	time.Sleep(delay)

	// 嘗試未認證請求
	reqBody, _ := json.Marshal(map[string]interface{}{
		"command": "health_check",
	})
	
	resp, err := http.Post(gatewayURL+"/command", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("錯誤: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("回應狀態碼: %d\n", resp.StatusCode)

	fmt.Println("\n步驟 2: 發送大量指令（flood attack）...")
	time.Sleep(delay)

	for i := 0; i < 15; i++ {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"command": fmt.Sprintf("test_command_%d", i),
		})
		
		httpReq, _ := http.NewRequest("POST", gatewayURL+"/command", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Authorization", "Bearer operator-token")
		httpReq.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 1 * time.Second}
		client.Do(httpReq)
		
		if i%5 == 0 {
			fmt.Printf("  已發送 %d 個指令...\n", i+1)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// replayCriticalPhaseViolation 重演關鍵階段違規場景。
func replayCriticalPhaseViolation(gatewayURL, token string, delay time.Duration) {
	if err := validateGatewayURL(gatewayURL); err != nil {
		fmt.Printf("警告: %v\n", err)
		return
	}
	fmt.Println("步驟 1: 模擬關鍵任務階段...")
	fmt.Println("（注意: 實際環境中需要設定 MISSION_PHASE 環境變數）")
	time.Sleep(delay)

	fmt.Println("步驟 2: 嘗試發送非關鍵指令...")
	time.Sleep(delay)

	nonCriticalCommands := []string{"payload_toggle", "diagnostics", "system_status"}
	for _, cmd := range nonCriticalCommands {
		resp, err := sendCommand(gatewayURL, token, cmd, nil)
		if err != nil {
			fmt.Printf("錯誤: %v\n", err)
			continue
		}
		fmt.Printf("  %s: %s\n", cmd, resp.Decision)
		time.Sleep(delay / 2)
	}
}

// CommandResponse 定義指令回應格式。
type CommandResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Decision    string `json:"decision"`
	Reason      string `json:"reason"`
	ProcessedAt string `json:"processedAt"`
}

// sendCommand 發送指令到 gateway。
func sendCommand(gatewayURL, token, command string, params map[string]interface{}) (*CommandResponse, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"command": command,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", gatewayURL+"/command", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cmdResp CommandResponse
	if err := json.Unmarshal(body, &cmdResp); err != nil {
		return nil, err
	}

	return &cmdResp, nil
}

