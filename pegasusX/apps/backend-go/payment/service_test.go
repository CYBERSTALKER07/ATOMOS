package payment

import (
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyGlobalPayBasicAuth(t *testing.T) {
	t.Parallel()

	secret := "test-gp-secret"
	valid := "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:"+secret))

	tests := []struct {
		name   string
		header string
		secret string
		want   bool
	}{
		{
			name:   "valid basic auth",
			header: valid,
			secret: secret,
			want:   true,
		},
		{
			name:   "wrong scheme",
			header: "Bearer token",
			secret: secret,
			want:   false,
		},
		{
			name:   "wrong credential",
			header: "Basic " + base64.StdEncoding.EncodeToString([]byte("Paycom:wrong")),
			secret: secret,
			want:   false,
		},
		{
			name:   "malformed base64",
			header: "Basic !!!",
			secret: secret,
			want:   false,
		},
		{
			name:   "empty secret",
			header: valid,
			secret: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := verifyGlobalPayBasicAuth(tt.header, tt.secret)
			if got != tt.want {
				t.Fatalf("verifyGlobalPayBasicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseGlobalPayWebhookRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		rawURL      string
		wantErr     bool
		errContains string
		assert      func(t *testing.T, req globalPayWebhookRequest)
	}{
		{
			name:   "valid json payload",
			body:   `{"session_id":"sess-1","transaction_id":"tx-1","status":"PAID","amount_minor":1200,"currency":"USD"}`,
			rawURL: "http://localhost/v1/webhooks/global-pay",
			assert: func(t *testing.T, req globalPayWebhookRequest) {
				t.Helper()
				if req.SessionID != "sess-1" || req.TransactionID != "tx-1" || req.Status != "PAID" {
					t.Fatalf("unexpected parsed request: %+v", req)
				}
				if req.AmountMinor != 1200 || req.Currency != "USD" {
					t.Fatalf("unexpected amount/currency: %+v", req)
				}
			},
		},
		{
			name:   "query fallback for transaction and status",
			body:   `{}`,
			rawURL: "http://localhost/v1/webhooks/global-pay?session_id=sess-q&payment_id=pay-q&state=CAPTURED&amount_minor=50&currency=UZS",
			assert: func(t *testing.T, req globalPayWebhookRequest) {
				t.Helper()
				if req.SessionID != "sess-q" || req.TransactionID != "pay-q" || req.Status != "CAPTURED" {
					t.Fatalf("unexpected query fallback parse: %+v", req)
				}
				if req.AmountMinor != 50 || req.Currency != "UZS" {
					t.Fatalf("unexpected query amount/currency: %+v", req)
				}
			},
		},
		{
			name:        "missing session id",
			body:        `{"transaction_id":"tx-1","status":"PAID"}`,
			rawURL:      "http://localhost/v1/webhooks/global-pay",
			wantErr:     true,
			errContains: "session_id is required",
		},
		{
			name:        "missing transaction id",
			body:        `{"session_id":"sess-1","status":"PAID"}`,
			rawURL:      "http://localhost/v1/webhooks/global-pay",
			wantErr:     true,
			errContains: "transaction_id",
		},
		{
			name:        "missing status",
			body:        `{"session_id":"sess-1","transaction_id":"tx-1"}`,
			rawURL:      "http://localhost/v1/webhooks/global-pay",
			wantErr:     true,
			errContains: "status is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("POST", tt.rawURL, strings.NewReader(tt.body))
			got, err := parseGlobalPayWebhookRequest([]byte(tt.body), r)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.assert != nil {
				tt.assert(t, got)
			}
		})
	}
}
