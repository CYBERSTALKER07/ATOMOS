package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

// EvidenceInspection captures canonical URI identity, bounded stream size, and SHA-256 digests
// verified outside any Spanner read-write transaction.
type EvidenceInspection struct {
	URI              string `json:"uri"`
	CanonicalURI     string `json:"canonical_uri"`
	Bucket           string `json:"bucket,omitempty"`
	Object           string `json:"object,omitempty"`
	UriSha256        string `json:"uri_sha256"`
	ContentSha256    string `json:"content_sha256"`
	ObjectGeneration int64  `json:"object_generation"`
	SizeBytes        int64  `json:"size_bytes"`
}

// EvidenceStreamOpener opens a readable evidence stream and returns (stream, objectGeneration, error).
type EvidenceStreamOpener func(ctx context.Context, bucket, object string) (io.ReadCloser, int64, error)

// EvidenceInspector inspects an evidence URI and computes bounded streaming SHA-256 digests.
type EvidenceInspector func(ctx context.Context, uri string) (EvidenceInspection, error)

// openEvidenceObject is nil in production (using Client) and overridable in unit tests when Client == nil.
var openEvidenceObject EvidenceStreamOpener

// SetOpenEvidenceObjectForTest overrides GCS object stream opening for offline tests and returns a restore fn.
func SetOpenEvidenceObjectForTest(fn EvidenceStreamOpener) func() {
	prev := openEvidenceObject
	openEvidenceObject = fn
	return func() { openEvidenceObject = prev }
}

func sha256HexString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// CanonicalEvidenceURI validates raw and normalizes it into a canonical URI without query/fragment
// plus extracted GCS bucket/object when applicable.
func CanonicalEvidenceURI(raw string) (canonical string, bucket string, object string, err error) {
	trimmed := strings.TrimSpace(raw)
	if err := ValidateEvidenceURI(trimmed); err != nil {
		return "", "", "", err
	}
	u, perr := url.Parse(trimmed)
	if perr != nil || u.Scheme == "" || u.Host == "" {
		return "", "", "", ErrInvalidEvidenceURI
	}

	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	cleanPath := path.Clean("/" + u.Path)
	if strings.Contains(cleanPath, "..") {
		return "", "", "", ErrInvalidEvidenceURI
	}

	switch {
	case scheme == "gs":
		bucket = strings.TrimSpace(u.Host)
		object = strings.TrimPrefix(cleanPath, "/")
		if bucket == "" || object == "" || strings.HasSuffix(u.Path, "/") {
			return "", "", "", ErrInvalidEvidenceURI
		}
		canonical = "https://storage.googleapis.com/" + bucket + "/" + object
	case scheme == "https" && host == "storage.googleapis.com":
		trimmedPath := strings.TrimPrefix(cleanPath, "/")
		parts := strings.SplitN(trimmedPath, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" || strings.HasSuffix(u.Path, "/") {
			return "", "", "", ErrInvalidEvidenceURI
		}
		bucket = strings.TrimSpace(parts[0])
		object = parts[1]
		canonical = "https://storage.googleapis.com/" + bucket + "/" + object
	case scheme == "https" && strings.HasSuffix(host, ".storage.googleapis.com"):
		bucket = strings.TrimSuffix(host, ".storage.googleapis.com")
		object = strings.TrimPrefix(cleanPath, "/")
		if bucket == "" || object == "" || strings.HasSuffix(u.Path, "/") {
			return "", "", "", ErrInvalidEvidenceURI
		}
		canonical = "https://storage.googleapis.com/" + bucket + "/" + object
	default:
		if cleanPath == "/" {
			cleanPath = ""
		}
		canonical = scheme + "://" + host + cleanPath
	}

	if configured := strings.TrimSpace(BucketName); configured != "" && bucket != "" && bucket != configured {
		return "", "", "", ErrInvalidEvidenceURI
	}
	return canonical, bucket, object, nil
}

// ComputeEvidenceURISha256 validates raw, canonicalizes it, and returns its lowercase hex SHA-256 digest.
func ComputeEvidenceURISha256(raw string) (uriSha256 string, canonical string, err error) {
	canonical, _, _, err = CanonicalEvidenceURI(raw)
	if err != nil {
		return "", "", err
	}
	return sha256HexString(canonical), canonical, nil
}

// InspectEvidenceStream hashes object bytes with a strict io.LimitReader bound of MaxEvidenceUploadBytes+1
// so hostile or oversized streams (> 10 MiB) fail closed without exhausting memory.
func InspectEvidenceStream(r io.Reader, uri string, generation int64) (EvidenceInspection, error) {
	if r == nil {
		return EvidenceInspection{}, ErrMediaStorageUnavailable
	}
	canonical, bucket, object, err := CanonicalEvidenceURI(uri)
	if err != nil {
		return EvidenceInspection{}, err
	}
	uriHash := sha256HexString(canonical)

	h := sha256.New()
	buf := make([]byte, 32*1024)
	limited := io.LimitReader(r, MaxEvidenceUploadBytes+1)
	written, err := io.CopyBuffer(h, limited, buf)
	if err != nil {
		return EvidenceInspection{}, fmt.Errorf("read evidence stream: %w", err)
	}
	if written < MinEvidenceUploadBytes || written > MaxEvidenceUploadBytes {
		return EvidenceInspection{}, fmt.Errorf("%w: size=%d must be in [%d,%d]", ErrInvalidUploadSize, written, MinEvidenceUploadBytes, MaxEvidenceUploadBytes)
	}

	return EvidenceInspection{
		URI:              strings.TrimSpace(uri),
		CanonicalURI:     canonical,
		Bucket:           bucket,
		Object:           object,
		UriSha256:        uriHash,
		ContentSha256:    hex.EncodeToString(h.Sum(nil)),
		ObjectGeneration: generation,
		SizeBytes:        written,
	}, nil
}

// InspectEvidenceObject downloads and verifies an uploaded evidence object in bounded streaming mode
// using either an injected test opener or the configured GCS Client. Fails closed when neither is available.
func InspectEvidenceObject(ctx context.Context, uri string) (EvidenceInspection, error) {
	canonical, bucket, object, err := CanonicalEvidenceURI(uri)
	if err != nil {
		return EvidenceInspection{}, err
	}
	uriHash := sha256HexString(canonical)
	base := EvidenceInspection{
		URI:          strings.TrimSpace(uri),
		CanonicalURI: canonical,
		Bucket:       bucket,
		Object:       object,
		UriSha256:    uriHash,
	}

	if openEvidenceObject != nil {
		if bucket == "" || object == "" {
			return EvidenceInspection{}, ErrInvalidEvidenceURI
		}
		rc, gen, err := openEvidenceObject(ctx, bucket, object)
		if err != nil {
			return base, err
		}
		insp, inspectErr := InspectEvidenceStream(rc, uri, gen)
		if closeErr := rc.Close(); closeErr != nil && inspectErr == nil {
			return base, closeErr
		}
		return insp, inspectErr
	}

	if Client == nil {
		return base, ErrMediaStorageUnavailable
	}
	if bucket == "" || object == "" {
		return EvidenceInspection{}, ErrInvalidEvidenceURI
	}

	reader, err := Client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return base, err
	}
	if reader.Attrs.Size > MaxEvidenceUploadBytes {
		closeErr := reader.Close()
		sizeErr := fmt.Errorf("%w: size=%d exceeds max=%d", ErrInvalidUploadSize, reader.Attrs.Size, MaxEvidenceUploadBytes)
		if closeErr != nil {
			return base, fmt.Errorf("%w (close: %v)", sizeErr, closeErr)
		}
		return base, sizeErr
	}

	insp, inspectErr := InspectEvidenceStream(reader, uri, reader.Attrs.Generation)
	if closeErr := reader.Close(); closeErr != nil && inspectErr == nil {
		return base, closeErr
	}
	return insp, inspectErr
}
