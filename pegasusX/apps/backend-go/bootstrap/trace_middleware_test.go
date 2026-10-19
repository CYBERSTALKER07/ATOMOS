package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

func TestTraceMiddleware_W3CTraceparent_Extraction(t *testing.T) {
	t.Parallel()

	inputTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	inputSpanID := "00f067aa0ba902b7"
	inputHeader := "00-" + inputTraceID + "-" + inputSpanID + "-01"

	var observedTraceID string
	var observedTraceparent string

	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedTraceID = outbox.TraceIDFromContext(r.Context())
		observedTraceparent = TraceParentFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("traceparent", inputHeader)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if observedTraceID != inputTraceID {
		t.Fatalf("expected observedTraceID %q, got %q", inputTraceID, observedTraceID)
	}
	if !strings.Contains(observedTraceparent, inputTraceID) {
		t.Fatalf("expected observedTraceparent to contain %q, got %q", inputTraceID, observedTraceparent)
	}

	respTraceID := rec.Header().Get("X-Trace-Id")
	if respTraceID != inputTraceID {
		t.Fatalf("expected response X-Trace-Id %q, got %q", inputTraceID, respTraceID)
	}

	respTraceparent := rec.Header().Get("traceparent")
	if !strings.HasPrefix(respTraceparent, "00-"+inputTraceID+"-") {
		t.Fatalf("expected response traceparent to start with '00-%s-', got %q", inputTraceID, respTraceparent)
	}
}

func TestTraceMiddleware_LegacyXTraceID_Fallback(t *testing.T) {
	t.Parallel()

	customTraceID := "req-trace-legacy-12345"

	var observedTraceID string
	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedTraceID = outbox.TraceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Trace-Id", customTraceID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if observedTraceID != customTraceID {
		t.Fatalf("expected context trace ID %q, got %q", customTraceID, observedTraceID)
	}

	if got := rec.Header().Get("X-Trace-Id"); got != customTraceID {
		t.Fatalf("expected response X-Trace-Id %q, got %q", customTraceID, got)
	}

	respTraceparent := rec.Header().Get("traceparent")
	if respTraceparent == "" {
		t.Fatal("expected response to include generated traceparent header")
	}
	traceID, spanID, flags, ok := ParseTraceParent(respTraceparent)
	if !ok {
		t.Fatalf("expected valid traceparent, got %q", respTraceparent)
	}
	if len(traceID) != 32 || len(spanID) != 16 || len(flags) != 2 {
		t.Fatalf("invalid traceparent parts length: %s, %s, %s", traceID, spanID, flags)
	}
}

func TestTraceMiddleware_GeneratedWhenUnset(t *testing.T) {
	t.Parallel()

	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	respTraceID := rec.Header().Get("X-Trace-Id")
	if respTraceID == "" {
		t.Fatal("expected non-empty generated X-Trace-Id header")
	}

	respTraceparent := rec.Header().Get("traceparent")
	if respTraceparent == "" {
		t.Fatal("expected non-empty generated traceparent header")
	}

	traceID, spanID, flags, ok := ParseTraceParent(respTraceparent)
	if !ok {
		t.Fatalf("expected valid generated traceparent, got %q", respTraceparent)
	}
	if traceID != respTraceID && len(respTraceID) != 32 {
		t.Logf("generated X-Trace-Id: %s, traceparent traceID: %s", respTraceID, traceID)
	}
	if len(spanID) != 16 || flags != "01" {
		t.Fatalf("expected 16-hex span and '01' flags, got %s, %s", spanID, flags)
	}
}

func TestParseTraceParent_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantOK  bool
		wantTID string
		wantSID string
	}{
		{
			name:    "valid sampled",
			raw:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantOK:  true,
			wantTID: "4bf92f3577b34da6a3ce929d0e0e4736",
			wantSID: "00f067aa0ba902b7",
		},
		{
			name:    "valid unsampled",
			raw:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
			wantOK:  true,
			wantTID: "4bf92f3577b34da6a3ce929d0e0e4736",
			wantSID: "00f067aa0ba902b7",
		},
		{
			name:   "unsupported version ff",
			raw:    "ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantOK: false,
		},
		{
			name:   "all zero trace id",
			raw:    "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
			wantOK: false,
		},
		{
			name:   "all zero span id",
			raw:    "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
			wantOK: false,
		},
		{
			name:   "malformed characters",
			raw:    "00-4bf92f3577b34da6a3ce929d0e0e47zz-00f067aa0ba902b7-01",
			wantOK: false,
		},
		{
			name:   "too short",
			raw:    "00-4bf92f35-00f067aa-01",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tid, sid, _, ok := ParseTraceParent(tc.raw)
			if ok != tc.wantOK {
				t.Fatalf("ParseTraceParent(%q) ok = %v, want %v", tc.raw, ok, tc.wantOK)
			}
			if tc.wantOK {
				if tid != tc.wantTID || sid != tc.wantSID {
					t.Fatalf("ParseTraceParent(%q) = (%q, %q), want (%q, %q)", tc.raw, tid, sid, tc.wantTID, tc.wantSID)
				}
			}
		})
	}
}
