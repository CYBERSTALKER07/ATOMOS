package entityresolution

import (
	"context"
	"testing"
	"time"
)

type fakeRepository struct {
	findExact   func(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error)
	listScoped  func(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error)
	loadLineage func(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error)
}

func (f fakeRepository) FindExactByID(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
	if f.findExact == nil {
		return nil, nil
	}
	return f.findExact(ctx, supplierID, entityType, entityID)
}

func (f fakeRepository) ListScopedRecords(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
	if f.listScoped == nil {
		return nil, nil
	}
	return f.listScoped(ctx, supplierID, entityType, perTypeLimit)
}

func (f fakeRepository) LoadLineage(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error) {
	if f.loadLineage == nil {
		return nil, nil
	}
	return f.loadLineage(ctx, supplierID, entityType, entityID)
}

func TestResolvePrefersDeterministicExactMatch(t *testing.T) {
	svc := NewService(fakeRepository{
		findExact: func(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
			return []EntityRecord{{
				EntityType: EntityTypeOrder,
				EntityID:   "ord-1",
				Label:      "ord-1",
				SearchText: "ord-1 delivered",
			}}, nil
		},
		listScoped: func(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
			return []EntityRecord{{
				EntityType: EntityTypeOrder,
				EntityID:   "ord-2",
				Label:      "ord-2",
				SearchText: "ord-2 pending",
			}}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	result, err := svc.Resolve(ctx, ResolveInput{
		SupplierID:    "sup-1",
		EntityType:    EntityTypeOrder,
		EntityID:      "ord-1",
		Query:         "ord",
		MaxCandidates: 3,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Resolved == nil {
		t.Fatalf("Resolve() resolved candidate is nil")
	}
	if result.Resolved.EntityID != "ord-1" {
		t.Fatalf("resolved entity_id = %q, want %q", result.Resolved.EntityID, "ord-1")
	}
	if !result.Resolved.Deterministic {
		t.Fatalf("resolved deterministic = false, want true")
	}
	if len(result.Candidates) == 0 {
		t.Fatalf("expected candidates")
	}
}

func TestResolveRanksProbabilisticMatches(t *testing.T) {
	svc := NewService(fakeRepository{
		listScoped: func(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
			return []EntityRecord{
				{
					EntityType: EntityTypeDriver,
					EntityID:   "drv-1",
					Label:      "Alex Karimov",
					SearchText: "alex karimov +998901112233",
				},
				{
					EntityType: EntityTypeWarehouse,
					EntityID:   "wh-1",
					Label:      "Central Hub",
					SearchText: "central hub tashkent",
				},
			}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	result, err := svc.Resolve(ctx, ResolveInput{
		SupplierID:    "sup-1",
		EntityType:    EntityTypeAny,
		Query:         "alex",
		MaxCandidates: 2,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Resolved == nil {
		t.Fatalf("Resolve() resolved candidate is nil")
	}
	if result.Resolved.EntityType != EntityTypeDriver {
		t.Fatalf("resolved entity_type = %q, want %q", result.Resolved.EntityType, EntityTypeDriver)
	}
	if result.Resolved.Deterministic {
		t.Fatalf("resolved deterministic = true, want false")
	}
}

func TestExplainBuildsProjection(t *testing.T) {
	svc := NewService(fakeRepository{
		findExact: func(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
			return []EntityRecord{{
				EntityType: EntityTypeDriver,
				EntityID:   "drv-1",
				Label:      "Driver A",
				SearchText: "driver a",
			}}, nil
		},
		loadLineage: func(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error) {
			return []LineageLink{
				{TargetType: EntityTypeSupplier, TargetID: "sup-1", TargetLabel: "Supplier 1", Relation: "belongs_to_supplier", Confidence: 0.99},
				{TargetType: EntityTypeVehicle, TargetID: "veh-7", TargetLabel: "Vehicle 7", Relation: "assigned_vehicle", Confidence: 0.97},
			}, nil
		},
	})
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	result, err := svc.Explain(ctx, ExplainInput{
		SupplierID: "sup-1",
		EntityType: EntityTypeDriver,
		EntityID:   "drv-1",
	})
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if result.Source.NodeID != "DRIVER:drv-1" {
		t.Fatalf("source node_id = %q, want %q", result.Source.NodeID, "DRIVER:drv-1")
	}
	if len(result.Projection.Nodes) != 3 {
		t.Fatalf("projection nodes len = %d, want %d", len(result.Projection.Nodes), 3)
	}
	if len(result.Projection.Edges) != 2 {
		t.Fatalf("projection edges len = %d, want %d", len(result.Projection.Edges), 2)
	}
}
