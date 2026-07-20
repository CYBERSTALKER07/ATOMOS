package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// PolicyRow is the durable client version policy for a role/platform/channel tuple.
type PolicyRow struct {
	Role               string
	Platform           string
	Channel            string
	MinimumVersion     string
	RecommendedVersion string
	UpdateURL          string
	ForceUpdate        bool
	UpdatedAt          time.Time
}

// DeviceTokenRow stores FCM/APNs registration for push fallback.
type DeviceTokenRow struct {
	ActorID   string
	ActorRole string
	Platform  string
	Token     string
	UpdatedAt time.Time
}

// PolicyRepository loads version policies.
type PolicyRepository interface {
	GetPolicy(ctx context.Context, role, platform, channel string) (PolicyRow, bool, error)
	UpsertPolicy(ctx context.Context, row PolicyRow) error
}

// DeviceTokenRepository persists push tokens.
type DeviceTokenRepository interface {
	UpsertToken(ctx context.Context, row DeviceTokenRow) error
	ListTokens(ctx context.Context, actorID, actorRole string) ([]string, error)
	DeleteToken(ctx context.Context, token string) error
}

// MemoryPolicyRepository is the scaffold fallback when Spanner is unavailable.
type MemoryPolicyRepository struct {
	mu   sync.RWMutex
	rows map[string]PolicyRow
}

// NewMemoryPolicyRepository seeds default permissive policies.
func NewMemoryPolicyRepository() *MemoryPolicyRepository {
	r := &MemoryPolicyRepository{rows: make(map[string]PolicyRow)}
	for _, role := range []string{"DRIVER", "RETAILER", "ADMIN", "PAYLOAD", "WAREHOUSE", "FACTORY"} {
		for _, platform := range []string{"ios", "android", "web", "desktop"} {
			key := policyKey(role, platform, "production")
			r.rows[key] = PolicyRow{
				Role: role, Platform: platform, Channel: "production",
				MinimumVersion: "0.0.0", RecommendedVersion: "0.0.0",
			}
			// Website-only enterprise channel defaults (manifest URL filled at evaluate).
			entKey := policyKey(role, platform, EnterpriseChannel)
			r.rows[entKey] = PolicyRow{
				Role: role, Platform: platform, Channel: EnterpriseChannel,
				MinimumVersion: "0.0.0", RecommendedVersion: "0.0.0",
				UpdateURL: DefaultEnterpriseManifestURL(role, platform),
			}
		}
	}
	return r
}

func policyKey(role, platform, channel string) string {
	return role + "|" + platform + "|" + channel
}

// GetPolicy implements PolicyRepository.
func (r *MemoryPolicyRepository) GetPolicy(ctx context.Context, role, platform, channel string) (PolicyRow, bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.rows[policyKey(role, platform, channel)]
	return row, ok, nil
}

// UpsertPolicy implements PolicyRepository.
func (r *MemoryPolicyRepository) UpsertPolicy(ctx context.Context, row PolicyRow) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	row.UpdatedAt = time.Now().UTC()
	r.rows[policyKey(row.Role, row.Platform, row.Channel)] = row
	return nil
}

// SpannerPolicyRepository implements PolicyRepository on ClientVersionPolicies.
type SpannerPolicyRepository struct {
	client *spanner.Client
}

// NewSpannerPolicyRepository creates a Spanner-backed policy repository.
func NewSpannerPolicyRepository(client *spanner.Client) *SpannerPolicyRepository {
	return &SpannerPolicyRepository{client: client}
}

// GetPolicy reads a policy row by composite key.
func (r *SpannerPolicyRepository) GetPolicy(ctx context.Context, role, platform, channel string) (PolicyRow, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Role, Platform, Channel, MinimumVersion, RecommendedVersion,
			UpdateURL, ForceUpdate, UpdatedAt
			FROM ClientVersionPolicies
			WHERE Role = @role AND Platform = @platform AND Channel = @channel`,
		Params: map[string]any{"role": role, "platform": platform, "channel": channel},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return PolicyRow{}, false, nil
	}
	if err != nil {
		return PolicyRow{}, false, fmt.Errorf("get client policy: %w", err)
	}
	var p PolicyRow
	var updateURL spanner.NullString
	if err := row.Columns(&p.Role, &p.Platform, &p.Channel, &p.MinimumVersion, &p.RecommendedVersion,
		&updateURL, &p.ForceUpdate, &p.UpdatedAt); err != nil {
		return PolicyRow{}, false, err
	}
	p.UpdateURL = updateURL.StringVal
	return p, true, nil
}

// UpsertPolicy writes or replaces a policy row.
func (r *SpannerPolicyRepository) UpsertPolicy(ctx context.Context, row PolicyRow) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("ClientVersionPolicies", map[string]any{
			"Role":               row.Role,
			"Platform":           row.Platform,
			"Channel":            row.Channel,
			"MinimumVersion":     row.MinimumVersion,
			"RecommendedVersion": row.RecommendedVersion,
			"UpdateURL":          spanner.NullString{StringVal: row.UpdateURL, Valid: row.UpdateURL != ""},
			"ForceUpdate":        row.ForceUpdate,
			"UpdatedAt":          spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("upsert client policy: %w", err)
	}
	return nil
}

// SpannerDeviceTokenRepository implements DeviceTokenRepository.
type SpannerDeviceTokenRepository struct {
	client *spanner.Client
}

// NewSpannerDeviceTokenRepository creates a Spanner device token repository.
func NewSpannerDeviceTokenRepository(client *spanner.Client) *SpannerDeviceTokenRepository {
	return &SpannerDeviceTokenRepository{client: client}
}

// UpsertToken inserts or updates a device token for an actor.
func (r *SpannerDeviceTokenRepository) UpsertToken(ctx context.Context, row DeviceTokenRow) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("DeviceTokens", map[string]any{
			"Token":     row.Token,
			"ActorId":   row.ActorID,
			"ActorRole": row.ActorRole,
			"Platform":  row.Platform,
			"UpdatedAt": spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("upsert device token: %w", err)
	}
	return nil
}

// ListTokens returns FCM tokens for an actor.
func (r *SpannerDeviceTokenRepository) ListTokens(ctx context.Context, actorID, actorRole string) ([]string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Token FROM DeviceTokens
			WHERE ActorId = @aid AND ActorRole = @role`,
		Params: map[string]any{"aid": actorID, "role": actorRole},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var tokens []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list device tokens: %w", err)
		}
		var token string
		if err := row.Columns(&token); err != nil {
			return nil, err
		}
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

// DeleteToken removes a stale token.
func (r *SpannerDeviceTokenRepository) DeleteToken(ctx context.Context, token string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Delete("DeviceTokens", spanner.Key{token})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}

// MemoryDeviceTokenRepository is an in-process token store for local SSMR.
type MemoryDeviceTokenRepository struct {
	mu     sync.RWMutex
	tokens map[string]DeviceTokenRow
}

// NewMemoryDeviceTokenRepository creates a memory-backed token repository.
func NewMemoryDeviceTokenRepository() *MemoryDeviceTokenRepository {
	return &MemoryDeviceTokenRepository{tokens: make(map[string]DeviceTokenRow)}
}

// UpsertToken implements DeviceTokenRepository.
func (r *MemoryDeviceTokenRepository) UpsertToken(ctx context.Context, row DeviceTokenRow) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	row.UpdatedAt = time.Now().UTC()
	r.tokens[row.Token] = row
	return nil
}

// ListTokens implements DeviceTokenRepository.
func (r *MemoryDeviceTokenRepository) ListTokens(ctx context.Context, actorID, actorRole string) ([]string, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for _, row := range r.tokens {
		if row.ActorID == actorID && row.ActorRole == actorRole {
			out = append(out, row.Token)
		}
	}
	return out, nil
}

// DeleteToken implements DeviceTokenRepository.
func (r *MemoryDeviceTokenRepository) DeleteToken(ctx context.Context, token string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tokens, token)
	return nil
}
