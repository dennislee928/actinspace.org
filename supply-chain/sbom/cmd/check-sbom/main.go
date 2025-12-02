package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"actinspace.org/supply-chain/sbom"
)

func main() {
	sbomFile := flag.String("sbom", "", "SBOM 檔案路徑（必填）")
	jsonOutput := flag.Bool("json", false, "以 JSON 格式輸出結果")
	flag.Parse()

	if *sbomFile == "" {
		fmt.Fprintln(os.Stderr, "錯誤: 必須指定 SBOM 檔案 (-sbom)")
		flag.Usage()
		os.Exit(1)
	}

	// 解析 SBOM
	sbomData, err := sbom.ParseSBOM(*sbomFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: %v\n", err)
		os.Exit(1)
	}

	// 檢查 policy
	result := sbom.CheckPolicy(sbomData)

	if *jsonOutput {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("SBOM Policy 檢查結果\n")
		fmt.Printf("==================\n\n")
		fmt.Printf("組件數量: %d\n", len(sbomData.Components))
		fmt.Printf("Policy 狀態: ")
		if result.Allowed {
			fmt.Printf("✅ 通過\n")
		} else {
			fmt.Printf("❌ 失敗\n")
		}
		fmt.Printf("違規數量: %d\n\n", len(result.Violations))

		if len(result.Violations) > 0 {
			fmt.Printf("違規詳情:\n")
			for i, v := range result.Violations {
				fmt.Printf("%d. [%s] %s@%s\n", i+1, v.Severity, v.Component, v.Version)
				fmt.Printf("   原因: %s\n", v.Reason)
				fmt.Printf("   說明: %s\n\n", v.Description)
			}
		}
	}

	if !result.Allowed {
		os.Exit(1)
	}
}

