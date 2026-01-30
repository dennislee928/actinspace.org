package integrations

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	clmsBaseURL = "https://land.copernicus.eu/api"
)

// CLMSDatasets 代理 GET land.copernicus.eu/api/@search 列出資料集。
// 支援 b_start、b_size 分頁；文件未要求必帶 token，先不帶；若實測需 token 可改為從 env 讀取。
func CLMSDatasets() gin.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}
	return func(c *gin.Context) {
		u, err := url.Parse(clmsBaseURL + "/@search")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid CLMS URL"})
			return
		}
		q := u.Query()
		q.Set("portal_type", "DataSet")
		q.Set("metadata_fields", "UID")
		q.Set("metadata_fields", "dataset_full_format")
		q.Set("metadata_fields", "dataset_download_information")
		if bStart := c.Query("b_start"); bStart != "" {
			q.Set("b_start", bStart)
		}
		if bSize := c.Query("b_size"); bSize != "" {
			q.Set("b_size", bSize)
		}
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
			return
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "CLMS request failed", "detail": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read CLMS response"})
			return
		}

		for k, vv := range resp.Header {
			if k == "Content-Type" || k == "Content-Length" {
				for _, v := range vv {
					c.Header(k, v)
				}
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

// CLMSDataRequest 代理 POST land.copernicus.eu/api/@datarequest_post。
// Body 原樣轉發；Header 從環境變數 CLMS_BEARER_TOKEN 加上 Authorization: Bearer <token>。
// 未設定 token 時回傳 503 並說明需註冊 land.copernicus.eu。
func CLMSDataRequest() gin.HandlerFunc {
	client := &http.Client{Timeout: 60 * time.Second}
	return func(c *gin.Context) {
		token := os.Getenv("CLMS_BEARER_TOKEN")
		if token == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "CLMS download not configured",
				"message": "需在 land.copernicus.eu 註冊並取得 Bearer token；請設定環境變數 CLMS_BEARER_TOKEN",
			})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		req, err := http.NewRequest(http.MethodPost, clmsBaseURL+"/@datarequest_post", bytes.NewReader(body))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "CLMS request failed", "detail": err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read CLMS response"})
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
	}
}
