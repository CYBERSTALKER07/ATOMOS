package driver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleWSAck_ValidationAndUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{})

	// 1. Unauthorized without driver claims
	reqUnauth := httptest.NewRequest(http.MethodPost, "/v1/ws/ack", strings.NewReader(`{"command_id":"cmd-1","event_type":"ORDER_ASSIGNED"}`))
	rrUnauth := httptest.NewRecorder()
	svc.HandleWSAck(rrUnauth, reqUnauth)
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", rrUnauth.Code)
	}

	claims := auth.Claims{
		Role:    auth.RoleDriver,
		Subject: "drv-1",
	}

	// 2. Missing event_type -> 400
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/ws/ack", strings.NewReader(`{"command_id":"cmd-1"}`))
	reqBad = reqBad.WithContext(auth.WithClaims(reqBad.Context(), claims))
	rrBad := httptest.NewRecorder()
	svc.HandleWSAck(rrBad, reqBad)
	if rrBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing event_type, got %d", rrBad.Code)
	}

	// 3. Valid payload with nil Spanner -> 503 command_ack_unavailable
	reqNoStore := httptest.NewRequest(http.MethodPost, "/v1/ws/ack", strings.NewReader(`{"command_id":"cmd-1","trace_id":"tr-1","event_type":"ORDER_ASSIGNED"}`))
	reqNoStore = reqNoStore.WithContext(auth.WithClaims(reqNoStore.Context(), claims))
	rrNoStore := httptest.NewRecorder()
	svc.HandleWSAck(rrNoStore, reqNoStore)
	if rrNoStore.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner is nil, got %d body=%s", rrNoStore.Code, rrNoStore.Body.String())
	}
}

func TestValidateIssuedCommandMatch_AndErrorMapping(t *testing.T) {
	if err := validateIssuedCommandMatch("ROUTE_REPLANNED", "ROUTE_REPLANNED"); err != nil {
		t.Fatalf("exact match failed: %v", err)
	}
	if err := validateIssuedCommandMatch("route_replanned", "ROUTE_REPLANNED"); err != nil {
		t.Fatalf("case-insensitive match failed: %v", err)
	}
	if err := validateIssuedCommandMatch("ROUTE_REPLANNED", "ORDER_CANCELLED"); err != errCommandEventMismatch {
		t.Fatalf("expected errCommandEventMismatch, got %v", err)
	}

	rrNotIssued := httptest.NewRecorder()
	writeCommandAckError(rrNotIssued, errCommandNotIssued)
	if rrNotIssued.Code != http.StatusNotFound || !strings.Contains(rrNotIssued.Body.String(), "command_not_issued") {
		t.Fatalf("expected 404 command_not_issued, got %d %s", rrNotIssued.Code, rrNotIssued.Body.String())
	}

	rrMismatch := httptest.NewRecorder()
	writeCommandAckError(rrMismatch, errCommandEventMismatch)
	if rrMismatch.Code != http.StatusConflict || !strings.Contains(rrMismatch.Body.String(), "command_event_type_mismatch") {
		t.Fatalf("expected 409 command_event_type_mismatch, got %d %s", rrMismatch.Code, rrMismatch.Body.String())
	}
}
