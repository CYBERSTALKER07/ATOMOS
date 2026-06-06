package ws

import (
	"net/http/httptest"
	"testing"
)

func TestWebsocketOriginAllowed_AllowsEmptyOriginForNativeClients(t *testing.T) {
	original := allowedOrigins
	allowedOriginMu.Lock()
	allowedOrigins = map[string]struct{}{}
	allowedOriginMu.Unlock()
	t.Cleanup(func() {
		allowedOriginMu.Lock()
		allowedOrigins = original
		allowedOriginMu.Unlock()
	})

	req := httptest.NewRequest("GET", "http://example.test/v1/ws", nil)
	if !websocketOriginAllowed(req) {
		t.Fatal("expected empty origin to be allowed")
	}
}

func TestWebsocketOriginAllowed_AllowsConfiguredOrigin(t *testing.T) {
	original := allowedOrigins
	SetAllowedOrigins([]string{"https://retailer.example.com"})
	t.Cleanup(func() {
		allowedOriginMu.Lock()
		allowedOrigins = original
		allowedOriginMu.Unlock()
	})

	req := httptest.NewRequest("GET", "http://example.test/v1/ws", nil)
	req.Header.Set("Origin", "https://retailer.example.com")
	if !websocketOriginAllowed(req) {
		t.Fatal("expected configured origin to be allowed")
	}
}

func TestWebsocketOriginAllowed_AllowsLocalDevelopmentOrigins(t *testing.T) {
	original := allowedOrigins
	SetAllowedOrigins(nil)
	t.Cleanup(func() {
		allowedOriginMu.Lock()
		allowedOrigins = original
		allowedOriginMu.Unlock()
	})

	req := httptest.NewRequest("GET", "http://example.test/v1/ws", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	if !websocketOriginAllowed(req) {
		t.Fatal("expected local development origin to be allowed")
	}
}

func TestWebsocketOriginAllowed_RejectsUnexpectedBrowserOrigin(t *testing.T) {
	original := allowedOrigins
	SetAllowedOrigins([]string{"https://retailer.example.com"})
	t.Cleanup(func() {
		allowedOriginMu.Lock()
		allowedOrigins = original
		allowedOriginMu.Unlock()
	})

	req := httptest.NewRequest("GET", "http://example.test/v1/ws", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	if websocketOriginAllowed(req) {
		t.Fatal("expected unexpected browser origin to be rejected")
	}
}
