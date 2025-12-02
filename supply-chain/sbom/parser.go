package sbom

import (
	"encoding/json"
	"fmt"
	"os"
)

// CycloneDX 定義 CycloneDX SBOM 的簡化結構。
type CycloneDX struct {
	BOMFormat   string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Metadata    Metadata    `json:"metadata"`
	Components  []Component `json:"components"`
}

// Metadata 定義 SBOM 元資料。
type Metadata struct {
	Timestamp string    `json:"timestamp"`
	Component Component `json:"component"`
}

// Component 定義軟體組件。
type Component struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Purl       string          `json:"purl,omitempty"`
	Properties []Property      `json:"properties,omitempty"`
	Licenses   []License       `json:"licenses,omitempty"`
	Hashes     []Hash          `json:"hashes,omitempty"`
}

// Property 定義組件屬性。
type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// License 定義授權資訊。
type License struct {
	License LicenseInfo `json:"license"`
}

// LicenseInfo 定義授權詳情。
type LicenseInfo struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Hash 定義檔案雜湊。
type Hash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

// PolicyViolation 定義 SBOM policy 違規。
type PolicyViolation struct {
	Severity    string `json:"severity"` // "low", "medium", "high", "critical"
	Component   string `json:"component"`
	Version     string `json:"version"`
	Reason      string `json:"reason"`
	Description string `json:"description"`
}

// PolicyResult 定義 policy 檢查結果。
type PolicyResult struct {
	Allowed    bool              `json:"allowed"`
	Violations []PolicyViolation `json:"violations"`
	Summary    string            `json:"summary"`
}

// ParseSBOM 解析 CycloneDX SBOM 檔案。
func ParseSBOM(filePath string) (*CycloneDX, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("無法讀取 SBOM 檔案: %w", err)
	}

	var sbom CycloneDX
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, fmt.Errorf("無法解析 SBOM: %w", err)
	}

	return &sbom, nil
}

// CheckPolicy 檢查 SBOM 是否符合 policy。
func CheckPolicy(sbom *CycloneDX) PolicyResult {
	var violations []PolicyViolation

	// Policy 1: 禁止已知有漏洞的套件（簡化版，實際應查詢漏洞資料庫）
	vulnerablePackages := map[string]string{
		"lodash@4.17.15":    "CVE-2020-8203: Prototype Pollution",
		"axios@0.18.0":      "CVE-2019-10742: SSRF",
		"express@4.16.0":    "CVE-2022-24999: Open Redirect",
	}

	for _, comp := range sbom.Components {
		key := fmt.Sprintf("%s@%s", comp.Name, comp.Version)
		if vuln, exists := vulnerablePackages[key]; exists {
			violations = append(violations, PolicyViolation{
				Severity:    "high",
				Component:   comp.Name,
				Version:     comp.Version,
				Reason:      "known_vulnerability",
				Description: vuln,
			})
		}
	}

	// Policy 2: 禁止某些高風險授權
	restrictedLicenses := map[string]bool{
		"AGPL-3.0": true,
		"GPL-3.0":  true,
	}

	for _, comp := range sbom.Components {
		for _, lic := range comp.Licenses {
			if restrictedLicenses[lic.License.ID] {
				violations = append(violations, PolicyViolation{
					Severity:    "medium",
					Component:   comp.Name,
					Version:     comp.Version,
					Reason:      "restricted_license",
					Description: fmt.Sprintf("License %s is restricted", lic.License.ID),
				})
			}
		}
	}

	// Policy 3: 檢查組件數量（異常大量依賴可能是供應鏈攻擊）
	if len(sbom.Components) > 500 {
		violations = append(violations, PolicyViolation{
			Severity:    "medium",
			Component:   "SBOM",
			Version:     "",
			Reason:      "excessive_dependencies",
			Description: fmt.Sprintf("SBOM contains %d components (threshold: 500)", len(sbom.Components)),
		})
	}

	allowed := len(violations) == 0
	summary := fmt.Sprintf("SBOM policy check: %d violations found", len(violations))
	if allowed {
		summary = "SBOM policy check: passed"
	}

	return PolicyResult{
		Allowed:    allowed,
		Violations: violations,
		Summary:    summary,
	}
}

