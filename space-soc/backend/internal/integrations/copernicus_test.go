package integrations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCopernicusWMTSConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/copernicus/wmts-config", nil)

	CopernicusWMTSConfig()(c)

	if w.Code != http.StatusOK {
		t.Fatalf("wmts-config status: got %d, want 200", w.Code)
	}
	var resp WMTSConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(resp.Layers) == 0 {
		t.Fatal("layers empty")
	}
	var found bool
	for _, l := range resp.Layers {
		if l.ID == "s2gm" && l.Type == "wms" && l.URL != "" && l.Layer == "S2GM" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected s2gm WMS layer in %+v", resp.Layers)
	}
	if resp.Note == "" {
		t.Error("expected non-empty note")
	}
}
