package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CheckLicenseMiddleware enforces that a valid license key is present.
// This simulates the "Economic Model" requirement for the ActInSpace challenge.
func CheckLicenseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a real scenario, this would verify a cryptographic signature.
		// For the prototype/scoring demonstration, we check for a specific env var.
		licenseKey := os.Getenv("ACTINSPACE_LICENSE_KEY")

		if licenseKey == "" {
			// Free Tier / Unlicensed
			// We inject a header to indicate status
			c.Writer.Header().Set("X-License-Status", "FREE_TIER")
			
			// Block access to "Enterprise" features
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/enterprise") {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Enterprise license required to access this feature.",
					"link":  "https://actinspace.org/buy-license",
				})
				c.Abort()
				return
			}
		} else {
			// Valid License (Simulated)
			c.Writer.Header().Set("X-License-Status", "ENTERPRISE")
		}

		c.Next()
	}
}
