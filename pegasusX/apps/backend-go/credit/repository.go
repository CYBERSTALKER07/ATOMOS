package credit

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ErrInvalidCursor reports a malformed credit pagination cursor.
var ErrInvalidCursor = errors.New("credit: invalid cursor")

// ProfilePager is an optional Repository extension supporting SQL keyset cursor pagination.
type ProfilePager interface {
	ListBySupplierPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Profile, string, error)
	ListDelinquentCandidatesPage(ctx context.Context, limit int, cursor string) ([]Profile, string, error)
}

// AtomicProfileMutator is an optional Repository extension performing atomic read-modify-write + outbox inside one transaction.
type AtomicProfileMutator interface {
	MutateProfileAtomic(ctx context.Context, retailerID, supplierID string, mutate func(existing *Profile) (Profile, error), emit func(outbox.TxnBuffer, Profile) error) (Profile, error)
}

// OrderBalanceClearer is an optional Repository extension supporting partial order reservation balance clearing.
type OrderBalanceClearer interface {
	ClearOrderBalance(ctx context.Context, retailerID, supplierID, orderID string, amountMinor int64, emit func(outbox.TxnBuffer, int64) error) error
}

var errProfileNoop = errors.New("credit: noop mutation")

// Repository persists retailer credit profiles and order reservations.
type Repository interface {
	GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error)
	ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Profile, error)
	UpsertProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error
	AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error
	GetScoresForRetailers(ctx context.Context, retailerIDs []string) (map[string]RetailerCreditScore, error)
	ReserveOrder(ctx context.Context, res OrderReservation, emit func(outbox.TxnBuffer) error) error
	ReleaseOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error
	ConvertOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error
	GetOrderReservation(ctx context.Context, orderID string) (OrderReservation, bool, error)
	ListDelinquentCandidates(ctx context.Context, limit int) ([]Profile, error)
}

// SpannerRepository is a Spanner-backed credit profile repository.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed credit profile repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

const profileSelectColumns = `RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, ReservedMinor, AvailableCreditMinor, RiskScore, DelinquencyCount, PrepayOnlyLocked, Status, LastEvaluatedAt, Version, CreatedAt, UpdatedAt`

// GetProfile reads one credit profile by retailer + supplier.
func (r *SpannerRepository) GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	if r == nil || r.client == nil {
		return Profile{}, false, fmt.Errorf("spanner credit repository: nil client")
	}
	stmt := spanner.Statement{
		SQL:    "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE RetailerId = @rid AND SupplierId = @sid",
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return Profile{}, false, nil
		}
		// Fallback if ReservedMinor or PrepayOnlyLocked column not yet migrated.
		if strings.Contains(err.Error(), "ReservedMinor") || strings.Contains(err.Error(), "PrepayOnlyLocked") {
			return r.getProfileLegacy(ctx, retailerID, supplierID)
		}
		return Profile{}, false, fmt.Errorf("get credit profile %s/%s: %w", retailerID, supplierID, err)
	}
	p, err := scanProfileRow(row)
	if err != nil {
		return Profile{}, false, err
	}
	return p, true, nil
}

func (r *SpannerRepository) getProfileLegacy(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, AvailableCreditMinor, RiskScore, DelinquencyCount, Status, LastEvaluatedAt, Version, CreatedAt, UpdatedAt FROM RetailerCreditProfiles WHERE RetailerId = @rid AND SupplierId = @sid",
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return Profile{}, false, nil
		}
		return Profile{}, false, err
	}
	var p Profile
	var status spanner.NullString
	var lastEvaluated spanner.NullTime
	if err := row.Columns(&p.RetailerID, &p.SupplierID, &p.CreditLimitMinor, &p.CurrentBalanceMinor, &p.AvailableCreditMinor,
		&p.RiskScore, &p.DelinquencyCount, &status, &lastEvaluated, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Profile{}, false, err
	}
	p.Status = Status(status.StringVal)
	p.AvailableCreditMinor = p.Available()
	if lastEvaluated.Valid {
		p.LastEvaluatedAt = lastEvaluated.Time
	}
	return p, true, nil
}

func encodeProfileCursor(updatedAt time.Time, retailerID, supplierID string) string {
	payload := updatedAt.UTC().Format(time.RFC3339Nano) + "|" + retailerID + "|" + supplierID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeProfileCursor(cursor string) (time.Time, string, string, bool, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return time.Time{}, "", "", false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", "", false, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return time.Time{}, "", "", false, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", "", false, ErrInvalidCursor
	}
	return ts.UTC(), parts[1], parts[2], true, nil
}

// ListBySupplier returns credit profiles for one supplier, newest first.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Profile, error) {
	page, _, err := r.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

// ListBySupplierPage returns keyset-paginated credit profiles for one supplier, newest first.
func (r *SpannerRepository) ListBySupplierPage(ctx context.Context, supplierID, status string, limit int, cursor string) ([]Profile, string, error) {
	if r == nil || r.client == nil {
		return nil, "", fmt.Errorf("spanner credit repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, "", fmt.Errorf("supplier_id_required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	cursorTs, cursorRid, cursorSid, hasCursor, err := decodeProfileCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	sql := "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE SupplierId = @sid"
	params := map[string]any{
		"sid":   supplierID,
		"limit": int64(limit + 1),
	}
	if status != "" {
		sql += " AND Status = @status"
		params["status"] = status
	}
	if hasCursor {
		sql += " AND (UpdatedAt < @cursorTs OR (UpdatedAt = @cursorTs AND (RetailerId < @cursorRid OR (RetailerId = @cursorRid AND SupplierId < @cursorSid))))"
		params["cursorTs"] = cursorTs
		params["cursorRid"] = cursorRid
		params["cursorSid"] = cursorSid
	}
	sql += " ORDER BY UpdatedAt DESC, RetailerId DESC, SupplierId DESC LIMIT @limit"

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Profile, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("list credit profiles: %w", err)
		}
		p, err := scanProfileRow(row)
		if err != nil {
			return nil, "", err
		}
		out = append(out, p)
	}
	if len(out) > limit {
		page := out[:limit]
		last := page[len(page)-1]
		return page, encodeProfileCursor(last.UpdatedAt, last.RetailerID, last.SupplierID), nil
	}
	return out, "", nil
}

// ListDelinquentCandidates returns active credit profiles that have open balance or delinquency count > 0.
func (r *SpannerRepository) ListDelinquentCandidates(ctx context.Context, limit int) ([]Profile, error) {
	page, _, err := r.ListDelinquentCandidatesPage(ctx, limit, "")
	return page, err
}

// ListDelinquentCandidatesPage returns keyset-paginated active delinquent credit candidates.
func (r *SpannerRepository) ListDelinquentCandidatesPage(ctx context.Context, limit int, cursor string) ([]Profile, string, error) {
	if r == nil || r.client == nil {
		return nil, "", fmt.Errorf("spanner credit repository: nil client")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	cursorTs, cursorRid, cursorSid, hasCursor, err := decodeProfileCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	sql := "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE Status = 'ACTIVE' AND (DelinquencyCount > 0 OR CurrentBalanceMinor > 0)"
	params := map[string]any{"limit": int64(limit + 1)}
	if hasCursor {
		sql += " AND (UpdatedAt < @cursorTs OR (UpdatedAt = @cursorTs AND (RetailerId < @cursorRid OR (RetailerId = @cursorRid AND SupplierId < @cursorSid))))"
		params["cursorTs"] = cursorTs
		params["cursorRid"] = cursorRid
		params["cursorSid"] = cursorSid
	}
	sql += " ORDER BY UpdatedAt DESC, RetailerId DESC, SupplierId DESC LIMIT @limit"

	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()
	out := make([]Profile, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("list delinquent candidates: %w", err)
		}
		p, err := scanProfileRow(row)
		if err != nil {
			return nil, "", err
		}
		out = append(out, p)
	}
	if len(out) > limit {
		page := out[:limit]
		last := page[len(page)-1]
		return page, encodeProfileCursor(last.UpdatedAt, last.RetailerID, last.SupplierID), nil
	}
	return out, "", nil
}

// MutateProfileAtomic executes a single-transaction read-modify-write + outbox emission.
func (r *SpannerRepository) MutateProfileAtomic(
	ctx context.Context,
	retailerID, supplierID string,
	mutate func(existing *Profile) (Profile, error),
	emit func(outbox.TxnBuffer, Profile) error,
) (Profile, error) {
	if r == nil || r.client == nil {
		return Profile{}, fmt.Errorf("spanner credit repository: nil client")
	}
	var committed Profile
	cols := []string{
		"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor",
		"ReservedMinor", "AvailableCreditMinor", "RiskScore", "DelinquencyCount",
		"PrepayOnlyLocked", "Status", "LastEvaluatedAt", "Version", "CreatedAt", "UpdatedAt",
	}
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var existingPtr *Profile
		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, cols)
		if err == nil {
			p, scanErr := scanProfileRow(row)
			if scanErr != nil {
				return scanErr
			}
			existingPtr = &p
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		next, err := mutate(existingPtr)
		if err != nil {
			if errors.Is(err, errProfileNoop) && existingPtr != nil {
				committed = *existingPtr
			}
			return err
		}
		if !next.Status.Valid() {
			return fmt.Errorf("invalid credit profile status: %s", next.Status)
		}
		if existingPtr != nil {
			next.Version = existingPtr.Version + 1
			if next.CreatedAt.IsZero() {
				next.CreatedAt = existingPtr.CreatedAt
			}
		} else {
			next.Version = 1
			if next.CreatedAt.IsZero() {
				next.CreatedAt = next.UpdatedAt
			}
		}
		next.AvailableCreditMinor = next.Available()

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf, next); err != nil {
				return err
			}
		}
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           next.RetailerID,
				"SupplierId":           next.SupplierID,
				"CreditLimitMinor":     next.CreditLimitMinor,
				"CurrentBalanceMinor":  next.CurrentBalanceMinor,
				"ReservedMinor":        next.ReservedMinor,
				"AvailableCreditMinor": next.AvailableCreditMinor,
				"RiskScore":            next.RiskScore,
				"DelinquencyCount":     next.DelinquencyCount,
				"PrepayOnlyLocked":     next.PrepayOnlyLocked,
				"Status":               string(next.Status),
				"LastEvaluatedAt":      next.LastEvaluatedAt,
				"Version":              next.Version,
				"CreatedAt":            next.CreatedAt,
				"UpdatedAt":            next.UpdatedAt,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		committed = next
		return txn.BufferWrite(mutations)
	})
	if errors.Is(err, errProfileNoop) {
		return committed, nil
	}
	if err != nil {
		return Profile{}, err
	}
	return committed, nil
}

// UpsertProfile creates or updates a credit profile atomically with an optional outbox event.
func (r *SpannerRepository) UpsertProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	p.AvailableCreditMinor = p.Available()
	if p.Status == "" {
		p.Status = StatusInactive
	}
	if !p.Status.Valid() {
		return fmt.Errorf("invalid credit profile status: %s", p.Status)
	}
	// The closure may be re-invoked by the Spanner client after an abort, so the
	// caller's expected version must be captured before the closure mutates p.
	callerVersion := p.Version
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		expectedVersion, reserved, balance, found, err := readProfileVersionReservedBalance(ctx, txn, p.RetailerID, p.SupplierID)
		if err != nil {
			return err
		}
		if found {
			if callerVersion != 0 && expectedVersion != callerVersion {
				return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", expectedVersion, callerVersion)
			}
			p.Version = expectedVersion + 1
			if p.ReservedMinor == 0 {
				p.ReservedMinor = reserved
			}
			if p.CurrentBalanceMinor == 0 && balance > 0 && p.CreditLimitMinor > 0 {
				p.CurrentBalanceMinor = balance
			}
		} else {
			p.Version = 1
			p.CreatedAt = p.UpdatedAt
		}
		p.AvailableCreditMinor = p.Available()

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           p.RetailerID,
				"SupplierId":           p.SupplierID,
				"CreditLimitMinor":     p.CreditLimitMinor,
				"CurrentBalanceMinor":  p.CurrentBalanceMinor,
				"ReservedMinor":        p.ReservedMinor,
				"AvailableCreditMinor": p.AvailableCreditMinor,
				"RiskScore":            p.RiskScore,
				"DelinquencyCount":     p.DelinquencyCount,
				"PrepayOnlyLocked":     p.PrepayOnlyLocked,
				"Status":               string(p.Status),
				"LastEvaluatedAt":      p.LastEvaluatedAt,
				"Version":              p.Version,
				"CreatedAt":            p.CreatedAt,
				"UpdatedAt":            p.UpdatedAt,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
}

// readProfileVersionReservedBalance prefers the modern column set and falls back
// when ReservedMinor has not been migrated yet (avoids aborting the RW txn).
func readProfileVersionReservedBalance(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID string) (version, reserved, balance int64, found bool, err error) {
	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"Version", "ReservedMinor", "CurrentBalanceMinor"})
	if err == nil {
		if colErr := row.Columns(&version, &reserved, &balance); colErr != nil {
			return 0, 0, 0, false, colErr
		}
		return version, reserved, balance, true, nil
	}
	if spanner.ErrCode(err) == codes.NotFound {
		return 0, 0, 0, false, nil
	}
	if !strings.Contains(err.Error(), "ReservedMinor") {
		return 0, 0, 0, false, err
	}
	row, err = txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"Version", "CurrentBalanceMinor"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return 0, 0, 0, false, nil
		}
		return 0, 0, 0, false, err
	}
	if colErr := row.Columns(&version, &balance); colErr != nil {
		return 0, 0, 0, false, colErr
	}
	return version, 0, balance, true, nil
}

// AdjustBalance atomically adds deltaMinor to the current balance and recomputes available credit.
func (r *SpannerRepository) AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrProfileNotFound
			}
			return err
		}
		var limit, balance, reserved, version int64
		if err := row.Columns(&limit, &balance, &reserved, &version); err != nil {
			return err
		}
		newBalance := balance + deltaMinor
		if newBalance < 0 {
			newBalance = 0
		}
		available := limit - newBalance - reserved
		if available < 0 {
			available = 0
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           retailerID,
				"SupplierId":           supplierID,
				"CurrentBalanceMinor":  newBalance,
				"AvailableCreditMinor": available,
				"Version":              version + 1,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
}

// ReserveOrder creates a RESERVED row and bumps profile.ReservedMinor (CAS idempotent).
func (r *SpannerRepository) ReserveOrder(ctx context.Context, res OrderReservation, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existing, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{res.OrderID}, []string{"Status", "AmountMinor"})
		if err == nil {
			var st string
			var amt int64
			if colErr := existing.Columns(&st, &amt); colErr != nil {
				return colErr
			}
			if st == string(ReservationReserved) || st == string(ReservationConverted) {
				return nil // idempotent
			}
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{res.RetailerID, res.SupplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status", "PrepayOnlyLocked", "Version"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrProfileNotFound
			}
			return err
		}
		var limit, balance, reserved, version int64
		var status string
		var prepayLocked spanner.NullBool
		if err := row.Columns(&limit, &balance, &reserved, &status, &prepayLocked, &version); err != nil {
			return err
		}
		if Status(status) != StatusActive {
			if Status(status) == StatusFrozen {
				return ErrProfileFrozen
			}
			return fmt.Errorf("%w: %s", ErrCreditNotEnabled, status)
		}
		if prepayLocked.Valid && prepayLocked.Bool {
			return fmt.Errorf("%w: prepay_only_locked", ErrCreditNotEnabled)
		}
		avail := limit - balance - reserved
		if avail < res.AmountMinor {
			return ErrLimitBreached
		}
		newReserved := reserved + res.AmountMinor
		available := limit - balance - newReserved
		if available < 0 {
			available = 0
		}
		now := time.Now().UTC()
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("OrderCreditReservations", map[string]any{
				"OrderId":     res.OrderID,
				"RetailerId":  res.RetailerID,
				"SupplierId":  res.SupplierID,
				"AmountMinor": res.AmountMinor,
				"Status":      string(ReservationReserved),
				"CreatedAt":   now,
				"UpdatedAt":   now,
			}),
			spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           res.RetailerID,
				"SupplierId":           res.SupplierID,
				"ReservedMinor":        newReserved,
				"AvailableCreditMinor": available,
				"Version":              version + 1,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
}

// ReleaseOrderReservation frees reserved headroom.
func (r *SpannerRepository) ReleaseOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
			[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return nil
			}
			return err
		}
		var retailerID, supplierID, status string
		var amount int64
		if err := row.Columns(&retailerID, &supplierID, &amount, &status); err != nil {
			return err
		}
		if status != string(ReservationReserved) {
			return nil
		}
		prof, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
		if err != nil {
			return err
		}
		var limit, balance, reserved, version int64
		if err := prof.Columns(&limit, &balance, &reserved, &version); err != nil {
			return err
		}
		newReserved := reserved - amount
		if newReserved < 0 {
			newReserved = 0
		}
		available := limit - balance - newReserved
		if available < 0 {
			available = 0
		}
		buf := &spannerTxnBuffer{}
		if emit != nil {
			_ = emit(buf)
		}
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("OrderCreditReservations", map[string]any{
				"OrderId":   orderID,
				"Status":    string(ReservationReleased),
				"UpdatedAt": spanner.CommitTimestamp,
			}),
			spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           retailerID,
				"SupplierId":           supplierID,
				"ReservedMinor":        newReserved,
				"AvailableCreditMinor": available,
				"Version":              version + 1,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
}

// ConvertOrderReservation moves reserved → balance.
func (r *SpannerRepository) ConvertOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return r.convertReservationInTxn(ctx, txn, "", "", orderID, 0, emit)
	})
}

type creditAdjustmentPlan struct {
	NewBalanceMinor       int64
	NewReservedMinor      int64
	AvailableCreditMinor  int64
	NextReservationStatus ReservationStatus
	AdjustInvoice         bool
	Reason                string
}

func computeCreditAdjustment(status ReservationStatus, limit, balance, reserved, currentAmount, newAmountMinor int64, orderID string) (creditAdjustmentPlan, bool, error) {
	if newAmountMinor < 0 {
		return creditAdjustmentPlan{}, false, fmt.Errorf("credit: negative reserve amount %d", newAmountMinor)
	}
	if currentAmount == newAmountMinor {
		return creditAdjustmentPlan{}, false, nil
	}
	delta := currentAmount - newAmountMinor

	switch status {
	case ReservationReserved:
		newReserved := reserved - delta
		if newReserved < 0 {
			newReserved = 0
		}
		available := limit - balance - newReserved
		if available < 0 {
			available = 0
		}
		nextStatus := ReservationReserved
		if newAmountMinor == 0 {
			nextStatus = ReservationReleased
		}
		return creditAdjustmentPlan{
			NewBalanceMinor:       balance,
			NewReservedMinor:      newReserved,
			AvailableCreditMinor:  available,
			NextReservationStatus: nextStatus,
			AdjustInvoice:         false,
			Reason:                fmt.Sprintf("credit_reserve_adjust:%s", orderID),
		}, true, nil

	case ReservationConverted:
		newBalance := balance - delta
		if newBalance < 0 {
			newBalance = 0
		}
		available := limit - newBalance - reserved
		if available < 0 {
			available = 0
		}
		nextStatus := ReservationConverted
		if newAmountMinor == 0 {
			nextStatus = ReservationCleared
		}
		return creditAdjustmentPlan{
			NewBalanceMinor:       newBalance,
			NewReservedMinor:      reserved,
			AvailableCreditMinor:  available,
			NextReservationStatus: nextStatus,
			AdjustInvoice:         true,
			Reason:                fmt.Sprintf("credit_balance_adjust:%s", orderID),
		}, true, nil

	default:
		return creditAdjustmentPlan{}, false, nil
	}
}

// AdjustReserveInTxn updates an order's credit hold (RESERVED) or converted balance/AR (CONVERTED)
// inside txn and emits outbox adjustment events so headroom and AR never drift.
func (r *SpannerRepository) AdjustReserveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, newAmountMinor int64) error {
	if txn == nil {
		return fmt.Errorf("nil spanner txn")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("order_id required")
	}
	if newAmountMinor < 0 {
		return fmt.Errorf("credit: negative reserve amount %d", newAmountMinor)
	}
	return r.AdjustReservesBatchInTxn(ctx, txn, map[string]int64{orderID: newAmountMinor})
}

// AdjustReservesBatchInTxn adjusts multiple orders atomically in a single Spanner txn,
// consolidating updates per (RetailerId, SupplierId) profile so multiple orders for the same
// retailer on one wave never collide or overwrite each other in txn.BufferWrite.
func (r *SpannerRepository) AdjustReservesBatchInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderNewAmounts map[string]int64) error {
	if txn == nil {
		return fmt.Errorf("nil spanner txn")
	}
	if len(orderNewAmounts) == 0 {
		return nil
	}

	orderIDs := make([]string, 0, len(orderNewAmounts))
	for oid, amt := range orderNewAmounts {
		oid = strings.TrimSpace(oid)
		if oid == "" {
			return fmt.Errorf("order_id required")
		}
		if amt < 0 {
			return fmt.Errorf("credit: negative reserve amount %d for order %s", amt, oid)
		}
		orderIDs = append(orderIDs, oid)
	}
	sort.Strings(orderIDs)

	type profileKey struct {
		retailerID string
		supplierID string
	}
	type profileState struct {
		limit    int64
		balance  int64
		reserved int64
		version  int64
		dirty    bool
	}
	type orderResRecord struct {
		orderID       string
		retailerID    string
		supplierID    string
		currentAmount int64
		newAmount     int64
		status        ReservationStatus
	}

	var activeOrders []orderResRecord
	neededProfiles := make(map[profileKey]struct{})
	for _, oid := range orderIDs {
		newAmountMinor := orderNewAmounts[oid]
		row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{oid},
			[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				continue
			}
			return err
		}
		var retailerID, supplierID, status string
		var currentAmount int64
		if err := row.Columns(&retailerID, &supplierID, &currentAmount, &status); err != nil {
			return err
		}
		resStatus := ReservationStatus(strings.ToUpper(strings.TrimSpace(status)))
		if (resStatus != ReservationReserved && resStatus != ReservationConverted) || currentAmount == newAmountMinor {
			continue
		}
		pk := profileKey{retailerID: retailerID, supplierID: supplierID}
		neededProfiles[pk] = struct{}{}
		activeOrders = append(activeOrders, orderResRecord{
			orderID:       oid,
			retailerID:    retailerID,
			supplierID:    supplierID,
			currentAmount: currentAmount,
			newAmount:     newAmountMinor,
			status:        resStatus,
		})
	}
	if len(activeOrders) == 0 {
		return nil
	}

	pKeys := make([]profileKey, 0, len(neededProfiles))
	for pk := range neededProfiles {
		pKeys = append(pKeys, pk)
	}
	sort.Slice(pKeys, func(i, j int) bool {
		if pKeys[i].retailerID != pKeys[j].retailerID {
			return pKeys[i].retailerID < pKeys[j].retailerID
		}
		return pKeys[i].supplierID < pKeys[j].supplierID
	})

	profiles := make(map[profileKey]*profileState, len(pKeys))
	for _, pk := range pKeys {
		prof, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{pk.retailerID, pk.supplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
		if err != nil {
			return err
		}
		var limit, balance, reserved, version int64
		if err := prof.Columns(&limit, &balance, &reserved, &version); err != nil {
			return err
		}
		profiles[pk] = &profileState{
			limit:    limit,
			balance:  balance,
			reserved: reserved,
			version:  version,
		}
	}

	buf := &spannerTxnBuffer{}
	var mutations []*spanner.Mutation
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)

	for _, rec := range activeOrders {
		pk := profileKey{retailerID: rec.retailerID, supplierID: rec.supplierID}
		ps := profiles[pk]
		plan, changed, err := computeCreditAdjustment(rec.status, ps.limit, ps.balance, ps.reserved, rec.currentAmount, rec.newAmount, rec.orderID)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}

		if plan.AdjustInvoice {
			idemKey := fmt.Sprintf("credit_adjust:%s:%d->%d", rec.orderID, rec.currentAmount, rec.newAmount)
			if _, err := ar.AdjustInvoiceBalanceForOrderInTxn(ctx, txn, rec.orderID, rec.currentAmount, rec.newAmount, idemKey); err != nil {
				return fmt.Errorf("adjust ar invoice: %w", err)
			}
		}

		ps.balance = plan.NewBalanceMinor
		ps.reserved = plan.NewReservedMinor
		ps.dirty = true

		if err := outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, rec.retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: nowStr},
			ProfileID:        profileID(rec.retailerID, rec.supplierID),
			RetailerID:       rec.retailerID,
			SupplierID:       rec.supplierID,
			CreditLimitMinor: ps.limit,
			CurrentBalance:   ps.balance,
			Reason:           plan.Reason,
		}); err != nil {
			return err
		}

		mutations = append(mutations, spanner.UpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     rec.orderID,
			"AmountMinor": rec.newAmount,
			"Status":      string(plan.NextReservationStatus),
			"UpdatedAt":   spanner.CommitTimestamp,
		}))
	}

	for _, pk := range pKeys {
		ps := profiles[pk]
		if !ps.dirty {
			continue
		}
		available := ps.limit - ps.balance - ps.reserved
		if available < 0 {
			available = 0
		}
		mutations = append(mutations, spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           pk.retailerID,
			"SupplierId":           pk.supplierID,
			"CurrentBalanceMinor":  ps.balance,
			"ReservedMinor":        ps.reserved,
			"AvailableCreditMinor": available,
			"Version":              ps.version + 1,
			"UpdatedAt":            spanner.CommitTimestamp,
		}))
	}

	mutations = append(mutations, bufferOutboxMutations(buf)...)
	if len(mutations) == 0 {
		return nil
	}
	return txn.BufferWrite(mutations)
}

// MarkBalanceInTxn converts reservation or marks balance in the same txn as credit leave.
func (r *SpannerRepository) MarkBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	return r.convertReservationInTxn(ctx, txn, retailerID, supplierID, orderID, amountMinor, nil)
}

type partialClearPlan struct {
	ClearAmountMinor     int64
	RemainingAmountMinor int64
	NextStatus           ReservationStatus
	NewBalanceMinor      int64
	AvailableCreditMinor int64
}

func computePartialClearPlan(status ReservationStatus, limit, balance, reserved, resAmount, amountMinor int64) (partialClearPlan, bool) {
	if status != ReservationConverted || amountMinor <= 0 {
		return partialClearPlan{}, false
	}
	clearAmt := amountMinor
	if resAmount > 0 && clearAmt > resAmount {
		clearAmt = resAmount
	}
	if clearAmt <= 0 {
		return partialClearPlan{}, false
	}
	remaining := resAmount - clearAmt
	if remaining < 0 {
		remaining = 0
	}
	nextStatus := ReservationConverted
	if remaining <= 0 {
		nextStatus = ReservationCleared
	}
	newBalance := balance - clearAmt
	if newBalance < 0 {
		newBalance = 0
	}
	available := limit - newBalance - reserved
	if available < 0 {
		available = 0
	}
	return partialClearPlan{
		ClearAmountMinor:     clearAmt,
		RemainingAmountMinor: remaining,
		NextStatus:           nextStatus,
		NewBalanceMinor:      newBalance,
		AvailableCreditMinor: available,
	}, true
}

// ClearOrderBalance implements OrderBalanceClearer inside a single Spanner RW txn.
func (r *SpannerRepository) ClearOrderBalance(ctx context.Context, retailerID, supplierID, orderID string, amountMinor int64, _ func(outbox.TxnBuffer, int64) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return r.ClearBalanceInTxn(ctx, txn, retailerID, supplierID, orderID, amountMinor)
	})
	return err
}

// ClearBalanceInTxn decreases profile balance when a credit-left order is paid
// (G1-A2). Only acts when OrderCreditReservations is CONVERTED for orderID;
// already CLEARED -> no-op. Supports partial clearing: keeps Status CONVERTED
// until remaining reservation amount hits 0. Fail-closed on profile/write errors.
func (r *SpannerRepository) ClearBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	if txn == nil {
		return fmt.Errorf("nil spanner txn")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" || amountMinor <= 0 {
		return nil
	}
	// Prefer reservation as source of truth for credit-leave mark + amount.
	row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
	if err != nil {
		if spanner.ErrCode(err) != codes.NotFound {
			return err
		}
		// Fallback: check Orders for MIGRATION order, or ArInvoices for migrated invoice.
		var isMigrated bool
		ordRow, oErr := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"RetailerId", "SupplierId", "OrderSource", "TotalMinor"})
		if oErr == nil {
			var ordRet, ordSup, ordSource string
			var ordTotal int64
			if colErr := ordRow.Columns(&ordRet, &ordSup, &ordSource, &ordTotal); colErr == nil {
				if ordSource == "MIGRATION" {
					isMigrated = true
					if ordRet != "" {
						retailerID = ordRet
					}
					if ordSup != "" {
						supplierID = ordSup
					}
				}
			}
		} else if spanner.ErrCode(oErr) != codes.NotFound {
			return oErr
		}

		if !isMigrated {
			// Fallback check ArInvoices for migrated invoice by OrderId
			invIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT RetailerId, SupplierId, Source
				      FROM ArInvoices
				      WHERE OrderId = @oid LIMIT 1`,
				Params: map[string]any{"oid": orderID},
			})
			invRow, iErr := invIter.Next()
			if iErr == nil {
				var invRet, invSup, invSource string
				if colErr := invRow.Columns(&invRet, &invSup, &invSource); colErr == nil {
					if invSource == "MIGRATION" {
						isMigrated = true
						if invRet != "" {
							retailerID = invRet
						}
						if invSup != "" {
							supplierID = invSup
						}
					}
				}
			} else if iErr != iterator.Done {
				invIter.Stop()
				return iErr
			}
			invIter.Stop()
		}

		if !isMigrated {
			// Never credit-left this order — no balance to clear.
			return nil
		}

		if retailerID == "" || supplierID == "" {
			return nil
		}

		prof, pErr := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
		if pErr != nil {
			if spanner.ErrCode(pErr) == codes.NotFound {
				return ErrProfileNotFound
			}
			return pErr
		}
		var limit, balance, reserved, version int64
		if colErr := prof.Columns(&limit, &balance, &reserved, &version); colErr != nil {
			return colErr
		}

		clearAmt := amountMinor
		if clearAmt > balance {
			clearAmt = balance
		}
		if clearAmt <= 0 {
			return nil
		}
		newBalance := balance - clearAmt
		if newBalance < 0 {
			newBalance = 0
		}
		avail := limit - newBalance - reserved
		if avail < 0 {
			avail = 0
		}

		buf := &spannerTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			ProfileID:      profileID(retailerID, supplierID),
			RetailerID:     retailerID,
			SupplierID:     supplierID,
			CurrentBalance: -clearAmt,
			Reason:         fmt.Sprintf("credit_payment:%s", orderID),
		}); emitErr != nil {
			return emitErr
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           retailerID,
				"SupplierId":           supplierID,
				"CurrentBalanceMinor":  newBalance,
				"AvailableCreditMinor": avail,
				"Version":              version + 1,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
			spanner.InsertOrUpdateMap("OrderCreditReservations", map[string]any{
				"OrderId":     orderID,
				"RetailerId":  retailerID,
				"SupplierId":  supplierID,
				"AmountMinor": int64(0),
				"Status":      string(ReservationCleared),
				"CreatedAt":   spanner.CommitTimestamp,
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		}
		mutations = append(mutations, bufferOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	}
	var rid, sid, status string
	var resAmount int64
	if err := row.Columns(&rid, &sid, &resAmount, &status); err != nil {
		return err
	}
	if status == string(ReservationCleared) {
		return nil // idempotent
	}
	if status != string(ReservationConverted) {
		// RESERVED/RELEASED: not on balance yet — nothing to clear.
		return nil
	}
	if rid != "" {
		retailerID = rid
	}
	if sid != "" {
		supplierID = sid
	}
	if retailerID == "" || supplierID == "" {
		return nil
	}

	prof, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ErrProfileNotFound
		}
		return err
	}
	var limit, balance, reserved, version int64
	if err := prof.Columns(&limit, &balance, &reserved, &version); err != nil {
		return err
	}
	plan, ok := computePartialClearPlan(ReservationStatus(status), limit, balance, reserved, resAmount, amountMinor)
	if !ok {
		return nil
	}

	buf := &spannerTxnBuffer{}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
		BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
		ProfileID:      profileID(retailerID, supplierID),
		RetailerID:     retailerID,
		SupplierID:     supplierID,
		CurrentBalance: -plan.ClearAmountMinor,
		Reason:         fmt.Sprintf("credit_payment:%s", orderID),
	}); err != nil {
		return err
	}

	mutations := []*spanner.Mutation{
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CurrentBalanceMinor":  plan.NewBalanceMinor,
			"AvailableCreditMinor": plan.AvailableCreditMinor,
			"Version":              version + 1,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
		spanner.UpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     orderID,
			"AmountMinor": plan.RemainingAmountMinor,
			"Status":      string(plan.NextStatus),
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
	}
	mutations = append(mutations, bufferOutboxMutations(buf)...)
	return txn.BufferWrite(mutations)
}

func (r *SpannerRepository) convertReservationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, fallbackAmount int64, emit func(outbox.TxnBuffer) error) error {
	row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
	var status string
	var amount int64
	if err == nil {
		var rid, sid string
		if err := row.Columns(&rid, &sid, &amount, &status); err != nil {
			return err
		}
		if rid != "" {
			retailerID = rid
		}
		if sid != "" {
			supplierID = sid
		}
		if status == string(ReservationConverted) {
			return nil // idempotent
		}
		if status == string(ReservationReserved) {
			return r.applyBalanceDeltaInTxn(ctx, txn, retailerID, supplierID, orderID, amount, true, emit)
		}
		// RELEASED or other — fall through to direct mark if needed
		amount = fallbackAmount
	} else if spanner.ErrCode(err) != codes.NotFound {
		return err
	} else {
		amount = fallbackAmount
	}
	if amount <= 0 {
		return nil
	}
	return r.applyBalanceDeltaInTxn(ctx, txn, retailerID, supplierID, orderID, amount, false, emit)
}

func (r *SpannerRepository) applyBalanceDeltaInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amount int64, convertRes bool, emit func(outbox.TxnBuffer) error) error {
	if retailerID == "" || supplierID == "" {
		// Recover keys from reservation if convertRes path already filled them.
		if orderID != "" {
			row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
				[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
			if err == nil {
				var st string
				var amt int64
				_ = row.Columns(&retailerID, &supplierID, &amt, &st)
				if amount <= 0 {
					amount = amt
				}
				convertRes = st == string(ReservationReserved)
			}
		}
	}
	if retailerID == "" || supplierID == "" || amount <= 0 {
		return nil
	}
	prof, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ErrProfileNotFound
		}
		return err
	}
	var limit, balance, reserved, version int64
	if err := prof.Columns(&limit, &balance, &reserved, &version); err != nil {
		return err
	}
	newBalance := balance + amount
	newReserved := reserved
	if convertRes {
		newReserved = reserved - amount
		if newReserved < 0 {
			newReserved = 0
		}
	}
	available := limit - newBalance - newReserved
	if available < 0 {
		available = 0
	}
	buf := &spannerTxnBuffer{}
	if emit != nil {
		_ = emit(buf)
	}
	mutations := []*spanner.Mutation{
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CurrentBalanceMinor":  newBalance,
			"ReservedMinor":        newReserved,
			"AvailableCreditMinor": available,
			"Version":              version + 1,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
	}
	if convertRes {
		mutations = append(mutations, spanner.UpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":   orderID,
			"Status":    string(ReservationConverted),
			"UpdatedAt": spanner.CommitTimestamp,
		}))
	} else if orderID != "" {
		// Idempotency marker: synthetic converted reservation so re-Mark is no-op.
		now := time.Now().UTC()
		mutations = append(mutations, spanner.InsertOrUpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     orderID,
			"RetailerId":  retailerID,
			"SupplierId":  supplierID,
			"AmountMinor": amount,
			"Status":      string(ReservationConverted),
			"CreatedAt":   now,
			"UpdatedAt":   now,
		}))
	}
	mutations = append(mutations, bufferOutboxMutations(buf)...)
	return txn.BufferWrite(mutations)
}

// GetOrderReservation loads a reservation by order id.
func (r *SpannerRepository) GetOrderReservation(ctx context.Context, orderID string) (OrderReservation, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"OrderId", "RetailerId", "SupplierId", "AmountMinor", "Status", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return OrderReservation{}, false, nil
		}
		return OrderReservation{}, false, err
	}
	var res OrderReservation
	var st string
	if err := row.Columns(&res.OrderID, &res.RetailerID, &res.SupplierID, &res.AmountMinor, &st, &res.CreatedAt, &res.UpdatedAt); err != nil {
		return OrderReservation{}, false, err
	}
	res.Status = ReservationStatus(st)
	return res, true, nil
}

func scanProfileRow(row *spanner.Row) (Profile, error) {
	var p Profile
	var prepayLocked spanner.NullBool
	var status spanner.NullString
	var lastEvaluated spanner.NullTime
	if err := row.Columns(&p.RetailerID, &p.SupplierID, &p.CreditLimitMinor, &p.CurrentBalanceMinor, &p.ReservedMinor, &p.AvailableCreditMinor,
		&p.RiskScore, &p.DelinquencyCount, &prepayLocked, &status, &lastEvaluated, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Profile{}, fmt.Errorf("scan credit profile row: %w", err)
	}
	if prepayLocked.Valid {
		p.PrepayOnlyLocked = prepayLocked.Bool
	}
	p.Status = Status(status.StringVal)
	p.AvailableCreditMinor = p.Available()
	if lastEvaluated.Valid {
		p.LastEvaluatedAt = lastEvaluated.Time
	}
	return p, nil
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func bufferOutboxMutations(buf *spannerTxnBuffer) []*spanner.Mutation {
	if buf == nil {
		return nil
	}
	out := make([]*spanner.Mutation, 0, len(buf.events))
	for _, e := range buf.events {
		out = append(out, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(e)))
	}
	return out
}

// GetScoresForRetailers returns empty without supplier scope — use Service.GetScoresForRetailers.
// Kept for Repository interface compatibility; real scoring is service-layer (G3.B).
func (r *SpannerRepository) GetScoresForRetailers(ctx context.Context, retailerIDs []string) (map[string]RetailerCreditScore, error) {
	_ = ctx
	_ = retailerIDs
	return map[string]RetailerCreditScore{}, nil
}

func (r *SpannerRepository) ReserveOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, res OrderReservation) error {
	existing, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{res.OrderID}, []string{"Status", "AmountMinor"})
	if err == nil {
		var st string
		var amt int64
		if colErr := existing.Columns(&st, &amt); colErr != nil {
			return colErr
		}
		if st == string(ReservationReserved) || st == string(ReservationConverted) {
			return nil // idempotent
		}
	} else if spanner.ErrCode(err) != codes.NotFound {
		return err
	}

	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{res.RetailerID, res.SupplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status", "PrepayOnlyLocked", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ErrProfileNotFound
		}
		return err
	}
	var limit, balance, reserved, version int64
	var st string
	var prepayLocked spanner.NullBool
	if colErr := row.Columns(&limit, &balance, &reserved, &st, &prepayLocked, &version); colErr != nil {
		return colErr
	}

	if st != string(StatusActive) || (prepayLocked.Valid && prepayLocked.Bool) {
		return ErrCreditNotEnabled
	}
	if limit-(balance+reserved) < res.AmountMinor {
		return ErrLimitBreached
	}

	muts := []*spanner.Mutation{
		spanner.InsertOrUpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     res.OrderID,
			"RetailerId":  res.RetailerID,
			"SupplierId":  res.SupplierID,
			"AmountMinor": res.AmountMinor,
			"Status":      string(ReservationReserved),
			"CreatedAt":   spanner.CommitTimestamp,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":    res.RetailerID,
			"SupplierId":    res.SupplierID,
			"ReservedMinor": reserved + res.AmountMinor,
			"Version":       version + 1,
			"UpdatedAt":     spanner.CommitTimestamp,
		}),
	}
	return txn.BufferWrite(muts)
}
