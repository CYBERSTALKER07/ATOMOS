package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Extended E2E tests — Require live backend + infrastructure
//
// Run: cd apps/backend-go && go test ./tests/e2e/... -v -count=1
//
// Prerequisites:
//   - docker-compose up (Spanner emulator, Redis, Kafka)
//   - Backend running on :8080 with seed data
// ═══════════════════════════════════════════════════════════════════════════════

func skipIfNoBackend(t *testing.T) {
	t.Helper()
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Skip("Backend not running at " + baseURL)
	}
}

// ─── Payloader Flow ─────────────────────────────────────────────────────────

func TestPayloaderFlow(t *testing.T) {
	skipIfNoBackend(t)

	// 1. Login as payloader
	payload := `{"phone": "+998901234000", "pin": "123456"}`
	resp, err := http.Post(baseURL+"/v1/auth/payloader/login", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Payloader login failed: %v", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Payloader login status %d: %s", resp.StatusCode, body)
	}
	var loginRes map[string]string
	json.NewDecoder(resp.Body).Decode(&loginRes)
	token := loginRes["token"]
	if token == "" {
		t.Fatal("empty payloader token")
	}

	// 2. List trucks
	req, _ := http.NewRequest("GET", baseURL+"/v1/payloader/trucks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Payloader trucks: %v", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Payloader trucks status %d: %s", resp.StatusCode, body)
	}
	t.Logf("Payloader trucks: %d", resp.StatusCode)

	// 3. List orders
	req, _ = http.NewRequest("GET", baseURL+"/v1/payloader/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Payloader orders: %v", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Payloader orders status %d: %s", resp.StatusCode, body)
	}
	t.Logf("Payloader orders: %d", resp.StatusCode)
}

// ─── Cross-Role Auth Denial ─────────────────────────────────────────────────

func TestCrossRoleAuthDenial(t *testing.T) {
	skipIfNoBackend(t)

	// Retailer shouldn't access driver endpoints
	retailerToken := loginRetailer(t)
	tests := []struct {
		name     string
		method   string
		path     string
		token    string
		wantCode int
	}{
		{"retailer→driver_earnings", "GET", "/v1/driver/earnings", retailerToken, 403},
		{"retailer→supplier_dashboard", "GET", "/v1/supplier/dashboard", retailerToken, 403},
		{"retailer→payloader_trucks", "GET", "/v1/payloader/trucks", retailerToken, 403},
		{"retailer→admin_nuke", "POST", "/v1/admin/nuke", retailerToken, 403},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, baseURL+tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != tc.wantCode {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantCode)
			}
		})
	}
}

// ─── Cash Webhook Settlement (Live) ────────────────────────────────────────

func TestCashWebhookSettlement(t *testing.T) {
	skipIfNoBackend(t)

	// Create an order first so we have an invoice to settle
	retailerToken := loginRetailer(t)
	orderID := createB2BOrder(t, retailerToken)

	// Send a Cash webhook with action=0 (prepare)
	body := fmt.Sprintf(`{
		"cash_trans_id": "CT-E2E-%d",
		"service_id": "SVC-1",
		"merchant_trans_id": "%s",
		"amount": 250000,
		"action": 0,
		"sign_time": "2026-04-12 10:00:00",
		"sign_string": "will-be-wrong-but-we-test-the-flow",
		"error": 0
	}`, time.Now().Unix(), orderID)

	resp, err := http.Post(baseURL+"/v1/webhooks/cash", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Cash webhook request failed: %v", err)
	}
	// We expect an error response since the signature is wrong, but the endpoint should be reachable
	if resp.StatusCode == 404 || resp.StatusCode == 405 {
		t.Errorf("Cash webhook endpoint not found or wrong method: %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	t.Logf("Cash webhook response: %v", result)
}

// ─── GlobalPay Webhook Settlement (Live) ────────────────────────────────────────

func TestGlobalPayWebhookSettlement(t *testing.T) {
	skipIfNoBackend(t)

	retailerToken := loginRetailer(t)
	orderID := createB2BOrder(t, retailerToken)

	// JSON-RPC CheckPerformTransaction
	body := fmt.Sprintf(`{
		"method": "CheckPerformTransaction",
		"params": {
			"account": {"order_id": "%s"},
			"amount": 250000
		},
		"id": 1
	}`, orderID)

	resp, err := http.Post(baseURL+"/v1/webhooks/global_pay", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("GlobalPay webhook request failed: %v", err)
	}
	// Without correct Basic Auth, we expect an auth error, but endpoint should be reachable
	if resp.StatusCode == 404 || resp.StatusCode == 405 {
		t.Errorf("GlobalPay webhook endpoint not found or wrong method: %d", resp.StatusCode)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	t.Logf("GlobalPay webhook response: %v", result)
}

// ─── WebSocket Telemetry ────────────────────────────────────────────────────

func TestWebSocketTelemetry(t *testing.T) {
	skipIfNoBackend(t)

	driverToken := loginDriver(t)

	// Connect to telemetry WS as driver
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/ws/telemetry?token=" + driverToken
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial failed: %v", err)
	}
	defer conn.Close()

	// Send a GPS payload
	gps := map[string]interface{}{
		"driver_id": "DRV-001",
		"latitude":  41.2995,
		"longitude": 69.2401,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if err := conn.WriteJSON(gps); err != nil {
		t.Fatalf("WS write failed: %v", err)
	}

	t.Log("GPS payload sent via WebSocket successfully")
}

// ─── Retailer WS Approach Push ──────────────────────────────────────────────

func TestRetailerWSConnection(t *testing.T) {
	skipIfNoBackend(t)

	retailerToken := loginRetailer(t)

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/v1/ws/retailer?token=" + retailerToken
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Retailer WS dial failed: %v", err)
	}
	defer conn.Close()

	// Connection established — the retailer is now listening for push events
	// In production, the proximity engine would push DRIVER_APPROACHING here
	t.Log("Retailer WebSocket connected successfully")
}

// ─── Driver WS Connection ───────────────────────────────────────────────────

func TestDriverWSConnection(t *testing.T) {
	skipIfNoBackend(t)

	driverToken := loginDriver(t)

	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/v1/ws/driver?token=" + driverToken
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Driver WS dial failed: %v", err)
	}
	defer conn.Close()

	t.Log("Driver WebSocket connected successfully")
}

// ─── Cash Payment Lifecycle ─────────────────────────────────────────────────

func TestCashPaymentLifecycle(t *testing.T) {
	skipIfNoBackend(t)

	// 1. Retailer creates a cash order
	retailerToken := loginRetailer(t)
	payload := `{
		"retailer_id": "RET-001",
		"payment_gateway": "CASH",
		"latitude": 41.2995,
		"longitude": 69.2401,
		"items": [
			{ "sku_id": "COKE-500-50", "quantity": 1, "unit_price_uzs": 50000 }
		]
	}`
	req, _ := http.NewRequest("POST", baseURL+"/v1/checkout/b2b", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Cash checkout failed: %d %s", resp.StatusCode, body)
	}
	var orderRes map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderRes)
	orderID := orderRes["order_id"].(string)
	t.Logf("Cash order created: %s", orderID)

	// 2. Supplier dispatches
	supplierToken := loginSupplier(t)
	dispatchPayload := map[string]interface{}{
		"order_ids": []string{orderID},
		"route_id":  "TRUCK-TASH-01",
	}
	b, _ := json.Marshal(dispatchPayload)
	req, _ = http.NewRequest("POST", baseURL+"/v1/fleet/dispatch", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+supplierToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	t.Logf("Cash order dispatched: status %d", resp.StatusCode)
}

// ─── Notification Inbox Access ──────────────────────────────────────────────

func TestNotificationInbox_AllRoles(t *testing.T) {
	skipIfNoBackend(t)

	tokens := map[string]func(*testing.T) string{
		"RETAILER": loginRetailer,
		"SUPPLIER": loginSupplier,
		"DRIVER":   loginDriver,
	}

	for role, loginFn := range tokens {
		t.Run(role, func(t *testing.T) {
			token := loginFn(t)
			req, _ := http.NewRequest("GET", baseURL+"/v1/user/notifications", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("notification inbox request: %v", err)
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("%s notification inbox: %d %s", role, resp.StatusCode, body)
			}
		})
	}
}
