// Package s2gm：S2GM Mosaic Downloader 完成產品監控；掃描目錄後上傳至 Object Storage 並寫入 source_job/asset。
// 不取代 Mosaic Downloader App，僅做「完成後入庫」。

package s2gm

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"actinspace.org/space-soc/backend/internal/model"
	"actinspace.org/space-soc/backend/internal/storage"
	"gorm.io/gorm"
)

const (
	defaultS2GMWatchInterval = 1 * time.Minute
	processedSubdir          = "processed"
)

// RunS2GMWatcher 週期掃描 watchDir 內新檔案，上傳至 objStore 並建立 source_job（source=s2gm）+ asset。
// 處理過的檔案會移至 watchDir/processed/。watchDir 為空時不啟動。
func RunS2GMWatcher(ctx context.Context, db *gorm.DB, objStore storage.ObjectStorage, watchDir string) {
	if watchDir == "" {
		log.Println("[s2gm] S2GM_WATCH_DIR not set, S2GM watcher disabled")
		return
	}
	if objStore == nil {
		log.Println("[s2gm] object storage not configured, S2GM watcher disabled")
		return
	}

	interval := defaultS2GMWatchInterval
	if v := os.Getenv("S2GM_WATCH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runS2GMPoll(ctx, db, objStore, watchDir)
		}
	}
}

func runS2GMPoll(ctx context.Context, db *gorm.DB, objStore storage.ObjectStorage, watchDir string) {
	entries, err := os.ReadDir(watchDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[s2gm] read dir %s: %v", watchDir, err)
		}
		return
	}

	processedDir := filepath.Join(watchDir, processedSubdir)
	_ = os.MkdirAll(processedDir, 0755)

	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fpath := filepath.Join(watchDir, e.Name())
		if err := ingestOneFile(ctx, db, objStore, fpath, processedDir); err != nil {
			log.Printf("[s2gm] ingest %s: %v", fpath, err)
		}
	}
}

func ingestOneFile(ctx context.Context, db *gorm.DB, objStore storage.ObjectStorage, fpath, processedDir string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	filename := filepath.Base(fpath)
	contentType := mimeByExt(filepath.Ext(filename))

	// 先建立 source_job（s2gm, completed），再上傳並寫 asset
	job := model.SourceJob{
		Source: "s2gm",
		Status: "completed",
	}
	if err := db.WithContext(ctx).Create(&job).Error; err != nil {
		return err
	}

	key := "s2gm/" + strconv.FormatUint(uint64(job.ID), 10) + "/" + filename

	putRes, err := objStore.Put(ctx, "", key, f, contentType)
	if err != nil {
		db.WithContext(ctx).Delete(&job)
		return err
	}

	size := putRes.SizeBytes
	if size == 0 {
		size = info.Size()
	}
	asset := model.Asset{
		JobID:      job.ID,
		Filename:   filename,
		Mime:       contentType,
		StorageURL: putRes.StorageURL,
		Checksum:   putRes.Checksum,
		SizeBytes:  &size,
	}
	if err := db.WithContext(ctx).Create(&asset).Error; err != nil {
		return err
	}

	// 移至 processed 避免重複處理
	dest := filepath.Join(processedDir, filename)
	if err := os.Rename(fpath, dest); err != nil {
		log.Printf("[s2gm] move to processed failed: %v", err)
	}
	return nil
}

func mimeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".tif", ".tiff":
		return "image/tiff"
	case ".jp2", ".j2k":
		return "image/jp2"
	case ".zip":
		return "application/zip"
	case ".nc":
		return "application/netcdf"
	default:
		return "application/octet-stream"
	}
}
