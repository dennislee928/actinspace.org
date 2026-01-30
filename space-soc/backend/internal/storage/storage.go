package storage

import (
	"context"
	"io"
)

// PutResult 上傳結果：storage_url、可選 checksum 與 size。
type PutResult struct {
	StorageURL string
	Checksum   string
	SizeBytes  int64
}

// ObjectStorage 物件儲存抽象（上傳 bytes/stream，回傳 URL、checksum、size）。
// 實作可綁 S3-compatible（MinIO、R2、S3）。
type ObjectStorage interface {
	// Put 上傳 body 到 bucket/key，回傳可存取的 URL 與可選 checksum、size。
	Put(ctx context.Context, bucket, key string, body io.Reader, contentType string) (*PutResult, error)
}
