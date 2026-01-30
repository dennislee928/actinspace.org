package integrations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPatentsSearch_NoParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/patents/search", nil)
	PatentsSearch()(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("patents/search no params: got %d, want 400", w.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if _, ok := out["error"]; !ok {
		t.Errorf("expected error field: %+v", out)
	}
}

func TestPatentsSearch_WithQ_NoCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/patents/search?q=test", nil)
	PatentsSearch()(c)

	// 未設定 EPO 憑證應回 503
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("patents/search with q, no creds: got %d, want 503", w.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if out["error"] != "EPO OPS not configured" {
		t.Errorf("expected EPO OPS not configured: %+v", out)
	}
}
