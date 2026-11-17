package as2

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client posts outbound AS2 messages.
type Client struct {
	HTTP              *http.Client
	Timeout           time.Duration
	AllowPrivateHosts bool
	AllowHTTP         bool
}

// NewClient returns an AS2 HTTP client with TLS 1.2+ and SSRF dial/redirect protections.
func NewClient() *Client {
	c := &Client{
		Timeout: 30 * time.Second,
	}
	c.HTTP = c.newGuardedHTTPClient()
	return c
}

// AllowPrivateForTest enables localhost/HTTP requests for hermetic unit tests using httptest.Server.
func (c *Client) AllowPrivateForTest() {
	if c == nil {
		return
	}
	c.AllowPrivateHosts = true
	c.AllowHTTP = true
	c.HTTP = c.newGuardedHTTPClient()
}

func (c *Client) newGuardedHTTPClient() *http.Client {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if c.AllowPrivateHosts {
					return dialer.DialContext(ctx, network, addr)
				}
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, fmt.Errorf("as2_ssrf_invalid_addr: %w", err)
				}
				hostLower := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
				if isAS2BlockedHost(hostLower) {
					return nil, fmt.Errorf("as2_url_host_forbidden")
				}
				if ip := net.ParseIP(hostLower); ip != nil {
					if isAS2BlockedIP(ip) {
						return nil, fmt.Errorf("as2_url_private_ip_forbidden")
					}
					return dialer.DialContext(ctx, network, addr)
				}
				addrs, err := net.DefaultResolver.LookupIPAddr(ctx, hostLower)
				if err != nil || len(addrs) == 0 {
					return nil, fmt.Errorf("as2_url_dns_failed")
				}
				for _, a := range addrs {
					if isAS2BlockedIP(a.IP) {
						return nil, fmt.Errorf("as2_url_private_ip_forbidden")
					}
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return validateAS2TargetURL(req.URL.String(), c.AllowHTTP, c.AllowPrivateHosts)
		},
	}
}

func validateAS2TargetURL(raw string, allowHTTP, allowPrivate bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("as2_url_invalid")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && !(allowHTTP && scheme == "http") {
		return fmt.Errorf("as2_url_https_required")
	}
	if u.User != nil {
		return fmt.Errorf("as2_url_credentials_forbidden")
	}
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(u.Hostname()), "."))
	if host == "" {
		return fmt.Errorf("as2_url_invalid")
	}
	if !allowPrivate {
		if isAS2BlockedHost(host) {
			return fmt.Errorf("as2_url_host_forbidden")
		}
		if ip := net.ParseIP(host); ip != nil && isAS2BlockedIP(ip) {
			return fmt.Errorf("as2_url_private_ip_forbidden")
		}
	}
	return nil
}

func isAS2BlockedHost(host string) bool {
	switch host {
	case "localhost", "localhost.localdomain", "metadata.google.internal",
		"metadata", "kubernetes.default", "kubernetes.default.svc":
		return true
	}
	return strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local")
}

func isAS2BlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

// SendRequest is an outbound AS2 POST.
type SendRequest struct {
	URL           string
	From          string
	To            string
	EDI           []byte
	Filename      string
	Plain         bool
	Signer        Material
	RecipientCert []byte // PEM
	RequestMDN    bool
}

// SendResult captures partner MDN response basics.
type SendResult struct {
	StatusCode int
	MessageID  string
	Body       []byte
	MDN        ParsedMDN
	ExpectedMIC string
}

// Send posts the EDI payload to the partner AS2 URL.
// When RequestMDN is true, a sync MDN is required: disposition must be
// "processed" and Received-Content-MIC must match SHA-256 MIC of the EDI body.
func (c *Client) Send(ctx context.Context, req SendRequest) (SendResult, error) {
	if c == nil {
		c = NewClient()
	}
	if strings.TrimSpace(req.URL) == "" {
		return SendResult{}, fmt.Errorf("as2_url_missing")
	}
	allowHTTP := c.AllowHTTP || (req.Plain && c.AllowPrivateHosts)
	if err := validateAS2TargetURL(req.URL, allowHTTP, c.AllowPrivateHosts); err != nil {
		return SendResult{}, err
	}
	filename := req.Filename
	if filename == "" {
		filename = "message.edi"
	}
	msgID := NewMessageID("pegasusx.local")
	expectedMIC := MICSHA256(req.EDI)

	var (
		ct   string
		body []byte
		err  error
	)
	if req.Plain {
		ct, body, _ = BuildPlainBody(req.EDI, filename)
	} else {
		ct, body, err = BuildEncryptedBody(req.EDI, req.Signer, req.RecipientCert, "smime.p7m")
		if err != nil {
			return SendResult{}, err
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.URL, bytes.NewReader(body))
	if err != nil {
		return SendResult{}, err
	}
	ApplyOutbound(httpReq.Header, req.From, req.To, msgID, req.RequestMDN)
	httpReq.Header.Set("Content-Type", ct)
	httpReq.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	httpReq.Header.Set("MIME-Version", "1.0")

	client := c.HTTP
	if client == nil {
		client = NewClient().HTTP
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return SendResult{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	out := SendResult{
		StatusCode:  resp.StatusCode,
		MessageID:   msgID,
		Body:        respBody,
		ExpectedMIC: expectedMIC,
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("as2_http_%d", resp.StatusCode)
	}
	if !req.RequestMDN {
		return out, nil
	}
	mdn, err := ParseSyncMDN(resp.Header.Get("Content-Type"), respBody)
	if err != nil {
		return out, fmt.Errorf("as2_mdn_parse: %w", err)
	}
	out.MDN = mdn
	if err := VerifyMDN(expectedMIC, mdn); err != nil {
		return out, err
	}
	return out, nil
}
