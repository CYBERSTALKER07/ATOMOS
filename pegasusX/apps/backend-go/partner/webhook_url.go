package partner

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// WebhookURLPolicy controls outbound webhook URL SSRF checks (P2-14).
type WebhookURLPolicy struct {
	// AllowHTTP permits http:// (default: https only). SSMR/local set via env.
	AllowHTTP bool
	// AllowPrivateHosts skips private/link-local IP rejection after DNS lookup
	// (SSMR host.docker.internal). Prefer HostAllowlist in prod.
	AllowPrivateHosts bool
	// HostAllowlist when non-empty: hostname must equal or be a subdomain of an entry.
	HostAllowlist []string
	// LookupIP overrides net.LookupIP (tests).
	LookupIP func(host string) ([]net.IP, error)
}

// DefaultWebhookURLPolicy reads PARTNER_WEBHOOK_* env knobs.
func DefaultWebhookURLPolicy() WebhookURLPolicy {
	return WebhookURLPolicy{
		AllowHTTP:         envTruthy("PARTNER_WEBHOOK_ALLOW_HTTP"),
		AllowPrivateHosts: envTruthy("PARTNER_WEBHOOK_ALLOW_PRIVATE"),
		HostAllowlist:     splitCSV(os.Getenv("PARTNER_WEBHOOK_HOST_ALLOWLIST")),
	}
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ValidateWebhookURL rejects non-https (unless allowed), credentials, and
// destinations that resolve to loopback / private / link-local / metadata IPs.
func ValidateWebhookURL(raw string, policy WebhookURLPolicy) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("invalid_url")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid_url")
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "https":
		// ok
	case "http":
		if !policy.AllowHTTP {
			return fmt.Errorf("webhook_url_https_required")
		}
	default:
		return fmt.Errorf("invalid_url")
	}
	if u.User != nil {
		return fmt.Errorf("webhook_url_credentials_forbidden")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("invalid_url")
	}
	if isBlockedHostname(host) {
		return fmt.Errorf("webhook_url_host_forbidden")
	}
	if len(policy.HostAllowlist) > 0 {
		if !hostAllowlisted(host, policy.HostAllowlist) {
			return fmt.Errorf("webhook_url_host_not_allowlisted")
		}
		// Allowlist is explicit trust — skip DNS private checks.
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if !policy.AllowPrivateHosts && isBlockedIP(ip) {
			return fmt.Errorf("webhook_url_private_ip_forbidden")
		}
		return nil
	}
	lookup := policy.LookupIP
	if lookup == nil {
		lookup = net.LookupIP
	}
	ips, err := lookup(host)
	if err != nil {
		return fmt.Errorf("webhook_url_dns_failed")
	}
	if len(ips) == 0 {
		return fmt.Errorf("webhook_url_dns_failed")
	}
	if policy.AllowPrivateHosts {
		return nil
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("webhook_url_private_ip_forbidden")
		}
	}
	return nil
}

func hostAllowlisted(host string, allow []string) bool {
	for _, a := range allow {
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

func isBlockedHostname(host string) bool {
	switch host {
	case "localhost", "localhost.localdomain", "metadata.google.internal",
		"metadata", "kubernetes.default", "kubernetes.default.svc":
		return true
	}
	if strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	return false
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// Cloud metadata / CGNAT commonly abused for SSRF.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 { // RFC 6598
			return true
		}
	}
	return false
}
