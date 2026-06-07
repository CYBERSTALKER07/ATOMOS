package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type dualSupplierRepo struct {
	profiles map[string]Profile
}

func (r *dualSupplierRepo) CountSuppliers(context.Context) (int64, error) {
	return int64(len(r.profiles)), nil
}
func (r *dualSupplierRepo) GetProfile(_ context.Context, id string) (Profile, bool, error) {
	p, ok := r.profiles[id]
	return p, ok, nil
}
func (r *dualSupplierRepo) UpdateProfile(_ context.Context, p Profile, _ func(outbox.TxnBuffer) error) error {
	r.profiles[p.SupplierID] = p
	return nil
}
func (r *dualSupplierRepo) GetAuthByPhone(context.Context, string) (SupplierAuthRecord, bool, error) {
	return SupplierAuthRecord{}, false, nil
}
func (r *dualSupplierRepo) GetTopology(_ context.Context, id string) (SupplierTopology, error) {
	return SupplierTopology{
		Warehouses: []WarehouseNode{{WarehouseID: "wh-" + id, Name: "WH " + id}},
	}, nil
}
func (r *dualSupplierRepo) ReplaceTopology(context.Context, string, SupplierTopology, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *dualSupplierRepo) ListOrgMembers(context.Context, string) ([]SupplierOrgMember, error) {
	return nil, nil
}
func (r *dualSupplierRepo) CreateOrgMember(context.Context, CreateOrgMemberParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *dualSupplierRepo) ListFleetDrivers(context.Context, string) ([]SupplierFleetDriver, error) {
	return nil, nil
}
func (r *dualSupplierRepo) CreateFleetDriver(context.Context, CreateFleetDriverParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *dualSupplierRepo) ListFleetVehicles(context.Context, string) ([]SupplierFleetVehicle, error) {
	return nil, nil
}
func (r *dualSupplierRepo) CreateFleetVehicle(context.Context, CreateFleetVehicleParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *dualSupplierRepo) GetPricingRule(context.Context, string) (SupplierPricingRule, bool, error) {
	return SupplierPricingRule{}, false, nil
}
func (r *dualSupplierRepo) UpsertPricingRule(context.Context, SupplierPricingRule, func(outbox.TxnBuffer) error) error {
	return nil
}

func TestHandleTopologyGet_IsolatesByJWTSupplierID(t *testing.T) {
	repo := &dualSupplierRepo{
		profiles: map[string]Profile{
			"supplier-a": {SupplierID: "supplier-a", LegalName: "Alpha"},
			"supplier-b": {SupplierID: "supplier-b", LegalName: "Beta"},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SupplierID:     "supplier-a",
		SeedSupplierID: "supplier-a",
		Country:        "UZ",
		Currency:       "UZS",
		MaxSuppliers:   10,
	})

	reqA := httptest.NewRequest(http.MethodGet, "/v1/supplier/topology", nil)
	reqA = reqA.WithContext(auth.WithClaims(reqA.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		SupplierID: "supplier-a",
	}))
	rrA := httptest.NewRecorder()
	svc.handleTopologyGet(rrA, reqA)
	if rrA.Code != http.StatusOK {
		t.Fatalf("supplier-a status=%d body=%s", rrA.Code, rrA.Body.String())
	}
	var respA map[string]any
	if err := json.Unmarshal(rrA.Body.Bytes(), &respA); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if respA["supplier_id"] != "supplier-a" {
		t.Fatalf("supplier_id=%v want supplier-a", respA["supplier_id"])
	}

	reqB := httptest.NewRequest(http.MethodGet, "/v1/supplier/topology", nil)
	reqB = reqB.WithContext(auth.WithClaims(reqB.Context(), auth.Claims{
		Role:       auth.RoleAdmin,
		SupplierID: "supplier-b",
	}))
	rrB := httptest.NewRecorder()
	svc.handleTopologyGet(rrB, reqB)
	if rrB.Code != http.StatusOK {
		t.Fatalf("supplier-b status=%d body=%s", rrB.Code, rrB.Body.String())
	}
	var respB map[string]any
	if err := json.Unmarshal(rrB.Body.Bytes(), &respB); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if respB["supplier_id"] != "supplier-b" {
		t.Fatalf("supplier_id=%v want supplier-b", respB["supplier_id"])
	}
	warehouses, _ := respB["warehouses"].([]any)
	if len(warehouses) == 0 {
		t.Fatal("expected warehouses for supplier-b")
	}
	first, _ := warehouses[0].(map[string]any)
	name, _ := first["Name"].(string)
	if name != "WH supplier-b" {
		t.Fatalf("warehouse Name=%v want WH supplier-b", first["Name"])
	}
}
