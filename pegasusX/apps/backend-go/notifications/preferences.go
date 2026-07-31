package notifications

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// NotificationPreference represents user preferences for notifications.
type NotificationPreference struct {
	PrincipalID   string    `json:"principal_id"`
	PrincipalType string    `json:"principal_type"`
	EventType     string    `json:"event_type"`
	Channel       string    `json:"channel"`
	Enabled       bool      `json:"enabled"`
	QuietFrom     string    `json:"quiet_from"`
	QuietTo       string    `json:"quiet_to"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func scanPreference(row *spanner.Row) (*NotificationPreference, error) {
	var p NotificationPreference
	var quietFrom, quietTo spanner.NullString
	if err := row.Columns(&p.PrincipalID, &p.PrincipalType, &p.EventType, &p.Channel, &p.Enabled, &quietFrom, &quietTo, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan preference: %w", err)
	}
	p.QuietFrom = quietFrom.StringVal
	p.QuietTo = quietTo.StringVal
	return &p, nil
}

// GetPreference fetches a specific notification preference. Returns nil, nil if not found.
func (r *SpannerRepository) GetPreference(ctx context.Context, principalID, eventType, channel string) (*NotificationPreference, error) {
	stmt := spanner.Statement{
		SQL: `SELECT PrincipalId, PrincipalType, EventType, Channel, Enabled, QuietFrom, QuietTo, UpdatedAt
			FROM NotificationPreferences
			WHERE PrincipalId = @pid AND EventType = @evt AND Channel = @ch`,
		Params: map[string]any{
			"pid": principalID,
			"evt": eventType,
			"ch":  channel,
		},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query preference: %w", err)
	}
	return scanPreference(row)
}

// ListPreferencesForPrincipal returns all preferences for a principal.
func (r *SpannerRepository) ListPreferencesForPrincipal(ctx context.Context, principalID string) ([]NotificationPreference, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT PrincipalId, PrincipalType, EventType, Channel, Enabled, QuietFrom, QuietTo, UpdatedAt
			FROM NotificationPreferences WHERE PrincipalId = @pid`,
		Params: map[string]any{"pid": principalID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []NotificationPreference
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list preferences: %w", err)
		}
		p, err := scanPreference(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, nil
}

// UpsertPreference inserts or updates a notification preference.
func (r *SpannerRepository) UpsertPreference(ctx context.Context, pref NotificationPreference) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("NotificationPreferences", map[string]any{
			"PrincipalId":   pref.PrincipalID,
			"PrincipalType": pref.PrincipalType,
			"EventType":     pref.EventType,
			"Channel":       pref.Channel,
			"Enabled":       pref.Enabled,
			"QuietFrom":     spanner.NullString{StringVal: pref.QuietFrom, Valid: pref.QuietFrom != ""},
			"QuietTo":       spanner.NullString{StringVal: pref.QuietTo, Valid: pref.QuietTo != ""},
			"UpdatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("upsert preference for %s: %w", pref.PrincipalID, err)
	}
	return nil
}
