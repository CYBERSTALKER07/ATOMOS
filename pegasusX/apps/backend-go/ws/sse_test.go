package ws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// fakeFlusherResponseWriter wraps httptest.ResponseRecorder with http.Flusher support.
type fakeFlusherResponseWriter struct {
	*httptest.ResponseRecorder
	flushed bool
	mu      sync.Mutex
}

func (f *fakeFlusherResponseWriter) Flush() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flushed = true
}

func TestSSEConn_ImplementsConnectionAndReapable(t *testing.T) {
	rec := httptest.NewRecorder()
	flusher := &fakeFlusherResponseWriter{ResponseRecorder: rec}
	claims := auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-test-1", Subject: "admin-1"}

	conn := NewSSEConn(claims, flusher, flusher)
	if conn.ID() == "" {
		t.Fatal("expected non-empty connection ID")
	}
	if conn.Identity().SupplierID != "sup-test-1" {
		t.Fatalf("identity mismatch: got %v", conn.Identity())
	}

	// Test Ping
	if err := conn.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ": ping\n\n") {
		t.Fatalf("expected ping in body, got %q", body)
	}
	if !flusher.flushed {
		t.Fatal("expected flusher.Flush to be called on ping")
	}

	// Test Send with JSON payload containing type and id
	flusher.flushed = false
	payload := []byte(`{"type":"ORDER_STATUS_CHANGED","id":"evt-100","status":"DISPATCHED"}`)
	if err := conn.Send(context.Background(), payload); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	body = rec.Body.String()
	if !strings.Contains(body, "id: evt-100\n") {
		t.Fatalf("expected event id, got %q", body)
	}
	if !strings.Contains(body, "event: ORDER_STATUS_CHANGED\n") {
		t.Fatalf("expected event type, got %q", body)
	}
	if !strings.Contains(body, `data: {"type":"ORDER_STATUS_CHANGED","id":"evt-100","status":"DISPATCHED"}`+"\n\n") {
		t.Fatalf("expected data line, got %q", body)
	}

	// Test Reap / Close
	conn.Reap()
	select {
	case <-conn.Done():
		// Closed successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected connection to be closed after Reap")
	}

	// Sending after close should return an error
	if err := conn.Send(context.Background(), payload); err == nil {
		t.Fatal("expected send after close to return error")
	}
	if err := conn.Ping(); err == nil {
		t.Fatal("expected ping after close to return error")
	}
}

func TestSSEConn_MultilinePayloadAndGeneratedID(t *testing.T) {
	rec := httptest.NewRecorder()
	flusher := &fakeFlusherResponseWriter{ResponseRecorder: rec}
	claims := auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-test-2"}

	conn := NewSSEConn(claims, flusher, flusher)

	// Send raw payload without explicit event_id or type
	payload := []byte("line1\nline2")
	if err := conn.Send(context.Background(), payload); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "id: ") {
		t.Fatalf("expected generated id in body, got %q", body)
	}
	if !strings.Contains(body, "data: line1\n") || !strings.Contains(body, "data: line2\n") {
		t.Fatalf("expected multiline data, got %q", body)
	}
}

func TestHandleSSEEvents_HeadersHandshakeAndEventDelivery(t *testing.T) {
	supplierHub := NewHub("supplier", nil, nil)
	telemetryHub := NewHub("telemetry", nil, nil)
	hubs := roleHubs{supplier: supplierHub, telemetry: telemetryHub}

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-99",
	}))
	router.Get("/v1/events", HandleSSEEvents(slog.Default(), testWSJWTSecret, nil, hubs, RegisterConfig{}))

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/v1/events", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http GET /v1/events failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != SSEContentType {
		t.Fatalf("expected Content-Type %q, got %q", SSEContentType, ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != SSECacheControl {
		t.Fatalf("expected Cache-Control %q, got %q", SSECacheControl, cc)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("expected X-Accel-Buffering 'no', got %q", xab)
	}

	reader := bufio.NewReader(resp.Body)

	// Verify handshake lines: "retry: 3000" and ": connected"
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read line 1: %v", err)
	}
	if strings.TrimSpace(line1) != fmt.Sprintf("retry: %d", SSERetryTimeoutMS) {
		t.Fatalf("expected retry directive, got %q", line1)
	}

	// Empty line separating retry frame
	_, _ = reader.ReadString('\n')

	line2, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read line 2: %v", err)
	}
	if strings.TrimSpace(line2) != ": connected" {
		t.Fatalf("expected : connected comment, got %q", line2)
	}

	// Broadcast an event to supplier room
	eventPayload := map[string]any{
		"type":        "ORDER_STATUS_CHANGED",
		"event_id":    "evt-9901",
		"order_id":    "ord-12345",
		"status":      "DISPATCHED",
		"supplier_id": "sup-99",
	}
	payloadBytes, _ := json.Marshal(eventPayload)

	go func() {
		time.Sleep(50 * time.Millisecond)
		supplierHub.Broadcast(context.Background(), "supplier:sup-99", payloadBytes)
	}()

	// Read event frame
	var received []string
	deadline := time.After(2 * time.Second)
	done := make(chan struct{})

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				received = append(received, trimmed)
			}
			if len(received) >= 3 {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		// Check that id, event, and data lines were received
		idFound, eventFound, dataFound := false, false, false
		for _, line := range received {
			if strings.HasPrefix(line, "id: evt-9901") {
				idFound = true
			}
			if strings.HasPrefix(line, "event: ORDER_STATUS_CHANGED") {
				eventFound = true
			}
			if strings.HasPrefix(line, "data:") && strings.Contains(line, "ord-12345") {
				dataFound = true
			}
		}
		if !idFound || !eventFound || !dataFound {
			t.Fatalf("incomplete event frame received: %v", received)
		}
	case <-deadline:
		t.Fatalf("timeout waiting for event frame, received: %v", received)
	}
}

func TestHandleSupplierEvents_SupplierScopedStreaming(t *testing.T) {
	supplierHub := NewHub("supplier", nil, nil)
	telemetryHub := NewHub("telemetry", nil, nil)

	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-2",
		SupplierID: "sup-alpha",
	}))
	router.Get("/v1/supplier/events", HandleSupplierEvents(slog.Default(), testWSJWTSecret, supplierHub, nil, telemetryHub))

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/v1/supplier/events", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http GET /v1/supplier/events failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)

	// Verify handshake
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read line 1: %v", err)
	}
	if !strings.Contains(line1, "retry: 3000") {
		t.Fatalf("expected retry: 3000, got %q", line1)
	}

	// Broadcast a telemetry event
	telemetryPayload := []byte(`{"type":"DRIVER_LOCATION_UPDATED","lat":41.311,"lng":69.240}`)
	go func() {
		time.Sleep(50 * time.Millisecond)
		telemetryHub.Broadcast(context.Background(), "telemetry:supplier:sup-alpha", telemetryPayload)
	}()

	var received []string
	deadline := time.After(2 * time.Second)
	done := make(chan struct{})

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				received = append(received, trimmed)
			}
			if len(received) >= 4 { // : connected + (id, event, data)
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		dataFound := false
		for _, line := range received {
			if strings.Contains(line, "DRIVER_LOCATION_UPDATED") {
				dataFound = true
			}
		}
		if !dataFound {
			t.Fatalf("telemetry event not received: %v", received)
		}
	case <-deadline:
		t.Fatalf("timeout waiting for telemetry event, received: %v", received)
	}
}

func TestHandleSupplierEvents_RejectsNonSupplierRole(t *testing.T) {
	router := chi.NewRouter()
	router.Use(testClaimsMiddleware(auth.Claims{
		Role:    auth.RoleDriver,
		Subject: "drv-99",
	}))
	router.Get("/v1/supplier/events", HandleSupplierEvents(slog.Default(), testWSJWTSecret, nil, nil, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/supplier/events")
	if err != nil {
		t.Fatalf("GET /v1/supplier/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 Forbidden for driver role", resp.StatusCode)
	}
}

func TestHandleSSEEvents_AcceptsBearerToken(t *testing.T) {
	supplierHub := NewHub("supplier", nil, nil)
	hubs := roleHubs{supplier: supplierHub}

	router := chi.NewRouter()
	router.Get("/v1/events", HandleSSEEvents(slog.Default(), testWSJWTSecret, nil, hubs, RegisterConfig{}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Mint a valid bearer token
	token, err := auth.Issue(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-bearer",
		SupplierID: "sup-bearer-1",
	}, auth.IssueOptions{
		Secret: testWSJWTSecret,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read first line: %v", err)
	}
	if !strings.Contains(line, "retry: 3000") {
		t.Fatalf("expected retry: 3000, got %q", line)
	}
}

func TestHandleSSEEvents_AcceptsQueryToken(t *testing.T) {
	supplierHub := NewHub("supplier", nil, nil)
	hubs := roleHubs{supplier: supplierHub}

	router := chi.NewRouter()
	router.Get("/v1/events", HandleSSEEvents(slog.Default(), testWSJWTSecret, nil, hubs, RegisterConfig{}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Mint a valid token
	token, err := auth.Issue(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-query",
		SupplierID: "sup-query-1",
	}, auth.IssueOptions{
		Secret: testWSJWTSecret,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url := fmt.Sprintf("%s/v1/events?token=%s", server.URL, token)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/events?token=...: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandleSSEEvents_RejectsUnauthenticated(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/v1/events", HandleSSEEvents(slog.Default(), testWSJWTSecret, nil, roleHubs{}, RegisterConfig{}))

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/events")
	if err != nil {
		t.Fatalf("GET /v1/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 Unauthorized", resp.StatusCode)
	}
}

func TestSSEConn_KeepAlivePing(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	flusher := &pipeFlusherWriter{PipeWriter: pipeWriter}
	claims := auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-ping"}

	conn := NewSSEConn(claims, flusher, flusher)

	go func() {
		// Emit ping
		_ = conn.Ping()
		_ = pipeWriter.Close()
	}()

	reader := bufio.NewReader(pipeReader)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read ping: %v", err)
	}
	if strings.TrimSpace(line) != ": ping" {
		t.Fatalf("expected ': ping', got %q", line)
	}
}

type pipeFlusherWriter struct {
	*io.PipeWriter
}

func (p *pipeFlusherWriter) Flush() {}
func (p *pipeFlusherWriter) Header() http.Header {
	return make(http.Header)
}
func (p *pipeFlusherWriter) WriteHeader(statusCode int) {}

func TestAcceptanceCriteria2_CurlStreaming(t *testing.T) {
	curlPath, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("curl not installed on test host")
	}

	supplierHub := NewHub("supplier", nil, nil)
	router := chi.NewRouter()
	router.Get("/v1/supplier/events", HandleSupplierEvents(slog.Default(), testWSJWTSecret, supplierHub, nil, nil))

	server := httptest.NewServer(router)
	defer server.Close()

	// Mint token for supplier admin
	token, err := auth.Issue(auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "supplier-admin-curl",
		SupplierID: "sup-curl-99",
	}, auth.IssueOptions{
		Secret: testWSJWTSecret,
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute: curl -N -s -H "Accept: text/event-stream" -H "Authorization: Bearer <token>" <url>
	cmd := exec.CommandContext(ctx, curlPath,
		"-N", "-s",
		"-H", "Accept: text/event-stream",
		"-H", "Authorization: Bearer "+token,
		server.URL+"/v1/supplier/events",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start curl: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	reader := bufio.NewReader(stdout)

	// 1. Read initial handshake line 1: "retry: 3000"
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("curl read line 1: %v", err)
	}
	if !strings.Contains(line1, "retry: 3000") {
		t.Fatalf("expected retry: 3000 from curl, got %q", line1)
	}

	// Read empty line
	_, _ = reader.ReadString('\n')

	// 2. Read initial handshake line 2: ": connected"
	line2, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("curl read line 2: %v", err)
	}
	if !strings.Contains(line2, ": connected") {
		t.Fatalf("expected : connected from curl, got %q", line2)
	}

	// 3. Broadcast an event to supplier room
	eventPayload := map[string]any{
		"type":        "ORDER_STATUS_CHANGED",
		"event_id":    "evt-curl-123",
		"order_id":    "ord-curl-456",
		"status":      "DELIVERED",
		"supplier_id": "sup-curl-99",
	}
	payloadBytes, _ := json.Marshal(eventPayload)

	go func() {
		time.Sleep(50 * time.Millisecond)
		supplierHub.Broadcast(context.Background(), "supplier:sup-curl-99", payloadBytes)
	}()

	// 4. Read streamed event from curl output
	var received []string
	done := make(chan struct{})
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				received = append(received, trimmed)
			}
			if len(received) >= 3 {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		idFound, eventFound, dataFound := false, false, false
		for _, l := range received {
			if strings.Contains(l, "id: evt-curl-123") {
				idFound = true
			}
			if strings.Contains(l, "event: ORDER_STATUS_CHANGED") {
				eventFound = true
			}
			if strings.Contains(l, "data:") && strings.Contains(l, "ord-curl-456") {
				dataFound = true
			}
		}
		if !idFound || !eventFound || !dataFound {
			t.Fatalf("curl did not receive full event frame: %v", received)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("curl stream timed out waiting for broadcast event, got: %v", received)
	}
}

