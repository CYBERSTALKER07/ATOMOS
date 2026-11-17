package claims

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const (
	claimsOrderPageSize       = 50
	defaultSupplierClaimsPage = 50
	maxSupplierClaimsPage     = 100
)

// ErrInvalidCursor reports a malformed pagination cursor.
var ErrInvalidCursor = errors.New("claims: invalid cursor")

// SupplierClaimsPager is an optional Repository extension supporting keyset pagination.
type SupplierClaimsPager interface {
	ListBySupplierPage(ctx context.Context, supplierID string, status Status, limit int, cursor string) ([]Claim, string, error)
}

// OrderClaimsPager is an optional Repository extension supporting keyset pagination.
type OrderClaimsPager interface {
	ListByOrderPage(ctx context.Context, orderID string, limit int, cursor string) ([]Claim, string, error)
}

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
	IsEvidenceURIReused(ctx context.Context, uri string) (bool, error)
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
		var escalationMeta []byte
		if c.Escalation != nil {
			b, err := json.Marshal(c.Escalation)
			if err != nil {
				return fmt.Errorf("marshal escalation metadata: %w", err)
			}
			escalationMeta = b
		}
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

func normalizeClaimsPageLimit(limit, def int) int {
	if limit <= 0 || limit > maxSupplierClaimsPage {
		return def
	}
	return limit
}

func encodeClaimCursor(createdAt time.Time, claimID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.UTC().Format(time.RFC3339Nano) + "|" + claimID))
}

func decodeClaimCursor(cursor string) (time.Time, string, bool, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return time.Time{}, "", false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, ErrInvalidCursor
	}
	return ts.UTC(), parts[1], true, nil
}

// ListByOrder returns all claims for an order newest first by sweeping keyset pages.
func (r *SpannerRepository) ListByOrder(ctx context.Context, orderID string) ([]Claim, error) {
	var all []Claim
	cursor := ""
	for {
		page, next, err := r.ListByOrderPage(ctx, orderID, claimsOrderPageSize, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if next == "" {
			return all, nil
		}
		cursor = next
	}
}

// ListByOrderPage returns a keyset-paginated slice of claims for an order newest first.
func (r *SpannerRepository) ListByOrderPage(ctx context.Context, orderID string, limit int, cursor string) ([]Claim, string, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, "", nil
	}
	limit = normalizeClaimsPageLimit(limit, claimsOrderPageSize)
	cursorTs, cursorID, hasCursor, err := decodeClaimCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	sql := `SELECT ClaimId, OrderId, SupplierId, RetailerId, FiledBy, FiledByRole,
			ClaimType, Status, Description, AmountMinor, Currency, LineItemsJSON,
			ResolutionNote, ResolvedBy, ResolvedAt, Source, TraceId, CreatedAt, UpdatedAt
			FROM Claims@{FORCE_INDEX=Idx_Claims_ByOrderCreated}
			WHERE OrderId = @oid`
	params := map[string]any{"oid": orderID, "lim": int64(limit + 1)}
	if hasCursor {
		sql += ` AND (CreatedAt < @cursorTs OR (CreatedAt = @cursorTs AND ClaimId < @cursorId))`
		params["cursorTs"] = cursorTs
		params["cursorId"] = cursorID
	}
	sql += ` ORDER BY CreatedAt DESC, ClaimId DESC LIMIT @lim`
	rows, err := r.queryClaims(ctx, spanner.Statement{SQL: sql, Params: params})
	if err != nil {
		return nil, "", err
	}
	if len(rows) > limit {
		page := rows[:limit]
		last := page[len(page)-1]
		return page, encodeClaimCursor(last.CreatedAt, last.ClaimID), nil
	}
	return rows, "", nil
}

// ListBySupplier returns claims for a supplier, optional status filter.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string, status Status, limit int) ([]Claim, error) {
	page, _, err := r.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

// ListBySupplierPage returns a keyset-paginated slice of claims for a supplier.
func (r *SpannerRepository) ListBySupplierPage(ctx context.Context, supplierID string, status Status, limit int, cursor string) ([]Claim, string, error) {
	limit = normalizeClaimsPageLimit(limit, defaultSupplierClaimsPage)
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, "", nil
	}
	cursorTs, cursorID, hasCursor, err := decodeClaimCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	sql := `SELECT ClaimId, OrderId, SupplierId, RetailerId, FiledBy, FiledByRole,
			ClaimType, Status, Description, AmountMinor, Currency, LineItemsJSON,
			ResolutionNote, ResolvedBy, ResolvedAt, Source, TraceId, CreatedAt, UpdatedAt
			FROM Claims@{FORCE_INDEX=Idx_Claims_BySupplierStatus}
			WHERE SupplierId = @sid`
	params := map[string]any{"sid": supplierID, "lim": int64(limit + 1)}
	if status != "" && status.Valid() {
		sql += ` AND Status = @st`
		params["st"] = string(status)
	}
	if hasCursor {
		sql += ` AND (CreatedAt < @cursorTs OR (CreatedAt = @cursorTs AND ClaimId < @cursorId))`
		params["cursorTs"] = cursorTs
		params["cursorId"] = cursorID
	}
	sql += ` ORDER BY CreatedAt DESC, ClaimId DESC LIMIT @lim`
	rows, err := r.queryClaims(ctx, spanner.Statement{SQL: sql, Params: params})
	if err != nil {
		return nil, "", err
	}
	if len(rows) > limit {
		page := rows[:limit]
		last := page[len(page)-1]
		return page, encodeClaimCursor(last.CreatedAt, last.ClaimID), nil
	}
	return rows, "", nil
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

// IsEvidenceURIReused checks if the given evidence URI (or its canonical URI hash) has already
// been submitted on any claim. Fails closed on nil Spanner client or any query/table error.
func (r *SpannerRepository) IsEvidenceURIReused(ctx context.Context, uri string) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("claims: nil spanner client")
	}
	target := strings.TrimSpace(uri)
	if target == "" {
		return false, nil
	}
	return checkEvidenceReuseQuery(ctx, r.client.Single(), target, "", "")
}

func (r *SpannerRepository) listEvidences(ctx context.Context, claimID string) ([]Evidence, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ClaimId, EvidenceId, EvidenceType, Uri, MimeType, CapturedAt, CapturedBy,
			ContentSha256, UriSha256, ObjectGeneration, CreatedAt
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
		var mime, by, cSha, uSha spanner.NullString
		var gen spanner.NullInt64
		var capAt spanner.NullTime
		var created time.Time
		var etype string
		if err := row.Columns(&e.ClaimID, &e.EvidenceID, &etype, &e.URI, &mime, &capAt, &by, &cSha, &uSha, &gen, &created); err != nil {
			return nil, err
		}
		e.EvidenceType = EvidenceType(etype)
		if mime.Valid {
			e.MimeType = mime.StringVal
		}
		if by.Valid {
			e.CapturedBy = by.StringVal
		}
		if cSha.Valid {
			e.ContentSha256 = cSha.StringVal
		}
		if uSha.Valid {
			e.UriSha256 = uSha.StringVal
		}
		if gen.Valid {
			e.ObjectGeneration = gen.Int64
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

func sortClaimsNewestFirst(claimsList []Claim) {
	sort.Slice(claimsList, func(i, j int) bool {
		if !claimsList[i].CreatedAt.Equal(claimsList[j].CreatedAt) {
			return claimsList[i].CreatedAt.After(claimsList[j].CreatedAt)
		}
		return claimsList[i].ClaimID > claimsList[j].ClaimID
	})
}

func paginateMemoryClaims(claimsList []Claim, limit int, cursor string) ([]Claim, string, error) {
	sortClaimsNewestFirst(claimsList)
	cursorTs, cursorID, hasCursor, err := decodeClaimCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	filtered := claimsList
	if hasCursor {
		filtered = make([]Claim, 0, len(claimsList))
		for _, c := range claimsList {
			ts := c.CreatedAt.UTC()
			if ts.Before(cursorTs) || (ts.Equal(cursorTs) && c.ClaimID < cursorID) {
				filtered = append(filtered, c)
			}
		}
	}
	if len(filtered) > limit {
		page := filtered[:limit]
		last := page[len(page)-1]
		return page, encodeClaimCursor(last.CreatedAt, last.ClaimID), nil
	}
	return filtered, "", nil
}

// ListByOrder lists by order id.
func (m *MemoryRepository) ListByOrder(_ context.Context, orderID string) ([]Claim, error) {
	var out []Claim
	for _, c := range m.byID {
		if c.OrderID == orderID {
			out = append(out, c)
		}
	}
	sortClaimsNewestFirst(out)
	return out, nil
}

// ListByOrderPage lists claims for an order with keyset cursor pagination.
func (m *MemoryRepository) ListByOrderPage(ctx context.Context, orderID string, limit int, cursor string) ([]Claim, string, error) {
	all, err := m.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, "", err
	}
	limit = normalizeClaimsPageLimit(limit, claimsOrderPageSize)
	return paginateMemoryClaims(all, limit, cursor)
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
func (m *MemoryRepository) ListBySupplier(ctx context.Context, supplierID string, status Status, limit int) ([]Claim, error) {
	page, _, err := m.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

// ListBySupplierPage filters memory claims with keyset cursor pagination.
func (m *MemoryRepository) ListBySupplierPage(_ context.Context, supplierID string, status Status, limit int, cursor string) ([]Claim, string, error) {
	limit = normalizeClaimsPageLimit(limit, defaultSupplierClaimsPage)
	var out []Claim
	for _, c := range m.byID {
		if c.SupplierID != supplierID {
			continue
		}
		if status != "" && c.Status != status {
			continue
		}
		out = append(out, c)
	}
	return paginateMemoryClaims(out, limit, cursor)
}

// IsEvidenceURIReused checks if the given URI (or canonical URI hash) is already associated
// with any in-memory claim.
func (m *MemoryRepository) IsEvidenceURIReused(ctx context.Context, uri string) (bool, error) {
	if m == nil || m.byID == nil {
		return false, nil
	}
	target := strings.TrimSpace(uri)
	if target == "" {
		return false, nil
	}
	return m.IsEvidenceReused(ctx, target, "", "")
}

