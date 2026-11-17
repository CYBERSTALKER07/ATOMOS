package e2ep0

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// memoryGLRepo provides an in-memory gl.Repository for NewTestHarness(t) when no
// Spanner emulator client is supplied, enabling opaque-box HTTP GL route & RBAC tests.
type memoryGLRepo struct {
	mu           sync.RWMutex
	entriesByKey map[string]gl.JournalEntryRecord // entityID|entryID
	idemIndex    map[string]string                // entityID|idempotencyKey -> entryID
	reversed     map[string]string                // entityID|origEntryID -> revEntryID
}

func newMemoryGLRepo() *memoryGLRepo {
	return &memoryGLRepo{
		entriesByKey: make(map[string]gl.JournalEntryRecord),
		idemIndex:    make(map[string]string),
		reversed:     make(map[string]string),
	}
}

func (r *memoryGLRepo) PostEntry(ctx context.Context, input gl.JournalEntryInput) (gl.JournalEntryResult, error) {
	cp := input
	td, tc, err := gl.ValidateJournalInput(ctx, &cp)
	if err != nil {
		return gl.JournalEntryResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ik := cp.EntityID + "|" + cp.IdempotencyKey
	if existingID, ok := r.idemIndex[ik]; ok {
		rec := r.entriesByKey[cp.EntityID+"|"+existingID]
		return gl.JournalEntryResult{
			EntryID:       rec.EntryID,
			EntityID:      rec.EntityID,
			PeriodID:      rec.PeriodID,
			PrevHash:      rec.PrevHash,
			Hash:          rec.EntryHash,
			AlreadyPosted: true,
			TotalDebit:    td,
			TotalCredit:   tc,
			LineCount:     len(rec.Postings),
		}, nil
	}

	postings := make([]gl.JournalPostingRecord, 0, len(cp.Postings))
	for idx, p := range cp.Postings {
		postings = append(postings, gl.JournalPostingRecord{
			EntityID:              cp.EntityID,
			EntryID:               cp.EntryID,
			Line:                  int64(idx + 1),
			AccountCode:           p.AccountCode,
			Direction:             p.Direction,
			AmountMinor:           p.AmountMinor,
			Currency:              p.Currency,
			FunctionalAmountMinor: p.FunctionalAmountMinor,
			FxRateID:              p.FxRateID,
			PartyType:             p.PartyType,
			PartyID:               p.PartyID,
			WarehouseID:           p.WarehouseID,
			ProductID:             p.ProductID,
			LotID:                 p.LotID,
			Description:           p.Description,
		})
	}

	rec := gl.JournalEntryRecord{
		EntityID:        cp.EntityID,
		EntryID:         cp.EntryID,
		PeriodID:        cp.PeriodID,
		SourceEventID:   cp.SourceEventID,
		SourceType:      cp.SourceType,
		SourceID:        cp.SourceID,
		OrderID:         cp.OrderID,
		Memo:            cp.Memo,
		Currency:        cp.Currency,
		CountryPack:     cp.CountryPack,
		ReversesEntryID: cp.ReversesEntryID,
		IdempotencyKey:  cp.IdempotencyKey,
		PrevHash:        gl.GenesisHash,
		EntryHash:       "mem-" + cp.EntryID,
		PostedBy:        cp.PostedBy,
		OccurredAt:      cp.OccurredAt,
		PostedAt:        time.Now().UTC(),
		Postings:        postings,
	}
	r.entriesByKey[cp.EntityID+"|"+cp.EntryID] = rec
	r.idemIndex[ik] = cp.EntryID
	if cp.ReversesEntryID != "" {
		r.reversed[cp.EntityID+"|"+cp.ReversesEntryID] = cp.EntryID
	}

	return gl.JournalEntryResult{
		EntryID:       rec.EntryID,
		EntityID:      rec.EntityID,
		PeriodID:      rec.PeriodID,
		PrevHash:      rec.PrevHash,
		Hash:          rec.EntryHash,
		AlreadyPosted: false,
		TotalDebit:    td,
		TotalCredit:   tc,
		LineCount:     len(rec.Postings),
	}, nil
}

func (r *memoryGLRepo) PostEvent(ctx context.Context, eventType string, ev gl.RuleEvalContext) (gl.JournalEntryResult, error) {
	in, err := gl.EvaluateEventToJournalInput(ctx, gl.DefaultPostingRules(ev.CountryPack), eventType, ev)
	if err != nil {
		return gl.JournalEntryResult{}, err
	}
	return r.PostEntry(ctx, in)
}

func (r *memoryGLRepo) ReverseEntry(ctx context.Context, entityID, entryID, idempotencyKey, reason, actor string) (gl.JournalEntryResult, error) {
	r.mu.RLock()
	orig, ok := r.entriesByKey[entityID+"|"+entryID]
	revID := r.reversed[entityID+"|"+entryID]
	r.mu.RUnlock()
	if !ok {
		return gl.JournalEntryResult{}, gl.ErrEntryNotFound
	}
	if revID != "" {
		return gl.JournalEntryResult{}, gl.ErrAlreadyReversed
	}
	invPostings := make([]gl.PostingInput, 0, len(orig.Postings))
	for _, p := range orig.Postings {
		dir := gl.DirectionCredit
		if p.Direction == gl.DirectionCredit {
			dir = gl.DirectionDebit
		}
		invPostings = append(invPostings, gl.PostingInput{
			AccountCode:           p.AccountCode,
			Direction:             dir,
			AmountMinor:           p.AmountMinor,
			Currency:              p.Currency,
			FunctionalAmountMinor: p.FunctionalAmountMinor,
			FxRateID:              p.FxRateID,
			PartyType:             p.PartyType,
			PartyID:               p.PartyID,
			Description:           "Reversal: " + reason,
		})
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "rev:" + entityID + ":" + entryID
	}
	return r.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:        entityID,
		SourceType:      "REVERSAL",
		SourceID:        entryID,
		OrderID:         orig.OrderID,
		Memo:            fmt.Sprintf("Reversal of %s: %s", entryID, reason),
		Currency:        orig.Currency,
		CountryPack:     orig.CountryPack,
		ReversesEntryID: entryID,
		IdempotencyKey:  idempotencyKey,
		PostedBy:        actor,
		OccurredAt:      time.Now().UTC(),
		Postings:        invPostings,
	})
}

func (r *memoryGLRepo) GetEntry(_ context.Context, entityID, entryID string) (gl.JournalEntryRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.entriesByKey[entityID+"|"+entryID]
	if !ok {
		return gl.JournalEntryRecord{}, gl.ErrEntryNotFound
	}
	return rec, nil
}

func (r *memoryGLRepo) GetTrialBalance(ctx context.Context, entityID, periodID string) (gl.TrialBalanceReport, error) {
	if strings.TrimSpace(periodID) == "" {
		periodID = gl.ComputePeriodID(ctx, entityID, "", time.Now().UTC())
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	lines := make(map[string]*gl.TrialBalanceLine)
	var totalDr, totalCr int64
	reportCur := ""
	for _, rec := range r.entriesByKey {
		if rec.EntityID != entityID || rec.PeriodID != periodID {
			continue
		}
		for _, p := range rec.Postings {
			if reportCur == "" {
				reportCur = p.Currency
			}
			line, ok := lines[p.AccountCode]
			if !ok {
				def, _ := gl.LookupSystemAccount(p.AccountCode)
				line = &gl.TrialBalanceLine{
					AccountCode: p.AccountCode,
					AccountName: def.Name,
					Class:       def.Class,
					Currency:    p.Currency,
				}
				lines[p.AccountCode] = line
			}
			if p.Direction == gl.DirectionDebit {
				line.DebitMinor += p.AmountMinor
				totalDr += p.AmountMinor
			} else {
				line.CreditMinor += p.AmountMinor
				totalCr += p.AmountMinor
			}
		}
	}
	out := make([]gl.TrialBalanceLine, 0, len(lines))
	for _, l := range lines {
		if gl.IsDebitNormalClass(l.Class) {
			l.NetMinor = l.DebitMinor - l.CreditMinor
		} else {
			l.NetMinor = l.CreditMinor - l.DebitMinor
		}
		out = append(out, *l)
	}
	return gl.TrialBalanceReport{
		EntityID:          entityID,
		PeriodID:          periodID,
		Currency:          reportCur,
		TotalDebitsMinor:  totalDr,
		TotalCreditsMinor: totalCr,
		Balanced:          totalDr == totalCr,
		Accounts:          out,
	}, nil
}

func (r *memoryGLRepo) GetAccountLedger(_ context.Context, entityID, accountCode, periodID string, limit int) ([]gl.AccountLedgerRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acct := strings.ToUpper(strings.TrimSpace(accountCode))
	var out []gl.AccountLedgerRow
	for _, rec := range r.entriesByKey {
		if rec.EntityID != entityID || (periodID != "" && rec.PeriodID != periodID) {
			continue
		}
		for _, p := range rec.Postings {
			if p.AccountCode != acct {
				continue
			}
			out = append(out, gl.AccountLedgerRow{
				EntryID:               rec.EntryID,
				PeriodID:              rec.PeriodID,
				Line:                  p.Line,
				AccountCode:           p.AccountCode,
				Direction:             p.Direction,
				AmountMinor:           p.AmountMinor,
				Currency:              p.Currency,
				FunctionalAmountMinor: p.FunctionalAmountMinor,
				PartyType:             p.PartyType,
				PartyID:               p.PartyID,
				SourceType:            rec.SourceType,
				SourceID:              rec.SourceID,
				OrderID:               rec.OrderID,
				EntryHash:             rec.EntryHash,
				OccurredAt:            rec.OccurredAt,
				PostedAt:              rec.PostedAt,
			})
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (r *memoryGLRepo) ListPostingRules(_ context.Context, countryPack, eventType string) ([]gl.PostingRule, error) {
	var out []gl.PostingRule
	for _, rule := range gl.DefaultPostingRules(countryPack) {
		if eventType == "" || strings.EqualFold(rule.EventType, eventType) {
			out = append(out, rule)
		}
	}
	return out, nil
}

func (r *memoryGLRepo) SeedDefaultAccounts(_ context.Context, _ string, countryPack, currency string) (int, error) {
	return len(gl.DefaultChartOfAccounts(countryPack, currency)), nil
}

func (r *memoryGLRepo) SeedPostingRules(_ context.Context, countryPack string) (int, error) {
	return len(gl.DefaultPostingRules(countryPack)), nil
}

func (r *memoryGLRepo) BackfillFromPaymentLedgerEntries(_ context.Context, entityID string, _ gl.BackfillOptions) (gl.BackfillReport, error) {
	return gl.BackfillReport{EntityID: entityID}, nil
}
