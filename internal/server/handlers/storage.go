package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/openprx/fenfa/internal/s3"
	"github.com/openprx/fenfa/internal/store"
	"gorm.io/gorm"
)

// s3StorageClient is the subset of S3 operations Fenfa needs for uploads.
type s3StorageClient interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	Delete(ctx context.Context, key string) error
}

// newS3StorageClient can be replaced in tests with a fake client.
var newS3StorageClient = func(settings store.SystemSettings) s3StorageClient {
	return s3.NewClient(settings.S3Endpoint, settings.S3Bucket, settings.S3AccessKey, settings.S3SecretKey)
}

func isS3StoragePath(storagePath string) bool {
	return strings.HasPrefix(storagePath, "s3://")
}

func s3KeyFromStoragePath(storagePath string) string {
	return strings.TrimPrefix(storagePath, "s3://")
}

func deleteS3Object(ctx context.Context, db *gorm.DB, storagePath string) error {
	key := s3KeyFromStoragePath(storagePath)

	var settings store.SystemSettings
	if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
		return fmt.Errorf("load s3 settings: %w", err)
	}
	if settings.S3Endpoint == "" || settings.S3Bucket == "" || settings.S3AccessKey == "" || settings.S3SecretKey == "" {
		return fmt.Errorf("s3 settings incomplete")
	}

	return newS3StorageClient(settings).Delete(ctx, key)
}

func removeLocalReleaseStaging(destPath, appDir string) {
	_ = os.Remove(destPath)
	_ = os.Remove(appDir)
}

// deleteReleaseFiles removes release payloads after database deletion succeeds.
func deleteReleaseFiles(db *gorm.DB, releases []store.Release) {
	for _, r := range releases {
		if r.StoragePath == "" {
			continue
		}
		if isS3StoragePath(r.StoragePath) {
			if err := deleteS3Object(context.Background(), db, r.StoragePath); err != nil {
				log.Printf("WARN: failed to delete s3 object %s: %v", r.StoragePath, err)
			}
			continue
		}
		if err := os.Remove(r.StoragePath); err != nil && !os.IsNotExist(err) {
			log.Printf("WARN: failed to delete file %s: %v", r.StoragePath, err)
		}
	}
}
