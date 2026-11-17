package partner

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryKeyRepository is an in-memory KeyRepository for tests / local.
type MemoryKeyRepository struct {
	mu   sync.RWMutex
	byID map[string]ApiKey
	byPx map[string]string
}

func NewMemoryKeyRepository() *MemoryKeyRepository {
	return &MemoryKeyRepository{byID: map[string]ApiKey{}, byPx: map[string]string{}}
}

func (r *MemoryKeyRepository) Insert(_ context.Context, k ApiKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[k.KeyID] = k
	r.byPx[k.KeyPrefix] = k.KeyID
	return nil
}

func (r *MemoryKeyRepository) GetByPrefix(_ context.Context, prefix string) (ApiKey, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byPx[prefix]
	if !ok {
		return ApiKey{}, false, nil
	}
	k, ok := r.byID[id]
	return k, ok, nil
}

func (r *MemoryKeyRepository) GetByID(_ context.Context, keyID string) (ApiKey, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	k, ok := r.byID[strings.TrimSpace(keyID)]
	return k, ok, nil
}

func (r *MemoryKeyRepository) ListByTenant(_ context.Context, tenantType, tenantID string, limit int) ([]ApiKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]ApiKey, 0)
	for _, k := range r.byID {
		if k.TenantType == tenantType && k.TenantID == tenantID {
			out = append(out, k)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryKeyRepository) Revoke(_ context.Context, keyID, tenantType, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.byID[keyID]
	if !ok || k.TenantType != tenantType || k.TenantID != tenantID {
		return errNotFound("key")
	}
	k.Status = KeyStatusRevoked
	r.byID[keyID] = k
	return nil
}

func (r *MemoryKeyRepository) TouchLastUsed(_ context.Context, keyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.byID[keyID]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	k.LastUsedAt = &now
	r.byID[keyID] = k
	return nil
}

// MemoryWebhookRepository for tests.
type MemoryWebhookRepository struct {
	mu    sync.RWMutex
	subs  map[string]WebhookSubscription
	atts  map[string]DeliveryAttempt
	byEvt map[string]string // subID|eventID -> attemptID
}

func NewMemoryWebhookRepository() *MemoryWebhookRepository {
	return &MemoryWebhookRepository{
		subs:  map[string]WebhookSubscription{},
		atts:  map[string]DeliveryAttempt{},
		byEvt: map[string]string{},
	}
}

func (r *MemoryWebhookRepository) InsertSubscription(_ context.Context, s WebhookSubscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[s.SubscriptionID] = s
	return nil
}

func (r *MemoryWebhookRepository) ListSubscriptions(_ context.Context, tenantType, tenantID string) ([]WebhookSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WebhookSubscription, 0)
	for _, s := range r.subs {
		if s.TenantType == tenantType && s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryWebhookRepository) ListActiveByEvent(_ context.Context, eventType string) ([]WebhookSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WebhookSubscription, 0)
	for _, s := range r.subs {
		if !s.IsActive {
			continue
		}
		if subscriptionWants(s.EventTypes, eventType) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryWebhookRepository) GetSubscription(_ context.Context, id string) (WebhookSubscription, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.subs[id]
	return s, ok, nil
}

func (r *MemoryWebhookRepository) DeactivateSubscription(_ context.Context, id, tenantType, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.subs[id]
	if !ok || s.TenantType != tenantType || s.TenantID != tenantID {
		return errNotFound("subscription")
	}
	s.IsActive = false
	r.subs[id] = s
	return nil
}

func (r *MemoryWebhookRepository) UpdateSubscriptionSecret(_ context.Context, id, tenantType, tenantID, secret string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.subs[id]
	if !ok || s.TenantType != tenantType || s.TenantID != tenantID {
		return errNotFound("subscription")
	}
	s.SigningSecret = secret
	r.subs[id] = s
	return nil
}

func (r *MemoryWebhookRepository) InsertAttempt(_ context.Context, a DeliveryAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := a.SubscriptionID + "|" + a.EventID
	if _, exists := r.byEvt[key]; exists {
		return nil // idempotent
	}
	r.atts[a.AttemptID] = a
	r.byEvt[key] = a.AttemptID
	return nil
}

func (r *MemoryWebhookRepository) GetAttemptBySubEvent(_ context.Context, subID, eventID string) (DeliveryAttempt, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEvt[subID+"|"+eventID]
	if !ok {
		return DeliveryAttempt{}, false, nil
	}
	a, ok := r.atts[id]
	return a, ok, nil
}

func (r *MemoryWebhookRepository) GetAttempt(_ context.Context, attemptID string) (DeliveryAttempt, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.atts[attemptID]
	return a, ok, nil
}

func (r *MemoryWebhookRepository) ListDueAttempts(_ context.Context, now time.Time, limit int) ([]DeliveryAttempt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]DeliveryAttempt, 0)
	for _, a := range r.atts {
		if a.Status != DeliveryPending && a.Status != DeliveryFailed {
			continue
		}
		if a.NextRetryAt != nil && a.NextRetryAt.After(now) {
			continue
		}
		out = append(out, a)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryWebhookRepository) UpdateAttempt(_ context.Context, a DeliveryAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.atts[a.AttemptID] = a
	return nil
}

func (r *MemoryWebhookRepository) ListDeadByTenant(_ context.Context, tenantType, tenantID string, limit int) ([]DeliveryAttempt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	subIDs := map[string]bool{}
	for _, s := range r.subs {
		if s.TenantType == tenantType && s.TenantID == tenantID {
			subIDs[s.SubscriptionID] = true
		}
	}
	out := make([]DeliveryAttempt, 0)
	for _, a := range r.atts {
		if a.Status != DeliveryDead || !subIDs[a.SubscriptionID] {
			continue
		}
		out = append(out, a)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func subscriptionWants(types []string, eventType string) bool {
	if len(types) == 0 {
		return true
	}
	et := strings.TrimSpace(eventType)
	for _, t := range types {
		if strings.TrimSpace(t) == "*" || strings.TrimSpace(t) == et {
			return true
		}
	}
	return false
}

type notFoundError struct{ what string }

func (e notFoundError) Error() string { return e.what + "_not_found" }

func errNotFound(what string) error { return notFoundError{what: what} }
