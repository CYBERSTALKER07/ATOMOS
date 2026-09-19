package mfa

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists PlatformAdminMFA.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) Get(ctx context.Context, subject string) (Record, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "PlatformAdminMFA", spanner.Key{subject},
		[]string{"Subject", "Secret", "Enabled", "CreatedAt", "EnabledAt"})
	if err != nil {
		if err == spanner.ErrRowNotFound {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	var rec Record
	var enabledAt spanner.NullTime
	if err := row.Columns(&rec.Subject, &rec.Secret, &rec.Enabled, &rec.CreatedAt, &enabledAt); err != nil {
		return Record{}, false, err
	}
	if enabledAt.Valid {
		rec.EnabledAt = enabledAt.Time
	}
	return rec, true, nil
}

func (r *SpannerRepository) Upsert(ctx context.Context, row Record) error {
	cols := map[string]any{
		"Subject":   row.Subject,
		"Secret":    row.Secret,
		"Enabled":   row.Enabled,
		"CreatedAt": row.CreatedAt,
	}
	if row.CreatedAt.IsZero() {
		cols["CreatedAt"] = spanner.CommitTimestamp
	}
	if row.Enabled && !row.EnabledAt.IsZero() {
		cols["EnabledAt"] = row.EnabledAt
	} else if row.Enabled {
		cols["EnabledAt"] = time.Now().UTC()
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("PlatformAdminMFA", cols)})
	return err
}

// ListEnabled is unused by handlers but keeps iterator import honest for future admin tooling.
func (r *SpannerRepository) ListEnabled(ctx context.Context, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 100
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT Subject, Secret, Enabled, CreatedAt, EnabledAt
			FROM PlatformAdminMFA WHERE Enabled = TRUE LIMIT @lim`,
		Params: map[string]any{"lim": int64(limit)},
	})
	defer iter.Stop()
	out := make([]Record, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec Record
		var enabledAt spanner.NullTime
		if err := row.Columns(&rec.Subject, &rec.Secret, &rec.Enabled, &rec.CreatedAt, &enabledAt); err != nil {
			return nil, err
		}
		if enabledAt.Valid {
			rec.EnabledAt = enabledAt.Time
		}
		out = append(out, rec)
	}
	return out, nil
}
