package bqexport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	kgo "github.com/segmentio/kafka-go"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openBQExportSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner bqexport test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner emulator connect: %v", err)
		}
		t.Skipf("spanner emulator unavailable: %v", err)
	}
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	_, probeErr := ro.ReadRow(ctx, checkpointsTable, spanner.Key{"__probe__", "__probe__", int64(0)}, []string{"SinkId"})
	ro.Close()
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("%s table unavailable (%s): %v", checkpointsTable, dbPath, probeErr)
		}
		t.Skipf("%s table unavailable (%s): %v", checkpointsTable, dbPath, probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestSpannerCheckpointStore_StrongReadAndMonotonicAdvance(t *testing.T) {
	client := openBQExportSpannerTestClient(t)
	ctx := context.Background()

	store, err := NewSpannerCheckpointStore(client)
	if err != nil {
		t.Fatal(err)
	}

	sinkID := fmt.Sprintf("test-bq-sink-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete(checkpointsTable, spanner.KeyRange{
				Start: spanner.Key{sinkID},
				End:   spanner.Key{sinkID},
				Kind:  spanner.ClosedClosed,
			}),
		})
	})

	if _, found, err := store.GetCheckpoint(ctx, sinkID, events.TopicIntel, 0); err != nil || found {
		t.Fatalf("expected not found before first save, found=%v err=%v", found, err)
	}

	if err := store.SaveCheckpoint(ctx, Checkpoint{
		SinkID:       sinkID,
		Topic:        events.TopicIntel,
		PartitionNum: 0,
		OffsetVal:    0,
		LastEventID:  "evt-0",
		RowsDelta:    2,
	}); err != nil {
		t.Fatalf("SaveCheckpoint(0): %v", err)
	}

	// Monotonic guard: regression or replay at offset 0 must not overwrite or double-count.
	if err := store.SaveCheckpoint(ctx, Checkpoint{
		SinkID:       sinkID,
		Topic:        events.TopicIntel,
		PartitionNum: 0,
		OffsetVal:    0,
		LastEventID:  "evt-replay",
		RowsDelta:    99,
	}); err != nil {
		t.Fatalf("SaveCheckpoint replay(0): %v", err)
	}

	if err := store.SaveCheckpoint(ctx, Checkpoint{
		SinkID:       sinkID,
		Topic:        events.TopicIntel,
		PartitionNum: 0,
		OffsetVal:    12,
		LastEventID:  "evt-12",
		RowsDelta:    3,
	}); err != nil {
		t.Fatalf("SaveCheckpoint(12): %v", err)
	}

	cp, found, err := store.GetCheckpoint(ctx, sinkID, events.TopicIntel, 0)
	if err != nil || !found {
		t.Fatalf("GetCheckpoint failed: found=%v err=%v", found, err)
	}
	if cp.OffsetVal != 12 || cp.LastEventID != "evt-12" || cp.RowsExported != 5 {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}

	list, err := store.ListCheckpoints(ctx, sinkID)
	if err != nil {
		t.Fatalf("ListCheckpoints: %v", err)
	}
	if len(list) != 1 || list[0].OffsetVal != 12 {
		t.Fatalf("unexpected listed checkpoints: %+v", list)
	}

	// End-to-end Exporter test with JSONL spool + live Spanner checkpoints.
	t.Setenv("PEGASUSX_ENV", "local")
	spool, err := NewJSONLSpoolSink(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exp, err := NewExporter(spool, store, "uz", nil)
	if err != nil {
		t.Fatal(err)
	}
	exp.SinkID = sinkID

	msg := kgo.Message{
		Topic:     events.TopicIntel,
		Partition: 0,
		Offset:    13,
		Time:      time.Now().UTC(),
		Value:     []byte(`{"type":"ontology.object.changed","event_id":"evt-13","cell_id":"uz","object_type":"SKU","object_id":"sku-1","supplier_id":"sup-1","version":2,"change_kind":"UPSERT","updated_at":"2026-11-17T00:00:00Z"}`),
	}
	if err := exp.HandleMessage(ctx, msg); err != nil {
		t.Fatalf("HandleMessage against live Spanner failed: %v", err)
	}
	// Replay offset 13 should skip cleanly.
	if err := exp.HandleMessage(ctx, msg); err != nil {
		t.Fatalf("HandleMessage replay failed: %v", err)
	}

	cpAfter, found, err := store.GetCheckpoint(ctx, sinkID, events.TopicIntel, 0)
	if err != nil || !found {
		t.Fatalf("GetCheckpoint after export: found=%v err=%v", found, err)
	}
	if cpAfter.OffsetVal != 13 || cpAfter.RowsExported != 7 {
		t.Fatalf("expected offset=13 rows=7, got %+v", cpAfter)
	}
}

func TestKafkaConsumer_LiveKafkaToSpoolAndSpannerCheckpoint(t *testing.T) {
	spannerClient := openBQExportSpannerTestClient(t)
	defer spannerClient.Close()

	brokersEnv := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if brokersEnv == "" {
		brokersEnv = "localhost:9092"
	}
	rawBrokers := strings.Split(brokersEnv, ",")
	var brokers []string
	for _, b := range rawBrokers {
		if b = strings.TrimSpace(b); b != "" {
			brokers = append(brokers, b)
		}
	}
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	requireKafka := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_KAFKA")) == "1"
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	dialer := &kgo.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(dialCtx, "tcp", brokers[0])
	if err != nil {
		if requireKafka {
			t.Fatalf("PARITY_REQUIRE_KAFKA=1 but kafka dial %s failed: %v", brokers[0], err)
		}
		t.Skipf("kafka broker %s unavailable (%v); skipping live Kafka consumer test", brokers[0], err)
	}
	controller, err := conn.Controller()
	if err != nil {
		_ = conn.Close()
		if requireKafka {
			t.Fatalf("kafka controller lookup failed: %v", err)
		}
		t.Skipf("kafka controller unavailable (%v); skipping live Kafka consumer test", err)
	}
	ctrlConn, err := kgo.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
	_ = conn.Close()
	if err != nil {
		if requireKafka {
			t.Fatalf("kafka controller dial failed: %v", err)
		}
		t.Skipf("kafka controller dial failed (%v); skipping live Kafka consumer test", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	topic := "bqexport-live-" + suffix
	sinkID := "void-bq-sink-live-" + suffix
	groupID := "void-bq-sink-test-" + suffix

	if err := ctrlConn.CreateTopics(kgo.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil {
		_ = ctrlConn.Close()
		t.Fatalf("CreateTopics(%s): %v", topic, err)
	}
	_ = ctrlConn.Close()

	t.Cleanup(func() {
		cleanupConn, cerr := kgo.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
		if cerr == nil {
			_ = cleanupConn.DeleteTopics(topic)
			_ = cleanupConn.Close()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Wait briefly until leader metadata is visible for the new topic before producing.
	deadlineLeader := time.Now().Add(10 * time.Second)
	for {
		lconn, lerr := kgo.DialLeader(ctx, "tcp", brokers[0], topic, 0)
		if lerr == nil {
			_ = lconn.Close()
			break
		}
		if time.Now().After(deadlineLeader) {
			t.Fatalf("topic %s leader not ready: %v", topic, lerr)
		}
		time.Sleep(150 * time.Millisecond)
	}

	now := time.Now().UTC()
	ontoBytes, err := json.Marshal(events.OntologyObjectChangedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventOntologyObjectChanged,
			Version:   1,
			Timestamp: now.Format(time.RFC3339Nano),
		},
		CellID:           "uz",
		ObjectType:       "Order",
		ObjectID:         "ord-live-" + suffix,
		SupplierID:       "sup-live-1",
		Version:          1,
		ChangeKind:       "UPSERT",
		SecurityMarkings: []string{"MARKING:FINANCE_ONLY"},
		UpdatedAt:        now.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	glBytes, err := json.Marshal(events.GLEntryPostedEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventGLEntryPosted,
			Version:   1,
			Timestamp: now.Format(time.RFC3339Nano),
		},
		EntryID:          "je-live-" + suffix,
		EntityID:         "sup-live-1",
		PeriodID:         "2026-11",
		SourceType:       "ORDER",
		SourceID:         "ord-live-" + suffix,
		Currency:         "UZS",
		TotalDebitMinor:  150000,
		TotalCreditMinor: 150000,
		Hash:             "livehash-" + suffix,
	})
	if err != nil {
		t.Fatal(err)
	}

	writer := &kgo.Writer{
		Addr:         kgo.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kgo.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
	}
	if err := writer.WriteMessages(ctx,
		kgo.Message{Key: []byte("ord-live"), Value: ontoBytes, Headers: []kgo.Header{{Key: "event_id", Value: []byte("evt-live-1-" + suffix)}}},
		kgo.Message{Key: []byte("je-live"), Value: glBytes, Headers: []kgo.Header{{Key: "event_id", Value: []byte("evt-live-2-" + suffix)}}},
	); err != nil {
		_ = writer.Close()
		t.Fatalf("WriteMessages to live Kafka %s: %v", topic, err)
	}
	_ = writer.Close()

	t.Setenv("PEGASUSX_ENV", "local")
	spoolDir := t.TempDir()
	spool, err := NewJSONLSpoolSink(spoolDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSpannerCheckpointStore(spannerClient)
	if err != nil {
		t.Fatal(err)
	}

	consumer, _, err := NewKafkaConsumer(ConsumerDeps{
		Brokers:       brokers,
		Topics:        []string{topic},
		GroupID:       groupID,
		SinkID:        sinkID,
		DefaultCellID: "uz",
		Sink:          spool,
		Checkpoints:   store,
	})
	if err != nil {
		t.Fatalf("NewKafkaConsumer: %v", err)
	}

	consCtx, consCancel := context.WithCancel(ctx)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		consumer.Start(consCtx)
	}()

	var cp Checkpoint
	var found bool
	pollDeadline := time.Now().Add(30 * time.Second)
	for {
		cp, found, err = store.GetCheckpoint(ctx, sinkID, topic, 0)
		if err != nil {
			consCancel()
			<-doneCh
			_ = consumer.Close()
			t.Fatalf("GetCheckpoint: %v", err)
		}
		if found && cp.OffsetVal >= 1 && cp.RowsExported >= 4 {
			break
		}
		if time.Now().After(pollDeadline) {
			consCancel()
			<-doneCh
			_ = consumer.Close()
			t.Fatalf("timed out waiting for live Kafka checkpoint on %s (found=%v cp=%+v)", topic, found, cp)
		}
		time.Sleep(150 * time.Millisecond)
	}

	consCancel()
	<-doneCh
	if cerr := consumer.Close(); cerr != nil {
		t.Fatalf("consumer.Close: %v", cerr)
	}

	if cp.OffsetVal != 1 || cp.RowsExported != 4 {
		t.Fatalf("expected checkpoint offset=1 rows_exported=4, got %+v", cp)
	}

	for _, check := range []struct {
		file      string
		wantLines int
	}{
		{"raw.domain_events.jsonl", 2},
		{"raw.ontology_objects.jsonl", 1},
		{"curated.gl_entries.jsonl", 1},
	} {
		data, rerr := os.ReadFile(filepath.Join(spoolDir, check.file))
		if rerr != nil {
			t.Fatalf("read %s: %v", check.file, rerr)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != check.wantLines {
			t.Fatalf("%s lines = %d, want %d", check.file, len(lines), check.wantLines)
		}
	}
}
