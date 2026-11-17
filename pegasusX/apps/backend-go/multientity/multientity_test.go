package multientity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/multientity"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type memAuditRecorder struct {
	mu     sync.Mutex
	events []multientity.AuditRecordInput
}

func (a *memAuditRecorder) RecordEvent(_ context.Context, ev multientity.AuditRecordInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, ev)
	return nil
}

func (a *memAuditRecorder) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev multientity.AuditRecordInput) error {
	return a.RecordEvent(context.Background(), ev)
}

func (a *memAuditRecorder) snapshot() []multientity.AuditRecordInput {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]multientity.AuditRecordInput, len(a.events))
	copy(out, a.events)
	return out
}

type spannerAuditAdapter struct {
	svc *audit.Service
}

func (a spannerAuditAdapter) RecordEvent(ctx context.Context, ev multientity.AuditRecordInput) error {
	if a.svc == nil {
		return nil
	}
	_, err := a.svc.Record(ctx, toAuditInput(ev))
	return err
}

func (a spannerAuditAdapter) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev multientity.AuditRecordInput) error {
	if a.svc == nil || txn == nil {
		return nil
	}
	_, err := a.svc.RecordInTxn(ctx, txn, toAuditInput(ev))
	return err
}

func toAuditInput(ev multientity.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

func openMultientitySpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner multientity integration test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
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
		if requireSpanner {
			t.Fatalf("spanner emulator connect (%s): %v", dbPath, err)
		}
		t.Skipf("spanner emulator unavailable (%s): %v", dbPath, err)
	}

	_, probeErr := client.Single().ReadRow(ctx, "LegalEntities", spanner.Key{"__me_probe__", "__me_probe__"}, []string{"EntityId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("LegalEntities table unavailable (%s): %v", dbPath, probeErr)
		}
		t.Skipf("LegalEntities table unavailable (%s): %v", dbPath, probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
}

func cleanupMultientityTenant(t *testing.T, client *spanner.Client, tenantID string, fxRateIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		muts := []*spanner.Mutation{
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("IntercompanyTransfers", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("FxRevaluationRuns", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("LegalEntities", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Periods", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		}
		for _, rid := range fxRateIDs {
			muts = append(muts, spanner.Delete("FxRates", spanner.Key{rid}))
		}

		ro := client.ReadOnlyTransaction()
		iter := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": tenantID},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		ro.Close()
		_, _ = client.Apply(ctx, muts)
	})
}

func newUnitMultientityStack() (*multientity.Service, *multientity.MemoryRepository, *gl.Service, *fxrates.MemoryRepository, *memAuditRecorder) {
	glMem := gl.NewMemoryRepository()
	glSvc := gl.NewServiceWithRepository(glMem)
	fxRepo := fxrates.NewMemoryRepository()
	fxSvc := fxrates.NewService(fxRepo)
	meRepo := multientity.NewMemoryRepository(glSvc)
	svc := multientity.NewServiceWithDeps(meRepo, glSvc, fxSvc, fxRepo)
	aud := &memAuditRecorder{}
	svc.SetAuditRecorder(aud)
	glSvc.SetLegalEntityOwnershipChecker(svc)
	return svc, meRepo, glSvc, fxRepo, aud
}

func TestEntityCRUD_And_Audit_Unit(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, aud := newUnitMultientityStack()
	tenantID := "ten_me_crud"

	p, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID:     tenantID,
		EntityID:     "ent_uz_hq",
		Name:         "Tashkent HQ",
		TIN:          "301122334",
		CountryCode:  "UZ",
		BaseCurrency: "UZS",
		IsParent:     true,
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("CreateEntity(parent): %v", err)
	}
	if p.Status != multientity.EntityStatusActive || p.BaseCurrency != "UZS" || !p.IsParent {
		t.Fatalf("unexpected created parent entity: %+v", p)
	}

	// Duplicate ID rejected with ErrEntityExists.
	if _, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID,
		EntityID: "ent_uz_hq",
		Name:     "Dup",
	}); !errors.Is(err, multientity.ErrEntityExists) {
		t.Fatalf("duplicate CreateEntity err = %v, want ErrEntityExists", err)
	}

	// Auto-generated entity ID + child currency.
	sub, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID:       tenantID,
		Name:           "Almaty Distribution",
		TIN:            "998877665544",
		CountryCode:    "KZ",
		BaseCurrency:   "KZT",
		ParentEntityID: p.EntityID,
		ActorID:        "admin-1",
	})
	if err != nil {
		t.Fatalf("CreateEntity(sub): %v", err)
	}
	if sub.EntityID == "" || sub.BaseCurrency != "KZT" || sub.ParentEntityID != p.EntityID {
		t.Fatalf("unexpected created sub entity: %+v", sub)
	}

	// Patch entity status to INACTIVE and update name.
	newName := "Almaty Branch Renamed"
	inactive := multientity.EntityStatusInactive
	patched, err := svc.PatchEntity(ctx, multientity.PatchEntityInput{
		TenantID: tenantID,
		EntityID: sub.EntityID,
		Name:     &newName,
		Status:   &inactive,
		ActorID:  "admin-2",
	})
	if err != nil {
		t.Fatalf("PatchEntity: %v", err)
	}
	if patched.Name != newName || patched.Status != multientity.EntityStatusInactive {
		t.Fatalf("unexpected patched entity: %+v", patched)
	}

	activeList, err := svc.ListEntities(ctx, tenantID, multientity.EntityStatusActive)
	if err != nil {
		t.Fatalf("ListEntities(ACTIVE): %v", err)
	}
	if len(activeList) != 1 || activeList[0].EntityID != p.EntityID {
		t.Fatalf("unexpected active list: %+v", activeList)
	}

	allList, err := svc.ListEntities(ctx, tenantID, "")
	if err != nil || len(allList) != 2 {
		t.Fatalf("ListEntities(all) got (%d, %v), want (2, nil)", len(allList), err)
	}

	evs := aud.snapshot()
	if len(evs) != 3 {
		t.Fatalf("audit events count = %d, want 3", len(evs))
	}
	if evs[2].Action != "org.entity.update" || evs[2].ObjectID != sub.EntityID || evs[2].BeforeJSON == "" || evs[2].AfterJSON == "" {
		t.Fatalf("unexpected patch audit event: %+v", evs[2])
	}
}

func TestIntercompanyTransfer_CrossCurrency_And_Guards_Unit(t *testing.T) {
	ctx := context.Background()
	svc, _, glSvc, fxRepo, aud := newUnitMultientityStack()
	tenantID := "ten_me_ic"

	_, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_uz", Name: "UZ Parent", CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_kz", Name: "KZ Sub", CountryCode: "KZ", BaseCurrency: "KZT", ParentEntityID: "ent_uz",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_off", Name: "Dormant Sub", CountryCode: "UZ", BaseCurrency: "UZS",
	})
	if err != nil {
		t.Fatal(err)
	}
	inactive := multientity.EntityStatusInactive
	if _, err := svc.PatchEntity(ctx, multientity.PatchEntityInput{TenantID: tenantID, EntityID: "ent_off", Status: &inactive}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_usd_uzs", BaseCurrency: "USD", QuoteCurrency: "UZS",
		RateScaled: 12800 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: now.Add(-time.Hour), Source: "CBU",
	}); err != nil {
		t.Fatal(err)
	}
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_usd_kzt", BaseCurrency: "USD", QuoteCurrency: "KZT",
		RateScaled: 480 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: now.Add(-time.Hour), Source: "NBK",
	}); err != nil {
		t.Fatal(err)
	}

	// Same entity rejected.
	if _, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID: tenantID, FromEntityID: "ent_uz", ToEntityID: "ent_uz", AmountMinor: 10000, Currency: "USD", PeriodID: "2026-11", IdempotencyKey: "ic-same",
	}); !errors.Is(err, multientity.ErrSameEntityTransfer) {
		t.Fatalf("same-entity err = %v, want ErrSameEntityTransfer", err)
	}

	// Inactive entity rejected.
	if _, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID: tenantID, FromEntityID: "ent_uz", ToEntityID: "ent_off", AmountMinor: 10000, Currency: "UZS", PeriodID: "2026-11", IdempotencyKey: "ic-inact",
	}); !errors.Is(err, multientity.ErrEntityInactive) {
		t.Fatalf("inactive-entity err = %v, want ErrEntityInactive", err)
	}

	// Cross-currency USD transfer between UZ (UZS) and KZ (KZT): 100.00 USD -> 1,280,000.00 UZS / 48,000.00 KZT.
	res1, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID:       tenantID,
		FromEntityID:   "ent_uz",
		ToEntityID:     "ent_kz",
		TransferType:   "CASH",
		AmountMinor:    10000,
		Currency:       "USD",
		PeriodID:       "2026-11",
		Memo:           "Working capital advance",
		IdempotencyKey: "ic-usd-1",
		CreatedBy:      "treasurer-1",
	})
	if err != nil {
		t.Fatalf("CreateIntercompanyTransfer: %v", err)
	}
	if res1.AlreadyPosted || res1.Transfer.DebitJournalID == "" || res1.Transfer.CreditJournalID == "" {
		t.Fatalf("unexpected transfer result: %+v", res1)
	}

	// Idempotent replay does not duplicate.
	res2, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID:       tenantID,
		FromEntityID:   "ent_uz",
		ToEntityID:     "ent_kz",
		TransferType:   "CASH",
		AmountMinor:    10000,
		Currency:       "USD",
		PeriodID:       "2026-11",
		IdempotencyKey: "ic-usd-1",
	})
	if err != nil || !res2.AlreadyPosted || res2.Transfer.TransferID != res1.Transfer.TransferID {
		t.Fatalf("idempotency replay got (%+v, %v), want AlreadyPosted=true, same TransferID", res2, err)
	}

	fromJ, err := glSvc.GetEntry(ctx, tenantID, res1.Transfer.DebitJournalID)
	if err != nil {
		t.Fatal(err)
	}
	if fromJ.LegalEntityID != "ent_uz" || fromJ.Postings[0].AccountCode != gl.AccountIntercompanyDueFrom || fromJ.Postings[0].FunctionalAmountMinor != 128000000 {
		t.Fatalf("unexpected debit IC journal: %+v", fromJ)
	}
	toJ, err := glSvc.GetEntry(ctx, tenantID, res1.Transfer.CreditJournalID)
	if err != nil {
		t.Fatal(err)
	}
	if toJ.LegalEntityID != "ent_kz" || toJ.Postings[1].AccountCode != gl.AccountIntercompanyDueTo || toJ.Postings[1].FunctionalAmountMinor != 4800000 {
		t.Fatalf("unexpected credit IC journal: %+v", toJ)
	}

	trs, err := svc.ListTransfers(ctx, tenantID, "2026-11", "ent_kz", 10)
	if err != nil || len(trs) != 1 {
		t.Fatalf("ListTransfers got (%d, %v), want (1, nil)", len(trs), err)
	}
	audEvs := aud.snapshot()
	if audEvs[len(audEvs)-1].Action != "finance.intercompany.transfer" {
		t.Fatalf("expected finance.intercompany.transfer audit event, got %+v", audEvs[len(audEvs)-1])
	}

	// Verify all portal/type IntercompanyTransferType variants map to valid offset accounts.
	uiTypes := map[string]string{
		"CASH_POOL":          gl.AccountBankOperating,
		"EXPENSE_RECHARGE":   gl.AccountClaimsSettlementExpense,
		"INVENTORY_TRANSFER": gl.AccountInventory,
		"LOAN":               gl.AccountBankOperating,
	}
	for tt, wantOffset := range uiTypes {
		resUI, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
			TenantID:       tenantID,
			FromEntityID:   "ent_uz",
			ToEntityID:     "ent_kz",
			TransferType:   tt,
			AmountMinor:    5000,
			Currency:       "USD",
			PeriodID:       "2026-11",
			IdempotencyKey: "ic-ui-type-" + tt,
			CreatedBy:      "treasurer-1",
		})
		if err != nil {
			t.Fatalf("CreateIntercompanyTransfer(%s): %v", tt, err)
		}
		dj, err := glSvc.GetEntry(ctx, tenantID, resUI.Transfer.DebitJournalID)
		if err != nil {
			t.Fatalf("GetEntry(%s): %v", tt, err)
		}
		if dj.Postings[1].AccountCode != wantOffset {
			t.Fatalf("%s offset account = %q, want %q", tt, dj.Postings[1].AccountCode, wantOffset)
		}
	}
}

func TestFxRevaluation_FailClosed_RoundingDelta_And_Incremental_Unit(t *testing.T) {
	ctx := context.Background()
	svc, meRepo, glSvc, fxRepo, _ := newUnitMultientityStack()
	tenantID := "ten_me_fx"

	_, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_uz", Name: "UZ Entity", CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	t0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 11, 30, 23, 59, 0, 0, time.UTC)

	// 1. Fail closed when any required FX rate is missing (CHF has no rate).
	meRepo.SeedForeignPositions(tenantID, []multientity.FxRevalPosition{
		{
			SourceType: "AR_INVOICE", SourceID: "inv-chf-1", AccountCode: gl.AccountAccountsReceivable,
			Direction: "ASSET", Currency: "CHF", ForeignAmountMinor: 5000, OccurredAt: t0,
		},
	})
	if _, err := svc.RunFxRevaluation(ctx, multientity.FxRevalInput{
		TenantID:       tenantID,
		EntityID:       "ent_uz",
		PeriodID:       "2026-11",
		BaseCurrency:   "UZS",
		AsOf:           t1.Format(time.RFC3339),
		IdempotencyKey: "fxr-missing",
	}); !errors.Is(err, multientity.ErrFxRateMissing) {
		t.Fatalf("missing FX rate err = %v, want ErrFxRateMissing", err)
	}

	// 2. Rounding delta test with fractional rate 12800.4 (1_280_040_000_000 / 1e8):
	// Two 1-cent USD assets carrying at 12500 each (sum carrying = 25000):
	// - Per item: round(1 * 12800.4) = 12800 -> item gain = +300 each, sumItemPnl = +600
	// - Aggregate: round(2 * 12800.4 = 25600.8) = 25601 -> aggPnl = +601
	// - Rounding residual absorbed into FX:ROUNDING = DR 1, FX:UNREALIZED_GAIN_LOSS = CR 601, AR DR 300, CASH DR 300.
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_usd_uzs_open", BaseCurrency: "USD", QuoteCurrency: "UZS",
		RateScaled: 12500 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0, Source: "CBU",
	}); err != nil {
		t.Fatal(err)
	}
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_usd_uzs_close", BaseCurrency: "USD", QuoteCurrency: "UZS",
		RateScaled: 1280040000000, Scale: fxrates.DefaultScale, EffectiveAt: t1, Source: "CBU",
	}); err != nil {
		t.Fatal(err)
	}

	meRepo.SeedForeignPositions(tenantID, []multientity.FxRevalPosition{
		{
			SourceType: "AR_INVOICE", SourceID: "inv-usd-1", AccountCode: gl.AccountAccountsReceivable,
			Direction: "ASSET", Currency: "USD", ForeignAmountMinor: 1,
			HasHistoricalFunctional: true, CarryingFunctionalMinor: 12500, OccurredAt: t0,
		},
		{
			SourceType: "CASH_BANK", SourceID: "cash-usd-2", AccountCode: gl.AccountCashOffice,
			Direction: "ASSET", Currency: "USD", ForeignAmountMinor: 1,
			HasHistoricalFunctional: true, CarryingFunctionalMinor: 12500, OccurredAt: t0,
		},
	})

	revalRes, err := svc.RunFxRevaluation(ctx, multientity.FxRevalInput{
		TenantID:       tenantID,
		EntityID:       "ent_uz",
		PeriodID:       "2026-11",
		BaseCurrency:   "UZS",
		AsOf:           t1.Format(time.RFC3339),
		IdempotencyKey: "fxr-round-1",
		CreatedBy:      "cfo-1",
	})
	if err != nil {
		t.Fatalf("RunFxRevaluation: %v", err)
	}
	if revalRes.AlreadyPosted || revalRes.PositionCount != 2 {
		t.Fatalf("unexpected reval result metadata: %+v", revalRes)
	}
	if revalRes.Run.UnrealizedGainLossMinor != 601 || revalRes.Run.RoundingDeltaMinor != -1 || revalRes.Run.JournalEntryID == "" {
		t.Fatalf("unexpected reval run totals: %+v", revalRes.Run)
	}

	je, err := glSvc.GetEntry(ctx, tenantID, revalRes.Run.JournalEntryID)
	if err != nil {
		t.Fatalf("GetEntry(fx reval journal): %v", err)
	}
	var hasRounding, hasUnrealized bool
	var sumDr, sumCr int64
	for _, p := range je.Postings {
		if p.Direction == gl.DirectionDebit {
			sumDr += p.AmountMinor
		} else {
			sumCr += p.AmountMinor
		}
		if p.AccountCode == gl.AccountFxRounding && p.AmountMinor == 1 && p.Direction == gl.DirectionDebit {
			hasRounding = true
		}
		if p.AccountCode == gl.AccountFxUnrealizedGainLoss && p.AmountMinor == 601 && p.Direction == gl.DirectionCredit {
			hasUnrealized = true
		}
	}
	if sumDr != sumCr || sumDr != 601 || !hasRounding || !hasUnrealized {
		t.Fatalf("unexpected FX revaluation postings: sumDr=%d sumCr=%d rounding=%v unrealized=%v postings=%+v",
			sumDr, sumCr, hasRounding, hasUnrealized, je.Postings)
	}

	// Idempotent replay returns AlreadyPosted=true.
	replayRes, err := svc.RunFxRevaluation(ctx, multientity.FxRevalInput{
		TenantID:       tenantID,
		EntityID:       "ent_uz",
		PeriodID:       "2026-11",
		BaseCurrency:   "UZS",
		AsOf:           t1.Format(time.RFC3339),
		IdempotencyKey: "fxr-round-1",
	})
	if err != nil || !replayRes.AlreadyPosted || replayRes.Run.RunID != revalRes.Run.RunID {
		t.Fatalf("reval idempotent replay got (%+v, %v), want AlreadyPosted=true", replayRes, err)
	}

	// GL journal is reversible.
	if _, err := glSvc.ReverseEntry(ctx, tenantID, je.EntryID, "fxr-rev-1", "month-end auto reversal", "cfo-1"); err != nil {
		t.Fatalf("ReverseEntry on FX revaluation journal: %v", err)
	}
}

func TestConsolidation_MultiCurrency_Eliminations_SubsetEntities_Unit(t *testing.T) {
	ctx := context.Background()
	svc, _, glSvc, fxRepo, _ := newUnitMultientityStack()
	tenantID := "ten_me_cons"

	_, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_uz", Name: "UZ Parent", TIN: "111222333", CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: "ent_kz", Name: "KZ Sub", TIN: "444555666", CountryCode: "KZ", BaseCurrency: "KZT", ParentEntityID: "ent_uz",
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	// 1 KZT = 25 UZS (in 1e8 fixed point: 25 * 1e8).
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_kzt_uzs", BaseCurrency: "KZT", QuoteCurrency: "UZS",
		RateScaled: 25 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: now.Add(-time.Hour), Source: "NBK",
	}); err != nil {
		t.Fatal(err)
	}
	// 1 USD = 12,800 UZS.
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_usd_uzs", BaseCurrency: "USD", QuoteCurrency: "UZS",
		RateScaled: 12800 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: now.Add(-time.Hour), Source: "CBU",
	}); err != nil {
		t.Fatal(err)
	}
	// 1 UZS -> KZT direct inverse test also covered via AllowInverse; add UZS->KZT rate = 1/25 = 0.04 (4_000_000 / 1e8).
	if err := fxRepo.Upsert(ctx, fxrates.Rate{
		RateID: "r_uzs_kzt", BaseCurrency: "UZS", QuoteCurrency: "KZT",
		RateScaled: 4_000_000, Scale: fxrates.DefaultScale, EffectiveAt: now.Add(-time.Hour), Source: "NBK",
	}); err != nil {
		t.Fatal(err)
	}

	// External sales: UZ 500,000 UZS; KZ 10,000 KZT (= 250,000 UZS). Total consolidated sales = 750,000 UZS.
	if _, err := glSvc.PostJournal(ctx, gl.PostInput{
		TenantID: tenantID, EntityID: "ent_uz", PeriodID: "2026-11",
		SourceType: "MANUAL", SourceID: "sale-uz", Currency: "UZS", CountryPack: "UZ",
		IdempotencyKey: "cons-sale-uz", OccurredAt: now,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 500000},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 500000},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := glSvc.PostJournal(ctx, gl.PostInput{
		TenantID: tenantID, EntityID: "ent_kz", PeriodID: "2026-11",
		SourceType: "MANUAL", SourceID: "sale-kz", Currency: "KZT", CountryPack: "KZ",
		IdempotencyKey: "cons-sale-kz", OccurredAt: now,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 10000},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 10000},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Intercompany transfer of 200,000 UZS (= 8,000 KZT) from ent_uz to ent_kz.
	if _, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID: tenantID, FromEntityID: "ent_uz", ToEntityID: "ent_kz",
		TransferType: "CASH", AmountMinor: 200000, Currency: "UZS", PeriodID: "2026-11", IdempotencyKey: "cons-ic-1",
	}); err != nil {
		t.Fatalf("CreateIntercompanyTransfer: %v", err)
	}

	// Full consolidation in UZS: IC:DUE_FROM (200k) and IC:DUE_TO (200k) fully eliminated!
	// Consolidated Debits = 500k (AR_UZ) + 250k (AR_KZ) + 200k (BANK_KZ) = 950k; BANK_UZ CR 200k? Wait:
	// Postings before elim:
	//   UZ: AR DR 500k, SALES CR 500k, IC:DUE_FROM DR 200k, BANK CR 200k (sum DR=700k, CR=700k)
	//   KZ: AR DR 10k (250k), SALES CR 10k (250k), BANK DR 8k (200k), IC:DUE_TO CR 8k (200k) (sum DR=450k, CR=450k)
	// Combined pre-elim DR = 1,150k, CR = 1,150k. Eliminated IC = 200k. Post-elim DR = 950k, CR = 950k!
	rep, err := svc.Consolidate(ctx, tenantID, "2026-11", "UZS", nil)
	if err != nil {
		t.Fatalf("Consolidate(UZS): %v", err)
	}
	if !rep.Balanced || rep.TotalDebitsMinor != 950000 || rep.TotalCreditsMinor != 950000 ||
		rep.TotalEliminationsMinor != 200000 || rep.ICVarianceMinor != 0 || len(rep.Entities) != 2 {
		t.Fatalf("unexpected UZS consolidation report: %+v", rep)
	}
	if len(rep.Eliminations) != 1 || len(rep.FxRatesApplied) == 0 {
		t.Fatalf("expected 1 elimination and FX rate applied, got elim=%+v rates=%+v", rep.Eliminations, rep.FxRatesApplied)
	}
	byCode := make(map[string]multientity.ConsolidatedAccountLine, len(rep.Accounts))
	for _, a := range rep.Accounts {
		byCode[a.AccountCode] = a
	}
	if byCode[gl.AccountIntercompanyDueFrom].NetMinor != 0 || byCode[gl.AccountIntercompanyDueFrom].EliminatedMinor != 200000 {
		t.Fatalf("IC:DUE_FROM should be completely eliminated: %+v", byCode[gl.AccountIntercompanyDueFrom])
	}
	if byCode[gl.AccountIntercompanyDueTo].NetMinor != 0 || byCode[gl.AccountIntercompanyDueTo].EliminatedMinor != 200000 {
		t.Fatalf("IC:DUE_TO should be completely eliminated: %+v", byCode[gl.AccountIntercompanyDueTo])
	}
	if byCode[gl.AccountSalesRevenue].NetMinor != 750000 {
		t.Fatalf("consolidated REV:SALES net = %d, want 750000", byCode[gl.AccountSalesRevenue].NetMinor)
	}

	// Subset consolidation (ent_uz only): IC:DUE_FROM with external ent_kz MUST NOT be eliminated!
	subRep, err := svc.Consolidate(ctx, tenantID, "2026-11", "UZS", []string{"ent_uz"})
	if err != nil {
		t.Fatalf("Consolidate(subset ent_uz): %v", err)
	}
	if !subRep.Balanced || subRep.TotalEliminationsMinor != 0 || subRep.ICVarianceMinor != 200000 || len(subRep.Entities) != 1 {
		t.Fatalf("unexpected subset consolidation report: %+v", subRep)
	}
}

func TestLiveSpanner_MultiEntity_IC_FxRevaluation_Consolidation(t *testing.T) {
	client := openMultientitySpannerClient(t)
	ctx := context.Background()

	tenantID := fmt.Sprintf("sup_me_live_%d", time.Now().UnixNano())
	rateEUROpenID := fmt.Sprintf("fx_eur_uzs_open_%d", time.Now().UnixNano())
	rateEURCloseID := fmt.Sprintf("fx_eur_uzs_close_%d", time.Now().UnixNano())
	rateKZTUZSID := fmt.Sprintf("fx_kzt_uzs_%d", time.Now().UnixNano())
	rateUZSKZTID := fmt.Sprintf("fx_uzs_kzt_%d", time.Now().UnixNano())
	cleanupMultientityTenant(t, client, tenantID, []string{rateEUROpenID, rateEURCloseID, rateKZTUZSID, rateUZSKZTID})

	glSvc := gl.NewService(client)
	fxRepo := fxrates.NewSpannerRepository(client)
	fxSvc := fxrates.NewService(fxRepo)
	svc := multientity.NewService(client, glSvc, fxSvc, fxRepo)
	svc.SetAuditRecorder(spannerAuditAdapter{svc: audit.NewService(client)})
	glSvc.SetLegalEntityOwnershipChecker(svc)

	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("gl.SeedDefaults: %v", err)
	}

	// 1. Create parent (UZ/UZS) & subsidiary (KZ/KZT) legal entities.
	uzEnt, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: tenantID + "_uz", Name: "Tashkent Holding LLC", TIN: "305001001",
		CountryCode: "UZ", BaseCurrency: "UZS", IsParent: true, ActorID: "cfo-live",
	})
	if err != nil {
		t.Fatalf("CreateEntity(UZ): %v", err)
	}
	kzEnt, err := svc.CreateEntity(ctx, multientity.CreateEntityInput{
		TenantID: tenantID, EntityID: tenantID + "_kz", Name: "Astana Sub LLP", TIN: "180140012345",
		CountryCode: "KZ", BaseCurrency: "KZT", ParentEntityID: uzEnt.EntityID, ActorID: "cfo-live",
	})
	if err != nil {
		t.Fatalf("CreateEntity(KZ): %v", err)
	}

	// Verify LegalEntities.Created + ontology.object.changed outbox + AuditEvents.
	newTIN := "305001099"
	if _, err := svc.PatchEntity(ctx, multientity.PatchEntityInput{
		TenantID: tenantID, EntityID: uzEnt.EntityID, TIN: &newTIN, ActorID: "cfo-live",
	}); err != nil {
		t.Fatalf("PatchEntity(UZ): %v", err)
	}

	t0 := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 16, 18, 0, 0, 0, time.UTC)

	// Seed deterministic test FX rates (EUR->UZS revalues from 13,500 to 13,700; 1 KZT = 25 UZS).
	for _, r := range []fxrates.Rate{
		{RateID: rateEUROpenID, BaseCurrency: "EUR", QuoteCurrency: "UZS", RateScaled: 13500 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0.Add(-time.Minute), Source: "CBU-TEST"},
		{RateID: rateEURCloseID, BaseCurrency: "EUR", QuoteCurrency: "UZS", RateScaled: 13700 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t1, Source: "CBU-TEST"},
		{RateID: rateKZTUZSID, BaseCurrency: "KZT", QuoteCurrency: "UZS", RateScaled: 25 * fxrates.DefaultScale, Scale: fxrates.DefaultScale, EffectiveAt: t0.Add(-time.Hour), Source: "NBK-TEST"},
		{RateID: rateUZSKZTID, BaseCurrency: "UZS", QuoteCurrency: "KZT", RateScaled: 4_000_000, Scale: fxrates.DefaultScale, EffectiveAt: t0.Add(-time.Hour), Source: "NBK-TEST"},
	} {
		if err := fxRepo.Upsert(ctx, r); err != nil {
			t.Fatalf("fxRepo.Upsert(%s): %v", r.RateID, err)
		}
	}

	// 2. Post open EUR receivable (100.00 EUR = 10,000 minor units) on uzEnt at opening rate 13,500 (= 135,000,000 UZS minor units).
	if _, err := glSvc.PostJournal(ctx, gl.PostInput{
		TenantID: tenantID, EntityID: uzEnt.EntityID, PeriodID: "2026-11",
		SourceType: "MANUAL", SourceID: "eur-sale-1", Memo: "Export EUR invoice",
		Currency: "EUR", CountryPack: "UZ", IdempotencyKey: tenantID + "-eur-1", OccurredAt: t0,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 10000, Currency: "EUR", FunctionalAmountMinor: 135000000, FxRateID: rateEUROpenID},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 10000, Currency: "EUR", FunctionalAmountMinor: 135000000, FxRateID: rateEUROpenID},
		},
	}); err != nil {
		t.Fatalf("PostJournal(EUR AR): %v", err)
	}

	// 3. Run FX revaluation at t1 (rate 13,700 -> +2,000,000 UZS unrealized gain).
	fx1, err := svc.RunFxRevaluation(ctx, multientity.FxRevalInput{
		TenantID: tenantID, EntityID: uzEnt.EntityID, PeriodID: "2026-11", BaseCurrency: "UZS",
		AsOf: t1.Format(time.RFC3339), IdempotencyKey: tenantID + "-fx1", CreatedBy: "cfo-live",
	})
	if err != nil {
		t.Fatalf("RunFxRevaluation(t1): %v", err)
	}
	if fx1.Run.UnrealizedGainLossMinor != 2000000 || fx1.Run.JournalEntryID == "" || fx1.PositionCount != 1 {
		t.Fatalf("unexpected live fx1 result: %+v", fx1)
	}

	// Run second incremental FX revaluation at t2 with same closing rate -> carrying now equals revalued (PriorRevalFunctionalMinor applied), PnL == 0!
	fx2, err := svc.RunFxRevaluation(ctx, multientity.FxRevalInput{
		TenantID: tenantID, EntityID: uzEnt.EntityID, PeriodID: "2026-11", BaseCurrency: "UZS",
		AsOf: t2.Format(time.RFC3339), IdempotencyKey: tenantID + "-fx2", CreatedBy: "cfo-live",
	})
	if err != nil {
		t.Fatalf("RunFxRevaluation(t2): %v", err)
	}
	if fx2.Run.UnrealizedGainLossMinor != 0 || fx2.Run.JournalEntryID != "" {
		t.Fatalf("expected 0 incremental PnL on unchanged rate rerun, got: %+v", fx2)
	}

	// 4. Fail closed on missing FX rate (XDR unseeded currency) & CLOSED period block.
	if _, err := svc.Consolidate(ctx, tenantID, "2026-11", "XDR", nil); !errors.Is(err, multientity.ErrFxRateMissing) {
		t.Fatalf("Consolidate missing XDR rate err = %v, want ErrFxRateMissing", err)
	}

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Periods", map[string]any{
			"EntityId":  tenantID,
			"PeriodId":  "2026-10",
			"Status":    "CLOSED",
			"Timezone":  "Asia/Tashkent",
			"CreatedAt": spanner.CommitTimestamp,
			"UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed closed Period 2026-10: %v", err)
	}
	if _, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID: tenantID, FromEntityID: uzEnt.EntityID, ToEntityID: kzEnt.EntityID,
		TransferType: "CASH", AmountMinor: 100000, Currency: "UZS", PeriodID: "2026-10", IdempotencyKey: tenantID + "-closed-ic",
	}); !errors.Is(err, multientity.ErrPeriodClosed) {
		t.Fatalf("IC transfer into CLOSED period err = %v, want ErrPeriodClosed", err)
	}

	// 5. Intercompany transfer in OPEN period 2026-11 + KZ external sale.
	if _, err := glSvc.PostJournal(ctx, gl.PostInput{
		TenantID: tenantID, EntityID: kzEnt.EntityID, PeriodID: "2026-11",
		SourceType: "MANUAL", SourceID: "kz-rev-1", Memo: "KZ sale",
		Currency: "KZT", CountryPack: "KZ", IdempotencyKey: tenantID + "-kz-rev-1", OccurredAt: t1,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountAccountsReceivable, Direction: gl.DirectionDebit, AmountMinor: 40000},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 40000},
		},
	}); err != nil {
		t.Fatalf("PostJournal(KZ): %v", err)
	}

	icRes, err := svc.CreateIntercompanyTransfer(ctx, multientity.CreateTransferInput{
		TenantID: tenantID, FromEntityID: uzEnt.EntityID, ToEntityID: kzEnt.EntityID,
		TransferType: "CASH", AmountMinor: 1000000, Currency: "UZS", PeriodID: "2026-11",
		IdempotencyKey: tenantID + "-ic-live-1", CreatedBy: "cfo-live",
	})
	if err != nil {
		t.Fatalf("CreateIntercompanyTransfer(live): %v", err)
	}
	if icRes.Transfer.DebitJournalID == "" || icRes.Transfer.CreditJournalID == "" {
		t.Fatalf("missing IC journal IDs: %+v", icRes)
	}

	// 6. Verify GL hash chain across all entries (EUR AR + FX reval + KZ sale + both IC legs = 5).
	chain, err := gl.VerifyEntityHashChain(ctx, client, tenantID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain: %v", err)
	}
	if !chain.Valid || chain.EntriesCount != 5 {
		t.Fatalf("unexpected GL hash chain: %+v", chain)
	}

	// 7. Consolidation in UZS across both entities eliminates the 1,000,000 UZS IC transfer.
	cons, err := svc.Consolidate(ctx, tenantID, "2026-11", "UZS", nil)
	if err != nil {
		t.Fatalf("Consolidate(live UZS): %v", err)
	}
	if !cons.Balanced || cons.TotalEliminationsMinor != 1000000 || cons.ICVarianceMinor != 0 {
		t.Fatalf("unexpected live consolidation: %+v", cons)
	}

	// 8. Assert OutboxEvents & AuditEvents persisted atomically.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	var outboxTypes []string
	obIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT AggregateType, Payload FROM OutboxEvents WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": tenantID},
	})
	for {
		row, rErr := obIter.Next()
		if rErr == iterator.Done {
			break
		}
		if rErr != nil {
			obIter.Stop()
			t.Fatalf("query OutboxEvents: %v", rErr)
		}
		var aggType string
		var payload []byte
		if cErr := row.Columns(&aggType, &payload); cErr != nil {
			obIter.Stop()
			t.Fatalf("scan OutboxEvents: %v", cErr)
		}
		var env struct {
			Type string `json:"type"`
		}
		if jErr := json.Unmarshal(payload, &env); jErr != nil {
			obIter.Stop()
			t.Fatalf("unmarshal OutboxEvents payload for %s: %v", aggType, jErr)
		}
		outboxTypes = append(outboxTypes, env.Type)
	}
	obIter.Stop()

	joinedOB := strings.Join(outboxTypes, ",")
	for _, want := range []string{"entity.intercompany.posted", "fx.revaluation.posted", "ontology.object.changed"} {
		if !strings.Contains(joinedOB, want) {
			t.Fatalf("missing outbox event type %q in %v", want, outboxTypes)
		}
	}

	var auditCount int64
	if err := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM AuditEvents WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tenantID},
	}).Do(func(r *spanner.Row) error {
		return r.Column(0, &auditCount)
	}); err != nil {
		t.Fatalf("count AuditEvents: %v", err)
	}
	if auditCount < 5 {
		t.Fatalf("AuditEvents count = %d, want >= 5 (2 creates + 1 patch + 1 ic + 1 fx reval)", auditCount)
	}

	// Verify RatesJSON serialized integer BigInt rate cleanly.
	var ratesApplied []multientity.FxRateApplied
	if err := json.Unmarshal([]byte(fx1.Run.RatesJSON), &ratesApplied); err != nil || len(ratesApplied) != 1 || ratesApplied[0].Scale != fxrates.DefaultScale {
		t.Fatalf("unexpected fx1.Run.RatesJSON: %q err=%v", fx1.Run.RatesJSON, err)
	}
}
