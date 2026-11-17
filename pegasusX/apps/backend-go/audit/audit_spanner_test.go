package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openAuditSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner audit integration test")
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
	_, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "AuditEvents", spanner.Key{"__probe__", "2026-01-01", "__probe__"}, []string{"EventId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("AuditEvents table unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("AuditEvents table unavailable (%s): %v", dbPath, probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

type stubWormSink struct {
	mu    sync.Mutex
	calls int
	blobs []AnchorBlob
}

func (s *stubWormSink) WriteAnchor(_ context.Context, blob AnchorBlob, _ []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.blobs = append(s.blobs, blob)
	return fmt.Sprintf("gs://stub-worm/anchors/%s/%s-%d.json", blob.TenantID, blob.DayBucket, s.calls), nil
}

func TestAudit_Spanner_BatchConcurrencyVerifyAnchorOutboxAndLegalHold(t *testing.T) {
	client := openAuditSpannerTestClient(t)
	ctx := context.Background()
	svc := NewService(client)

	tenantID := fmt.Sprintf("sup-aud-%d", time.Now().UnixNano())
	dayBucket := time.Now().UTC().Format("2006-01-02")
	legacyID := fmt.Sprintf("leg-aud-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditChainAnchors", spanner.Key{tenantID, dayBucket}),
			spanner.Delete("AuditLegalHolds", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditLog", spanner.Key{legacyID}),
		})
	})

	// 1. Intra-transaction BatchRecorder chaining (3 events -> SeqNum 1..3).
	if err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		br := NewBatchRecorder(txn)
		for i := 1; i <= 3; i++ {
			if _, err := br.Record(ctx, EventInput{
				TenantID:   tenantID,
				DayBucket:  dayBucket,
				ActorID:    fmt.Sprintf("usr-batch-%d", i),
				ActorType:  ActorTypeHuman,
				Role:       "OWNER",
				Action:     "batch.step",
				ObjectType: "order",
				ObjectID:   "ord-spanner-1",
				AfterJSON:  fmt.Sprintf(`{"step":%d,"phone":"+998901234567"}`, i),
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("BatchRecorder RW txn: %v", err)
	}

	// 2. 6 concurrent goroutines calling Record on the same (tenantID, dayBucket) -> SeqNum 4..9.
	var wg sync.WaitGroup
	errCh := make(chan error, 6)
	for g := 0; g < 6; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := svc.Record(ctx, EventInput{
				TenantID:   tenantID,
				DayBucket:  dayBucket,
				ActorID:    fmt.Sprintf("agent-%d", idx),
				ActorType:  ActorTypeAgent,
				Role:       "AGENT",
				Action:     "agent.proposal",
				ObjectType: "order",
				ObjectID:   "ord-spanner-1",
				AfterJSON:  fmt.Sprintf(`{"goroutine":%d,"inn":"123456789"}`, idx),
			})
			if err != nil {
				errCh <- err
			}
		}(g)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Record failed: %v", err)
	}

	// 3. CreateLegalHold + subsequent Record inherits LegalHold=true.
	hold, err := svc.CreateLegalHold(ctx, LegalHoldInput{
		TenantID:   tenantID,
		ObjectType: "order",
		ObjectID:   "ord-spanner-1",
		Reason:     "regulatory audit",
		CreatedBy:  "usr-owner",
	})
	if err != nil {
		t.Fatalf("CreateLegalHold: %v", err)
	}
	postHoldEv, err := svc.Record(ctx, EventInput{
		TenantID:   tenantID,
		DayBucket:  dayBucket,
		ActorID:    "usr-fin",
		ActorType:  ActorTypeHuman,
		Role:       "FINANCE_MANAGER",
		Action:     "order.reconcile",
		ObjectType: "order",
		ObjectID:   "ord-spanner-1",
		AfterJSON:  `{"phone":"+998917776655","inn":"309876543"}`,
	})
	if err != nil {
		t.Fatalf("Record under legal hold: %v", err)
	}
	if !postHoldEv.LegalHold || postHoldEv.LegalHoldReason != hold.Reason {
		t.Fatalf("expected legal hold flag inherited, got %+v", postHoldEv)
	}

	holds, err := svc.ListLegalHolds(ctx, tenantID, "order", "ord-spanner-1", LegalHoldStatusActive, 20)
	if err != nil || len(holds) != 1 || holds[0].HoldID != hold.HoldID {
		t.Fatalf("ListLegalHolds failed: %+v err=%v", holds, err)
	}

	// 4. VerifyDayChain across all 11 events (3 batch + 6 concurrent + 1 legal hold + 1 post-hold).
	ver, err := svc.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil {
		t.Fatalf("VerifyDayChain: %v", err)
	}
	if !ver.Valid || ver.EventCount != 11 || ver.RootHash != postHoldEv.EntryHash {
		t.Fatalf("VerifyDayChain unexpected result: %+v (expected root %s)", ver, postHoldEv.EntryHash)
	}

	// 5. WriteDailyAnchor writes AuditChainAnchors + OutboxEvents (EventAuditAnchorWritten) idempotently.
	sink := &stubWormSink{}
	anchor1, err := svc.WriteDailyAnchor(ctx, tenantID, dayBucket, "usr-owner", sink)
	if err != nil {
		t.Fatalf("WriteDailyAnchor: %v", err)
	}
	if anchor1.RootHash != ver.RootHash || anchor1.EventCount != 11 || !strings.HasPrefix(anchor1.WormURI, "gs://stub-worm/") {
		t.Fatalf("unexpected anchor1: %+v", anchor1)
	}
	anchor2, err := svc.WriteDailyAnchor(ctx, tenantID, dayBucket, "usr-owner", sink)
	if err != nil {
		t.Fatalf("idempotent WriteDailyAnchor: %v", err)
	}
	if anchor2.WormURI != anchor1.WormURI || sink.calls != 1 {
		t.Fatalf("expected idempotent anchor without duplicate WORM write, calls=%d a2=%+v", sink.calls, anchor2)
	}

	// Check OutboxEvents for EventAuditAnchorWritten.
	var outboxCount int
	if err := spannerutils.RunReadOnlyTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT TopicName, Payload, SupplierId FROM OutboxEvents WHERE AggregateId = @aggId`,
			Params: map[string]any{
				"aggId": tenantID + ":" + dayBucket,
			},
		})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return nextErr
			}
			var topic, supOutbox string
			var payload []byte
			if err := row.Columns(&topic, &payload, &supOutbox); err != nil {
				return err
			}
			var ev struct {
				Type     string `json:"type"`
				RootHash string `json:"root_hash"`
			}
			if err := json.Unmarshal(payload, &ev); err == nil && ev.Type == events.EventAuditAnchorWritten {
				outboxCount++
				if topic != events.TopicLedger || ev.RootHash != anchor1.RootHash || supOutbox != tenantID {
					return fmt.Errorf("unexpected outbox topic=%s sup=%s root=%s", topic, supOutbox, ev.RootHash)
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("query OutboxEvents: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected exactly 1 EventAuditAnchorWritten outbox row, got %d", outboxCount)
	}

	// VerifyDayChain also validates anchor match.
	verAnchored, err := svc.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil || !verAnchored.Valid || verAnchored.AnchorMatched == nil || !*verAnchored.AnchorMatched {
		t.Fatalf("expected VerifyDayChain anchor matched, got %+v err=%v", verAnchored, err)
	}

	// 6. Seed legacy AuditLog entry and verify GetObjectHistory combines + applies PII masking by role.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("AuditLog", map[string]any{
			"AuditId":       legacyID,
			"SupplierId":    tenantID,
			"ActorId":       "usr-legacy-1",
			"ActorRole":     "ADMIN",
			"Action":        "ORDER_CREATED",
			"AggregateType": "order",
			"AggregateId":   "ord-spanner-1",
			"DetailsJson":   []byte(`{"phone":"+998905554433","inn":"200111222"}`),
			"TraceId":       "trace-legacy",
			"CreatedAt":     time.Now().UTC().Add(-2 * time.Hour),
		}),
	}); err != nil {
		t.Fatalf("seed legacy AuditLog: %v", err)
	}

	auditorHist, err := svc.GetObjectHistory(ctx, tenantID, "order", "ord-spanner-1", "AUDITOR")
	if err != nil {
		t.Fatalf("GetObjectHistory auditor: %v", err)
	}
	if len(auditorHist) < 12 {
		t.Fatalf("expected >=12 combined AuditEvents+AuditLog rows, got %d", len(auditorHist))
	}
	for _, h := range auditorHist {
		if strings.Contains(h.AfterJSON, "998905554433") || strings.Contains(h.AfterJSON, "998917776655") || strings.Contains(h.AfterJSON, "200111222") || strings.Contains(h.AfterJSON, "309876543") {
			t.Fatalf("auditor leaked PII in %s: %s", h.EventID, h.AfterJSON)
		}
	}

	finHist, err := svc.GetObjectHistory(ctx, tenantID, "order", "ord-spanner-1", "FINANCE_MANAGER")
	if err != nil {
		t.Fatalf("GetObjectHistory finance: %v", err)
	}
	var sawLegacyUnmasked bool
	for _, h := range finHist {
		if h.EventID == legacyID && strings.Contains(h.AfterJSON, "+998905554433") && h.Source == "legacy_audit_log" {
			sawLegacyUnmasked = true
		}
	}
	if !sawLegacyUnmasked {
		t.Fatalf("expected unmasked legacy AuditLog row in finance history")
	}

	// 7. Direct Spanner tamper test: insert a broken EntryHash row as SeqNum 12 and confirm VerifyDayChain detects it.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("AuditEvents", map[string]any{
			"TenantId":  tenantID,
			"DayBucket": dayBucket,
			"EventId":   "aud-tampered-12",
			"SeqNum":    int64(12),
			"ActorId":   "intruder",
			"ActorType": "human",
			"Action":    "tamper.test",
			"PrevHash":  postHoldEv.EntryHash,
			"EntryHash": strings.Repeat("f", 64),
			"LegalHold": false,
			"CreatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("insert tampered row: %v", err)
	}
	brokenRes, err := svc.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil {
		t.Fatalf("VerifyDayChain tampered: %v", err)
	}
	if brokenRes.Valid || brokenRes.BrokenEventID != "aud-tampered-12" {
		t.Fatalf("expected tamper detection on aud-tampered-12, got %+v", brokenRes)
	}

	// 8. Verify ListEvents filters (ActorID, ActorType, Action, cross-tenant) and ListAnchors.
	actorEvents, err := svc.ListEvents(ctx, Filter{
		TenantID:  tenantID,
		ActorID:   "usr-fin",
		ActorType: ActorTypeHuman,
		DayBucket: dayBucket,
		Limit:     10,
	})
	if err != nil || len(actorEvents) != 1 || actorEvents[0].EventID != postHoldEv.EventID {
		t.Fatalf("filtered ListEvents unexpected: %+v err=%v", actorEvents, err)
	}
	anchors, err := svc.ListAnchors(ctx, tenantID, dayBucket, 10)
	if err != nil || len(anchors) != 1 || anchors[0].RootHash != anchor1.RootHash {
		t.Fatalf("ListAnchors unexpected: %+v err=%v", anchors, err)
	}
}

func TestAudit_Spanner_LargeDiffOffload(t *testing.T) {
	client := openAuditSpannerTestClient(t)
	ctx := context.Background()
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("WORM_ANCHOR_GCS_BUCKET", "")
	wormDir := t.TempDir()
	t.Setenv("WORM_ANCHOR_LOCAL_DIR", wormDir)

	svc := NewService(client)
	tenantID := fmt.Sprintf("sup-big-sp-%d", time.Now().UnixNano())
	dayBucket := time.Now().UTC().Format("2006-01-02")
	t.Cleanup(func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		})
	})

	huge := `{"blob":"` + strings.Repeat("Z", MaxInlineDiffBytes+1024) + `"}`
	ev, err := svc.Record(ctx, EventInput{
		TenantID:   tenantID,
		DayBucket:  dayBucket,
		ActorID:    "usr-big",
		ActorType:  ActorTypeHuman,
		Action:     "catalog.bulk_upsert",
		ObjectType: "catalog",
		ObjectID:   "cat-big-1",
		AfterJSON:  huge,
	})
	if err != nil {
		t.Fatalf("Record large diff on Spanner: %v", err)
	}
	if !strings.HasPrefix(ev.DiffGcsURI, "file://"+wormDir) {
		t.Fatalf("expected local WORM DiffGcsUri, got %q", ev.DiffGcsURI)
	}
	if len(ev.AfterJSON) >= MaxInlineDiffBytes || !strings.Contains(ev.AfterJSON, `"sha256"`) {
		t.Fatalf("expected compact SHA-256 summary in AfterJSON, got len=%d body=%s", len(ev.AfterJSON), ev.AfterJSON)
	}
	ver, err := svc.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil || !ver.Valid || ver.EventCount != 1 {
		t.Fatalf("VerifyDayChain large diff on Spanner failed: %+v err=%v", ver, err)
	}
}
