package auth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// MutationGuardConfig controls how mutation routes authenticate callers.
type MutationGuardConfig struct {
	// AllowBypass mounts protected handlers without auth (e2e/smoke only).
	AllowBypass bool
}

// ProtectMutations mounts a protected route subtree. When AllowBypass is false,
// callers must present a valid session (Bearer or supplier cookie via
// SessionAuth) and per-route RequireRole middleware enforces role scope.
func ProtectMutations(r chi.Router, cfg MutationGuardConfig, mount func(chi.Router)) {
	if cfg.AllowBypass {
		mount(r)
		return
	}
	r.Group(func(gr chi.Router) {
		gr.Use(RequireAnyAuthenticated())
		mount(gr)
	})
}

// RequireDeviceCert enforces mTLS or hardware TPM certificate validation for terminals
// in production (including behind TLS-terminating load balancers where r.TLS == nil).
func RequireDeviceCert(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsProduction() && !hasValidDeviceAttestation(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "hardware_attestation_required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hasValidDeviceAttestation(r *http.Request) bool {
	if r == nil {
		return false
	}
	now := time.Now().UTC()
	caPool := loadDeviceCAPool()
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		for _, cert := range r.TLS.PeerCertificates {
			if cert != nil && isValidDeviceCertificate(cert, caPool, now) {
				return true
			}
		}
	}
	if !isTrustedDeviceProxy(r) {
		return false
	}
	if certHeader := strings.TrimSpace(r.Header.Get("X-Device-Cert")); certHeader != "" {
		if isValidDeviceCertHeader(certHeader, caPool, now) {
			return true
		}
	}
	if fccHeader := strings.TrimSpace(r.Header.Get("X-Forwarded-Client-Cert")); fccHeader != "" {
		if isValidForwardedClientCert(fccHeader, caPool, now) {
			return true
		}
	}
	return false
}

func loadDeviceCAPool() *x509.CertPool {
	rawCA := strings.TrimSpace(os.Getenv("PAYLOADER_DEVICE_CA_PEM"))
	if rawCA == "" {
		rawCA = strings.TrimSpace(os.Getenv("DEVICE_CERT_CA_PEM"))
	}
	if rawCA == "" {
		return nil
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(strings.ReplaceAll(rawCA, `\n`, "\n"))) {
		// Fail closed on misconfigured CA: return empty non-nil pool that matches no certs.
		return x509.NewCertPool()
	}
	return pool
}

func isIPInCIDRs(ip net.IP, cidrsRaw string) bool {
	if ip == nil || strings.TrimSpace(cidrsRaw) == "" {
		return false
	}
	for _, entry := range strings.Split(cidrsRaw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, ipNet, err := net.ParseCIDR(entry); err == nil {
			if ipNet.Contains(ip) {
				return true
			}
			continue
		}
		if exact := net.ParseIP(entry); exact != nil && exact.Equal(ip) {
			return true
		}
	}
	return false
}

func isTrustedDeviceProxy(r *http.Request) bool {
	if r == nil {
		return false
	}
	cidrsRaw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if cidrsRaw == "" {
		return false
	}
	ip := parseRemoteIP(r.RemoteAddr)
	return isIPInCIDRs(ip, cidrsRaw)
}

func parseRemoteIP(remoteAddr string) net.IP {
	remote := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(remote); err == nil && host != "" {
		remote = host
	}
	return net.ParseIP(remote)
}

// ResolveTrustedClientIP returns the client IP using right-to-left X-Forwarded-For traversal
// only when the immediate peer (r.RemoteAddr) is inside TRUSTED_PROXY_CIDRS; otherwise r.RemoteAddr is returned.
func ResolveTrustedClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	peerIP := parseRemoteIP(r.RemoteAddr)
	cidrsRaw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if peerIP != nil && cidrsRaw != "" && isIPInCIDRs(peerIP, cidrsRaw) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				candidate := net.ParseIP(strings.TrimSpace(parts[i]))
				if candidate == nil {
					continue
				}
				if !isIPInCIDRs(candidate, cidrsRaw) {
					return candidate.String()
				}
			}
		}
	}
	if peerIP != nil {
		return peerIP.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func isValidDeviceCertificate(cert *x509.Certificate, caPool *x509.CertPool, now time.Time) bool {
	if !isValidX509TimeWindow(cert, now) {
		return false
	}
	if caPool != nil {
		if _, err := cert.Verify(x509.VerifyOptions{
			Roots:       caPool,
			CurrentTime: now,
			KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		}); err != nil {
			return false
		}
	}
	return true
}

func isValidX509TimeWindow(cert *x509.Certificate, now time.Time) bool {
	if cert == nil {
		return false
	}
	if !cert.NotBefore.IsZero() && now.Before(cert.NotBefore) {
		return false
	}
	if !cert.NotAfter.IsZero() && now.After(cert.NotAfter) {
		return false
	}
	return len(cert.Raw) > 0
}

func isRejectedPlaceholder(val string) bool {
	lower := strings.ToLower(strings.TrimSpace(val))
	switch lower {
	case "", "null", "undefined", "none", "nil", "false", "true", "mock", "test", "dummy", "dev":
		return true
	}
	return false
}

func isValidDeviceCertHeader(raw string, caPool *x509.CertPool, now time.Time) bool {
	if isRejectedPlaceholder(raw) {
		return false
	}
	normalizedPEM := strings.ReplaceAll(raw, `\n`, "\n")
	if strings.Contains(normalizedPEM, "-----BEGIN CERTIFICATE-----") {
		block, _ := pem.Decode([]byte(normalizedPEM))
		if block == nil || block.Type != "CERTIFICATE" {
			return false
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return false
		}
		return isValidDeviceCertificate(cert, caPool, now)
	}
	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) > 0 {
		if cert, parseErr := x509.ParseCertificate(decoded); parseErr == nil {
			return isValidDeviceCertificate(cert, caPool, now)
		}
	}
	// Never accept arbitrary opaque non-X.509 strings as hardware certificates.
	return false
}

func isValidForwardedClientCert(raw string, caPool *x509.CertPool, now time.Time) bool {
	if isRejectedPlaceholder(raw) {
		return false
	}
	if strings.Contains(raw, "-----BEGIN CERTIFICATE-----") {
		return isValidDeviceCertHeader(raw, caPool, now)
	}
	// Envoy / Cloud Armor / Istio XFCC format: comma/semicolon-separated key=value pairs.
	for _, entry := range strings.Split(raw, ",") {
		hasCert := false
		hasHash := false
		for _, part := range strings.Split(entry, ";") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(kv[0]))
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			if isRejectedPlaceholder(val) {
				continue
			}
			switch key {
			case "cert", "chain":
				if isValidDeviceCertHeader(val, caPool, now) {
					hasCert = true
				}
			case "hash":
				if caPool == nil && len(val) >= 32 {
					hasHash = true
				}
			}
		}
		if hasCert || (hasHash && caPool == nil) {
			return true
		}
	}
	return false
}
