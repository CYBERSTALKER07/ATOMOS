package warehouse

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func openWarehouseSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner warehouse valuation hook test")
	}

	project := envDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner client (%s): %v", dbPath, err)
	}
	t.Cleanup(client.Close)
	return client
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func TestParseReceiveItems_ValuationFields(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"items":[{"item_id":"item-1","received_quantity":12,"unit_cost_minor":4500,"vat_minor":6480,"credit_account":"AP:VENDOR","vendor_id":"vend-99","expiry_date":"2026-12-31"}]}`)
	lines := parseReceiveItems(raw)
	if len(lines) != 1 {
		t.Fatalf("parsed %d lines, want 1", len(lines))
	}
	l := lines[0]
	if l.ItemID != "item-1" || l.ReceivedQuantity != 12 || l.UnitCostMinor != 4500 || l.VatMinor != 6480 || l.CreditAccount != "AP:VENDOR" || l.VendorID != "vend-99" || l.ExpiryDate != "2026-12-31" {
		t.Fatalf("unexpected parsed receive line: %+v", l)
	}
}

func TestWarehouse_ReceiveTransfer_InvokesValuationReceiptHook(t *testing.T) {
	client := openWarehouseSpannerClient(t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	supID := "sup-wvh-" + suffix
	whID := "wh-wvh-" + suffix
	reqID := "req-wvh-" + suffix
	itemID := "item-wvh-" + suffix
	trfID := "trf-wvh-" + suffix
	prodID := "prod-wvh-" + suffix
	now := time.Now().UTC()

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("FactoryInternalTransfers", spanner.Key{trfID}),
			spanner.Delete("WarehouseSupplyRequests", spanner.Key{reqID}),
			spanner.Delete("SupplierInventoryV2", spanner.Key{supID, prodID, whID}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("WarehouseSupplyRequests", map[string]any{
			"RequestId": reqID, "SupplierId": supID, "WarehouseId": whID,
			"State": "SHIPPED", "CoverageStartDate": "2026-11-16", "CoverageDays": int64(7),
			"ProjectedUnits": int64(10), "CommittedUnits": int64(10), "PendingConfirmationUnits": int64(0),
			"CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("WarehouseSupplyRequestItems", map[string]any{
			"RequestId": reqID, "ItemId": itemID, "ProductId": prodID,
			"RequestedQuantity": int64(10), "ShippedQuantity": int64(10), "CreatedAt": now,
		}),
		spanner.InsertMap("FactoryInternalTransfers", map[string]any{
			"TransferId": trfID, "FactoryId": "fac-" + suffix, "SupplierId": supID,
			"WarehouseId": whID, "SupplyRequestId": reqID, "State": "APPROVED", "CreatedAt": now, "UpdatedAt": now,
		}),
	}); err != nil {
		t.Fatalf("seed warehouse receive fixtures: %v", err)
	}

	svc := NewService(ServiceConfig{Spanner: client, Currency: "UZS"})
	var captured []ReceiptValuationArgs
	svc.SetValuationReceiptHook(func(_ context.Context, _ *spanner.ReadWriteTransaction, items []ReceiptValuationArgs) error {
		captured = append(captured, items...)
		return nil
	})

	ops := &auth.WarehouseOps{SupplierID: supID, WarehouseID: whID, Subject: "wh-mgr-1"}
	lines := []receiveLineInput{{
		ItemID:           itemID,
		ReceivedQuantity: 10,
		UnitCostMinor:    3_500,
		VatMinor:         4_200,
		CreditAccount:    "AP:VENDOR",
		VendorID:         "vend-1",
		ExpiryDate:       "2027-06-30",
	}}
	if err := svc.receiveTransferWithItems(auth.WithClaims(ctx, auth.Claims{Subject: "wh-mgr-1", Role: auth.RoleWarehouseAdmin, SupplierID: supID}), ops, trfID, lines); err != nil {
		t.Fatalf("receiveTransferWithItems: %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("captured valuation items = %d, want 1 (%+v)", len(captured), captured)
	}
	got := captured[0]
	if got.EntityID != supID || got.WarehouseID != whID || got.ProductID != prodID || got.Quantity != 10 || got.UnitCostMinor != 3_500 || got.VatMinor != 4_200 || got.CreditAccount != "AP:VENDOR" || got.VendorID != "vend-1" || got.ActorID != "wh-mgr-1" {
		t.Fatalf("unexpected ReceiptValuationArgs: %+v", got)
	}
}
