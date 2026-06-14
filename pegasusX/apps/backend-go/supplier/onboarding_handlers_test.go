package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type onboardingTestRepo struct {
	pricingTestRepo
	topology    SupplierTopology
	members     []SupplierOrgMember
	drivers     []SupplierFleetDriver
	vehicles    []SupplierFleetVehicle
	lastMember  CreateOrgMemberParams
	lastDriver  CreateFleetDriverParams
	lastVehicle CreateFleetVehicleParams
}

func (r *onboardingTestRepo) GetTopology(_ context.Context, _ string) (SupplierTopology, error) {
	return r.topology, nil
}

func (r *onboardingTestRepo) ListOrgMembers(_ context.Context, _ string) ([]SupplierOrgMember, error) {
	out := make([]SupplierOrgMember, len(r.members))
	copy(out, r.members)
	return out, nil
}

func (r *onboardingTestRepo) CreateOrgMember(_ context.Context, member CreateOrgMemberParams, _ func(outbox.TxnBuffer) error) error {
	r.lastMember = member
	r.members = append(r.members, SupplierOrgMember{
		UserID:              member.UserID,
		SupplierID:          member.SupplierID,
		Name:                member.Name,
		Email:               member.Email,
		Phone:               member.Phone,
		SupplierRole:        member.SupplierRole,
		AssignedWarehouseID: member.AssignedWarehouseID,
		AssignedFactoryID:   member.AssignedFactoryID,
		IsActive:            member.IsActive,
		CreatedAt:           member.CreatedAt,
		UpdatedAt:           member.UpdatedAt,
	})
	return nil
}

func (r *onboardingTestRepo) UpdateOrgMember(_ context.Context, supplierID, userID string, patch UpdateOrgMemberPatch, _ func(outbox.TxnBuffer) error) error {
	for i := range r.members {
		if r.members[i].UserID != userID || r.members[i].SupplierID != supplierID {
			continue
		}
		if patch.Name != nil {
			r.members[i].Name = *patch.Name
		}
		if patch.SupplierRole != nil {
			r.members[i].SupplierRole = *patch.SupplierRole
		}
		if patch.AssignedWarehouseID != nil {
			r.members[i].AssignedWarehouseID = *patch.AssignedWarehouseID
		}
		if patch.AssignedFactoryID != nil {
			r.members[i].AssignedFactoryID = *patch.AssignedFactoryID
		}
		if patch.IsActive != nil {
			r.members[i].IsActive = *patch.IsActive
		}
		return nil
	}
	return errOrgMemberNotFound
}

func (r *onboardingTestRepo) ListFleetDrivers(_ context.Context, _ string) ([]SupplierFleetDriver, error) {
	out := make([]SupplierFleetDriver, len(r.drivers))
	copy(out, r.drivers)
	return out, nil
}

func (r *onboardingTestRepo) CreateFleetDriver(_ context.Context, driver CreateFleetDriverParams, _ func(outbox.TxnBuffer) error) error {
	r.lastDriver = driver
	r.drivers = append(r.drivers, SupplierFleetDriver{
		DriverID:     driver.DriverID,
		SupplierID:   driver.SupplierID,
		Name:         driver.Name,
		Phone:        driver.Phone,
		HomeNodeType: driver.HomeNodeType,
		HomeNodeID:   driver.HomeNodeID,
		VehicleID:    driver.VehicleID,
		IsActive:     driver.IsActive,
		CreatedAt:    driver.CreatedAt,
		UpdatedAt:    driver.UpdatedAt,
	})
	return nil
}

func (r *onboardingTestRepo) ListFleetVehicles(_ context.Context, _ string) ([]SupplierFleetVehicle, error) {
	out := make([]SupplierFleetVehicle, len(r.vehicles))
	copy(out, r.vehicles)
	return out, nil
}

func (r *onboardingTestRepo) CreateFleetVehicle(_ context.Context, vehicle CreateFleetVehicleParams, _ func(outbox.TxnBuffer) error) error {
	r.lastVehicle = vehicle
	r.vehicles = append(r.vehicles, SupplierFleetVehicle{
		VehicleID:    vehicle.VehicleID,
		SupplierID:   vehicle.SupplierID,
		Label:        vehicle.Label,
		LicensePlate: vehicle.LicensePlate,
		HomeNodeType: vehicle.HomeNodeType,
		HomeNodeID:   vehicle.HomeNodeID,
		IsActive:     vehicle.IsActive,
		CreatedAt:    vehicle.CreatedAt,
		UpdatedAt:    vehicle.UpdatedAt,
	})
	return nil
}

func TestHandleOrgMembersPostCreatesWarehouseAdmin(t *testing.T) {
	repo := &onboardingTestRepo{topology: SupplierTopology{Warehouses: []WarehouseNode{{WarehouseID: "wh-1", Name: "Primary"}}}}
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Country: "UZ", Currency: "UZS", Now: func() time.Time { return now }})

	body := `{"name":"Warehouse Admin","phone":"+998901234567","password":"secretpass","supplier_role":"WAREHOUSE_ADMIN","assigned_warehouse_id":"wh-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/org/members", strings.NewReader(body))
	rr := httptest.NewRecorder()

	svc.HandleOrgMembers(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if repo.lastMember.SupplierRole != auth.RoleWarehouseAdmin {
		t.Fatalf("supplier_role=%q want=%q", repo.lastMember.SupplierRole, auth.RoleWarehouseAdmin)
	}
	if repo.lastMember.AssignedWarehouseID != "wh-1" {
		t.Fatalf("assigned_warehouse_id=%q want=%q", repo.lastMember.AssignedWarehouseID, "wh-1")
	}
	var payload supplierOrgMembersResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items=%d want=1", len(payload.Items))
	}
	if payload.Items[0].SupplierRole != auth.RoleWarehouseAdmin {
		t.Fatalf("response supplier_role=%q want=%q", payload.Items[0].SupplierRole, auth.RoleWarehouseAdmin)
	}
}

func TestHandleFleetDriversPostRejectsUnknownHomeNode(t *testing.T) {
	repo := &onboardingTestRepo{topology: SupplierTopology{Warehouses: []WarehouseNode{{WarehouseID: "wh-1", Name: "Primary"}}}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Country: "UZ", Currency: "UZS"})

	body := `{"name":"Driver One","phone":"+998901234567","pin":"1234","home_node_type":"FACTORY","home_node_id":"fc-9"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/fleet/drivers", strings.NewReader(body))
	rr := httptest.NewRecorder()

	svc.HandleFleetDrivers(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusUnprocessableEntity, rr.Body.String())
	}
}

func TestHandleFleetVehiclesPostCreatesVehicle(t *testing.T) {
	repo := &onboardingTestRepo{topology: SupplierTopology{Factories: []FactoryNode{{FactoryID: "fc-1", Name: "Factory"}}}}
	now := time.Date(2026, 5, 23, 13, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Country: "UZ", Currency: "UZS", Now: func() time.Time { return now }})

	body := `{"label":"Truck 7","license_plate":"80A777AA","home_node_type":"FACTORY","home_node_id":"fc-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/fleet/vehicles", strings.NewReader(body))
	rr := httptest.NewRecorder()

	svc.HandleFleetVehicles(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if repo.lastVehicle.HomeNodeType != auth.HomeNodeFactory {
		t.Fatalf("home_node_type=%q want=%q", repo.lastVehicle.HomeNodeType, auth.HomeNodeFactory)
	}
	if repo.lastVehicle.LicensePlate != "80A777AA" {
		t.Fatalf("license_plate=%q want=%q", repo.lastVehicle.LicensePlate, "80A777AA")
	}
	var payload supplierFleetVehiclesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items=%d want=1", len(payload.Items))
	}
}
