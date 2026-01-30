package storage

import "os"

// FromEnv 依環境變數建立 ObjectStorage。
// 若 S3_BUCKET 已設定則建立 S3Storage（MinIO/R2/S3）；否則回傳 NoopStorage。
func FromEnv() (ObjectStorage, error) {
	if os.Getenv("S3_BUCKET") != "" {
		return NewS3(nil)
	}
	return &NoopStorage{}, nil
}
