package warehouse

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleOpsDrivers_CreateDriver_DefaultPIN(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-test"})

	body := []byte(`{"name":"John Doe","phone":"+1234567890"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/drivers", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	rr := httptest.NewRecorder()

	svc.HandleOpsDrivers(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	driverID, ok := resp["driver_id"].(string)
	if !ok || !strings.HasPrefix(driverID, "drv-") {
		t.Fatalf("expected driver_id starting with drv-, got: %v", resp["driver_id"])
	}

	pin, ok := resp["pin"].(string)
	if !ok || len(pin) != 4 {
		t.Fatalf("expected 4-digit pin, got: %v", resp["pin"])
	}

	// Verify driver added in in-memory state
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	found := false
	for _, d := range svc.drivers {
		if d.DriverID == driverID {
			found = true
			if d.Name != "John Doe" || d.Phone != "+1234567890" {
				t.Fatalf("driver in state mismatch: %+v", d)
			}
			break
		}
	}
	if !found {
		t.Fatalf("driver %s not found in service state", driverID)
	}
}

func TestHandleOpsDrivers_CreateDriver_CustomPIN(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-test"})

	body := []byte(`{"name":"Jane Smith","phone":"+9876543210","pin":"9876"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/drivers", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	rr := httptest.NewRecorder()

	svc.HandleOpsDrivers(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pin, ok := resp["pin"].(string)
	if !ok || pin != "9876" {
		t.Fatalf("expected pin '9876', got: %v", resp["pin"])
	}
}

func TestHandleOpsDrivers_BcryptHashVerification(t *testing.T) {
	t.Parallel()
	pin := "4321"
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt generation failed: %v", err)
	}

	if !strings.HasPrefix(string(hash), "$2a$") {
		t.Fatalf("expected bcrypt $2a$ prefix, got %s", string(hash))
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(pin)); err != nil {
		t.Fatalf("bcrypt comparison failed: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrongpin")); err == nil {
		t.Fatal("expected bcrypt compare error on wrong pin, got nil")
	}
}

func TestHandleOpsDrivers_ListDrivers(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-test"})

	// First create a driver via POST
	body := []byte(`{"name":"Alex Rivera","phone":"+1555123456"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/drivers", bytes.NewReader(body))
	postReq = postReq.WithContext(auth.WithClaims(postReq.Context(), auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	postRr := httptest.NewRecorder()
	svc.HandleOpsDrivers(postRr, postReq)
	if postRr.Code != http.StatusCreated {
		t.Fatalf("create failed: status %d", postRr.Code)
	}

	// Now list drivers via GET
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/drivers", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	rr := httptest.NewRecorder()

	svc.HandleOpsDrivers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	drivers, ok := resp["drivers"].([]any)
	if !ok {
		t.Fatalf("expected drivers array, got: %T", resp["drivers"])
	}
	if len(drivers) == 0 {
		t.Fatal("expected at least 1 driver, got 0")
	}
}

func TestHandleOpsDrivers_InvalidJSON(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-test"})

	body := []byte(`{invalid-json}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/drivers", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-1",
	}))
	rr := httptest.NewRecorder()

	svc.HandleOpsDrivers(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
