package gl

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestCOA_Wave4ACodesAndAliases(t *testing.T) {
	coa := DefaultChartOfAccounts("UZ", "UZS")
	byCode := make(map[string]AccountDef, len(coa))
	for _, a := range coa {
		byCode[a.AccountCode] = a
	}

	required := []string{
		AccountFxRounding,
		AccountFxUnrealizedGainLoss,
		AccountIntercompanyDueFrom,
		AccountIntercompanyDueTo,
		AccountGRNI,
		AccountVatInput,
	}
	for _, code := range required {
		if _, ok := byCode[code]; !ok {
			t.Fatalf("missing required Wave 4A account %q in DefaultChartOfAccounts", code)
		}
	}

	if norm, pt, pid := NormalizeAccountCode("VAT:INPUT"); norm != AccountVatInput || pt != "" || pid != "" {
		t.Fatalf("NormalizeAccountCode(VAT:INPUT) = (%q, %q, %q), want (%q, '', '')", norm, pt, pid, AccountVatInput)
	}
	if norm, pt, pid := NormalizeAccountCode("IC:DUE_FROM:ent-2"); norm != AccountIntercompanyDueFrom || pt != "ENTITY" || pid != "ent-2" {
		t.Fatalf("NormalizeAccountCode(IC:DUE_FROM:ent-2) = (%q, %q, %q), want (%q, ENTITY, ent-2)", norm, pt, pid, AccountIntercompanyDueFrom)
	}
	if norm, pt, pid := NormalizeAccountCode("IC:DUE_TO:ent-1"); norm != AccountIntercompanyDueTo || pt != "ENTITY" || pid != "ent-1" {
		t.Fatalf("NormalizeAccountCode(IC:DUE_TO:ent-1) = (%q, %q, %q), want (%q, ENTITY, ent-1)", norm, pt, pid, AccountIntercompanyDueTo)
	}
}

func TestMemoryRepository_PostJournal_EntityFiltering_And_Reversal(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryRepository()
	svc := NewServiceWithRepository(mem)

	tenantID := "ten_gl_mem_1"
	entA := "ent_a"
	entB := "ent_b"
	mem.RegisterLegalEntity(tenantID, entA)
	mem.RegisterLegalEntity(tenantID, entB)

	if err := svc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	if owned, err := svc.IsLegalEntityOwned(ctx, tenantID, entA); err != nil || !owned {
		t.Fatalf("IsLegalEntityOwned(entA) = (%v, %v), want (true, nil)", owned, err)
	}
	if owned, err := svc.IsLegalEntityOwned(ctx, tenantID, tenantID); err != nil || !owned {
		t.Fatalf("IsLegalEntityOwned(tenantID) = (%v, %v), want (true, nil)", owned, err)
	}
	if owned, err := svc.IsLegalEntityOwned(ctx, tenantID, "other_ent"); err != nil || owned {
		t.Fatalf("IsLegalEntityOwned(other_ent) = (%v, %v), want (false, nil)", owned, err)
	}

	occ := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	resA, err := svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		EntityID:       entA,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "src-a-1",
		Memo:           "Entity A sale",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "mem-idem-a-1",
		PostedBy:       "acct-1",
		OccurredAt:     occ,
		Postings: []PostingInput{
			{AccountCode: AccountAccountsReceivable, Direction: DirectionDebit, AmountMinor: 100000},
			{AccountCode: AccountSalesRevenue, Direction: DirectionCredit, AmountMinor: 100000},
		},
	})
	if err != nil {
		t.Fatalf("PostJournal(entA): %v", err)
	}
	if resA.LegalEntityID != entA {
		t.Fatalf("resA.LegalEntityID = %q, want %q", resA.LegalEntityID, entA)
	}

	// Idempotent replay retains LegalEntityID.
	resAReplay, err := svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		EntityID:       entA,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "src-a-1",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "mem-idem-a-1",
		OccurredAt:     occ,
		Postings: []PostingInput{
			{AccountCode: AccountAccountsReceivable, Direction: DirectionDebit, AmountMinor: 100000},
			{AccountCode: AccountSalesRevenue, Direction: DirectionCredit, AmountMinor: 100000},
		},
	})
	if err != nil || !resAReplay.AlreadyPosted || resAReplay.LegalEntityID != entA {
		t.Fatalf("replay got (%+v, %v), want AlreadyPosted=true, LegalEntityID=%q", resAReplay, err, entA)
	}

	_, err = svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		EntityID:       entB,
		PeriodID:       "2026-11",
		SourceType:     "INTERCOMPANY",
		SourceID:       "ic-b-1",
		Memo:           "Entity B IC funding",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "mem-idem-b-1",
		PostedBy:       "acct-1",
		OccurredAt:     occ,
		Postings: []PostingInput{
			{AccountCode: AccountBankOperating, Direction: DirectionDebit, AmountMinor: 250000},
			{AccountCode: AccountIntercompanyDueTo, Direction: DirectionCredit, AmountMinor: 250000},
		},
	})
	if err != nil {
		t.Fatalf("PostJournal(entB): %v", err)
	}

	_, err = svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "src-ten-1",
		Memo:           "Tenant shared entry",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: "mem-idem-ten-1",
		OccurredAt:     occ,
		Postings: []PostingInput{
			{AccountCode: AccountCashOffice, Direction: DirectionDebit, AmountMinor: 50000},
			{AccountCode: AccountCashOverShortRevenue, Direction: DirectionCredit, AmountMinor: 50000},
		},
	})
	if err != nil {
		t.Fatalf("PostJournal(tenant): %v", err)
	}

	tbA, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", entA)
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(entA): %v", err)
	}
	if !tbA.Balanced || tbA.TotalDebitsMinor != 100000 || tbA.TotalCreditsMinor != 100000 || tbA.LegalEntityID != entA {
		t.Fatalf("unexpected tbA: %+v", tbA)
	}

	tbB, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", entB)
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(entB): %v", err)
	}
	if !tbB.Balanced || tbB.TotalDebitsMinor != 250000 || tbB.TotalCreditsMinor != 250000 || tbB.LegalEntityID != entB {
		t.Fatalf("unexpected tbB: %+v", tbB)
	}

	tbAll, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", "")
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(all): %v", err)
	}
	if !tbAll.Balanced || tbAll.TotalDebitsMinor != 400000 {
		t.Fatalf("unexpected tbAll: %+v", tbAll)
	}

	journalsA, err := svc.ListJournals(ctx, JournalFilter{TenantID: tenantID, LegalEntityID: entA, PeriodID: "2026-11"})
	if err != nil {
		t.Fatalf("ListJournals(entA): %v", err)
	}
	if len(journalsA) != 1 || journalsA[0].EntryID != resA.EntryID || journalsA[0].LegalEntityID != entA {
		t.Fatalf("unexpected journalsA: %+v", journalsA)
	}

	journalsIC, err := svc.ListJournals(ctx, JournalFilter{TenantID: tenantID, SourceType: "INTERCOMPANY", AccountCode: AccountIntercompanyDueTo})
	if err != nil {
		t.Fatalf("ListJournals(IC): %v", err)
	}
	if len(journalsIC) != 1 || journalsIC[0].LegalEntityID != entB {
		t.Fatalf("unexpected journalsIC: %+v", journalsIC)
	}

	// Reversal preserves LegalEntityID and zeros entA TB.
	revRes, err := svc.ReverseEntry(ctx, tenantID, resA.EntryID, "mem-rev-a-1", "test reversal", "acct-2")
	if err != nil {
		t.Fatalf("ReverseEntry(entA): %v", err)
	}
	if revRes.LegalEntityID != entA {
		t.Fatalf("reversal LegalEntityID = %q, want %q", revRes.LegalEntityID, entA)
	}
	tbAAfterRev, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", entA)
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(entA after reverse): %v", err)
	}
	if !tbAAfterRev.Balanced || tbAAfterRev.TotalDebitsMinor != 200000 || tbAAfterRev.TotalCreditsMinor != 200000 {
		t.Fatalf("unexpected tbAAfterRev totals: %+v", tbAAfterRev)
	}
	for _, acct := range tbAAfterRev.Accounts {
		if acct.NetMinor != 0 {
			t.Fatalf("expected zero net after reversal for %s, got %d", acct.AccountCode, acct.NetMinor)
		}
	}
}

func TestLiveSpanner_PostJournal_LegalEntityFilter_HashChain_Reversal(t *testing.T) {
	client := openGLSpannerTestClient(t)
	ctx := context.Background()

	tenantID := uniqueGLEntity("sup_gl_me")
	cleanupGLEntityState(t, client, tenantID, nil)
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cctx, []*spanner.Mutation{
			spanner.Delete("LegalEntities", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		})
	})

	entUZ := tenantID + "_uz"
	entKZ := tenantID + "_kz"
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("LegalEntities", map[string]any{
			"TenantId":     tenantID,
			"EntityId":     entUZ,
			"Name":         "Tashkent HQ LLC",
			"CountryCode":  "UZ",
			"BaseCurrency": "UZS",
			"IsParent":     true,
			"Status":       "ACTIVE",
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
		spanner.InsertMap("LegalEntities", map[string]any{
			"TenantId":     tenantID,
			"EntityId":     entKZ,
			"Name":         "Almaty Sub LLP",
			"CountryCode":  "KZ",
			"BaseCurrency": "KZT",
			"IsParent":     false,
			"Status":       "ACTIVE",
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed LegalEntities: %v", err)
	}

	svc := NewService(client)
	if err := svc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	if owned, err := svc.IsLegalEntityOwned(ctx, tenantID, entUZ); err != nil || !owned {
		t.Fatalf("IsLegalEntityOwned(entUZ) = (%v, %v), want (true, nil)", owned, err)
	}
	if owned, err := svc.IsLegalEntityOwned(ctx, tenantID, "foreign_ent"); err != nil || owned {
		t.Fatalf("IsLegalEntityOwned(foreign_ent) = (%v, %v), want (false, nil)", owned, err)
	}

	occ := time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)
	resUZ, err := svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		EntityID:       entUZ,
		PeriodID:       "2026-11",
		SourceType:     "MANUAL",
		SourceID:       "uz-src-1",
		Memo:           "UZ legal entity entry",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: tenantID + "-uz-1",
		PostedBy:       "cfo-1",
		OccurredAt:     occ,
		Postings: []PostingInput{
			{AccountCode: AccountAccountsReceivable, Direction: DirectionDebit, AmountMinor: 300000},
			{AccountCode: AccountSalesRevenue, Direction: DirectionCredit, AmountMinor: 300000},
		},
	})
	if err != nil {
		t.Fatalf("PostJournal(entUZ): %v", err)
	}
	if resUZ.LegalEntityID != entUZ {
		t.Fatalf("resUZ.LegalEntityID = %q, want %q", resUZ.LegalEntityID, entUZ)
	}

	resKZ, err := svc.PostJournal(ctx, PostInput{
		TenantID:       tenantID,
		EntityID:       entKZ,
		PeriodID:       "2026-11",
		SourceType:     "INTERCOMPANY",
		SourceID:       "kz-src-1",
		Memo:           "KZ legal entity entry",
		Currency:       "UZS",
		CountryPack:    "UZ",
		IdempotencyKey: tenantID + "-kz-1",
		PostedBy:       "cfo-1",
		OccurredAt:     occ.Add(time.Minute),
		Postings: []PostingInput{
			{AccountCode: AccountBankOperating, Direction: DirectionDebit, AmountMinor: 150000},
			{AccountCode: AccountIntercompanyDueTo, Direction: DirectionCredit, AmountMinor: 150000},
		},
	})
	if err != nil {
		t.Fatalf("PostJournal(entKZ): %v", err)
	}
	if resKZ.LegalEntityID != entKZ {
		t.Fatalf("resKZ.LegalEntityID = %q, want %q", resKZ.LegalEntityID, entKZ)
	}

	// Verify hash chain includes LegalEntityId canonically.
	chain, err := VerifyEntityHashChain(ctx, client, tenantID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain: %v", err)
	}
	if !chain.Valid || chain.EntriesCount != 2 {
		t.Fatalf("unexpected hash chain verification: %+v", chain)
	}

	// Filtered TrialBalance per legal entity.
	tbUZ, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", entUZ)
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(entUZ): %v", err)
	}
	if !tbUZ.Balanced || tbUZ.TotalDebitsMinor != 300000 || len(tbUZ.Accounts) != 2 {
		t.Fatalf("unexpected tbUZ: %+v", tbUZ)
	}

	tbKZ, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", entKZ)
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(entKZ): %v", err)
	}
	if !tbKZ.Balanced || tbKZ.TotalDebitsMinor != 150000 || len(tbKZ.Accounts) != 2 {
		t.Fatalf("unexpected tbKZ: %+v", tbKZ)
	}

	tbAll, err := svc.TrialBalanceForEntity(ctx, tenantID, "2026-11", "")
	if err != nil {
		t.Fatalf("TrialBalanceForEntity(all): %v", err)
	}
	if !tbAll.Balanced || tbAll.TotalDebitsMinor != 450000 {
		t.Fatalf("unexpected tbAll: %+v", tbAll)
	}

	// ListJournals filter.
	listUZ, err := svc.ListJournals(ctx, JournalFilter{TenantID: tenantID, LegalEntityID: entUZ, PeriodID: "2026-11"})
	if err != nil {
		t.Fatalf("ListJournals(entUZ): %v", err)
	}
	if len(listUZ) != 1 || listUZ[0].EntryID != resUZ.EntryID || listUZ[0].LegalEntityID != entUZ {
		t.Fatalf("unexpected listUZ: %+v", listUZ)
	}

	// Reverse entUZ entry and verify LegalEntityID preservation + chain integrity.
	revUZ, err := svc.ReverseEntry(ctx, tenantID, resUZ.EntryID, tenantID+"-rev-uz-1", "legal entity correction", "cfo-2")
	if err != nil {
		t.Fatalf("ReverseEntry(resUZ): %v", err)
	}
	if revUZ.LegalEntityID != entUZ {
		t.Fatalf("reversal LegalEntityID = %q, want %q", revUZ.LegalEntityID, entUZ)
	}

	chainAfter, err := VerifyEntityHashChain(ctx, client, tenantID)
	if err != nil {
		t.Fatalf("VerifyEntityHashChain(after reverse): %v", err)
	}
	if !chainAfter.Valid || chainAfter.EntriesCount != 3 {
		t.Fatalf("unexpected hash chain after reverse: %+v", chainAfter)
	}
}
