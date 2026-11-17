package exportpacks

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
)

// PackStorage stores and retrieves immutable export pack binary artifacts.
type PackStorage interface {
	PutObject(ctx context.Context, key, contentType string, data []byte) (string, error)
	GetObject(ctx context.Context, uriOrKey string) ([]byte, error)
}

// MemoryPackStorage stores pack artifacts in memory (for isolated unit tests).
type MemoryPackStorage struct {
	mu   sync.RWMutex
	objs map[string][]byte
}

// NewMemoryPackStorage returns an empty in-memory artifact store.
func NewMemoryPackStorage() *MemoryPackStorage {
	return &MemoryPackStorage{objs: make(map[string][]byte)}
}

func (m *MemoryPackStorage) PutObject(_ context.Context, key, _ string, data []byte) (string, error) {
	clean, err := sanitizeObjectKey(key)
	if err != nil {
		return "", err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	uri := "mem://" + clean
	m.mu.Lock()
	m.objs[uri] = cp
	m.objs[clean] = cp
	m.mu.Unlock()
	return uri, nil
}

func (m *MemoryPackStorage) GetObject(_ context.Context, uriOrKey string) ([]byte, error) {
	k := strings.TrimSpace(uriOrKey)
	m.mu.RLock()
	b, ok := m.objs[k]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: object %q not found", ErrNotFound, uriOrKey)
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp, nil
}

// LocalPackStorage stores pack artifacts safely under rootDir for local dev/tests.
type LocalPackStorage struct {
	rootDir string
}

// NewLocalPackStorage initializes filesystem storage anchored at rootDir.
func NewLocalPackStorage(rootDir string) (*LocalPackStorage, error) {
	root := strings.TrimSpace(rootDir)
	if root == "" {
		root = filepath.Join(os.TempDir(), "atomos-export-packs")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local pack root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create local pack root: %w", err)
	}
	return &LocalPackStorage{rootDir: abs}, nil
}

func (l *LocalPackStorage) resolvePath(uriOrKey string) (string, string, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(uriOrKey), "file://")
	clean, err := sanitizeObjectKey(raw)
	if err != nil {
		return "", "", err
	}
	full := filepath.Join(l.rootDir, filepath.FromSlash(clean))
	rel, err := filepath.Rel(l.rootDir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("%w: invalid object key %q", ErrInvalidInput, uriOrKey)
	}
	return full, clean, nil
}

func (l *LocalPackStorage) PutObject(_ context.Context, key, _ string, data []byte) (string, error) {
	full, clean, err := l.resolvePath(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return "", fmt.Errorf("mkdir export pack dir: %w", err)
	}
	if err := os.WriteFile(full, data, 0o600); err != nil {
		return "", fmt.Errorf("write export pack file: %w", err)
	}
	return "file://" + clean, nil
}

func (l *LocalPackStorage) GetObject(_ context.Context, uriOrKey string) ([]byte, error) {
	full, _, err := l.resolvePath(uriOrKey)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: artifact missing", ErrNotFound)
		}
		return nil, fmt.Errorf("read export pack file: %w", err)
	}
	return b, nil
}

// GCSPackStorage persists and streams pack artifacts to Google Cloud Storage.
type GCSPackStorage struct {
	client *storage.Client
	bucket string
}

// NewGCSPackStorage constructs a GCS artifact backend for the configured bucket.
func NewGCSPackStorage(client *storage.Client, bucket string) (*GCSPackStorage, error) {
	b := strings.TrimSpace(bucket)
	if client == nil || b == "" {
		return nil, fmt.Errorf("%w: gcs client and bucket required", ErrStorageUnavailable)
	}
	return &GCSPackStorage{client: client, bucket: b}, nil
}

func (g *GCSPackStorage) PutObject(ctx context.Context, key, contentType string, data []byte) (string, error) {
	clean, err := sanitizeObjectKey(key)
	if err != nil {
		return "", err
	}
	w := g.client.Bucket(g.bucket).Object(clean).NewWriter(ctx)
	if contentType != "" {
		w.ContentType = contentType
	}
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("gcs write %s: %w", clean, err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("gcs close %s: %w", clean, err)
	}
	return "gs://" + g.bucket + "/" + clean, nil
}

func (g *GCSPackStorage) GetObject(ctx context.Context, uriOrKey string) ([]byte, error) {
	bucket := g.bucket
	objKey := strings.TrimSpace(uriOrKey)
	if strings.HasPrefix(objKey, "gs://") {
		trimmed := strings.TrimPrefix(objKey, "gs://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("%w: malformed gcs uri", ErrInvalidInput)
		}
		bucket = parts[0]
		objKey = parts[1]
	}
	clean, err := sanitizeObjectKey(objKey)
	if err != nil {
		return nil, err
	}
	rc, err := g.client.Bucket(bucket).Object(clean).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("%w: gcs object missing", ErrNotFound)
		}
		return nil, fmt.Errorf("gcs read %s: %w", clean, err)
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 64<<20))
}

// NewDefaultPackStorage picks GCS when EXPORT_PACKS_GCS_BUCKET is set, failing closed in production.
func NewDefaultPackStorage(ctx context.Context) (PackStorage, error) {
	bucket := strings.TrimSpace(os.Getenv("EXPORT_PACKS_GCS_BUCKET"))
	if bucket != "" {
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
		}
		return NewGCSPackStorage(client, bucket)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production") {
		return nil, fmt.Errorf("%w: EXPORT_PACKS_GCS_BUCKET must be configured in production", ErrStorageUnavailable)
	}
	return NewLocalPackStorage(os.Getenv("EXPORT_PACKS_LOCAL_ROOT"))
}

func sanitizeObjectKey(raw string) (string, error) {
	trimmed := strings.TrimLeft(strings.TrimSpace(raw), "/")
	if trimmed == "" || strings.Contains(trimmed, "..") || strings.ContainsRune(trimmed, '\\') {
		return "", fmt.Errorf("%w: invalid object key %q", ErrInvalidInput, raw)
	}
	return trimmed, nil
}
