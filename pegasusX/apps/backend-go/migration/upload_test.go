package migration

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	gcs "cloud.google.com/go/storage"
)

func TestCreateArtifactUploadTicket_ResumableThresholdAndSecurity(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	var seenMethod string
	var seenHeaders []string
	signer := func(bucket, objectName string, opts *gcs.SignedURLOptions) (string, error) {
		seenMethod = opts.Method
		seenHeaders = append([]string(nil), opts.Headers...)
		return "https://signed.example/" + objectName, nil
	}

	// Small file (<= 50 MiB) -> direct PUT with x-goog-content-length-range signed header matching gcs.go
	small, err := CreateArtifactUploadTicket("cell-uz", "sup-1", "proj-1", "bnd-1", ArtifactDescriptor{
		ArtifactID: "art-1",
		FileName:   "catalog.xlsx",
		ByteSize:   1024 * 1024,
	}, now, signer)
	if err != nil {
		t.Fatalf("small ticket: %v", err)
	}
	if small.Resumable || small.Method != "PUT" || seenMethod != "PUT" {
		t.Fatalf("expected PUT for small file, got %+v", small)
	}
	wantRange := "x-goog-content-length-range:1,52428800"
	if !slices.Contains(seenHeaders, wantRange) {
		t.Fatalf("expected %q in signed headers, got %v", wantRange, seenHeaders)
	}
	for _, h := range seenHeaders {
		if strings.HasPrefix(strings.ToLower(h), "content-length:") {
			t.Fatalf("signed headers must not bind browser-forbidden content-length header: %v", seenHeaders)
		}
	}
	if !strings.HasPrefix(small.GCSPath, "migrations/cell-uz/sup-1/projects/proj-1/bundles/bnd-1/art-1/") {
		t.Fatalf("unexpected gcs path: %s", small.GCSPath)
	}

	// Large file (> 50 MiB) -> resumable POST
	large, err := CreateArtifactUploadTicket("cell-uz", "sup-1", "proj-1", "bnd-1", ArtifactDescriptor{
		ArtifactID: "art-2",
		FileName:   "full_erp_dump.xml",
		ByteSize:   60 << 20,
	}, now, signer)
	if err != nil {
		t.Fatalf("large ticket: %v", err)
	}
	if !large.Resumable || large.Method != "POST" || seenMethod != "POST" {
		t.Fatalf("expected resumable POST for >50MiB file, got %+v", large)
	}
	if large.RequiredHeaders["x-goog-resumable"] != "start" {
		t.Fatalf("expected x-goog-resumable=start header, got %+v", large.RequiredHeaders)
	}

	// Traversal & unsupported extension rejection
	if _, err := CreateArtifactUploadTicket("cell-uz", "sup-1", "proj-1", "bnd-1", ArtifactDescriptor{
		ArtifactID: "art-3", FileName: "../escape.csv", ByteSize: 100,
	}, now, signer); !errors.Is(err, ErrInvalidArtifactPath) {
		t.Fatalf("expected ErrInvalidArtifactPath, got %v", err)
	}
	if _, err := CreateArtifactUploadTicket("cell-uz", "sup-1", "proj-1", "bnd-1", ArtifactDescriptor{
		ArtifactID: "art-4", FileName: "scan.pdf", ByteSize: 100,
	}, now, signer); !errors.Is(err, ErrUnsupportedFileExtension) {
		t.Fatalf("expected ErrUnsupportedFileExtension for PDF, got %v", err)
	}

	// Enforced env without GCS signer must fail closed
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	if _, err := CreateArtifactUploadTicket("cell-uz", "sup-1", "proj-1", "bnd-1", ArtifactDescriptor{
		ArtifactID: "art-5", FileName: "stock.csv", ByteSize: 100,
	}, now, nil); !errors.Is(err, ErrUploadStorageUnavailable) {
		t.Fatalf("expected ErrUploadStorageUnavailable when enforced without signer, got %v", err)
	}
}

func TestLocalArtifactUpload_SignedDirectResumableAndFailClosed(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	t.Setenv("IMPORT_LOCAL_FILE_ROOT", root)
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "local")

	h := NewHandlers(HandlerDeps{
		JWTSecret: "local-test-secret-0123456789abcdef",
		Now:       func() time.Time { return now },
	})

	// 1. Direct PUT local upload ticket + upload + ReadArtifactBytes + oversized rejection
	payload := []byte("sku,name\nSKU1,Tea\n")
	ticket, err := CreateArtifactUploadTicketWithLocalFallback(
		"cell-uz", "sup-1", "proj-1", "bnd-1",
		ArtifactDescriptor{ArtifactID: "art-local-1", FileName: "catalog.csv", ByteSize: int64(len(payload))},
		now, nil, "http://127.0.0.1:8080", h.localUploadSecret,
	)
	if err != nil {
		t.Fatalf("CreateArtifactUploadTicketWithLocalFallback: %v", err)
	}
	if !strings.HasPrefix(ticket.UploadURL, "http://127.0.0.1:8080/v1/supplier/migrations/local-upload?") {
		t.Fatalf("expected signed local upload URL, got %s", ticket.UploadURL)
	}

	req := httptest.NewRequest(http.MethodPut, ticket.UploadURL, bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT local upload=%d body=%s", rec.Code, rec.Body.String())
	}
	got, err := ReadArtifactBytes(context.Background(), ticket.GCSPath)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("ReadArtifactBytes mismatch: got=%q err=%v", got, err)
	}

	overReq := httptest.NewRequest(http.MethodPut, ticket.UploadURL, bytes.NewReader(append(append([]byte(nil), payload...), '!')))
	overRec := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(overRec, overReq)
	if overRec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 on oversized upload, got %d body=%s", overRec.Code, overRec.Body.String())
	}

	// 2. Tampered signature -> 403 invalid_upload_signature
	badReq := httptest.NewRequest(http.MethodPut, ticket.UploadURL+"tampered", bytes.NewReader(payload))
	badRec := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(badRec, badReq)
	if badRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on bad signature, got %d body=%s", badRec.Code, badRec.Body.String())
	}

	// 3. Expired ticket -> 403 upload_ticket_expired
	hExpired := NewHandlers(HandlerDeps{
		JWTSecret: "local-test-secret-0123456789abcdef",
		Now:       func() time.Time { return now.Add(2 * time.Hour) },
	})
	expReq := httptest.NewRequest(http.MethodPut, ticket.UploadURL, bytes.NewReader(payload))
	expRec := httptest.NewRecorder()
	hExpired.HandleLocalArtifactUpload(expRec, expReq)
	if expRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on expired ticket, got %d body=%s", expRec.Code, expRec.Body.String())
	}

	// 4. Resumable POST start -> Location -> PUT Content-Range 308 -> 200
	resumableTicket, err := CreateArtifactUploadTicketWithLocalFallback(
		"cell-uz", "sup-1", "proj-1", "bnd-1",
		ArtifactDescriptor{ArtifactID: "art-local-chunk", FileName: "big.csv", ByteSize: ResumableUploadThreshold + 1024},
		now, nil, "http://127.0.0.1:8080", h.localUploadSecret,
	)
	if err != nil || !resumableTicket.Resumable {
		t.Fatalf("resumable ticket err=%v ticket=%+v", err, resumableTicket)
	}
	startReq := httptest.NewRequest(http.MethodPost, resumableTicket.UploadURL, nil)
	startReq.Header.Set("x-goog-resumable", "start")
	startRec := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(startRec, startReq)
	if startRec.Code != http.StatusCreated || startRec.Header().Get("Location") == "" {
		t.Fatalf("resumable start=%d loc=%q body=%s", startRec.Code, startRec.Header().Get("Location"), startRec.Body.String())
	}
	sessionURL := startRec.Header().Get("Location")
	c1 := []byte("0123456789")
	c2 := []byte("abcdef")
	put1 := httptest.NewRequest(http.MethodPut, sessionURL, bytes.NewReader(c1))
	put1.Header.Set("Content-Range", "bytes 0-9/16")
	rec1 := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(rec1, put1)
	if rec1.Code != 308 {
		t.Fatalf("intermediate chunk want 308, got %d body=%s", rec1.Code, rec1.Body.String())
	}
	put2 := httptest.NewRequest(http.MethodPut, sessionURL, bytes.NewReader(c2))
	put2.Header.Set("Content-Range", "bytes 10-15/16")
	rec2 := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(rec2, put2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("final chunk want 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	chunkedData, err := ReadArtifactBytes(context.Background(), resumableTicket.GCSPath)
	if err != nil || !bytes.Equal(chunkedData, append(c1, c2...)) {
		t.Fatalf("chunked ReadArtifactBytes=%q err=%v", chunkedData, err)
	}

	// 5. Enforced env fails closed on HandleLocalArtifactUpload
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	enfReq := httptest.NewRequest(http.MethodPut, ticket.UploadURL, bytes.NewReader(payload))
	enfRec := httptest.NewRecorder()
	h.HandleLocalArtifactUpload(enfRec, enfReq)
	if enfRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when REQUIRE_INFRA_ADAPTERS=true, got %d body=%s", enfRec.Code, enfRec.Body.String())
	}
}

func TestDeriveLocalUploadBaseURL_IgnoresUnprefixedProxyForwardedHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/v1/supplier/migrations/projects/p1/bundles", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-Host", "localhost:3000")

	if got := deriveLocalUploadBaseURL(req); got != "http://127.0.0.1:8080" {
		t.Fatalf("expected direct listener base URL, got %q", got)
	}

	req.Header.Set("X-Forwarded-Prefix", "/api")
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := deriveLocalUploadBaseURL(req); got != "https://localhost:3000/api" {
		t.Fatalf("expected prefixed proxy base URL, got %q", got)
	}
}
