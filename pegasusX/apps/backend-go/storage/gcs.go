package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

var (
	// Client is nil in local dev when GCS is unavailable; upload tickets fall back to placeholders.
	Client     *storage.Client
	BucketName string
)

// InitGCS boots the storage client. An empty bucket name skips client init and
// leaves Client nil so GenerateUploadTicket can return dev placeholders.
func InitGCS(ctx context.Context, bucket string) error {
	BucketName = strings.TrimSpace(bucket)
	if BucketName == "" {
		return nil
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("init gcs client: %w", err)
	}
	Client = client
	return nil
}

// GenerateUploadTicket creates a short-lived signed PUT URL for direct client uploads.
func GenerateUploadTicket(supplierID, extension string) (uploadURL string, publicURL string, err error) {
	filename := fmt.Sprintf("%s-%d.%s", supplierID, time.Now().UnixNano(), extension)
	objectName := fmt.Sprintf("catalog/%s/%s", supplierID, filename)

	if Client == nil {
		placeholder := fmt.Sprintf("https://placehold.co/400x400/1a1a2e/e0e0e0?text=%s", filename)
		return placeholder, placeholder, nil
	}

	mimeTypes := map[string]string{
		"jpg": "image/jpeg", "jpeg": "image/jpeg",
		"png": "image/png", "webp": "image/webp",
	}
	contentType := mimeTypes[extension]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	opts := &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     time.Now().Add(15 * time.Minute),
		ContentType: contentType,
	}

	url, err := Client.Bucket(BucketName).SignedURL(objectName, opts)
	if err != nil {
		return "", "", fmt.Errorf("sign upload url: %w", err)
	}

	public := fmt.Sprintf("https://storage.googleapis.com/%s/%s", BucketName, objectName)
	return url, public, nil
}
