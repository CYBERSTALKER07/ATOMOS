package multientity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// MemoryRepository implements Repository in-memory for unit tests and harness fallback.
type MemoryRepository struct {
	mu           sync.RWMutex
	glSvc        *gl.Service
	glMem        *gl.MemoryRepository
	entities     map[string]LegalEntity          // tenantID|entityID -> LegalEntity
	transfers    map[string]IntercompanyTransfer // tenantID|transferID -> IntercompanyTransfer
	transferIdem map[string]string               // tenantID|idem -> transferID
	revalRuns    map[string]FxRevaluationRun     // tenantID|runID -> FxRevaluationRun
	revalIdem    map[string]string               // tenantID|idem -> runID
	positions    map[string][]FxRevalPosition    // tenantID -> seeded open positions
}

// NewMemoryRepository constructs an in-memory multi-entity Repository.
func NewMemoryRepository(glSvc *gl.Service) *MemoryRepository {
	var glMem *gl.MemoryRepository
	if glSvc != nil {
		if mr, ok := glSvc.Repo().(*gl.MemoryRepository); ok {
			glMem = mr
		}
	}
	return &MemoryRepository{
		glSvc:        glSvc,
		glMem:        glMem,
		entities:     make(map[string]LegalEntity),
		transfers:    make(map[string]IntercompanyTransfer),
		transferIdem: make(map[string]string),
		revalRuns:    make(map[string]FxRevaluationRun),
		revalIdem:    make(map[string]string),
		positions:    make(map[string][]FxRevalPosition),
	}
}

// SeedForeignPositions injects open foreign-currency balances for isolated unit tests.
func (r *MemoryRepository) SeedForeignPositions(tenantID string, pos []FxRevalPosition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.positions[strings.TrimSpace(tenantID)] = append([]FxRevalPosition(nil), pos...)
}

func (r *MemoryRepository) ListEntities(_ context.Context, tenantID, statusFilter string) ([]LegalEntity, error) {
	tenantID = strings.TrimSpace(tenantID)
	sf := strings.ToUpper(strings.TrimSpace(statusFilter))
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LegalEntity, 0)
	for _, ent := range r.entities {
		if ent.TenantID != tenantID {
			continue
		}
		if sf != "" && strings.ToUpper(ent.Status) != sf {
			continue
		}
		out = append(out, ent)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsParent != out[j].IsParent {
			return out[i].IsParent
		}
		return out[i].EntityID < out[j].EntityID
	})
	return out, nil
}

func (r *MemoryRepository) GetEntity(_ context.Context, tenantID, entityID string) (LegalEntity, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ent, ok := r.entities[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(entityID)]
	return ent, ok, nil
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
	_, ok := r.entities[tenantID+"|"+entityID]
	return ok, nil
}

func (r *MemoryRepository) CreateEntity(ctx context.Context, ent LegalEntity, _ bool, auditRec AuditRecorder, auditIn AuditRecordInput) (LegalEntity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := ent.TenantID + "|" + ent.EntityID
	if _, exists := r.entities[key]; exists {
		return LegalEntity{}, fmt.Errorf("%w: %s", ErrEntityExists, ent.EntityID)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	ent.CreatedAt = now
	ent.UpdatedAt = now
	r.entities[key] = ent
	if r.glMem != nil {
		r.glMem.RegisterLegalEntityCurrency(ent.TenantID, ent.EntityID, ent.BaseCurrency)
	}
	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return ent, nil
}

func (r *MemoryRepository) PatchEntity(
	ctx context.Context,
	tenantID, entityID string,
	mutate func(cur LegalEntity) (LegalEntity, error),
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (LegalEntity, LegalEntity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimSpace(tenantID) + "|" + strings.TrimSpace(entityID)
	cur, ok := r.entities[key]
	if !ok {
		return LegalEntity{}, LegalEntity{}, fmt.Errorf("%w: %s", ErrEntityNotFound, entityID)
	}
	next, err := mutate(cur)
	if err != nil {
		return LegalEntity{}, LegalEntity{}, err
	}
	next.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	r.entities[key] = next
	if r.glMem != nil {
		r.glMem.RegisterLegalEntityCurrency(next.TenantID, next.EntityID, next.BaseCurrency)
	}
	if auditRec != nil {
		if auditIn.BeforeJSON == "" {
			if bb, bErr := json.Marshal(cur); bErr == nil {
				auditIn.BeforeJSON = string(bb)
			}
		}
		if ab, aErr := json.Marshal(next); aErr == nil {
			auditIn.AfterJSON = string(ab)
		}
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return cur, next, nil
}

func (r *MemoryRepository) ListTransfers(_ context.Context, tenantID, periodID, entityID string, limit int) ([]IntercompanyTransfer, error) {
	tenantID = strings.TrimSpace(tenantID)
	periodID = strings.TrimSpace(periodID)
	entityID = strings.TrimSpace(entityID)
	if limit <= 0 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]IntercompanyTransfer, 0)
	for _, tr := range r.transfers {
		if tr.TenantID != tenantID {
			continue
		}
		if periodID != "" && tr.PeriodID != periodID {
			continue
		}
		if entityID != "" && tr.FromEntityID != entityID && tr.ToEntityID != entityID {
			continue
		}
		out = append(out, tr)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) CreateTransferAtomic(
	ctx context.Context,
	tr IntercompanyTransfer,
	debitJE, creditJE gl.JournalEntryInput,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (IntercompanyTransfer, bool, error) {
	r.mu.Lock()
	if tr.IdempotencyKey != "" {
		if existingID, ok := r.transferIdem[tr.TenantID+"|"+tr.IdempotencyKey]; ok {
			existing := r.transfers[tr.TenantID+"|"+existingID]
			r.mu.Unlock()
			return existing, true, nil
		}
	}
	fromEnt, okFrom := r.entities[tr.TenantID+"|"+tr.FromEntityID]
	if !okFrom {
		r.mu.Unlock()
		return IntercompanyTransfer{}, false, fmt.Errorf("%w: from_entity_id %s", ErrEntityNotFound, tr.FromEntityID)
	}
	if strings.ToUpper(fromEnt.Status) != EntityStatusActive {
		r.mu.Unlock()
		return IntercompanyTransfer{}, false, fmt.Errorf("%w: %s", ErrEntityInactive, tr.FromEntityID)
	}
	toEnt, okTo := r.entities[tr.TenantID+"|"+tr.ToEntityID]
	if !okTo {
		r.mu.Unlock()
		return IntercompanyTransfer{}, false, fmt.Errorf("%w: to_entity_id %s", ErrEntityNotFound, tr.ToEntityID)
	}
	if strings.ToUpper(toEnt.Status) != EntityStatusActive {
		r.mu.Unlock()
		return IntercompanyTransfer{}, false, fmt.Errorf("%w: %s", ErrEntityInactive, tr.ToEntityID)
	}
	r.mu.Unlock()

	if r.glSvc != nil {
		drRes, err := r.glSvc.PostEntry(ctx, debitJE)
		if err != nil {
			return IntercompanyTransfer{}, false, err
		}
		crRes, err := r.glSvc.PostEntry(ctx, creditJE)
		if err != nil {
			return IntercompanyTransfer{}, false, err
		}
		tr.DebitJournalID = drRes.EntryID
		tr.CreditJournalID = crRes.EntryID
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	tr.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	r.transfers[tr.TenantID+"|"+tr.TransferID] = tr
	if tr.IdempotencyKey != "" {
		r.transferIdem[tr.TenantID+"|"+tr.IdempotencyKey] = tr.TransferID
	}
	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return tr, false, nil
}

func (r *MemoryRepository) ListFxRevaluationRuns(_ context.Context, tenantID, periodID, entityID string, limit int) ([]FxRevaluationRun, error) {
	tenantID = strings.TrimSpace(tenantID)
	periodID = strings.TrimSpace(periodID)
	entityID = strings.TrimSpace(entityID)
	if limit <= 0 {
		limit = 50
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]FxRevaluationRun, 0)
	for _, run := range r.revalRuns {
		if run.TenantID != tenantID {
			continue
		}
		if periodID != "" && run.PeriodID != periodID {
			continue
		}
		if entityID != "" && run.EntityID != entityID {
			continue
		}
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRepository) CollectOpenForeignCurrencyPositions(
	ctx context.Context,
	tenantID, entityID, baseCurrency string,
	asOf time.Time,
) ([]FxRevalPosition, error) {
	r.mu.RLock()
	seeded := append([]FxRevalPosition(nil), r.positions[strings.TrimSpace(tenantID)]...)
	r.mu.RUnlock()

	if r.glSvc != nil {
		journals, err := r.glSvc.ListJournals(ctx, gl.JournalFilter{
			TenantID:      tenantID,
			LegalEntityID: entityID,
			Limit:         500,
		})
		if err == nil {
			type posKey struct {
				acct string
				curr string
			}
			byKey := make(map[posKey]*FxRevalPosition)
			priorReval := make(map[posKey]int64)
			for _, je := range journals {
				if !asOf.IsZero() && je.OccurredAt.After(asOf) {
					continue
				}
				for _, p := range je.Postings {
					if !isRevalMonetaryAccount(p.AccountCode) {
						continue
					}
					if p.PartyType == "FX_CURRENCY" && p.PartyID != "" && strings.EqualFold(p.Currency, baseCurrency) {
						k := posKey{acct: p.AccountCode, curr: strings.ToUpper(p.PartyID)}
						priorReval[k] += signedAccountAmount(p.AccountCode, p.Direction, p.FunctionalAmountMinor)
						continue
					}
					curr := strings.ToUpper(strings.TrimSpace(p.Currency))
					if curr == "" || curr == strings.ToUpper(strings.TrimSpace(baseCurrency)) {
						continue
					}
					k := posKey{acct: p.AccountCode, curr: curr}
					pos, ok := byKey[k]
					if !ok {
						dir := "ASSET"
						if !gl.IsDebitNormalClass(accountClassFor(p.AccountCode)) {
							dir = "LIABILITY"
						}
						pos = &FxRevalPosition{
							SourceType:  "GL_BALANCE",
							SourceID:    p.AccountCode + ":" + curr,
							AccountCode: p.AccountCode,
							Direction:   dir,
							Currency:    curr,
							OccurredAt:  je.OccurredAt,
						}
						byKey[k] = pos
					}
					pos.ForeignAmountMinor += signedAccountAmount(p.AccountCode, p.Direction, p.AmountMinor)
					if p.FxRateID != "" || p.FunctionalAmountMinor != p.AmountMinor {
						pos.CarryingFunctionalMinor += signedAccountAmount(p.AccountCode, p.Direction, p.FunctionalAmountMinor)
						pos.HasHistoricalFunctional = true
					}
					if je.OccurredAt.After(pos.OccurredAt) {
						pos.OccurredAt = je.OccurredAt
					}
				}
			}
			for k, pos := range byKey {
				if pos.ForeignAmountMinor != 0 {
					if prior, ok := priorReval[k]; ok && prior != 0 {
						pos.PriorRevalFunctionalMinor += prior
						delete(priorReval, k)
					}
					seeded = append(seeded, *pos)
				}
			}
			for i := range seeded {
				k := posKey{acct: seeded[i].AccountCode, curr: seeded[i].Currency}
				if prior, ok := priorReval[k]; ok && prior != 0 {
					seeded[i].PriorRevalFunctionalMinor += prior
					delete(priorReval, k)
				}
			}
		}
	}
	return seeded, nil
}

func (r *MemoryRepository) CreateFxRevaluationRunAtomic(
	ctx context.Context,
	run FxRevaluationRun,
	journalIn *gl.JournalEntryInput,
	auditRec AuditRecorder,
	auditIn AuditRecordInput,
) (FxRevaluationRun, bool, error) {
	r.mu.Lock()
	if run.IdempotencyKey != "" {
		if existingID, ok := r.revalIdem[run.TenantID+"|"+run.IdempotencyKey]; ok {
			existing := r.revalRuns[run.TenantID+"|"+existingID]
			r.mu.Unlock()
			return existing, true, nil
		}
	}
	r.mu.Unlock()

	if journalIn != nil && r.glSvc != nil {
		jeRes, err := r.glSvc.PostEntry(ctx, *journalIn)
		if err != nil {
			return FxRevaluationRun{}, false, err
		}
		run.JournalEntryID = jeRes.EntryID
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	run.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	r.revalRuns[run.TenantID+"|"+run.RunID] = run
	if run.IdempotencyKey != "" {
		r.revalIdem[run.TenantID+"|"+run.IdempotencyKey] = run.RunID
	}
	if auditRec != nil {
		_ = auditRec.RecordEvent(ctx, auditIn)
	}
	return run, false, nil
}

func (r *MemoryRepository) ReadConsolidationSnapshot(ctx context.Context, tenantID, periodID string, entityIDs []string) (ConsolidationSnapshot, error) {
	entities, err := r.ListEntities(ctx, tenantID, EntityStatusActive)
	if err != nil {
		return ConsolidationSnapshot{}, err
	}
	if len(entityIDs) > 0 {
		allowed := make(map[string]struct{}, len(entityIDs))
		for _, id := range entityIDs {
			allowed[id] = struct{}{}
		}
		filtered := make([]LegalEntity, 0, len(entityIDs))
		for _, e := range entities {
			if _, ok := allowed[e.EntityID]; ok {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}
	tbs := make(map[string]gl.TrialBalanceReport, len(entities))
	if r.glSvc != nil {
		for _, e := range entities {
			tb, err := r.glSvc.TrialBalanceForEntity(ctx, tenantID, periodID, e.EntityID)
			if err != nil {
				return ConsolidationSnapshot{}, err
			}
			tbs[e.EntityID] = tb
		}
	}
	transfers, err := r.ListTransfers(ctx, tenantID, periodID, "", 500)
	if err != nil {
		return ConsolidationSnapshot{}, err
	}
	return ConsolidationSnapshot{
		Entities:      entities,
		TrialBalances: tbs,
		Transfers:     transfers,
	}, nil
}
