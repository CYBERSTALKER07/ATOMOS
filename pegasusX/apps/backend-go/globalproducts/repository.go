package globalproducts

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Repository persists global products, offers, and match queue rows.
type Repository interface {
	EnsureStandardUoM(ctx context.Context) error
	GetByGtin(ctx context.Context, gtin string) (*GlobalProduct, error)
	GetByID(ctx context.Context, id string) (*GlobalProduct, error)
	ListByNormalizedKey(ctx context.Context, key string) ([]GlobalProduct, error)
	ListAll(ctx context.Context, limit int) ([]GlobalProduct, error)
	UpsertGlobal(ctx context.Context, gp GlobalProduct) error
	UpsertOffer(ctx context.Context, o Offer) error
	GetOffer(ctx context.Context, supplierID, productID string) (*Offer, error)
	ListOffersByGlobal(ctx context.Context, globalProductID string) ([]Offer, error)
	EnqueueMatch(ctx context.Context, item MatchQueueItem) error
	ListMatchQueue(ctx context.Context, status string, limit int) ([]MatchQueueItem, error)
	GetMatchQueueItem(ctx context.Context, queueID string) (*MatchQueueItem, error)
	UpdateMatchQueue(ctx context.Context, item MatchQueueItem) error
}

// MemoryRepository is an in-memory repo for unit tests.
type MemoryRepository struct {
	mu       sync.Mutex
	byID     map[string]GlobalProduct
	byGtin   map[string]string
	byKey    map[string][]string
	offers   map[string]Offer // supplier|product
	queue    map[string]MatchQueueItem
	uomReady bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:   map[string]GlobalProduct{},
		byGtin: map[string]string{},
		byKey:  map[string][]string{},
		offers: map[string]Offer{},
		queue:  map[string]MatchQueueItem{},
	}
}

func (r *MemoryRepository) EnsureStandardUoM(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uomReady = true
	return nil
}

func (r *MemoryRepository) GetByGtin(_ context.Context, gtin string) (*GlobalProduct, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byGtin[gtin]
	if !ok {
		return nil, nil
	}
	gp := r.byID[id]
	return &gp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (*GlobalProduct, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	gp, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	return &gp, nil
}

func (r *MemoryRepository) ListByNormalizedKey(_ context.Context, key string) ([]GlobalProduct, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byKey[key]
	out := make([]GlobalProduct, 0, len(ids))
	for _, id := range ids {
		if gp, ok := r.byID[id]; ok {
			out = append(out, gp)
		}
	}
	return out, nil
}

func (r *MemoryRepository) ListAll(_ context.Context, limit int) ([]GlobalProduct, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]GlobalProduct, 0, len(r.byID))
	for _, gp := range r.byID {
		out = append(out, gp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) UpsertGlobal(_ context.Context, gp GlobalProduct) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if gp.GlobalProductID == "" {
		gp.GlobalProductID = uuid.NewString()
	}
	now := time.Now().UTC()
	if gp.CreatedAt.IsZero() {
		gp.CreatedAt = now
	}
	gp.UpdatedAt = now
	r.byID[gp.GlobalProductID] = gp
	if gp.Gtin != "" {
		r.byGtin[gp.Gtin] = gp.GlobalProductID
	}
	if gp.NormalizedKey != "" {
		ids := r.byKey[gp.NormalizedKey]
		found := false
		for _, id := range ids {
			if id == gp.GlobalProductID {
				found = true
				break
			}
		}
		if !found {
			r.byKey[gp.NormalizedKey] = append(ids, gp.GlobalProductID)
		}
	}
	return nil
}

func offerKey(supplierID, productID string) string { return supplierID + "|" + productID }

func (r *MemoryRepository) UpsertOffer(_ context.Context, o Offer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
	r.offers[offerKey(o.SupplierID, o.ProductID)] = o
	return nil
}

func (r *MemoryRepository) GetOffer(_ context.Context, supplierID, productID string) (*Offer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.offers[offerKey(supplierID, productID)]
	if !ok {
		return nil, nil
	}
	return &o, nil
}

func (r *MemoryRepository) ListOffersByGlobal(_ context.Context, globalProductID string) ([]Offer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Offer
	for _, o := range r.offers {
		if o.GlobalProductID == globalProductID && o.Status == StatusLinked {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *MemoryRepository) EnqueueMatch(_ context.Context, item MatchQueueItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item.QueueID == "" {
		item.QueueID = uuid.NewString()
	}
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	r.queue[item.QueueID] = item
	return nil
}

func (r *MemoryRepository) ListMatchQueue(_ context.Context, status string, limit int) ([]MatchQueueItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []MatchQueueItem
	for _, q := range r.queue {
		if status != "" && q.Status != status {
			continue
		}
		out = append(out, q)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryRepository) GetMatchQueueItem(_ context.Context, queueID string) (*MatchQueueItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, ok := r.queue[queueID]
	if !ok {
		return nil, nil
	}
	return &q, nil
}

func (r *MemoryRepository) UpdateMatchQueue(_ context.Context, item MatchQueueItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.UpdatedAt = time.Now().UTC()
	r.queue[item.QueueID] = item
	return nil
}
