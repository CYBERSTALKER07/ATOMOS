package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// inMemoryCreditRepo is a test/bootstrap stub for credit profiles.
type inMemoryCreditRepo struct {
	mu     sync.RWMutex
	byKey  map[string]credit.Profile
	outbox OutboxAppender
}

// NewCreditRepo builds an in-memory credit profile repository.
func NewCreditRepo(outboxAppender OutboxAppender) credit.Repository {
	return &inMemoryCreditRepo{
		byKey:  make(map[string]credit.Profile),
		outbox: outboxAppender,
	}
}

func creditKey(retailerID, supplierID string) string {
	return fmt.Sprintf("%s:%s", retailerID, supplierID)
}

func (r *inMemoryCreditRepo) GetProfile(ctx context.Context, retailerID, supplierID string) (credit.Profile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byKey[creditKey(retailerID, supplierID)]
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
	p.AvailableCreditMinor = p.CreditLimitMinor - p.CurrentBalanceMinor
	if p.AvailableCreditMinor < 0 {
		p.AvailableCreditMinor = 0
	}
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
