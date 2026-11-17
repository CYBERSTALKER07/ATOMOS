package gl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryRepository implements Repository and MultiEntityJournalRepository in-memory for unit tests.
type MemoryRepository struct {
	mu           sync.RWMutex
	entriesByKey map[string]JournalEntryRecord
	order        []string
	idemIndex    map[string]string
	reversed     map[string]string
	ownedLegal   map[string]bool
	legalCur     map[string]string
}

// NewMemoryRepository constructs an in-memory GL Repository supporting multi-entity queries.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		entriesByKey: make(map[string]JournalEntryRecord),
		idemIndex:    make(map[string]string),
		reversed:     make(map[string]string),
		ownedLegal:   make(map[string]bool),
		legalCur:     make(map[string]string),
	}
}

// RegisterLegalEntity marks (tenantID, entityID) as owned for unit tests.
func (r *MemoryRepository) RegisterLegalEntity(tenantID, entityID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ownedLegal[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(entityID)] = true
}

// RegisterLegalEntityCurrency marks (tenantID, entityID) as owned with baseCurrency.
func (r *MemoryRepository) RegisterLegalEntityCurrency(tenantID, entityID, baseCurrency string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := strings.TrimSpace(tenantID) + "|" + strings.TrimSpace(entityID)
	r.ownedLegal[k] = true
	if cur := strings.ToUpper(strings.TrimSpace(baseCurrency)); cur != "" {
		r.legalCur[k] = cur
	}
}

func (r *MemoryRepository) IsLegalEntityOwned(_ context.Context, tenantID, entityID string) (bool, error) {
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	if tenantID == "" || entityID == "" {
		return false, nil
	}
	if tenantID == entityID {
		return true, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ownedLegal[tenantID+"|"+entityID], nil
}

func (r *MemoryRepository) PostEntry(ctx context.Context, input JournalEntryInput) (JournalEntryResult, error) {
	cp := input
	td, tc, err := ValidateJournalInput(ctx, &cp)
	if err != nil {
		return JournalEntryResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ik := cp.EntityID + "|" + cp.IdempotencyKey
	if existingID, ok := r.idemIndex[ik]; ok {
		rec := r.entriesByKey[cp.EntityID+"|"+existingID]
		return JournalEntryResult{
			EntryID:       rec.EntryID,
			EntityID:      rec.EntityID,
			LegalEntityID: rec.LegalEntityID,
			PeriodID:      rec.PeriodID,
			PrevHash:      rec.PrevHash,
			Hash:          rec.EntryHash,
			AlreadyPosted: true,
			TotalDebit:    td,
			TotalCredit:   tc,
			LineCount:     len(rec.Postings),
		}, nil
	}

	for _, p := range cp.Postings {
		if _, isSys := LookupSystemAccount(p.AccountCode); !isSys {
			return JournalEntryResult{}, fmt.Errorf("%w: %s (entity=%s)", ErrUnknownAccount, p.AccountCode, cp.EntityID)
		}
	}

	if cp.ReversesEntryID != "" {
		if _, exists := r.entriesByKey[cp.EntityID+"|"+cp.ReversesEntryID]; !exists {
			return JournalEntryResult{}, fmt.Errorf("%w: target %s", ErrEntryNotFound, cp.ReversesEntryID)
		}
		if r.reversed[cp.EntityID+"|"+cp.ReversesEntryID] != "" {
			return JournalEntryResult{}, fmt.Errorf("%w: %s", ErrAlreadyReversed, cp.ReversesEntryID)
		}
	}

	prevHash := GenesisHash
	for i := len(r.order) - 1; i >= 0; i-- {
		if rec, ok := r.entriesByKey[r.order[i]]; ok && rec.EntityID == cp.EntityID {
			prevHash = rec.EntryHash
			break
		}
	}

	postings := make([]JournalPostingRecord, 0, len(cp.Postings))
	canonPostings := make([]CanonicalPosting, 0, len(cp.Postings))
	for idx, p := range cp.Postings {
		lineNum := int64(idx + 1)
		postings = append(postings, JournalPostingRecord{
			EntityID:              cp.EntityID,
			EntryID:               cp.EntryID,
			Line:                  lineNum,
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
		canonPostings = append(canonPostings, CanonicalPosting{
			Line:                  lineNum,
			AccountCode:           p.AccountCode,
			Direction:             string(p.Direction),
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

	canonBytes, err := CanonicalEntryBytes(CanonicalEntryPayload{
		EntityID:         cp.EntityID,
		EntryID:          cp.EntryID,
		LegalEntityID:    cp.LegalEntityID,
		PeriodID:         cp.PeriodID,
		OriginalPeriodID: cp.OriginalPeriodID,
		SourceEventID:    cp.SourceEventID,
		SourceType:       cp.SourceType,
		SourceID:         cp.SourceID,
		OrderID:          cp.OrderID,
		Memo:             cp.Memo,
		Currency:         cp.Currency,
		CountryPack:      cp.CountryPack,
		ReversesEntryID:  cp.ReversesEntryID,
		IdempotencyKey:   cp.IdempotencyKey,
		PostedBy:         cp.PostedBy,
		OccurredAt:       cp.OccurredAt.UTC().Format(time.RFC3339Nano),
		Postings:         canonPostings,
	})
	if err != nil {
		return JournalEntryResult{}, err
	}
	entryHash := ComputeEntryHash(prevHash, canonBytes)

	rec := JournalEntryRecord{
		EntityID:        cp.EntityID,
		EntryID:         cp.EntryID,
		LegalEntityID:   cp.LegalEntityID,
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
		PrevHash:        prevHash,
		EntryHash:       entryHash,
		PostedBy:        cp.PostedBy,
		OccurredAt:      cp.OccurredAt,
		PostedAt:        time.Now().UTC(),
		Postings:        postings,
	}
	key := cp.EntityID + "|" + cp.EntryID
	r.entriesByKey[key] = rec
	r.order = append(r.order, key)
	r.idemIndex[ik] = cp.EntryID
	if cp.ReversesEntryID != "" {
		r.reversed[cp.EntityID+"|"+cp.ReversesEntryID] = cp.EntryID
	}

	return JournalEntryResult{
		EntryID:       rec.EntryID,
		EntityID:      rec.EntityID,
		LegalEntityID: rec.LegalEntityID,
		PeriodID:      rec.PeriodID,
		PrevHash:      rec.PrevHash,
		Hash:          rec.EntryHash,
		AlreadyPosted: false,
		TotalDebit:    td,
		TotalCredit:   tc,
		LineCount:     len(rec.Postings),
	}, nil
}

func (r *MemoryRepository) PostEvent(ctx context.Context, eventType string, ev RuleEvalContext) (JournalEntryResult, error) {
	in, err := EvaluateEventToJournalInput(ctx, DefaultPostingRules(ev.CountryPack), eventType, ev)
	if err != nil {
		return JournalEntryResult{}, err
	}
	return r.PostEntry(ctx, in)
}

func (r *MemoryRepository) ReverseEntry(ctx context.Context, entityID, entryID, idempotencyKey, reason, actor string) (JournalEntryResult, error) {
	r.mu.RLock()
	orig, ok := r.entriesByKey[entityID+"|"+entryID]
	revID := r.reversed[entityID+"|"+entryID]
	r.mu.RUnlock()
	if !ok {
		return JournalEntryResult{}, ErrEntryNotFound
	}
	if revID != "" {
		return JournalEntryResult{}, ErrAlreadyReversed
	}
	invPostings := make([]PostingInput, 0, len(orig.Postings))
	for _, p := range orig.Postings {
		dir := DirectionCredit
		if p.Direction == DirectionCredit {
			dir = DirectionDebit
		}
		invPostings = append(invPostings, PostingInput{
			AccountCode:           p.AccountCode,
			Direction:             dir,
			AmountMinor:           p.AmountMinor,
			Currency:              p.Currency,
			FunctionalAmountMinor: p.FunctionalAmountMinor,
			FxRateID:              p.FxRateID,
			PartyType:             p.PartyType,
			PartyID:               p.PartyID,
			WarehouseID:           p.WarehouseID,
			ProductID:             p.ProductID,
			LotID:                 p.LotID,
			Description:           coalesceNonEmpty(p.Description, "Reversal of "+entryID),
		})
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "reverse:" + entryID
	}
	return r.PostEntry(ctx, JournalEntryInput{
		EntityID:        entityID,
		LegalEntityID:   orig.LegalEntityID,
		SourceType:      "REVERSAL",
		SourceID:        entryID,
		OrderID:         orig.OrderID,
		Memo:            coalesceNonEmpty(strings.TrimSpace(reason), "Reversal of journal entry "+entryID),
		Currency:        orig.Currency,
		CountryPack:     orig.CountryPack,
		ReversesEntryID: entryID,
		IdempotencyKey:  idempotencyKey,
		PostedBy:        actor,
		OccurredAt:      time.Now().UTC(),
		Postings:        invPostings,
	})
}

func (r *MemoryRepository) GetEntry(_ context.Context, entityID, entryID string) (JournalEntryRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.entriesByKey[entityID+"|"+entryID]
	if !ok {
		return JournalEntryRecord{}, ErrEntryNotFound
	}
	return rec, nil
}

func (r *MemoryRepository) GetTrialBalance(ctx context.Context, entityID, periodID string) (TrialBalanceReport, error) {
	return r.GetTrialBalanceFiltered(ctx, entityID, periodID, "")
}

func (r *MemoryRepository) GetTrialBalanceFiltered(ctx context.Context, tenantID, periodID, legalEntityID string) (TrialBalanceReport, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if strings.TrimSpace(periodID) == "" {
		periodID = ComputePeriodID(ctx, tenantID, "", time.Now().UTC())
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	type tbKey struct {
		code string
		curr string
	}
	lines := make(map[tbKey]*TrialBalanceLine)
	var totalDr, totalCr int64
	entityBaseCur := ""
	if legalEntityID != "" {
		entityBaseCur = r.legalCur[tenantID+"|"+legalEntityID]
	}
	reportCur := entityBaseCur
	for _, rec := range r.entriesByKey {
		if rec.EntityID != tenantID || rec.PeriodID != periodID {
			continue
		}
		if legalEntityID != "" && rec.LegalEntityID != legalEntityID {
			continue
		}
		for _, p := range rec.Postings {
			effCurr := strings.ToUpper(strings.TrimSpace(p.Currency))
			amt := p.AmountMinor
			if strings.TrimSpace(p.FxRateID) != "" {
				if p.FunctionalAmountMinor > 0 {
					amt = p.FunctionalAmountMinor
				}
				if entityBaseCur != "" {
					effCurr = entityBaseCur
				}
			} else if p.FunctionalAmountMinor != 0 && p.FunctionalAmountMinor == p.AmountMinor {
				amt = p.FunctionalAmountMinor
			}
			if reportCur == "" {
				reportCur = effCurr
			}
			k := tbKey{code: p.AccountCode, curr: effCurr}
			line, ok := lines[k]
			if !ok {
				def, _ := LookupSystemAccount(p.AccountCode)
				line = &TrialBalanceLine{
					AccountCode: p.AccountCode,
					AccountName: def.Name,
					Class:       def.Class,
					Currency:    effCurr,
				}
				lines[k] = line
			}
			if p.Direction == DirectionDebit {
				line.DebitMinor += amt
				totalDr += amt
			} else {
				line.CreditMinor += amt
				totalCr += amt
			}
		}
	}
	out := make([]TrialBalanceLine, 0, len(lines))
	for _, l := range lines {
		if IsDebitNormalClass(l.Class) {
			l.NetMinor = l.DebitMinor - l.CreditMinor
		} else {
			l.NetMinor = l.CreditMinor - l.DebitMinor
		}
		out = append(out, *l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AccountCode != out[j].AccountCode {
			return out[i].AccountCode < out[j].AccountCode
		}
		return out[i].Currency < out[j].Currency
	})
	if reportCur == "" {
		reportCur = DefaultCurrencyForPack("")
	}
	reportEntity := tenantID
	if legalEntityID != "" {
		reportEntity = legalEntityID
	}
	rep := TrialBalanceReport{
		EntityID:          reportEntity,
		PeriodID:          periodID,
		Currency:          reportCur,
		TotalDebitsMinor:  totalDr,
		TotalCreditsMinor: totalCr,
		Balanced:          totalDr == totalCr,
		Accounts:          out,
	}
	if legalEntityID != "" {
		rep.TenantID = tenantID
		rep.LegalEntityID = legalEntityID
	}
	return rep, nil
}

func (r *MemoryRepository) ListJournals(_ context.Context, filter JournalFilter) ([]JournalEntryRecord, error) {
	tenantID := strings.TrimSpace(filter.TenantID)
	legalEntityID := strings.TrimSpace(filter.LegalEntityID)
	periodID := strings.TrimSpace(filter.PeriodID)
	sourceType := strings.ToUpper(strings.TrimSpace(filter.SourceType))
	accountCode := strings.ToUpper(strings.TrimSpace(filter.AccountCode))
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]JournalEntryRecord, 0)
	for i := len(r.order) - 1; i >= 0; i-- {
		rec := r.entriesByKey[r.order[i]]
		if rec.EntityID != tenantID {
			continue
		}
		if legalEntityID != "" && rec.LegalEntityID != legalEntityID {
			continue
		}
		if periodID != "" && rec.PeriodID != periodID {
			continue
		}
		if sourceType != "" && rec.SourceType != sourceType {
			continue
		}
		if accountCode != "" {
			matched := false
			for _, p := range rec.Postings {
				if p.AccountCode == accountCode {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, rec)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) GetAccountLedger(_ context.Context, entityID, accountCode, periodID string, limit int) ([]AccountLedgerRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acct := strings.ToUpper(strings.TrimSpace(accountCode))
	if limit <= 0 {
		limit = 50
	}
	var out []AccountLedgerRow
	for i := len(r.order) - 1; i >= 0; i-- {
		rec := r.entriesByKey[r.order[i]]
		if rec.EntityID != entityID || (periodID != "" && rec.PeriodID != periodID) {
			continue
		}
		for _, p := range rec.Postings {
			if p.AccountCode != acct {
				continue
			}
			out = append(out, AccountLedgerRow{
				EntryID:               rec.EntryID,
				PeriodID:              rec.PeriodID,
				Line:                  p.Line,
				AccountCode:           p.AccountCode,
				Direction:             p.Direction,
				AmountMinor:           p.AmountMinor,
				Currency:              p.Currency,
				FunctionalAmountMinor: p.FunctionalAmountMinor,
				FxRateID:              p.FxRateID,
				PartyType:             p.PartyType,
				PartyID:               p.PartyID,
				SourceType:            rec.SourceType,
				SourceID:              rec.SourceID,
				OrderID:               rec.OrderID,
				ReversesEntryID:       rec.ReversesEntryID,
				Memo:                  rec.Memo,
				Description:           p.Description,
				EntryHash:             rec.EntryHash,
				OccurredAt:            rec.OccurredAt,
				PostedAt:              rec.PostedAt,
			})
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (r *MemoryRepository) ListPostingRules(_ context.Context, countryPack, eventType string) ([]PostingRule, error) {
	var out []PostingRule
	for _, rule := range DefaultPostingRules(countryPack) {
		if eventType == "" || strings.EqualFold(rule.EventType, eventType) {
			out = append(out, rule)
		}
	}
	return out, nil
}

func (r *MemoryRepository) SeedDefaultAccounts(_ context.Context, _ string, countryPack, currency string) (int, error) {
	return len(DefaultChartOfAccounts(countryPack, currency)), nil
}

func (r *MemoryRepository) SeedPostingRules(_ context.Context, countryPack string) (int, error) {
	return len(DefaultPostingRules(countryPack)), nil
}

func (r *MemoryRepository) BackfillFromPaymentLedgerEntries(_ context.Context, entityID string, _ BackfillOptions) (BackfillReport, error) {
	return BackfillReport{EntityID: entityID}, nil
}
