package migration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	posixpath "path"
	"strconv"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

func uploadStorageFailClosed() bool {
	if auth.IsEnforcedEnv() {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_INFRA_ADAPTERS")), "true")
}

const (
	// MaxArtifactBytes bounds a single raw export upload to 500 MiB.
	MaxArtifactBytes int64 = 500 << 20
	// ResumableUploadThreshold switches to GCS resumable upload negotiation above 50 MiB.
	ResumableUploadThreshold int64 = 50 << 20
)

var (
	// ErrUnsupportedFileExtension indicates a file outside the supported digital export formats.
	ErrUnsupportedFileExtension = errors.New("unsupported_file_extension")
	// ErrInvalidArtifactSize indicates byte_size <= 0 or > 500 MiB.
	ErrInvalidArtifactSize = errors.New("invalid_artifact_size")
	// ErrInvalidArtifactPath indicates a path-traversal or empty filename attempt.
	ErrInvalidArtifactPath = errors.New("invalid_artifact_path")
	// ErrUploadStorageUnavailable indicates signed GCS upload URLs could not be issued in an enforced env.
	ErrUploadStorageUnavailable = errors.New("upload_storage_unavailable")
)

var allowedArtifactMIMEs = map[string]string{
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xls":  "application/vnd.ms-excel",
	"csv":  "text/csv",
	"tsv":  "text/tab-separated-values",
	"xml":  "application/xml",
	"json": "application/json",
}

// ArtifactDescriptor identifies a file declared by the user in a bundle upload request.
type ArtifactDescriptor struct {
	ArtifactID    string
	FileName      string
	ByteSize      int64
	ContentSHA256 string
}

// SignURLFunc overrides GCS V4 signing in unit tests.
type SignURLFunc func(bucket, objectName string, opts *gcs.SignedURLOptions) (string, error)

// SanitizeArtifactFilename strips directory components and rejects traversal or unsupported extensions.
func SanitizeArtifactFilename(rawName string) (cleanName, mimeType string, err error) {
	trimmed := strings.TrimSpace(rawName)
	if trimmed == "" || strings.Contains(trimmed, "..") || strings.ContainsAny(trimmed, "\\/\x00") {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidArtifactPath, rawName)
	}
	ext := strings.TrimPrefix(strings.ToLower(posixpath.Ext(trimmed)), ".")
	mime, ok := allowedArtifactMIMEs[ext]
	if !ok {
		return "", "", fmt.Errorf("%w: allowed extensions xlsx, xls, csv, tsv, xml, json (got %q)", ErrUnsupportedFileExtension, ext)
	}
	return trimmed, mime, nil
}

// BuildArtifactObjectPath constructs a deterministic cell-isolated GCS key for an uploaded artifact.
func BuildArtifactObjectPath(homeCell, supplierID, projectID, bundleID, artifactID, cleanFileName string) (string, error) {
	cell := strings.TrimSpace(homeCell)
	if cell == "" {
		cell = "cell-uz"
	}
	for _, part := range []string{cell, supplierID, projectID, bundleID, artifactID, cleanFileName} {
		p := strings.TrimSpace(part)
		if p == "" || strings.Contains(p, "..") || strings.ContainsAny(p, "\\/\x00") {
			return "", fmt.Errorf("%w: invalid segment %q", ErrInvalidArtifactPath, part)
		}
	}
	return posixpath.Join("migrations", cell, supplierID, "projects", projectID, "bundles", bundleID, artifactID, cleanFileName), nil
}

// CreateArtifactUploadTicket generates a V4 signed GCS upload URL (resumable POST above 50 MiB, direct PUT up to 50 MiB).
// Fails closed with ErrUploadStorageUnavailable whenever storage.EvidenceFailClosed() is true and GCS is unavailable.
func CreateArtifactUploadTicket(
	homeCell, supplierID, projectID, bundleID string,
	desc ArtifactDescriptor,
	now time.Time,
	customSigner SignURLFunc,
) (UploadTicket, error) {
	return CreateArtifactUploadTicketWithLocalFallback(homeCell, supplierID, projectID, bundleID, desc, now, customSigner, "", nil)
}

// CreateArtifactUploadTicketWithLocalFallback behaves like CreateArtifactUploadTicket and, in
// non-enforced local environments where GCS is not configured and IMPORT_LOCAL_FILE_ROOT is set,
// mints an HMAC-SHA256 signed upload URL against localBaseURL.
func CreateArtifactUploadTicketWithLocalFallback(
	homeCell, supplierID, projectID, bundleID string,
	desc ArtifactDescriptor,
	now time.Time,
	customSigner SignURLFunc,
	localBaseURL string,
	localSecret []byte,
) (UploadTicket, error) {
	cleanName, mime, err := SanitizeArtifactFilename(desc.FileName)
	if err != nil {
		return UploadTicket{}, err
	}
	if desc.ByteSize <= 0 || desc.ByteSize > MaxArtifactBytes {
		return UploadTicket{}, fmt.Errorf("%w: got %d, want 1..%d", ErrInvalidArtifactSize, desc.ByteSize, MaxArtifactBytes)
	}
	objPath, err := BuildArtifactObjectPath(homeCell, supplierID, projectID, bundleID, desc.ArtifactID, cleanName)
	if err != nil {
		return UploadTicket{}, err
	}

	resumable := desc.ByteSize > ResumableUploadThreshold
	method := "PUT"
	headersMap := map[string]string{
		"Content-Type": mime,
	}
	signedHeaders := []string{}
	if resumable {
		method = "POST"
		headersMap["x-goog-resumable"] = "start"
		signedHeaders = append(signedHeaders, "x-goog-resumable:start")
	} else {
		headersMap["x-goog-content-length-range"] = fmt.Sprintf("1,%d", ResumableUploadThreshold)
		signedHeaders = append(signedHeaders, fmt.Sprintf("x-goog-content-length-range:1,%d", ResumableUploadThreshold))
	}

	expiresAt := now.UTC().Add(30 * time.Minute)
	signer := customSigner
	if signer == nil && storage.Client != nil && strings.TrimSpace(storage.BucketName) != "" {
		signer = func(bucket, object string, opts *gcs.SignedURLOptions) (string, error) {
			return storage.Client.Bucket(bucket).SignedURL(object, opts)
		}
	}

	var uploadURL string
	if signer != nil {
		bucket := strings.TrimSpace(storage.BucketName)
		if bucket == "" {
			bucket = "pegasusx-dev-migrations"
		}
		signed, signErr := signer(bucket, objPath, &gcs.SignedURLOptions{
			Scheme:      gcs.SigningSchemeV4,
			Method:      method,
			Expires:     expiresAt,
			ContentType: mime,
			Headers:     signedHeaders,
		})
		if signErr != nil {
			if uploadStorageFailClosed() {
				return UploadTicket{}, fmt.Errorf("%w: sign url: %v", ErrUploadStorageUnavailable, signErr)
			}
		} else {
			uploadURL = signed
		}
	}

	if uploadURL == "" {
		if uploadStorageFailClosed() {
			return UploadTicket{}, fmt.Errorf("%w: gcs bucket/client not configured", ErrUploadStorageUnavailable)
		}
		if base := strings.TrimRight(strings.TrimSpace(localBaseURL), "/"); base != "" &&
			strings.TrimSpace(os.Getenv("IMPORT_LOCAL_FILE_ROOT")) != "" &&
			len(localSecret) > 0 {
			maxAllowed := desc.ByteSize
			if resumable && maxAllowed < MaxArtifactBytes {
				maxAllowed = MaxArtifactBytes
			}
			uploadURL = BuildSignedLocalUploadURL(base, objPath, mime, expiresAt, maxAllowed, localSecret)
		} else {
			uploadURL = fmt.Sprintf("https://local.invalid/upload/%s", objPath)
		}
	}

	return UploadTicket{
		ArtifactID:      desc.ArtifactID,
		FileName:        cleanName,
		GCSPath:         objPath,
		UploadURL:       uploadURL,
		Method:          method,
		ContentType:     mime,
		Resumable:       resumable,
		RequiredHeaders: headersMap,
		ExpiresAtUTC:    expiresAt,
	}, nil
}

// LocalUploadSignature computes the HMAC-SHA256 signature for non-enforced local migration uploads.
func LocalUploadSignature(gcsPath, mime string, expUnix, maxBytes int64, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	payload := fmt.Sprintf("%s\n%d\n%d\n%s", gcsPath, expUnix, maxBytes, mime)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyLocalUploadSignature verifies HMAC-SHA256 signature in constant time.
func VerifyLocalUploadSignature(gcsPath, mime string, expUnix, maxBytes int64, providedSig string, secret []byte) bool {
	if len(secret) == 0 || strings.TrimSpace(providedSig) == "" {
		return false
	}
	expected := LocalUploadSignature(gcsPath, mime, expUnix, maxBytes, secret)
	return hmac.Equal([]byte(expected), []byte(providedSig))
}

// BuildSignedLocalUploadURL builds an HMAC-signed non-enforced local upload endpoint URL.
func BuildSignedLocalUploadURL(baseURL, gcsPath, mime string, expiresAt time.Time, maxBytes int64, secret []byte) string {
	expUnix := expiresAt.UTC().Unix()
	sig := LocalUploadSignature(gcsPath, mime, expUnix, maxBytes, secret)
	q := url.Values{}
	q.Set("path", gcsPath)
	q.Set("exp", strconv.FormatInt(expUnix, 10))
	q.Set("max", strconv.FormatInt(maxBytes, 10))
	q.Set("mime", mime)
	q.Set("sig", sig)
	return strings.TrimRight(baseURL, "/") + "/v1/supplier/migrations/local-upload?" + q.Encode()
}

// ReadArtifactBytes reads a staged migration artifact from IMPORT_LOCAL_FILE_ROOT (dev/smoke) or GCS.
func ReadArtifactBytes(ctx context.Context, gcsPath string) ([]byte, error) {
	cleanRel := strings.TrimLeft(posixpath.Clean("/"+strings.TrimSpace(gcsPath)), "/")
	if cleanRel == "" || cleanRel == "." || strings.Contains(cleanRel, "..") {
		return nil, fmt.Errorf("%w: %q", ErrInvalidArtifactPath, gcsPath)
	}
	if localRoot := strings.TrimSpace(os.Getenv("IMPORT_LOCAL_FILE_ROOT")); localRoot != "" {
		localPath := filepath.Join(localRoot, filepath.FromSlash(cleanRel))
		if data, err := os.ReadFile(localPath); err == nil {
			return data, nil
		}
	}
	if storage.Client == nil || strings.TrimSpace(storage.BucketName) == "" {
		return nil, fmt.Errorf("%w: gcs client unavailable for %s", ErrUploadStorageUnavailable, cleanRel)
	}
	rc, err := storage.Client.Bucket(storage.BucketName).Object(cleanRel).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open gcs object %s: %w", cleanRel, err)
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}
