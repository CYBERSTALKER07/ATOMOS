package partner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerKeyRepository persists PartnerApiKeys.
type SpannerKeyRepository struct {
	client *spanner.Client
}

func NewSpannerKeyRepository(client *spanner.Client) *SpannerKeyRepository {
	return &SpannerKeyRepository{client: client}
}

func (r *SpannerKeyRepository) Insert(ctx context.Context, k ApiKey) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	m := map[string]any{
		"KeyId":          k.KeyID,
		"TenantType":     k.TenantType,
		"TenantId":       k.TenantID,
		"KeyPrefix":      k.KeyPrefix,
		"KeyHash":        k.KeyHash,
		"Scopes":         k.Scopes,
		"RateLimitClass": k.RateLimitClass,
		"Status":         k.Status,
		"CreatedBy":      k.CreatedBy,
		"CreatedAt":      spanner.CommitTimestamp,
	}
	if k.ExpiresAt != nil {
		m["ExpiresAt"] = *k.ExpiresAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("PartnerApiKeys", m)})
	return err
}

func (r *SpannerKeyRepository) GetByPrefix(ctx context.Context, prefix string) (ApiKey, bool, error) {
	if r == nil || r.client == nil {
		return ApiKey{}, false, fmt.Errorf("spanner_unavailable")
	}
	stmt := spanner.Statement{
		SQL: `SELECT KeyId, TenantType, TenantId, KeyPrefix, KeyHash, Scopes, RateLimitClass, Status, ExpiresAt, CreatedBy, CreatedAt, LastUsedAt
FROM PartnerApiKeys WHERE KeyPrefix = @px LIMIT 1`,
		Params: map[string]any{"px": prefix},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ApiKey{}, false, nil
	}
	if err != nil {
		return ApiKey{}, false, err
	}
	k, err := scanKey(row)
	return k, err == nil, err
}

func (r *SpannerKeyRepository) ListByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]ApiKey, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner_unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT KeyId, TenantType, TenantId, KeyPrefix, KeyHash, Scopes, RateLimitClass, Status, ExpiresAt, CreatedBy, CreatedAt, LastUsedAt
FROM PartnerApiKeys WHERE TenantType = @tt AND TenantId = @tid ORDER BY CreatedAt DESC LIMIT @lim`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID, "lim": int64(limit)},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ApiKey, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		k, err := scanKey(row)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, nil
}

func (r *SpannerKeyRepository) Revoke(ctx context.Context, keyID, tenantType, tenantID string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PartnerApiKeys", spanner.Key{keyID}, []string{"TenantType", "TenantId", "Status"})
		if err != nil {
			return errNotFound("key")
		}
		var tt, tid, status string
		if err := row.Columns(&tt, &tid, &status); err != nil {
			return err
		}
		if tt != tenantType || tid != tenantID {
			return errNotFound("key")
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PartnerApiKeys", map[string]any{"KeyId": keyID, "Status": KeyStatusRevoked}),
		})
	})
	return err
}

func (r *SpannerKeyRepository) TouchLastUsed(ctx context.Context, keyID string) error {
	if r == nil || r.client == nil {
		return nil
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("PartnerApiKeys", map[string]any{"KeyId": keyID, "LastUsedAt": spanner.CommitTimestamp}),
	})
	return err
}

func scanKey(row *spanner.Row) (ApiKey, error) {
	var k ApiKey
	var scopes []string
	var expires, lastUsed spanner.NullTime
	var createdBy spanner.NullString
	var createdAt time.Time
	if err := row.Columns(
		&k.KeyID, &k.TenantType, &k.TenantID, &k.KeyPrefix, &k.KeyHash, &scopes,
		&k.RateLimitClass, &k.Status, &expires, &createdBy, &createdAt, &lastUsed,
	); err != nil {
		return ApiKey{}, err
	}
	k.Scopes = scopes
	k.CreatedBy = createdBy.StringVal
	k.CreatedAt = createdAt
	if expires.Valid {
		t := expires.Time
		k.ExpiresAt = &t
	}
	if lastUsed.Valid {
		t := lastUsed.Time
		k.LastUsedAt = &t
	}
	return k, nil
}

// SpannerWebhookRepository persists webhook tables.
type SpannerWebhookRepository struct {
	client *spanner.Client
}

func NewSpannerWebhookRepository(client *spanner.Client) *SpannerWebhookRepository {
	return &SpannerWebhookRepository{client: client}
}

func (r *SpannerWebhookRepository) InsertSubscription(ctx context.Context, s WebhookSubscription) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("WebhookSubscriptions", map[string]any{
			"SubscriptionId": s.SubscriptionID,
			"TenantType":     s.TenantType,
			"TenantId":       s.TenantID,
			"Url":            s.URL,
			"SigningSecret":  s.SigningSecret,
			"EventTypes":     s.EventTypes,
			"IsActive":       s.IsActive,
			"CreatedAt":      spanner.CommitTimestamp,
			"UpdatedAt":      spanner.CommitTimestamp,
		}),
	})
	return err
}

func (r *SpannerWebhookRepository) ListSubscriptions(ctx context.Context, tenantType, tenantID string) ([]WebhookSubscription, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SubscriptionId, TenantType, TenantId, Url, SigningSecret, EventTypes, IsActive, CreatedAt
FROM WebhookSubscriptions WHERE TenantType = @tt AND TenantId = @tid`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID},
	}
	return r.querySubs(ctx, stmt)
}

func (r *SpannerWebhookRepository) ListActiveByEvent(ctx context.Context, eventType string) ([]WebhookSubscription, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SubscriptionId, TenantType, TenantId, Url, SigningSecret, EventTypes, IsActive, CreatedAt
FROM WebhookSubscriptions WHERE IsActive = TRUE`,
	}
	subs, err := r.querySubs(ctx, stmt)
	if err != nil {
		return nil, err
	}
	out := make([]WebhookSubscription, 0, len(subs))
	for _, s := range subs {
		if subscriptionWants(s.EventTypes, eventType) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *SpannerWebhookRepository) GetSubscription(ctx context.Context, id string) (WebhookSubscription, bool, error) {
	if r == nil || r.client == nil {
		return WebhookSubscription{}, false, fmt.Errorf("spanner_unavailable")
	}
	row, err := r.client.Single().ReadRow(ctx, "WebhookSubscriptions", spanner.Key{id},
		[]string{"SubscriptionId", "TenantType", "TenantId", "Url", "SigningSecret", "EventTypes", "IsActive", "CreatedAt"})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return WebhookSubscription{}, false, nil
		}
		return WebhookSubscription{}, false, err
	}
	s, err := scanSub(row)
	return s, err == nil, err
}

func (r *SpannerWebhookRepository) DeactivateSubscription(ctx context.Context, id, tenantType, tenantID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "WebhookSubscriptions", spanner.Key{id}, []string{"TenantType", "TenantId"})
		if err != nil {
			return errNotFound("subscription")
		}
		var tt, tid string
		if err := row.Columns(&tt, &tid); err != nil {
			return err
		}
		if tt != tenantType || tid != tenantID {
			return errNotFound("subscription")
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("WebhookSubscriptions", map[string]any{
				"SubscriptionId": id, "IsActive": false, "UpdatedAt": spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

func (r *SpannerWebhookRepository) InsertAttempt(ctx context.Context, a DeliveryAttempt) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner_unavailable")
	}
	var payload spanner.NullJSON
	if len(a.PayloadJSON) > 0 {
		var v any
		if err := json.Unmarshal(a.PayloadJSON, &v); err == nil {
			payload = spanner.NullJSON{Value: v, Valid: true}
		}
	}
	m := map[string]any{
		"AttemptId":      a.AttemptID,
		"SubscriptionId": a.SubscriptionID,
		"EventId":        a.EventID,
		"EventType":      a.EventType,
		"PayloadJson":    payload,
		"Status":         a.Status,
		"AttemptCount":   a.AttemptCount,
		"CreatedAt":      spanner.CommitTimestamp,
		"UpdatedAt":      spanner.CommitTimestamp,
	}
	if a.NextRetryAt != nil {
		m["NextRetryAt"] = *a.NextRetryAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("WebhookDeliveryAttempts", m)})
	return err
}

func (r *SpannerWebhookRepository) GetAttemptBySubEvent(ctx context.Context, subID, eventID string) (DeliveryAttempt, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AttemptId, SubscriptionId, EventId, EventType, PayloadJson, Status, HttpCode, NextRetryAt, AttemptCount, LastError, CreatedAt
FROM WebhookDeliveryAttempts WHERE SubscriptionId = @sid AND EventId = @eid LIMIT 1`,
		Params: map[string]any{"sid": subID, "eid": eventID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return DeliveryAttempt{}, false, nil
	}
	if err != nil {
		return DeliveryAttempt{}, false, err
	}
	a, err := scanAttempt(row)
	return a, err == nil, err
}

func (r *SpannerWebhookRepository) GetAttempt(ctx context.Context, attemptID string) (DeliveryAttempt, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "WebhookDeliveryAttempts", spanner.Key{attemptID},
		[]string{"AttemptId", "SubscriptionId", "EventId", "EventType", "PayloadJson", "Status", "HttpCode", "NextRetryAt", "AttemptCount", "LastError", "CreatedAt"})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return DeliveryAttempt{}, false, nil
		}
		return DeliveryAttempt{}, false, err
	}
	a, err := scanAttempt(row)
	return a, err == nil, err
}

func (r *SpannerWebhookRepository) ListDueAttempts(ctx context.Context, now time.Time, limit int) ([]DeliveryAttempt, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT AttemptId, SubscriptionId, EventId, EventType, PayloadJson, Status, HttpCode, NextRetryAt, AttemptCount, LastError, CreatedAt
FROM WebhookDeliveryAttempts
WHERE Status IN ('PENDING','FAILED') AND (NextRetryAt IS NULL OR NextRetryAt <= @now)
LIMIT @lim`,
		Params: map[string]any{"now": now, "lim": int64(limit)},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]DeliveryAttempt, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		a, err := scanAttempt(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *SpannerWebhookRepository) UpdateAttempt(ctx context.Context, a DeliveryAttempt) error {
	m := map[string]any{
		"AttemptId":    a.AttemptID,
		"Status":       a.Status,
		"HttpCode":     a.HTTPCode,
		"AttemptCount": a.AttemptCount,
		"LastError":    a.LastError,
		"UpdatedAt":    spanner.CommitTimestamp,
	}
	if a.NextRetryAt != nil {
		m["NextRetryAt"] = *a.NextRetryAt
	} else {
		m["NextRetryAt"] = spanner.NullTime{}
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.UpdateMap("WebhookDeliveryAttempts", m)})
	return err
}

func (r *SpannerWebhookRepository) ListDeadByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]DeliveryAttempt, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT a.AttemptId, a.SubscriptionId, a.EventId, a.EventType, a.PayloadJson, a.Status, a.HttpCode, a.NextRetryAt, a.AttemptCount, a.LastError, a.CreatedAt
FROM WebhookDeliveryAttempts a
JOIN WebhookSubscriptions s ON s.SubscriptionId = a.SubscriptionId
WHERE a.Status = 'DEAD' AND s.TenantType = @tt AND s.TenantId = @tid
ORDER BY a.UpdatedAt DESC LIMIT @lim`,
		Params: map[string]any{"tt": tenantType, "tid": tenantID, "lim": int64(limit)},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]DeliveryAttempt, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		a, err := scanAttempt(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *SpannerWebhookRepository) querySubs(ctx context.Context, stmt spanner.Statement) ([]WebhookSubscription, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner_unavailable")
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]WebhookSubscription, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		s, err := scanSub(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func scanSub(row *spanner.Row) (WebhookSubscription, error) {
	var s WebhookSubscription
	var types []string
	var created time.Time
	if err := row.Columns(&s.SubscriptionID, &s.TenantType, &s.TenantID, &s.URL, &s.SigningSecret, &types, &s.IsActive, &created); err != nil {
		return WebhookSubscription{}, err
	}
	s.EventTypes = types
	s.CreatedAt = created
	return s, nil
}

func scanAttempt(row *spanner.Row) (DeliveryAttempt, error) {
	var a DeliveryAttempt
	var payload spanner.NullJSON
	var httpCode spanner.NullInt64
	var nextRetry spanner.NullTime
	var lastErr spanner.NullString
	var created time.Time
	if err := row.Columns(
		&a.AttemptID, &a.SubscriptionID, &a.EventID, &a.EventType, &payload,
		&a.Status, &httpCode, &nextRetry, &a.AttemptCount, &lastErr, &created,
	); err != nil {
		return DeliveryAttempt{}, err
	}
	if payload.Valid {
		b, _ := json.Marshal(payload.Value)
		a.PayloadJSON = b
	}
	if httpCode.Valid {
		a.HTTPCode = httpCode.Int64
	}
	if nextRetry.Valid {
		t := nextRetry.Time
		a.NextRetryAt = &t
	}
	a.LastError = lastErr.StringVal
	a.CreatedAt = created
	return a, nil
}
