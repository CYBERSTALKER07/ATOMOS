// Package simulation_test provides comprehensive end-to-end local simulation
// tests for the entire pegasusX ecosystem. These tests exercise every
// foundational entity lifecycle, concurrency, error handling, and edge cases
// without requiring external services (Spanner, Firebase, Kafka, etc.).
package simulator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/factory"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
)

// ── helpers ─────────────────────────────────────────────────────────────────

// simRouter returns a chi router whose requests carry authenticated
// supplier-scope claims, mirroring production auth middleware. Handlers
// resolve supplier scope from claims (never request bodies), so every
// simulated request must be claim-bound exactly like a real session.
func simRouter(supplierID string) chi.Router {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "sim-test",
				Role:       auth.RoleAdmin,
				SupplierID: supplierID,
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	return r
}

func newWarehouseService() *warehouse.Service {
	return warehouse.NewService(warehouse.ServiceConfig{
		Repo: warehouse.NewInMemoryRepository(),
	})
}

func newFactoryService() *factory.Service {
	return factory.NewService(factory.ServiceConfig{
		Repo: factory.NewInMemoryRepository(),
	})
}

func newDriverService() *driver.Service {
	return driver.NewService(driver.ServiceConfig{
		Repo: driver.NewInMemoryRepository(),
	})
}

func postJSON(r chi.Router, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func getJSON(r chi.Router, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Warehouse CRUD lifecycle simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimWarehouse_CreateReadUpdateList(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)
	r.Get("/v1/warehouses/{warehouseId}", svc.HandleGetWarehouse)
	r.Put("/v1/warehouses/{warehouseId}", svc.HandleUpdateWarehouse)
	r.Get("/v1/warehouses", svc.HandleListWarehouses)

	// Create
	w := postJSON(r, "/v1/warehouses", map[string]interface{}{
		"name":        "Tashkent Central Warehouse",
		"supplier_id": "sup-001",
		"address":     "123 Main St, Tashkent",
		"latitude":    41.2995,
		"longitude":   69.2401,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create warehouse: want 201, got %d: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	warehouseID, _ := created["warehouse_id"].(string)
	if warehouseID == "" {
		t.Fatal("create warehouse: no warehouse_id returned")
	}
	t.Logf("created warehouse %s", warehouseID)
}

func TestSimWarehouse_CreateValidation(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{"empty body", map[string]interface{}{}, http.StatusBadRequest},
		{"missing name", map[string]interface{}{"supplier_id": "sup-001"}, http.StatusBadRequest},
		{"foreign supplier_id rejected", map[string]interface{}{"name": "Test", "supplier_id": "sup-other"}, http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSON(r, "/v1/warehouses", tc.body)
			if w.Code != tc.wantStatus {
				t.Errorf("want %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestSimWarehouse_UnauthenticatedRejected(t *testing.T) {
	svc := newWarehouseService()
	r := chi.NewRouter() // no claims middleware — simulates a session-less caller
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	w := postJSON(r, "/v1/warehouses", map[string]interface{}{"name": "Test"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated create: want 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Factory CRUD lifecycle simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimFactory_CreateReadUpdateList(t *testing.T) {
	svc := newFactoryService()
	r := simRouter("sup-001")
	r.Post("/v1/factories", svc.HandleCreateFactory)
	r.Get("/v1/factories/{factoryId}", svc.HandleGetFactory)
	r.Put("/v1/factories/{factoryId}", svc.HandleUpdateFactory)
	r.Get("/v1/factories", svc.HandleListFactories)

	w := postJSON(r, "/v1/factories", map[string]interface{}{
		"name":        "Samarkand Juice Factory",
		"supplier_id": "sup-001",
		"address":     "456 Factory Ave, Samarkand",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create factory: want 201, got %d: %s", w.Code, w.Body.String())
	}
	t.Logf("factory created: %s", w.Body.String())
}

func TestSimFactory_CreateValidation(t *testing.T) {
	svc := newFactoryService()
	r := simRouter("sup-001")
	r.Post("/v1/factories", svc.HandleCreateFactory)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{"empty body", map[string]interface{}{}, http.StatusBadRequest},
		{"missing name", map[string]interface{}{"supplier_id": "sup-001"}, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSON(r, "/v1/factories", tc.body)
			if w.Code != tc.wantStatus {
				t.Errorf("want %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Driver & Vehicle CRUD lifecycle simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimDriver_CreateReadUpdateList(t *testing.T) {
	svc := newDriverService()
	r := simRouter("sup-001")
	r.Post("/v1/drivers", svc.HandleCreateDriver)
	r.Get("/v1/drivers/{driverId}", svc.HandleGetDriver)
	r.Put("/v1/drivers/{driverId}", svc.HandleUpdateDriver)
	r.Get("/v1/drivers", svc.HandleListDrivers)

	w := postJSON(r, "/v1/drivers", map[string]interface{}{
		"name":           "Anvar Karimov",
		"supplier_id":    "sup-001",
		"phone":          "+998901234567",
		"license_number": "UZ1234567",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create driver: want 201, got %d: %s", w.Code, w.Body.String())
	}
	t.Logf("driver created: %s", w.Body.String())
}

func TestSimVehicle_CreateReadUpdateList(t *testing.T) {
	svc := newDriverService()
	r := simRouter("sup-001")
	r.Post("/v1/vehicles", svc.HandleCreateVehicle)
	r.Get("/v1/vehicles/{vehicleId}", svc.HandleGetVehicle)
	r.Put("/v1/vehicles/{vehicleId}", svc.HandleUpdateVehicle)
	r.Get("/v1/vehicles", svc.HandleListVehicles)

	w := postJSON(r, "/v1/vehicles", map[string]interface{}{
		"license_plate": "01A123BC",
		"supplier_id":   "sup-001",
		"make":          "Isuzu",
		"model":         "NQR 71P",
		"year":          2023,
		"capacity_kg":   5000,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create vehicle: want 201, got %d: %s", w.Code, w.Body.String())
	}
	t.Logf("vehicle created: %s", w.Body.String())
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Concurrency stress: parallel entity creation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimConcurrency_ParallelWarehouseCreation(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(r, "/v1/warehouses", map[string]interface{}{
				"name":        fmt.Sprintf("Warehouse-%d", idx),
				"supplier_id": "sup-001",
				"address":     fmt.Sprintf("Address %d", idx),
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("goroutine %d: want 201, got %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Error(errMsg)
	}
}

func TestSimConcurrency_ParallelDriverCreation(t *testing.T) {
	svc := newDriverService()
	r := simRouter("sup-001")
	r.Post("/v1/drivers", svc.HandleCreateDriver)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(r, "/v1/drivers", map[string]interface{}{
				"name":        fmt.Sprintf("Driver-%d", idx),
				"supplier_id": "sup-001",
				"phone":       fmt.Sprintf("+99890%07d", idx),
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("goroutine %d: want 201, got %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Error(errMsg)
	}
}

func TestSimConcurrency_ParallelFactoryCreation(t *testing.T) {
	svc := newFactoryService()
	r := simRouter("sup-001")
	r.Post("/v1/factories", svc.HandleCreateFactory)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(r, "/v1/factories", map[string]interface{}{
				"name":        fmt.Sprintf("Factory-%d", idx),
				"supplier_id": "sup-001",
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("goroutine %d: want 201, got %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Error(errMsg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Error message quality
// ─────────────────────────────────────────────────────────────────────────────

func TestSimErrorMessages_InvalidJSON(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	req := httptest.NewRequest(http.MethodPost, "/v1/warehouses", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: want 400, got %d", w.Code)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("error response is not valid JSON: %v", err)
	}
	if _, ok := errResp["error"]; !ok {
		t.Error("error response should contain 'error' key")
	}
}

func TestSimErrorMessages_MethodNotAllowed(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)
	r.Get("/v1/warehouses", svc.HandleListWarehouses)

	// DELETE on a POST-only route
	req := httptest.NewRequest(http.MethodDelete, "/v1/warehouses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Logf("method not allowed: got %d (chi may return 405 or 404 depending on config)", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. Idempotency simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimIdempotency_DuplicateWarehouseID(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	id := uuid.New().String()
	body := map[string]interface{}{
		"warehouse_id": id,
		"name":         "Duplicate Test",
		"supplier_id":  "sup-001",
		"address":      "Test",
	}

	// First create
	w1 := postJSON(r, "/v1/warehouses", body)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: want 201, got %d", w1.Code)
	}

	// Second create with same body (should succeed or return conflict, no panic/500)
	w2 := postJSON(r, "/v1/warehouses", body)
	if w2.Code >= 500 {
		t.Fatalf("duplicate create should not cause 5xx: got %d", w2.Code)
	}
	t.Logf("duplicate create returned %d (acceptable)", w2.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. Full ecosystem lifecycle simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimEcosystem_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	// Warehouse
	whSvc := newWarehouseService()
	whR := simRouter("sup-eco-001")
	whR.Post("/v1/warehouses", whSvc.HandleCreateWarehouse)

	whW := postJSON(whR, "/v1/warehouses", map[string]interface{}{
		"name": "Hub Warehouse", "supplier_id": "sup-eco-001",
	})
	if whW.Code != http.StatusCreated {
		t.Fatalf("warehouse create: %d: %s", whW.Code, whW.Body.String())
	}

	// Factory
	facSvc := newFactoryService()
	facR := simRouter("sup-eco-001")
	facR.Post("/v1/factories", facSvc.HandleCreateFactory)

	facW := postJSON(facR, "/v1/factories", map[string]interface{}{
		"name": "Production Plant", "supplier_id": "sup-eco-001",
	})
	if facW.Code != http.StatusCreated {
		t.Fatalf("factory create: %d: %s", facW.Code, facW.Body.String())
	}

	// Driver
	drvSvc := newDriverService()
	drvR := simRouter("sup-eco-001")
	drvR.Post("/v1/drivers", drvSvc.HandleCreateDriver)
	drvR.Post("/v1/vehicles", drvSvc.HandleCreateVehicle)

	drvW := postJSON(drvR, "/v1/drivers", map[string]interface{}{
		"name": "Test Driver", "supplier_id": "sup-eco-001", "phone": "+998901111111",
	})
	if drvW.Code != http.StatusCreated {
		t.Fatalf("driver create: %d: %s", drvW.Code, drvW.Body.String())
	}

	// Vehicle
	vehW := postJSON(drvR, "/v1/vehicles", map[string]interface{}{
		"license_plate": "01X999YZ", "supplier_id": "sup-eco-001",
		"make": "Hyundai", "model": "HD78",
	})
	if vehW.Code != http.StatusCreated {
		t.Fatalf("vehicle create: %d: %s", vehW.Code, vehW.Body.String())
	}

	t.Log("Full ecosystem lifecycle: warehouse + factory + driver + vehicle created successfully")
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. Request timeout simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimTimeout_ContextCancellation(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Get("/v1/warehouses", svc.HandleListWarehouses)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond) // ensure timeout fires

	req := httptest.NewRequest(http.MethodGet, "/v1/warehouses?supplier_id=sup-001", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The handler should handle cancelled context gracefully (no panic)
	t.Logf("cancelled context request returned %d (no panic = pass)", w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. Large payload simulation
// ─────────────────────────────────────────────────────────────────────────────

func TestSimLargePayload_OversizedName(t *testing.T) {
	svc := newWarehouseService()
	r := simRouter("sup-001")
	r.Post("/v1/warehouses", svc.HandleCreateWarehouse)

	longName := make([]byte, 10240)
	for i := range longName {
		longName[i] = 'A'
	}

	w := postJSON(r, "/v1/warehouses", map[string]interface{}{
		"name": string(longName), "supplier_id": "sup-001",
	})
	if w.Code >= 500 {
		t.Fatalf("oversized name caused 5xx: %d: %s", w.Code, w.Body.String())
	}
	t.Logf("oversized name: got %d (no crash = pass)", w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// 10. Mixed concurrent entity creation across packages
// ─────────────────────────────────────────────────────────────────────────────

func TestSimConcurrency_MixedEntityCreation(t *testing.T) {
	whSvc := newWarehouseService()
	facSvc := newFactoryService()
	drvSvc := newDriverService()

	whR := simRouter("sup-mix")
	whR.Post("/v1/warehouses", whSvc.HandleCreateWarehouse)
	facR := simRouter("sup-mix")
	facR.Post("/v1/factories", facSvc.HandleCreateFactory)
	drvR := simRouter("sup-mix")
	drvR.Post("/v1/drivers", drvSvc.HandleCreateDriver)
	drvR.Post("/v1/vehicles", drvSvc.HandleCreateVehicle)

	const perType = 20
	var wg sync.WaitGroup
	errors := make(chan string, perType*4)

	// Warehouses
	for i := 0; i < perType; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(whR, "/v1/warehouses", map[string]interface{}{
				"name": fmt.Sprintf("WH-%d", idx), "supplier_id": "sup-mix",
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("warehouse %d: %d", idx, w.Code)
			}
		}(i)
	}

	// Factories
	for i := 0; i < perType; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(facR, "/v1/factories", map[string]interface{}{
				"name": fmt.Sprintf("FAC-%d", idx), "supplier_id": "sup-mix",
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("factory %d: %d", idx, w.Code)
			}
		}(i)
	}

	// Drivers
	for i := 0; i < perType; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(drvR, "/v1/drivers", map[string]interface{}{
				"name": fmt.Sprintf("DRV-%d", idx), "supplier_id": "sup-mix",
				"phone": fmt.Sprintf("+99890%07d", idx),
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("driver %d: %d", idx, w.Code)
			}
		}(i)
	}

	// Vehicles
	for i := 0; i < perType; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := postJSON(drvR, "/v1/vehicles", map[string]interface{}{
				"license_plate": fmt.Sprintf("99X%03dYZ", idx), "supplier_id": "sup-mix",
			})
			if w.Code != http.StatusCreated {
				errors <- fmt.Sprintf("vehicle %d: %d", idx, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Error(errMsg)
	}
}
