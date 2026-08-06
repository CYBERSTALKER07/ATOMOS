// Package fxrates provides integer-scaled FX conversion (theatre #13 Wave 1).
// Missing rates fail closed — never silent 1:1 across currencies.
package fxrates

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
)

const DefaultScale int64 = 100_000_000 // 1e8

var (
	ErrRateMissing      = errors.New("fx_rate_missing")
	ErrInvalidAmount    = errors.New("fx_invalid_amount")
	ErrInvalidCurrency  = errors.New("fx_invalid_currency")
	ErrInvalidRate      = errors.New("fx_invalid_rate")
	ErrCurrencyMismatch = errors.New("currency_mismatch")
)

// Rate is one FX quote: QuoteCurrency per 1 BaseCurrency, stored as RateScaled/Scale.
type Rate struct {
	RateID        string
	BaseCurrency  string
	QuoteCurrency string
	RateScaled    int64
	Scale         int64
	EffectiveAt   time.Time
	Source        string
	CreatedAt     time.Time
}

// Repository persists FX rates.
type Repository interface {
	Upsert(ctx context.Context, r Rate) error
	GetAsOf(ctx context.Context, base, quote string, at time.Time) (Rate, bool, error)
	ListLatest(ctx context.Context, limit int) ([]Rate, error)
}

// Service converts minor units using stored rates.
type Service struct {
	repo         Repository
	AllowInverse bool
}

// NewService constructs an FX converter. AllowInverse defaults true.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, AllowInverse: true}
}

// NormalizeCurrency uppercases and trims an ISO-4217 code.
func NormalizeCurrency(c string) string {
	return strings.ToUpper(strings.TrimSpace(c))
}

// AssertSameCurrency returns ErrCurrencyMismatch when both are non-empty and differ.
func AssertSameCurrency(orderCurrency, requestCurrency string) error {
	o := NormalizeCurrency(orderCurrency)
	r := NormalizeCurrency(requestCurrency)
	if o == "" || r == "" {
		return nil
	}
	if o != r {
		return ErrCurrencyMismatch
	}
	return nil
}

// ConvertMinor converts amountMinor from→to at time at.
// Same currency is identity. Missing rate returns ErrRateMissing (never silent 1:1).
func (s *Service) ConvertMinor(ctx context.Context, from, to string, amountMinor int64, at time.Time) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("fx_unavailable")
	}
	from = NormalizeCurrency(from)
	to = NormalizeCurrency(to)
	if len(from) != 3 || len(to) != 3 {
		return 0, ErrInvalidCurrency
	}
	if amountMinor < 0 {
		return 0, ErrInvalidAmount
	}
	if from == to {
		return amountMinor, nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	rate, ok, err := s.repo.GetAsOf(ctx, from, to, at)
	if err != nil {
		return 0, err
	}
	if ok {
		return applyRate(amountMinor, rate.RateScaled, rate.Scale, false)
	}
	if s.AllowInverse {
		inv, okInv, err := s.repo.GetAsOf(ctx, to, from, at)
		if err != nil {
			return 0, err
		}
		if okInv {
			return applyRate(amountMinor, inv.RateScaled, inv.Scale, true)
		}
	}
	return 0, ErrRateMissing
}

// applyRate: forward amount*rate/scale; inverse amount*scale/rate. Half away from zero.
func applyRate(amount, rateScaled, scale int64, inverse bool) (int64, error) {
	if rateScaled <= 0 || scale <= 0 {
		return 0, ErrInvalidRate
	}
	a := big.NewInt(amount)
	r := big.NewInt(rateScaled)
	sc := big.NewInt(scale)
	num := new(big.Int)
	den := new(big.Int)
	if inverse {
		num.Mul(a, sc)
		den.Set(r)
	} else {
		num.Mul(a, r)
		den.Set(sc)
	}
	// half away from zero: (num + den/2 * sign) / den
	half := new(big.Int).Div(den, big.NewInt(2))
	if num.Sign() >= 0 {
		num.Add(num, half)
	} else {
		num.Sub(num, half)
	}
	out := new(big.Int).Quo(num, den)
	if !out.IsInt64() {
		return 0, fmt.Errorf("fx_overflow")
	}
	v := out.Int64()
	if v < 0 || v > math.MaxInt64 {
		return 0, fmt.Errorf("fx_overflow")
	}
	return v, nil
}

// IdentityRate builds a 1:1 seed rate for a currency.
func IdentityRate(currency, source string, at time.Time) Rate {
	c := NormalizeCurrency(currency)
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return Rate{
		BaseCurrency:  c,
		QuoteCurrency: c,
		RateScaled:    DefaultScale,
		Scale:         DefaultScale,
		EffectiveAt:   at,
		Source:        source,
	}
}

// ScaledRate builds a quote-per-base rate (rateScaled / DefaultScale).
func ScaledRate(base, quote string, rateScaled int64, source string, at time.Time) Rate {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return Rate{
		BaseCurrency:  NormalizeCurrency(base),
		QuoteCurrency: NormalizeCurrency(quote),
		RateScaled:    rateScaled,
		Scale:         DefaultScale,
		EffectiveAt:   at,
		Source:        source,
	}
}
