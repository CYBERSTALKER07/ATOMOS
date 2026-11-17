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

func TestNormalizeTINAndUZPhone(t *testing.T) {
	if got := NormalizeTIN(" INN 301-234 567 "); got != "301234567" {
		t.Fatalf("NormalizeTIN = %q, want 301234567", got)
	}
	for _, raw := range []string{
		"+998 (90) 123-45-67",
		"998901234567",
		"90 123 45 67",
		"090-123-45-67",
		"00998901234567",
	} {
		if got := NormalizeUZPhone(raw); got != "+998901234567" {
			t.Fatalf("NormalizeUZPhone(%q) = %q, want +998901234567", raw, got)
		}
	}
}

func TestMemoryRetailerLinkRepository_TenantSafeDedupAndResolve(t *testing.T) {
	repo := NewMemoryRetailerLinkRepository()
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)

	r1, peers, err := repo.Upsert(ctx, SupplierRetailerLink{
		SupplierID:      "sup-1",
		RetailerID:      "ret-1",
		TaxIDTIN:        "301-234-567",
		NormalizedPhone: "90 111 22 33",
		LegalName:       "Baraka LLC",
	}, now)
	if err != nil {
		t.Fatalf("upsert r1: %v", err)
	}
	if r1.DedupStatus != DedupStatusClean || len(peers) != 0 {
		t.Fatalf("expected initial link CLEAN with 0 peers, got %+v peers=%v", r1, peers)
	}

	// Same TIN/phone under another supplier must stay strictly isolated and CLEAN.
	otherTenant, peers, err := repo.Upsert(ctx, SupplierRetailerLink{
		SupplierID:      "sup-2",
		RetailerID:      "ret-other",
		TaxIDTIN:        "INN 301234567",
		NormalizedPhone: "+998901112233",
		LegalName:       "Baraka LLC",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("upsert otherTenant: %v", err)
	}
	if otherTenant.DedupStatus != DedupStatusClean || len(peers) != 0 {
		t.Fatalf("cross-tenant link leaked into cluster: %+v peers=%v", otherTenant, peers)
	}

	// Same supplier phone variant triggers shared cluster + DUPLICATE_CANDIDATE on both rows.
	r2, peers, err := repo.Upsert(ctx, SupplierRetailerLink{
		SupplierID:      "sup-1",
		RetailerID:      "ret-2",
		NormalizedPhone: "0901112233",
		LegalName:       "Baraka Filial 2",
	}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("upsert r2: %v", err)
	}
	if r2.DedupStatus != DedupStatusDuplicateCandidate || r2.DedupClusterID == "" {
		t.Fatalf("expected DUPLICATE_CANDIDATE with cluster id, got %+v", r2)
	}
	if len(peers) != 1 || peers[0].RetailerID != "ret-1" || peers[0].DedupClusterID != r2.DedupClusterID {
		t.Fatalf("expected ret-1 updated to cluster %s, got %+v", r2.DedupClusterID, peers)
	}

	// Third duplicate via normalized TIN joins the same cluster.
	r3, peers, err := repo.Upsert(ctx, SupplierRetailerLink{
		SupplierID: "sup-1",
		RetailerID: "ret-3",
		TaxIDTIN:   "INN 301 234 567",
	}, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("upsert r3: %v", err)
	}
	if r3.DedupClusterID != r2.DedupClusterID || len(peers) != 0 {
		t.Fatalf("expected r3 in existing cluster %s without rewriting peers, got %+v peers=%v", r2.DedupClusterID, r3, peers)
	}

	// Merge ret-2 into ret-1; cluster still has ret-1 + ret-3.
	merged, cleaned, err := repo.Resolve(ctx, "sup-1", "ret-2", "merge_into", "ret-1", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("resolve merge_into: %v", err)
	}
	if merged.DedupStatus != DedupStatusMerged || merged.MergedIntoID != "ret-1" || len(cleaned) != 0 {
		t.Fatalf("unexpected merge result %+v cleaned=%v", merged, cleaned)
	}

	// Mark ret-3 clean; sole remaining candidate ret-1 also becomes CLEAN cleanly.
	r3Clean, cleaned, err := repo.Resolve(ctx, "sup-1", "ret-3", "mark_clean", "", now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("resolve mark_clean: %v", err)
	}
	if r3Clean.DedupStatus != DedupStatusClean || r3Clean.DedupClusterID != "" {
		t.Fatalf("expected ret-3 CLEAN, got %+v", r3Clean)
	}
	if len(cleaned) != 1 || cleaned[0].RetailerID != "ret-1" || cleaned[0].DedupStatus != DedupStatusClean {
		t.Fatalf("expected sole survivor ret-1 auto-cleaned, got %+v", cleaned)
	}

	// Cross-supplier resolution must not see sup-1 retailers.
	if _, _, err := repo.Resolve(ctx, "sup-2", "ret-1", "mark_clean", "", now); err == nil {
		t.Fatal("expected error when sup-2 tries to resolve sup-1 retailer")
	}
}

func TestResolveMatchesNormalizedTINAndUZPhoneDeterministically(t *testing.T) {
	svc := NewService(fakeRepository{
		listScoped: func(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
			return []EntityRecord{
				{
					EntityType: EntityTypeRetailer,
					EntityID:   "ret-1",
					Label:      "Navoiy Trade",
					Metadata: map[string]string{
						"tax_id_tin":       "301234567",
						"normalized_phone": "+998901234567",
					},
				},
			}, nil
		},
	})

	ctx := context.Background()
	resTIN, err := svc.Resolve(ctx, ResolveInput{
		SupplierID: "sup-1",
		EntityType: EntityTypeRetailer,
		Query:      "INN 301-234-567",
	})
	if err != nil {
		t.Fatalf("Resolve TIN error: %v", err)
	}
	if resTIN.Resolved == nil || !resTIN.Resolved.Deterministic || resTIN.Resolved.EntityID != "ret-1" {
		t.Fatalf("expected deterministic TIN match for ret-1, got %+v", resTIN.Resolved)
	}

	resPhone, err := svc.Resolve(ctx, ResolveInput{
		SupplierID: "sup-1",
		EntityType: EntityTypeRetailer,
		Query:      "90 123 45 67",
	})
	if err != nil {
		t.Fatalf("Resolve phone error: %v", err)
	}
	if resPhone.Resolved == nil || !resPhone.Resolved.Deterministic || resPhone.Resolved.EntityID != "ret-1" {
		t.Fatalf("expected deterministic UZ phone match for ret-1, got %+v", resPhone.Resolved)
	}
}
