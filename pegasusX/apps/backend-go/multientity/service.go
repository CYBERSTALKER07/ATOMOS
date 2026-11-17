package multientity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

var periodRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Service orchestrates multi-entity management, intercompany transfers,
// month-end FX revaluation, and consolidated financial reporting.
type Service struct {
	repo     Repository
	glSvc    *gl.Service
	fxSvc    *fxrates.Service
	fxRepo   fxrates.Repository
	auditRec AuditRecorder
}

// NewService constructs a Spanner-backed multi-entity Service.
func NewService(client *spanner.Client, glSvc *gl.Service, fxSvc *fxrates.Service, fxRepo fxrates.Repository) *Service {
	if client == nil {
		return &Service{glSvc: glSvc, fxSvc: fxSvc, fxRepo: fxRepo}
	}
	if fxRepo == nil {
		fxRepo = fxrates.NewSpannerRepository(client)
	}
	if fxSvc == nil && fxRepo != nil {
		fxSvc = fxrates.NewService(fxRepo)
	}
	if glSvc == nil {
		glSvc = gl.NewService(client)
	}
	return &Service{
		repo:   NewSpannerRepository(client),
		glSvc:  glSvc,
		fxSvc:  fxSvc,
		fxRepo: fxRepo,
	}
}

// NewServiceWithDeps constructs a Service with custom Repository and FX dependencies (for tests).
func NewServiceWithDeps(repo Repository, glSvc *gl.Service, fxSvc *fxrates.Service, fxRepo fxrates.Repository) *Service {
	if fxSvc == nil && fxRepo != nil {
		fxSvc = fxrates.NewService(fxRepo)
	}
	return &Service{
		repo:   repo,
		glSvc:  glSvc,
		fxSvc:  fxSvc,
		fxRepo: fxRepo,
	}
}

// SetAuditRecorder attaches an AuditRecorder for durable AuditEvents recording.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.auditRec = rec
	}
}

// Repo returns the configured Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// IsLegalEntityOwned implements gl.LegalEntityOwnershipChecker so GL endpoints
// can validate ?entity= parameters against tenant LegalEntities.
func (s *Service) IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error) {
	if s == nil || s.repo == nil {
		return strings.TrimSpace(tenantID) != "" && strings.TrimSpace(tenantID) == strings.TrimSpace(entityID), nil
	}
	return s.repo.IsLegalEntityOwned(ctx, tenantID, entityID)
}

// ListEntities lists all legal entities for tenantID (optionally filtered by status).
func (s *Service) ListEntities(ctx context.Context, tenantID, statusFilter string) ([]LegalEntity, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.ListEntities(ctx, tenantID, statusFilter)
}

// CreateEntity validates and creates a new LegalEntity for tenantID, seeding default GL accounts if missing.
func (s *Service) CreateEntity(ctx context.Context, in CreateEntityInput) (LegalEntity, error) {
	if s == nil || s.repo == nil {
		return LegalEntity{}, ErrRepoUnavailable
	}
	tenantID := strings.TrimSpace(in.TenantID)
	name := strings.TrimSpace(in.Name)
	if tenantID == "" || name == "" {
		return LegalEntity{}, fmt.Errorf("%w: tenant_id and name are required", ErrInvalidInput)
	}
	if len(name) > 160 {
		return LegalEntity{}, fmt.Errorf("%w: name exceeds 160 chars", ErrInvalidInput)
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		entityID = deterministicEntityID(tenantID, name, in.TIN)
	}
	if len(entityID) > 64 {
		return LegalEntity{}, fmt.Errorf("%w: entity_id exceeds 64 chars", ErrInvalidInput)
	}

	country := auth.NormalizeCountryCode(in.CountryCode)
	if country == "" {
		country = auth.DefaultMarketCodeFromEnv()
	}
	if country == "" {
		country = "UZ"
	}
	baseCur := fxrates.NormalizeCurrency(in.BaseCurrency)
	if baseCur == "" {
		baseCur = gl.DefaultCurrencyForPack(country)
	}
	if len(baseCur) != 3 {
		return LegalEntity{}, fmt.Errorf("%w: invalid base_currency %q", ErrInvalidCurrency, in.BaseCurrency)
	}

	parentID := strings.TrimSpace(in.ParentEntityID)
	if parentID == entityID {
		return LegalEntity{}, fmt.Errorf("%w: parent_entity_id cannot equal entity_id", ErrInvalidInput)
	}

	ent := LegalEntity{
		TenantID:       tenantID,
		EntityID:       entityID,
		Name:           name,
		TIN:            strings.TrimSpace(in.TIN),
		CountryCode:    country,
		BaseCurrency:   baseCur,
		IsParent:       in.IsParent,
		ParentEntityID: parentID,
		Status:         EntityStatusActive,
	}

	afterBytes, _ := json.Marshal(ent)
	auditIn := AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    coalesceNonEmpty(in.ActorID, "system"),
		ActorType:  coalesceNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.Role,
		Action:     "org.entity.create",
		ObjectType: "LegalEntity",
		ObjectID:   entityID,
		AfterJSON:  string(afterBytes),
		TraceID:    in.TraceID,
	}
	return s.repo.CreateEntity(ctx, ent, true, s.auditRec, auditIn)
}

// PatchEntity updates mutable metadata on a LegalEntity and records AuditEvents + OutboxEvents.
func (s *Service) PatchEntity(ctx context.Context, in PatchEntityInput) (LegalEntity, error) {
	if s == nil || s.repo == nil {
		return LegalEntity{}, ErrRepoUnavailable
	}
	tenantID := strings.TrimSpace(in.TenantID)
	entityID := strings.TrimSpace(in.EntityID)
	if tenantID == "" || entityID == "" {
		return LegalEntity{}, fmt.Errorf("%w: tenant_id and entity_id are required", ErrInvalidInput)
	}

	mutate := func(cur LegalEntity) (LegalEntity, error) {
		next := cur
		if in.Name != nil {
			n := strings.TrimSpace(*in.Name)
			if n == "" || len(n) > 160 {
				return LegalEntity{}, fmt.Errorf("%w: valid name (1..160 chars) is required", ErrInvalidInput)
			}
			next.Name = n
		}
		if in.TIN != nil {
			t := strings.TrimSpace(*in.TIN)
			if len(t) > 32 {
				return LegalEntity{}, fmt.Errorf("%w: tin exceeds 32 chars", ErrInvalidInput)
			}
			next.TIN = t
		}
		if in.CountryCode != nil {
			cc := auth.NormalizeCountryCode(*in.CountryCode)
			if cc == "" {
				return LegalEntity{}, fmt.Errorf("%w: invalid country_code", ErrInvalidInput)
			}
			next.CountryCode = cc
		}
		if in.BaseCurrency != nil {
			bc := fxrates.NormalizeCurrency(*in.BaseCurrency)
			if len(bc) != 3 {
				return LegalEntity{}, fmt.Errorf("%w: invalid base_currency", ErrInvalidCurrency)
			}
			next.BaseCurrency = bc
		}
		if in.IsParent != nil {
			next.IsParent = *in.IsParent
		}
		if in.ParentEntityID != nil {
			pid := strings.TrimSpace(*in.ParentEntityID)
			if pid == entityID {
				return LegalEntity{}, fmt.Errorf("%w: parent_entity_id cannot equal entity_id", ErrInvalidInput)
			}
			next.ParentEntityID = pid
		}
		if in.Status != nil {
			st := strings.ToUpper(strings.TrimSpace(*in.Status))
			if st != EntityStatusActive && st != EntityStatusInactive {
				return LegalEntity{}, fmt.Errorf("%w: status must be ACTIVE or INACTIVE", ErrInvalidInput)
			}
			next.Status = st
		}
		return next, nil
	}

	afterSnap, _ := json.Marshal(in)
	auditIn := AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    coalesceNonEmpty(in.ActorID, "system"),
		ActorType:  coalesceNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.Role,
		Action:     "org.entity.update",
		ObjectType: "LegalEntity",
		ObjectID:   entityID,
		AfterJSON:  string(afterSnap),
		TraceID:    in.TraceID,
	}
	_, updated, err := s.repo.PatchEntity(ctx, tenantID, entityID, mutate, s.auditRec, auditIn)
	return updated, err
}

// ListTransfers returns intercompany transfers for tenantID filtered by period/entity.
func (s *Service) ListTransfers(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]IntercompanyTransfer, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if p := strings.TrimSpace(periodID); p != "" && !periodRegex.MatchString(p) {
		return nil, ErrInvalidPeriod
	}
	return s.repo.ListTransfers(ctx, tenantID, periodID, entityID, limit)
}

// CreateIntercompanyTransfer validates two distinct active entities and atomically posts
// balanced IC:DUE_FROM / IC:DUE_TO journals across both entities plus IntercompanyTransfers, Outbox, and AuditEvents.
func (s *Service) CreateIntercompanyTransfer(ctx context.Context, in CreateTransferInput) (CreateTransferResult, error) {
	if s == nil || s.repo == nil {
		return CreateTransferResult{}, ErrRepoUnavailable
	}
	tenantID := strings.TrimSpace(in.TenantID)
	fromID := strings.TrimSpace(in.FromEntityID)
	toID := strings.TrimSpace(in.ToEntityID)
	trType := strings.ToUpper(strings.TrimSpace(in.TransferType))
	idemKey := strings.TrimSpace(in.IdempotencyKey)

	if tenantID == "" || fromID == "" || toID == "" {
		return CreateTransferResult{}, fmt.Errorf("%w: tenant_id, from_entity_id, and to_entity_id are required", ErrInvalidInput)
	}
	if fromID == toID {
		return CreateTransferResult{}, ErrSameEntityTransfer
	}
	if in.AmountMinor <= 0 {
		return CreateTransferResult{}, fmt.Errorf("%w: amount_minor must be > 0", ErrInvalidInput)
	}
	if idemKey == "" {
		return CreateTransferResult{}, fmt.Errorf("%w: idempotency_key is required", ErrInvalidInput)
	}
	if trType == "" {
		trType = "CASH"
	}

	offsetAcct, err := resolveIntercompanyOffsetAccount(trType, in.OffsetAccount)
	if err != nil {
		return CreateTransferResult{}, err
	}

	fromEnt, okFrom, err := s.repo.GetEntity(ctx, tenantID, fromID)
	if err != nil {
		return CreateTransferResult{}, err
	}
	if !okFrom {
		return CreateTransferResult{}, fmt.Errorf("%w: from_entity_id %s", ErrEntityNotFound, fromID)
	}
	if strings.ToUpper(fromEnt.Status) != EntityStatusActive {
		return CreateTransferResult{}, fmt.Errorf("%w: %s", ErrEntityInactive, fromID)
	}

	toEnt, okTo, err := s.repo.GetEntity(ctx, tenantID, toID)
	if err != nil {
		return CreateTransferResult{}, err
	}
	if !okTo {
		return CreateTransferResult{}, fmt.Errorf("%w: to_entity_id %s", ErrEntityNotFound, toID)
	}
	if strings.ToUpper(toEnt.Status) != EntityStatusActive {
		return CreateTransferResult{}, fmt.Errorf("%w: %s", ErrEntityInactive, toID)
	}

	cur := fxrates.NormalizeCurrency(in.Currency)
	if cur == "" {
		cur = fxrates.NormalizeCurrency(fromEnt.BaseCurrency)
	}
	if len(cur) != 3 {
		return CreateTransferResult{}, fmt.Errorf("%w: invalid currency %q", ErrInvalidCurrency, in.Currency)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = gl.ComputePeriodID(ctx, tenantID, fromEnt.CountryCode, now)
	} else if !periodRegex.MatchString(periodID) {
		return CreateTransferResult{}, ErrInvalidPeriod
	}

	fromFuncMinor := in.AmountMinor
	fromFxRateID := ""
	if cur != fxrates.NormalizeCurrency(fromEnt.BaseCurrency) {
		rr, rErr := resolveRateAsOf(ctx, s.fxRepo, s.fxSvc, cur, fromEnt.BaseCurrency, now)
		if rErr != nil {
			return CreateTransferResult{}, rErr
		}
		conv, cErr := convertSignedMinor(in.AmountMinor, rr, cur, fromEnt.BaseCurrency)
		if cErr != nil {
			return CreateTransferResult{}, cErr
		}
		if conv <= 0 {
			return CreateTransferResult{}, fmt.Errorf("%w: converted functional amount <= 0", ErrInvalidInput)
		}
		fromFuncMinor = conv
		fromFxRateID = coalesceNonEmpty(rr.Rate.RateID, "fx:"+cur+":"+fromEnt.BaseCurrency)
	}

	toFuncMinor := in.AmountMinor
	toFxRateID := ""
	if cur != fxrates.NormalizeCurrency(toEnt.BaseCurrency) {
		rr, rErr := resolveRateAsOf(ctx, s.fxRepo, s.fxSvc, cur, toEnt.BaseCurrency, now)
		if rErr != nil {
			return CreateTransferResult{}, rErr
		}
		conv, cErr := convertSignedMinor(in.AmountMinor, rr, cur, toEnt.BaseCurrency)
		if cErr != nil {
			return CreateTransferResult{}, cErr
		}
		if conv <= 0 {
			return CreateTransferResult{}, fmt.Errorf("%w: converted functional amount <= 0", ErrInvalidInput)
		}
		toFuncMinor = conv
		toFxRateID = coalesceNonEmpty(rr.Rate.RateID, "fx:"+cur+":"+toEnt.BaseCurrency)
	}

	transferID := deterministicID("ict", tenantID, idemKey)
	actor := coalesceNonEmpty(in.CreatedBy, "system")
	memo := coalesceNonEmpty(strings.TrimSpace(in.Memo), fmt.Sprintf("Intercompany %s transfer %s -> %s", trType, fromID, toID))

	debitJE := gl.JournalEntryInput{
		EntityID:       tenantID,
		LegalEntityID:  fromID,
		PeriodID:       periodID,
		SourceType:     "INTERCOMPANY_TRANSFER",
		SourceID:       transferID,
		Memo:           memo + " (Due From)",
		Currency:       cur,
		CountryPack:    fromEnt.CountryCode,
		IdempotencyKey: "ic:" + idemKey + ":from",
		PostedBy:       actor,
		OccurredAt:     now,
		Postings: []gl.PostingInput{
			{
				AccountCode:           gl.AccountIntercompanyDueFrom,
				Direction:             gl.DirectionDebit,
				AmountMinor:           in.AmountMinor,
				Currency:              cur,
				FunctionalAmountMinor: fromFuncMinor,
				FxRateID:              fromFxRateID,
				PartyType:             "ENTITY",
				PartyID:               toID,
				Description:           fmt.Sprintf("IC due from %s (%s)", toID, trType),
			},
			{
				AccountCode:           offsetAcct,
				Direction:             gl.DirectionCredit,
				AmountMinor:           in.AmountMinor,
				Currency:              cur,
				FunctionalAmountMinor: fromFuncMinor,
				FxRateID:              fromFxRateID,
				PartyType:             "ENTITY",
				PartyID:               toID,
				Description:           fmt.Sprintf("IC transfer out to %s (%s)", toID, trType),
			},
		},
	}

	creditJE := gl.JournalEntryInput{
		EntityID:       tenantID,
		LegalEntityID:  toID,
		PeriodID:       periodID,
		SourceType:     "INTERCOMPANY_TRANSFER",
		SourceID:       transferID,
		Memo:           memo + " (Due To)",
		Currency:       cur,
		CountryPack:    toEnt.CountryCode,
		IdempotencyKey: "ic:" + idemKey + ":to",
		PostedBy:       actor,
		OccurredAt:     now,
		Postings: []gl.PostingInput{
			{
				AccountCode:           offsetAcct,
				Direction:             gl.DirectionDebit,
				AmountMinor:           in.AmountMinor,
				Currency:              cur,
				FunctionalAmountMinor: toFuncMinor,
				FxRateID:              toFxRateID,
				PartyType:             "ENTITY",
				PartyID:               fromID,
				Description:           fmt.Sprintf("IC transfer in from %s (%s)", fromID, trType),
			},
			{
				AccountCode:           gl.AccountIntercompanyDueTo,
				Direction:             gl.DirectionCredit,
				AmountMinor:           in.AmountMinor,
				Currency:              cur,
				FunctionalAmountMinor: toFuncMinor,
				FxRateID:              toFxRateID,
				PartyType:             "ENTITY",
				PartyID:               fromID,
				Description:           fmt.Sprintf("IC due to %s (%s)", fromID, trType),
			},
		},
	}

	tr := IntercompanyTransfer{
		TenantID:       tenantID,
		TransferID:     transferID,
		FromEntityID:   fromID,
		ToEntityID:     toID,
		TransferType:   trType,
		AmountMinor:    in.AmountMinor,
		Currency:       cur,
		PeriodID:       periodID,
		ReferenceType:  strings.TrimSpace(in.ReferenceType),
		ReferenceID:    strings.TrimSpace(in.ReferenceID),
		Status:         TransferStatusPosted,
		IdempotencyKey: idemKey,
		CreatedBy:      actor,
	}

	afterBytes, _ := json.Marshal(tr)
	auditIn := AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    actor,
		ActorType:  coalesceNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.Role,
		Action:     "finance.intercompany.transfer",
		ObjectType: "IntercompanyTransfer",
		ObjectID:   transferID,
		AfterJSON:  string(afterBytes),
		TraceID:    in.TraceID,
	}

	persisted, already, err := s.repo.CreateTransferAtomic(ctx, tr, debitJE, creditJE, s.auditRec, auditIn)
	if err != nil {
		if errors.Is(err, gl.ErrPeriodClosed) {
			return CreateTransferResult{}, fmt.Errorf("%w: %v", ErrPeriodClosed, err)
		}
		return CreateTransferResult{}, err
	}
	return CreateTransferResult{Transfer: persisted, AlreadyPosted: already}, nil
}

// ListFxRevaluationRuns lists FX revaluation runs for tenantID.
func (s *Service) ListFxRevaluationRuns(ctx context.Context, tenantID, periodID, entityID string, limit int) ([]FxRevaluationRun, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if p := strings.TrimSpace(periodID); p != "" && !periodRegex.MatchString(p) {
		return nil, ErrInvalidPeriod
	}
	return s.repo.ListFxRevaluationRuns(ctx, tenantID, periodID, entityID, limit)
}

// RunFxRevaluation computes unrealized FX gain/loss and FX:ROUNDING across open foreign-currency
// AR/AP/Cash positions in a ReadOnlyTransaction, failing closed if any required FX rate is missing,
// and atomically posts a reversible balanced GL journal + FxRevaluationRuns + Outbox + AuditEvents.
func (s *Service) RunFxRevaluation(ctx context.Context, in FxRevalInput) (FxRevalResult, error) {
	if s == nil || s.repo == nil {
		return FxRevalResult{}, ErrRepoUnavailable
	}
	tenantID := strings.TrimSpace(in.TenantID)
	entityID := strings.TrimSpace(in.EntityID)
	idemKey := strings.TrimSpace(in.IdempotencyKey)
	if tenantID == "" || idemKey == "" {
		return FxRevalResult{}, fmt.Errorf("%w: tenant_id and idempotency_key are required", ErrInvalidInput)
	}

	asOf := time.Now().UTC().Truncate(time.Microsecond)
	if raw := strings.TrimSpace(in.AsOf); raw != "" {
		parsed, pErr := time.Parse(time.RFC3339Nano, raw)
		if pErr != nil {
			return FxRevalResult{}, fmt.Errorf("%w: as_of must be RFC3339", ErrInvalidInput)
		}
		asOf = parsed.UTC().Truncate(time.Microsecond)
	}

	countryPack := "UZ"
	baseCur := fxrates.NormalizeCurrency(in.BaseCurrency)
	if entityID != "" {
		ent, ok, err := s.repo.GetEntity(ctx, tenantID, entityID)
		if err != nil {
			return FxRevalResult{}, err
		}
		if !ok {
			return FxRevalResult{}, fmt.Errorf("%w: %s", ErrEntityNotFound, entityID)
		}
		if baseCur == "" {
			baseCur = ent.BaseCurrency
		}
		if ent.CountryCode != "" {
			countryPack = ent.CountryCode
		}
	}
	if baseCur == "" {
		if c, err := auth.CurrencyFromContext(ctx, tenantID); err == nil && c != "" {
			baseCur = fxrates.NormalizeCurrency(c)
		}
	}
	if baseCur == "" {
		baseCur = gl.DefaultCurrencyForPack(countryPack)
	}
	if len(baseCur) != 3 {
		return FxRevalResult{}, fmt.Errorf("%w: invalid base_currency %q", ErrInvalidCurrency, in.BaseCurrency)
	}

	periodID := strings.TrimSpace(in.PeriodID)
	if periodID == "" {
		periodID = gl.ComputePeriodID(ctx, tenantID, countryPack, asOf)
	} else if !periodRegex.MatchString(periodID) {
		return FxRevalResult{}, ErrInvalidPeriod
	}

	positions, err := s.repo.CollectOpenForeignCurrencyPositions(ctx, tenantID, entityID, baseCur, asOf)
	if err != nil {
		return FxRevalResult{}, err
	}

	// Cache closing rates per foreign currency and FAIL CLOSED on any missing rate.
	closingRates := make(map[string]resolvedRate)
	appliedRates := make([]FxRateApplied, 0)

	type aggBucketKey struct {
		curr string
		dir  string
	}
	aggForeign := make(map[aggBucketKey]int64)
	aggCarrying := make(map[aggBucketKey]int64)
	aggOrder := make([]aggBucketKey, 0)

	type bsAcctKey struct {
		acct string
		curr string
	}
	bsNetDrMinusCrByAcct := make(map[bsAcctKey]int64)
	bsAcctOrder := make([]bsAcctKey, 0)

	var sumItemPnl int64
	evaluated := make([]FxRevalPosition, 0, len(positions))

	for _, pos := range positions {
		fc := fxrates.NormalizeCurrency(pos.Currency)
		if len(fc) != 3 || fc == baseCur || pos.ForeignAmountMinor == 0 {
			continue
		}
		closingRR, ok := closingRates[fc]
		if !ok {
			rr, rErr := resolveRateAsOf(ctx, s.fxRepo, s.fxSvc, fc, baseCur, asOf)
			if rErr != nil {
				return FxRevalResult{}, rErr
			}
			closingRR = rr
			closingRates[fc] = rr
			appliedRates = append(appliedRates, toFxRateApplied(rr, fc, baseCur))
		}

		revaluedFunc, err := convertSignedMinor(pos.ForeignAmountMinor, closingRR, fc, baseCur)
		if err != nil {
			return FxRevalResult{}, err
		}

		carryingFunc := pos.CarryingFunctionalMinor
		if !pos.HasHistoricalFunctional {
			histAt := pos.OccurredAt
			if histAt.IsZero() {
				histAt = asOf
			}
			histRR, hErr := resolveRateAsOf(ctx, s.fxRepo, s.fxSvc, fc, baseCur, histAt)
			if hErr != nil {
				return FxRevalResult{}, hErr
			}
			carryingFunc, err = convertSignedMinor(pos.ForeignAmountMinor, histRR, fc, baseCur)
			if err != nil {
				return FxRevalResult{}, err
			}
		}
		carryingFunc += pos.PriorRevalFunctionalMinor

		var itemPnl int64
		var bsDrMinusCr int64
		if strings.EqualFold(pos.Direction, "LIABILITY") {
			// Liability increase (revalued > carrying) is an FX loss (-), credited to liability (- DrMinusCr).
			itemPnl = carryingFunc - revaluedFunc
			bsDrMinusCr = carryingFunc - revaluedFunc // positive when liability decreases -> DEBIT liability
		} else {
			// Asset increase (revalued > carrying) is an FX gain (+), debited to asset (+ DrMinusCr).
			itemPnl = revaluedFunc - carryingFunc
			bsDrMinusCr = revaluedFunc - carryingFunc
		}

		pos.CarryingFunctionalMinor = carryingFunc
		pos.RevaluedFunctionalMinor = revaluedFunc
		pos.UnrealizedGainLossMinor = itemPnl
		pos.RateID = closingRR.Rate.RateID
		pos.RateScaled = closingRR.Rate.RateScaled
		pos.Scale = closingRR.Rate.Scale
		if pos.Scale <= 0 {
			pos.Scale = fxrates.DefaultScale
		}
		evaluated = append(evaluated, pos)

		sumItemPnl += itemPnl
		ak := aggBucketKey{curr: fc, dir: strings.ToUpper(pos.Direction)}
		if _, seen := aggForeign[ak]; !seen {
			aggOrder = append(aggOrder, ak)
		}
		aggForeign[ak] += pos.ForeignAmountMinor
		aggCarrying[ak] += carryingFunc

		bk := bsAcctKey{acct: pos.AccountCode, curr: fc}
		if _, seen := bsNetDrMinusCrByAcct[bk]; !seen {
			bsAcctOrder = append(bsAcctOrder, bk)
		}
		bsNetDrMinusCrByAcct[bk] += bsDrMinusCr
	}

	var aggPnl int64
	for _, ak := range aggOrder {
		rr := closingRates[ak.curr]
		aggRev, err := convertSignedMinor(aggForeign[ak], rr, ak.curr, baseCur)
		if err != nil {
			return FxRevalResult{}, err
		}
		if ak.dir == "LIABILITY" {
			aggPnl += aggCarrying[ak] - aggRev
		} else {
			aggPnl += aggRev - aggCarrying[ak]
		}
	}

	var bsTotalDrMinusCr int64
	postings := make([]gl.PostingInput, 0, len(bsAcctOrder)+2)
	for _, bk := range bsAcctOrder {
		delta := bsNetDrMinusCrByAcct[bk]
		if delta == 0 {
			continue
		}
		bsTotalDrMinusCr += delta
		dir := gl.DirectionDebit
		amt := delta
		if delta < 0 {
			dir = gl.DirectionCredit
			amt = -delta
		}
		postings = append(postings, gl.PostingInput{
			AccountCode:           bk.acct,
			Direction:             dir,
			AmountMinor:           amt,
			Currency:              baseCur,
			FunctionalAmountMinor: amt,
			PartyType:             "FX_CURRENCY",
			PartyID:               bk.curr,
			Description:           fmt.Sprintf("FX revaluation adjustment (%s -> %s)", bk.curr, baseCur),
		})
	}

	// Positive aggPnl = Unrealized FX Gain (CREDIT FX:UNREALIZED_GAIN_LOSS, so DrMinusCr = -aggPnl).
	pnlDrMinusCr := -aggPnl
	if pnlDrMinusCr != 0 {
		dir := gl.DirectionDebit
		amt := pnlDrMinusCr
		if pnlDrMinusCr < 0 {
			dir = gl.DirectionCredit
			amt = -pnlDrMinusCr
		}
		postings = append(postings, gl.PostingInput{
			AccountCode:           gl.AccountFxUnrealizedGainLoss,
			Direction:             dir,
			AmountMinor:           amt,
			Currency:              baseCur,
			FunctionalAmountMinor: amt,
			Description:           fmt.Sprintf("Unrealized FX gain/loss for period %s", periodID),
		})
	}

	// Rounding residual absorbed into FX:ROUNDING so sum(Debits) == sum(Credits) strictly.
	roundingDrMinusCr := -(bsTotalDrMinusCr + pnlDrMinusCr)
	roundingDeltaMinor := -roundingDrMinusCr
	if roundingDrMinusCr != 0 {
		dir := gl.DirectionDebit
		amt := roundingDrMinusCr
		if roundingDrMinusCr < 0 {
			dir = gl.DirectionCredit
			amt = -roundingDrMinusCr
		}
		postings = append(postings, gl.PostingInput{
			AccountCode:           gl.AccountFxRounding,
			Direction:             dir,
			AmountMinor:           amt,
			Currency:              baseCur,
			FunctionalAmountMinor: amt,
			Description:           fmt.Sprintf("FX revaluation rounding delta for period %s", periodID),
		})
	}

	runID := deterministicID("fxr", tenantID, idemKey)
	actor := coalesceNonEmpty(in.CreatedBy, "system")

	var journalIn *gl.JournalEntryInput
	if len(postings) >= 2 {
		glOccurredAt := asOf
		if nowUTC := time.Now().UTC().Truncate(time.Microsecond); glOccurredAt.After(nowUTC) {
			glOccurredAt = nowUTC
		}
		journalIn = &gl.JournalEntryInput{
			EntityID:       tenantID,
			LegalEntityID:  entityID,
			PeriodID:       periodID,
			SourceType:     "FX_REVALUATION",
			SourceID:       runID,
			Memo:           fmt.Sprintf("Month-end FX revaluation (%s) as of %s", baseCur, asOf.Format(time.RFC3339)),
			Currency:       baseCur,
			CountryPack:    countryPack,
			IdempotencyKey: "fxr:" + idemKey,
			PostedBy:       actor,
			OccurredAt:     glOccurredAt,
			Postings:       postings,
		}
	}

	ratesBytes, _ := json.Marshal(appliedRates)
	run := FxRevaluationRun{
		TenantID:                tenantID,
		RunID:                   runID,
		EntityID:                entityID,
		PeriodID:                periodID,
		BaseCurrency:            baseCur,
		UnrealizedGainLossMinor: aggPnl,
		RoundingDeltaMinor:      roundingDeltaMinor,
		RatesJSON:               string(ratesBytes),
		IdempotencyKey:          idemKey,
		CreatedBy:               actor,
	}

	afterBytes, _ := json.Marshal(run)
	auditIn := AuditRecordInput{
		TenantID:   tenantID,
		ActorID:    actor,
		ActorType:  coalesceNonEmpty(in.ActorType, auth.ActorTypeHuman),
		Role:       in.Role,
		Action:     "finance.fx_revaluation.run",
		ObjectType: "FxRevaluationRun",
		ObjectID:   runID,
		AfterJSON:  string(afterBytes),
		TraceID:    in.TraceID,
	}

	persisted, already, err := s.repo.CreateFxRevaluationRunAtomic(ctx, run, journalIn, s.auditRec, auditIn)
	if err != nil {
		if errors.Is(err, gl.ErrPeriodClosed) {
			return FxRevalResult{}, fmt.Errorf("%w: %v", ErrPeriodClosed, err)
		}
		return FxRevalResult{}, err
	}
	return FxRevalResult{
		Run:           persisted,
		AlreadyPosted: already,
		PositionCount: len(evaluated),
		Positions:     evaluated,
	}, nil
}

// Consolidate produces a multi-entity trial balance in reportingCurrency with FX translation,
// intercompany eliminations (IC:DUE_FROM, IC:DUE_TO, IntercompanyTransfers), and per-entity breakdown.
func (s *Service) Consolidate(ctx context.Context, tenantID, periodID, reportingCurrency string, entityIDs []string) (ConsolidationReport, error) {
	if s == nil || s.repo == nil {
		return ConsolidationReport{}, ErrRepoUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return ConsolidationReport{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	periodID = strings.TrimSpace(periodID)
	if periodID == "" {
		periodID = gl.ComputePeriodID(ctx, tenantID, "", time.Now().UTC())
	} else if !periodRegex.MatchString(periodID) {
		return ConsolidationReport{}, ErrInvalidPeriod
	}

	snap, err := s.repo.ReadConsolidationSnapshot(ctx, tenantID, periodID, entityIDs)
	if err != nil {
		return ConsolidationReport{}, err
	}

	repCur := fxrates.NormalizeCurrency(reportingCurrency)
	if repCur == "" {
		for _, ent := range snap.Entities {
			if ent.IsParent && strings.TrimSpace(ent.BaseCurrency) != "" {
				repCur = fxrates.NormalizeCurrency(ent.BaseCurrency)
				break
			}
		}
	}
	if repCur == "" && len(snap.Entities) > 0 {
		repCur = fxrates.NormalizeCurrency(snap.Entities[0].BaseCurrency)
	}
	if repCur == "" {
		if c, err := auth.CurrencyFromContext(ctx, tenantID); err == nil && c != "" {
			repCur = fxrates.NormalizeCurrency(c)
		}
	}
	if repCur == "" {
		repCur = gl.DefaultCurrencyForPack("")
	}
	if len(repCur) != 3 {
		return ConsolidationReport{}, fmt.Errorf("%w: invalid reporting currency %q", ErrInvalidCurrency, reportingCurrency)
	}

	asOf := time.Now().UTC()
	rateCache := make(map[string]resolvedRate)
	appliedRates := make([]FxRateApplied, 0)
	resolveCachedRate := func(fromCur string) (resolvedRate, error) {
		fromCur = fxrates.NormalizeCurrency(fromCur)
		if fromCur == "" {
			fromCur = repCur
		}
		if rr, ok := rateCache[fromCur]; ok {
			return rr, nil
		}
		rr, rErr := resolveRateAsOf(ctx, s.fxRepo, s.fxSvc, fromCur, repCur, asOf)
		if rErr != nil {
			return resolvedRate{}, rErr
		}
		rateCache[fromCur] = rr
		if fromCur != repCur {
			appliedRates = append(appliedRates, toFxRateApplied(rr, fromCur, repCur))
		}
		return rr, nil
	}

	type accBucket struct {
		code     string
		name     string
		class    string
		preDr    int64
		preCr    int64
		elimDr   int64
		elimCr   int64
		byEntity map[string]int64
	}
	accountsMap := make(map[string]*accBucket)
	getAccBucket := func(code, name, class string) *accBucket {
		code = strings.ToUpper(strings.TrimSpace(code))
		b, ok := accountsMap[code]
		if !ok {
			cls := strings.ToUpper(strings.TrimSpace(class))
			nm := strings.TrimSpace(name)
			if cls == "" || nm == "" {
				if def, found := gl.LookupSystemAccount(code); found {
					if cls == "" {
						cls = def.Class
					}
					if nm == "" {
						nm = def.Name
					}
				}
			}
			b = &accBucket{
				code:     code,
				name:     nm,
				class:    cls,
				byEntity: make(map[string]int64),
			}
			accountsMap[code] = b
		}
		return b
	}

	selectedSet := make(map[string]struct{}, len(snap.Entities))
	resolvedEntityIDs := make([]string, 0, len(snap.Entities))
	entitySummaries := make([]ConsolidatedEntitySummary, 0, len(snap.Entities))

	for _, ent := range snap.Entities {
		selectedSet[ent.EntityID] = struct{}{}
		resolvedEntityIDs = append(resolvedEntityIDs, ent.EntityID)

		tb := snap.TrialBalances[ent.EntityID]
		entRate, err := resolveCachedRate(ent.BaseCurrency)
		if err != nil {
			return ConsolidationReport{}, err
		}

		var convDrTotal, convCrTotal int64
		for _, line := range tb.Accounts {
			lineCur := fxrates.NormalizeCurrency(line.Currency)
			if lineCur == "" {
				lineCur = fxrates.NormalizeCurrency(ent.BaseCurrency)
			}
			rr, err := resolveCachedRate(lineCur)
			if err != nil {
				return ConsolidationReport{}, err
			}
			convDr, err := convertSignedMinor(line.DebitMinor, rr, lineCur, repCur)
			if err != nil {
				return ConsolidationReport{}, err
			}
			convCr, err := convertSignedMinor(line.CreditMinor, rr, lineCur, repCur)
			if err != nil {
				return ConsolidationReport{}, err
			}
			convDrTotal += convDr
			convCrTotal += convCr

			b := getAccBucket(line.AccountCode, line.AccountName, line.Class)
			b.preDr += convDr
			b.preCr += convCr
			var lineConvNet int64
			if gl.IsDebitNormalClass(b.class) {
				lineConvNet = convDr - convCr
			} else {
				lineConvNet = convCr - convDr
			}
			b.byEntity[ent.EntityID] += lineConvNet
		}

		// Absorb any integer translation rounding drift across translated lines into FX:ROUNDING.
		if convDrTotal != convCrTotal {
			diff := convDrTotal - convCrTotal
			b := getAccBucket(gl.AccountFxRounding, "FX Conversion & Rounding Difference", gl.ClassExpense)
			if diff > 0 {
				b.preCr += diff
				convCrTotal += diff
				b.byEntity[ent.EntityID] -= diff
			} else {
				absDiff := -diff
				b.preDr += absDiff
				convDrTotal += absDiff
				b.byEntity[ent.EntityID] += absDiff
			}
		}

		sumNet := tb.TotalDebitsMinor - tb.TotalCreditsMinor
		convNet := convDrTotal - convCrTotal
		summary := ConsolidatedEntitySummary{
			EntityID:             ent.EntityID,
			Name:                 ent.Name,
			TIN:                  ent.TIN,
			CountryCode:          ent.CountryCode,
			BaseCurrency:         ent.BaseCurrency,
			IsParent:             ent.IsParent,
			DebitMinor:           tb.TotalDebitsMinor,
			CreditMinor:          tb.TotalCreditsMinor,
			NetMinor:             sumNet,
			ConvertedDebitMinor:  convDrTotal,
			ConvertedCreditMinor: convCrTotal,
			ConvertedNetMinor:    convNet,
			Balanced:             tb.Balanced && convDrTotal == convCrTotal,
		}
		if ent.BaseCurrency != repCur {
			summary.FxRateScaled = entRate.Rate.RateScaled
			summary.FxScale = entRate.Rate.Scale
			summary.FxInverse = entRate.Inverse
		} else {
			summary.FxRateScaled = fxrates.DefaultScale
			summary.FxScale = fxrates.DefaultScale
		}
		entitySummaries = append(entitySummaries, summary)
	}

	// Compute Intercompany Eliminations across IC:DUE_FROM / IC:DUE_TO & IntercompanyTransfers.
	eliminations := make([]IntercompanyElimination, 0)
	var transferElimSum int64
	for _, tr := range snap.Transfers {
		_, fromIn := selectedSet[tr.FromEntityID]
		_, toIn := selectedSet[tr.ToEntityID]
		if !fromIn || !toIn {
			continue
		}
		rr, err := resolveCachedRate(tr.Currency)
		if err != nil {
			return ConsolidationReport{}, err
		}
		convAmt, err := convertSignedMinor(tr.AmountMinor, rr, tr.Currency, repCur)
		if err != nil {
			return ConsolidationReport{}, err
		}
		transferElimSum += convAmt
		eliminations = append(eliminations, IntercompanyElimination{
			TransferID:           tr.TransferID,
			AccountCode:          gl.AccountIntercompanyDueFrom + " / " + gl.AccountIntercompanyDueTo,
			FromEntityID:         tr.FromEntityID,
			ToEntityID:           tr.ToEntityID,
			Currency:             tr.Currency,
			AmountMinor:          tr.AmountMinor,
			ConvertedAmountMinor: convAmt,
			TransferType:         tr.TransferType,
			Description:          fmt.Sprintf("Eliminate %s intercompany transfer %s", tr.TransferType, tr.TransferID),
		})
	}

	var dueFromNet, dueToNet int64
	if b, ok := accountsMap[gl.AccountIntercompanyDueFrom]; ok {
		dueFromNet = b.preDr - b.preCr
	}
	if b, ok := accountsMap[gl.AccountIntercompanyDueTo]; ok {
		dueToNet = b.preCr - b.preDr
	}
	eliminatedIC := minPositiveInt64(dueFromNet, dueToNet)
	if eliminatedIC > 0 {
		if b, ok := accountsMap[gl.AccountIntercompanyDueFrom]; ok {
			b.elimDr = eliminatedIC
		}
		if b, ok := accountsMap[gl.AccountIntercompanyDueTo]; ok {
			b.elimCr = eliminatedIC
		}
		if len(eliminations) == 0 || eliminatedIC > transferElimSum {
			rem := eliminatedIC - transferElimSum
			if rem > 0 {
				eliminations = append(eliminations, IntercompanyElimination{
					AccountCode:          gl.AccountIntercompanyDueFrom + " / " + gl.AccountIntercompanyDueTo,
					FromEntityID:         "*",
					ToEntityID:           "*",
					Currency:             repCur,
					AmountMinor:          rem,
					ConvertedAmountMinor: rem,
					TransferType:         "NET_IC_BALANCE",
					Description:          "Eliminate matched IC:DUE_FROM and IC:DUE_TO balances",
				})
			}
		}
	}

	icVariance := dueFromNet - dueToNet
	if icVariance < 0 {
		icVariance = -icVariance
	}

	codes := make([]string, 0, len(accountsMap))
	for c := range accountsMap {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	outAccounts := make([]ConsolidatedAccountLine, 0, len(codes))
	var totalDr, totalCr int64
	for _, code := range codes {
		b := accountsMap[code]
		consDr := b.preDr - b.elimDr
		consCr := b.preCr - b.elimCr
		var consNet int64
		if gl.IsDebitNormalClass(b.class) {
			consNet = consDr - consCr
		} else {
			consNet = consCr - consDr
		}
		elimAmt := b.elimDr + b.elimCr
		totalDr += consDr
		totalCr += consCr
		outAccounts = append(outAccounts, ConsolidatedAccountLine{
			AccountCode:               b.code,
			AccountName:               b.name,
			Class:                     b.class,
			Currency:                  repCur,
			PreEliminationDebitMinor:  b.preDr,
			PreEliminationCreditMinor: b.preCr,
			DebitMinor:                consDr,
			CreditMinor:               consCr,
			EliminatedMinor:           elimAmt,
			NetMinor:                  consNet,
			ByEntity:                  b.byEntity,
		})
	}

	return ConsolidationReport{
		TenantID:               tenantID,
		PeriodID:               periodID,
		ReportingCurrency:      repCur,
		EntityIDs:              resolvedEntityIDs,
		Entities:               entitySummaries,
		Eliminations:           eliminations,
		Accounts:               outAccounts,
		TotalDebitsMinor:       totalDr,
		TotalCreditsMinor:      totalCr,
		TotalEliminationsMinor: eliminatedIC,
		ICVarianceMinor:        icVariance,
		Balanced:               totalDr == totalCr,
		FxRatesApplied:         appliedRates,
	}, nil
}

func resolveIntercompanyOffsetAccount(transferType, explicitOffset string) (string, error) {
	if explicit := strings.TrimSpace(explicitOffset); explicit != "" {
		norm, _, _ := gl.NormalizeAccountCode(explicit)
		if _, ok := gl.LookupSystemAccount(norm); !ok {
			return "", fmt.Errorf("%w: unknown offset_account %q", ErrInvalidInput, explicit)
		}
		return norm, nil
	}
	switch strings.ToUpper(strings.TrimSpace(transferType)) {
	case "INVENTORY", "INVENTORY_TRANSFER", "STOCK":
		return gl.AccountInventory, nil
	case "PETTY_CASH", "OFFICE_CASH":
		return gl.AccountCashOffice, nil
	case "EXPENSE_RECHARGE", "COST_ALLOCATION", "SERVICE":
		return gl.AccountClaimsSettlementExpense, nil
	case "CASH", "CASH_POOL", "FUNDS", "FUNDING", "BANK", "LOAN", "INTERCOMPANY_LOAN", "BALANCE", "":
		return gl.AccountBankOperating, nil
	default:
		return "", fmt.Errorf("%w: unsupported transfer_type %q", ErrInvalidInput, transferType)
	}
}

func deterministicEntityID(tenantID, name, tin string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(tenantID) + "|" + strings.TrimSpace(name) + "|" + strings.TrimSpace(tin) + "|" + fmt.Sprintf("%d", time.Now().UnixNano())))
	return "ent_" + hex.EncodeToString(sum[:12])
}

func deterministicID(prefix, tenantID, key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(tenantID) + "|" + strings.TrimSpace(key)))
	return prefix + "_" + hex.EncodeToString(sum[:24])
}

func minPositiveInt64(a, b int64) int64 {
	if a <= 0 || b <= 0 {
		return 0
	}
	if a < b {
		return a
	}
	return b
}

func coalesceNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
