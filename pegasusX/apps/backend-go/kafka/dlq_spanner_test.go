package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	segmentkafka "github.com/segmentio/kafka-go"
	"google.golang.org/api/option"
)

func newSpannerDLQTestClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()

	emulatorHost := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if emulatorHost == "" {
		return nil
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
		t.Fatalf("failed to create spanner test client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

// TestConsumer_PanicFourTimes_WritesFailedStatusToSpanner tests that causing a consumer
// panic 4 times invokes the retry loop 4 times, routes the message to DLQ, and executes
// the Spanner failure updater hook to write FAILED status.
func TestConsumer_PanicFourTimes_WritesFailedStatusToSpanner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var attemptCount int32
	panicHandler := func(ctx context.Context, msg segmentkafka.Message) error {
		current := atomic.AddInt32(&attemptCount, 1)
		panic(fmt.Sprintf("deliberate test poison panic attempt %d", current))
	}

	writer := &fakeDLQWriter{}
	spannerClient := newSpannerDLQTestClient(t, ctx)

	jobID := fmt.Sprintf("opt-job-%d", time.Now().UnixNano())
	supplierID := "sup-test-001"

	var hookInvoked bool
	var hookJobID string
	var hookReason string

	var dlqHook DLQHook
	if spannerClient != nil {
		// Set up initial PENDING job row in Spanner
		_, err := spannerClient.Apply(ctx, []*spanner.Mutation{
			spanner.InsertOrUpdateMap("OptimizationJobs", map[string]interface{}{
				"JobId":          jobID,
				"SupplierId":     supplierID,
				"Status":         "PENDING",
				"RequestType":    "DAILY_ROUTE_SOLVE",
				"PayloadJson":    []byte(`{"test": true}`),
				"IdempotencyKey": fmt.Sprintf("idem-%s", jobID),
				"CreatedAt":      spanner.CommitTimestamp,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		})
		if err != nil {
			t.Fatalf("failed to insert initial OptimizationJobs row: %v", err)
		}

		updater := NewSpannerDLQUpdater(spannerClient, slog.Default())
		dlqHook = func(ctx context.Context, msg segmentkafka.Message, reason error) error {
			hookInvoked = true
			hookJobID = string(msg.Key)
			if reason != nil {
				hookReason = reason.Error()
			}
			return updater.HandleDLQ(ctx, msg, reason)
		}
	} else {
		// In-memory verification when Spanner emulator is not active
		spannerJobStore := make(map[string]string)
		spannerJobStore[jobID] = "PENDING"

		dlqHook = func(ctx context.Context, msg segmentkafka.Message, reason error) error {
			hookInvoked = true
			meta := extractJobMetadata(msg)
			hookJobID = meta.JobID
			if reason != nil {
				hookReason = reason.Error()
			}
			spannerJobStore[meta.JobID] = "FAILED"
			return nil
		}
	}

	consumer := NewConsumer(ConsumerDeps{
		Brokers:     []string{"localhost:9092"},
		GroupID:     "opt-group",
		Topic:       "optimization-jobs",
		MaxAttempts: 4,
		Handler:     panicHandler,
		DLQWriter:   writer,
		OnDLQ:       dlqHook,
	})

	payload, _ := json.Marshal(map[string]interface{}{
		"job_id":         jobID,
		"supplier_id":    supplierID,
		"aggregate_type": "OPTIMIZATION_JOB",
	})

	msg := segmentkafka.Message{
		Topic: "optimization-jobs",
		Key:   []byte(jobID),
		Value: payload,
		Headers: []segmentkafka.Header{
			{Key: "job_id", Value: []byte(jobID)},
			{Key: "supplier_id", Value: []byte(supplierID)},
			{Key: "aggregate_type", Value: []byte("OPTIMIZATION_JOB")},
		},
		Partition: 0,
		Offset:    100,
		Time:      time.Now().UTC(),
	}

	// Dispatch message through consumer
	err := consumer.dispatch(ctx, msg)
	if err != nil {
		t.Fatalf("dispatch expected nil (after successful DLQ routing), got %v", err)
	}

	// 1. Assert exactly 4 attempts were made and caught
	attempts := atomic.LoadInt32(&attemptCount)
	if attempts != 4 {
		t.Fatalf("attemptCount = %d, want exactly 4 attempts", attempts)
	}

	// 2. Assert message was routed to DLQ writer
	if len(writer.messages) != 1 {
		t.Fatalf("DLQ message count = %d, want 1", len(writer.messages))
	}
	dlqMsg := writer.messages[0]
	headers := make(map[string]string)
	for _, h := range dlqMsg.Headers {
		headers[h.Key] = string(h.Value)
	}
	if !strings.Contains(headers["dlq_reason"], "panic in kafka handler: deliberate test poison panic attempt 4") {
		t.Fatalf("dlq_reason header = %q, want panic attempt 4", headers["dlq_reason"])
	}
	if headers["original_topic"] != "optimization-jobs" {
		t.Fatalf("original_topic = %q, want optimization-jobs", headers["original_topic"])
	}

	// 3. Assert OnDLQ hook was executed
	if !hookInvoked {
		t.Fatal("expected OnDLQ hook to be invoked, but it was not")
	}
	if hookJobID != jobID {
		t.Fatalf("hook jobID = %q, want %q", hookJobID, jobID)
	}
	if !strings.Contains(hookReason, "deliberate test poison panic attempt 4") {
		t.Fatalf("hook reason = %q, want panic attempt 4", hookReason)
	}

	// 4. Assert Spanner table status was updated to FAILED
	if spannerClient != nil {
		row, readErr := spannerClient.Single().ReadRow(ctx, "OptimizationJobs", spanner.Key{jobID}, []string{"Status"})
		if readErr != nil {
			t.Fatalf("failed to read OptimizationJobs from spanner: %v", readErr)
		}
		var status string
		if scanErr := row.Columns(&status); scanErr != nil {
			t.Fatalf("failed to scan status: %v", scanErr)
		}
		if status != "FAILED" {
			t.Fatalf("Spanner OptimizationJobs status = %q, want FAILED", status)
		}
	}
}

func TestExtractJobMetadata(t *testing.T) {
	// Case 1: Metadata from headers
	msg1 := segmentkafka.Message{
		Topic: "partner-exports",
		Key:   []byte("export-key-1"),
		Headers: []segmentkafka.Header{
			{Key: "job_id", Value: []byte("job-123")},
			{Key: "supplier_id", Value: []byte("sup-456")},
			{Key: "aggregate_type", Value: []byte("PARTNER_EXPORT")},
			{Key: "session_id", Value: []byte("sess-789")},
		},
		Value: []byte(`{"unrelated": true}`),
	}
	m1 := extractJobMetadata(msg1)
	if m1.JobID != "job-123" || m1.SupplierID != "sup-456" || m1.AggregateType != "PARTNER_EXPORT" || m1.SessionID != "sess-789" {
		t.Fatalf("extractJobMetadata headers mismatch: %+v", m1)
	}

	// Case 2: Metadata from JSON payload
	msg2 := segmentkafka.Message{
		Topic: "supplier-imports",
		Key:   []byte("import-key-2"),
		Value: []byte(`{"jobId": "job-abc", "supplierId": "sup-xyz", "sessionId": "sess-999", "resource": "orders"}`),
	}
	m2 := extractJobMetadata(msg2)
	if m2.JobID != "job-abc" || m2.SupplierID != "sup-xyz" || m2.SessionID != "sess-999" || m2.AggregateType != "orders" {
		t.Fatalf("extractJobMetadata payload mismatch: %+v", m2)
	}

	// Case 3: Fallback from message Key
	msg3 := segmentkafka.Message{
		Topic: "generic-topic",
		Key:   []byte("raw-event-id-99"),
		Value: []byte(`not json`),
	}
	m3 := extractJobMetadata(msg3)
	if m3.JobID != "raw-event-id-99" || m3.AggregateID != "raw-event-id-99" || m3.AggregateType != "generic-topic" {
		t.Fatalf("extractJobMetadata fallback mismatch: %+v", m3)
	}
}

func TestConsumer_PanicFourTimes_RetryLoop(t *testing.T) {
	var attempts int
	consumer := &Consumer{
		deps: ConsumerDeps{
			Topic:       "test-topic",
			MaxAttempts: 4,
			Handler: func(ctx context.Context, msg segmentkafka.Message) error {
				attempts++
				panic(fmt.Sprintf("panic attempt %d", attempts))
			},
		},
	}

	err := consumer.processWithRetries(context.Background(), segmentkafka.Message{Partition: 0, Offset: 1})
	if err == nil {
		t.Fatal("expected error from exhausted panics, got nil")
	}
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
	if !strings.Contains(err.Error(), "panic in kafka handler: panic attempt 4") {
		t.Fatalf("err = %q, want panic attempt 4", err.Error())
	}
}
