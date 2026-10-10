package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Repository persists claims + evidences.
type Repository interface {
	CreateClaim(ctx context.Context, c Claim, emit func(outbox.TxnBuffer) error) error
	UpdateClaim(ctx context.Context, c Claim, emit func(outbox.TxnBuffer) error) error
	// TransitionStatus CAS-updates Status only when current status is in fromStatuses.
	// Returns ErrInvalidClaimState when no row matched.
	TransitionStatus(ctx context.Context, claimID string, fromStatuses []Status, to Claim, emit func(outbox.TxnBuffer) error) error
	GetClaim(ctx context.Context, claimID string) (Claim, bool, error)
	ListByOrder(ctx context.Context, orderID string) ([]Claim, error)
	ListBySupplier(ctx context.Context, supplierID string, status Status, limit int) ([]Claim, error)
}

// SpannerRepository is the production store.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed claims repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// CreateClaim inserts claim + evidences and optional outbox events atomically.
func (r *SpannerRepository) CreateClaim(ctx context.Context, c Claim, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("claims: nil spanner client")
	}
	lineJSON, err := json.Marshal(c.LineItems)
	if err != nil {
		return fmt.Errorf("marshal line items: %w", err)
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{
			spanner.InsertMap("Claims", map[string]any{
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
			}),
		}
		for _, e := range c.Evidences {
			row := map[string]any{
				"ClaimId":      c.ClaimID,
				"EvidenceId":   e.EvidenceID,
				"EvidenceType": string(e.EvidenceType),
				"Uri":          e.URI,
				"MimeType":     nullableStr(e.MimeType),
				"CapturedBy":   nullableStr(e.CapturedBy),
				"CreatedAt":    spanner.CommitTimestamp,
			}
			if e.CapturedAt != nil {
				row["CapturedAt"] = e.CapturedAt.UTC()
			}
			muts = append(muts, spanner.InsertMap("ClaimEvidences", row))
		}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			for _, ev := range buf.events {
				muts = append(muts, outboxMutation(ev))
			}
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func outboxMutation(e outbox.Event) *spanner.Mutation {
	return spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e))
}

// UpdateClaim patches claim status/resolution fields and emits outbox events.
func (r *SpannerRepository) UpdateClaim(ctx context.Context, c Claim, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("claims: nil spanner client")
	}
	lineJSON, err := json.Marshal(c.LineItems)
	if err != nil {
		return fmt.Errorf("marshal line items: %w", err)
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"ClaimId":        c.ClaimID,
			"Status":         string(c.Status),
			"Description":    nullableStr(c.Description),
			"AmountMinor":    c.AmountMinor,
			"Currency":       nullableStr(c.Currency),
			"LineItemsJSON":  lineJSON,
			"ResolutionNote": nullableStr(c.ResolutionNote),
			"ResolvedBy":     nullableStr(c.ResolvedBy),
			"UpdatedAt":      spanner.CommitTimestamp,
		}
		if c.ResolvedAt != nil {
			row["ResolvedAt"] = c.ResolvedAt.UTC()
		}
		muts := []*spanner.Mutation{spanner.UpdateMap("Claims", row)}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			for _, ev := range buf.events {
				muts = append(muts, outboxMutation(ev))
			}
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// TransitionStatus performs a status CAS then writes resolution fields + outbox.
func (r *SpannerRepository) TransitionStatus(ctx context.Context, claimID string, fromStatuses []Status, to Claim, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("claims: nil spanner client")
	}
	claimID = strings.TrimSpace(claimID)
	if claimID == "" || len(fromStatuses) == 0 {
		return ErrInvalidClaimState
	}
	fromStrs := make([]string, 0, len(fromStatuses))
	for _, st := range fromStatuses {
		fromStrs = append(fromStrs, string(st))
	}
	lineJSON, err := json.Marshal(to.LineItems)
	if err != nil {
		return fmt.Errorf("marshal line items: %w", err)
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Claims", spanner.Key{claimID}, []string{"Status"})
		if err != nil {
			if err == spanner.ErrRowNotFound || strings.Contains(err.Error(), "not found") {
				return ErrClaimNotFound
			}
			return err
		}
		var cur string
		if err := row.Columns(&cur); err != nil {
			return err
		}
		okFrom := false
		for _, st := range fromStrs {
			if cur == st {
				okFrom = true
				break
			}
		}
		if !okFrom {
			return fmt.Errorf("%w: status %s", ErrInvalidClaimState, cur)
		}
		update := map[string]any{
			"ClaimId":        claimID,
			"Status":         string(to.Status),
			"AmountMinor":    to.AmountMinor,
			"LineItemsJSON":  lineJSON,
			"ResolutionNote": nullableStr(to.ResolutionNote),
			"ResolvedBy":     nullableStr(to.ResolvedBy),
			"UpdatedAt":      spanner.CommitTimestamp,
		}
		if to.ResolvedAt != nil {
			update["ResolvedAt"] = to.ResolvedAt.UTC()
		}
		muts := []*spanner.Mutation{spanner.UpdateMap("Claims", update)}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			for _, ev := range buf.events {
				muts = append(muts, outboxMutation(ev))
			}
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetClaim loads one claim with evidences.
func (r *SpannerRepository) GetClaim(ctx context.Context, claimID string) (Claim, bool, error) {
	if r == nil || r.client == nil {
		return Claim{}, false, fmt.Errorf("claims: nil spanner client")
	}
	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		return Claim{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "Claims", spanner.Key{claimID}, []string{
		"ClaimId", "OrderId", "SupplierId", "RetailerId", "FiledBy", "FiledByRole",
		"ClaimType", "Status", "Description", "AmountMinor", "Currency", "LineItemsJSON",
		"ResolutionNote", "ResolvedBy", "ResolvedAt", "Source", "TraceId", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		if err == spanner.ErrRowNotFound || strings.Contains(err.Error(), "not found") {
			return Claim{}, false, nil
		}
		return Claim{}, false, err
	}
	c, err := scanClaim(row)
	if err != nil {
		return Claim{}, false, err
	}
	evs, err := r.listEvidences(ctx, claimID)
	if err != nil {
		return Claim{}, false, err
	}
	c.Evidences = evs
	return c, true, nil
}

// ListByOrder returns claims for an order newest first.
func (r *SpannerRepository) ListByOrder(ctx context.Context, orderID string) ([]Claim, error) {
	return r.queryClaims(ctx, spanner.Statement{
		SQL: `SELECT ClaimId, OrderId, SupplierId, RetailerId, FiledBy, FiledByRole,
			ClaimType, Status, Description, AmountMinor, Currency, LineItemsJSON,
			ResolutionNote, ResolvedBy, ResolvedAt, Source, TraceId, CreatedAt, UpdatedAt
			FROM Claims@{FORCE_INDEX=Idx_Claims_ByOrderCreated}
			WHERE OrderId = @oid
			ORDER BY CreatedAt DESC
			LIMIT 50`,
		Params: map[string]any{"oid": strings.TrimSpace(orderID)},
	})
}

// ListBySupplier returns claims for a supplier, optional status filter.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string, status Status, limit int) ([]Claim, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, nil
	}
	if status != "" && status.Valid() {
		return r.queryClaims(ctx, spanner.Statement{
			SQL: `SELECT ClaimId, OrderId, SupplierId, RetailerId, FiledBy, FiledByRole,
				ClaimType, Status, Description, AmountMinor, Currency, LineItemsJSON,
				ResolutionNote, ResolvedBy, ResolvedAt, Source, TraceId, CreatedAt, UpdatedAt
				FROM Claims@{FORCE_INDEX=Idx_Claims_BySupplierStatus}
				WHERE SupplierId = @sid AND Status = @st
				ORDER BY CreatedAt DESC
				LIMIT @lim`,
			Params: map[string]any{"sid": supplierID, "st": string(status), "lim": int64(limit)},
		})
	}
	return r.queryClaims(ctx, spanner.Statement{
		SQL: `SELECT ClaimId, OrderId, SupplierId, RetailerId, FiledBy, FiledByRole,
			ClaimType, Status, Description, AmountMinor, Currency, LineItemsJSON,
			ResolutionNote, ResolvedBy, ResolvedAt, Source, TraceId, CreatedAt, UpdatedAt
			FROM Claims@{FORCE_INDEX=Idx_Claims_BySupplierStatus}
			WHERE SupplierId = @sid
			ORDER BY CreatedAt DESC
			LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
	})
}

func (r *SpannerRepository) queryClaims(ctx context.Context, stmt spanner.Statement) ([]Claim, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("claims: nil spanner client")
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Claim
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Claims") {
				return nil, nil
			}
			return nil, err
		}
		c, err := scanClaim(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *SpannerRepository) listEvidences(ctx context.Context, claimID string) ([]Evidence, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ClaimId, EvidenceId, EvidenceType, Uri, MimeType, CapturedAt, CapturedBy, CreatedAt
			FROM ClaimEvidences WHERE ClaimId = @cid`,
		Params: map[string]any{"cid": claimID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []Evidence
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e Evidence
		var mime, by spanner.NullString
		var capAt spanner.NullTime
		var created time.Time
		var etype string
		if err := row.Columns(&e.ClaimID, &e.EvidenceID, &etype, &e.URI, &mime, &capAt, &by, &created); err != nil {
			return nil, err
		}
		e.EvidenceType = EvidenceType(etype)
		if mime.Valid {
			e.MimeType = mime.StringVal
		}
		if by.Valid {
			e.CapturedBy = by.StringVal
		}
		if capAt.Valid {
			t := capAt.Time.UTC()
			e.CapturedAt = &t
		}
		e.CreatedAt = created.UTC()
		out = append(out, e)
	}
	return out, nil
}

func scanClaim(row *spanner.Row) (Claim, error) {
	var c Claim
	var desc, cur, resNote, resBy, trace spanner.NullString
	var amount spanner.NullInt64
	var lineJSON []byte
	var resAt spanner.NullTime
	var created, updated time.Time
	var ctype, status, source string
	if err := row.Columns(
		&c.ClaimID, &c.OrderID, &c.SupplierID, &c.RetailerID, &c.FiledBy, &c.FiledByRole,
		&ctype, &status, &desc, &amount, &cur, &lineJSON,
		&resNote, &resBy, &resAt, &source, &trace, &created, &updated,
	); err != nil {
		return Claim{}, err
	}
	c.ClaimType = ClaimType(ctype)
	c.Status = Status(status)
	c.Source = Source(source)
	if desc.Valid {
		c.Description = desc.StringVal
	}
	if amount.Valid {
		c.AmountMinor = amount.Int64
	}
	if cur.Valid {
		c.Currency = cur.StringVal
	}
	if len(lineJSON) > 0 {
		_ = json.Unmarshal(lineJSON, &c.LineItems)
	}
	if resNote.Valid {
		c.ResolutionNote = resNote.StringVal
	}
	if resBy.Valid {
		c.ResolvedBy = resBy.StringVal
	}
	if resAt.Valid {
		t := resAt.Time.UTC()
		c.ResolvedAt = &t
	}
	if trace.Valid {
		c.TraceID = trace.StringVal
	}
	c.CreatedAt = created.UTC()
	c.UpdatedAt = updated.UTC()
	return c, nil
}

func nullableStr(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

// MemoryRepository is a test double.
type MemoryRepository struct {
	byID map[string]Claim
}

// NewMemoryRepository builds an in-memory claims store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{byID: make(map[string]Claim)}
}

// CreateClaim stores the claim in memory.
func (m *MemoryRepository) CreateClaim(_ context.Context, c Claim, emit func(outbox.TxnBuffer) error) error {
	if m.byID == nil {
		m.byID = make(map[string]Claim)
	}
	if emit != nil {
		buf := &spannerTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
	}
	m.byID[c.ClaimID] = c
	return nil
}

// GetClaim loads from memory.
func (m *MemoryRepository) GetClaim(_ context.Context, claimID string) (Claim, bool, error) {
	c, ok := m.byID[strings.TrimSpace(claimID)]
	return c, ok, nil
}

// ListByOrder lists by order id.
func (m *MemoryRepository) ListByOrder(_ context.Context, orderID string) ([]Claim, error) {
	var out []Claim
	for _, c := range m.byID {
		if c.OrderID == orderID {
			out = append(out, c)
		}
	}
	return out, nil
}

// UpdateClaim updates an in-memory claim.
func (m *MemoryRepository) UpdateClaim(_ context.Context, c Claim, emit func(outbox.TxnBuffer) error) error {
	if m.byID == nil {
		m.byID = make(map[string]Claim)
	}
	if emit != nil {
		buf := &spannerTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
	}
	m.byID[c.ClaimID] = c
	return nil
}

// TransitionStatus CAS for memory repo.
func (m *MemoryRepository) TransitionStatus(_ context.Context, claimID string, fromStatuses []Status, to Claim, emit func(outbox.TxnBuffer) error) error {
	if m.byID == nil {
		return ErrClaimNotFound
	}
	cur, ok := m.byID[strings.TrimSpace(claimID)]
	if !ok {
		return ErrClaimNotFound
	}
	okFrom := false
	for _, st := range fromStatuses {
		if cur.Status == st {
			okFrom = true
			break
		}
	}
	if !okFrom {
		return fmt.Errorf("%w: status %s", ErrInvalidClaimState, cur.Status)
	}
	if emit != nil {
		buf := &spannerTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
	}
	m.byID[to.ClaimID] = to
	return nil
}

// ListBySupplier filters memory claims.
func (m *MemoryRepository) ListBySupplier(_ context.Context, supplierID string, status Status, limit int) ([]Claim, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []Claim
	for _, c := range m.byID {
		if c.SupplierID != supplierID {
			continue
		}
		if status != "" && c.Status != status {
			continue
		}
		out = append(out, c)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
