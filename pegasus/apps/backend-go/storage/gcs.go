package storage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
)

var Client *storage.Client
var BucketName string

// InitGCS boots the storage client
func InitGCS(ctx context.Context, bucket string) error {
	var err error
	Client, err = storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to init GCS client: %w", err)
	}
	BucketName = bucket
	// Check if bucket is empty strings to prevent cryptic errors
	if BucketName == "" {
		return fmt.Errorf("GCS bucket name cannot be empty")
	}
	return nil
}

// GenerateUploadTicket creates a 15-minute cryptographic URL for direct Next.js uploads.
// In local dev (GCS offline), returns a placeholder so product creation isn't blocked.
func GenerateUploadTicket(supplierId string, extension string) (string, string, error) {
	filename := fmt.Sprintf("%s-%d.%s", supplierId, time.Now().UnixNano(), extension)
	objectName := fmt.Sprintf("catalog/%s/%s", supplierId, filename)

	if Client == nil {
		// Local dev fallback — no real GCS, return dummy URLs
		placeholder := fmt.Sprintf("https://placehold.co/400x400/1a1a2e/e0e0e0?text=%s", filename)
		return placeholder, placeholder, nil
	}

	// Map extension to MIME type for signed URL content-type enforcement
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
		return "", "", fmt.Errorf("failed to sign URL: %w", err)
	}

	publicUrl := fmt.Sprintf("https://storage.googleapis.com/%s/%s", BucketName, objectName)

	return url, publicUrl, nil
}
