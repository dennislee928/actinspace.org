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
)

// CommandRequest 定義要發送的指令格式。
type CommandRequest struct {
	Command     string                 `json:"command"`
	Params      map[string]interface{} `json:"params,omitempty"`
	SatelliteID string                 `json:"satelliteId,omitempty"`
}

// CommandResponse 是 gateway 的回應格式。
type CommandResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Decision    string `json:"decision"`
	Reason      string `json:"reason,omitempty"`
	ProcessedAt string `json:"processedAt"`
}

func main() {
	gatewayURL := flag.String("gateway", "http://localhost:8081", "TT&C Gateway URL")
	command := flag.String("cmd", "", "指令名稱（必填）")
	token := flag.String("token", "operator-token", "認證 token（預設: operator-token）")
	satelliteID := flag.String("satellite", "", "衛星 ID（選填）")
	flag.Parse()

	if *command == "" {
		fmt.Fprintf(os.Stderr, "錯誤: 必須指定指令 (-cmd)\n")
		flag.Usage()
		os.Exit(1)
	}

	// 驗證 gateway URL（防止 SSRF）
	gatewayURLStr := strings.TrimSpace(*gatewayURL)
	parsedURL, err := url.Parse(gatewayURLStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無效的 gateway URL: %v\n", err)
		os.Exit(1)
	}
	
	// 只允許 http/https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		fmt.Fprintf(os.Stderr, "錯誤: Gateway URL 必須使用 http:// 或 https://\n")
		os.Exit(1)
	}
	
	// 嚴格驗證 host（只允許 localhost、127.0.0.1 或私有網路）
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
		fmt.Fprintf(os.Stderr, "錯誤: Gateway URL 必須指向 localhost 或私有網路 (目前: %s)\n", host)
		os.Exit(1)
	}

	req := CommandRequest{
		Command:     *command,
		SatelliteID: *satelliteID,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法序列化請求: %v\n", err)
		os.Exit(1)
	}

	httpReq, err := http.NewRequest("POST", gatewayURLStr+"/command", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法建立請求: %v\n", err)
		os.Exit(1)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+*token)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法發送請求: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法讀取回應: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "錯誤: Gateway 回應狀態碼 %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "回應內容: %s\n", string(body))
		os.Exit(1)
	}

	var cmdResp CommandResponse
	if err := json.Unmarshal(body, &cmdResp); err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: 無法解析回應: %v\n", err)
		fmt.Fprintf(os.Stderr, "原始回應: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Printf("指令發送成功！\n")
	fmt.Printf("狀態: %s\n", cmdResp.Status)
	fmt.Printf("決策: %s\n", cmdResp.Decision)
	if cmdResp.Reason != "" {
		fmt.Printf("原因: %s\n", cmdResp.Reason)
	}
	fmt.Printf("處理時間: %s\n", cmdResp.ProcessedAt)
}

