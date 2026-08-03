package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// inMemoryCreditRepo is a test/bootstrap stub for credit profiles.
type inMemoryCreditRepo struct {
	mu           sync.RWMutex
	byKey        map[string]credit.Profile
	reservations map[string]credit.OrderReservation
	outbox       OutboxAppender
}

// NewCreditRepo builds an in-memory credit profile repository.
func NewCreditRepo(outboxAppender OutboxAppender) credit.Repository {
	return &inMemoryCreditRepo{
		byKey:        make(map[string]credit.Profile),
		reservations: make(map[string]credit.OrderReservation),
		outbox:       outboxAppender,
	}
}

func creditKey(retailerID, supplierID string) string {
	return fmt.Sprintf("%s:%s", retailerID, supplierID)
}

func (r *inMemoryCreditRepo) GetProfile(ctx context.Context, retailerID, supplierID string) (credit.Profile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byKey[creditKey(retailerID, supplierID)]
	if ok {
		p.AvailableCreditMinor = p.Available()
	}
	return p, ok, nil
}

func (r *inMemoryCreditRepo) ListBySupplier(_ context.Context, supplierID, status string, limit int) ([]credit.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]credit.Profile, 0, 8)
	for _, p := range r.byKey {
		if p.SupplierID != supplierID {
			continue
		}
		if status != "" && string(p.Status) != status {
			continue
		}
		p.AvailableCreditMinor = p.Available()
		out = append(out, p)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *inMemoryCreditRepo) UpsertProfile(ctx context.Context, p credit.Profile, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outbox != nil {
			if err := r.outbox.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	p.AvailableCreditMinor = p.Available()
	r.byKey[creditKey(p.RetailerID, p.SupplierID)] = p
	return nil
}

func (r *inMemoryCreditRepo) AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := creditKey(retailerID, supplierID)
	p, ok := r.byKey[key]
	if !ok {
		return credit.ErrProfileNotFound
	}
	p.CurrentBalanceMinor += deltaMinor
	if p.CurrentBalanceMinor < 0 {
		p.CurrentBalanceMinor = 0
	}
	p.AvailableCreditMinor = p.Available()
	p.Version++
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outbox != nil {
			if err := r.outbox.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	r.byKey[key] = p
	return nil
}

func (r *inMemoryCreditRepo) GetScoresForRetailers(_ context.Context, retailerIDs []string) (map[string]credit.RetailerCreditScore, error) {
	return map[string]credit.RetailerCreditScore{}, nil
}

func (r *inMemoryCreditRepo) ReserveOrder(ctx context.Context, res credit.OrderReservation, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.reservations[res.OrderID]; ok {
		if existing.Status == credit.ReservationReserved || existing.Status == credit.ReservationConverted {
			return nil
		}
	}
	key := creditKey(res.RetailerID, res.SupplierID)
	p, ok := r.byKey[key]
	if !ok {
		return credit.ErrProfileNotFound
	}
	if p.Status != credit.StatusActive {
		if p.Status == credit.StatusFrozen {
			return credit.ErrProfileFrozen
		}
		return credit.ErrCreditNotEnabled
	}
	if p.Available() < res.AmountMinor {
		return credit.ErrLimitBreached
	}
	p.ReservedMinor += res.AmountMinor
	p.AvailableCreditMinor = p.Available()
	p.Version++
	now := time.Now().UTC()
	res.Status = credit.ReservationReserved
	res.CreatedAt = now
	res.UpdatedAt = now
	r.reservations[res.OrderID] = res
	r.byKey[key] = p
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outbox != nil {
			_ = r.outbox.Append(ctx, txn.events)
		}
	}
	return nil
}

func (r *inMemoryCreditRepo) ReleaseOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	if !ok || res.Status != credit.ReservationReserved {
		return nil
	}
	key := creditKey(res.RetailerID, res.SupplierID)
	p := r.byKey[key]
	p.ReservedMinor -= res.AmountMinor
	if p.ReservedMinor < 0 {
		p.ReservedMinor = 0
	}
	p.AvailableCreditMinor = p.Available()
	p.Version++
	res.Status = credit.ReservationReleased
	res.UpdatedAt = time.Now().UTC()
	r.reservations[orderID] = res
	r.byKey[key] = p
	return nil
}

func (r *inMemoryCreditRepo) ConvertOrderReservation(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	if !ok {
		return nil
	}
	if res.Status == credit.ReservationConverted {
		return nil
	}
	if res.Status != credit.ReservationReserved {
		return nil
	}
	key := creditKey(res.RetailerID, res.SupplierID)
	p := r.byKey[key]
	p.ReservedMinor -= res.AmountMinor
	if p.ReservedMinor < 0 {
		p.ReservedMinor = 0
	}
	p.CurrentBalanceMinor += res.AmountMinor
	p.AvailableCreditMinor = p.Available()
	p.Version++
	res.Status = credit.ReservationConverted
	res.UpdatedAt = time.Now().UTC()
	r.reservations[orderID] = res
	r.byKey[key] = p
	return nil
}

func (r *inMemoryCreditRepo) GetOrderReservation(_ context.Context, orderID string) (credit.OrderReservation, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.reservations[orderID]
	return res, ok, nil
}
