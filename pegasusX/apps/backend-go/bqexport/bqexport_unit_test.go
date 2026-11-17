package bqexport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	kgo "github.com/segmentio/kafka-go"
)

func TestMapKafkaMessage_DomainOntologyAndGL(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)

	ontoPayload, err := json.Marshal(events.OntologyObjectChangedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventOntologyObjectChanged,
			Version:   1,
			Timestamp: now.Format(time.RFC3339Nano),
		},
		CellID:           "uz",
		ObjectType:       "Order",
		ObjectID:         "ord-42",
		SupplierID:       "sup-1",
		Version:          7,
		ChangeKind:       "UPSERT",
		SourceEventType:  events.EventOrderCreated,
		SourceEventID:    "evt-src-42",
		SecurityMarkings: []string{"MARKING:FINANCE_ONLY"},
		UpdatedAt:        now.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}

	evID, rows, err := MapKafkaMessage(kgo.Message{
		Topic:     events.TopicIntel,
		Partition: 2,
		Offset:    105,
		Key:       []byte("ord-42"),
		Value:     ontoPayload,
		Time:      now,
		Headers:   []kgo.Header{{Key: "event_id", Value: []byte("evt-onto-105")}},
	}, "uz")
	if err != nil {
		t.Fatalf("MapKafkaMessage ontology failed: %v", err)
	}
	if evID != "evt-onto-105" {
		t.Fatalf("eventID = %q, want evt-onto-105", evID)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (domain_events + ontology_objects), got %d", len(rows))
	}
	if rows[0].Table != TableRawDomainEvents || rows[1].Table != TableRawOntologyObjects {
		t.Fatalf("unexpected tables: %s, %s", rows[0].Table, rows[1].Table)
	}
	if rows[1].InsertID != "onto:uz:Order:ord-42:7" {
		t.Fatalf("unexpected ontology insertID: %q", rows[1].InsertID)
	}

	glPayload, err := json.Marshal(events.GLEntryPostedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventGLEntryPosted,
			Version:   1,
			Timestamp: now.Format(time.RFC3339Nano),
		},
		EntryID:          "je-99",
		EntityID:         "sup-1",
		PeriodID:         "2026-11",
		SourceType:       "ORDER",
		SourceID:         "ord-42",
		Currency:         "UZS",
		TotalDebitMinor:  250000,
		TotalCreditMinor: 250000,
		Hash:             "abc123hash",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, glRows, err := MapKafkaMessage(kgo.Message{
		Topic:     events.TopicLedger,
		Partition: 0,
		Offset:    11,
		Value:     glPayload,
		Time:      now,
		Headers:   []kgo.Header{{Key: "event_id", Value: []byte("evt-gl-11")}},
	}, "uz")
	if err != nil {
		t.Fatalf("MapKafkaMessage gl failed: %v", err)
	}
	if len(glRows) != 2 {
		t.Fatalf("expected 2 rows (domain_events + gl_entries), got %d", len(glRows))
	}
	if glRows[1].Table != TableCuratedGLEntries {
		t.Fatalf("expected curated.gl_entries, got %q", glRows[1].Table)
	}
	if glRows[1].Data["total_debit_minor"] != int64(250000) || glRows[1].Data["total_credit_minor"] != int64(250000) {
		t.Fatalf("unexpected GL minor amounts: %+v", glRows[1].Data)
	}
}

func TestMapKafkaMessage_RejectsMalformedJSON(t *testing.T) {
	t.Parallel()
	_, _, err := MapKafkaMessage(kgo.Message{
		Topic:     events.TopicMain,
		Partition: 0,
		Offset:    1,
		Value:     []byte("{not-json"),
	}, "uz")
	if !errors.Is(err, ErrMalformedEventPayload) {
		t.Fatalf("expected ErrMalformedEventPayload, got %v", err)
	}
}

func TestBigQuerySink_StreamingInsertHealthAndInsertErrors(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	var (
		mu          sync.Mutex
		requests    []string
		failNextRow bool
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/datasets/") {
			_, _ = w.Write([]byte(`{"kind":"bigquery#dataset","id":"pegasusx-test:uz_raw"}`))
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/insertAll") {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			requests = append(requests, r.URL.Path+":"+string(body))
			fail := failNextRow
			mu.Unlock()
			if fail {
				_, _ = w.Write([]byte(`{"kind":"bigquery#tableDataInsertAllResponse","insertErrors":[{"index":0,"errors":[{"reason":"invalid","message":"schema mismatch"}]}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"kind":"bigquery#tableDataInsertAllResponse"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	sink, err := NewBigQuerySink(context.Background(), BigQueryConfig{
		ProjectID:     "pegasusx-test",
		DatasetPrefix: "uz",
		Endpoint:      srv.URL,
		MaxBatchRows:  2,
	})
	if err != nil {
		t.Fatalf("NewBigQuerySink: %v", err)
	}

	status, err := sink.Health(context.Background())
	if err != nil || !status.Healthy {
		t.Fatalf("Health failed: status=%+v err=%v", status, err)
	}

	rows := []Row{
		{Table: TableRawDomainEvents, InsertID: "e1", Data: map[string]any{"event_id": "e1"}},
		{Table: TableRawDomainEvents, InsertID: "e2", Data: map[string]any{"event_id": "e2"}},
		{Table: TableRawDomainEvents, InsertID: "e3", Data: map[string]any{"event_id": "e3"}},
		{Table: TableCuratedGLEntries, InsertID: "g1", Data: map[string]any{"entry_id": "g1"}},
	}
	if err := sink.WriteBatch(context.Background(), rows); err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}

	mu.Lock()
	reqCount := len(requests)
	failNextRow = true
	mu.Unlock()
	if reqCount != 3 {
		t.Fatalf("expected 3 chunked insertAll HTTP calls (2+1 raw.domain_events + 1 curated.gl_entries), got %d", reqCount)
	}

	err = sink.WriteBatch(context.Background(), []Row{rows[0]})
	if !errors.Is(err, ErrInsertRowsRejected) {
		t.Fatalf("expected ErrInsertRowsRejected on insertErrors response, got %v", err)
	}
}

func TestSinkFactory_FailClosedInProduction(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("BIGQUERY_PROJECT_ID", "")
	t.Setenv("BIGQUERY_LOCAL_SPOOL_DIR", t.TempDir())
	t.Setenv("BIGQUERY_ENDPOINT", "")

	if _, err := NewSinkFromEnv(context.Background()); !errors.Is(err, ErrBigQueryConfigRequired) {
		t.Fatalf("expected ErrBigQueryConfigRequired without project in production, got %v", err)
	}
	if _, err := NewJSONLSpoolSink(t.TempDir()); !errors.Is(err, ErrLocalSpoolDisallowedInProduction) {
		t.Fatalf("expected ErrLocalSpoolDisallowedInProduction in production, got %v", err)
	}

	t.Setenv("BIGQUERY_PROJECT_ID", "pegasusx-prod-uz")
	t.Setenv("BIGQUERY_ENDPOINT", "http://127.0.0.1:9999")
	if _, err := NewSinkFromEnv(context.Background()); !errors.Is(err, ErrBigQueryConfigRequired) {
		t.Fatalf("expected ErrBigQueryConfigRequired when custom endpoint used in production, got %v", err)
	}

	t.Setenv("BIGQUERY_ENDPOINT", "")
	t.Setenv("BIGQUERY_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing-creds.json"))
	if _, err := NewSinkFromEnv(context.Background()); !errors.Is(err, ErrBigQueryConfigRequired) {
		t.Fatalf("expected ErrBigQueryConfigRequired when credentials missing in production, got %v", err)
	}
}

func TestJSONLSpoolSink_DeduplicatesAcrossRestart(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	dir := t.TempDir()

	s1, err := NewJSONLSpoolSink(dir)
	if err != nil {
		t.Fatal(err)
	}
	status, err := s1.Health(context.Background())
	if err != nil || !status.Healthy {
		t.Fatalf("Health failed: %+v err=%v", status, err)
	}

	row := Row{
		Table:    TableRawDomainEvents,
		InsertID: "evt:dup-1",
		Data:     map[string]any{"event_id": "dup-1", "total_minor": int64(500)},
	}
	if err := s1.WriteBatch(context.Background(), []Row{row, row}); err != nil {
		t.Fatal(err)
	}

	// Simulate process restart against the same spool directory.
	s2, err := NewJSONLSpoolSink(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s2.WriteBatch(context.Background(), []Row{row}); err != nil {
		t.Fatal(err)
	}

	rawBytes, err := os.ReadFile(filepath.Join(dir, "raw.domain_events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(rawBytes)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 deduplicated JSONL line, got %d (%s)", len(lines), string(rawBytes))
	}
}

func TestExporter_IdempotentCheckpointingAndFlushBatch(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "local")
	dir := t.TempDir()
	sink, err := NewJSONLSpoolSink(dir)
	if err != nil {
		t.Fatal(err)
	}
	cpStore := NewMemoryCheckpointStore()
	exp, err := NewExporter(sink, cpStore, "uz", nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	msg0 := kgo.Message{
		Topic:     events.TopicOrders,
		Partition: 0,
		Offset:    0,
		Time:      now,
		Value:     []byte(`{"type":"ORDER_CREATED","event_id":"e-0","supplier_id":"s1"}`),
	}
	msg1 := kgo.Message{
		Topic:     events.TopicLedger,
		Partition: 1,
		Offset:    5,
		Time:      now,
		Value:     []byte(`{"type":"gl.entry.posted","event_id":"e-1","entity_id":"s1","entry_id":"je-1","period_id":"2026-11","source_type":"ORDER","source_id":"o1","currency":"UZS","total_debit_minor":1000,"total_credit_minor":1000,"hash":"h1"}`),
	}

	// Offset 0 must work properly (not skipped as uninitialized 0).
	if err := exp.HandleMessage(context.Background(), msg0); err != nil {
		t.Fatalf("HandleMessage offset 0 failed: %v", err)
	}
	// Replay offset 0 must be idempotently skipped.
	if err := exp.HandleMessage(context.Background(), msg0); err != nil {
		t.Fatalf("replay HandleMessage failed: %v", err)
	}

	if err := exp.FlushBatch(context.Background(), []kgo.Message{msg0, msg1}); err != nil {
		t.Fatalf("FlushBatch failed: %v", err)
	}

	cp0, ok, err := cpStore.GetCheckpoint(context.Background(), DefaultSinkID, events.TopicOrders, 0)
	if err != nil || !ok || cp0.OffsetVal != 0 || cp0.RowsExported != 1 {
		t.Fatalf("unexpected cp0: ok=%v cp=%+v err=%v", ok, cp0, err)
	}
	cp1, ok, err := cpStore.GetCheckpoint(context.Background(), DefaultSinkID, events.TopicLedger, 1)
	if err != nil || !ok || cp1.OffsetVal != 5 || cp1.RowsExported != 2 {
		t.Fatalf("unexpected cp1: ok=%v cp=%+v err=%v", ok, cp1, err)
	}

	topics := DefaultSinkTopics()
	for _, want := range []string{events.TopicMain, events.TopicLedger, events.TopicIntel, events.TopicConnectors} {
		if !slices.Contains(topics, want) {
			t.Fatalf("DefaultSinkTopics missing %q: %v", want, topics)
		}
	}
	if slices.Contains(topics, events.TopicWebhooks) || slices.Contains(topics, events.TopicDeadLetter) {
		t.Fatalf("DefaultSinkTopics must exclude retired/DLQ topics: %v", topics)
	}
}

func TestNoSingleDotQueryInBQExport(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), ".Single().Query") {
			t.Fatalf("%s must not contain .Single().Query (use StrongRead() ReadOnlyTransaction)", entry.Name())
		}
	}
}
