package integrations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCLMSDataRequest_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/clms/datarequest", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	CLMSDataRequest()(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("clms/datarequest no token: got %d, want 503", w.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if out["error"] != "CLMS download not configured" {
		t.Errorf("expected CLMS download not configured: %+v", out)
	}
}

func TestNewCLMSAuthFromEnv_EmptyEnv(t *testing.T) {
	// 未設定 Service Key 時應回傳 (nil, nil)
	os.Unsetenv("CLMS_CLIENT_ID")
	os.Unsetenv("CLMS_PRIVATE_KEY")
	os.Unsetenv("CLMS_USER_ID")
	auth, err := NewCLMSAuthFromEnv()
	if err != nil {
		t.Fatalf("NewCLMSAuthFromEnv empty env: want nil error, got %v", err)
	}
	if auth != nil {
		t.Fatalf("NewCLMSAuthFromEnv empty env: want nil auth, got %+v", auth)
	}
}

func TestNewCLMSAuthFromEnv_InvalidPrivateKey(t *testing.T) {
	os.Setenv("CLMS_CLIENT_ID", "test-client")
	os.Setenv("CLMS_USER_ID", "test-user")
	os.Setenv("CLMS_PRIVATE_KEY", "not-valid-pem")
	defer func() {
		os.Unsetenv("CLMS_CLIENT_ID")
		os.Unsetenv("CLMS_USER_ID")
		os.Unsetenv("CLMS_PRIVATE_KEY")
	}()
	_, err := NewCLMSAuthFromEnv()
	if err == nil {
		t.Fatal("NewCLMSAuthFromEnv invalid PEM: want error, got nil")
	}
}
