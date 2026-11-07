// Package as2 implements a pragmatic RFC 4130 AS2 transport for EDI-lite bytes.
// Not Drummond-certified — see docs/PARTNER_AS2.md.
package as2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.mozilla.org/pkcs7"
)

// Material holds loaded PEM cert/key for a station.
type Material struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

// LoadCertificatePEM parses a PEM certificate.
func LoadCertificatePEM(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("pem_cert_missing")
	}
	return x509.ParseCertificate(block.Bytes)
}

// LoadPrivateKeyPEM parses a PEM RSA private key (PKCS1 or PKCS8).
func LoadPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("pem_key_missing")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("pem_key_not_rsa")
	}
	return rsaKey, nil
}

// LoadMaterial loads cert + matching private key.
func LoadMaterial(certPEM, keyPEM []byte) (Material, error) {
	cert, err := LoadCertificatePEM(certPEM)
	if err != nil {
		return Material{}, err
	}
	key, err := LoadPrivateKeyPEM(keyPEM)
	if err != nil {
		return Material{}, err
	}
	return Material{Cert: cert, Key: key}, nil
}

// MICSHA256 returns "base64(SHA-256(content)), sha-256" for Received-Content-MIC.
func MICSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return base64.StdEncoding.EncodeToString(sum[:]) + ", sha-256"
}

// SignAttached creates PKCS7 signed-data (attached content).
func SignAttached(content []byte, m Material) ([]byte, error) {
	if m.Cert == nil || m.Key == nil {
		return nil, fmt.Errorf("signer_material_missing")
	}
	sd, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, err
	}
	if err := sd.AddSigner(m.Cert, m.Key, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, err
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	return sd.Finish()
}

// VerifyAttached verifies PKCS7 signed-data and returns the content.
func VerifyAttached(signed []byte, expectedSigner *x509.Certificate) ([]byte, error) {
	p7, err := pkcs7.Parse(signed)
	if err != nil {
		return nil, err
	}
	if expectedSigner != nil {
		p7.Certificates = []*x509.Certificate{expectedSigner}
	}
	if err := p7.Verify(); err != nil {
		return nil, err
	}
	return p7.Content, nil
}

// EncryptCMS encrypts content for recipient cert (enveloped-data).
func EncryptCMS(content []byte, recipient *x509.Certificate) ([]byte, error) {
	if recipient == nil {
		return nil, fmt.Errorf("recipient_missing")
	}
	return pkcs7.Encrypt(content, []*x509.Certificate{recipient})
}

// DecryptCMS decrypts enveloped-data with our material.
func DecryptCMS(enveloped []byte, m Material) ([]byte, error) {
	if m.Cert == nil || m.Key == nil {
		return nil, fmt.Errorf("decrypt_material_missing")
	}
	p7, err := pkcs7.Parse(enveloped)
	if err != nil {
		return nil, err
	}
	return p7.Decrypt(m.Cert, m.Key)
}

// SignThenEncrypt applies AS2 profile: sign then encrypt.
func SignThenEncrypt(content []byte, signer Material, recipient *x509.Certificate) ([]byte, error) {
	signed, err := SignAttached(content, signer)
	if err != nil {
		return nil, err
	}
	return EncryptCMS(signed, recipient)
}

// DecryptThenVerify reverses SignThenEncrypt.
func DecryptThenVerify(enveloped []byte, our Material, partnerCert *x509.Certificate) ([]byte, error) {
	signed, err := DecryptCMS(enveloped, our)
	if err != nil {
		return nil, err
	}
	return VerifyAttached(signed, partnerCert)
}

// GenerateSelfSignedRSA creates a throwaway RSA cert+key for unit tests.
func GenerateSelfSignedRSA(cn string) (Material, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Material{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return Material{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return Material{}, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return Material{}, err
	}
	return Material{Cert: cert, Key: key}, nil
}

// EncodeCertPEM encodes a certificate as PEM.
func EncodeCertPEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

// EncodeKeyPEM encodes an RSA private key as PEM.
func EncodeKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

// NormalizeAS2ID strips optional quotes/angle brackets around AS2 ids.
func NormalizeAS2ID(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "\"")
	v = strings.TrimSuffix(v, "\"")
	v = strings.TrimPrefix(v, "<")
	v = strings.TrimSuffix(v, ">")
	return strings.TrimSpace(v)
}
