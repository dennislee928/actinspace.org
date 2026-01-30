package integrations

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	epoOpsBaseURL = "https://ops.epo.org/3.2/rest-services"
)

// PatentsClient 用於代理 EPO Open Patent Services (OPS) 搜尋。
type PatentsClient struct {
	consumerKey    string
	consumerSecret string
	client         *http.Client
}

// NewPatentsClient 從環境變數 EPO_OPS_CONSUMER_KEY、EPO_OPS_CONSUMER_SECRET 建立 client；若未設定則回傳 nil。
func NewPatentsClient() *PatentsClient {
	key := os.Getenv("EPO_OPS_CONSUMER_KEY")
	secret := os.Getenv("EPO_OPS_CONSUMER_SECRET")
	if key == "" || secret == "" {
		return nil
	}
	return &PatentsClient{
		consumerKey:    key,
		consumerSecret: secret,
		client:         &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured 回傳是否已設定 EPO OPS 憑證。
func (p *PatentsClient) IsConfigured() bool {
	return p != nil
}

// Search 以 CQL 查詢 EPO OPS published-data search；q 為查詢字串（如 "applicant:Airbus Defence"），raw 回傳原始 XML。
func (p *PatentsClient) Search(q string) (body []byte, contentType string, err error) {
	if p == nil {
		return nil, "", fmt.Errorf("EPO OPS not configured: set EPO_OPS_CONSUMER_KEY and EPO_OPS_CONSUMER_SECRET")
	}
	u := epoOpsBaseURL + "/published-data/search?q=" + url.QueryEscape(q)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(p.consumerKey + ":" + p.consumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("EPO OPS returned %d: %s", resp.StatusCode, string(b))
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	return body, contentType, nil
}

// SearchFromParams 依關鍵字 q 與 applicant 組成 CQL 並呼叫 Search。
func (p *PatentsClient) SearchFromParams(q, applicant string) (body []byte, contentType string, err error) {
	var parts []string
	if strings.TrimSpace(q) != "" {
		parts = append(parts, "("+q+")")
	}
	if strings.TrimSpace(applicant) != "" {
		parts = append(parts, "applicant:("+applicant+")")
	}
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("至少需提供 q 或 applicant")
	}
	cql := strings.Join(parts, " AND ")
	return p.Search(cql)
}
