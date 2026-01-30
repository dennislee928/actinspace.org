package storage

import (
	"context"
	"fmt"
	"io"
)

// NoopStorage 不實際寫入的實作；S3 未設定時可用，避免 Fetcher 因缺少儲存而失敗。
// Put 會回傳錯誤，呼叫端應檢查 ObjectStorage 是否為 NoopStorage 再決定是否下載。
type NoopStorage struct{}

// Put 回傳錯誤，表示未設定實際儲存。
func (NoopStorage) Put(ctx context.Context, bucket, key string, body io.Reader, contentType string) (*PutResult, error) {
	return nil, fmt.Errorf("storage: object storage not configured (set S3_* or provide S3Config)")
}
