package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"actinspace.org/space-soc/backend/internal/integrations"

	"github.com/gin-gonic/gin"
)

// testRouter 建立僅含新 API 的路由，不呼叫 initDB，供測試使用。
func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/resources", getResourcesHandler())
	r.GET("/api/v1/copernicus/wmts-config", integrations.CopernicusWMTSConfig())
	r.GET("/api/v1/clms/datasets", integrations.CLMSDatasets())
	r.POST("/api/v1/clms/datarequest", integrations.CLMSDataRequest())
	r.GET("/api/v1/patents/search", integrations.PatentsSearch())
	return r
}

func TestGETResources_ACRI_STBlock(t *testing.T) {
	r := testRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/resources: got %d, want 200", w.Code)
	}
	var body struct {
		Categories []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Items []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"items"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	var acriStItems []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	for i := range body.Categories {
		if body.Categories[i].ID == "acri_st" {
			acriStItems = body.Categories[i].Items
			break
		}
	}
	if acriStItems == nil {
		t.Fatal("categories: missing acri_st")
	}
	titles := make(map[string]bool)
	for _, it := range acriStItems {
		titles[it.Title] = true
	}
	if !titles["Copernicus Marine"] {
		t.Error("acri_st items: missing Copernicus Marine")
	}
	if !titles["OCDB"] {
		t.Error("acri_st items: missing OCDB")
	}
	if !titles["ACRI-ST #1"] {
		t.Error("acri_st items: missing ACRI-ST #1")
	}
}

func TestGETCopernicusWMTSConfig(t *testing.T) {
	r := testRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/copernicus/wmts-config", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/copernicus/wmts-config: got %d, want 200", w.Code)
	}
	var body struct {
		Layers []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(body.Layers) == 0 {
		t.Fatal("wmts-config: layers empty")
	}
	var found bool
	for _, l := range body.Layers {
		if l.ID == "s2gm" && l.Type == "wms" && l.URL != "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("wmts-config: expected s2gm wms layer in %+v", body.Layers)
	}
}

func TestPOSTClmsDatarequest_NoToken(t *testing.T) {
	r := testRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clms/datarequest", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/v1/clms/datarequest (no token): got %d, want 503", w.Code)
	}
}

func TestGETPatentsSearch_NoParams(t *testing.T) {
	r := testRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/patents/search", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("GET /api/v1/patents/search (no params): got %d, want 400", w.Code)
	}
}

func TestGETPatentsSearch_NoCredentials(t *testing.T) {
	r := testRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/patents/search?q=test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/v1/patents/search (q=test, no creds): got %d, want 503", w.Code)
	}
}
