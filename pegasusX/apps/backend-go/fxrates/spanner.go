package fxrates

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists FxRates.
type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) Upsert(ctx context.Context, rate Rate) error {
	if rate.RateID == "" {
		rate.RateID = uuid.NewString()
	}
	if rate.Scale <= 0 {
		rate.Scale = DefaultScale
	}
	rate.BaseCurrency = NormalizeCurrency(rate.BaseCurrency)
	rate.QuoteCurrency = NormalizeCurrency(rate.QuoteCurrency)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("FxRates", map[string]any{
			"RateId":        rate.RateID,
			"BaseCurrency":  rate.BaseCurrency,
			"QuoteCurrency": rate.QuoteCurrency,
			"RateScaled":    rate.RateScaled,
			"Scale":         rate.Scale,
			"EffectiveAt":   rate.EffectiveAt,
			"Source":        rate.Source,
			"CreatedAt":     spanner.CommitTimestamp,
		}),
	})
	return err
}

func (r *SpannerRepository) GetAsOf(ctx context.Context, base, quote string, at time.Time) (Rate, bool, error) {
	base = NormalizeCurrency(base)
	quote = NormalizeCurrency(quote)
	stmt := spanner.Statement{
		SQL: `SELECT RateId, BaseCurrency, QuoteCurrency, RateScaled, Scale, EffectiveAt, Source, CreatedAt
		      FROM FxRates
		      WHERE BaseCurrency = @base AND QuoteCurrency = @quote AND EffectiveAt <= @at
		      ORDER BY EffectiveAt DESC
		      LIMIT 1`,
		Params: map[string]any{"base": base, "quote": quote, "at": at},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Rate{}, false, nil
	}
	if err != nil {
		if isNotFound(err) {
			return Rate{}, false, nil
		}
		return Rate{}, false, err
	}
	rate, err := scanRate(row)
	if err != nil {
		return Rate{}, false, err
	}
	return rate, true, nil
}

func (r *SpannerRepository) ListLatest(ctx context.Context, limit int) ([]Rate, error) {
	if limit <= 0 {
		limit = 100
	}
	// Spanner lacks DISTINCT ON; fetch recent rows and collapse in Go.
	stmt := spanner.Statement{
		SQL: `SELECT RateId, BaseCurrency, QuoteCurrency, RateScaled, Scale, EffectiveAt, Source, CreatedAt
		      FROM FxRates
		      ORDER BY EffectiveAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"lim": limit * 10},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	type key struct{ b, q string }
	latest := map[key]Rate{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		rate, err := scanRate(row)
		if err != nil {
			return nil, err
		}
		k := key{rate.BaseCurrency, rate.QuoteCurrency}
		if _, ok := latest[k]; !ok {
			latest[k] = rate
		}
		if len(latest) >= limit {
			break
		}
	}
	out := make([]Rate, 0, len(latest))
	for _, rate := range latest {
		out = append(out, rate)
	}
	return out, nil
}

func scanRate(row *spanner.Row) (Rate, error) {
	var rate Rate
	if err := row.Columns(
		&rate.RateID, &rate.BaseCurrency, &rate.QuoteCurrency,
		&rate.RateScaled, &rate.Scale, &rate.EffectiveAt, &rate.Source, &rate.CreatedAt,
	); err != nil {
		return Rate{}, err
	}
	return rate, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") || strings.Contains(msg, "NotFound") ||
		strings.Contains(msg, "Table not found") || strings.Contains(msg, "code = NotFound")
}

// Ensure table presence helper for seed/SSMR skip.
func TableMissing(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "FxRates") &&
		(strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound"))
}

func FormatRate(r Rate) string {
	return fmt.Sprintf("%s/%s scaled=%d/%d @%s src=%s",
		r.BaseCurrency, r.QuoteCurrency, r.RateScaled, r.Scale,
		r.EffectiveAt.UTC().Format(time.RFC3339), r.Source)
}
