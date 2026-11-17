package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultConnectorHTTPTimeout = 10 * time.Second

var (
	pooledTransportOnce      sync.Once
	pooledConnectorTransport *http.Transport
	pooledClientsMu          sync.Mutex
	pooledClientsByTimeout   = map[time.Duration]*http.Client{}
)

func allowPrivateConnectorHostsFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CONNECTOR_ALLOW_PRIVATE_HOSTS")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func isAlwaysBlockedHostname(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "metadata.google.internal", "metadata", "kubernetes.default", "kubernetes.default.svc":
		return true
	}
	return strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal")
}

func isLoopbackHostname(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "localhost" || h == "localhost.localdomain" || strings.HasSuffix(h, ".localhost")
}

func isConnectorBlockedIP(ip net.IP, allowPrivate bool) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
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
	if !allowPrivate && (ip.IsLoopback() || ip.IsPrivate()) {
		return true
	}
	return false
}

// ValidateConnectorBaseURL blocks metadata/internal hosts, credentials in URLs,
// non-HTTP(S) schemes, and (unless allowPrivate) loopback/RFC1918 destinations.
func ValidateConnectorBaseURL(rawURL string, allowPrivate bool) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Errorf("%w: base_url is required", ErrInvalidInput)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("%w: invalid connector base_url %q", ErrInvalidInput, trimmed)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: connector base_url must not embed credentials", ErrInvalidInput)
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return fmt.Errorf("%w: invalid connector host", ErrInvalidInput)
	}
	if isAlwaysBlockedHostname(host) {
		return fmt.Errorf("%w: connector host %q is forbidden", ErrInvalidInput, host)
	}
	if !allowPrivate && isLoopbackHostname(host) {
		return fmt.Errorf("%w: connector loopback host %q is forbidden", ErrInvalidInput, host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isConnectorBlockedIP(ip, allowPrivate) {
			return fmt.Errorf("%w: connector IP %s is forbidden by SSRF policy", ErrInvalidInput, ip.String())
		}
		return nil
	}
	if !strings.HasSuffix(host, ".test") && !strings.HasSuffix(host, ".invalid") && !strings.HasSuffix(host, ".example") {
		dnsCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		ips, lerr := net.DefaultResolver.LookupIPAddr(dnsCtx, host)
		cancel()
		if lerr == nil && len(ips) > 0 {
			for _, ipAddr := range ips {
				if isConnectorBlockedIP(ipAddr.IP, allowPrivate) {
					return fmt.Errorf("%w: connector host %q resolved to forbidden IP %s", ErrInvalidInput, host, ipAddr.IP.String())
				}
			}
		}
	}
	return nil
}

func sharedConnectorTransport() *http.Transport {
	pooledTransportOnce.Do(func() {
		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		pooledConnectorTransport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				allowPrivate := allowPrivateConnectorHostsFromEnv()
				if isAlwaysBlockedHostname(host) || (!allowPrivate && isLoopbackHostname(host)) {
					return nil, fmt.Errorf("%w: connector host %q blocked at dial", ErrInvalidInput, host)
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ipAddr := range ips {
					if isConnectorBlockedIP(ipAddr.IP, allowPrivate) {
						return nil, fmt.Errorf("%w: connector target IP %s blocked by SSRF policy", ErrInvalidInput, ipAddr.IP.String())
					}
				}
				if len(ips) == 0 {
					return nil, fmt.Errorf("%w: connector host %q resolved no addresses", ErrInvalidInput, host)
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		}
	})
	return pooledConnectorTransport
}

func defaultPooledConnectorClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultConnectorHTTPTimeout
	}
	pooledClientsMu.Lock()
	defer pooledClientsMu.Unlock()
	if c, ok := pooledClientsByTimeout[timeout]; ok {
		return c
	}
	c := &http.Client{
		Timeout:   timeout,
		Transport: sharedConnectorTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("%w: too many connector redirects", ErrConnectorUpstreamError)
			}
			return ValidateConnectorBaseURL(req.URL.String(), allowPrivateConnectorHostsFromEnv())
		},
	}
	pooledClientsByTimeout[timeout] = c
	return c
}

func selectConnectorHTTPClient(injected *http.Client, timeout time.Duration) (*http.Client, bool) {
	if injected != nil {
		return injected, true
	}
	return defaultPooledConnectorClient(timeout), false
}

// resolveEndpointConfig resolves the connector baseURL and secret token, failing closed
// with ErrConnectorNotConfigured whenever either is unset (including in PEGASUSX_ENV=production).
func resolveEndpointConfig(
	ctx context.Context,
	cred ExternalCredential,
	secrets SecretResolver,
	envBaseURL func() string,
	envToken func() string,
	provider string,
	baseEnvName string,
	tokenEnvName string,
	allowPrivate ...bool,
) (string, string, error) {
	baseEnv := ""
	if envBaseURL != nil {
		baseEnv = envBaseURL()
	}
	baseURL := strings.TrimRight(firstNonEmpty(cred.BaseURL, baseEnv), "/")
	if baseURL == "" {
		return "", "", fmt.Errorf("%w: %s base_url or %s is required", ErrConnectorNotConfigured, provider, baseEnvName)
	}
	allowPriv := allowPrivateConnectorHostsFromEnv()
	if len(allowPrivate) > 0 && allowPrivate[0] {
		allowPriv = true
	}
	if err := ValidateConnectorBaseURL(baseURL, allowPriv); err != nil {
		return "", "", err
	}

	var token string
	if strings.TrimSpace(cred.SecretRef) != "" {
		if secrets == nil {
			secrets = NewDefaultSecretResolver(nil)
		}
		var resolved string
		var resErr error
		if scoped, ok := secrets.(interface {
			ResolveForTenant(ctx context.Context, tenantID, secretRef string) (string, error)
		}); ok {
			resolved, resErr = scoped.ResolveForTenant(ctx, cred.TenantID, cred.SecretRef)
		} else {
			resolved, resErr = secrets.Resolve(ctx, cred.SecretRef)
		}
		if resErr != nil {
			return "", "", resErr
		}
		token = resolved
	} else if envToken != nil {
		token = strings.TrimSpace(envToken())
	}
	if token == "" {
		return "", "", fmt.Errorf("%w: %s secret_ref or %s is required", ErrConnectorNotConfigured, provider, tokenEnvName)
	}
	return baseURL, token, nil
}

// doHTTPRequest executes an outbound connector HTTP call with standard auth headers,
// secret-scrubbed error wrapping, timeout mapping, and status classification.
func doHTTPRequest(
	ctx context.Context,
	httpClient *http.Client,
	provider string,
	method string,
	rawURL string,
	authType string,
	token string,
	contentType string,
	body []byte,
	extraHeaders map[string]string,
) (int, []byte, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: build %s request: %v", ErrInvalidInput, provider, err)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range extraHeaders {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	applyAuthHeader(req, authType, token)

	client, _ := selectConnectorHTTPClient(httpClient, defaultConnectorHTTPTimeout)
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, nil, fmt.Errorf("%w: %s request timed out", ErrConnectorTimeout, provider)
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return 0, nil, fmt.Errorf("%w: %s network timeout", ErrConnectorTimeout, provider)
		}
		return 0, nil, fmt.Errorf("%w: %s request error: %s", ErrConnectorUpstreamError, provider, scrubSecrets(err.Error(), token))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if readErr != nil {
		return resp.StatusCode, nil, fmt.Errorf("%w: read %s body: %v", ErrConnectorUpstreamError, provider, readErr)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.StatusCode, respBody, fmt.Errorf("%w: %s returned HTTP %d", ErrConnectorAuthFailed, provider, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, respBody, fmt.Errorf("%w: %s returned HTTP %d", ErrConnectorUpstreamError, provider, resp.StatusCode)
	}
	return resp.StatusCode, respBody, nil
}

// extractJSONEnvelopeRows unpacks standard list/object envelopes from JSON APIs without floats.
func extractJSONEnvelopeRows(body []byte, provider string, candidateKeys ...string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("%w: empty %s response body", ErrConnectorBadPayload, provider)
	}
	var raw any
	if err := decodeJSONWithNumbers([]byte(trimmed), &raw); err != nil {
		return nil, fmt.Errorf("%w: decode %s json: %v", ErrConnectorBadPayload, provider, err)
	}
	switch tv := raw.(type) {
	case []any:
		return toMapSlice(tv)
	case map[string]any:
		keys := append(append([]string(nil), candidateKeys...), "rows", "items", "data", "results")
		for _, k := range keys {
			if k == "" {
				continue
			}
			if sub, ok := tv[k]; ok && sub != nil {
				if arr, isArr := sub.([]any); isArr {
					return toMapSlice(arr)
				}
			}
		}
		if len(tv) == 0 {
			return []map[string]any{}, nil
		}
		if _, hasErr := tv["error"]; hasErr {
			return nil, fmt.Errorf("%w: %s upstream error payload", ErrConnectorUpstreamError, provider)
		}
		if _, hasErrs := tv["errors"]; hasErrs {
			return nil, fmt.Errorf("%w: %s upstream error payload", ErrConnectorUpstreamError, provider)
		}
		return []map[string]any{}, nil
	default:
		return nil, fmt.Errorf("%w: unexpected %s json root %T", ErrConnectorBadPayload, provider, raw)
	}
}

// prepareJSONPushBody validates RawPayload (zero floats) or encodes records into a clean integer-only JSON envelope.
func prepareJSONPushBody(provider string, cred ExternalCredential, req PushRequest, stream string) ([]byte, int64, error) {
	if len(bytes.TrimSpace(req.RawPayload)) > 0 {
		var generic any
		if err := decodeJSONWithNumbers(req.RawPayload, &generic); err != nil {
			return nil, 0, fmt.Errorf("%w: invalid %s push json payload: %v", ErrInvalidInput, provider, err)
		}
		if err := validateNoFloats(generic); err != nil {
			return nil, 0, err
		}
		count := int64(len(req.Records))
		if count == 0 {
			count = 1
		}
		return req.RawPayload, count, nil
	}
	if len(req.Records) == 0 {
		return nil, 0, fmt.Errorf("%w: %s push requires records or raw_payload", ErrInvalidInput, provider)
	}
	items := make([]map[string]any, 0, len(req.Records))
	for _, rec := range req.Records {
		item := map[string]any{
			"object_type":      firstNonEmpty(rec.ObjectType, ObjectTypeInventory),
			"external_id":      firstNonEmpty(rec.ExternalID, rec.SKU, rec.OrderID),
			"quantity_on_hand": rec.QuantityOnHand,
			"price_minor":      rec.PriceMinor,
			"total_minor":      rec.TotalMinor,
			"balance_minor":    rec.BalanceMinor,
		}
		if rec.SKU != "" {
			item["sku"] = rec.SKU
		}
		if rec.Barcode != "" {
			item["barcode"] = rec.Barcode
		}
		if rec.Name != "" {
			item["name"] = rec.Name
		}
		if rec.WarehouseID != "" {
			item["warehouse_id"] = rec.WarehouseID
		}
		if rec.OrderID != "" {
			item["order_id"] = rec.OrderID
		}
		if rec.RetailerID != "" {
			item["retailer_id"] = rec.RetailerID
		}
		if rec.Currency != "" {
			item["currency"] = rec.Currency
		}
		if rec.Status != "" {
			item["status"] = rec.Status
		}
		items = append(items, item)
	}
	payload := map[string]any{
		"provider":      provider,
		"account_label": normalizeAccountLabel(cred.AccountLabel),
		"stream":        stream,
		"approval_id":   strings.TrimSpace(req.ApprovalID),
		"items":         items,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: marshal %s push payload: %v", ErrInvalidInput, provider, err)
	}
	return raw, int64(len(req.Records)), nil
}

// extractExternalDocID reads a document/record ID from a JSON push response body, or falls back deterministically.
func extractExternalDocID(body []byte, prefix string, now time.Time) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed != "" {
		var m map[string]any
		if err := decodeJSONWithNumbers([]byte(trimmed), &m); err == nil && m != nil {
			if id := firstMapStr(m, "external_doc_id", "document_id", "doc_id", "id", "number"); id != "" {
				return id
			}
			for _, nest := range []string{"data", "result", "document"} {
				if sub, ok := m[nest].(map[string]any); ok {
					if id := firstMapStr(sub, "external_doc_id", "document_id", "doc_id", "id"); id != "" {
						return id
					}
				}
			}
		}
	}
	return fmt.Sprintf("%s-%d", prefix, now.UnixNano())
}
