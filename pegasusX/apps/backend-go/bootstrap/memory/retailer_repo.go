package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

// ── Scaffold in-memory retailer repository ─────────────────────────────────
// Replaced wholesale by the Spanner-backed implementation once available.
// Keeps the build green and lets the retailer registration handler exercise
// its happy path under unit tests.

type inMemoryRetailerRepo struct {
	mu             sync.RWMutex
	byID           map[string]retailer.Retailer
	byPhone        map[string]string // phone -> id
	outboxAppender OutboxAppender
}

func NewRetailerRepo(outboxAppender OutboxAppender) *inMemoryRetailerRepo {
	return &inMemoryRetailerRepo{
		byID:           make(map[string]retailer.Retailer),
		byPhone:        make(map[string]string),
		outboxAppender: outboxAppender,
	}
}

func (r *inMemoryRetailerRepo) CreateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byPhone[ret.Phone]; exists {
		return fmt.Errorf("retailer phone already registered")
	}
	if emit != nil {
		// Scaffold txn: persist the row and the outbox event in-memory together.
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	if ret.CreatedAt.IsZero() {
		ret.CreatedAt = time.Now().UTC()
	}
	r.byID[ret.RetailerID] = ret
	r.byPhone[ret.Phone] = ret.RetailerID
	return nil
}

func (r *inMemoryRetailerRepo) FindByPhone(_ context.Context, phone string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byPhone[phone]
	if !ok {
		return retailer.Retailer{}, false, nil
	}
	return r.byID[id], true, nil
}

func (r *inMemoryRetailerRepo) GetRetailer(_ context.Context, retailerID string) (retailer.Retailer, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ret, ok := r.byID[retailerID]
	return ret, ok, nil
}

func (r *inMemoryRetailerRepo) UpdateRetailer(ctx context.Context, ret retailer.Retailer, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[ret.RetailerID]; !exists {
		return fmt.Errorf("retailer not found")
	}
	if emit != nil {
		txn := &inMemoryTxnBuffer{}
		if err := emit(txn); err != nil {
			return err
		}
		if r.outboxAppender != nil {
			if err := r.outboxAppender.Append(ctx, txn.events); err != nil {
				return err
			}
		}
	}
	ret.UpdatedAt = time.Now().UTC()
	r.byID[ret.RetailerID] = ret
	r.byPhone[ret.Phone] = ret.RetailerID
	_ = ctx
	return nil
}

func (r *inMemoryRetailerRepo) ListRetailersBySupplier(_ context.Context, supplierID string) ([]retailer.Retailer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]retailer.Retailer, 0, len(r.byID))
	for _, ret := range r.byID {
		if supplierID != "" && ret.SupplierID != supplierID {
			continue
		}
		out = append(out, ret)
	}
	return out, nil
}

func (r *inMemoryRetailerRepo) GetSupplierPricingRule(_ context.Context, _ string) (retailer.SupplierPricingRule, bool, error) {
	return retailer.SupplierPricingRule{}, false, nil
}

func (r *inMemoryRetailerRepo) ListTrackingOrders(_ context.Context, _ string, _, _ int) ([]retailer.TrackingOrder, error) {
	return []retailer.TrackingOrder{}, nil
}

func (r *inMemoryRetailerRepo) ListRecentReceipts(_ context.Context, _ string, _ int) ([]retailer.TrackingOrder, error) {
	return []retailer.TrackingOrder{}, nil
}
