package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

var (
	// Client is nil in local dev when GCS is unavailable; upload tickets fall back to placeholders
	// only when EvidenceFailClosed() is false.
	Client     *storage.Client
	BucketName string
)

var (
	// ErrMediaStorageUnavailable is returned when evidence signing cannot produce a real GCS URL.
	ErrMediaStorageUnavailable = errors.New("media_storage_unavailable")
	// ErrInvalidEvidenceURI is returned when a claim photo URI is not an allowed GCS host.
	ErrInvalidEvidenceURI = errors.New("invalid_evidence_uri")
)

// InitGCS boots the storage client. An empty bucket name skips client init and
// leaves Client nil so GenerateUploadTicket can return dev placeholders (non-fail-closed only).
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

// EvidenceFailClosed is true for production / SSMR / staging or when REQUIRE_INFRA_ADAPTERS=true.
// Local stacks with REQUIRE_INFRA_ADAPTERS=false may still use placeholders for catalog/dev.
func EvidenceFailClosed() bool {
	if envBool("REQUIRE_INFRA_ADAPTERS", true) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV"))) {
	case "production", "prod", "ssmr", "staging":
		return true
	default:
		return false
	}
}

func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// IsPlaceholderMediaURL reports placehold.co (and empty) media URLs.
func IsPlaceholderMediaURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	return host == "placehold.co" || strings.HasSuffix(host, ".placehold.co")
}

// ValidateEvidenceURI accepts public GCS object URLs for the configured bucket (or any
// storage.googleapis.com object when bucket is unset in tests). Rejects placeholders.
func ValidateEvidenceURI(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrInvalidEvidenceURI
	}
	if IsPlaceholderMediaURL(raw) {
		return ErrInvalidEvidenceURI
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return ErrInvalidEvidenceURI
	}
	host := strings.ToLower(u.Hostname())
	// Signed PUT URLs use googleapis.com / storage.googleapis.com; public URLs use storage.googleapis.com/bucket/...
	switch {
	case host == "storage.googleapis.com":
		if BucketName == "" {
			return nil
		}
		// path: /{bucket}/object...
		path := strings.TrimPrefix(u.Path, "/")
		if !strings.HasPrefix(path, BucketName+"/") && path != BucketName {
			return ErrInvalidEvidenceURI
		}
		return nil
	case strings.HasSuffix(host, ".storage.googleapis.com"):
		// bucket.storage.googleapis.com/object
		bucketHost := strings.TrimSuffix(host, ".storage.googleapis.com")
		if BucketName != "" && bucketHost != BucketName {
			return ErrInvalidEvidenceURI
		}
		return nil
	default:
		return ErrInvalidEvidenceURI
	}
}

// GenerateUploadTicket creates a short-lived signed PUT URL for catalog image uploads.
func GenerateUploadTicket(supplierID, extension string) (uploadURL string, publicURL string, err error) {
	return GenerateUploadTicketFor(fmt.Sprintf("catalog/%s", strings.TrimSpace(supplierID)), extension)
}

// GenerateUploadTicketFor creates a short-lived signed PUT URL under objectPrefix.
// purpose examples: evidence/claim, evidence/driver, evidence/credit.
func GenerateUploadTicketFor(objectPrefix, extension string) (uploadURL string, publicURL string, err error) {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" {
		extension = "jpg"
	}
	objectPrefix = strings.Trim(strings.TrimSpace(objectPrefix), "/")
	if objectPrefix == "" {
		objectPrefix = "uploads"
	}
	filename := fmt.Sprintf("%d.%s", time.Now().UnixNano(), extension)
	objectName := objectPrefix + "/" + filename

	if Client == nil {
		if EvidenceFailClosed() {
			return "", "", fmt.Errorf("%w: gcs client not initialized", ErrMediaStorageUnavailable)
		}
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

	signed, err := Client.Bucket(BucketName).SignedURL(objectName, opts)
	if err != nil {
		if EvidenceFailClosed() {
			return "", "", fmt.Errorf("%w: signBlob failed (bind roles/iam.serviceAccountTokenCreator): %v", ErrMediaStorageUnavailable, err)
		}
		placeholder := fmt.Sprintf("https://placehold.co/400x400/1a1a2e/e0e0e0?text=%s", filename)
		return placeholder, placeholder, nil
	}

	public := fmt.Sprintf("https://storage.googleapis.com/%s/%s", BucketName, objectName)
	return signed, public, nil
}
