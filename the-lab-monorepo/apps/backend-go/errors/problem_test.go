package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteProblem(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/orders/123", nil)

	WriteProblem(w, r, 404, "error/not-found", "Resource Not Found", "Order 123 does not exist")

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %s", ct)
	}

	var pd ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&pd); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if pd.Type != "error/not-found" {
		t.Errorf("type = %q, want error/not-found", pd.Type)
	}
	if pd.Title != "Resource Not Found" {
		t.Errorf("title = %q, want Resource Not Found", pd.Title)
	}
	if pd.Status != 404 {
		t.Errorf("status = %d, want 404", pd.Status)
	}
	if pd.Detail != "Order 123 does not exist" {
		t.Errorf("detail = %q", pd.Detail)
	}
	if pd.TraceID == "" {
		t.Error("trace_id should not be empty")
	}
	if pd.Instance != "/v1/orders/123" {
		t.Errorf("instance = %q, want /v1/orders/123", pd.Instance)
	}
}

func TestConvenienceWrappers(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(http.ResponseWriter, *http.Request)
		status int
		ptype  string
	}{
		{"BadRequest", func(w http.ResponseWriter, r *http.Request) { BadRequest(w, r, "bad") }, 400, "error/bad-request"},
		{"Unauthorized", func(w http.ResponseWriter, r *http.Request) { Unauthorized(w, r, "no token") }, 401, "error/unauthorized"},
		{"Forbidden", func(w http.ResponseWriter, r *http.Request) { Forbidden(w, r, "no access") }, 403, "error/forbidden"},
		{"NotFound", func(w http.ResponseWriter, r *http.Request) { NotFound(w, r, "gone") }, 404, "error/not-found"},
		{"Conflict", func(w http.ResponseWriter, r *http.Request) { Conflict(w, r, "dup") }, 409, "error/conflict"},
		{"InternalError", func(w http.ResponseWriter, r *http.Request) { InternalError(w, r, "oops") }, 500, "error/internal"},
		{"ServiceUnavailable", func(w http.ResponseWriter, r *http.Request) { ServiceUnavailable(w, r, "down") }, 503, "error/service-unavailable"},
		{"MethodNotAllowed", func(w http.ResponseWriter, r *http.Request) { MethodNotAllowed(w, r) }, 405, "error/method-not-allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.fn(w, r)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}

			var pd ProblemDetail
			if err := json.NewDecoder(w.Body).Decode(&pd); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if pd.Type != tt.ptype {
				t.Errorf("type = %q, want %q", pd.Type, tt.ptype)
			}
			if pd.TraceID == "" {
				t.Error("trace_id empty")
			}
		})
	}
}

func TestTooManyRequests_RetryAfter(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	TooManyRequests(w, r, 42)

	if w.Code != 429 {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if ra := w.Header().Get("Retry-After"); ra != "42" {
		t.Errorf("Retry-After = %q, want 42", ra)
	}
}

func TestWriteOperational_AllFields(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orders/checkout", nil)

	WriteOperational(w, r, ProblemDetail{
		Type:       "error/payment/auth-declined",
		Title:      "Payment Declined",
		Status:     402,
		Detail:     "Your card was declined by the issuing bank.",
		Code:       CodeGPAuthDeclined,
		MessageKey: MsgKeyCardDeclined,
		Retryable:  true,
		Action:     ActionSelectCard,
	})

	if w.Code != 402 {
		t.Fatalf("status = %d, want 402", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}

	var pd ProblemDetail
	if err := json.NewDecoder(w.Body).Decode(&pd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pd.Code != "GP_AUTH_DECLINED" {
		t.Errorf("code = %q, want GP_AUTH_DECLINED", pd.Code)
	}
	if pd.MessageKey != "payment.error.card_declined" {
		t.Errorf("message_key = %q, want payment.error.card_declined", pd.MessageKey)
	}
	if !pd.Retryable {
		t.Error("retryable should be true")
	}
	if pd.Action != "SELECT_CARD" {
		t.Errorf("action = %q, want SELECT_CARD", pd.Action)
	}
	if pd.TraceID == "" {
		t.Error("trace_id should be auto-generated")
	}
	if pd.Instance != "/v1/orders/checkout" {
		t.Errorf("instance = %q, want /v1/orders/checkout", pd.Instance)
	}
}

func TestWriteOperational_MinimalFields(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)

	WriteOperational(w, r, ProblemDetail{
		Type:   "error/internal",
		Title:  "Internal Server Error",
		Status: 500,
		Code:   CodeSpannerTimeout,
	})

	// Verify omitempty: message_key, retryable, action should be absent from JSON
	raw := w.Body.String()
	if strings.Contains(raw, "message_key") {
		t.Error("message_key should be omitted when empty")
	}
	if strings.Contains(raw, `"retryable"`) {
		t.Error("retryable should be omitted when false")
	}
	if strings.Contains(raw, `"action"`) {
		t.Error("action should be omitted when empty")
	}

	var pd ProblemDetail
	if err := json.Unmarshal([]byte(raw), &pd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pd.Code != "SPANNER_LOCK_TIMEOUT" {
		t.Errorf("code = %q, want SPANNER_LOCK_TIMEOUT", pd.Code)
	}
}

func TestWriteProblem_BackwardCompat(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)

	WriteProblem(w, r, 404, "error/not-found", "Not Found", "gone")

	raw := w.Body.String()
	if strings.Contains(raw, `"code":`) {
		t.Error("WriteProblem should not emit code field")
	}
	if strings.Contains(raw, "message_key") {
		t.Error("WriteProblem should not emit message_key field")
	}
	if strings.Contains(raw, `"retryable"`) {
		t.Error("WriteProblem should not emit retryable field")
	}
	if strings.Contains(raw, `"action"`) {
		t.Error("WriteProblem should not emit action field")
	}
}
