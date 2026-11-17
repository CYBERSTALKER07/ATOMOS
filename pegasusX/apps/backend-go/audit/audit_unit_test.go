package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
)

func TestCanonicalHashChain_RecordVerifyAndTamperDetection(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	day := "2026-11-16"
	ev1, err := svc.Record(ctx, EventInput{
		TenantID:   "sup-chain-1",
		DayBucket:  day,
		ActorID:    "usr-1",
		ActorType:  ActorTypeHuman,
		Role:       "OWNER",
		Action:     "order.approve",
		ObjectType: "order",
		ObjectID:   "ord-100",
		AfterJSON:  `{"status":"APPROVED"}`,
	})
	if err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if ev1.SeqNum != 1 || ev1.PrevHash != GenesisHash || len(ev1.EntryHash) != 64 {
		t.Fatalf("unexpected ev1: %+v", ev1)
	}

	ev2, err := svc.Record(ctx, EventInput{
		TenantID:   "sup-chain-1",
		DayBucket:  day,
		ActorID:    "agent-reorder",
		ActorType:  ActorTypeAgent,
		Role:       "AGENT",
		Action:     "proposal.create",
		ObjectType: "proposal",
		ObjectID:   "prop-200",
	})
	if err != nil {
		t.Fatalf("record 2: %v", err)
	}
	if ev2.SeqNum != 2 || ev2.PrevHash != ev1.EntryHash {
		t.Fatalf("unexpected ev2 linkage: %+v", ev2)
	}

	ev3, err := svc.Record(ctx, EventInput{
		TenantID:   "sup-chain-1",
		DayBucket:  day,
		ActorID:    "key:onec-1",
		ActorType:  ActorTypeConnector,
		Role:       "PARTNER",
		Action:     "connector.sync",
		ObjectType: "invoice",
		ObjectID:   "inv-300",
	})
	if err != nil {
		t.Fatalf("record 3: %v", err)
	}
	if ev3.SeqNum != 3 || ev3.PrevHash != ev2.EntryHash {
		t.Fatalf("unexpected ev3 linkage: %+v", ev3)
	}

	res, err := svc.VerifyDayChain(ctx, "sup-chain-1", day)
	if err != nil {
		t.Fatalf("VerifyDayChain: %v", err)
	}
	if !res.Valid || res.EventCount != 3 || res.RootHash != ev3.EntryHash || res.HeadEventID != ev3.EventID {
		t.Fatalf("unexpected verify result: %+v", res)
	}

	if !repo.TamperEventForTest("sup-chain-1", day, ev2.EventID, `{"tampered":true}`) {
		t.Fatal("failed to tamper test event")
	}
	tamperedRes, err := svc.VerifyDayChain(ctx, "sup-chain-1", day)
	if err != nil {
		t.Fatalf("VerifyDayChain after tamper: %v", err)
	}
	if tamperedRes.Valid || tamperedRes.BrokenEventID != ev2.EventID {
		t.Fatalf("expected tamper detection at %s, got %+v", ev2.EventID, tamperedRes)
	}
}

func TestMaskPIIForRole(t *testing.T) {
	raw := `{"retailer_id":"ret-1","phone":"+998901234567","PhoneNumber":"+998917654321","inn":"123456789","nested":[{"INN":987654321,"amount_minor":500000}]}`

	for _, allowedRole := range []string{"ADMIN", "PLATFORM_ADMIN", "OWNER", "FINANCE_MANAGER", "ACCOUNTANT"} {
		got := MaskPIIForRole(allowedRole, raw)
		if got != raw {
			t.Fatalf("role %s should see unmasked PII, got %s", allowedRole, got)
		}
	}

	for _, maskedRole := range []string{"AUDITOR", "WAREHOUSE_MANAGER", "SALES_MANAGER", "DISPATCHER", "CREDIT_OFFICER", "AGENT", ""} {
		got := MaskPIIForRole(maskedRole, raw)
		if strings.Contains(got, "998901234567") || strings.Contains(got, "998917654321") ||
			strings.Contains(got, "123456789") || strings.Contains(got, "987654321") {
			t.Fatalf("role %q leaked PII in %s", maskedRole, got)
		}
		if !strings.Contains(got, `"amount_minor":500000`) {
			t.Fatalf("role %q corrupted integer field in %s", maskedRole, got)
		}
	}
}

func TestLargeDiffOffload_Over32KB(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo).WithDiffWriter(repo)

	big := strings.Repeat("x", MaxInlineDiffBytes+512)
	beforeJSON := `{"payload":"` + big + `"}`
	afterJSON := `{"payload":"` + big + `y"}`

	ev, err := svc.Record(ctx, EventInput{
		TenantID:   "sup-big-1",
		DayBucket:  "2026-11-16",
		ActorID:    "usr-1",
		ActorType:  ActorTypeHuman,
		Action:     "catalog.bulk_import",
		ObjectType: "catalog",
		ObjectID:   "cat-1",
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
	})
	if err != nil {
		t.Fatalf("record large diff: %v", err)
	}
	if ev.DiffGcsURI == "" {
		t.Fatal("expected DiffGcsURI populated for >32KB diff")
	}
	if len(ev.BeforeJSON) >= MaxInlineDiffBytes || len(ev.AfterJSON) >= MaxInlineDiffBytes {
		t.Fatalf("expected inline summaries < 32KB, got before=%d after=%d", len(ev.BeforeJSON), len(ev.AfterJSON))
	}
	if !strings.Contains(ev.AfterJSON, ev.DiffGcsURI) || !strings.Contains(ev.AfterJSON, `"sha256"`) {
		t.Fatalf("expected SHA-256 summary in AfterJSON, got %s", ev.AfterJSON)
	}
	rawOffloaded, ok := repo.GetOffloadedDiffForTest(ev.DiffGcsURI)
	if !ok || len(rawOffloaded) <= MaxInlineDiffBytes {
		t.Fatalf("expected offloaded diff bytes stored, ok=%v len=%d", ok, len(rawOffloaded))
	}
	ver, err := svc.VerifyDayChain(ctx, "sup-big-1", "2026-11-16")
	if err != nil || !ver.Valid {
		t.Fatalf("expected valid hash chain after large diff offload, got %+v err=%v", ver, err)
	}
}

func TestWriteDailyAnchor_LocalWormAndProductionFailClosed(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	day := "2026-11-16"
	if _, err := svc.Record(ctx, EventInput{
		TenantID:  "sup-worm-1",
		DayBucket: day,
		ActorID:   "usr-1",
		ActorType: ActorTypeHuman,
		Action:    "order.create",
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	wormDir := t.TempDir()
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("WORM_ANCHOR_GCS_BUCKET", "")
	t.Setenv("WORM_ANCHOR_LOCAL_DIR", wormDir)

	anchor, err := svc.WriteDailyAnchor(ctx, "sup-worm-1", day, "usr-admin", nil)
	if err != nil {
		t.Fatalf("WriteDailyAnchor local worm: %v", err)
	}
	if !strings.HasPrefix(anchor.WormURI, "file://"+wormDir) {
		t.Fatalf("unexpected WormURI: %s", anchor.WormURI)
	}
	localPath := strings.TrimPrefix(anchor.WormURI, "file://")
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		t.Fatalf("stat local worm file: %v", statErr)
	}
	if info.Mode().Perm() != 0o444 {
		t.Fatalf("expected 0444 read-only mode on WORM file, got %v", info.Mode().Perm())
	}
	rawAnchor, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("read worm anchor file: %v", err)
	}
	var blob AnchorBlob
	if err := json.Unmarshal(rawAnchor, &blob); err != nil {
		t.Fatalf("decode worm anchor blob: %v", err)
	}
	if blob.RootHash != anchor.RootHash || blob.Signature == "" || filepath.Clean(localPath) == "" {
		t.Fatalf("invalid worm anchor blob: %+v", blob)
	}

	ver, err := svc.VerifyDayChain(ctx, "sup-worm-1", day)
	if err != nil || !ver.Valid || ver.AnchorMatched == nil || !*ver.AnchorMatched {
		t.Fatalf("expected VerifyDayChain anchor match, got %+v err=%v", ver, err)
	}

	// Production must fail closed if WORM_ANCHOR_GCS_BUCKET is set without usable GCS credentials.
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("WORM_ANCHOR_GCS_BUCKET", "pegasusx-prod-worm-anchors")
	t.Setenv("WORM_ANCHOR_LOCAL_DIR", "")
	origStorageFn := newStorageClientFunc
	newStorageClientFunc = func(context.Context) (*storage.Client, error) {
		return nil, errors.New("adc credentials missing")
	}
	t.Cleanup(func() { newStorageClientFunc = origStorageFn })

	// Add a new event so WriteDailyAnchor attempts a fresh anchor write instead of returning idempotently.
	if _, err := svc.Record(ctx, EventInput{
		TenantID:  "sup-worm-1",
		DayBucket: day,
		ActorID:   "usr-2",
		ActorType: ActorTypeHuman,
		Action:    "order.update",
	}); err != nil {
		t.Fatalf("record 2: %v", err)
	}
	if _, err := svc.WriteDailyAnchor(ctx, "sup-worm-1", day, "usr-admin", nil); !errors.Is(err, ErrAnchorWormRequired) {
		t.Fatalf("expected ErrAnchorWormRequired in production without GCS creds, got %v", err)
	}

	if _, err := NewLocalWormSink(wormDir); !errors.Is(err, ErrAnchorWormRequired) {
		t.Fatalf("expected NewLocalWormSink rejected in production, got %v", err)
	}
}

func TestLegalHold_AndLegacyFallbackHistory(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	hold, err := svc.CreateLegalHold(ctx, LegalHoldInput{
		TenantID:   "sup-hold-1",
		ObjectType: "order",
		ObjectID:   "ord-hold-9",
		Reason:     "tax dispute",
		CreatedBy:  "usr-legal",
	})
	if err != nil {
		t.Fatalf("CreateLegalHold: %v", err)
	}
	if hold.Status != LegalHoldStatusActive {
		t.Fatalf("expected ACTIVE hold, got %s", hold.Status)
	}

	repo.SeedLegacyAuditLogForTest(AuditEvent{
		TenantID:   "sup-hold-1",
		EventID:    "leg-1",
		ActorID:    "usr-legacy",
		ActorType:  ActorTypeHuman,
		Role:       "ADMIN",
		Action:     "ORDER_CREATED",
		ObjectType: "order",
		ObjectID:   "ord-hold-9",
		AfterJSON:  `{"phone":"+998901112233","inn":"305123456"}`,
		CreatedAt:  time.Now().UTC().Add(-1 * time.Hour),
	})

	ev, err := svc.Record(ctx, EventInput{
		TenantID:   "sup-hold-1",
		ActorID:    "usr-fin",
		ActorType:  ActorTypeHuman,
		Role:       "FINANCE_MANAGER",
		Action:     "order.amend",
		ObjectType: "order",
		ObjectID:   "ord-hold-9",
		AfterJSON:  `{"phone":"+998909998877","inn":"305123456"}`,
	})
	if err != nil {
		t.Fatalf("record under legal hold: %v", err)
	}
	if !ev.LegalHold || ev.LegalHoldReason != "tax dispute" {
		t.Fatalf("expected legal hold inherited on subsequent event, got %+v", ev)
	}

	auditorHist, err := svc.GetObjectHistory(ctx, "sup-hold-1", "order", "ord-hold-9", "AUDITOR")
	if err != nil {
		t.Fatalf("GetObjectHistory auditor: %v", err)
	}
	if len(auditorHist) != 3 {
		t.Fatalf("expected 3 history rows (hold + amend + legacy), got %d", len(auditorHist))
	}
	for _, h := range auditorHist {
		if strings.Contains(h.AfterJSON, "998901112233") || strings.Contains(h.AfterJSON, "998909998877") || strings.Contains(h.AfterJSON, "305123456") {
			t.Fatalf("auditor leaked PII in history row %s: %s", h.EventID, h.AfterJSON)
		}
	}

	finHist, err := svc.GetObjectHistory(ctx, "sup-hold-1", "order", "ord-hold-9", "FINANCE_MANAGER")
	if err != nil {
		t.Fatalf("GetObjectHistory finance: %v", err)
	}
	foundUnmasked := false
	for _, h := range finHist {
		if strings.Contains(h.AfterJSON, "+998909998877") && !h.PIIMasked {
			foundUnmasked = true
		}
	}
	if !foundUnmasked {
		t.Fatalf("expected finance manager to see unmasked PII in history: %+v", finHist)
	}
}

func TestAuditMiddleware_MutationsOutcomesAndActors(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	r := chi.NewRouter()
	r.Use(AuditMiddleware(svc))
	r.Get("/v1/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Post("/v1/orders/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
		_ = WithMutationContext(r.Context(), MutationContext{
			ObjectType: "order",
			ObjectID:   chi.URLParam(r, "id"),
			BeforeJSON: `{"status":"PENDING"}`,
			AfterJSON:  `{"status":"APPROVED"}`,
			ReasonCode: "CREDIT_OK",
			ApprovalID: "apr-77",
		})
		w.WriteHeader(http.StatusCreated)
	})
	r.Patch("/v1/orders/{id}/fail4xx", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	r.Delete("/v1/orders/{id}/boom5xx", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// 1. GET must not record an audit event.
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/orders/ord-1", nil)
	r.ServeHTTP(httptest.NewRecorder(), reqGet)

	// 2. POST with human claims + TraceID + IP + DeviceId + MutationContext.
	reqPost := httptest.NewRequest(http.MethodPost, "/v1/orders/ord-1/approve", nil)
	reqPost.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
	reqPost.Header.Set("X-Device-Id", "ios-dev-42")
	ctxPost := outbox.WithTraceID(reqPost.Context(), "trace-human-1")
	ctxPost = auth.WithClaims(ctxPost, auth.Claims{Subject: "usr-approver", Role: auth.RoleAdmin, SupplierID: "sup-mw-1", SupplierRole: auth.Role("FINANCE_MANAGER")})
	ctxPost = auth.WithTenant(ctxPost, auth.TenantContext{SupplierID: "sup-mw-1", Source: "jwt"})
	r.ServeHTTP(httptest.NewRecorder(), reqPost.WithContext(ctxPost))

	// 3. PATCH returning 403 as agent actor.
	reqPatch := httptest.NewRequest(http.MethodPatch, "/v1/orders/ord-1/fail4xx", nil)
	ctxPatch := auth.WithClaims(reqPatch.Context(), auth.Claims{Subject: "agent-1", Role: auth.RoleAdmin, SupplierID: "sup-mw-1", ActorType: auth.ActorTypeAgent})
	ctxPatch = auth.WithTenant(ctxPatch, auth.TenantContext{SupplierID: "sup-mw-1", Source: "jwt"})
	r.ServeHTTP(httptest.NewRecorder(), reqPatch.WithContext(ctxPatch))

	// 4. DELETE returning 500 as partner connector.
	reqDel := httptest.NewRequest(http.MethodDelete, "/v1/orders/ord-1/boom5xx", nil)
	ctxDel := partner.WithPrincipal(reqDel.Context(), partner.Principal{KeyID: "pk_onec_9", TenantType: partner.TenantSupplier, TenantID: "sup-mw-1"})
	ctxDel = auth.WithTenant(ctxDel, auth.TenantContext{SupplierID: "sup-mw-1", TenantType: "SUPPLIER", TenantID: "sup-mw-1", KeyID: "pk_onec_9", Source: "partner"})
	r.ServeHTTP(httptest.NewRecorder(), reqDel.WithContext(ctxDel))

	events, err := svc.ListEvents(ctx, Filter{TenantID: "sup-mw-1", Limit: 20})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected exactly 3 mutating audit events (GET ignored), got %d", len(events))
	}

	// Verify day chain across all middleware-recorded events.
	today := time.Now().UTC().Format("2006-01-02")
	ver, err := svc.VerifyDayChain(ctx, "sup-mw-1", today)
	if err != nil || !ver.Valid || ver.EventCount != 3 {
		t.Fatalf("VerifyDayChain middleware events failed: %+v err=%v", ver, err)
	}

	var sawHuman, sawAgent403, sawConnector500 bool
	for _, ev := range events {
		switch ev.ActorType {
		case ActorTypeHuman:
			sawHuman = true
			if ev.Action != "POST /v1/orders/{id}/approve" || ev.ApprovalID != "apr-77" || ev.ReasonCode != "CREDIT_OK" || ev.TraceID != "trace-human-1" || ev.IP != "198.51.100.7" || ev.DeviceID != "ios-dev-42" {
				t.Fatalf("unexpected human event: %+v", ev)
			}
		case ActorTypeAgent:
			sawAgent403 = true
			if !strings.Contains(ev.AfterJSON, `"http_status":403`) || !strings.Contains(ev.AfterJSON, `"client_error"`) {
				t.Fatalf("expected 403 client_error metadata on agent event: %s", ev.AfterJSON)
			}
		case ActorTypeConnector:
			sawConnector500 = true
			if !strings.Contains(ev.AfterJSON, `"http_status":500`) || !strings.Contains(ev.AfterJSON, `"server_error"`) {
				t.Fatalf("expected 500 server_error metadata on connector event: %s", ev.AfterJSON)
			}
		}
	}
	if !sawHuman || !sawAgent403 || !sawConnector500 {
		t.Fatalf("missing actor event types: human=%v agent=%v connector=%v", sawHuman, sawAgent403, sawConnector500)
	}
}

func TestStartNightlyAnchorWorker_AndPrefixAnchorMismatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("WORM_ANCHOR_GCS_BUCKET", "")
	t.Setenv("WORM_ANCHOR_LOCAL_DIR", t.TempDir())

	ev1, err := svc.Record(ctx, EventInput{
		TenantID:  "sup-nightly-1",
		DayBucket: yesterday,
		ActorID:   "usr-1",
		ActorType: ActorTypeHuman,
		Action:    "order.create",
	})
	if err != nil {
		t.Fatalf("record yesterday event: %v", err)
	}

	done := make(chan struct{})
	go func() {
		StartNightlyAnchorWorker(ctx, 25*time.Millisecond, svc)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var anchors []Anchor
	for time.Now().Before(deadline) {
		anchors, err = svc.ListAnchors(ctx, "sup-nightly-1", yesterday, 10)
		if err == nil && len(anchors) > 0 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	cancel()
	<-done

	if len(anchors) != 1 || anchors[0].RootHash != ev1.EntryHash || anchors[0].WormURI == "" {
		t.Fatalf("expected nightly worker anchor for %s, got %+v", yesterday, anchors)
	}

	// Verify prefix-anchor holds even when another event appends to the same day after anchoring.
	if _, err := svc.Record(context.Background(), EventInput{
		TenantID:  "sup-nightly-1",
		DayBucket: yesterday,
		ActorID:   "usr-2",
		ActorType: ActorTypeHuman,
		Action:    "order.amend",
	}); err != nil {
		t.Fatalf("record post-anchor event: %v", err)
	}
	ver, err := svc.VerifyDayChain(context.Background(), "sup-nightly-1", yesterday)
	if err != nil || !ver.Valid || ver.EventCount != 2 || ver.AnchorMatched == nil || !*ver.AnchorMatched {
		t.Fatalf("expected prefix-anchor valid after subsequent append, got %+v err=%v", ver, err)
	}

	// Tamper event 1 -> prefix anchor & entry hash verification must fail.
	repo.TamperEventForTest("sup-nightly-1", yesterday, ev1.EventID, `{"tampered":"prefix"}`)
	badVer, err := svc.VerifyDayChain(context.Background(), "sup-nightly-1", yesterday)
	if err != nil || badVer.Valid || badVer.BrokenEventID != ev1.EventID {
		t.Fatalf("expected prefix tamper caught at %s, got %+v err=%v", ev1.EventID, badVer, err)
	}
}
