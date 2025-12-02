package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

// SignedMetadata 是最小簽章輸出格式，供 OTA / SOC 使用。
type SignedMetadata struct {
	Artefact string    `json:"artefact"`
	Digest   string    `json:"digest"`
	Signature string   `json:"signature"`
	SignedAt time.Time `json:"signedAt"`
	Signer   string    `json:"signer"`
}

func main() {
	outPath := flag.String("o", "", "輸出 JSON 檔案路徑（預設輸出到 stdout）")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sign-artifact [-o output.json] <artefact-identifier>")
		os.Exit(1)
	}

	artefact := flag.Arg(0)
	secret := os.Getenv("SIGNING_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	digestBytes := sha256.Sum256([]byte(artefact))
	digest := hex.EncodeToString(digestBytes[:])

	sigBytes := sha256.Sum256([]byte(digest + ":" + secret))
	signature := hex.EncodeToString(sigBytes[:])

	meta := SignedMetadata{
		Artefact: artefact,
		Digest:   digest,
		Signature: signature,
		SignedAt: time.Now().UTC(),
		Signer:   "local-dev-signer",
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal metadata: %v\n", err)
		os.Exit(1)
	}

	if *outPath == "" {
		fmt.Println(string(data))
		return
	}

	// 簽章工具僅允許寫入相對且不含「..」的路徑，以降低 Path Traversal 風險。
	if filepath.IsAbs(*outPath) || strings.Contains(*outPath, "..") {
		fmt.Fprintln(os.Stderr, "unsafe output path: only simple relative paths without '..' are allowed")
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write file: %v\n", err)
		os.Exit(1)
	}
}


