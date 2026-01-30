package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config S3-compatible 儲存設定（MinIO / R2 / S3）
type S3Config struct {
	Endpoint        string // 例: play.min.io 或 s3.amazonaws.com
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	// PublicBaseURL 若設為非空，PutResult.StorageURL 使用此基底 + key；否則用內部位址。
	PublicBaseURL string
}

// S3Storage 實作 ObjectStorage，適用 MinIO、R2、S3。
type S3Storage struct {
	client *minio.Client
	bucket string
	baseURL string
}

// NewS3 從 S3Config 建立 S3Storage。若 config 為 nil，會從環境變數讀取：
// S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_BUCKET, S3_USE_SSL, S3_PUBLIC_BASE_URL
func NewS3(config *S3Config) (*S3Storage, error) {
	if config == nil {
		config = &S3Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			Bucket:          os.Getenv("S3_BUCKET"),
			UseSSL:          strings.EqualFold(os.Getenv("S3_USE_SSL"), "true"),
			PublicBaseURL:   os.Getenv("S3_PUBLIC_BASE_URL"),
		}
	}
	if config.Bucket == "" {
		return nil, fmt.Errorf("storage: S3_BUCKET is required")
	}
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: create minio client: %w", err)
	}
	baseURL := config.PublicBaseURL
	if baseURL == "" && config.Endpoint != "" {
		scheme := "https"
		if !config.UseSSL {
			scheme = "http"
		}
		baseURL = fmt.Sprintf("%s://%s/%s", scheme, config.Endpoint, config.Bucket)
	}
	return &S3Storage{client: client, bucket: config.Bucket, baseURL: baseURL}, nil
}

// Put 上傳 body 到 bucket/key；contentType 可為空。size 未知時傳 -1。
func (s *S3Storage) Put(ctx context.Context, bucket, key string, body io.Reader, contentType string) (*PutResult, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	var objectSize int64 = -1
	if r, ok := body.(interface{ Len() int }); ok {
		objectSize = int64(r.Len())
	}
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	info, err := s.client.PutObject(ctx, bucket, key, body, objectSize, opts)
	if err != nil {
		return nil, err
	}
	url := s.baseURL
	if url != "" {
		url = strings.TrimSuffix(url, "/") + "/" + key
	} else {
		url = key
	}
	out := &PutResult{StorageURL: url, SizeBytes: info.Size}
	if info.ETag != "" {
		out.Checksum = strings.Trim(info.ETag, "\"")
	}
	return out, nil
}
