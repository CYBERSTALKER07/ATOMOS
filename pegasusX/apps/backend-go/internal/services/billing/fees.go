package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// FeeSchedule prices a supplier: per-order fee + GMV basis points + monthly
// subscription. Resolution order: supplier-specific row > tier row matching
// SupplierProfiles.Tier > STANDARD tier row > zero schedule (billing off for
// that supplier — honest default, never an invented charge).
type FeeSchedule struct {
	FeeScheduleID            string
	SupplierID               string
	Tier                     string
	PerOrderMinor            int64
	GmvBps                   int64
	MonthlySubscriptionMinor int64
	Currency                 string
	EffectiveFrom            time.Time
	EffectiveTo              *time.Time
}

// ZeroSchedule is the no-billing default (all fees zero). Empty currency stays
// empty — never invent UZS; callers pass the shipped pack when they have one.
func ZeroSchedule(currency string) FeeSchedule {
	return FeeSchedule{Tier: "STANDARD", Currency: strings.ToUpper(strings.TrimSpace(currency))}
}

// MonthlyFee computes the total monthly fee in minor units.
func (s FeeSchedule) MonthlyFee(orderCount, gmvMinor int64) int64 {
	return s.PerOrderMinor*orderCount + gmvMinor*s.GmvBps/10000 + s.MonthlySubscriptionMinor
}

// FeeScheduleResolver loads schedules from Spanner.
type FeeScheduleResolver struct {
	client *spanner.Client
	now    func() time.Time
}

func NewFeeScheduleResolver(client *spanner.Client) *FeeScheduleResolver {
	return &FeeScheduleResolver{client: client, now: func() time.Time { return time.Now().UTC() }}
}

func cleanQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	clean := context.Background()
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(clean, deadline)
	}
	return context.WithTimeout(clean, 10*time.Second)
}

// Resolve returns the active schedule for the supplier (never errors on
// missing rows — absence means billing is off for this supplier).
func (r *FeeScheduleResolver) Resolve(ctx context.Context, supplierID string) (FeeSchedule, error) {
	cleanCtx, cancel := cleanQueryContext(ctx)
	defer cancel()

	now := r.now()
	if sched, found, err := r.find(cleanCtx, `SupplierId = @id`, map[string]any{"id": supplierID}, now); err != nil {
		return FeeSchedule{}, err
	} else if found {
		return sched, nil
	}
	tier := "STANDARD"
	row, err := r.client.Single().ReadRow(cleanCtx, "SupplierProfiles", spanner.Key{supplierID}, []string{"Tier"})
	if err == nil {
		var t spanner.NullString
		if cErr := row.Column(0, &t); cErr == nil && strings.TrimSpace(t.StringVal) != "" {
			tier = strings.ToUpper(strings.TrimSpace(t.StringVal))
		}
	}
	if sched, found, err := r.find(cleanCtx, `SupplierId = '' AND Tier = @tier`, map[string]any{"tier": tier}, now); err != nil {
		return FeeSchedule{}, err
	} else if found {
		return sched, nil
	}
	if tier != "STANDARD" {
		if sched, found, err := r.find(cleanCtx, `SupplierId = '' AND Tier = 'STANDARD'`, nil, now); err != nil {
			return FeeSchedule{}, err
		} else if found {
			return sched, nil
		}
	}
	return ZeroSchedule(packCurrencyOrEmpty(ctx, supplierID)), nil
}

func (r *FeeScheduleResolver) find(ctx context.Context, where string, params map[string]any, now time.Time) (FeeSchedule, bool, error) {
	cleanCtx, cancel := cleanQueryContext(ctx)
	defer cancel()

	if params == nil {
		params = map[string]any{}
	}
	params["now"] = now
	iter := r.client.Single().Query(cleanCtx, spanner.Statement{
		SQL: `SELECT FeeScheduleId, SupplierId, Tier, PerOrderMinor, GmvBps, MonthlySubscriptionMinor, Currency, EffectiveFrom, EffectiveTo
		      FROM BillingFeeSchedules
		      WHERE ` + where + ` AND EffectiveFrom <= @now AND (EffectiveTo IS NULL OR EffectiveTo > @now)
		      ORDER BY EffectiveFrom DESC LIMIT 1`,
		Params: params,
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return FeeSchedule{}, false, nil
	}
	if err != nil {
		return FeeSchedule{}, false, err
	}
	var s FeeSchedule
	var effTo spanner.NullTime
	if err := row.Columns(&s.FeeScheduleID, &s.SupplierID, &s.Tier, &s.PerOrderMinor, &s.GmvBps,
		&s.MonthlySubscriptionMinor, &s.Currency, &s.EffectiveFrom, &effTo); err != nil {
		return FeeSchedule{}, false, err
	}
	if effTo.Valid {
		t := effTo.Time
		s.EffectiveTo = &t
	}
	return s, true, nil
}

// CommissionMinor prices the payout commission: the GMV-bps portion of the
// schedule. Per-order and subscription fees are charged on the monthly billing
// invoice — including them here would double-charge the supplier.
func (r *FeeScheduleResolver) CommissionMinor(ctx context.Context, supplierID string, grossCapturedMinor int64, currency string) (int64, error) {
	sched, err := r.Resolve(ctx, supplierID)
	if err != nil {
		return 0, err
	}
	if sched.Currency != "" && currency != "" && !strings.EqualFold(sched.Currency, currency) {
		return 0, fmt.Errorf("fee schedule currency %s mismatches payout currency %s", sched.Currency, currency)
	}
	return grossCapturedMinor * sched.GmvBps / 10000, nil
}

func packCurrencyOrEmpty(ctx context.Context, supplierID string) string {
	c, err := auth.CurrencyFromContext(ctx, supplierID)
	if err != nil {
		return ""
	}
	return c
}
