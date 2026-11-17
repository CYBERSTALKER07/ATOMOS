package multientity

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

type resolvedRate struct {
	Rate    fxrates.Rate
	Inverse bool
}

func resolveRateAsOf(
	ctx context.Context,
	fxRepo fxrates.Repository,
	fxSvc *fxrates.Service,
	fromCur, toCur string,
	asOf time.Time,
) (resolvedRate, error) {
	fromCur = fxrates.NormalizeCurrency(fromCur)
	toCur = fxrates.NormalizeCurrency(toCur)
	if len(fromCur) != 3 || len(toCur) != 3 {
		return resolvedRate{}, fmt.Errorf("%w: from=%q to=%q", ErrInvalidCurrency, fromCur, toCur)
	}
	if fromCur == toCur {
		return resolvedRate{
			Rate: fxrates.IdentityRate(fromCur, "IDENTITY", asOf),
		}, nil
	}
	if fxRepo == nil {
		return resolvedRate{}, fmt.Errorf("%w: fx repository unavailable for %s->%s", ErrFxRateMissing, fromCur, toCur)
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}

	rate, ok, err := fxRepo.GetAsOf(ctx, fromCur, toCur, asOf)
	if err != nil {
		return resolvedRate{}, fmt.Errorf("lookup fx rate %s->%s: %w", fromCur, toCur, err)
	}
	if ok {
		if rate.Scale <= 0 {
			rate.Scale = fxrates.DefaultScale
		}
		if rate.RateScaled <= 0 {
			return resolvedRate{}, fmt.Errorf("%w: non-positive rate %s->%s", ErrFxRateMissing, fromCur, toCur)
		}
		return resolvedRate{Rate: rate, Inverse: false}, nil
	}

	allowInv := true
	if fxSvc != nil {
		allowInv = fxSvc.AllowInverse
	}
	if allowInv {
		inv, okInv, err := fxRepo.GetAsOf(ctx, toCur, fromCur, asOf)
		if err != nil {
			return resolvedRate{}, fmt.Errorf("lookup inverse fx rate %s->%s: %w", toCur, fromCur, err)
		}
		if okInv {
			if inv.Scale <= 0 {
				inv.Scale = fxrates.DefaultScale
			}
			if inv.RateScaled <= 0 {
				return resolvedRate{}, fmt.Errorf("%w: non-positive inverse rate %s->%s", ErrFxRateMissing, toCur, fromCur)
			}
			return resolvedRate{Rate: inv, Inverse: true}, nil
		}
	}

	return resolvedRate{}, fmt.Errorf("%w: no FX rate for %s/%s as of %s", ErrFxRateMissing, fromCur, toCur, asOf.UTC().Format(time.RFC3339))
}

// convertSignedMinor converts any signed int64 minor-unit amount using pure BigInt 1e8 integer math
// with half-away-from-zero rounding (zero floating point arithmetic).
func convertSignedMinor(amountMinor int64, rr resolvedRate, fromCur, toCur string) (int64, error) {
	if amountMinor == 0 {
		return 0, nil
	}
	fromCur = fxrates.NormalizeCurrency(fromCur)
	toCur = fxrates.NormalizeCurrency(toCur)
	if fromCur == toCur {
		return amountMinor, nil
	}
	scale := rr.Rate.Scale
	if scale <= 0 {
		scale = fxrates.DefaultScale
	}
	if rr.Rate.RateScaled <= 0 {
		return 0, fmt.Errorf("%w: invalid rate scaled=%d scale=%d", ErrFxRateMissing, rr.Rate.RateScaled, scale)
	}

	neg := amountMinor < 0
	absAmt := amountMinor
	if neg {
		if amountMinor == math.MinInt64 {
			return 0, fmt.Errorf("fx overflow on MinInt64")
		}
		absAmt = -amountMinor
	}

	a := big.NewInt(absAmt)
	r := big.NewInt(rr.Rate.RateScaled)
	sc := big.NewInt(scale)
	num := new(big.Int)
	den := new(big.Int)
	if rr.Inverse {
		num.Mul(a, sc)
		den.Set(r)
	} else {
		num.Mul(a, r)
		den.Set(sc)
	}

	expDiff := fxrates.CurrencyExponent(toCur) - fxrates.CurrencyExponent(fromCur)
	if expDiff > 0 {
		pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(expDiff)), nil)
		num.Mul(num, pow)
	} else if expDiff < 0 {
		pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-expDiff)), nil)
		den.Mul(den, pow)
	}

	half := new(big.Int).Div(den, big.NewInt(2))
	num.Add(num, half)
	out := new(big.Int).Quo(num, den)
	if !out.IsInt64() {
		return 0, fmt.Errorf("fx conversion overflow")
	}
	res := out.Int64()
	if neg {
		res = -res
	}
	return res, nil
}

func toFxRateApplied(rr resolvedRate, fromCur, toCur string) FxRateApplied {
	scale := rr.Rate.Scale
	if scale <= 0 {
		scale = fxrates.DefaultScale
	}
	b := strings.ToUpper(strings.TrimSpace(rr.Rate.BaseCurrency))
	q := strings.ToUpper(strings.TrimSpace(rr.Rate.QuoteCurrency))
	if b == "" {
		b = fxrates.NormalizeCurrency(fromCur)
	}
	if q == "" {
		q = fxrates.NormalizeCurrency(toCur)
	}
	return FxRateApplied{
		RateID:        rr.Rate.RateID,
		BaseCurrency:  b,
		QuoteCurrency: q,
		RateScaled:    rr.Rate.RateScaled,
		Scale:         scale,
		Inverse:       rr.Inverse,
		EffectiveAt:   rr.Rate.EffectiveAt.UTC(),
		Source:        rr.Rate.Source,
	}
}
