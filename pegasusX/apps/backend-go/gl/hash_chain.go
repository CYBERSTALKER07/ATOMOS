package gl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// GenesisHash is the PrevHash of the first journal entry for an entity (64 zeros).
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// CanonicalPosting is the deterministic posting representation included in the
// SHA-256 hash chain payload.
type CanonicalPosting struct {
	Line                  int64  `json:"line"`
	AccountCode           string `json:"account_code"`
	Direction             string `json:"direction"`
	AmountMinor           int64  `json:"amount_minor"`
	Currency              string `json:"currency"`
	FunctionalAmountMinor int64  `json:"functional_amount_minor"`
	FxRateID              string `json:"fx_rate_id,omitempty"`
	PartyType             string `json:"party_type,omitempty"`
	PartyID               string `json:"party_id,omitempty"`
	WarehouseID           string `json:"warehouse_id,omitempty"`
	ProductID             string `json:"product_id,omitempty"`
	LotID                 string `json:"lot_id,omitempty"`
	Description           string `json:"description,omitempty"`
}

// CanonicalEntryPayload is the deterministic pre-commit journal payload hashed as
// SHA256(PrevHash || canonical_json).
type CanonicalEntryPayload struct {
	EntityID         string             `json:"entity_id"`
	EntryID          string             `json:"entry_id"`
	LegalEntityID    string             `json:"legal_entity_id,omitempty"`
	PeriodID         string             `json:"period_id"`
	OriginalPeriodID string             `json:"original_period_id,omitempty"`
	SourceEventID    string             `json:"source_event_id,omitempty"`
	SourceType       string             `json:"source_type"`
	SourceID         string             `json:"source_id"`
	OrderID          string             `json:"order_id,omitempty"`
	Memo             string             `json:"memo,omitempty"`
	Currency         string             `json:"currency"`
	CountryPack      string             `json:"country_pack"`
	ReversesEntryID  string             `json:"reverses_entry_id,omitempty"`
	IdempotencyKey   string             `json:"idempotency_key"`
	PostedBy         string             `json:"posted_by"`
	OccurredAt       string             `json:"occurred_at"`
	Postings         []CanonicalPosting `json:"postings"`
}

// CanonicalEntryBytes serializes payload deterministically with Postings sorted by Line ascending.
func CanonicalEntryBytes(payload CanonicalEntryPayload) ([]byte, error) {
	cp := payload
	cp.Postings = make([]CanonicalPosting, len(payload.Postings))
	copy(cp.Postings, payload.Postings)
	sort.Slice(cp.Postings, func(i, j int) bool {
		return cp.Postings[i].Line < cp.Postings[j].Line
	})
	raw, err := json.Marshal(cp)
	if err != nil {
		return nil, fmt.Errorf("gl canonical json: %w", err)
	}
	return raw, nil
}

// ComputeEntryHash computes SHA256(PrevHash || canonical_json) as lowercase 64-char hex.
func ComputeEntryHash(prevHash string, canonicalJSON []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(prevHash))
	_, _ = h.Write(canonicalJSON)
	return hex.EncodeToString(h.Sum(nil))
}

// ChainVerificationResult summarizes cryptographic verification of an entity's GL hash chain.
type ChainVerificationResult struct {
	EntityID      string `json:"entity_id"`
	Valid         bool   `json:"valid"`
	EntriesCount  int    `json:"entries_count"`
	HeadHash      string `json:"head_hash"`
	BrokenEntryID string `json:"broken_entry_id,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
}

type chainTipRow struct {
	EntryID   string
	PostedAt  time.Time
	PrevHash  string
	EntryHash string
}

// readLatestEntityHashInTxn reads the current hash-chain head for entityID inside txn.
// When multiple entries share the same commit timestamp (e.g., a batch commit), it resolves
// the topological sink whose EntryHash is not referenced as any sibling's PrevHash.
func readLatestEntityHashInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID string) (string, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntryId, PostedAt, PrevHash, EntryHash
		      FROM JournalEntries@{FORCE_INDEX=Idx_JournalEntries_ByEntityPosted}
		      WHERE EntityId = @entityId
		      ORDER BY PostedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"entityId": entityID},
	})
	defer iter.Stop()

	var rows []chainTipRow
	var maxPostedAt time.Time
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read entity chain head: %w", err)
		}
		var r chainTipRow
		if err := row.Columns(&r.EntryID, &r.PostedAt, &r.PrevHash, &r.EntryHash); err != nil {
			return "", fmt.Errorf("scan entity chain head: %w", err)
		}
		if len(rows) == 0 {
			maxPostedAt = r.PostedAt
		}
		if !r.PostedAt.Equal(maxPostedAt) {
			break
		}
		rows = append(rows, r)
	}
	if len(rows) == 0 {
		return GenesisHash, nil
	}
	if len(rows) == 1 {
		return rows[0].EntryHash, nil
	}
	// If a single batch wrote >=100 rows at maxPostedAt, fetch all rows at maxPostedAt.
	if len(rows) == 100 {
		allIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntryId, PostedAt, PrevHash, EntryHash
			      FROM JournalEntries@{FORCE_INDEX=Idx_JournalEntries_ByEntityPosted}
			      WHERE EntityId = @entityId AND PostedAt = @postedAt`,
			Params: map[string]any{"entityId": entityID, "postedAt": maxPostedAt},
		})
		defer allIter.Stop()
		rows = rows[:0]
		for {
			row, err := allIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return "", fmt.Errorf("read full tie-break batch: %w", err)
			}
			var r chainTipRow
			if err := row.Columns(&r.EntryID, &r.PostedAt, &r.PrevHash, &r.EntryHash); err != nil {
				return "", fmt.Errorf("scan tie-break row: %w", err)
			}
			rows = append(rows, r)
		}
	}

	return resolveBatchTipHash(rows), nil
}

func resolveBatchTipHash(rows []chainTipRow) string {
	prevSet := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		prevSet[r.PrevHash] = struct{}{}
	}
	var sinks []chainTipRow
	for _, r := range rows {
		if _, isParent := prevSet[r.EntryHash]; !isParent {
			sinks = append(sinks, r)
		}
	}
	if len(sinks) > 0 {
		sort.Slice(sinks, func(i, j int) bool {
			return sinks[i].EntryID > sinks[j].EntryID
		})
		return sinks[0].EntryHash
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].EntryID > rows[j].EntryID
	})
	return rows[0].EntryHash
}

// VerifyEntityHashChain reads all JournalEntries and JournalPostings for entityID in a
// consistent ReadOnlyTransaction snapshot and verifies every PrevHash -> EntryHash link
// from GenesisHash onward.
func VerifyEntityHashChain(ctx context.Context, client *spanner.Client, entityID string) (ChainVerificationResult, error) {
	if client == nil {
		return ChainVerificationResult{}, fmt.Errorf("spanner client unavailable")
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return ChainVerificationResult{}, fmt.Errorf("%w: entity_id is required", ErrInvalidJournalInput)
	}

	var entries []JournalEntryRecord
	postingsByEntry := make(map[string][]CanonicalPosting)

	err := spannerutils.RunReadOnlyTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		eIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntityId, EntryId, LegalEntityId, PeriodId, OriginalPeriodId, SourceEventId, SourceType, SourceId, OrderId,
			             Memo, Currency, CountryPack, ReversesEntryId, IdempotencyKey, PrevHash, EntryHash,
			             PostedBy, OccurredAt, PostedAt
			      FROM JournalEntries
			      WHERE EntityId = @entityId
			      ORDER BY PostedAt ASC, EntryId ASC`,
			Params: map[string]any{"entityId": entityID},
		})
		defer eIter.Stop()
		for {
			row, err := eIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query journal entries: %w", err)
			}
			rec, err := scanJournalEntryRow(row)
			if err != nil {
				return err
			}
			entries = append(entries, rec)
		}

		pIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT EntryId, Line, AccountCode, Direction, AmountMinor, Currency,
			             FunctionalAmountMinor, FxRateId, PartyType, PartyId, WarehouseId,
			             ProductId, LotId, Description
			      FROM JournalPostings
			      WHERE EntityId = @entityId
			      ORDER BY EntryId ASC, Line ASC`,
			Params: map[string]any{"entityId": entityID},
		})
		defer pIter.Stop()
		for {
			row, err := pIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("query journal postings: %w", err)
			}
			var entryID string
			var cp CanonicalPosting
			var fxRateID, partyType, partyID, warehouseID, productID, lotID, desc spanner.NullString
			if err := row.Columns(
				&entryID, &cp.Line, &cp.AccountCode, &cp.Direction, &cp.AmountMinor, &cp.Currency,
				&cp.FunctionalAmountMinor, &fxRateID, &partyType, &partyID, &warehouseID,
				&productID, &lotID, &desc,
			); err != nil {
				return fmt.Errorf("scan journal posting: %w", err)
			}
			cp.FxRateID = nullStrVal(fxRateID)
			cp.PartyType = nullStrVal(partyType)
			cp.PartyID = nullStrVal(partyID)
			cp.WarehouseID = nullStrVal(warehouseID)
			cp.ProductID = nullStrVal(productID)
			cp.LotID = nullStrVal(lotID)
			cp.Description = nullStrVal(desc)
			postingsByEntry[entryID] = append(postingsByEntry[entryID], cp)
		}
		return nil
	})
	if err != nil {
		return ChainVerificationResult{}, err
	}

	res := ChainVerificationResult{
		EntityID:     entityID,
		EntriesCount: len(entries),
		HeadHash:     GenesisHash,
		Valid:        true,
	}
	if len(entries) == 0 {
		return res, nil
	}

	ordered, orderErr := orderEntriesTopologically(entries)
	if orderErr != nil {
		res.Valid = false
		res.FailureReason = orderErr.Error()
		return res, nil
	}

	expectedPrev := GenesisHash
	for _, e := range ordered {
		if e.PrevHash != expectedPrev {
			res.Valid = false
			res.BrokenEntryID = e.EntryID
			res.FailureReason = fmt.Sprintf("prev_hash mismatch: got %s want %s", e.PrevHash, expectedPrev)
			return res, nil
		}
		payload := CanonicalEntryPayload{
			EntityID:         e.EntityID,
			EntryID:          e.EntryID,
			LegalEntityID:    e.LegalEntityID,
			PeriodID:         e.PeriodID,
			OriginalPeriodID: e.OriginalPeriodID,
			SourceEventID:    e.SourceEventID,
			SourceType:       e.SourceType,
			SourceID:         e.SourceID,
			OrderID:          e.OrderID,
			Memo:             e.Memo,
			Currency:         e.Currency,
			CountryPack:      e.CountryPack,
			ReversesEntryID:  e.ReversesEntryID,
			IdempotencyKey:   e.IdempotencyKey,
			PostedBy:         e.PostedBy,
			OccurredAt:       e.OccurredAt.UTC().Format(time.RFC3339Nano),
			Postings:         postingsByEntry[e.EntryID],
		}
		canonical, err := CanonicalEntryBytes(payload)
		if err != nil {
			res.Valid = false
			res.BrokenEntryID = e.EntryID
			res.FailureReason = err.Error()
			return res, nil
		}
		expectedHash := ComputeEntryHash(e.PrevHash, canonical)
		if e.EntryHash != expectedHash {
			res.Valid = false
			res.BrokenEntryID = e.EntryID
			res.FailureReason = fmt.Sprintf("entry_hash mismatch: got %s want %s", e.EntryHash, expectedHash)
			return res, nil
		}
		expectedPrev = e.EntryHash
		res.HeadHash = e.EntryHash
	}
	return res, nil
}

// orderEntriesTopologically orders entries by PostedAt ascending and resolves exact-timestamp
// batch groups via their PrevHash -> EntryHash link order.
func orderEntriesTopologically(entries []JournalEntryRecord) ([]JournalEntryRecord, error) {
	ordered := make([]JournalEntryRecord, 0, len(entries))
	prevTip := GenesisHash
	i := 0
	for i < len(entries) {
		j := i + 1
		for j < len(entries) && entries[j].PostedAt.Equal(entries[i].PostedAt) {
			j++
		}
		group := entries[i:j]
		if len(group) == 1 {
			ordered = append(ordered, group[0])
			prevTip = group[0].EntryHash
			i = j
			continue
		}
		byPrev := make(map[string]JournalEntryRecord, len(group))
		for _, g := range group {
			if _, dup := byPrev[g.PrevHash]; dup {
				return nil, fmt.Errorf("fork detected at prev_hash %s near entry %s", g.PrevHash, g.EntryID)
			}
			byPrev[g.PrevHash] = g
		}
		cur := prevTip
		for step := 0; step < len(group); step++ {
			next, ok := byPrev[cur]
			if !ok {
				return nil, fmt.Errorf("broken batch link: missing entry with prev_hash %s", cur)
			}
			ordered = append(ordered, next)
			cur = next.EntryHash
		}
		prevTip = cur
		i = j
	}
	return ordered, nil
}
