package fxrates

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// SeedOptions configures bootstrap FX rates.
type SeedOptions struct {
	OperatingCurrency string // stored ISO; empty reads shipped pack (never invents UZS)
	USDToUZSScaled    int64  // optional; 0 = skip USD/UZS pair
	EffectiveAt       time.Time
	Log               *slog.Logger
}

// SeedBootstrapRates upserts identity rate for the operating currency and optional USD/UZS.
func SeedBootstrapRates(ctx context.Context, repo Repository, opts SeedOptions) error {
	if repo == nil {
		return fmt.Errorf("fxrates: nil repository")
	}
	op := seedOperatingCurrency(ctx, opts.OperatingCurrency)
	at := opts.EffectiveAt
	if at.IsZero() {
		at = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	if op != "" {
		identity := IdentityRate(op, "SEED", at)
		identity.RateID = deterministicRateID(op, op, "SEED")
		if err := repo.Upsert(ctx, identity); err != nil {
			return fmt.Errorf("seed identity %s/%s: %w", op, op, err)
		}
		log.Info("fx rate seeded", "base", op, "quote", op, "source", "SEED")
	} else {
		log.Info("fx identity seed skipped", "reason", "empty_operating_currency")
	}

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

// seedOperatingCurrency is empty-currency law for FX identity seed: stored ISO,
// else the shipped pack. Planned/unknown packs stay empty — never invent UZS.
func seedOperatingCurrency(ctx context.Context, stored string) string {
	c, err := auth.CoalesceCurrency(ctx, "", stored)
	if err != nil {
		return NormalizeCurrency(stored)
	}
	return c
}
