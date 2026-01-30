// Package integrations：CLMS JWT 認證（Service Key → Bearer token）。
// 依 EEA CLMS API 文件：用 private key 簽 RS256 JWT，POST 到 @@oauth2-token 換取 access_token。

package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	clmsGrantTypeJWTBearer = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	defaultCLMSTokenURI    = "https://land.copernicus.eu/@@oauth2-token"
	jwtExpiry              = 55 * time.Minute // 略短於 1h，避免邊界過期
)

// CLMSTokenResponse 為 @@oauth2-token 回傳的 JSON。
type CLMSTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// CLMSAuth 從環境變數讀取 Service Key，負責簽 JWT 與換 Bearer token。
// 環境變數：CLMS_CLIENT_ID, CLMS_PRIVATE_KEY, CLMS_TOKEN_URI（可選）, CLMS_USER_ID。
type CLMSAuth struct {
	ClientID  string
	PrivateKey []byte
	TokenURI  string
	UserID    string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// NewCLMSAuthFromEnv 從環境變數建立 CLMSAuth；若未設定完整 Service Key 則回傳 nil, nil（呼叫端可改回退 CLMS_BEARER_TOKEN）。
func NewCLMSAuthFromEnv() (*CLMSAuth, error) {
	clientID := os.Getenv("CLMS_CLIENT_ID")
	privateKeyPEM := os.Getenv("CLMS_PRIVATE_KEY")
	tokenURI := os.Getenv("CLMS_TOKEN_URI")
	if tokenURI == "" {
		tokenURI = defaultCLMSTokenURI
	}
	userID := os.Getenv("CLMS_USER_ID")

	if clientID == "" || privateKeyPEM == "" || userID == "" {
		return nil, nil
	}

	// 還原 PEM 中的 \n（若從 JSON/env 貼上時被轉成字面 \n）
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\\n", "\n")
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("CLMS private key: %w", err)
	}
	_ = key // 僅驗證可解析，實際簽名時再 Parse 一次以保持無狀態

	return &CLMSAuth{
		ClientID:   clientID,
		PrivateKey: []byte(privateKeyPEM),
		TokenURI:   tokenURI,
		UserID:     userID,
	}, nil
}

// Valid 回報是否已設定完整 Service Key（可取得 token）。
func (a *CLMSAuth) Valid() bool {
	return a != nil && a.ClientID != "" && len(a.PrivateKey) > 0 && a.UserID != ""
}

// BearerToken 回傳當前有效的 Bearer token；若過期或尚未取得則先換 token 再回傳。
func (a *CLMSAuth) BearerToken(httpClient *http.Client) (string, error) {
	if !a.Valid() {
		return "", fmt.Errorf("CLMS auth not configured")
	}

	a.mu.Lock()
	if a.token != "" && time.Now().Before(a.expiresAt) {
		tok := a.token
		a.mu.Unlock()
		return tok, nil
	}
	a.mu.Unlock()

	tok, expiresAt, err := a.exchangeToken(httpClient)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.token = tok
	a.expiresAt = expiresAt
	a.mu.Unlock()
	return tok, nil
}

func (a *CLMSAuth) exchangeToken(httpClient *http.Client) (accessToken string, expiresAt time.Time, err error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(a.PrivateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse CLMS private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": a.ClientID,
		"aud": a.TokenURI,
		"sub": a.UserID,
		"iat": now.Unix(),
		"exp": now.Add(jwtExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	raw, err := token.SignedString(key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", clmsGrantTypeJWTBearer)
	form.Set("assertion", raw)
	req, err := http.NewRequest(http.MethodPost, a.TokenURI, bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	var tr CLMSTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", time.Time{}, fmt.Errorf("decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}
	if tr.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("empty access_token in response")
	}

	expIn := time.Duration(tr.ExpiresIn) * time.Second
	if expIn <= 0 {
		expIn = time.Hour
	}
	expiresAt = time.Now().Add(expIn)
	return tr.AccessToken, expiresAt, nil
}
