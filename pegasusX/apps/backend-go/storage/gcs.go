package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	posixpath "path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	// MinEvidenceUploadBytes is the minimum allowed size for uploaded evidence objects (1 byte).
	MinEvidenceUploadBytes int64 = 1
	// MaxEvidenceUploadBytes is the maximum allowed size for uploaded evidence objects (10 MiB).
	MaxEvidenceUploadBytes int64 = 10 << 20
)

var (
	// Client is nil in local dev when GCS is unavailable; upload tickets fall back to placeholders
	// only when EvidenceFailClosed() is false.
	Client     *storage.Client
	BucketName string

	// signUploadURL is nil in production (using Client.Bucket().SignedURL) and overridable in unit tests.
	signUploadURL func(bucket, objectName string, opts *storage.SignedURLOptions) (string, error)
)

var (
	// ErrMediaStorageUnavailable is returned when evidence signing cannot produce a real GCS URL.
	ErrMediaStorageUnavailable = errors.New("media_storage_unavailable")
	// ErrInvalidEvidenceURI is returned when a claim photo URI is not an allowed GCS host.
	ErrInvalidEvidenceURI = errors.New("invalid_evidence_uri")
	// ErrInvalidUploadExtension is returned when an upload ticket requests a disallowed file extension.
	ErrInvalidUploadExtension = errors.New("invalid_upload_extension")
	// ErrInvalidUploadSize is returned when an upload size falls outside 1..10 MiB.
	ErrInvalidUploadSize = errors.New("invalid_upload_size")
	// ErrInvalidUploadPrefix is returned when an upload object prefix is empty or contains path traversal.
	ErrInvalidUploadPrefix = errors.New("invalid_upload_prefix")
)

var allowedUploadMimeTypes = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"webp": "image/webp",
}

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

// EvidenceFailClosed is true for production / sandbox / staging or when REQUIRE_INFRA_ADAPTERS=true.
// Local stacks with REQUIRE_INFRA_ADAPTERS=false may still use placeholders for catalog/dev.
func EvidenceFailClosed() bool {
	if envBool("REQUIRE_INFRA_ADAPTERS", true) {
		return true
	}
	switch auth.EnvClass() {
	case auth.EnvClassProduction, auth.EnvClassSandbox, auth.EnvClassStaging:
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
	if strings.Contains(raw, "\\") || strings.Contains(u.Path, "..") || strings.Contains(strings.ToLower(u.RawPath), "%2e%2e") {
		return ErrInvalidEvidenceURI
	}
	cleaned := posixpath.Clean("/" + strings.TrimPrefix(u.Path, "/"))
	trimmed := strings.TrimPrefix(cleaned, "/")
	if trimmed == "" || trimmed == "." {
		return ErrInvalidEvidenceURI
	}
	host := strings.ToLower(u.Hostname())
	// Signed PUT URLs use googleapis.com / storage.googleapis.com; public URLs use storage.googleapis.com/bucket/...
	switch {
	case host == "storage.googleapis.com":
		if BucketName == "" {
			// Require bucket/object shape when BucketName is unset in tests.
			slashIdx := strings.IndexByte(trimmed, '/')
			if slashIdx <= 0 || slashIdx == len(trimmed)-1 {
				return ErrInvalidEvidenceURI
			}
			return nil
		}
		if !strings.HasPrefix(trimmed, BucketName+"/") || len(trimmed) <= len(BucketName)+1 {
			return ErrInvalidEvidenceURI
		}
		return nil
	case strings.HasSuffix(host, ".storage.googleapis.com"):
		// bucket.storage.googleapis.com/object
		bucketHost := strings.TrimSuffix(host, ".storage.googleapis.com")
		if bucketHost == "" {
			return ErrInvalidEvidenceURI
		}
		if BucketName != "" && bucketHost != BucketName {
			return ErrInvalidEvidenceURI
		}
		return nil
	default:
		return ErrInvalidEvidenceURI
	}
}

// NormalizeUploadExtension validates extension against the image allowlist and returns (ext, contentType).
func NormalizeUploadExtension(extension string) (string, string, error) {
	ext := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(extension)), ".")
	contentType, ok := allowedUploadMimeTypes[ext]
	if !ok {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidUploadExtension, extension)
	}
	return ext, contentType, nil
}

// ValidateUploadSize enforces the 1 byte .. 10 MiB evidence upload size bounds.
func ValidateUploadSize(sizeBytes int64) error {
	if sizeBytes < MinEvidenceUploadBytes || sizeBytes > MaxEvidenceUploadBytes {
		return fmt.Errorf("%w: got %d, want %d..%d", ErrInvalidUploadSize, sizeBytes, MinEvidenceUploadBytes, MaxEvidenceUploadBytes)
	}
	return nil
}

// ValidateObjectPrefix rejects empty or traversal-containing object prefixes and returns a clean relative prefix.
func ValidateObjectPrefix(objectPrefix string) (string, error) {
	raw := strings.TrimSpace(objectPrefix)
	if raw == "" || strings.Contains(raw, "\\") || strings.Contains(raw, "..") {
		return "", fmt.Errorf("%w: %q", ErrInvalidUploadPrefix, objectPrefix)
	}
	cleaned := strings.Trim(posixpath.Clean("/"+strings.Trim(raw, "/")), "/")
	if cleaned == "" || cleaned == "." || strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("%w: %q", ErrInvalidUploadPrefix, objectPrefix)
	}
	return cleaned, nil
}

// EvidenceUploadHeaders returns required GCS V4 signed headers enforcing create-only
// semantics (If-Generation-Match: 0) and content-length bounds.
func EvidenceUploadHeaders(minBytes, maxBytes int64) []string {
	return []string{
		"x-goog-if-generation-match:0",
		fmt.Sprintf("x-goog-content-length-range:%d,%d", minBytes, maxBytes),
	}
}

// GenerateUploadTicket creates a short-lived signed PUT URL for catalog image uploads.
func GenerateUploadTicket(supplierID, extension string) (uploadURL string, publicURL string, err error) {
	sup := strings.TrimSpace(supplierID)
	if sup == "" {
		return "", "", fmt.Errorf("%w: empty supplier_id", ErrInvalidUploadPrefix)
	}
	return GenerateUploadTicketFor("catalog/"+sup, extension)
}

// GenerateUploadTicketFor creates a short-lived signed PUT URL under objectPrefix.
// Enforces extension allowlist, safe object prefix, 1..10 MiB size bounds, and
// If-Generation-Match: 0 create-only precondition on signed uploads.
func GenerateUploadTicketFor(objectPrefix, extension string) (uploadURL string, publicURL string, err error) {
	cleanPrefix, err := ValidateObjectPrefix(objectPrefix)
	if err != nil {
		return "", "", err
	}
	ext, contentType, err := NormalizeUploadExtension(extension)
	if err != nil {
		return "", "", err
	}

	filename := strings.ReplaceAll(uuid.NewString(), "-", "") + "." + ext
	objectName := cleanPrefix + "/" + filename

	signer := signUploadURL
	if signer == nil {
		if Client == nil || strings.TrimSpace(BucketName) == "" {
			if EvidenceFailClosed() {
				return "", "", fmt.Errorf("%w: gcs client not initialized", ErrMediaStorageUnavailable)
			}
			placeholder := fmt.Sprintf("https://placehold.co/400x400/1a1a2e/e0e0e0?text=%s", filename)
			return placeholder, placeholder, nil
		}
		signer = func(bucket, object string, opts *storage.SignedURLOptions) (string, error) {
			return Client.Bucket(bucket).SignedURL(object, opts)
		}
	}

	opts := &storage.SignedURLOptions{
		Scheme:      storage.SigningSchemeV4,
		Method:      "PUT",
		Expires:     time.Now().Add(15 * time.Minute),
		ContentType: contentType,
		Headers:     EvidenceUploadHeaders(MinEvidenceUploadBytes, MaxEvidenceUploadBytes),
	}

	signed, err := signer(BucketName, objectName, opts)
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
