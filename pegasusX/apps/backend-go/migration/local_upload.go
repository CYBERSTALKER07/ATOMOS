package migration

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	posixpath "path"
	"strconv"
	"strings"
	"sync"
)

var (
	ephemeralLocalSecretOnce sync.Once
	ephemeralLocalSecretVal  []byte
)

func resolveLocalUploadSecret(configured string) []byte {
	trimmed := strings.TrimSpace(configured)
	if trimmed == "" {
		trimmed = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if trimmed != "" {
		sum := sha256.Sum256([]byte("pegasusx-migration-local-upload:" + trimmed))
		return sum[:]
	}
	ephemeralLocalSecretOnce.Do(func() {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			panic("crypto/rand failed while initializing local upload key: " + err.Error())
		}
		ephemeralLocalSecretVal = buf
	})
	return ephemeralLocalSecretVal
}

func deriveLocalUploadBaseURL(r *http.Request) string {
	if override := strings.TrimRight(strings.TrimSpace(os.Getenv("MIGRATION_LOCAL_UPLOAD_BASE_URL")), "/"); override != "" {
		return override
	}
	if r == nil {
		return ""
	}
	// Prefer the listener host (r.Host) over X-Forwarded-Host unless an explicit
	// X-Forwarded-Prefix is present: BFF proxies (e.g. Next.js /api/[...path])
	// strip their /api prefix before forwarding and attach X-Forwarded-Host for
	// the UI origin, which does not directly serve /v1/supplier/migrations/local-upload.
	prefix := strings.TrimRight(strings.TrimSpace(r.Header.Get("X-Forwarded-Prefix")), "/")
	xfHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if idx := strings.IndexByte(xfHost, ','); idx >= 0 {
		xfHost = strings.TrimSpace(xfHost[:idx])
	}
	if xfHost != "" && prefix != "" {
		scheme := "http"
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") || r.TLS != nil {
			scheme = "https"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		return scheme + "://" + xfHost + prefix
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = xfHost
	}
	if host == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil || (host == xfHost && strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")) {
		scheme = "https"
	}
	return scheme + "://" + host
}

func buildLocalUploadSelfURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	base := deriveLocalUploadBaseURL(r)
	reqURI := r.URL.RequestURI()
	if base == "" {
		return reqURI
	}
	return base + reqURI
}

func resolveSafeLocalArtifactPath(localRoot, rawGCSPath string) (string, error) {
	cleanRoot := strings.TrimSpace(localRoot)
	if cleanRoot == "" {
		return "", fmt.Errorf("%w: local root missing", ErrUploadStorageUnavailable)
	}
	cleanRel := strings.TrimLeft(posixpath.Clean("/"+strings.TrimSpace(rawGCSPath)), "/")
	if cleanRel == "" || cleanRel == "." || strings.Contains(cleanRel, "..") || strings.ContainsRune(rawGCSPath, '\x00') {
		return "", fmt.Errorf("%w: %q", ErrInvalidArtifactPath, rawGCSPath)
	}
	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(cleanRel)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: escaped local root", ErrInvalidArtifactPath)
	}
	return absTarget, nil
}

func parseContentRangeHeader(header string) (start, end, total int64, ok bool) {
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(trimmed), "bytes ") {
		return 0, 0, 0, false
	}
	spec := strings.TrimSpace(trimmed[len("bytes "):])
	slashIdx := strings.IndexByte(spec, '/')
	dashIdx := strings.IndexByte(spec, '-')
	if dashIdx <= 0 || slashIdx <= dashIdx+1 {
		return 0, 0, 0, false
	}
	s, err1 := strconv.ParseInt(strings.TrimSpace(spec[:dashIdx]), 10, 64)
	e, err2 := strconv.ParseInt(strings.TrimSpace(spec[dashIdx+1:slashIdx]), 10, 64)
	tot, err3 := strconv.ParseInt(strings.TrimSpace(spec[slashIdx+1:]), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || s < 0 || e < s || tot <= e {
		return 0, 0, 0, false
	}
	return s, e, tot, true
}

// HandleLocalArtifactUpload receives HMAC-signed migration artifact uploads exclusively in
// non-enforced local environments when IMPORT_LOCAL_FILE_ROOT is configured.
func (h *Handlers) HandleLocalArtifactUpload(w http.ResponseWriter, r *http.Request) {
	if IsAuthBypassActive(h.allowAuthBypass) {
		WriteAuthzError(w, ErrAuthBypassForbidden)
		return
	}
	if uploadStorageFailClosed() {
		writeMigrationError(w, http.StatusForbidden, "local_upload_disabled", "local artifact uploads are disabled in enforced environments")
		return
	}
	localRoot := strings.TrimSpace(os.Getenv("IMPORT_LOCAL_FILE_ROOT"))
	if localRoot == "" {
		writeMigrationError(w, http.StatusNotFound, "local_upload_unconfigured", "IMPORT_LOCAL_FILE_ROOT is not configured")
		return
	}

	q := r.URL.Query()
	rawPath := strings.TrimSpace(q.Get("path"))
	mime := strings.TrimSpace(q.Get("mime"))
	sig := strings.TrimSpace(q.Get("sig"))
	expUnix, expErr := strconv.ParseInt(strings.TrimSpace(q.Get("exp")), 10, 64)
	maxBytes, maxErr := strconv.ParseInt(strings.TrimSpace(q.Get("max")), 10, 64)
	if rawPath == "" || expErr != nil || maxErr != nil || maxBytes <= 0 || maxBytes > MaxArtifactBytes {
		writeMigrationError(w, http.StatusBadRequest, "invalid_upload_ticket", "malformed signed local upload parameters")
		return
	}
	if !VerifyLocalUploadSignature(rawPath, mime, expUnix, maxBytes, sig, h.localUploadSecret) {
		writeMigrationError(w, http.StatusForbidden, "invalid_upload_signature", "local upload ticket signature mismatch")
		return
	}
	if h.now().UTC().Unix() > expUnix {
		writeMigrationError(w, http.StatusForbidden, "upload_ticket_expired", "local upload ticket has expired")
		return
	}

	absTarget, err := resolveSafeLocalArtifactPath(localRoot, rawPath)
	if err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_artifact_path", err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		writeMigrationError(w, http.StatusInternalServerError, "prepare_local_dir_failed", err.Error())
		return
	}

	switch r.Method {
	case http.MethodPost:
		if !strings.EqualFold(strings.TrimSpace(r.Header.Get("x-goog-resumable")), "start") {
			writeMigrationError(w, http.StatusBadRequest, "resumable_header_required", "x-goog-resumable: start required for POST")
			return
		}
		if err := os.WriteFile(absTarget, nil, 0o644); err != nil {
			writeMigrationError(w, http.StatusInternalServerError, "init_resumable_failed", err.Error())
			return
		}
		if selfURL, uErr := url.Parse(buildLocalUploadSelfURL(r)); uErr == nil && selfURL.String() != "" {
			w.Header().Set("Location", selfURL.String())
		}
		writeMigrationJSON(w, http.StatusCreated, map[string]any{"status": "resumable_started", "gcs_path": rawPath})

	case http.MethodPut:
		readCap := maxBytes + 1
		if rangeHdr := strings.TrimSpace(r.Header.Get("Content-Range")); rangeHdr != "" {
			start, end, total, ok := parseContentRangeHeader(rangeHdr)
			if !ok || total > maxBytes {
				writeMigrationError(w, http.StatusBadRequest, "invalid_content_range", "invalid or oversized Content-Range header")
				return
			}
			expectedLen := end - start + 1
			if expectedLen > 64<<20 {
				writeMigrationError(w, http.StatusRequestEntityTooLarge, "chunk_too_large", "chunk exceeds maximum bounded upload slice")
				return
			}
			chunkReader := http.MaxBytesReader(w, r.Body, expectedLen+1)
			chunk, readErr := io.ReadAll(chunkReader)
			if readErr != nil || int64(len(chunk)) != expectedLen {
				writeMigrationError(w, http.StatusBadRequest, "chunk_size_mismatch", "chunk byte count does not match Content-Range")
				return
			}
			f, openErr := os.OpenFile(absTarget, os.O_CREATE|os.O_RDWR, 0o644)
			if openErr != nil {
				writeMigrationError(w, http.StatusInternalServerError, "open_local_artifact_failed", openErr.Error())
				return
			}
			_, wErr := f.WriteAt(chunk, start)
			cErr := f.Close()
			if wErr != nil || cErr != nil {
				writeMigrationError(w, http.StatusInternalServerError, "write_chunk_failed", "failed persisting chunk to local artifact root")
				return
			}
			if end+1 < total {
				w.WriteHeader(308)
				return
			}
			writeMigrationJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "gcs_path": rawPath, "bytes": total})
			return
		}

		bodyReader := http.MaxBytesReader(w, r.Body, readCap)
		data, readErr := io.ReadAll(bodyReader)
		if readErr != nil || int64(len(data)) > maxBytes {
			writeMigrationError(w, http.StatusRequestEntityTooLarge, "artifact_too_large", "uploaded artifact exceeds ticket max byte limit")
			return
		}
		tmpFile, tmpErr := os.CreateTemp(filepath.Dir(absTarget), ".upload-*")
		if tmpErr != nil {
			writeMigrationError(w, http.StatusInternalServerError, "create_temp_failed", tmpErr.Error())
			return
		}
		tmpPath := tmpFile.Name()
		if _, wErr := tmpFile.Write(data); wErr != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			writeMigrationError(w, http.StatusInternalServerError, "write_temp_failed", wErr.Error())
			return
		}
		if cErr := tmpFile.Close(); cErr != nil {
			_ = os.Remove(tmpPath)
			writeMigrationError(w, http.StatusInternalServerError, "close_temp_failed", cErr.Error())
			return
		}
		if rErr := os.Rename(tmpPath, absTarget); rErr != nil {
			_ = os.Remove(tmpPath)
			writeMigrationError(w, http.StatusInternalServerError, "finalize_upload_failed", rErr.Error())
			return
		}
		writeMigrationJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "gcs_path": rawPath, "bytes": len(data)})

	default:
		writeMigrationError(w, http.StatusMethodNotAllowed, "method_not_allowed", "PUT or POST required")
	}
}
