package fiscal

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

// buildTestP12 generates an RSA key + self-signed cert and encodes a real
// PKCS#12 container, so the signer is exercised against genuine key material.
// Returns the .p12 bytes.
func buildTestP12(t *testing.T, password string) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "PegasusX Test", SerialNumber: "TEST-123"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	p12, err := pkcs12.Modern.Encode(key, cert, nil, password)
	if err != nil {
		t.Fatalf("encode p12: %v", err)
	}
	return p12
}

// writeTestP12 writes the container to a temp file for file-based loaders.
func writeTestP12(t *testing.T, password string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "key.p12")
	if err := os.WriteFile(f, buildTestP12(t, password), 0o600); err != nil {
		t.Fatalf("write p12: %v", err)
	}
	return f
}

func loadTestSigner(t *testing.T, password string) *PKCS12Signer {
	t.Helper()
	signer, err := NewPKCS12Signer(buildTestP12(t, password), password)
	if err != nil {
		t.Fatalf("NewPKCS12Signer: %v", err)
	}
	return signer
}

func TestPKCS12Signer_RoundTrip(t *testing.T) {
	const pw = "test-pass"
	signer := loadTestSigner(t, pw)
	payload := []byte(`{"document":"ehf","amount":1000}`)
	sigBytes, err := signer.Sign(context.Background(), payload)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	sig := string(sigBytes)
	if !strings.HasPrefix(sig, "EIMZO.") {
		t.Fatalf("signature missing EIMZO prefix: %q", sig)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sig, "EIMZO."))
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := sha256.Sum256(payload)
	if err := rsa.VerifyPSS(&signer.key.PublicKey, crypto.SHA256, digest[:], raw, nil); err != nil {
		t.Fatalf("VerifyPSS: %v", err)
	}
	if subj := signer.CertSubject(); !strings.Contains(subj, "TEST-123") {
		t.Fatalf("CertSubject missing serial: %q", subj)
	}
}

func TestPKCS12Signer_FailClosed(t *testing.T) {
	if _, err := NewPKCS12Signer(nil, "x"); err == nil {
		t.Fatal("expected error for empty container")
	}
	// Wrong password must fail to decode.
	if _, err := NewPKCS12Signer(buildTestP12(t, "right"), "wrong"); err == nil {
		t.Fatal("expected error for wrong password")
	}
	if _, err := NewPKCS12SignerFromFile("", "x"); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := NewPKCS12SignerFromFile("/nonexistent/key.p12", "x"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSignerFromEnv_PKCS12MissingFile(t *testing.T) {
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_FILE", "")
	if _, err := SignerFromEnv("production"); err == nil {
		t.Fatal("expected error when pkcs12 file not set")
	}
}

func TestSignerFromEnv_PKCS12Loads(t *testing.T) {
	const pw = "env-pass"
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_FILE", writeTestP12(t, pw))
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_PASSWORD", pw)
	s, err := SignerFromEnv("production")
	if err != nil {
		t.Fatalf("SignerFromEnv: %v", err)
	}
	if _, ok := s.(*PKCS12Signer); !ok {
		t.Fatalf("expected *PKCS12Signer, got %T", s)
	}
}
