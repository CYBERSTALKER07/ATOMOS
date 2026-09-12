package credit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

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
}

// SpannerRepository is a Spanner-backed credit profile repository.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed credit profile repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

const profileSelectColumns = `RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, ReservedMinor, AvailableCreditMinor, RiskScore, DelinquencyCount, Status, LastEvaluatedAt, Version, CreatedAt, UpdatedAt`

// GetProfile reads one credit profile by retailer + supplier.
func (r *SpannerRepository) GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	if r == nil || r.client == nil {
		return Profile{}, false, fmt.Errorf("spanner credit repository: nil client")
	}
	stmt := spanner.Statement{
		SQL:    "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE RetailerId = @rid AND SupplierId = @sid",
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	}
	row, err := r.client.Single().Query(ctx, stmt).Next()
	if err != nil {
		if err == iterator.Done {
			return Profile{}, false, nil
		}
		// Fallback if ReservedMinor column not yet migrated.
		if strings.Contains(err.Error(), "ReservedMinor") {
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
	row, err := r.client.Single().Query(ctx, stmt).Next()
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

// ListBySupplier returns credit profiles for one supplier, newest first.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Profile, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner credit repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id_required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	sql := "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE SupplierId = @sid"
	params := map[string]any{"sid": supplierID}
	if status != "" {
		sql += " AND Status = @status"
		params["status"] = status
	}
	sql += fmt.Sprintf(" ORDER BY UpdatedAt DESC LIMIT %d", limit)

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]Profile, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list credit profiles: %w", err)
		}
		p, err := scanProfileRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
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
				// keep existing balance unless caller set it
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
			_ = existing.Columns(&st, &amt)
			if st == string(ReservationReserved) || st == string(ReservationConverted) {
				return nil // idempotent
			}
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}

		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{res.RetailerID, res.SupplierID},
			[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status", "Version"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrProfileNotFound
			}
			return err
		}
		var limit, balance, reserved, version int64
		var status string
		if err := row.Columns(&limit, &balance, &reserved, &status, &version); err != nil {
			return err
		}
		if Status(status) != StatusActive {
			if Status(status) == StatusFrozen {
				return ErrProfileFrozen
			}
			return fmt.Errorf("%w: %s", ErrCreditNotEnabled, status)
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

// AdjustReserveInTxn updates an order's reservation and adjusts the retailer's ReservedMinor.
func (r *SpannerRepository) AdjustReserveInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, newAmountMinor int64) error {
	row, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{orderID},
		[]string{"RetailerId", "SupplierId", "AmountMinor", "Status"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil
		}
		return err
	}
	var retailerID, supplierID, status string
	var currentAmount int64
	if err := row.Columns(&retailerID, &supplierID, &currentAmount, &status); err != nil {
		return err
	}
	if status != string(ReservationReserved) || currentAmount == newAmountMinor {
		return nil
	}

	delta := currentAmount - newAmountMinor
	prof, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Version"})
	if err != nil {
		return err
	}
	var limit, balance, reserved, version int64
	if err := prof.Columns(&limit, &balance, &reserved, &version); err != nil {
		return err
	}
	newReserved := reserved - delta
	if newReserved < 0 {
		newReserved = 0
	}
	available := limit - balance - newReserved
	if available < 0 {
		available = 0
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.UpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     orderID,
			"AmountMinor": newAmountMinor,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"ReservedMinor":        newReserved,
			"AvailableCreditMinor": available,
			"Version":              version + 1,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
	})
}

// MarkBalanceInTxn converts reservation or marks balance in the same txn as credit leave.
func (r *SpannerRepository) MarkBalanceInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	return r.convertReservationInTxn(ctx, txn, retailerID, supplierID, orderID, amountMinor, nil)
}

// ClearBalanceInTxn decreases profile balance when a credit-left order is paid
// (G1-A2). Only acts when OrderCreditReservations is CONVERTED for orderID;
// already CLEARED → no-op. Fail-closed on profile/write errors.
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
		if spanner.ErrCode(err) == codes.NotFound {
			// Never credit-left this order — no balance to clear.
			return nil
		}
		return err
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
	clearAmt := amountMinor
	if resAmount > 0 {
		clearAmt = resAmount
	}
	if retailerID == "" || supplierID == "" || clearAmt <= 0 {
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
	newBalance := balance - clearAmt
	if newBalance < 0 {
		newBalance = 0
	}
	available := limit - newBalance - reserved
	if available < 0 {
		available = 0
	}

	buf := &spannerTxnBuffer{}
	_ = outbox.EmitJSON(ctx, buf, events.AggregateCreditProfile, retailerID, events.TopicMain, events.CreditProfileEvent{
		BaseEvent:      events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
		ProfileID:      profileID(retailerID, supplierID),
		RetailerID:     retailerID,
		SupplierID:     supplierID,
		CurrentBalance: -clearAmt,
		Reason:         fmt.Sprintf("credit_payment:%s", orderID),
	})

	mutations := []*spanner.Mutation{
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CurrentBalanceMinor":  newBalance,
			"AvailableCreditMinor": available,
			"Version":              version + 1,
			"UpdatedAt":            spanner.CommitTimestamp,
		}),
		spanner.UpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":   orderID,
			"Status":    string(ReservationCleared),
			"UpdatedAt": spanner.CommitTimestamp,
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
	var status spanner.NullString
	var lastEvaluated spanner.NullTime
	if err := row.Columns(&p.RetailerID, &p.SupplierID, &p.CreditLimitMinor, &p.CurrentBalanceMinor, &p.ReservedMinor, &p.AvailableCreditMinor,
		&p.RiskScore, &p.DelinquencyCount, &status, &lastEvaluated, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Profile{}, fmt.Errorf("scan credit profile row: %w", err)
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
		_ = existing.Columns(&st, &amt)
		if st == string(ReservationReserved) || st == string(ReservationConverted) {
			return nil // idempotent
		}
	} else if spanner.ErrCode(err) != 5 {
		return err
	}

	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{res.RetailerID, res.SupplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return ErrProfileNotFound
		}
		return err
	}
	var limit, balance, reserved, version int64
	var st string
	_ = row.Columns(&limit, &balance, &reserved, &st, &version)
	
	if st != string(StatusActive) {
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
