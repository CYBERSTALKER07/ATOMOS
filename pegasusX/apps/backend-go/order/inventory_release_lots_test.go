package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

func TestAggregateSKUQuantities_MergesDuplicateSKUsAndSkipsNonPositive(t *testing.T) {
	got := aggregateSKUQuantities([]LineItem{
		{SKU: " sku-a ", Quantity: 2},
		{SKU: "sku-b", Quantity: 0},
		{SKU: "sku-a", Quantity: 3},
		{SKU: "", Quantity: 5},
		{SKU: "sku-c", Quantity: -1},
	})
	if len(got) != 1 || got["sku-a"] != 5 {
		t.Fatalf("unexpected aggregated quantities: %v", got)
	}
}

func TestReconcilePreorderReservationsInTxn_NilTxnAndEmptyWarehouseNoop(t *testing.T) {
	lines := []LineItem{{SKU: "sku-a", Quantity: 2}}
	if err := reconcilePreorderReservationsInTxn(t.Context(), nil, "sup-1", "wh-1", "ord-1", OrderSourceManualPreorder, lines, lines); err != nil {
		t.Fatalf("nil txn should no-op for memory repo, got %v", err)
	}
	if err := ReleaseReservationsForOrderInTxn(t.Context(), nil, "sup-1", "", "ord-1", OrderSourceManual, lines); err != nil {
		t.Fatalf("empty warehouse should no-op, got %v", err)
	}
	for _, src := range []OrderSource{OrderSourceBackorder, OrderSourceMigration} {
		if err := ReleaseReservationsForOrderInTxn(t.Context(), nil, "sup-1", "wh-1", "ord-1", src, lines); err != nil {
			t.Fatalf("source %s should no-op before touching txn, got %v", src, err)
		}
	}
}

func TestBuildOrderLotReleaseMutationsInTxn_NilAndEmptyOrderGuard(t *testing.T) {
	lotQty, muts, err := buildOrderLotReleaseMutationsInTxn(t.Context(), nil, "ord-1")
	if err != nil || lotQty != nil || muts != nil {
		t.Fatalf("expected (nil, nil, nil) for nil txn, got (%v, %v, %v)", lotQty, muts, err)
	}
}

func TestBuildInventoryLotReleasedEvent_DeterministicOrderAndFields(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 25, 0, 123456000, time.UTC)
	lotQty := map[string]int64{
		"lot-z":    3,
		"lot-a":    5,
		"lot-zero": 0,
		"lot-neg":  -2,
	}

	ev := buildInventoryLotReleasedEvent("ord-42", "sup-7", "wh-3", lotQty, now)
	if ev.Type != events.EventInventoryLotReleased {
		t.Fatalf("Type = %q, want %q", ev.Type, events.EventInventoryLotReleased)
	}
	if ev.Version != 1 {
		t.Fatalf("Version = %d, want 1", ev.Version)
	}
	if ev.Timestamp != now.Format(time.RFC3339Nano) {
		t.Fatalf("Timestamp = %q, want %q", ev.Timestamp, now.Format(time.RFC3339Nano))
	}
	if ev.OrderID != "ord-42" || ev.SupplierID != "sup-7" || ev.WarehouseID != "wh-3" {
		t.Fatalf("unexpected identifiers in event: %+v", ev)
	}
	wantLots := []events.ReleasedLotQuantity{
		{LotID: "lot-a", Quantity: 5},
		{LotID: "lot-z", Quantity: 3},
	}
	if !reflect.DeepEqual(ev.ReleasedLots, wantLots) {
		t.Fatalf("ReleasedLots = %+v, want %+v", ev.ReleasedLots, wantLots)
	}

	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal event map: %v", err)
	}
	if decoded["order_id"] != "ord-42" || decoded["supplier_id"] != "sup-7" || decoded["warehouse_id"] != "wh-3" {
		t.Fatalf("unexpected JSON payload fields: %v", decoded)
	}
}

type lotReleaseSeed struct {
	lotID            string
	lotReserved      int64
	orderReservation int64
}

type testOutboxRow struct {
	eventID       string
	aggregateType string
	aggregateID   string
	topicName     string
	supplierID    string
	payload       []byte
}

func requireLotReleaseSpannerClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	client := newSpannerIntegrationClient(t, ctx)
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := client.Single().ReadRow(probeCtx, "StockLots", spanner.Key{"__t1_probe__"}, []string{"LotId"})
	if err != nil && !isSpannerNotFound(err) {
		code := spanner.ErrCode(err)
		if code == codes.Unavailable || code == codes.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
			t.Skipf("spanner emulator unreachable at %q: %v", os.Getenv("SPANNER_EMULATOR_HOST"), err)
		}
		t.Fatalf("probe StockLots on live emulator: %v", err)
	}
	return client
}

func seedOrderLotReleaseFixture(
	t *testing.T,
	ctx context.Context,
	client *spanner.Client,
	orderID, supplierID, warehouseID, productID string,
	bagReserved int64,
	lots []lotReleaseSeed,
) {
	t.Helper()
	muts := []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"SupplierId":         supplierID,
			"RetailerId":         "ret-t1-release",
			"WarehouseId":        warehouseID,
			"Status":             string(StatusCancelled),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(1000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       supplierID,
			"WarehouseId":      warehouseID,
			"ProductId":        productID,
			"QuantityOnHand":   int64(50),
			"QuantityReserved": bagReserved,
			"ReorderThreshold": int64(0),
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
	}
	for _, lot := range lots {
		muts = append(muts,
			spanner.InsertMap("StockLots", map[string]any{
				"LotId":            lot.lotID,
				"SupplierId":       supplierID,
				"WarehouseId":      warehouseID,
				"ProductId":        productID,
				"LocationId":       "loc-t1-release",
				"LotCode":          "LOT-" + lot.lotID,
				"QuantityOnHand":   int64(20),
				"QuantityReserved": lot.lotReserved,
				"Status":           "AVAILABLE",
				"ReceivedAt":       spanner.CommitTimestamp,
				"CreatedAt":        spanner.CommitTimestamp,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
			spanner.InsertMap("OrderLotReservations", map[string]any{
				"OrderId":   orderID,
				"LotId":     lot.lotID,
				"Quantity":  lot.orderReservation,
				"CreatedAt": spanner.CommitTimestamp,
			}),
		)
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed lot release fixture: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = client.ReadWriteTransaction(cleanCtx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, _ = txn.Update(ctx, spanner.Statement{
				SQL:    "DELETE FROM OutboxEvents WHERE AggregateType = @agg AND AggregateId = @id",
				Params: map[string]any{"agg": events.AggregateOrder, "id": orderID},
			})
			delMuts := make([]*spanner.Mutation, 0, len(lots)*2+2)
			for _, lot := range lots {
				delMuts = append(delMuts,
					spanner.Delete("OrderLotReservations", spanner.Key{orderID, lot.lotID}),
					spanner.Delete("StockLots", spanner.Key{lot.lotID}),
				)
			}
			delMuts = append(delMuts,
				spanner.Delete("SupplierInventoryV2", spanner.Key{supplierID, warehouseID, productID}),
				spanner.Delete("Orders", spanner.Key{orderID}),
			)
			return txn.BufferWrite(delMuts)
		})
	})
}

func readOutboxEventsByOrderID(t *testing.T, ctx context.Context, client *spanner.Client, orderID string) []testOutboxRow {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, TopicName, SupplierId, Payload
		      FROM OutboxEvents
		      WHERE AggregateType = @agg AND AggregateId = @id`,
		Params: map[string]any{"agg": events.AggregateOrder, "id": orderID},
	})
	defer iter.Stop()
	var rows []testOutboxRow
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var out testOutboxRow
		var sup spanner.NullString
		if err := row.Columns(&out.eventID, &out.aggregateType, &out.aggregateID, &out.topicName, &sup, &out.payload); err != nil {
			t.Fatalf("scan OutboxEvents row: %v", err)
		}
		out.supplierID = sup.StringVal
		rows = append(rows, out)
	}
	return rows
}

func readStockLotReservedForTest(t *testing.T, ctx context.Context, client *spanner.Client, lotID string) int64 {
	t.Helper()
	row, err := client.Single().ReadRow(ctx, "StockLots", spanner.Key{lotID}, []string{"QuantityReserved"})
	if err != nil {
		t.Fatalf("read StockLots(%s): %v", lotID, err)
	}
	var qr int64
	if err := row.Column(0, &qr); err != nil {
		t.Fatalf("decode StockLots.QuantityReserved: %v", err)
	}
	return qr
}

func readSupplierInventoryReservedForTest(t *testing.T, ctx context.Context, client *spanner.Client, supplierID, warehouseID, productID string) int64 {
	t.Helper()
	row, err := client.Single().ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supplierID, warehouseID, productID}, []string{"QuantityReserved"})
	if err != nil {
		t.Fatalf("read SupplierInventoryV2(%s,%s,%s): %v", supplierID, warehouseID, productID, err)
	}
	var qr int64
	if err := row.Column(0, &qr); err != nil {
		t.Fatalf("decode SupplierInventoryV2.QuantityReserved: %v", err)
	}
	return qr
}

func countOrderLotReservationsForTest(t *testing.T, ctx context.Context, client *spanner.Client, orderID string) int64 {
	t.Helper()
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT COUNT(*) FROM OrderLotReservations WHERE OrderId = @oid",
		Params: map[string]any{"oid": orderID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		t.Fatalf("count OrderLotReservations: %v", err)
	}
	var cnt int64
	if err := row.Column(0, &cnt); err != nil {
		t.Fatalf("decode OrderLotReservations count: %v", err)
	}
	return cnt
}

func TestSpannerReleaseReservationsForOrderInTxn_EmitsLotReleasedEventAndDeletesReservations(t *testing.T) {
	ctx := context.Background()
	client := requireLotReleaseSpannerClient(t, ctx)
	defer client.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	orderID := "ord-t1-rel-" + suffix
	supplierID := "sup-t1-rel-" + suffix
	warehouseID := "wh-t1-rel-" + suffix
	productID := "sku-t1-rel-" + suffix
	lotA := "lot-a-" + suffix
	lotB := "lot-b-" + suffix

	seedOrderLotReleaseFixture(t, ctx, client, orderID, supplierID, warehouseID, productID, 9, []lotReleaseSeed{
		{lotID: lotB, lotReserved: 6, orderReservation: 5},
		{lotID: lotA, lotReserved: 3, orderReservation: 2},
	})

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, orderID, OrderSourceManual, []LineItem{{SKU: productID, Quantity: 7}})
	})
	if err != nil {
		t.Fatalf("ReleaseReservationsForOrderInTxn: %v", err)
	}

	if got := readStockLotReservedForTest(t, ctx, client, lotA); got != 1 {
		t.Fatalf("lotA QuantityReserved = %d, want 1", got)
	}
	if got := readStockLotReservedForTest(t, ctx, client, lotB); got != 1 {
		t.Fatalf("lotB QuantityReserved = %d, want 1", got)
	}
	if got := readSupplierInventoryReservedForTest(t, ctx, client, supplierID, warehouseID, productID); got != 2 {
		t.Fatalf("SupplierInventoryV2 QuantityReserved = %d, want 2", got)
	}
	if got := countOrderLotReservationsForTest(t, ctx, client, orderID); got != 0 {
		t.Fatalf("OrderLotReservations remaining = %d, want 0", got)
	}

	outboxRows := readOutboxEventsByOrderID(t, ctx, client, orderID)
	if len(outboxRows) != 1 {
		t.Fatalf("OutboxEvents count = %d, want 1", len(outboxRows))
	}
	row := outboxRows[0]
	if row.topicName != events.TopicMain || row.supplierID != supplierID {
		t.Fatalf("unexpected outbox metadata: %+v", row)
	}
	var gotEv events.InventoryLotReleasedEvent
	if err := json.Unmarshal(row.payload, &gotEv); err != nil {
		t.Fatalf("unmarshal OutboxEvents payload: %v", err)
	}
	if gotEv.Type != events.EventInventoryLotReleased || gotEv.OrderID != orderID || gotEv.SupplierID != supplierID || gotEv.WarehouseID != warehouseID {
		t.Fatalf("unexpected InventoryLotReleasedEvent: %+v", gotEv)
	}
	wantLots := []events.ReleasedLotQuantity{
		{LotID: lotA, Quantity: 2},
		{LotID: lotB, Quantity: 5},
	}
	if !reflect.DeepEqual(gotEv.ReleasedLots, wantLots) {
		t.Fatalf("ReleasedLots = %+v, want %+v", gotEv.ReleasedLots, wantLots)
	}
}

func TestSpannerReleaseReservationsForOrderInTxn_NoOutboxWhenNoLotReservations(t *testing.T) {
	ctx := context.Background()
	client := requireLotReleaseSpannerClient(t, ctx)
	defer client.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	orderID := "ord-t1-nolot-" + suffix
	supplierID := "sup-t1-nolot-" + suffix
	warehouseID := "wh-t1-nolot-" + suffix
	productID := "sku-t1-nolot-" + suffix

	seedOrderLotReleaseFixture(t, ctx, client, orderID, supplierID, warehouseID, productID, 4, nil)

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, orderID, OrderSourceManual, []LineItem{{SKU: productID, Quantity: 4}})
	})
	if err != nil {
		t.Fatalf("ReleaseReservationsForOrderInTxn: %v", err)
	}

	if got := readSupplierInventoryReservedForTest(t, ctx, client, supplierID, warehouseID, productID); got != 0 {
		t.Fatalf("SupplierInventoryV2 QuantityReserved = %d, want 0", got)
	}
	if rows := readOutboxEventsByOrderID(t, ctx, client, orderID); len(rows) != 0 {
		t.Fatalf("expected 0 OutboxEvents when no lot reservations exist, got %d", len(rows))
	}
}

func TestSpannerReleaseReservationsForOrderInTxn_RollbackLeavesNoOutboxOrLotMutation(t *testing.T) {
	ctx := context.Background()
	client := requireLotReleaseSpannerClient(t, ctx)
	defer client.Close()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	orderID := "ord-t1-abort-" + suffix
	supplierID := "sup-t1-abort-" + suffix
	warehouseID := "wh-t1-abort-" + suffix
	productID := "sku-t1-abort-" + suffix
	lotID := "lot-abort-" + suffix

	seedOrderLotReleaseFixture(t, ctx, client, orderID, supplierID, warehouseID, productID, 5, []lotReleaseSeed{
		{lotID: lotID, lotReserved: 5, orderReservation: 5},
	})

	errAbort := errors.New("intentional abort to verify atomicity")
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := ReleaseReservationsForOrderInTxn(ctx, txn, supplierID, warehouseID, orderID, OrderSourceManual, []LineItem{{SKU: productID, Quantity: 5}}); err != nil {
			return err
		}
		return errAbort
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}

	if got := readStockLotReservedForTest(t, ctx, client, lotID); got != 5 {
		t.Fatalf("StockLots QuantityReserved after abort = %d, want 5", got)
	}
	if got := readSupplierInventoryReservedForTest(t, ctx, client, supplierID, warehouseID, productID); got != 5 {
		t.Fatalf("SupplierInventoryV2 QuantityReserved after abort = %d, want 5", got)
	}
	if got := countOrderLotReservationsForTest(t, ctx, client, orderID); got != 1 {
		t.Fatalf("OrderLotReservations count after abort = %d, want 1", got)
	}
	if rows := readOutboxEventsByOrderID(t, ctx, client, orderID); len(rows) != 0 {
		t.Fatalf("OutboxEvents count after abort = %d, want 0", len(rows))
	}
}
