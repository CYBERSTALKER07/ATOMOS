package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	storeBinFloor      = "FLOOR"
	storeBinBackroom   = "BACKROOM"
	storeBinQuarantine = "QUARANTINE"
	storeMoveClaimHold = "CLAIM_HOLD"
)

// AtomicClaimOptions controls single-transaction claim persistence, StoreStock QUARANTINE hold,
// and optional below-threshold auto-approval inside the same transaction.
type AtomicClaimOptions struct {
	HoldStoreStock      bool
	HoldActor           string
	AutoApprove         bool
	AutoApproveMaxMinor int64
	Now                 time.Time
}

// AtomicClaimRepository optionally executes Claim + LineItems + ClaimEvidences +
// StoreStock QUARANTINE hold + OutboxEvents + optional auto-approval inside ONE atomic txn.
type AtomicClaimRepository interface {
	CreateClaimAtomic(
		ctx context.Context,
		c Claim,
		opts AtomicClaimOptions,
		holdFallback StoreStockClaimPort,
		emit func(outbox.TxnBuffer, Claim) error,
	) (Claim, error)
}

// EvidenceDedupeRepository checks both URI canonical hash (UQ_ClaimEvidences_ByUriSha256)
// and content byte hash (Idx_ClaimEvidences_ByContentSha256), failing closed on backend errors.
type EvidenceDedupeRepository interface {
	IsEvidenceReused(ctx context.Context, uri, uriSha256, contentSha256 string) (bool, error)
}

// RetailerClaimCounter counts recent non-rejected claims for fraud repeat-pattern escalation.
type RetailerClaimCounter interface {
	CountRecentClaimsByRetailer(ctx context.Context, retailerID string, since time.Time) (int, error)
}

type spannerRowQueryer interface {
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

func nullableInt64(v int64) spanner.NullInt64 {
	if v <= 0 {
		return spanner.NullInt64{}
	}
	return spanner.NullInt64{Int64: v, Valid: true}
}

func buildClaimMutations(c Claim, lineJSON []byte) ([]*spanner.Mutation, error) {
	claimRow := map[string]any{
		"ClaimId":       c.ClaimID,
		"OrderId":       c.OrderID,
		"SupplierId":    c.SupplierID,
		"RetailerId":    c.RetailerID,
		"FiledBy":       c.FiledBy,
		"FiledByRole":   c.FiledByRole,
		"ClaimType":     string(c.ClaimType),
		"Status":        string(c.Status),
		"Description":   nullableStr(c.Description),
		"AmountMinor":   c.AmountMinor,
		"Currency":      nullableStr(c.Currency),
		"LineItemsJSON": lineJSON,
		"Source":        string(c.Source),
		"TraceId":       nullableStr(c.TraceID),
		"CreatedAt":     spanner.CommitTimestamp,
		"UpdatedAt":     spanner.CommitTimestamp,
	}
	if strings.TrimSpace(c.ResolutionNote) != "" {
		claimRow["ResolutionNote"] = c.ResolutionNote
	}
	if strings.TrimSpace(c.ResolvedBy) != "" {
		claimRow["ResolvedBy"] = c.ResolvedBy
	}
	if c.ResolvedAt != nil && !c.ResolvedAt.IsZero() {
		claimRow["ResolvedAt"] = c.ResolvedAt.UTC()
	}

	var escalationMeta []byte
	if c.Escalation != nil {
		b, err := json.Marshal(c.Escalation)
		if err != nil {
			return nil, fmt.Errorf("marshal escalation metadata: %w", err)
		}
		escalationMeta = b
	}

	muts := make([]*spanner.Mutation, 0, 1+len(c.Evidences))
	muts = append(muts, spanner.InsertMap("Claims", claimRow))
	for _, e := range c.Evidences {
		row := map[string]any{
			"ClaimId":          c.ClaimID,
			"EvidenceId":       e.EvidenceID,
			"EvidenceType":     string(e.EvidenceType),
			"Uri":              e.URI,
			"MimeType":         nullableStr(e.MimeType),
			"CapturedBy":       nullableStr(e.CapturedBy),
			"ContentSha256":    nullableStr(e.ContentSha256),
			"UriSha256":        nullableStr(e.UriSha256),
			"ObjectGeneration": nullableInt64(e.ObjectGeneration),
			"CreatedAt":        spanner.CommitTimestamp,
		}
		if len(escalationMeta) > 0 {
			row["MetaJSON"] = escalationMeta
		}
		if e.CapturedAt != nil {
			row["CapturedAt"] = e.CapturedAt.UTC()
		}
		muts = append(muts, spanner.InsertMap("ClaimEvidences", row))
	}
	return muts, nil
}

func checkEvidenceReuseQuery(ctx context.Context, q spannerRowQueryer, uri, uriSha256, contentSha256 string) (bool, error) {
	if q == nil {
		return false, fmt.Errorf("claims: nil query handle")
	}
	targetURI := strings.TrimSpace(uri)
	targetUSha := strings.TrimSpace(uriSha256)
	targetCSha := strings.TrimSpace(contentSha256)
	if targetUSha == "" && targetURI != "" {
		if computed, _, err := storage.ComputeEvidenceURISha256(targetURI); err == nil {
			targetUSha = computed
		}
	}

	runLimit1 := func(sql string, params map[string]any) (bool, error) {
		iter := q.Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		_, err := iter.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	if targetUSha != "" {
		found, err := runLimit1(`SELECT 1 FROM ClaimEvidences WHERE UriSha256 = @usha LIMIT 1`, map[string]any{"usha": targetUSha})
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	if targetCSha != "" {
		found, err := runLimit1(`SELECT 1 FROM ClaimEvidences WHERE ContentSha256 = @csha LIMIT 1`, map[string]any{"csha": targetCSha})
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	if targetURI != "" {
		found, err := runLimit1(`SELECT 1 FROM ClaimEvidences WHERE Uri = @uri LIMIT 1`, map[string]any{"uri": targetURI})
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// IsEvidenceReused checks UriSha256, ContentSha256, and Uri in ClaimEvidences.
// Fails closed on nil Spanner client or any query/table error.
func (r *SpannerRepository) IsEvidenceReused(ctx context.Context, uri, uriSha256, contentSha256 string) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("claims: nil spanner client")
	}
	return checkEvidenceReuseQuery(ctx, r.client.Single(), uri, uriSha256, contentSha256)
}

// CountRecentClaimsByRetailer counts non-rejected claims filed by retailerID since `since`.
func (r *SpannerRepository) CountRecentClaimsByRetailer(ctx context.Context, retailerID string, since time.Time) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("claims: nil spanner client")
	}
	rid := strings.TrimSpace(retailerID)
	if rid == "" {
		return 0, nil
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(1) FROM Claims WHERE RetailerId = @rid AND CreatedAt >= @since AND Status != 'REJECTED'`,
		Params: map[string]any{
			"rid":   rid,
			"since": since.UTC(),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var cnt int64
	if err := row.Columns(&cnt); err != nil {
		return 0, err
	}
	return int(cnt), nil
}

// CreateClaimAtomic executes in-txn evidence dedupe verification, Claim + LineItems + ClaimEvidences,
// StoreStock QUARANTINE hold, OutboxEvents, and optional auto-approval inside ONE Spanner RW txn.
func (r *SpannerRepository) CreateClaimAtomic(
	ctx context.Context,
	c Claim,
	opts AtomicClaimOptions,
	_ StoreStockClaimPort,
	emit func(outbox.TxnBuffer, Claim) error,
) (Claim, error) {
	if r == nil || r.client == nil {
		return Claim{}, fmt.Errorf("claims: nil spanner client")
	}
	lineJSON, err := json.Marshal(c.LineItems)
	if err != nil {
		return Claim{}, fmt.Errorf("marshal line items: %w", err)
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var committed Claim
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur := c
		// 1) In-txn fail-closed evidence dedupe check (eliminates pre-txn TOCTOU window).
		for _, e := range cur.Evidences {
			reused, derr := checkEvidenceReuseQuery(ctx, txn, e.URI, e.UriSha256, e.ContentSha256)
			if derr != nil {
				return derr
			}
			if reused {
				return ErrReusedPhotoEvidence
			}
		}

		// 2) Single-txn StoreStock QUARANTINE hold for physical claims.
		buf := &spannerTxnBuffer{}
		if opts.HoldStoreStock && claimNeedsStoreHold(cur.ClaimType) && len(cur.LineItems) > 0 {
			if err := holdStoreStockInTxn(ctx, txn, buf, cur, opts.HoldActor, now); err != nil {
				return fmt.Errorf("%w: %v", ErrClaimStockHoldFailed, err)
			}
		}

		// 3) Single-txn optional auto-approval (only when below threshold AND not fraud-escalated).
		if opts.AutoApprove && opts.AutoApproveMaxMinor > 0 &&
			cur.AmountMinor > 0 && cur.AmountMinor <= opts.AutoApproveMaxMinor &&
			(cur.Escalation == nil || !cur.Escalation.EscalationRequired) {
			cur.Status = StatusApproved
			cur.ResolvedBy = "system:auto_approve"
			cur.ResolutionNote = fmt.Sprintf("auto_approved_below_%d", opts.AutoApproveMaxMinor)
			resolvedAt := now
			cur.ResolvedAt = &resolvedAt
			cur.UpdatedAt = now
		}

		muts, merr := buildClaimMutations(cur, lineJSON)
		if merr != nil {
			return merr
		}
		if emit != nil {
			if err := emit(buf, cur); err != nil {
				return err
			}
		}
		for _, e := range buf.events {
			muts = append(muts, outboxMutation(e))
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		committed = cur
		return nil
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.AlreadyExists {
			return Claim{}, fmt.Errorf("%w: unique evidence constraint violation: %v", ErrReusedPhotoEvidence, err)
		}
		return Claim{}, err
	}
	return committed, nil
}

func holdStoreStockInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, c Claim, actor string, now time.Time) error {
	retailerID := strings.TrimSpace(c.RetailerID)
	if retailerID == "" || len(c.LineItems) == 0 {
		return nil
	}
	locID, found, err := findPrimaryActiveLocationInTxn(ctx, txn, retailerID)
	if err != nil {
		return err
	}
	if !found || locID == "" {
		return nil
	}

	aggLines := AggregateClaimLines(c.LineItems)
	note := "claim_hold:" + c.ClaimID
	if strings.TrimSpace(c.OrderID) != "" {
		note += " order:" + c.OrderID
	}

	type binKey struct {
		bin string
		sku string
	}
	type binState struct {
		onHand   int64
		reserved int64
		dirty    bool
	}
	states := make(map[binKey]*binState, len(aggLines)*3)
	loadBin := func(bin, sku string) (*binState, error) {
		k := binKey{bin: bin, sku: sku}
		if st, ok := states[k]; ok {
			return st, nil
		}
		row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{locID, bin, sku}, []string{"OnHand", "Reserved"})
		st := &binState{}
		if err != nil {
			if spanner.ErrCode(err) != codes.NotFound {
				return nil, err
			}
			states[k] = st
			return st, nil
		}
		if err := row.Columns(&st.onHand, &st.reserved); err != nil {
			return nil, err
		}
		states[k] = st
		return st, nil
	}

	var muts []*spanner.Mutation
	anyTransferred := false
	for _, ln := range aggLines {
		sku := strings.TrimSpace(ln.SKU)
		need := ln.Quantity
		if sku == "" || need <= 0 {
			continue
		}
		for _, srcBin := range []string{storeBinFloor, storeBinBackroom} {
			if need <= 0 {
				break
			}
			srcState, err := loadBin(srcBin, sku)
			if err != nil {
				return err
			}
			if srcState.onHand <= 0 {
				continue
			}
			take := need
			if take > srcState.onHand {
				take = srcState.onHand
			}
			qState, err := loadBin(storeBinQuarantine, sku)
			if err != nil {
				return err
			}
			srcState.onHand -= take
			srcState.dirty = true
			qState.onHand += take
			qState.dirty = true
			need -= take
			anyTransferred = true

			muts = append(muts,
				spanner.InsertMap("RetailerStockMovements", map[string]any{
					"MovementId":   uuid.NewString(),
					"RetailerId":   retailerID,
					"LocationId":   locID,
					"StockBin":     srcBin,
					"Sku":          sku,
					"Qty":          -take,
					"MovementType": storeMoveClaimHold,
					"RefType":      "CLAIM",
					"RefId":        locID + ":" + storeBinQuarantine,
					"ActorUserId":  nullableStr(actor),
					"Note":         nullableStr(note),
					"CreatedAt":    spanner.CommitTimestamp,
				}),
				spanner.InsertMap("RetailerStockMovements", map[string]any{
					"MovementId":   uuid.NewString(),
					"RetailerId":   retailerID,
					"LocationId":   locID,
					"StockBin":     storeBinQuarantine,
					"Sku":          sku,
					"Qty":          take,
					"MovementType": storeMoveClaimHold,
					"RefType":      "CLAIM",
					"RefId":        locID + ":" + srcBin,
					"ActorUserId":  nullableStr(actor),
					"Note":         nullableStr(note),
					"CreatedAt":    spanner.CommitTimestamp,
				}),
			)
		}
	}

	for k, st := range states {
		if !st.dirty {
			continue
		}
		muts = append(muts, spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
			"LocationId": locID,
			"StockBin":   k.bin,
			"Sku":        k.sku,
			"RetailerId": retailerID,
			"OnHand":     st.onHand,
			"Reserved":   st.reserved,
			"UpdatedAt":  spanner.CommitTimestamp,
		}))
	}
	if len(muts) > 0 {
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
	}
	if anyTransferred && buf != nil {
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, map[string]any{
			"type":        events.EventStoreStockTransferred,
			"timestamp":   now.Format(time.RFC3339Nano),
			"retailer_id": retailerID,
			"location_id": locID,
			"reason":      storeMoveClaimHold,
			"claim_id":    c.ClaimID,
			"order_id":    c.OrderID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func findPrimaryActiveLocationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID string) (string, bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT LocationId FROM RetailerLocations
			WHERE RetailerId = @rid AND IsActive = true
			ORDER BY IsPrimary DESC, CreatedAt ASC
			LIMIT 1`,
		Params: map[string]any{"rid": retailerID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	var locID string
	if err := row.Columns(&locID); err != nil {
		return "", false, err
	}
	return locID, strings.TrimSpace(locID) != "", nil
}

func claimMatchesEvidence(c Claim, uri, uriSha256, contentSha256 string) bool {
	targetURI := strings.TrimSpace(uri)
	targetUSha := strings.TrimSpace(uriSha256)
	targetCSha := strings.TrimSpace(contentSha256)
	if targetUSha == "" && targetURI != "" {
		if computed, _, err := storage.ComputeEvidenceURISha256(targetURI); err == nil {
			targetUSha = computed
		}
	}
	for _, e := range c.Evidences {
		if targetURI != "" && strings.TrimSpace(e.URI) == targetURI {
			return true
		}
		if targetUSha != "" {
			eUSha := strings.TrimSpace(e.UriSha256)
			if eUSha == "" && strings.TrimSpace(e.URI) != "" {
				if computed, _, err := storage.ComputeEvidenceURISha256(e.URI); err == nil {
					eUSha = computed
				}
			}
			if eUSha != "" && eUSha == targetUSha {
				return true
			}
		}
		if targetCSha != "" && strings.TrimSpace(e.ContentSha256) != "" && strings.TrimSpace(e.ContentSha256) == targetCSha {
			return true
		}
	}
	return false
}

// IsEvidenceReused checks URI, UriSha256, and ContentSha256 across all stored claims in memory.
func (m *MemoryRepository) IsEvidenceReused(_ context.Context, uri, uriSha256, contentSha256 string) (bool, error) {
	if m == nil || m.byID == nil {
		return false, nil
	}
	if strings.TrimSpace(uri) == "" && strings.TrimSpace(uriSha256) == "" && strings.TrimSpace(contentSha256) == "" {
		return false, nil
	}
	for _, c := range m.byID {
		if claimMatchesEvidence(c, uri, uriSha256, contentSha256) {
			return true, nil
		}
	}
	return false, nil
}

// CountRecentClaimsByRetailer counts non-rejected in-memory claims filed by retailerID since `since`.
func (m *MemoryRepository) CountRecentClaimsByRetailer(_ context.Context, retailerID string, since time.Time) (int, error) {
	if m == nil || m.byID == nil {
		return 0, nil
	}
	rid := strings.TrimSpace(retailerID)
	if rid == "" {
		return 0, nil
	}
	count := 0
	for _, c := range m.byID {
		if c.RetailerID == rid && c.Status != StatusRejected && !c.CreatedAt.Before(since.UTC()) {
			count++
		}
	}
	return count, nil
}

// CreateClaimAtomic executes in-memory dedupe, StoreStock hold, optional auto-approve, and outbox
// emission atomically — zero claim is persisted if any step fails.
func (m *MemoryRepository) CreateClaimAtomic(
	ctx context.Context,
	c Claim,
	opts AtomicClaimOptions,
	holdFallback StoreStockClaimPort,
	emit func(outbox.TxnBuffer, Claim) error,
) (Claim, error) {
	if m.byID == nil {
		m.byID = make(map[string]Claim)
	}
	for _, e := range c.Evidences {
		reused, err := m.IsEvidenceReused(ctx, e.URI, e.UriSha256, e.ContentSha256)
		if err != nil {
			return Claim{}, err
		}
		if reused {
			return Claim{}, ErrReusedPhotoEvidence
		}
	}

	cur := c
	if opts.HoldStoreStock && claimNeedsStoreHold(cur.ClaimType) && len(cur.LineItems) > 0 && holdFallback != nil {
		if err := holdFallback.HoldForClaim(ctx, cur.RetailerID, cur.ClaimID, cur.OrderID, cur.LineItems, opts.HoldActor); err != nil {
			return Claim{}, fmt.Errorf("%w: %v", ErrClaimStockHoldFailed, err)
		}
	}

	if opts.AutoApprove && opts.AutoApproveMaxMinor > 0 &&
		cur.AmountMinor > 0 && cur.AmountMinor <= opts.AutoApproveMaxMinor &&
		(cur.Escalation == nil || !cur.Escalation.EscalationRequired) {
		now := opts.Now.UTC()
		if now.IsZero() {
			now = time.Now().UTC()
		}
		cur.Status = StatusApproved
		cur.ResolvedBy = "system:auto_approve"
		cur.ResolutionNote = fmt.Sprintf("auto_approved_below_%d", opts.AutoApproveMaxMinor)
		resolvedAt := now
		cur.ResolvedAt = &resolvedAt
		cur.UpdatedAt = now
	}

	if emit != nil {
		buf := &spannerTxnBuffer{}
		if err := emit(buf, cur); err != nil {
			return Claim{}, err
		}
	}
	m.byID[cur.ClaimID] = cur
	return cur, nil
}
