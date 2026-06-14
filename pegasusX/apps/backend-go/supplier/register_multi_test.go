package supplier

import (
	"context"
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type countingSupplierRepo struct {
	profiles map[string]Profile
	count    int64
}

func (r *countingSupplierRepo) CountSuppliers(context.Context) (int64, error) {
	return r.count, nil
}

func (r *countingSupplierRepo) GetProfile(_ context.Context, supplierID string) (Profile, bool, error) {
	p, ok := r.profiles[supplierID]
	return p, ok, nil
}

func (r *countingSupplierRepo) UpdateProfile(_ context.Context, p Profile, _ func(outbox.TxnBuffer) error) error {
	if r.profiles == nil {
		r.profiles = make(map[string]Profile)
	}
	r.profiles[p.SupplierID] = p
	return nil
}

func (r *countingSupplierRepo) GetAuthByPhone(context.Context, string) (SupplierAuthRecord, bool, error) {
	return SupplierAuthRecord{}, false, nil
}
func (r *countingSupplierRepo) GetTopology(context.Context, string) (SupplierTopology, error) {
	return SupplierTopology{}, nil
}
func (r *countingSupplierRepo) ReplaceTopology(context.Context, string, SupplierTopology, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *countingSupplierRepo) ListOrgMembers(context.Context, string) ([]SupplierOrgMember, error) {
	return nil, nil
}
func (r *countingSupplierRepo) CreateOrgMember(context.Context, CreateOrgMemberParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *countingSupplierRepo) UpdateOrgMember(context.Context, string, string, UpdateOrgMemberPatch, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *countingSupplierRepo) ListFleetDrivers(context.Context, string) ([]SupplierFleetDriver, error) {
	return nil, nil
}
func (r *countingSupplierRepo) CreateFleetDriver(context.Context, CreateFleetDriverParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *countingSupplierRepo) ListFleetVehicles(context.Context, string) ([]SupplierFleetVehicle, error) {
	return nil, nil
}
func (r *countingSupplierRepo) CreateFleetVehicle(context.Context, CreateFleetVehicleParams, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *countingSupplierRepo) GetPricingRule(context.Context, string) (SupplierPricingRule, bool, error) {
	return SupplierPricingRule{}, false, nil
}
func (r *countingSupplierRepo) UpsertPricingRule(context.Context, SupplierPricingRule, func(outbox.TxnBuffer) error) error {
	return nil
}

func TestResolveRegistrationSupplierIDUsesSeedWhenUnregistered(t *testing.T) {
	repo := &countingSupplierRepo{
		count: 1,
		profiles: map[string]Profile{
			"seed-1": {SupplierID: "seed-1", IsRegistered: false},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SupplierID:     "seed-1",
		SeedSupplierID: "seed-1",
		MaxSuppliers:   10,
	})
	id, err := svc.resolveRegistrationSupplierID(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != "seed-1" {
		t.Fatalf("id=%q want seed-1", id)
	}
}

func TestResolveRegistrationSupplierIDRejectsAtCap(t *testing.T) {
	repo := &countingSupplierRepo{
		count: 10,
		profiles: map[string]Profile{
			"seed-1": {SupplierID: "seed-1", IsRegistered: true},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SupplierID:     "seed-1",
		SeedSupplierID: "seed-1",
		MaxSuppliers:   10,
	})
	_, err := svc.resolveRegistrationSupplierID(context.Background())
	if !errors.Is(err, ErrSupplierCapReached) {
		t.Fatalf("err=%v want supplier_cap_reached", err)
	}
}
