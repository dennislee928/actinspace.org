package ota

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// UpdateResponse 定義 OTA controller 的回應。
type UpdateResponse struct {
	Available     bool      `json:"available"`
	Version       string    `json:"version,omitempty"`
	ImageDigest   string    `json:"imageDigest,omitempty"`
	SBOMURL       string    `json:"sbomUrl,omitempty"`
	Attestation   string    `json:"attestation,omitempty"`
	Message       string    `json:"message"`
	UpdateAllowed bool      `json:"updateAllowed"`
	DenialReason  string    `json:"denialReason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Client 是 OTA 客戶端。
type Client struct {
	controllerURL  string
	component      string
	currentVersion string
	signingSecret  string
}

// NewClient 創建新的 OTA 客戶端。
func NewClient(controllerURL, component, currentVersion string) *Client {
	secret := os.Getenv("SIGNING_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	return &Client{
		controllerURL:  controllerURL,
		component:      component,
		currentVersion: currentVersion,
		signingSecret:  secret,
	}
}

// CheckForUpdates 檢查是否有可用更新。
func (c *Client) CheckForUpdates() (*UpdateResponse, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"component":      c.component,
		"currentVersion": c.currentVersion,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.controllerURL+"/api/v1/updates/check", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateResp UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, err
	}

	return &updateResp, nil
}

// VerifySignature 驗證簽章。
func (c *Client) VerifySignature(imageDigest, attestation string) (bool, error) {
	// 解析 attestation（簡化版）
	var meta struct {
		Digest    string `json:"digest"`
		Signature string `json:"signature"`
	}

	if err := json.Unmarshal([]byte(attestation), &meta); err != nil {
		return false, fmt.Errorf("無法解析 attestation: %w", err)
	}

	// 驗證 digest
	if meta.Digest != imageDigest {
		return false, fmt.Errorf("digest mismatch")
	}

	// 重新計算簽章
	sigBytes := sha256.Sum256([]byte(meta.Digest + ":" + c.signingSecret))
	expectedSignature := hex.EncodeToString(sigBytes[:])

	if meta.Signature != expectedSignature {
		return false, fmt.Errorf("signature verification failed")
	}

	return true, nil
}

// ApplyUpdate 應用更新（模擬）。
func (c *Client) ApplyUpdate(updateResp *UpdateResponse) error {
	log.Printf("開始應用更新: %s -> %s", c.currentVersion, updateResp.Version)

	// 驗證簽章
	if updateResp.Attestation != "" {
		valid, err := c.VerifySignature(updateResp.ImageDigest, updateResp.Attestation)
		if err != nil || !valid {
			return fmt.Errorf("簽章驗證失敗: %v", err)
		}
		log.Println("✅ 簽章驗證通過")
	}

	// 模擬下載和應用更新
	log.Printf("下載映像檔: %s", updateResp.ImageDigest)
	time.Sleep(1 * time.Second) // 模擬下載時間

	// 實際環境中，這裡會：
	// 1. 下載新映像檔
	// 2. 驗證 SBOM policy
	// 3. 重啟服務或熱更新

	log.Println("✅ 更新應用成功")
	c.currentVersion = updateResp.Version

	return nil
}

// StartUpdateLoop 啟動週期性更新檢查。
func (c *Client) StartUpdateLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("OTA client 已啟動，每 %v 檢查一次更新", interval)

	for range ticker.C {
		updateResp, err := c.CheckForUpdates()
		if err != nil {
			log.Printf("檢查更新失敗: %v", err)
			continue
		}

		if !updateResp.Available {
			log.Printf("無可用更新: %s", updateResp.Message)
			continue
		}

		if !updateResp.UpdateAllowed {
			log.Printf("更新被拒絕: %s", updateResp.DenialReason)
			continue
		}

		log.Printf("發現新版本: %s", updateResp.Version)

		if err := c.ApplyUpdate(updateResp); err != nil {
			log.Printf("應用更新失敗: %v", err)
			continue
		}

		log.Printf("成功更新到版本: %s", updateResp.Version)
	}
}

