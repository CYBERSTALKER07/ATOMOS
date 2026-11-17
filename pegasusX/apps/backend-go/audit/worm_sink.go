package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// AnchorBlob is the signed canonical daily root JSON payload written to WORM storage.
type AnchorBlob struct {
	TenantID    string `json:"tenant_id"`
	DayBucket   string `json:"day_bucket"`
	HeadEventID string `json:"head_event_id"`
	EventCount  int64  `json:"event_count"`
	RootHash    string `json:"root_hash"`
	AnchoredBy  string `json:"anchored_by"`
	AnchoredAt  string `json:"anchored_at"`
	SignerKeyID string `json:"signer_key_id,omitempty"`
	Signature   string `json:"signature"`
}

// WormAnchorSink writes signed canonical daily anchor JSON payloads to append-only WORM storage.
type WormAnchorSink interface {
	WriteAnchor(ctx context.Context, blob AnchorBlob, canonicalJSON []byte) (wormURI string, err error)
}

// SignAnchorBlob signs the canonical anchor metadata and returns both the populated AnchorBlob and JSON bytes.
func SignAnchorBlob(anchor Anchor) (AnchorBlob, []byte, error) {
	signingKey := strings.TrimSpace(os.Getenv("AUDIT_ANCHOR_SIGNING_KEY"))
	signerKeyID := strings.TrimSpace(os.Getenv("AUDIT_ANCHOR_SIGNER_KEY_ID"))
	if signerKeyID == "" {
		if signingKey != "" {
			signerKeyID = "hmac-sha256-env-v1"
		} else {
			signerKeyID = "sha256-root-seal-v1"
		}
	}

	unsigned := struct {
		TenantID    string `json:"tenant_id"`
		DayBucket   string `json:"day_bucket"`
		HeadEventID string `json:"head_event_id"`
		EventCount  int64  `json:"event_count"`
		RootHash    string `json:"root_hash"`
		AnchoredBy  string `json:"anchored_by"`
		AnchoredAt  string `json:"anchored_at"`
		SignerKeyID string `json:"signer_key_id"`
	}{
		TenantID:    anchor.TenantID,
		DayBucket:   anchor.DayBucket,
		HeadEventID: anchor.HeadEventID,
		EventCount:  anchor.EventCount,
		RootHash:    anchor.RootHash,
		AnchoredBy:  anchor.AnchoredBy,
		AnchoredAt:  anchor.AnchoredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		SignerKeyID: signerKeyID,
	}
	rawUnsigned, err := json.Marshal(unsigned)
	if err != nil {
		return AnchorBlob{}, nil, fmt.Errorf("audit sign anchor marshal: %w", err)
	}

	var sig string
	if signingKey != "" {
		mac := hmac.New(sha256.New, []byte(signingKey))
		_, _ = mac.Write(rawUnsigned)
		sig = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256(rawUnsigned)
		sig = hex.EncodeToString(sum[:])
	}

	blob := AnchorBlob{
		TenantID:    unsigned.TenantID,
		DayBucket:   unsigned.DayBucket,
		HeadEventID: unsigned.HeadEventID,
		EventCount:  unsigned.EventCount,
		RootHash:    unsigned.RootHash,
		AnchoredBy:  unsigned.AnchoredBy,
		AnchoredAt:  unsigned.AnchoredAt,
		SignerKeyID: signerKeyID,
		Signature:   sig,
	}
	raw, err := json.Marshal(blob)
	if err != nil {
		return AnchorBlob{}, nil, fmt.Errorf("audit marshal signed anchor blob: %w", err)
	}
	return blob, raw, nil
}

// GCSWormSink persists immutable anchor blobs and large diffs to Google Cloud Storage
// using create-only object preconditions (DoesNotExist: true).
type GCSWormSink struct {
	client *storage.Client
	bucket string
	prefix string
}

// NewGCSWormSink constructs a GCS WORM sink with an initialized GCS client.
func NewGCSWormSink(client *storage.Client, bucket, prefix string) (*GCSWormSink, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, fmt.Errorf("%w: empty GCS bucket", ErrAnchorWormRequired)
	}
	if client == nil {
		return nil, fmt.Errorf("%w: nil GCS storage client", ErrAnchorWormRequired)
	}
	return &GCSWormSink{
		client: client,
		bucket: bucket,
		prefix: strings.Trim(strings.TrimSpace(prefix), "/"),
	}, nil
}

// WriteAnchor writes the signed canonical anchor JSON to GCS with DoesNotExist precondition.
func (s *GCSWormSink) WriteAnchor(ctx context.Context, blob AnchorBlob, canonicalJSON []byte) (string, error) {
	if s == nil || s.client == nil || strings.TrimSpace(s.bucket) == "" {
		return "", ErrAnchorWormRequired
	}
	shortRoot := blob.RootHash
	if len(shortRoot) > 12 {
		shortRoot = shortRoot[:12]
	}
	objectName := fmt.Sprintf("anchors/%s/%s-seq%d-%s.json",
		sanitizePathSegment(blob.TenantID),
		sanitizePathSegment(blob.DayBucket),
		blob.EventCount,
		shortRoot,
	)
	if s.prefix != "" {
		objectName = s.prefix + "/" + objectName
	}
	if err := s.writeImmutableObject(ctx, objectName, canonicalJSON); err != nil {
		return "", err
	}
	return fmt.Sprintf("gs://%s/%s", s.bucket, objectName), nil
}

// WriteDiff writes an oversized audit diff payload (>32KB) to GCS with DoesNotExist precondition.
func (s *GCSWormSink) WriteDiff(ctx context.Context, tenantID, dayBucket, eventID, field string, raw []byte) (string, error) {
	if s == nil || s.client == nil || strings.TrimSpace(s.bucket) == "" {
		return "", ErrAnchorWormRequired
	}
	objectName := fmt.Sprintf("diffs/%s/%s/%s-%s.json",
		sanitizePathSegment(tenantID),
		sanitizePathSegment(dayBucket),
		sanitizePathSegment(eventID),
		sanitizePathSegment(field),
	)
	if s.prefix != "" {
		objectName = s.prefix + "/" + objectName
	}
	if err := s.writeImmutableObject(ctx, objectName, raw); err != nil {
		return "", err
	}
	return fmt.Sprintf("gs://%s/%s", s.bucket, objectName), nil
}

func (s *GCSWormSink) writeImmutableObject(ctx context.Context, objectName string, payload []byte) error {
	obj := s.client.Bucket(s.bucket).Object(objectName).If(storage.Conditions{DoesNotExist: true})
	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return fmt.Errorf("gcs worm write %s: %w", objectName, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("gcs worm close %s: %w", objectName, err)
	}
	return nil
}

// LocalWormSink persists append-only read-only (0444) files inside a local WORM directory (non-production only).
type LocalWormSink struct {
	rootDir string
}

// NewLocalWormSink creates a local append-only WORM sink rooted at dir.
// Refuses initialization in production (auth.IsProduction() == true).
func NewLocalWormSink(dir string) (*LocalWormSink, error) {
	if auth.IsProduction() {
		return nil, fmt.Errorf("%w: local WORM directory disallowed in production", ErrAnchorWormRequired)
	}
	clean := strings.TrimSpace(dir)
	if clean == "" {
		return nil, fmt.Errorf("%w: empty local WORM directory", ErrAnchorWormRequired)
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return nil, fmt.Errorf("resolve local worm dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir local worm dir: %w", err)
	}
	return &LocalWormSink{rootDir: abs}, nil
}

// WriteAnchor writes the signed canonical anchor JSON using O_CREATE|O_EXCL|O_WRONLY and mode 0444.
func (s *LocalWormSink) WriteAnchor(_ context.Context, blob AnchorBlob, canonicalJSON []byte) (string, error) {
	if auth.IsProduction() {
		return "", fmt.Errorf("%w: local WORM write disallowed in production", ErrAnchorWormRequired)
	}
	shortRoot := blob.RootHash
	if len(shortRoot) > 12 {
		shortRoot = shortRoot[:12]
	}
	rel := filepath.Join("anchors", sanitizePathSegment(blob.TenantID),
		fmt.Sprintf("%s-seq%d-%s.json", sanitizePathSegment(blob.DayBucket), blob.EventCount, shortRoot))
	target := filepath.Join(s.rootDir, rel)
	if err := writeAppendOnlyFile(s.rootDir, target, canonicalJSON); err != nil {
		return "", err
	}
	return "file://" + target, nil
}

// WriteDiff writes an oversized diff payload using O_CREATE|O_EXCL|O_WRONLY and mode 0444.
func (s *LocalWormSink) WriteDiff(_ context.Context, tenantID, dayBucket, eventID, field string, raw []byte) (string, error) {
	if auth.IsProduction() {
		return "", fmt.Errorf("%w: local WORM diff disallowed in production", ErrAnchorWormRequired)
	}
	rel := filepath.Join("diffs", sanitizePathSegment(tenantID), sanitizePathSegment(dayBucket),
		fmt.Sprintf("%s-%s.json", sanitizePathSegment(eventID), sanitizePathSegment(field)))
	target := filepath.Join(s.rootDir, rel)
	if err := writeAppendOnlyFile(s.rootDir, target, raw); err != nil {
		return "", err
	}
	return "file://" + target, nil
}

func writeAppendOnlyFile(rootDir, target string, payload []byte) error {
	cleanTarget := filepath.Clean(target)
	if !strings.HasPrefix(cleanTarget, rootDir+string(filepath.Separator)) {
		return fmt.Errorf("%w: path escapes WORM root", ErrInvalidAuditInput)
	}
	if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
		return fmt.Errorf("worm mkdir: %w", err)
	}
	f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o444)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("worm file already locked at %s: %w", cleanTarget, err)
		}
		return fmt.Errorf("worm open: %w", err)
	}
	_, writeErr := f.Write(payload)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("worm write: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("worm close: %w", closeErr)
	}
	// Ensure read-only permissions regardless of process umask.
	_ = os.Chmod(cleanTarget, 0o444)
	return nil
}

var newStorageClientFunc = func(ctx context.Context) (*storage.Client, error) {
	return storage.NewClient(ctx)
}

// NewDefaultWormSinkFromEnv builds the configured WormAnchorSink from environment variables.
//
// Rules:
//   - If WORM_ANCHOR_GCS_BUCKET is set: attempts real GCS client initialization. If initialization
//     fails (e.g., missing credentials), fails closed when auth.IsProduction() is true (and
//     returns an error whenever GCS bucket is configured without usable credentials).
//   - Else if !auth.IsProduction() and WORM_ANCHOR_LOCAL_DIR is set: uses LocalWormSink.
//   - Else if auth.IsProduction(): fails closed with ErrAnchorWormRequired.
//   - Else (local/dev without env vars): returns (nil, nil).
func NewDefaultWormSinkFromEnv(ctx context.Context) (WormAnchorSink, error) {
	bucket := strings.TrimSpace(os.Getenv("WORM_ANCHOR_GCS_BUCKET"))
	prefix := strings.TrimSpace(os.Getenv("WORM_ANCHOR_GCS_PREFIX"))
	if bucket != "" {
		client, err := newStorageClientFunc(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: gcs credentials/client init failed: %v", ErrAnchorWormRequired, err)
		}
		return NewGCSWormSink(client, bucket, prefix)
	}
	if auth.IsProduction() {
		return nil, fmt.Errorf("%w: WORM_ANCHOR_GCS_BUCKET is required in production", ErrAnchorWormRequired)
	}
	if localDir := strings.TrimSpace(os.Getenv("WORM_ANCHOR_LOCAL_DIR")); localDir != "" {
		return NewLocalWormSink(localDir)
	}
	return nil, nil
}

// ResolveDiffWriterFromEnv resolves an optional DiffWriter from WORM env vars.
func ResolveDiffWriterFromEnv(ctx context.Context) (DiffWriter, error) {
	sink, err := NewDefaultWormSinkFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if dw, ok := sink.(DiffWriter); ok {
		return dw, nil
	}
	return nil, nil
}

func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), ".")
	if out == "" {
		return "unknown"
	}
	return out
}
