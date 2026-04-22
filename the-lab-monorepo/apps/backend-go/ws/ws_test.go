package ws

import (
	"net/http"
	"testing"
)

// ─── CheckWSOrigin ──────────────────────────────────────────────────────────

func TestCheckWSOrigin_EmptyOrigin_Allowed(t *testing.T) {
	r, _ := http.NewRequest("GET", "/ws", nil)
	if !CheckWSOrigin(r) {
		t.Error("empty origin should be allowed (native mobile clients)")
	}
}

func TestCheckWSOrigin_LocalhostAllowed(t *testing.T) {
	origins := []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:3002",
		"http://localhost:8081",
		"http://localhost:19006",
	}
	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/ws", nil)
			r.Header.Set("Origin", origin)
			if !CheckWSOrigin(r) {
				t.Errorf("origin %q should be allowed", origin)
			}
		})
	}
}

func TestCheckWSOrigin_LANAllowed(t *testing.T) {
	origins := []string{
		"http://192.168.0.101:3000",
		"http://192.168.0.101:3001",
		"http://192.168.0.101:19006",
		"http://192.168.1.50:3000",
		"http://10.0.2.2:8080",
	}
	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/ws", nil)
			r.Header.Set("Origin", origin)
			if !CheckWSOrigin(r) {
				t.Errorf("LAN origin %q should be allowed", origin)
			}
		})
	}
}

func TestCheckWSOrigin_NgrokAllowed(t *testing.T) {
	r, _ := http.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://abc123.ngrok-free.app")
	if !CheckWSOrigin(r) {
		t.Error("ngrok origin should be allowed")
	}
}

func TestCheckWSOrigin_ExpoAllowed(t *testing.T) {
	r, _ := http.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://u.expo.dev")
	if !CheckWSOrigin(r) {
		t.Error("expo origin should be allowed")
	}
}

func TestCheckWSOrigin_UnknownBlocked(t *testing.T) {
	blocked := []string{
		"https://evil.com",
		"http://attacker.local:3000",
		"http://localhost:9999",
		"https://phishing.ngrok.io",
	}
	for _, origin := range blocked {
		t.Run(origin, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/ws", nil)
			r.Header.Set("Origin", origin)
			if CheckWSOrigin(r) {
				t.Errorf("origin %q should be blocked", origin)
			}
		})
	}
}

// ─── FleetHub ───────────────────────────────────────────────────────────────

func TestNewFleetHub_NotNil(t *testing.T) {
	hub := NewFleetHub()
	if hub == nil {
		t.Fatal("NewFleetHub returned nil")
	}
	if hub.clients == nil {
		t.Fatal("FleetHub.clients map not initialized")
	}
}

func TestNewFleetHub_EmptyClients(t *testing.T) {
	hub := NewFleetHub()
	if len(hub.clients) != 0 {
		t.Errorf("clients = %d, want 0", len(hub.clients))
	}
}

// ─── RetailerHub ────────────────────────────────────────────────────────────

func TestNewRetailerHub_NotNil(t *testing.T) {
	hub := NewRetailerHub()
	if hub == nil {
		t.Fatal("NewRetailerHub returned nil")
	}
	if hub.clients == nil {
		t.Fatal("RetailerHub.clients map not initialized")
	}
}

func TestRetailerHub_IsConnected_Empty(t *testing.T) {
	hub := NewRetailerHub()
	if hub.IsConnected("non-existent") {
		t.Error("empty hub should report IsConnected=false")
	}
}

// ─── Struct Serialization ───────────────────────────────────────────────────

func TestLocationUpdate_Fields(t *testing.T) {
	u := LocationUpdate{
		DriverID:  "drv-1",
		Latitude:  41.311,
		Longitude: 69.279,
	}
	if u.DriverID != "drv-1" || u.Latitude != 41.311 || u.Longitude != 69.279 {
		t.Errorf("unexpected: %+v", u)
	}
}

func TestApproachPayload_Fields(t *testing.T) {
	p := ApproachPayload{
		Type:            "DRIVER_APPROACHING",
		OrderID:         "ord-1",
		SupplierID:      "sup-1",
		SupplierName:    "Nestle",
		RetailerID:      "ret-1",
		DeliveryToken:   "tok-abc",
		DriverLatitude:  41.31,
		DriverLongitude: 69.27,
	}
	if p.Type != "DRIVER_APPROACHING" || p.DeliveryToken != "tok-abc" {
		t.Errorf("unexpected: %+v", p)
	}
}
