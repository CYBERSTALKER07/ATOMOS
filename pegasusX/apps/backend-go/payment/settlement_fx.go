package payment

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

// rollupOperatingCurrencyMinor converts native settlement groups into operating
// currency minors for display. Groups that cannot convert are omitted and
// partial is set (never silent 1:1).
func rollupOperatingCurrencyMinor(
	ctx context.Context,
	fx *fxrates.Service,
	operating string,
	rows []SettlementAuthorityRow,
	nowFn func() time.Time,
) (total int64, partial bool) {
	operating = fxrates.NormalizeCurrency(operating)
	if operating == "" {
		if cur, err := auth.CurrencyFromContext(ctx, ""); err == nil {
			operating = cur
		}
	}
	if operating == "" {
		return 0, true
	}
	now := time.Now().UTC()
	if nowFn != nil {
		now = nowFn()
	}
	for _, row := range rows {
		cur := fxrates.NormalizeCurrency(row.Currency)
		if cur == "" {
			partial = true
			continue
		}
		if cur == operating {
			total += row.AmountMinorTotal
			continue
		}
		if fx == nil {
			partial = true
			continue
		}
		at := row.LastOccurredAt
		if at.IsZero() {
			at = now
		}
		converted, err := fx.ConvertMinor(ctx, cur, operating, row.AmountMinorTotal, at)
		if err != nil {
			partial = true
			continue
		}
		total += converted
	}
	return total, partial
}
