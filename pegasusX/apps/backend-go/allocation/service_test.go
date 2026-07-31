package allocation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func newSpannerIntegrationClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()

	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	emulatorHost := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if emulatorHost == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping integration test")
	}

	project := os.Getenv("SPANNER_PROJECT")
	if project == "" {
		project = "pegasusx-local"
	}
	instance := os.Getenv("SPANNER_INSTANCE")
	if instance == "" {
		instance = "pegasusx-instance"
	}
	database := os.Getenv("SPANNER_DATABASE")
	if database == "" {
		database = "pegasusx-db"
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator database unavailable (%s): %v", dbPath, err)
		}
		t.Skipf("spanner emulator database unavailable (%s): %v", dbPath, err)
	}

	return client
}

func TestAllocateOrder(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	supplierID := "S1-" + suffix
	wh1 := "W1-" + suffix
	wh2 := "W2-" + suffix
	whInactive := "W3-" + suffix

	muts := []*spanner.Mutation{
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId":        wh1,
			"SupplierId":         supplierID,
			"Name":               "WH1",
			"CoverageRadiusKm":   10.0,
			"IsActive":           true,
			"IsOnShift":          true,
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId":        wh2,
			"SupplierId":         supplierID,
			"Name":               "WH2",
			"CoverageRadiusKm":   10.0,
			"IsActive":           true,
			"IsOnShift":          true,
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId":        whInactive,
			"SupplierId":         supplierID,
			"Name":               "WH3",
			"CoverageRadiusKm":   10.0,
			"IsActive":           false,
			"IsOnShift":          false,
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      wh1,
			"ProductId":        "P1",
			"QuantityOnHand":   100,
			"QuantityReserved": 10,
			"ReorderThreshold": 0,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      wh2,
			"ProductId":        "P2",
			"QuantityOnHand":   50,
			"QuantityReserved": 0,
			"ReorderThreshold": 0,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      whInactive,
			"ProductId":        "P3",
			"QuantityOnHand":   1000,
			"QuantityReserved": 0,
			"ReorderThreshold": 0,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	}
	_, err := client.Apply(ctx, muts)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	defer func() {
		client.Apply(ctx, []*spanner.Mutation{
			spanner.Delete("SupplierInventoryV2", spanner.Key{supplierID, wh1, "P1"}),
			spanner.Delete("SupplierInventoryV2", spanner.Key{supplierID, wh2, "P2"}),
			spanner.Delete("SupplierInventoryV2", spanner.Key{supplierID, whInactive, "P3"}),
			spanner.Delete("Warehouses", spanner.Key{wh1}),
			spanner.Delete("Warehouses", spanner.Key{wh2}),
			spanner.Delete("Warehouses", spanner.Key{whInactive}),
		})
	}()

	svc := NewAllocationService(client)

	req := &AllocationRequest{
		SupplierId: supplierID,
		Items: []AllocationItem{
			{ProductId: "P1", QuantityRequired: 50},
			{ProductId: "P2", QuantityRequired: 25},
		},
	}
	res, err := svc.AllocateOrder(ctx, req)
	if err != nil {
		t.Fatalf("AllocateOrder failed: %v", err)
	}
	if res.Fulfillments["P1"] != wh1 {
		t.Errorf("Expected %s for P1, got %s", wh1, res.Fulfillments["P1"])
	}
	if res.Fulfillments["P2"] != wh2 {
		t.Errorf("Expected %s for P2, got %s", wh2, res.Fulfillments["P2"])
	}

	reqFail := &AllocationRequest{
		SupplierId: supplierID,
		Items: []AllocationItem{
			{ProductId: "P1", QuantityRequired: 100},
		},
	}
	_, err = svc.AllocateOrder(ctx, reqFail)
	if err == nil {
		t.Errorf("Expected error for insufficient stock, got nil")
	}

	reqInactive := &AllocationRequest{
		SupplierId: supplierID,
		Items: []AllocationItem{
			{ProductId: "P3", QuantityRequired: 1},
		},
	}
	_, err = svc.AllocateOrder(ctx, reqInactive)
	if err == nil {
		t.Errorf("Expected error for inactive warehouse product, got nil")
	}

	reqFailNotFound := &AllocationRequest{
		SupplierId: supplierID,
		Items: []AllocationItem{
			{ProductId: "P99", QuantityRequired: 1},
		},
	}
	_, err = svc.AllocateOrder(ctx, reqFailNotFound)
	if err == nil {
		t.Errorf("Expected error for product not found, got nil")
	}
}
