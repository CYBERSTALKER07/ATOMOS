package driver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type driverOwnershipRepo struct {
	driver  Driver
	vehicle Vehicle
}

func (o *driverOwnershipRepo) Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (o *driverOwnershipRepo) CreateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (o *driverOwnershipRepo) GetDriver(ctx context.Context, driverID string) (Driver, error) {
	return o.driver, nil
}
func (o *driverOwnershipRepo) UpdateDriver(ctx context.Context, d Driver, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (o *driverOwnershipRepo) ListDrivers(ctx context.Context, supplierID string, limit, offset int) ([]Driver, error) {
	return nil, nil
}
func (o *driverOwnershipRepo) CreateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (o *driverOwnershipRepo) GetVehicle(ctx context.Context, vehicleID string) (Vehicle, error) {
	return o.vehicle, nil
}
func (o *driverOwnershipRepo) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (o *driverOwnershipRepo) ListVehicles(ctx context.Context, supplierID string, limit, offset int) ([]Vehicle, error) {
	return nil, nil
}
func (o *driverOwnershipRepo) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}

func TestHandleGetDriver_CrossTenant_404(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-a",
		Repo:       &driverOwnershipRepo{driver: Driver{DriverID: "drv-1", SupplierID: "sup-b"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/drivers/drv-1", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-a"}))
	req = req.WithContext(auth.WithTenant(req.Context(), auth.TenantContext{SupplierID: "sup-a", Source: "jwt"}))
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/drivers/{driverId}", svc.HandleGetDriver)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 cross-tenant, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "driver_not_found") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandleGetDriver_SameTenant_OK(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-a",
		Repo:       &driverOwnershipRepo{driver: Driver{DriverID: "drv-1", SupplierID: "sup-a"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/drivers/drv-1", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-a"}))
	req = req.WithContext(auth.WithTenant(req.Context(), auth.TenantContext{SupplierID: "sup-a", Source: "jwt"}))
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/drivers/{driverId}", svc.HandleGetDriver)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetDriver_DriverSelf_OK(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-a",
		Repo:       &driverOwnershipRepo{driver: Driver{DriverID: "drv-1", SupplierID: "sup-a"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/drivers/drv-1", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "drv-1", Role: auth.RoleDriver, SupplierID: "sup-a"}))
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/drivers/{driverId}", svc.HandleGetDriver)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 self, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetDriver_DriverOther_404(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-a",
		Repo:       &driverOwnershipRepo{driver: Driver{DriverID: "drv-2", SupplierID: "sup-a"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/drivers/drv-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "drv-1", Role: auth.RoleDriver, SupplierID: "sup-a"}))
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/drivers/{driverId}", svc.HandleGetDriver)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 other driver, got %d", rr.Code)
	}
}

func TestHandleGetVehicle_CrossTenant_404(t *testing.T) {
	svc := NewService(ServiceConfig{
		SupplierID: "sup-a",
		Repo:       &driverOwnershipRepo{vehicle: Vehicle{VehicleID: "veh-1", SupplierID: "sup-b"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/vehicles/veh-1", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-a"}))
	req = req.WithContext(auth.WithTenant(req.Context(), auth.TenantContext{SupplierID: "sup-a", Source: "jwt"}))
	rr := httptest.NewRecorder()
	mux := chi.NewRouter()
	mux.Get("/v1/vehicles/{vehicleId}", svc.HandleGetVehicle)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}
