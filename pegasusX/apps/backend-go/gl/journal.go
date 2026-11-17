package gl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Direction designates DEBIT or CREDIT in double-entry bookkeeping.
type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"

	// BalanceShardCount distributes hot GlBalances control-account writes across buckets.
	BalanceShardCount int64 = 8
)

// Domain invariant errors.
var (
	ErrNilTransaction          = errors.New("gl: nil spanner read-write transaction")
	ErrInvalidJournalInput     = errors.New("gl: invalid journal entry input")
	ErrInsufficientPostings    = errors.New("gl: journal entry must contain at least 2 postings")
	ErrInvalidPostingAmount    = errors.New("gl: posting amount_minor and functional_amount_minor must be > 0")
	ErrInvalidPostingDirection = errors.New("gl: invalid posting direction (must be DEBIT or CREDIT)")
	ErrCurrencyMismatch        = errors.New("gl: posting currency does not match journal entry currency")
	ErrUnbalancedEntry         = errors.New("gl: sum of debits must strictly equal sum of credits")
	ErrUnknownAccount          = errors.New("gl: unknown account code")
	ErrInactiveAccount         = errors.New("gl: account is inactive for occurred_at")
	ErrEntryNotFound           = errors.New("gl: journal entry not found")
	ErrAlreadyReversed         = errors.New("gl: journal entry already reversed")
	ErrPeriodClosed            = errors.New("gl: accounting period is closed")
)

// PostingInput represents one debit or credit leg of a balanced journal entry.
type PostingInput struct {
	AccountCode           string    `json:"account_code"`
	Direction             Direction `json:"direction"`
	AmountMinor           int64     `json:"amount_minor"`
	Currency              string    `json:"currency,omitempty"`
	FunctionalAmountMinor int64     `json:"functional_amount_minor,omitempty"`
	FxRateID              string    `json:"fx_rate_id,omitempty"`
	PartyType             string    `json:"party_type,omitempty"`
	PartyID               string    `json:"party_id,omitempty"`
	WarehouseID           string    `json:"warehouse_id,omitempty"`
	ProductID             string    `json:"product_id,omitempty"`
	LotID                 string    `json:"lot_id,omitempty"`
	Description           string    `json:"description,omitempty"`
}

// JournalEntryInput is the normalized command payload for BufferJournalEntryInTxn.
type JournalEntryInput struct {
	EntryID          string         `json:"entry_id,omitempty"`
	EntityID         string         `json:"entity_id"`
	LegalEntityID    string         `json:"legal_entity_id,omitempty"`
	PeriodID         string         `json:"period_id,omitempty"`
	OriginalPeriodID string         `json:"original_period_id,omitempty"`
	SourceEventID    string         `json:"source_event_id,omitempty"`
	SourceType       string         `json:"source_type"`
	SourceID         string         `json:"source_id"`
	OrderID          string         `json:"order_id,omitempty"`
	Memo             string         `json:"memo,omitempty"`
	Currency         string         `json:"currency"`
	CountryPack      string         `json:"country_pack,omitempty"`
	ReversesEntryID  string         `json:"reverses_entry_id,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key"`
	PostedBy         string         `json:"posted_by,omitempty"`
	OccurredAt       time.Time      `json:"occurred_at,omitempty"`
	Postings         []PostingInput `json:"postings"`
}

// PostInput is the multi-entity-aware GL posting payload separating TenantID (partition key)
// from optional EntityID (LegalEntityId within the tenant).
type PostInput struct {
	EntryID          string         `json:"entry_id,omitempty"`
	TenantID         string         `json:"tenant_id"`
	EntityID         string         `json:"entity_id,omitempty"`
	PeriodID         string         `json:"period_id,omitempty"`
	OriginalPeriodID string         `json:"original_period_id,omitempty"`
	SourceEventID    string         `json:"source_event_id,omitempty"`
	SourceType       string         `json:"source_type"`
	SourceID         string         `json:"source_id"`
	OrderID          string         `json:"order_id,omitempty"`
	Memo             string         `json:"memo,omitempty"`
	Currency         string         `json:"currency"`
	CountryPack      string         `json:"country_pack,omitempty"`
	ReversesEntryID  string         `json:"reverses_entry_id,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key"`
	PostedBy         string         `json:"posted_by,omitempty"`
	OccurredAt       time.Time      `json:"occurred_at,omitempty"`
	Postings         []PostingInput `json:"postings"`
}

// ToJournalEntryInput converts a PostInput into a JournalEntryInput preserving TenantID as the
// GL partition key (JournalEntries.EntityId) and optional EntityID as LegalEntityId.
func (p PostInput) ToJournalEntryInput() JournalEntryInput {
	tenantID := strings.TrimSpace(p.TenantID)
	legalEntityID := strings.TrimSpace(p.EntityID)
	if tenantID == "" && legalEntityID != "" {
		tenantID = legalEntityID
		legalEntityID = ""
	}
	if legalEntityID == tenantID {
		legalEntityID = ""
	}
	return JournalEntryInput{
		EntryID:          p.EntryID,
		EntityID:         tenantID,
		LegalEntityID:    legalEntityID,
		PeriodID:         p.PeriodID,
		OriginalPeriodID: p.OriginalPeriodID,
		SourceEventID:    p.SourceEventID,
		SourceType:       p.SourceType,
		SourceID:         p.SourceID,
		OrderID:          p.OrderID,
		Memo:             p.Memo,
		Currency:         p.Currency,
		CountryPack:      p.CountryPack,
		ReversesEntryID:  p.ReversesEntryID,
		IdempotencyKey:   p.IdempotencyKey,
		PostedBy:         p.PostedBy,
		OccurredAt:       p.OccurredAt,
		Postings:         p.Postings,
	}
}

// JournalFilter defines query filters for ListJournals.
type JournalFilter struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	PeriodID      string `json:"period_id,omitempty"`
	SourceType    string `json:"source_type,omitempty"`
	AccountCode   string `json:"account_code,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

// JournalEntryResult reports the committed or deduplicated journal entry state.
type JournalEntryResult struct {
	EntryID          string `json:"entry_id"`
	EntityID         string `json:"entity_id"`
	LegalEntityID    string `json:"legal_entity_id,omitempty"`
	PeriodID         string `json:"period_id"`
	OriginalPeriodID string `json:"original_period_id,omitempty"`
	PrevHash         string `json:"prev_hash"`
	Hash             string `json:"hash"`
	AlreadyPosted    bool   `json:"already_posted"`
	TotalDebit       int64  `json:"total_debit_minor"`
	TotalCredit      int64  `json:"total_credit_minor"`
	LineCount        int    `json:"line_count"`
}

// JournalPostingRecord represents one persisted JournalPostings row.
type JournalPostingRecord struct {
	EntityID              string    `json:"entity_id"`
	EntryID               string    `json:"entry_id"`
	Line                  int64     `json:"line"`
	AccountCode           string    `json:"account_code"`
	Direction             Direction `json:"direction"`
	AmountMinor           int64     `json:"amount_minor"`
	Currency              string    `json:"currency"`
	FunctionalAmountMinor int64     `json:"functional_amount_minor"`
	FxRateID              string    `json:"fx_rate_id,omitempty"`
	PartyType             string    `json:"party_type,omitempty"`
	PartyID               string    `json:"party_id,omitempty"`
	WarehouseID           string    `json:"warehouse_id,omitempty"`
	ProductID             string    `json:"product_id,omitempty"`
	LotID                 string    `json:"lot_id,omitempty"`
	Description           string    `json:"description,omitempty"`
}

// JournalEntryRecord represents a persisted JournalEntries header + its postings.
type JournalEntryRecord struct {
	EntityID         string                 `json:"entity_id"`
	EntryID          string                 `json:"entry_id"`
	LegalEntityID    string                 `json:"legal_entity_id,omitempty"`
	PeriodID         string                 `json:"period_id"`
	OriginalPeriodID string                 `json:"original_period_id,omitempty"`
	SourceEventID    string                 `json:"source_event_id,omitempty"`
	SourceType       string                 `json:"source_type"`
	SourceID         string                 `json:"source_id"`
	OrderID          string                 `json:"order_id,omitempty"`
	Memo             string                 `json:"memo,omitempty"`
	Currency         string                 `json:"currency"`
	CountryPack      string                 `json:"country_pack"`
	ReversesEntryID  string                 `json:"reverses_entry_id,omitempty"`
	IdempotencyKey   string                 `json:"idempotency_key"`
	PrevHash         string                 `json:"prev_hash"`
	EntryHash        string                 `json:"entry_hash"`
	PostedBy         string                 `json:"posted_by"`
	OccurredAt       time.Time              `json:"occurred_at"`
	PostedAt         time.Time              `json:"posted_at"`
	Postings         []JournalPostingRecord `json:"postings,omitempty"`
}

// TrialBalanceLine summarizes one COA account across all shards in GlBalances.
type TrialBalanceLine struct {
	AccountCode string `json:"account_code"`
	AccountName string `json:"account_name,omitempty"`
	Class       string `json:"class,omitempty"`
	Currency    string `json:"currency"`
	DebitMinor  int64  `json:"debit_minor"`
	CreditMinor int64  `json:"credit_minor"`
	NetMinor    int64  `json:"net_minor"`
}

// TrialBalanceReport is the period-scoped trial balance for an entity.
type TrialBalanceReport struct {
	EntityID          string             `json:"entity_id"`
	TenantID          string             `json:"tenant_id,omitempty"`
	LegalEntityID     string             `json:"legal_entity_id,omitempty"`
	PeriodID          string             `json:"period_id"`
	Currency          string             `json:"currency"`
	TotalDebitsMinor  int64              `json:"total_debits_minor"`
	TotalCreditsMinor int64              `json:"total_credits_minor"`
	Balanced          bool               `json:"balanced"`
	Accounts          []TrialBalanceLine `json:"accounts"`
}

// AccountLedgerRow is one posting line enriched with its parent JournalEntry metadata.
type AccountLedgerRow struct {
	EntryID               string    `json:"entry_id"`
	PeriodID              string    `json:"period_id"`
	Line                  int64     `json:"line"`
	AccountCode           string    `json:"account_code"`
	Direction             Direction `json:"direction"`
	AmountMinor           int64     `json:"amount_minor"`
	Currency              string    `json:"currency"`
	FunctionalAmountMinor int64     `json:"functional_amount_minor"`
	FxRateID              string    `json:"fx_rate_id,omitempty"`
	PartyType             string    `json:"party_type,omitempty"`
	PartyID               string    `json:"party_id,omitempty"`
	SourceType            string    `json:"source_type"`
	SourceID              string    `json:"source_id"`
	OrderID               string    `json:"order_id,omitempty"`
	ReversesEntryID       string    `json:"reverses_entry_id,omitempty"`
	Memo                  string    `json:"memo,omitempty"`
	Description           string    `json:"description,omitempty"`
	EntryHash             string    `json:"entry_hash"`
	OccurredAt            time.Time `json:"occurred_at"`
	PostedAt              time.Time `json:"posted_at"`
}

// ResolveEntityLocation resolves the IANA *time.Location for entityID / countryPack
// via auth/timezone_pack.go, falling back cleanly for planned packs (e.g. KZ).
func ResolveEntityLocation(ctx context.Context, entityID, countryPack string) *time.Location {
	if loc, err := auth.TimezoneFromContext(ctx, entityID); err == nil && loc != nil {
		packCode := auth.NormalizeCountryCode(countryPack)
		if packCode == "" {
			return loc
		}
		if resolvedCountry, cErr := auth.CountryFromContext(ctx, entityID); cErr == nil && resolvedCountry == packCode {
			return loc
		}
	}
	packCode := auth.NormalizeCountryCode(countryPack)
	if packCode != "" {
		if pack, ok := auth.ResolveMarketPack(packCode); ok && strings.TrimSpace(pack.Timezone) != "" {
			if loc, err := time.LoadLocation(strings.TrimSpace(pack.Timezone)); err == nil {
				return loc
			}
		}
	}
	if loc, err := auth.TimezoneFromContext(ctx, entityID); err == nil && loc != nil {
		return loc
	}
	if loc, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return loc
	}
	return time.UTC
}

// ComputePeriodID returns the accounting period string ("YYYY-MM") for occurredAt
// evaluated in the entity's timezone.
func ComputePeriodID(ctx context.Context, entityID, countryPack string, occurredAt time.Time) string {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	loc := ResolveEntityLocation(ctx, entityID, countryPack)
	return occurredAt.In(loc).Format("2006-01")
}

func resolveCountryPack(ctx context.Context, entityID, explicit string) string {
	if c := auth.NormalizeCountryCode(explicit); c != "" {
		return c
	}
	if c, err := auth.CountryFromContext(ctx, entityID); err == nil && c != "" {
		return c
	}
	if claims, ok := auth.FromContext(ctx); ok {
		if c := auth.NormalizeCountryCode(claims.MarketCode); c != "" {
			return c
		}
	}
	if c := auth.DefaultMarketCodeFromEnv(); c != "" {
		return c
	}
	return "UZ"
}

// DeterministicEntryID builds a deterministic STRING(128)-safe EntryId for (entityID, idempotencyKey).
func DeterministicEntryID(entityID, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(entityID) + "|" + strings.TrimSpace(idempotencyKey)))
	return "gle_" + hex.EncodeToString(sum[:24])
}

// BalancePartyKey formats the GlBalances PartyKey dimension.
func BalancePartyKey(partyType, partyID string) string {
	pt := strings.ToUpper(strings.TrimSpace(partyType))
	pid := strings.TrimSpace(partyID)
	if pt != "" && pid != "" {
		return pt + ":" + pid
	}
	if pid != "" {
		return pid
	}
	return ""
}

// BalanceShard returns the deterministic GlBalances bucket shard in [0, BalanceShardCount).
func BalanceShard(entityID, periodID, accountCode, partyKey, seed string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(entityID))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write([]byte(periodID))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write([]byte(accountCode))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write([]byte(partyKey))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write([]byte(seed))
	return int64(h.Sum64() % uint64(BalanceShardCount))
}

// ValidateJournalInput normalizes in and enforces all double-entry invariants.
func ValidateJournalInput(ctx context.Context, in *JournalEntryInput) (totalDebit, totalCredit int64, err error) {
	if in == nil {
		return 0, 0, fmt.Errorf("%w: nil input", ErrInvalidJournalInput)
	}
	in.EntityID = strings.TrimSpace(in.EntityID)
	in.LegalEntityID = strings.TrimSpace(in.LegalEntityID)
	in.SourceEventID = strings.TrimSpace(in.SourceEventID)
	in.SourceType = strings.ToUpper(strings.TrimSpace(in.SourceType))
	in.SourceID = strings.TrimSpace(in.SourceID)
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.Memo = strings.TrimSpace(in.Memo)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CountryPack = resolveCountryPack(ctx, in.EntityID, in.CountryPack)
	in.ReversesEntryID = strings.TrimSpace(in.ReversesEntryID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.PostedBy = strings.TrimSpace(in.PostedBy)
	if in.PostedBy == "" {
		in.PostedBy = "system"
	}

	if in.EntityID == "" {
		return 0, 0, fmt.Errorf("%w: entity_id is required", ErrInvalidJournalInput)
	}
	if in.SourceType == "" || in.SourceID == "" {
		return 0, 0, fmt.Errorf("%w: source_type and source_id are required", ErrInvalidJournalInput)
	}
	if in.IdempotencyKey == "" {
		return 0, 0, fmt.Errorf("%w: idempotency_key is required", ErrInvalidJournalInput)
	}
	if len(in.Currency) != 3 {
		return 0, 0, fmt.Errorf("%w: valid 3-letter currency is required, got %q", ErrInvalidJournalInput, in.Currency)
	}

	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}
	// Cloud Spanner TIMESTAMP has microsecond resolution; truncate before hashing
	// so stored OccurredAt and canonical hash payload remain bit-identical.
	in.OccurredAt = in.OccurredAt.UTC().Truncate(time.Microsecond)

	in.PeriodID = strings.TrimSpace(in.PeriodID)
	if in.PeriodID == "" {
		in.PeriodID = ComputePeriodID(ctx, in.EntityID, in.CountryPack, in.OccurredAt)
	}
	in.EntryID = strings.TrimSpace(in.EntryID)
	if in.EntryID == "" {
		in.EntryID = DeterministicEntryID(in.EntityID, in.IdempotencyKey)
	}

	if len(in.Postings) < 2 {
		return 0, 0, ErrInsufficientPostings
	}

	var sumDebit, sumCredit int64
	var sumFuncDebit, sumFuncCredit int64

	for i := range in.Postings {
		p := &in.Postings[i]
		p.AccountCode = strings.TrimSpace(p.AccountCode)
		if p.AccountCode == "" {
			return 0, 0, fmt.Errorf("%w: empty account_code at posting %d", ErrInvalidJournalInput, i)
		}
		// Normalize legacy suffixed codes when applicable while keeping custom COA codes uppercase.
		normCode, inferredPartyType, inferredPartyID := NormalizeAccountCode(p.AccountCode)
		p.AccountCode = normCode
		p.PartyType = strings.ToUpper(strings.TrimSpace(p.PartyType))
		p.PartyID = strings.TrimSpace(p.PartyID)
		if p.PartyType == "" && inferredPartyType != "" {
			p.PartyType = inferredPartyType
		}
		if p.PartyID == "" && inferredPartyID != "" {
			p.PartyID = inferredPartyID
		}

		p.Direction = Direction(strings.ToUpper(strings.TrimSpace(string(p.Direction))))
		if p.Direction != DirectionDebit && p.Direction != DirectionCredit {
			return 0, 0, fmt.Errorf("%w: %q at posting %d", ErrInvalidPostingDirection, p.Direction, i)
		}

		if p.AmountMinor <= 0 {
			return 0, 0, fmt.Errorf("%w: amount_minor=%d at posting %d", ErrInvalidPostingAmount, p.AmountMinor, i)
		}

		p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
		if p.Currency == "" {
			p.Currency = in.Currency
		} else if p.Currency != in.Currency {
			return 0, 0, fmt.Errorf("%w: posting %s has %s want %s", ErrCurrencyMismatch, p.AccountCode, p.Currency, in.Currency)
		}

		p.FxRateID = strings.TrimSpace(p.FxRateID)
		if p.FunctionalAmountMinor == 0 && p.FxRateID == "" {
			p.FunctionalAmountMinor = p.AmountMinor
		}
		if p.FunctionalAmountMinor <= 0 {
			return 0, 0, fmt.Errorf("%w: functional_amount_minor=%d at posting %d", ErrInvalidPostingAmount, p.FunctionalAmountMinor, i)
		}

		p.WarehouseID = strings.TrimSpace(p.WarehouseID)
		p.ProductID = strings.TrimSpace(p.ProductID)
		p.LotID = strings.TrimSpace(p.LotID)
		p.Description = strings.TrimSpace(p.Description)

		if p.Direction == DirectionDebit {
			sumDebit += p.AmountMinor
			sumFuncDebit += p.FunctionalAmountMinor
		} else {
			sumCredit += p.AmountMinor
			sumFuncCredit += p.FunctionalAmountMinor
		}
	}

	if sumDebit != sumCredit {
		return 0, 0, fmt.Errorf("%w: transaction sum(debits)=%d != sum(credits)=%d", ErrUnbalancedEntry, sumDebit, sumCredit)
	}
	if sumFuncDebit != sumFuncCredit {
		return 0, 0, fmt.Errorf("%w: functional sum(debits)=%d != sum(credits)=%d", ErrUnbalancedEntry, sumFuncDebit, sumFuncCredit)
	}

	return sumDebit, sumCredit, nil
}

// BufferJournalEntryInTxn validates input, checks (EntityId, IdempotencyKey) in txn,
// computes the per-entity SHA-256 hash chain, inserts JournalEntries + JournalPostings
// via InsertMap, updates sharded GlBalances, and emits EventGLEntryPosted/Reversed.
// If outboxBuf is nil, an internal outbox.SpannerTxnBuffer is flushed onto txn.
func BufferJournalEntryInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	input JournalEntryInput,
) (JournalEntryResult, error) {
	results, err := BufferJournalEntriesBatchInTxn(ctx, txn, outboxBuf, []JournalEntryInput{input})
	if err != nil {
		return JournalEntryResult{}, err
	}
	if len(results) == 0 {
		return JournalEntryResult{}, nil
	}
	return results[0], nil
}

type existingEntryInfo struct {
	EntryID          string
	LegalEntityID    string
	PeriodID         string
	OriginalPeriodID string
	PrevHash         string
	Hash             string
}

type glAccountMeta struct {
	Currency   string
	ActiveFrom *time.Time
	ActiveTo   *time.Time
}

type balanceBucketKey struct {
	EntityID    string
	PeriodID    string
	AccountCode string
	PartyKey    string
	Shard       int64
}

type balanceDelta struct {
	DebitMinor  int64
	CreditMinor int64
	Currency    string
}

type entityStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

// BufferJournalEntriesBatchInTxn atomically validates and buffers a batch of journal
// entries inside a single Spanner ReadWriteTransaction, chaining PrevHash -> EntryHash
// in memory and coalescing GlBalances updates across the batch.
func BufferJournalEntriesBatchInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	inputs []JournalEntryInput,
) ([]JournalEntryResult, error) {
	if txn == nil {
		return nil, ErrNilTransaction
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	normalized := make([]JournalEntryInput, len(inputs))
	totals := make([]int64, len(inputs))
	keysByEntity := make(map[string][]string)

	for i := range inputs {
		cp := inputs[i]
		if len(inputs[i].Postings) > 0 {
			cp.Postings = make([]PostingInput, len(inputs[i].Postings))
			copy(cp.Postings, inputs[i].Postings)
		}
		td, _, err := ValidateJournalInput(ctx, &cp)
		if err != nil {
			return nil, fmt.Errorf("entry[%d]: %w", i, err)
		}
		normalized[i] = cp
		totals[i] = td
		keysByEntity[cp.EntityID] = append(keysByEntity[cp.EntityID], cp.IdempotencyKey)
	}

	existingByEntityKey, err := readExistingIdempotencyKeysInTxn(ctx, txn, keysByEntity)
	if err != nil {
		return nil, err
	}

	results := make([]JournalEntryResult, len(normalized))
	pendingIdx := make([]int, 0, len(normalized))
	seenInBatch := make(map[string]int, len(normalized))

	for i, in := range normalized {
		ek := in.EntityID + "\x00" + in.IdempotencyKey
		if ex, ok := existingByEntityKey[ek]; ok {
			results[i] = JournalEntryResult{
				EntryID:          ex.EntryID,
				EntityID:         in.EntityID,
				LegalEntityID:    ex.LegalEntityID,
				PeriodID:         ex.PeriodID,
				OriginalPeriodID: ex.OriginalPeriodID,
				PrevHash:         ex.PrevHash,
				Hash:             ex.Hash,
				AlreadyPosted:    true,
				TotalDebit:       totals[i],
				TotalCredit:      totals[i],
				LineCount:        len(in.Postings),
			}
			continue
		}
		if firstIdx, dup := seenInBatch[ek]; dup {
			results[i] = JournalEntryResult{
				EntryID:          normalized[firstIdx].EntryID,
				EntityID:         in.EntityID,
				LegalEntityID:    normalized[firstIdx].LegalEntityID,
				PeriodID:         normalized[firstIdx].PeriodID,
				OriginalPeriodID: normalized[firstIdx].OriginalPeriodID,
				AlreadyPosted:    true,
				TotalDebit:       totals[i],
				TotalCredit:      totals[i],
				LineCount:        len(in.Postings),
			}
			// PrevHash/Hash filled after pending items are hashed below.
			seenInBatch["dup:"+fmt.Sprintf("%d", i)] = firstIdx
			continue
		}
		seenInBatch[ek] = i
		pendingIdx = append(pendingIdx, i)
	}

	if len(pendingIdx) == 0 {
		return results, nil
	}

	// Enforce Period Close locks and late operational event rollover inside txn.
	periodChecker := activePeriodLockChecker()
	for _, idx := range pendingIdx {
		in := &normalized[idx]
		effPeriod, origPeriod, pErr := periodChecker(ctx, txn, in.EntityID, in.OccurredAt, in.SourceType, in.PeriodID)
		if pErr != nil {
			return nil, pErr
		}
		if strings.TrimSpace(effPeriod) != "" {
			in.PeriodID = strings.TrimSpace(effPeriod)
		}
		if strings.TrimSpace(origPeriod) != "" {
			in.OriginalPeriodID = strings.TrimSpace(origPeriod)
		} else {
			in.OriginalPeriodID = ""
		}
	}

	// Verify reversal targets and prevent duplicate reversals on the same original entry.
	if err := validateReversalsInTxn(ctx, txn, normalized, pendingIdx); err != nil {
		return nil, err
	}

	// Validate accounts against GlAccounts (falling back to shipped system COA for unseeded tests).
	accountMetaMap, err := readAccountMetaInTxn(ctx, txn, normalized, pendingIdx)
	if err != nil {
		return nil, err
	}
	for _, idx := range pendingIdx {
		in := normalized[idx]
		for _, p := range in.Postings {
			key := in.EntityID + "\x00" + p.AccountCode
			if meta, ok := accountMetaMap[key]; ok {
				if meta.ActiveFrom != nil && in.OccurredAt.Before(*meta.ActiveFrom) {
					return nil, fmt.Errorf("%w: %s active from %s, occurred_at %s", ErrInactiveAccount, p.AccountCode, meta.ActiveFrom.Format(time.RFC3339), in.OccurredAt.Format(time.RFC3339))
				}
				if meta.ActiveTo != nil && !in.OccurredAt.Before(*meta.ActiveTo) {
					return nil, fmt.Errorf("%w: %s active until %s, occurred_at %s", ErrInactiveAccount, p.AccountCode, meta.ActiveTo.Format(time.RFC3339), in.OccurredAt.Format(time.RFC3339))
				}
				continue
			}
			if _, isSys := LookupSystemAccount(p.AccountCode); !isSys {
				return nil, fmt.Errorf("%w: %s (entity=%s)", ErrUnknownAccount, p.AccountCode, in.EntityID)
			}
		}
	}

	// Read latest hash chain head once per distinct EntityId with pending writes.
	chainTip := make(map[string]string)
	for _, idx := range pendingIdx {
		eid := normalized[idx].EntityID
		if _, ok := chainTip[eid]; ok {
			continue
		}
		head, err := readLatestEntityHashInTxn(ctx, txn, eid)
		if err != nil {
			return nil, err
		}
		chainTip[eid] = head
	}

	muts := make([]*spanner.Mutation, 0, len(pendingIdx)*4)
	balDeltas := make(map[balanceBucketKey]*balanceDelta)
	balOrder := make([]balanceBucketKey, 0, len(pendingIdx)*2)

	for _, idx := range pendingIdx {
		in := normalized[idx]
		prevHash := chainTip[in.EntityID]

		canonPostings := make([]CanonicalPosting, 0, len(in.Postings))
		for pIdx, p := range in.Postings {
			canonPostings = append(canonPostings, CanonicalPosting{
				Line:                  int64(pIdx + 1),
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
			EntityID:         in.EntityID,
			EntryID:          in.EntryID,
			LegalEntityID:    in.LegalEntityID,
			PeriodID:         in.PeriodID,
			OriginalPeriodID: in.OriginalPeriodID,
			SourceEventID:    in.SourceEventID,
			SourceType:       in.SourceType,
			SourceID:         in.SourceID,
			OrderID:          in.OrderID,
			Memo:             in.Memo,
			Currency:         in.Currency,
			CountryPack:      in.CountryPack,
			ReversesEntryID:  in.ReversesEntryID,
			IdempotencyKey:   in.IdempotencyKey,
			PostedBy:         in.PostedBy,
			OccurredAt:       in.OccurredAt.UTC().Format(time.RFC3339Nano),
			Postings:         canonPostings,
		})
		if err != nil {
			return nil, err
		}
		entryHash := ComputeEntryHash(prevHash, canonBytes)
		chainTip[in.EntityID] = entryHash

		muts = append(muts, spanner.InsertMap("JournalEntries", map[string]any{
			"EntityId":         in.EntityID,
			"EntryId":          in.EntryID,
			"LegalEntityId":    nullIfEmpty(in.LegalEntityID),
			"PeriodId":         in.PeriodID,
			"OriginalPeriodId": nullIfEmpty(in.OriginalPeriodID),
			"SourceEventId":    nullIfEmpty(in.SourceEventID),
			"SourceType":       in.SourceType,
			"SourceId":         in.SourceID,
			"OrderId":          nullIfEmpty(in.OrderID),
			"Memo":             nullIfEmpty(in.Memo),
			"Currency":         in.Currency,
			"CountryPack":      in.CountryPack,
			"ReversesEntryId":  nullIfEmpty(in.ReversesEntryID),
			"IdempotencyKey":   in.IdempotencyKey,
			"PrevHash":         prevHash,
			"EntryHash":        entryHash,
			"PostedBy":         in.PostedBy,
			"OccurredAt":       in.OccurredAt,
			"PostedAt":         spanner.CommitTimestamp,
		}))

		for pIdx, p := range in.Postings {
			lineNum := int64(pIdx + 1)
			muts = append(muts, spanner.InsertMap("JournalPostings", map[string]any{
				"EntityId":              in.EntityID,
				"EntryId":               in.EntryID,
				"Line":                  lineNum,
				"AccountCode":           p.AccountCode,
				"Direction":             string(p.Direction),
				"AmountMinor":           p.AmountMinor,
				"Currency":              p.Currency,
				"FunctionalAmountMinor": p.FunctionalAmountMinor,
				"FxRateId":              nullIfEmpty(p.FxRateID),
				"PartyType":             nullIfEmpty(p.PartyType),
				"PartyId":               nullIfEmpty(p.PartyID),
				"WarehouseId":           nullIfEmpty(p.WarehouseID),
				"ProductId":             nullIfEmpty(p.ProductID),
				"LotId":                 nullIfEmpty(p.LotID),
				"Description":           nullIfEmpty(p.Description),
			}))

			pk := BalancePartyKey(p.PartyType, p.PartyID)
			shard := BalanceShard(in.EntityID, in.PeriodID, p.AccountCode, pk, in.EntryID)
			bk := balanceBucketKey{
				EntityID:    in.EntityID,
				PeriodID:    in.PeriodID,
				AccountCode: p.AccountCode,
				PartyKey:    pk,
				Shard:       shard,
			}
			d, exists := balDeltas[bk]
			if !exists {
				balCur := in.Currency
				if meta, ok := accountMetaMap[in.EntityID+"\x00"+p.AccountCode]; ok && strings.TrimSpace(meta.Currency) != "" {
					balCur = strings.ToUpper(strings.TrimSpace(meta.Currency))
				}
				d = &balanceDelta{Currency: balCur}
				balDeltas[bk] = d
				balOrder = append(balOrder, bk)
			}
			if p.Direction == DirectionDebit {
				d.DebitMinor += p.FunctionalAmountMinor
			} else {
				d.CreditMinor += p.FunctionalAmountMinor
			}
		}

		results[idx] = JournalEntryResult{
			EntryID:          in.EntryID,
			EntityID:         in.EntityID,
			LegalEntityID:    in.LegalEntityID,
			PeriodID:         in.PeriodID,
			OriginalPeriodID: in.OriginalPeriodID,
			PrevHash:         prevHash,
			Hash:             entryHash,
			AlreadyPosted:    false,
			TotalDebit:       totals[idx],
			TotalCredit:      totals[idx],
			LineCount:        len(in.Postings),
		}
	}

	// Copy PeriodID/OriginalPeriodID/PrevHash/Hash onto any intra-batch duplicate results.
	for i := range results {
		if firstIdx, ok := seenInBatch["dup:"+fmt.Sprintf("%d", i)]; ok {
			results[i].PeriodID = results[firstIdx].PeriodID
			results[i].OriginalPeriodID = results[firstIdx].OriginalPeriodID
			results[i].PrevHash = results[firstIdx].PrevHash
			results[i].Hash = results[firstIdx].Hash
		}
	}

	balMuts, err := buildBalanceMutationsInTxn(ctx, txn, balOrder, balDeltas)
	if err != nil {
		return nil, err
	}
	muts = append(muts, balMuts...)

	if err := txn.BufferWrite(muts); err != nil {
		return nil, fmt.Errorf("buffer gl mutations: %w", err)
	}

	var localFlush *outbox.SpannerTxnBuffer
	targetBuf := outboxBuf
	if targetBuf == nil {
		localFlush = outbox.NewSpannerTxnBuffer(txn)
		targetBuf = localFlush
	}

	for _, idx := range pendingIdx {
		in := normalized[idx]
		res := results[idx]
		ts := in.OccurredAt.UTC().Format(time.RFC3339Nano)
		stampedBuf := &entityStampedOutboxBuf{inner: targetBuf, entityID: in.EntityID}

		if in.ReversesEntryID != "" {
			ev := events.GLEntryReversedEvent{
				BaseEvent: events.BaseEvent{
					Type:      events.EventGLEntryReversed,
					Version:   1,
					Timestamp: ts,
				},
				EntryID:          res.EntryID,
				EntityID:         res.EntityID,
				PeriodID:         res.PeriodID,
				SourceType:       in.SourceType,
				SourceID:         in.SourceID,
				Currency:         in.Currency,
				TotalDebitMinor:  res.TotalDebit,
				TotalCreditMinor: res.TotalCredit,
				Hash:             res.Hash,
				ReversesEntryID:  in.ReversesEntryID,
			}
			if err := outbox.EmitJSON(ctx, stampedBuf, events.AggregateJournalEntry, res.EntryID, events.TopicLedger, ev); err != nil {
				return nil, fmt.Errorf("emit gl reversed outbox: %w", err)
			}
			continue
		}

		ev := events.GLEntryPostedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventGLEntryPosted,
				Version:   1,
				Timestamp: ts,
			},
			EntryID:          res.EntryID,
			EntityID:         res.EntityID,
			PeriodID:         res.PeriodID,
			SourceType:       in.SourceType,
			SourceID:         in.SourceID,
			Currency:         in.Currency,
			TotalDebitMinor:  res.TotalDebit,
			TotalCreditMinor: res.TotalCredit,
			Hash:             res.Hash,
		}
		if err := outbox.EmitJSON(ctx, stampedBuf, events.AggregateJournalEntry, res.EntryID, events.TopicLedger, ev); err != nil {
			return nil, fmt.Errorf("emit gl posted outbox: %w", err)
		}
	}

	if localFlush != nil {
		if err := localFlush.Flush(ctx); err != nil {
			return nil, fmt.Errorf("flush gl outbox: %w", err)
		}
	}

	return results, nil
}

func readExistingIdempotencyKeysInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	keysByEntity map[string][]string,
) (map[string]existingEntryInfo, error) {
	out := make(map[string]existingEntryInfo)
	for entityID, keys := range keysByEntity {
		uniq := uniqueStrings(keys)
		if len(uniq) == 0 {
			continue
		}
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntryId, LegalEntityId, PeriodId, OriginalPeriodId, PrevHash, EntryHash, IdempotencyKey
			      FROM JournalEntries@{FORCE_INDEX=UQ_JournalEntries_EntityIdempotency}
			      WHERE EntityId = @entityId AND IdempotencyKey IN UNNEST(@keys)`,
			Params: map[string]any{
				"entityId": entityID,
				"keys":     uniq,
			},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, fmt.Errorf("read journal idempotency: %w", err)
			}
			var info existingEntryInfo
			var legalEntityID, origPeriod spanner.NullString
			var idemKey string
			if err := row.Columns(&info.EntryID, &legalEntityID, &info.PeriodID, &origPeriod, &info.PrevHash, &info.Hash, &idemKey); err != nil {
				iter.Stop()
				return nil, fmt.Errorf("scan journal idempotency: %w", err)
			}
			info.LegalEntityID = nullStrVal(legalEntityID)
			info.OriginalPeriodID = nullStrVal(origPeriod)
			out[entityID+"\x00"+idemKey] = info
		}
		iter.Stop()
	}
	return out, nil
}

func validateReversalsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	inputs []JournalEntryInput,
	pendingIdx []int,
) error {
	seenTargets := make(map[string]struct{})
	for _, idx := range pendingIdx {
		in := inputs[idx]
		if in.ReversesEntryID == "" {
			continue
		}
		targetKey := in.EntityID + "\x00" + in.ReversesEntryID
		if _, dup := seenTargets[targetKey]; dup {
			return fmt.Errorf("%w: %s", ErrAlreadyReversed, in.ReversesEntryID)
		}
		seenTargets[targetKey] = struct{}{}

		// Verify original entry exists.
		_, err := txn.ReadRow(ctx, "JournalEntries", spanner.Key{in.EntityID, in.ReversesEntryID}, []string{"EntryId"})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("%w: target %s", ErrEntryNotFound, in.ReversesEntryID)
			}
			return fmt.Errorf("read reversal target %s: %w", in.ReversesEntryID, err)
		}

		// Verify target has not already been reversed by another entry.
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntryId
			      FROM JournalEntries@{FORCE_INDEX=Idx_JournalEntries_ByReversal,spanner_emulator.disable_query_null_filtered_index_check=true}
			      WHERE EntityId = @entityId AND ReversesEntryId = @revId AND ReversesEntryId IS NOT NULL
			      LIMIT 1`,
			Params: map[string]any{
				"entityId": in.EntityID,
				"revId":    in.ReversesEntryID,
			},
		})
		_, rErr := iter.Next()
		iter.Stop()
		if rErr == nil {
			return fmt.Errorf("%w: %s", ErrAlreadyReversed, in.ReversesEntryID)
		}
		if rErr != iterator.Done {
			return fmt.Errorf("check existing reversal for %s: %w", in.ReversesEntryID, rErr)
		}
	}
	return nil
}

func readAccountMetaInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	inputs []JournalEntryInput,
	pendingIdx []int,
) (map[string]glAccountMeta, error) {
	seen := make(map[string]struct{})
	keys := make([]spanner.Key, 0)
	for _, idx := range pendingIdx {
		in := inputs[idx]
		for _, p := range in.Postings {
			k := in.EntityID + "\x00" + p.AccountCode
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			keys = append(keys, spanner.Key{in.EntityID, p.AccountCode})
		}
	}
	out := make(map[string]glAccountMeta, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	iter := txn.Read(ctx, "GlAccounts", spanner.KeySetFromKeys(keys...), []string{
		"EntityId", "AccountCode", "Currency", "ActiveFrom", "ActiveTo",
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read GlAccounts batch: %w", err)
		}
		var eid, code, cur string
		var from, to spanner.NullTime
		if err := row.Columns(&eid, &code, &cur, &from, &to); err != nil {
			return nil, fmt.Errorf("scan GlAccounts batch: %w", err)
		}
		var meta glAccountMeta
		meta.Currency = cur
		if from.Valid {
			t := from.Time.UTC()
			meta.ActiveFrom = &t
		}
		if to.Valid {
			t := to.Time.UTC()
			meta.ActiveTo = &t
		}
		out[eid+"\x00"+strings.ToUpper(strings.TrimSpace(code))] = meta
	}
	return out, nil
}

func buildBalanceMutationsInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	order []balanceBucketKey,
	deltas map[balanceBucketKey]*balanceDelta,
) ([]*spanner.Mutation, error) {
	if len(order) == 0 {
		return nil, nil
	}
	keys := make([]spanner.Key, 0, len(order))
	for _, bk := range order {
		keys = append(keys, spanner.Key{bk.EntityID, bk.PeriodID, bk.AccountCode, bk.PartyKey, bk.Shard})
	}
	type curBal struct {
		debit    int64
		credit   int64
		currency string
	}
	existing := make(map[balanceBucketKey]curBal, len(order))
	iter := txn.Read(ctx, "GlBalances", spanner.KeySetFromKeys(keys...), []string{
		"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard", "Currency", "DebitMinor", "CreditMinor",
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read GlBalances batch: %w", err)
		}
		var bk balanceBucketKey
		var cur string
		var deb, cred int64
		if err := row.Columns(&bk.EntityID, &bk.PeriodID, &bk.AccountCode, &bk.PartyKey, &bk.Shard, &cur, &deb, &cred); err != nil {
			return nil, fmt.Errorf("scan GlBalances batch: %w", err)
		}
		existing[bk] = curBal{debit: deb, credit: cred, currency: cur}
	}

	muts := make([]*spanner.Mutation, 0, len(order))
	for _, bk := range order {
		d := deltas[bk]
		prev := existing[bk]
		cur := prev.currency
		if cur == "" {
			cur = d.Currency
		}
		muts = append(muts, spanner.InsertOrUpdateMap("GlBalances", map[string]any{
			"EntityId":    bk.EntityID,
			"PeriodId":    bk.PeriodID,
			"AccountCode": bk.AccountCode,
			"PartyKey":    bk.PartyKey,
			"Shard":       bk.Shard,
			"Currency":    cur,
			"DebitMinor":  prev.debit + d.DebitMinor,
			"CreditMinor": prev.credit + d.CreditMinor,
			"UpdatedAt":   spanner.CommitTimestamp,
		}))
	}
	return muts, nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func nullStrVal(ns spanner.NullString) string {
	if ns.Valid {
		return ns.StringVal
	}
	return ""
}

func scanJournalEntryRow(row *spanner.Row) (JournalEntryRecord, error) {
	var rec JournalEntryRecord
	var legalEntityID, origPeriod, srcEv, orderID, memo, revID spanner.NullString
	if err := row.Columns(
		&rec.EntityID, &rec.EntryID, &legalEntityID, &rec.PeriodID, &origPeriod, &srcEv, &rec.SourceType, &rec.SourceID,
		&orderID, &memo, &rec.Currency, &rec.CountryPack, &revID, &rec.IdempotencyKey,
		&rec.PrevHash, &rec.EntryHash, &rec.PostedBy, &rec.OccurredAt, &rec.PostedAt,
	); err != nil {
		return JournalEntryRecord{}, fmt.Errorf("scan journal entry: %w", err)
	}
	rec.LegalEntityID = nullStrVal(legalEntityID)
	rec.OriginalPeriodID = nullStrVal(origPeriod)
	rec.SourceEventID = nullStrVal(srcEv)
	rec.OrderID = nullStrVal(orderID)
	rec.Memo = nullStrVal(memo)
	rec.ReversesEntryID = nullStrVal(revID)
	rec.OccurredAt = rec.OccurredAt.UTC()
	rec.PostedAt = rec.PostedAt.UTC()
	return rec, nil
}
