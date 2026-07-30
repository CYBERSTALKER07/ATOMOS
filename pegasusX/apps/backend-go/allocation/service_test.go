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

	inventoryId1 := fmt.Sprintf("inv1-%d", time.Now().UnixNano())
	inventoryId2 := fmt.Sprintf("inv2-%d", time.Now().UnixNano())

	muts := []*spanner.Mutation{
		spanner.Insert("InventoryLevels",
			[]string{"InventoryId", "ProductId", "WarehouseId", "SupplierId", "QuantityOnHand", "QuantityReserved", "ReorderThreshold", "Version", "UpdatedAt"},
			[]interface{}{inventoryId1, "P1", "W1", "S1", 100, 10, 0, 1, spanner.CommitTimestamp}),
		spanner.Insert("InventoryLevels",
			[]string{"InventoryId", "ProductId", "WarehouseId", "SupplierId", "QuantityOnHand", "QuantityReserved", "ReorderThreshold", "Version", "UpdatedAt"},
			[]interface{}{inventoryId2, "P2", "W2", "S1", 50, 0, 0, 1, spanner.CommitTimestamp}),
	}
	_, err := client.Apply(ctx, muts)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	defer func() {
		client.Apply(ctx, []*spanner.Mutation{
			spanner.Delete("InventoryLevels", spanner.Key{inventoryId1}),
			spanner.Delete("InventoryLevels", spanner.Key{inventoryId2}),
		})
	}()

	svc := NewAllocationService(client)

	// Test success
	req := &AllocationRequest{
		Items: []AllocationItem{
			{ProductId: "P1", QuantityRequired: 50},
			{ProductId: "P2", QuantityRequired: 25},
		},
	}
	res, err := svc.AllocateOrder(ctx, req)
	if err != nil {
		t.Fatalf("AllocateOrder failed: %v", err)
	}
	if res.Fulfillments["P1"] != "W1" {
		t.Errorf("Expected W1 for P1, got %s", res.Fulfillments["P1"])
	}
	if res.Fulfillments["P2"] != "W2" {
		t.Errorf("Expected W2 for P2, got %s", res.Fulfillments["P2"])
	}

	// Test failure (insufficient stock)
	reqFail := &AllocationRequest{
		Items: []AllocationItem{
			{ProductId: "P1", QuantityRequired: 100}, // 100-10 = 90 available, so 100 should fail
		},
	}
	_, err = svc.AllocateOrder(ctx, reqFail)
	if err == nil {
		t.Errorf("Expected error for insufficient stock, got nil")
	}

	// Test failure (product not found)
	reqFailNotFound := &AllocationRequest{
		Items: []AllocationItem{
			{ProductId: "P3", QuantityRequired: 1},
		},
	}
	_, err = svc.AllocateOrder(ctx, reqFailNotFound)
	if err == nil {
		t.Errorf("Expected error for product not found, got nil")
	}
}
