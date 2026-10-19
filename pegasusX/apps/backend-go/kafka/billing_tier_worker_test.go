package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/services/billing"
	segkafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/option"
)

func TestBillingTierWorker_NilSafety(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var nilWorker *BillingTierWorker
	if err := nilWorker.HandleMessage(ctx, []byte(`{}`)); err != nil {
		t.Fatalf("nilWorker.HandleMessage failed: %v", err)
	}

	workerWithNilMeter := NewBillingTierWorker(nil)
	if err := workerWithNilMeter.HandleMessage(ctx, []byte(`{}`)); err != nil {
		t.Fatalf("workerWithNilMeter.HandleMessage failed: %v", err)
	}
}

func TestBillingTierWorker_IgnoresNonOrderFinalized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	worker := NewBillingTierWorker(billing.NewMeterWorker(nil))

	msg := []byte(`{"type":"ORDER_CREATED","order_id":"ord_1","amount_minor":5000}`)
	if err := worker.HandleMessage(ctx, msg); err != nil {
		t.Fatalf("HandleMessage should ignore non-finalized event, got error: %v", err)
	}

	msg2 := []byte(`{"type":"ORDER_STATUS_CHANGED","order_id":"ord_2","amount_minor":5000}`)
	if err := worker.HandleMessage(ctx, msg2); err != nil {
		t.Fatalf("HandleMessage should ignore non-finalized event, got error: %v", err)
	}
}

func TestBillingTierWorker_RejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	worker := NewBillingTierWorker(billing.NewMeterWorker(nil))

	msg := []byte(`{invalid-json`)
	if err := worker.HandleMessage(ctx, msg); err == nil {
		t.Fatal("HandleMessage with malformed JSON expected error, got nil")
	}
}

func newSpannerTestClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()

	emulatorHost := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if emulatorHost == "" {
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping integration test")
	}

	project := "pegasusx-local"
	if p := os.Getenv("SPANNER_PROJECT"); p != "" {
		project = p
	}
	instance := "pegasusx-instance"
	if i := os.Getenv("SPANNER_INSTANCE"); i != "" {
		instance = i
	}
	database := "pegasusx-db"
	if d := os.Getenv("SPANNER_DATABASE"); d != "" {
		database = d
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create spanner client: %v", err)
	}
	return client
}

func TestBillingTierWorker_HandleEvent_AmountExtraction(t *testing.T) {
	ctx := context.Background()
	client := newSpannerTestClient(t, ctx)
	defer client.Close()

	meterWorker := billing.NewMeterWorker(client)
	worker := NewBillingTierWorker(meterWorker)

	testCases := []struct {
		name        string
		payloadJSON string
		orderID     string
		supplierID  string
		wantMinor   int64
	}{
		{
			name: "amount_minor primary field",
			payloadJSON: `{
				"type": "ORDER_FINALIZED",
				"order_id": "%s",
				"supplier_id": "%s",
				"amount_minor": 2500000
			}`,
			orderID:    "ord_bt_1_" + uuid.NewString(),
			supplierID: "sup_bt_1_" + uuid.NewString(),
			wantMinor:  2500000,
		},
		{
			name: "total_minor fallback field",
			payloadJSON: `{
				"type": "ORDER_FINALIZED",
				"order_id": "%s",
				"supplier_id": "%s",
				"total_minor": 1750000
			}`,
			orderID:    "ord_bt_2_" + uuid.NewString(),
			supplierID: "sup_bt_2_" + uuid.NewString(),
			wantMinor:  1750000,
		},
		{
			name: "total.amount nested object field",
			payloadJSON: `{
				"type": "ORDER_FINALIZED",
				"order_id": "%s",
				"supplier_id": "%s",
				"total": {
					"amount": 3200000,
					"currency": "UZS"
				}
			}`,
			orderID:    "ord_bt_3_" + uuid.NewString(),
			supplierID: "sup_bt_3_" + uuid.NewString(),
			wantMinor:  3200000,
		},
		{
			name: "legacy float amount converted to minor",
			payloadJSON: `{
				"type": "ORDER_FINALIZED",
				"order_id": "%s",
				"supplier_id": "%s",
				"amount": 450.50
			}`,
			orderID:    "ord_bt_4_" + uuid.NewString(),
			supplierID: "sup_bt_4_" + uuid.NewString(),
			wantMinor:  45050,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rawJSON := fmt.Sprintf(tc.payloadJSON, tc.orderID, tc.supplierID)
			msg := segkafka.Message{Value: []byte(rawJSON)}

			if err := worker.HandleEvent(ctx, msg); err != nil {
				t.Fatalf("HandleEvent failed: %v", err)
			}

			// Verify in Spanner BillingSupplierMeters
			row, err := client.Single().ReadRow(ctx, "BillingSupplierMeters", spanner.Key{tc.supplierID, int64(0)}, []string{"CurrentValue"})
			if err != nil {
				t.Fatalf("read BillingSupplierMeters failed: %v", err)
			}
			var val int64
			if err := row.Column(0, &val); err != nil {
				t.Fatalf("decode CurrentValue failed: %v", err)
			}
			if val != tc.wantMinor {
				t.Fatalf("CurrentValue = %d, want %d", val, tc.wantMinor)
			}
		})
	}
}
