package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVerifyGlobalPayBasicAuth(t *testing.T) {
	t.Parallel()

	secret := "gp-secret"
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
			rawURL: "http://example.test/v1/webhooks/global-pay",
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
			rawURL: "http://example.test/v1/webhooks/global-pay?session_id=sess-q&payment_id=pay-q&state=CAPTURED&amount_minor=50&currency=UZS",
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
			rawURL:      "http://example.test/v1/webhooks/global-pay",
			wantErr:     true,
			errContains: "session_id is required",
		},
		{
			name:        "missing transaction id",
			body:        `{"session_id":"sess-1","status":"PAID"}`,
			rawURL:      "http://example.test/v1/webhooks/global-pay",
			wantErr:     true,
			errContains: "transaction_id",
		},
		{
			name:        "missing status",
			body:        `{"session_id":"sess-1","transaction_id":"tx-1"}`,
			rawURL:      "http://example.test/v1/webhooks/global-pay",
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

func TestVerifyStripeSignatureHeader(t *testing.T) {
	t.Parallel()

	secret := "whsec_test"
	body := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)
	now := time.Unix(1_700_000_000, 0).UTC()

	buildHeader := func(ts time.Time, sigSecret string) string {
		timestamp := strconv.FormatInt(ts.Unix(), 10)
		signedPayload := timestamp + "." + string(body)
		mac := hmac.New(sha256.New, []byte(sigSecret))
		_, _ = mac.Write([]byte(signedPayload))
		sig := hex.EncodeToString(mac.Sum(nil))
		return "t=" + timestamp + ",v1=" + sig
	}

	validHeader := buildHeader(now, secret)

	tests := []struct {
		name   string
		header string
		secret string
		now    time.Time
		want   bool
	}{
		{
			name:   "valid signature",
			header: validHeader,
			secret: secret,
			now:    now,
			want:   true,
		},
		{
			name:   "stale timestamp",
			header: buildHeader(now.Add(-6*time.Minute), secret),
			secret: secret,
			now:    now,
			want:   false,
		},
		{
			name:   "missing v1",
			header: "t=" + strconv.FormatInt(now.Unix(), 10),
			secret: secret,
			now:    now,
			want:   false,
		},
		{
			name:   "signature mismatch",
			header: buildHeader(now, "wrong-secret"),
			secret: secret,
			now:    now,
			want:   false,
		},
		{
			name:   "empty secret",
			header: validHeader,
			secret: "",
			now:    now,
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := verifyStripeSignatureHeader(body, tt.header, tt.secret, tt.now)
			if got != tt.want {
				t.Fatalf("verifyStripeSignatureHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAdyenNotificationItem(t *testing.T) {
	t.Parallel()

	base := validAdyenItemForTest()

	tests := []struct {
		name        string
		mutate      func(item *adyenNotificationItem)
		errContains string
	}{
		{
			name: "valid item",
		},
		{
			name: "missing psp reference",
			mutate: func(item *adyenNotificationItem) {
				item.PspReference = ""
			},
			errContains: "pspReference is required",
		},
		{
			name: "missing event code",
			mutate: func(item *adyenNotificationItem) {
				item.EventCode = ""
			},
			errContains: "eventCode is required",
		},
		{
			name: "missing merchant reference",
			mutate: func(item *adyenNotificationItem) {
				item.MerchantReference = ""
			},
			errContains: "merchantReference is required",
		},
		{
			name: "missing merchant account",
			mutate: func(item *adyenNotificationItem) {
				item.MerchantAccountCode = ""
			},
			errContains: "merchantAccountCode is required",
		},
		{
			name: "missing success",
			mutate: func(item *adyenNotificationItem) {
				item.Success = ""
			},
			errContains: "success is required",
		},
		{
			name: "missing currency",
			mutate: func(item *adyenNotificationItem) {
				item.Amount.Currency = ""
			},
			errContains: "amount.currency is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := base
			if tt.mutate != nil {
				tt.mutate(&item)
			}
			err := validateAdyenNotificationItem(item)
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestVerifyAdyenNotificationSignature(t *testing.T) {
	t.Parallel()

	secret := "adyen-secret"
	item := validAdyenItemForTest()
	item.AdditionalData = map[string]string{}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(adyenSigningData(item)))
	item.AdditionalData["hmacSignature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name   string
		item   adyenNotificationItem
		secret string
		want   bool
	}{
		{
			name:   "valid signature",
			item:   item,
			secret: secret,
			want:   true,
		},
		{
			name:   "wrong secret",
			item:   item,
			secret: "wrong-secret",
			want:   false,
		},
		{
			name: "invalid base64 signature",
			item: func() adyenNotificationItem {
				copy := item
				copy.AdditionalData = map[string]string{"hmacSignature": "***"}
				return copy
			}(),
			secret: secret,
			want:   false,
		},
		{
			name: "missing additional data",
			item: func() adyenNotificationItem {
				copy := item
				copy.AdditionalData = nil
				return copy
			}(),
			secret: secret,
			want:   false,
		},
		{
			name:   "empty secret",
			item:   item,
			secret: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := verifyAdyenNotificationSignature(tt.item, tt.secret)
			if got != tt.want {
				t.Fatalf("verifyAdyenNotificationSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func validAdyenItemForTest() adyenNotificationItem {
	return adyenNotificationItem{
		PspReference:        "psp_123",
		OriginalReference:   "orig_001",
		MerchantReference:   "order_001",
		MerchantAccountCode: "merchant_001",
		EventCode:           "AUTHORISATION",
		Success:             "true",
		Amount: adyenAmount{
			Currency: "USD",
			Value:    1500,
		},
	}
}
