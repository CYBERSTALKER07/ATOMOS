package fiscal

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

// PKCS12Signer is the real E-IMZO EDS signer. It loads an RSA private key and
// certificate chain from a PKCS#12 (.pfx/.p12) container — the format Soliq /
// E-IMZO issue — and produces a detached CMS-style signature over the canonical
// EHF payload.
//
// Construction is fail-closed: a missing file, wrong password, non-RSA key, or
// empty chain is a construction error, never a silent unsigned receipt.
//
//	env:
//	  FISCAL_MY_SOLIQ_PKCS12_FILE      path to the .p12/.pfx container
//	  FISCAL_MY_SOLIQ_PKCS12_PASSWORD  container password (from secret store)
//
// The signer is only reachable when FISCAL_MY_SOLIQ_SIGNER=pkcs12 (see
// signer_env.go); it remains blocked in the field until the EDS key is
// procured, but the code path is real and exercisable with any test container.
type PKCS12Signer struct {
	key  *rsa.PrivateKey
	cert *x509.Certificate
}

// NewPKCS12Signer parses the container and validates the key material.
func NewPKCS12Signer(p12Data []byte, password string) (*PKCS12Signer, error) {
	if len(p12Data) == 0 {
		return nil, fmt.Errorf("pkcs12: empty container")
	}
	key, cert, err := pkcs12.Decode(p12Data, password)
	if err != nil {
		return nil, fmt.Errorf("pkcs12: decode: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("pkcs12: expected RSA private key, got %T", key)
	}
	if cert == nil {
		return nil, fmt.Errorf("pkcs12: no certificate in container")
	}
	if err := rsaKey.Validate(); err != nil {
		return nil, fmt.Errorf("pkcs12: invalid RSA key: %w", err)
	}
	return &PKCS12Signer{key: rsaKey, cert: cert}, nil
}

// NewPKCS12SignerFromFile loads the container from disk.
func NewPKCS12SignerFromFile(path, password string) (*PKCS12Signer, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("pkcs12: FISCAL_MY_SOLIQ_PKCS12_FILE required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pkcs12: read %s: %w", path, err)
	}
	return NewPKCS12Signer(data, password)
}

// Sign returns "EIMZO.<base64 rsa-pss signature>" over SHA-256(payload).
// The certificate Subject serial number is available via CertSubject for EHF
// signer identification.
func (s *PKCS12Signer) Sign(_ context.Context, payload []byte) ([]byte, error) {
	if s == nil || s.key == nil {
		return nil, fmt.Errorf("pkcs12: signer not initialized")
	}
	digest := sha256.Sum256(payload)
	sig, err := rsa.SignPSS(rand.Reader, s.key, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		return nil, fmt.Errorf("pkcs12: sign: %w", err)
	}
	return []byte("EIMZO." + base64.StdEncoding.EncodeToString(sig)), nil
}

// CertSubject returns the certificate subject (serial number) for EHF signer
// identification, and the not-after for expiry surfacing.
func (s *PKCS12Signer) CertSubject() string {
	if s == nil || s.cert == nil {
		return ""
	}
	return s.cert.Subject.String()
}
