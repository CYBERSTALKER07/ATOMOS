package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	gcs "cloud.google.com/go/storage"
)

func TestIsPlaceholderMediaURL(t *testing.T) {
	if !IsPlaceholderMediaURL("") {
		t.Fatal("empty should be placeholder")
	}
	if !IsPlaceholderMediaURL("https://placehold.co/400x400?text=x") {
		t.Fatal("placehold.co should be placeholder")
	}
	if IsPlaceholderMediaURL("https://storage.googleapis.com/bucket/obj.jpg") {
		t.Fatal("gcs should not be placeholder")
	}
}

func TestValidateEvidenceURI(t *testing.T) {
	BucketName = "pegasusx-evidence"
	t.Cleanup(func() { BucketName = "" })

	if err := ValidateEvidenceURI("https://placehold.co/x"); !errors.Is(err, ErrInvalidEvidenceURI) {
		t.Fatalf("want invalid for placehold, got %v", err)
	}
	if err := ValidateEvidenceURI("https://storage.googleapis.com/pegasusx-evidence/evidence/claims/a.jpg"); err != nil {
		t.Fatalf("want ok for bucket path: %v", err)
	}
	if err := ValidateEvidenceURI("https://storage.googleapis.com/pegasusx-evidence/claims/ret-1/dos-1/a.jpg"); err != nil {
		t.Fatalf("want ok for claims path: %v", err)
	}
	if err := ValidateEvidenceURI("https://storage.googleapis.com/other-bucket/a.jpg"); !errors.Is(err, ErrInvalidEvidenceURI) {
		t.Fatalf("want invalid for wrong bucket, got %v", err)
	}
	for _, bad := range []string{
		"https://storage.googleapis.com/pegasusx-evidence/../../other-bucket/a.jpg",
		"https://storage.googleapis.com/pegasusx-evidence/claims/%2e%2e/other/a.jpg",
		"https://storage.googleapis.com/pegasusx-evidence/claims\\evil.jpg",
		"https://storage.googleapis.com/pegasusx-evidence/",
		"http://storage.googleapis.com/pegasusx-evidence/claims/a.jpg",
	} {
		if err := ValidateEvidenceURI(bad); !errors.Is(err, ErrInvalidEvidenceURI) {
			t.Fatalf("want invalid for %q, got %v", bad, err)
		}
	}
}

func TestEvidenceFailClosed_RequireInfra(t *testing.T) {
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	t.Setenv("PEGASUSX_ENV", "local")
	if !EvidenceFailClosed() {
		t.Fatal("REQUIRE_INFRA_ADAPTERS=true must fail closed")
	}
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !EvidenceFailClosed() {
		t.Fatal("PEGASUSX_ENV=ssmr must fail closed")
	}
	t.Setenv("PEGASUSX_ENV", "sandbox")
	if !EvidenceFailClosed() {
		t.Fatal("PEGASUSX_ENV=sandbox must fail closed")
	}
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "local")
	if EvidenceFailClosed() {
		t.Fatal("local + adapters false must allow placeholders")
	}
}

func TestGenerateUploadTicketFor_FailClosedNoClient(t *testing.T) {
	prev := Client
	Client = nil
	t.Cleanup(func() { Client = prev })
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	_, _, err := GenerateUploadTicketFor("evidence/claims/x", "jpg")
	if !errors.Is(err, ErrMediaStorageUnavailable) {
		t.Fatalf("want media_storage_unavailable, got %v", err)
	}
}

func TestGenerateUploadTicketFor_ExtensionAllowlistAndPrefixBounds(t *testing.T) {
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "local")

	for _, badExt := range []string{"", "svg", "html", "exe", "js", "sh", "pdf", "../jpg"} {
		if _, _, err := GenerateUploadTicketFor("claims/ret-1/dos-1", badExt); !errors.Is(err, ErrInvalidUploadExtension) {
			t.Fatalf("extension %q: want ErrInvalidUploadExtension, got %v", badExt, err)
		}
	}
	for _, badPrefix := range []string{"", " ", "/", "../escape", "claims/../other", `claims\win`} {
		if _, _, err := GenerateUploadTicketFor(badPrefix, "jpg"); !errors.Is(err, ErrInvalidUploadPrefix) {
			t.Fatalf("prefix %q: want ErrInvalidUploadPrefix, got %v", badPrefix, err)
		}
	}
	for _, okExt := range []string{"jpg", ".jpeg", "PNG", "webp"} {
		up, pub, err := GenerateUploadTicketFor("claims/ret-1/dos-1", okExt)
		if err != nil {
			t.Fatalf("extension %q: unexpected err: %v", okExt, err)
		}
		if up == "" || pub == "" {
			t.Fatalf("extension %q: empty urls", okExt)
		}
	}
}

func TestValidateUploadSizeBounds(t *testing.T) {
	for _, bad := range []int64{-1, 0, MaxEvidenceUploadBytes + 1} {
		if err := ValidateUploadSize(bad); !errors.Is(err, ErrInvalidUploadSize) {
			t.Fatalf("size %d: want ErrInvalidUploadSize, got %v", bad, err)
		}
	}
	for _, ok := range []int64{MinEvidenceUploadBytes, 1024, MaxEvidenceUploadBytes} {
		if err := ValidateUploadSize(ok); err != nil {
			t.Fatalf("size %d: unexpected err: %v", ok, err)
		}
	}
}

func TestGenerateUploadTicketFor_SignedURLSecurityHeadersAndUUIDName(t *testing.T) {
	prevBucket := BucketName
	prevSigner := signUploadURL
	BucketName = "pegasusx-evidence"
	t.Cleanup(func() {
		BucketName = prevBucket
		signUploadURL = prevSigner
	})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	var capturedObject string
	var capturedOpts *gcs.SignedURLOptions
	signUploadURL = func(bucket, objectName string, opts *gcs.SignedURLOptions) (string, error) {
		capturedObject = objectName
		capturedOpts = opts
		opts.GoogleAccessID = "signer@pegasusx-test.iam.gserviceaccount.com"
		opts.PrivateKey = pemKey
		return gcs.SignedURL(bucket, objectName, opts)
	}

	uploadURL, publicURL, err := GenerateUploadTicketFor("claims/ret-123/dos-456", "png")
	if err != nil {
		t.Fatalf("GenerateUploadTicketFor: %v", err)
	}
	if !strings.HasPrefix(capturedObject, "claims/ret-123/dos-456/") || !strings.HasSuffix(capturedObject, ".png") {
		t.Fatalf("unexpected objectName %q", capturedObject)
	}
	objectFile := strings.TrimPrefix(capturedObject, "claims/ret-123/dos-456/")
	if !regexp.MustCompile(`^[0-9a-f]{32}\.png$`).MatchString(objectFile) {
		t.Fatalf("expected 32-hex UUID filename, got %q", objectFile)
	}
	if publicURL != "https://storage.googleapis.com/pegasusx-evidence/"+capturedObject {
		t.Fatalf("unexpected publicURL %q", publicURL)
	}
	if capturedOpts.Scheme != gcs.SigningSchemeV4 || capturedOpts.Method != "PUT" {
		t.Fatalf("expected V4 PUT signed url, got scheme=%v method=%q", capturedOpts.Scheme, capturedOpts.Method)
	}
	if capturedOpts.ContentType != "image/png" {
		t.Fatalf("expected content-type image/png, got %q", capturedOpts.ContentType)
	}
	if !slices.Contains(capturedOpts.Headers, "x-goog-if-generation-match:0") {
		t.Fatalf("missing create-only x-goog-if-generation-match:0 header in %v", capturedOpts.Headers)
	}
	if !slices.Contains(capturedOpts.Headers, "x-goog-content-length-range:1,10485760") {
		t.Fatalf("missing 1..10 MiB x-goog-content-length-range header in %v", capturedOpts.Headers)
	}

	parsed, err := url.Parse(uploadURL)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	signedHeaders := parsed.Query().Get("X-Goog-SignedHeaders")
	for _, hdr := range []string{"content-type", "x-goog-content-length-range", "x-goog-if-generation-match"} {
		if !strings.Contains(signedHeaders, hdr) {
			t.Fatalf("signed URL missing required signed header %q in %q", hdr, signedHeaders)
		}
	}
}


type countingFailReader struct {
	remaining int64
	readBytes int64
}

func (c *countingFailReader) Read(p []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > c.remaining {
		n = c.remaining
	}
	for i := int64(0); i < n; i++ {
		p[i] = 'a'
	}
	c.remaining -= n
	c.readBytes += n
	return int(n), nil
}

func TestInspectEvidenceStream_BoundsAndHashes(t *testing.T) {
	uri := "https://storage.googleapis.com/pegasusx-evidence/claims/ret-1/photo.jpg"
	payload := []byte("evidence-image-bytes")
	sum := sha256.Sum256(payload)
	wantContentHash := hex.EncodeToString(sum[:])

	insp, err := InspectEvidenceStream(bytes.NewReader(payload), uri, 987654321)
	if err != nil {
		t.Fatalf("InspectEvidenceStream valid: %v", err)
	}
	if insp.ContentSha256 != wantContentHash {
		t.Fatalf("ContentSha256 = %q, want %q", insp.ContentSha256, wantContentHash)
	}
	if insp.ObjectGeneration != 987654321 || insp.SizeBytes != int64(len(payload)) {
		t.Fatalf("unexpected generation/size: %+v", insp)
	}
	if len(insp.UriSha256) != 64 {
		t.Fatalf("UriSha256 len = %d, want 64", len(insp.UriSha256))
	}

	// Canonicalization removes query string and normalizes host/scheme casing & double slashes.
	altURI := "HTTPS://STORAGE.GOOGLEAPIS.COM/pegasusx-evidence//claims/ret-1/photo.jpg?X-Goog-Signature=deadbeef"
	inspAlt, err := InspectEvidenceStream(bytes.NewReader(payload), altURI, 1)
	if err != nil {
		t.Fatalf("InspectEvidenceStream alt: %v", err)
	}
	if inspAlt.UriSha256 != insp.UriSha256 || inspAlt.CanonicalURI != uri {
		t.Fatalf("canonicalization mismatch: got (%q, %q) want (%q, %q)", inspAlt.CanonicalURI, inspAlt.UriSha256, uri, insp.UriSha256)
	}

	// Exact 10 MiB limit succeeds.
	exact := make([]byte, MaxEvidenceUploadBytes)
	exact[0] = 1
	exact[len(exact)-1] = 2
	if _, err := InspectEvidenceStream(bytes.NewReader(exact), uri, 2); err != nil {
		t.Fatalf("exact 10 MiB should pass: %v", err)
	}

	// Over 10 MiB stops bounded read at MaxEvidenceUploadBytes+1 and returns ErrInvalidUploadSize.
	huge := &countingFailReader{remaining: MaxEvidenceUploadBytes * 4}
	if _, err := InspectEvidenceStream(huge, uri, 3); !errors.Is(err, ErrInvalidUploadSize) {
		t.Fatalf("over 10 MiB error = %v, want ErrInvalidUploadSize", err)
	}
	if huge.readBytes > MaxEvidenceUploadBytes+1 {
		t.Fatalf("bounded reader read %d bytes > MaxEvidenceUploadBytes+1 (%d)", huge.readBytes, MaxEvidenceUploadBytes+1)
	}

	// Empty object fails closed.
	if _, err := InspectEvidenceStream(bytes.NewReader(nil), uri, 4); !errors.Is(err, ErrInvalidUploadSize) {
		t.Fatalf("0-byte error = %v, want ErrInvalidUploadSize", err)
	}
}

func TestInspectEvidenceObject_MockOpenerAndFailClosed(t *testing.T) {
	prevClient := Client
	Client = nil
	t.Cleanup(func() { Client = prevClient })
	prevBucket := BucketName
	BucketName = "pegasusx-evidence"
	t.Cleanup(func() { BucketName = prevBucket })

	uri := "https://storage.googleapis.com/pegasusx-evidence/claims/ret-1/box.jpg"

	// Fail closed when Client == nil and no opener hook is injected.
	if _, err := InspectEvidenceObject(context.Background(), uri); !errors.Is(err, ErrMediaStorageUnavailable) {
		t.Fatalf("nil client without opener: got %v want ErrMediaStorageUnavailable", err)
	}

	// Injectable offline opener works cleanly and enforces bounded read + generation capture.
	restore := SetOpenEvidenceObjectForTest(func(_ context.Context, bucket, object string) (io.ReadCloser, int64, error) {
		if bucket != "pegasusx-evidence" || object != "claims/ret-1/box.jpg" {
			return nil, 0, errors.New("unexpected bucket/object")
		}
		return io.NopCloser(strings.NewReader("jpeg-stream-content")), 555001, nil
	})
	defer restore()

	insp, err := InspectEvidenceObject(context.Background(), uri)
	if err != nil {
		t.Fatalf("InspectEvidenceObject with mock opener: %v", err)
	}
	wantSum := sha256.Sum256([]byte("jpeg-stream-content"))
	if insp.ContentSha256 != hex.EncodeToString(wantSum[:]) {
		t.Fatalf("ContentSha256 mismatch: %q", insp.ContentSha256)
	}
	if insp.ObjectGeneration != 555001 {
		t.Fatalf("ObjectGeneration = %d, want 555001", insp.ObjectGeneration)
	}
}
