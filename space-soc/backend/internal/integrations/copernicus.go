package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// S2GM 第三方 WMS 服務（Brockmann Consult），可供底圖顯示。
	// 來源：Copernicus Global Land / Sentinel-2 Global Mosaic；正式產品下載請使用官方 s2gm.land.copernicus.eu Mosaic Hub。
	s2gmWMSBaseURL = "https://s2gm-wms.brockmann-consult.de/"
)

// WMTSLayerConfig 單一圖層設定，供前端地圖使用。
type WMTSLayerConfig struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"`        // "wmts" | "wms"
	URL         string `json:"url"`         // WMTS capabilities 或 WMS GetMap 基底 URL
	Layer       string `json:"layer"`       // 圖層名稱（WMS 必填；WMTS 視實作）
	Description string `json:"description,omitempty"`
}

// WMTSConfigResponse GET /api/v1/copernicus/wmts-config 回傳結構。
type WMTSConfigResponse struct {
	Layers []WMTSLayerConfig `json:"layers"`
	Note   string            `json:"note,omitempty"`
}

// CopernicusWMTSConfig 回傳 Copernicus WMTS/WMS 圖層設定，包含 S2GM WMS 選項。
// 前端地圖可依此顯示 S2GM 底圖。S2GM 來源：Copernicus Global Land / 第三方 WMS。
func CopernicusWMTSConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		layers := []WMTSLayerConfig{
			{
				ID:          "s2gm",
				Label:       "S2GM (Sentinel-2 Global Mosaic)",
				Type:        "wms",
				URL:         s2gmWMSBaseURL,
				Layer:       "S2GM", // 圖層名可依實際 GetCapabilities 調整
				Description: "第三方 WMS 底圖；來源 Copernicus Global Land。正式產品請至 s2gm.land.copernicus.eu",
			},
		}
		c.JSON(http.StatusOK, WMTSConfigResponse{
			Layers: layers,
			Note:   "S2GM 來源：Copernicus Global Land Service / 第三方 WMS。官方 Mosaic Hub: https://s2gm.land.copernicus.eu/",
		})
	}
}
