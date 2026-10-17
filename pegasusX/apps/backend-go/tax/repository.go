package tax

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Repository is the storage seam for tax regime operations.
type Repository interface {
	// GetActiveRegime returns the regime effective at the given timestamp for a country.
	GetActiveRegime(ctx context.Context, txn *spanner.ReadWriteTransaction, countryCode string, ts time.Time) (TaxRegimeVersion, bool, error)
	GetRegime(ctx context.Context, id string) (TaxRegimeVersion, bool, error)
	ListRegimes(ctx context.Context, countryCode string, limit int) ([]TaxRegimeVersion, error)
	CreateRegime(ctx context.Context, regime TaxRegimeVersion) error
	InsertLineSnapshot(ctx context.Context, txn *spanner.ReadWriteTransaction, snapshot OrderLineFiscalSnapshot) error
}

// SpannerRepository implements Repository against Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed tax repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) GetActiveRegime(ctx context.Context, txn *spanner.ReadWriteTransaction, countryCode string, ts time.Time) (TaxRegimeVersion, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Id, CountryCode, EffectiveFrom, EffectiveTo, Currency,
		             VatRatesBps, SimplifiedRules, CreatedAt, CreatedBy, UpdatedAt
		      FROM TaxRegimeVersions
		      WHERE CountryCode = @country
		        AND EffectiveFrom <= @ts
		        AND (EffectiveTo IS NULL OR EffectiveTo > @ts)
		      ORDER BY EffectiveFrom DESC
		      LIMIT 1`,
		Params: map[string]interface{}{
			"country": countryCode,
			"ts":      ts,
		},
	}
	
	var iter *spanner.RowIterator
	if txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = r.client.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return TaxRegimeVersion{}, false, nil
	}
	if err != nil {
		return TaxRegimeVersion{}, false, fmt.Errorf("get active regime: %w", err)
	}
	regime, err := scanRegime(row)
	if err != nil {
		return TaxRegimeVersion{}, false, err
	}
	return regime, true, nil
}

func (r *SpannerRepository) GetRegime(ctx context.Context, id string) (TaxRegimeVersion, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "TaxRegimeVersions",
		spanner.Key{id},
		[]string{"Id", "CountryCode", "EffectiveFrom", "EffectiveTo", "Currency",
			"VatRatesBps", "SimplifiedRules", "CreatedAt", "CreatedBy", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == 5 { // NOT_FOUND
			return TaxRegimeVersion{}, false, nil
		}
		return TaxRegimeVersion{}, false, fmt.Errorf("get regime: %w", err)
	}
	regime, scanErr := scanRegime(row)
	if scanErr != nil {
		return TaxRegimeVersion{}, false, scanErr
	}
	return regime, true, nil
}

func (r *SpannerRepository) ListRegimes(ctx context.Context, countryCode string, limit int) ([]TaxRegimeVersion, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT Id, CountryCode, EffectiveFrom, EffectiveTo, Currency,
		             VatRatesBps, SimplifiedRules, CreatedAt, CreatedBy, UpdatedAt
		      FROM TaxRegimeVersions
		      WHERE CountryCode = @country
		      ORDER BY EffectiveFrom DESC
		      LIMIT @limit`,
		Params: map[string]interface{}{
			"country": countryCode,
			"limit":   int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var regimes []TaxRegimeVersion
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list regimes: %w", err)
		}
		regime, scanErr := scanRegime(row)
		if scanErr != nil {
			return nil, scanErr
		}
		regimes = append(regimes, regime)
	}
	return regimes, nil
}

func (r *SpannerRepository) CreateRegime(ctx context.Context, regime TaxRegimeVersion) error {
	m, err := regimeToMutation(regime)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return fmt.Errorf("create regime: %w", err)
	}
	return nil
}

func (r *SpannerRepository) InsertLineSnapshot(ctx context.Context, txn *spanner.ReadWriteTransaction, s OrderLineFiscalSnapshot) error {
	m, err := spanner.InsertOrUpdateStruct("OrderLineFiscalSnapshots", &struct {
		OrderId     string    `spanner:"OrderId"`
		OrderLineId string    `spanner:"OrderLineId"`
		RegimeId    string    `spanner:"RegimeId"`
		VatRateBps  int64     `spanner:"VatRateBps"`
		NetMinor    int64     `spanner:"NetMinor"`
		VatMinor    int64     `spanner:"VatMinor"`
		GrossMinor  int64     `spanner:"GrossMinor"`
		SnapshotAt  time.Time `spanner:"SnapshotAt"`
		CreatedAt   time.Time `spanner:"CreatedAt"`
	}{
		OrderId:     s.OrderId,
		OrderLineId: s.OrderLineId,
		RegimeId:    s.RegimeId,
		VatRateBps:  s.VatRateBps, // this tracks the single applied rate out of the array
		NetMinor:    s.NetMinor,
		VatMinor:    s.VatMinor,
		GrossMinor:  s.GrossMinor,
		SnapshotAt:  s.SnapshotAt,
		CreatedAt:   s.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("build fiscal snapshot mutation: %w", err)
	}
	if txn != nil {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func scanRegime(row *spanner.Row) (TaxRegimeVersion, error) {
	var r TaxRegimeVersion
	var effectiveTo spanner.NullTime
	var simplifiedRules spanner.NullJSON

	// Must read vatRates using []spanner.NullInt64 or direct []int64. Wait, direct []int64 can panic if null.
	// So we'll use a local var for it if nulls are possible, or just []spanner.NullInt64.
	// Since ARRAY<INT64> can have NULL elements or be NULL itself, let's read into []spanner.NullInt64.
	var vatRatesBps []spanner.NullInt64

	if err := row.Columns(
		&r.Id, &r.CountryCode, &r.EffectiveFrom, &effectiveTo, &r.Currency,
		&vatRatesBps, &simplifiedRules, &r.CreatedAt, &r.CreatedBy, &r.UpdatedAt,
	); err != nil {
		return TaxRegimeVersion{}, fmt.Errorf("scan regime: %w", err)
	}
	if effectiveTo.Valid {
		r.EffectiveTo = &effectiveTo.Time
	}
	
	for _, bps := range vatRatesBps {
		if bps.Valid {
			r.VatRatesBps = append(r.VatRatesBps, bps.Int64)
		}
	}
	if r.VatRatesBps == nil {
		r.VatRatesBps = []int64{}
	}

	if simplifiedRules.Valid {
		raw, _ := json.Marshal(simplifiedRules.Value)
		r.SimplifiedRules = raw
	}
	return r, nil
}

func regimeToMutation(r TaxRegimeVersion) (*spanner.Mutation, error) {
	cols := []string{"Id", "CountryCode", "EffectiveFrom", "EffectiveTo", "Currency",
		"VatRatesBps", "SimplifiedRules", "CreatedAt", "CreatedBy", "UpdatedAt"}
	
	var effectiveTo interface{} = nil
	if r.EffectiveTo != nil {
		effectiveTo = *r.EffectiveTo
	}
	
	var simplifiedRules interface{} = nil
	if len(r.SimplifiedRules) > 0 {
		simplifiedRules = spanner.NullJSON{Value: json.RawMessage(r.SimplifiedRules), Valid: true}
	}
	
	// Convert []int64 to []spanner.NullInt64 to ensure it maps correctly if we need,
	// but Spanner client accepts []int64 natively for ARRAY<INT64>.
	vatRates := r.VatRatesBps
	if vatRates == nil {
		vatRates = []int64{}
	}

	return spanner.InsertOrUpdate("TaxRegimeVersions", cols,
		[]interface{}{r.Id, r.CountryCode, r.EffectiveFrom, effectiveTo, r.Currency,
			vatRates, simplifiedRules, spanner.CommitTimestamp, r.CreatedBy, spanner.CommitTimestamp}), nil
}
