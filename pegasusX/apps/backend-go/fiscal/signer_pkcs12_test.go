package fiscal

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mozilla.org/pkcs7"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

// buildTestP12 generates an RSA key + self-signed cert and encodes a real
// PKCS#12 container, so the signer is exercised against genuine key material.
// Returns the .p12 bytes.
func buildTestP12(t *testing.T, password string) []byte {
	return buildTestP12WithDates(t, password, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
}

func buildTestP12WithDates(t *testing.T, password string, notBefore, notAfter time.Time) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "PegasusX Test", SerialNumber: "TEST-123"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
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
	if strings.HasPrefix(sig, "EIMZO.") {
		t.Fatalf("signature must be genuine CMS/PKCS#7, not EIMZO prefix stub: %q", sig)
	}
	der, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("decode base64 PKCS#7 signature: %v", err)
	}
	p7, err := pkcs7.Parse(der)
	if err != nil {
		t.Fatalf("pkcs7.Parse: %v", err)
	}
	if err := p7.Verify(); err != nil {
		t.Fatalf("p7.Verify: %v", err)
	}
	if !bytes.Equal(p7.Content, payload) {
		t.Fatalf("signed PKCS#7 content mismatch: got %q want %q", p7.Content, payload)
	}
	if signerCert := p7.GetOnlySigner(); signerCert == nil || !signerCert.Equal(signer.cert) {
		t.Fatalf("unexpected signer cert in PKCS#7: %v", signerCert)
	}
	// Mutating signed content after parsing must fail verification.
	p7.Content = []byte(`{"document":"ehf","amount":9999}`)
	if err := p7.Verify(); err == nil {
		t.Fatal("expected p7.Verify to fail on tampered content")
	}

	// Verify AttachSignature embeds a verifiable Base64 PKCS#7 envelope.
	attachedBytes, err := AttachSignature(context.Background(), signer, map[string]any{"amount": 1000, "document": "ehf"})
	if err != nil {
		t.Fatalf("AttachSignature: %v", err)
	}
	var env map[string]any
	if err := json.Unmarshal(attachedBytes, &env); err != nil {
		t.Fatalf("unmarshal attached payload: %v", err)
	}
	attachedSig, _ := env["signature"].(string)
	attachedDER, err := base64.StdEncoding.DecodeString(attachedSig)
	if err != nil {
		t.Fatalf("decode attached signature: %v", err)
	}
	attachedP7, err := pkcs7.Parse(attachedDER)
	if err != nil {
		t.Fatalf("parse attached PKCS#7: %v", err)
	}
	if err := attachedP7.Verify(); err != nil {
		t.Fatalf("verify attached PKCS#7: %v", err)
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

	// Expired certificate must fail closed
	expiredP12 := buildTestP12WithDates(t, "pwd", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	if _, err := NewPKCS12Signer(expiredP12, "pwd"); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected certificate expired error, got: %v", err)
	}

	// Not-yet-valid certificate must fail closed
	futureP12 := buildTestP12WithDates(t, "pwd", time.Now().Add(24*time.Hour), time.Now().Add(48*time.Hour))
	if _, err := NewPKCS12Signer(futureP12, "pwd"); err == nil || !strings.Contains(err.Error(), "not yet valid") {
		t.Fatalf("expected certificate not yet valid error, got: %v", err)
	}
}

func TestSignerFromEnv_DevHMACForbiddenInSandbox(t *testing.T) {
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "dev-hmac")
	t.Setenv("FISCAL_MY_SOLIQ_SIGN_KEY", "0123456789abcdef")
	for _, env := range []string{"sandbox", "ssmr", "production", "prod"} {
		if _, err := SignerFromEnv(env); err == nil {
			t.Fatalf("expected dev-hmac forbidden in %s", env)
		}
	}
}

func TestSignerFromEnv_PKCS12MissingFile(t *testing.T) {
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_FILE", "")
	if _, err := SignerFromEnv("production"); err == nil {
		t.Fatal("expected error when pkcs12 file not set")
	}
}

func TestSignerFromEnv_SandboxMYSoliqMissingCreds(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "sandbox")
	t.Setenv("FISCAL_PROVIDER", "MY_SOLIQ")
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "")
	if _, err := SignerFromEnv("sandbox"); err == nil {
		t.Fatal("sandbox MY_SOLIQ without signer must fail closed")
	}
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	t.Setenv("FISCAL_MY_SOLIQ_PKCS12_FILE", "")
	if _, err := SignerFromEnv("sandbox"); err == nil {
		t.Fatal("sandbox MY_SOLIQ pkcs12 without file must fail closed")
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
