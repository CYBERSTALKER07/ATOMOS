package treasury

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const defaultSpannerIntegrationDatabase = "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db"

func TestHandleInvoiceStatusOverride_DisputedTransitionWritesOutboxEvent_Integration(t *testing.T) {
	if os.Getenv("RUN_SPANNER_INTEGRATION") != "1" {
		t.Skip("set RUN_SPANNER_INTEGRATION=1 to run Spanner-backed treasury integration tests")
	}
	ensureSpannerEmulatorReachable(t)

	ctx := context.Background()
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(connectCtx, spannerIntegrationDatabase())
	if err != nil {
		t.Fatalf("spanner.NewClient() error = %v", err)
	}
	defer client.Close()

	orderID := uuid.NewString()
	invoiceID := uuid.NewString()
	sessionID := uuid.NewString()
	supplierID := uuid.NewString()
	retailerID := uuid.NewString()
	driverID := uuid.NewString()
	actorID := uuid.NewString()
	reason := "integration-test-dispute"

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Orders",
			[]string{"OrderId", "RetailerId", "SupplierId", "DriverId", "State"},
			[]interface{}{orderID, retailerID, supplierID, driverID, "ARRIVED"},
		),
		spanner.Insert("MasterInvoices",
			[]string{"InvoiceId", "RetailerId", "Total", "State", "OrderId", "CustodyStatus"},
			[]interface{}{invoiceID, retailerID, int64(125000), "PENDING", orderID, "PENDING"},
		),
		spanner.Insert("DeliverySessions",
			[]string{"SessionId", "OrderId", "RetailerId", "DriverId", "SupplierId", "State", "CreatedAt"},
			[]interface{}{sessionID, orderID, retailerID, driverID, supplierID, "SETTLEMENT_AWAIT", spanner.CommitTimestamp},
		),
	}); err != nil {
		t.Fatalf("seed test rows: %v", err)
	}

	var outboxEventID string
	t.Cleanup(func() {
		var cleanupMutations []*spanner.Mutation
		cleanupMutations = append(cleanupMutations,
			spanner.Delete("MasterInvoices", spanner.Key{invoiceID}),
			spanner.Delete("DeliverySessions", spanner.Key{sessionID}),
			spanner.Delete("Orders", spanner.Key{orderID}),
		)
		if outboxEventID != "" {
			cleanupMutations = append(cleanupMutations, spanner.Delete("OutboxEvents", spanner.Key{outboxEventID}))
		}
		_, _ = client.Apply(context.Background(), cleanupMutations)
	})

	reqBody, err := json.Marshal(InvoiceStatusRequest{
		InvoiceID: invoiceID,
		Status:    "DISPUTED",
		Reason:    reason,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/v1/treasury/invoice/status", bytes.NewReader(reqBody))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.PegasusClaims{
		UserID:       actorID,
		Role:         "SUPPLIER",
		SupplierID:   supplierID,
		SupplierRole: "GLOBAL_ADMIN",
	}))
	rec := httptest.NewRecorder()

	HandleInvoiceStatusOverride(client)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	invoiceRow, err := client.Single().ReadRow(ctx, "MasterInvoices", spanner.Key{invoiceID}, []string{"CustodyStatus"})
	if err != nil {
		t.Fatalf("read invoice after override: %v", err)
	}
	var custodyStatus spanner.NullString
	if err := invoiceRow.Columns(&custodyStatus); err != nil {
		t.Fatalf("decode invoice custody status: %v", err)
	}
	if got := custodyStatus.StringVal; got != "DISPUTED" {
		t.Fatalf("MasterInvoices.CustodyStatus = %q, want %q", got, "DISPUTED")
	}

	outboxStmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, EventType, TopicName, Payload
		      FROM OutboxEvents
		      WHERE AggregateId = @aggregateId AND EventType = @eventType
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"aggregateId": sessionID,
			"eventType":   kafkaEvents.EventDeliveryDisputed,
		},
	}
	iter := client.Single().Query(ctx, outboxStmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		t.Fatalf("expected outbox event %s for aggregate %s, found none", kafkaEvents.EventDeliveryDisputed, sessionID)
	}
	if err != nil {
		t.Fatalf("query outbox events: %v", err)
	}

	var aggregateType string
	var aggregateID string
	var eventType string
	var topicName string
	var payload []byte
	if err := row.Columns(&outboxEventID, &aggregateType, &aggregateID, &eventType, &topicName, &payload); err != nil {
		t.Fatalf("decode outbox row: %v", err)
	}

	if aggregateType != "DeliverySession" {
		t.Fatalf("AggregateType = %q, want %q", aggregateType, "DeliverySession")
	}
	if aggregateID != sessionID {
		t.Fatalf("AggregateId = %q, want %q", aggregateID, sessionID)
	}
	if eventType != kafkaEvents.EventDeliveryDisputed {
		t.Fatalf("EventType = %q, want %q", eventType, kafkaEvents.EventDeliveryDisputed)
	}
	if topicName != kafkaEvents.TopicMain {
		t.Fatalf("TopicName = %q, want %q", topicName, kafkaEvents.TopicMain)
	}

	var disputedEvent kafkaEvents.DeliveryDisputedEvent
	if err := json.Unmarshal(payload, &disputedEvent); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}

	if disputedEvent.OrderID != orderID {
		t.Fatalf("payload.order_id = %q, want %q", disputedEvent.OrderID, orderID)
	}
	if disputedEvent.SessionID != sessionID {
		t.Fatalf("payload.session_id = %q, want %q", disputedEvent.SessionID, sessionID)
	}
	if disputedEvent.RetailerID != retailerID {
		t.Fatalf("payload.retailer_id = %q, want %q", disputedEvent.RetailerID, retailerID)
	}
	if disputedEvent.DriverID != driverID {
		t.Fatalf("payload.driver_id = %q, want %q", disputedEvent.DriverID, driverID)
	}
	if disputedEvent.SupplierID != supplierID {
		t.Fatalf("payload.supplier_id = %q, want %q", disputedEvent.SupplierID, supplierID)
	}
	if disputedEvent.Reason != reason {
		t.Fatalf("payload.reason = %q, want %q", disputedEvent.Reason, reason)
	}
	if disputedEvent.DisputedBy != actorID {
		t.Fatalf("payload.disputed_by = %q, want %q", disputedEvent.DisputedBy, actorID)
	}
	if disputedEvent.Timestamp.IsZero() {
		t.Fatal("payload.timestamp should be set")
	}
}

func spannerIntegrationDatabase() string {
	if direct := os.Getenv("SPANNER_DATABASE_URI"); direct != "" {
		return direct
	}
	project := os.Getenv("SPANNER_PROJECT")
	instance := os.Getenv("SPANNER_INSTANCE")
	database := os.Getenv("SPANNER_DATABASE")
	if project != "" && instance != "" && database != "" {
		return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	}
	return defaultSpannerIntegrationDatabase
}

func ensureSpannerEmulatorReachable(t *testing.T) {
	t.Helper()
	host := os.Getenv("SPANNER_EMULATOR_HOST")
	if host == "" {
		host = "localhost:9010"
		_ = os.Setenv("SPANNER_EMULATOR_HOST", host)
	}

	conn, err := net.DialTimeout("tcp", host, 1500*time.Millisecond)
	if err != nil {
		t.Skipf("spanner emulator is not reachable at %s: %v", host, err)
		return
	}
	_ = conn.Close()
}
