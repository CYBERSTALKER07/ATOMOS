package as2

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client posts outbound AS2 messages.
type Client struct {
	HTTP    *http.Client
	Timeout time.Duration
}

// NewClient returns an AS2 HTTP client with TLS 1.2+.
func NewClient() *Client {
	return &Client{
		Timeout: 30 * time.Second,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}
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
