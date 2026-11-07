package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/packages/handoff"
)

func TestHandoffService_Healthz(t *testing.T) {
	engine := handoff.New(handoff.Config{})
	handler := buildRouter(engine, "test-api-key")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode healthz response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp["status"])
	}
}

func TestHandoffService_InternalAuth(t *testing.T) {
	engine := handoff.New(handoff.Config{})
	handler := buildRouter(engine, "secret-key")

	req := httptest.NewRequest(http.MethodPost, "/internal/v1/handoff/validate", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without header, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/internal/v1/handoff/validate", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-Internal-Api-Key", "wrong-key")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized with wrong key, got %d", rr.Code)
	}
}

func TestHandoffService_Validate_Success(t *testing.T) {
	engine := handoff.New(handoff.Config{})
	handler := buildRouter(engine, "secret-key")

	orderID := "ord-123"
	token := engine.Mint()

	body, _ := json.Marshal(validateRequest{
		OrderID:        orderID,
		StoredToken:    token,
		PresentedToken: token,
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/v1/handoff/validate", bytes.NewReader(body))
	req.Header.Set("X-Internal-Api-Key", "secret-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandoffService_GracefulShutdown(t *testing.T) {
	engine := handoff.New(handoff.Config{})
	handler := buildRouter(engine, "secret-key")

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: handler,
	}

	serverStopped := make(chan struct{})
	go func() {
		_ = srv.ListenAndServe()
		close(serverStopped)
	}()

	// Wait briefly for server to bind
	time.Sleep(50 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("srv.Shutdown failed: %v", err)
	}

	select {
	case <-serverStopped:
		// Srv stopped successfully
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time after shutdown")
	}
}

func TestHandoffFallbackForbidden(t *testing.T) {
	orig := os.Getenv("PEGASUSX_ENV")
	defer os.Setenv("PEGASUSX_ENV", orig)

	os.Setenv("PEGASUSX_ENV", "production")
	if !handoffFallbackForbidden() {
		t.Fatal("expected true for production")
	}

	os.Setenv("PEGASUSX_ENV", "staging")
	if !handoffFallbackForbidden() {
		t.Fatal("expected true for staging")
	}

	os.Setenv("PEGASUSX_ENV", "sandbox")
	if handoffFallbackForbidden() {
		t.Fatal("expected false for sandbox")
	}

	_ = syscall.SIGTERM // compile check
}
