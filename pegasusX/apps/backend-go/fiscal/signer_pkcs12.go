package fiscal

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mozilla.org/pkcs7"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

// PKCS12Signer is the real E-IMZO EDS signer. It loads an RSA private key and
// certificate chain from a PKCS#12 (.pfx/.p12) container — the format Soliq /
// E-IMZO issue — and produces a Base64-encoded DER CMS/PKCS#7 SignedData payload
// over the canonical EHF bytes.
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
	key     *rsa.PrivateKey
	cert    *x509.Certificate
	caCerts []*x509.Certificate
}

// NewPKCS12Signer parses the container and validates the key material.
func NewPKCS12Signer(p12Data []byte, password string) (*PKCS12Signer, error) {
	if len(p12Data) == 0 {
		return nil, fmt.Errorf("pkcs12: empty container")
	}
	key, cert, caCerts, err := pkcs12.DecodeChain(p12Data, password)
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
	now := time.Now().UTC()
	if now.After(cert.NotAfter) {
		return nil, fmt.Errorf("pkcs12: certificate expired at %s", cert.NotAfter.Format(time.RFC3339))
	}
	if now.Before(cert.NotBefore) {
		return nil, fmt.Errorf("pkcs12: certificate not yet valid until %s", cert.NotBefore.Format(time.RFC3339))
	}
	if err := rsaKey.Validate(); err != nil {
		return nil, fmt.Errorf("pkcs12: invalid RSA key: %w", err)
	}
	return &PKCS12Signer{key: rsaKey, cert: cert, caCerts: caCerts}, nil
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

// Sign builds a Base64-encoded DER CMS/PKCS#7 SignedData envelope over payload
// using SHA-256 and the loaded E-IMZO leaf certificate and chain.
// The certificate Subject serial number is available via CertSubject for EHF
// signer identification.
func (s *PKCS12Signer) Sign(_ context.Context, payload []byte) ([]byte, error) {
	if s == nil || s.key == nil || s.cert == nil {
		return nil, fmt.Errorf("pkcs12: signer not initialized")
	}
	sd, err := pkcs7.NewSignedData(payload)
	if err != nil {
		return nil, fmt.Errorf("pkcs12: init signed data: %w", err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if len(s.caCerts) > 0 {
		err = sd.AddSignerChain(s.cert, s.key, s.caCerts, pkcs7.SignerInfoConfig{})
	} else {
		err = sd.AddSigner(s.cert, s.key, pkcs7.SignerInfoConfig{})
	}
	if err != nil {
		return nil, fmt.Errorf("pkcs12: add signer: %w", err)
	}
	der, err := sd.Finish()
	if err != nil {
		return nil, fmt.Errorf("pkcs12: finish signed data: %w", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(der)), nil
}

// CertSubject returns the certificate subject (serial number) for EHF signer
// identification, and the not-after for expiry surfacing.
func (s *PKCS12Signer) CertSubject() string {
	if s == nil || s.cert == nil {
		return ""
	}
	return s.cert.Subject.String()
}
