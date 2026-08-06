package fxrates

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// SeedOptions configures bootstrap FX rates.
type SeedOptions struct {
	OperatingCurrency string // default UZS
	USDToUZSScaled    int64  // optional; 0 = skip USD/UZS pair
	EffectiveAt       time.Time
	Log               *slog.Logger
}

// SeedBootstrapRates upserts identity rate for the operating currency and optional USD/UZS.
func SeedBootstrapRates(ctx context.Context, repo Repository, opts SeedOptions) error {
	if repo == nil {
		return fmt.Errorf("fxrates: nil repository")
	}
	op := NormalizeCurrency(opts.OperatingCurrency)
	if op == "" {
		op = "UZS"
	}
	at := opts.EffectiveAt
	if at.IsZero() {
		at = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	identity := IdentityRate(op, "SEED", at)
	identity.RateID = deterministicRateID(op, op, "SEED")
	if err := repo.Upsert(ctx, identity); err != nil {
		return fmt.Errorf("seed identity %s/%s: %w", op, op, err)
	}
	log.Info("fx rate seeded", "base", op, "quote", op, "source", "SEED")

	if opts.USDToUZSScaled > 0 {
		usd := ScaledRate("USD", "UZS", opts.USDToUZSScaled, "SEED", at)
		usd.RateID = deterministicRateID("USD", "UZS", "SEED")
		if err := repo.Upsert(ctx, usd); err != nil {
			return fmt.Errorf("seed USD/UZS: %w", err)
		}
		log.Info("fx rate seeded", "base", "USD", "quote", "UZS", "rate_scaled", opts.USDToUZSScaled, "source", "SEED")
	}
	return nil
}

func deterministicRateID(base, quote, source string) string {
	// Stable 36-char-ish id for idempotent bootstrap upserts.
	raw := strings.ToLower(NormalizeCurrency(base) + "-" + NormalizeCurrency(quote) + "-" + strings.ToLower(source))
	const prefix = "fxseed-"
	id := prefix + raw
	if len(id) > 36 {
		id = id[:36]
	}
	return id
}
