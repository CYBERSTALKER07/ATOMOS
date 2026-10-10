package soliq

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

func TestClient_Submit_SuccessDirect(t *testing.T) {
	var gotAuth, gotIdem, gotContentType string
	var gotBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ehf/submit" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotIdem = r.Header.Get("Idempotency-Key")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"EHF-12345"}}`))
	}))
	defer ts.Close()

	c := NewClient(SoliqConfig{
		BaseURL: ts.URL,
		APIKey:  "test-secret-key",
		TIN:     "123456789",
		Timeout: 5 * time.Second,
	})

	resp, err := c.Submit(context.Background(), []byte(`{"doc":"content"}`), "attempt-uuid-1")
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected Success=true, got %v", resp.Success)
	}
	if resp.EhfID != "EHF-12345" {
		t.Fatalf("expected EhfID=EHF-12345, got %q", resp.EhfID)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected StatusCode=200, got %d", resp.StatusCode)
	}
	if gotAuth != "Bearer test-secret-key" {
		t.Fatalf("expected Bearer test-secret-key, got %q", gotAuth)
	}
	if gotIdem != "attempt-uuid-1" {
		t.Fatalf("expected attempt-uuid-1, got %q", gotIdem)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", gotContentType)
	}
	if string(gotBody) != `{"doc":"content"}` {
		t.Fatalf("expected payload match, got %s", string(gotBody))
	}
}

func TestClient_Submit_DidoxOperator(t *testing.T) {
	var pathHit string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathHit = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"DIDOX-999"}}`))
	}))
	defer ts.Close()

	c := NewClient(SoliqConfig{
		BaseURL:  ts.URL,
		Operator: "didox",
		APIKey:   "didox-key",
		TIN:      "987654321",
	})

	resp, err := c.Submit(context.Background(), []byte(`{}`), "attempt-2")
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}
	if pathHit != "/api/v1/documents" {
		t.Fatalf("expected didox path /api/v1/documents, got %s", pathHit)
	}
	if resp.EhfID != "DIDOX-999" {
		t.Fatalf("expected EhfID DIDOX-999, got %s", resp.EhfID)
	}
}

func TestClient_Submit_PermanentErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
		wantCode   string
	}{
		{
			name:       "400_Bad_Request",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"error":{"code":"ERR_SCHEMA","message":"Malformed payload"}}`,
			wantCode:   "ERR_SCHEMA",
		},
		{
			name:       "422_Unprocessable_Entity",
			statusCode: http.StatusUnprocessableEntity,
			body:       `{"success":false,"error":{"code":"INVALID_TIN","message":"TIN not registered"}}`,
			wantCode:   "INVALID_TIN",
		},
		{
			name:       "401_Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"success":false,"error":{"code":"AUTH_FAIL","message":"Invalid token"}}`,
			wantCode:   "AUTH_FAIL",
		},
		{
			name:       "403_Forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"success":false,"error":{"code":"FORBIDDEN","message":"Access denied"}}`,
			wantCode:   "FORBIDDEN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer ts.Close()

			c := NewClient(SoliqConfig{BaseURL: ts.URL, APIKey: "test"})
			resp, err := c.Submit(context.Background(), []byte(`{}`), "idem")
			if err != nil {
				t.Fatalf("Submit unexpected error: %v", err)
			}
			if resp.Success {
				t.Fatalf("expected Success=false")
			}
			if !resp.Permanent {
				t.Fatalf("expected Permanent=true for status %d", tc.statusCode)
			}
			if resp.ErrorCode != tc.wantCode {
				t.Fatalf("expected ErrorCode=%q, got %q", tc.wantCode, resp.ErrorCode)
			}
		})
	}
}

func TestClient_Submit_TransientErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "500_Internal_Error",
			statusCode: http.StatusInternalServerError,
			body:       `{"success":false,"error":{"code":"DOWN","message":"Database unavailable"}}`,
		},
		{
			name:       "502_Bad_Gateway",
			statusCode: http.StatusBadGateway,
			body:       `Bad Gateway`,
		},
		{
			name:       "503_Service_Unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `Service Unavailable`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer ts.Close()

			c := NewClient(SoliqConfig{BaseURL: ts.URL, APIKey: "test"})
			resp, err := c.Submit(context.Background(), []byte(`{}`), "idem")
			if err != nil {
				t.Fatalf("Submit unexpected error: %v", err)
			}
			if resp.Success {
				t.Fatalf("expected Success=false")
			}
			if resp.Permanent {
				t.Fatalf("expected Permanent=false (transient/retryable) for status %d", tc.statusCode)
			}
		})
	}
}

func TestClient_CheckStatus_DirectAndDidox(t *testing.T) {
	directHit := false
	didoxHit := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/ehf/EHF-100/status":
			directHit = true
			_, _ = w.Write([]byte(`{"success":true,"data":{"status":"ACCEPTED"}}`))
		case "/api/v1/documents/DIDOX-200/status":
			didoxHit = true
			_, _ = w.Write([]byte(`{"success":true,"data":{"status":"REJECTED"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	directClient := NewClient(SoliqConfig{BaseURL: ts.URL, APIKey: "key"})
	st1, err := directClient.CheckStatus(context.Background(), "EHF-100")
	if err != nil {
		t.Fatalf("CheckStatus direct error: %v", err)
	}
	if !directHit || st1.Status != "ACCEPTED" {
		t.Fatalf("direct status mismatch: hit=%v status=%s", directHit, st1.Status)
	}

	didoxClient := NewClient(SoliqConfig{BaseURL: ts.URL, Operator: "didox", APIKey: "key"})
	st2, err := didoxClient.CheckStatus(context.Background(), "DIDOX-200")
	if err != nil {
		t.Fatalf("CheckStatus didox error: %v", err)
	}
	if !didoxHit || st2.Status != "REJECTED" {
		t.Fatalf("didox status mismatch: hit=%v status=%s", didoxHit, st2.Status)
	}
}

func TestClient_CheckStatus_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`Gateway Timeout`))
	}))
	defer ts.Close()

	c := NewClient(SoliqConfig{BaseURL: ts.URL})
	_, err := c.CheckStatus(context.Background(), "EHF-ERR")
	if err == nil {
		t.Fatal("expected error on 504 Gateway Timeout")
	}
}

func TestClient_Submit_CircuitBreaker_TripsAndRecovers(t *testing.T) {
	var failMode atomic.Bool
	failMode.Store(true)
	var requestCount atomic.Int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if failMode.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"SERVICE_UNAVAILABLE","message":"government gateway down"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"EHF-RECOVERED-123"}}`))
	}))
	defer ts.Close()

	breaker := circuit.New("test-soliq-breaker", circuit.Config{
		FailureThreshold: 3,
		FailureWindow:    5 * time.Second,
		OpenDuration:     50 * time.Millisecond,
	})

	c := NewClientWithBreaker(SoliqConfig{
		BaseURL: ts.URL,
		Timeout: 2 * time.Second,
	}, breaker)

	ctx := context.Background()

	// 1. Initial 3 calls fail with 503 -> trips breaker
	for i := 1; i <= 3; i++ {
		resp, err := c.Submit(ctx, []byte(`{"test":true}`), "idem-key")
		if err == nil && (resp != nil && resp.Success) {
			t.Fatalf("call %d: expected failure", i)
		}
	}

	if breaker.State() != circuit.StateOpen {
		t.Fatalf("expected breaker StateOpen, got %v", breaker.State())
	}

	// 2. 4th call: breaker is OPEN -> fail fast without hitting upstream
	reqBefore := requestCount.Load()
	start := time.Now()
	resp, err := c.Submit(ctx, []byte(`{"test":true}`), "idem-key")
	duration := time.Since(start)

	if !errors.Is(err, circuit.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable when open, got %v", err)
	}
	if resp == nil || resp.ErrorCode != "SOLIQ_CIRCUIT_OPEN" {
		t.Fatalf("expected SOLIQ_CIRCUIT_OPEN, got %+v", resp)
	}
	if duration > 10*time.Millisecond {
		t.Fatalf("fail-fast took too long: %v", duration)
	}
	if requestCount.Load() != reqBefore {
		t.Fatalf("open circuit must not hit upstream Soliq endpoint")
	}

	// 3. Wait for OpenDuration (50ms) to elapse -> half-open
	time.Sleep(60 * time.Millisecond)

	// 4. Recover upstream
	failMode.Store(false)

	// 5. Probe request succeeds -> breaker resets to StateClosed
	resp, err = c.Submit(ctx, []byte(`{"test":true}`), "idem-key")
	if err != nil {
		t.Fatalf("expected success on recovery, got: %v", err)
	}
	if !resp.Success || resp.EhfID != "EHF-RECOVERED-123" {
		t.Fatalf("expected recovery success, got %+v", resp)
	}
	if breaker.State() != circuit.StateClosed {
		t.Fatalf("expected StateClosed, got %v", breaker.State())
	}
}
