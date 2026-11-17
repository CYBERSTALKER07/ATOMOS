// Package seed provides the single-tenant supplier bootstrap used by the local
// setup command, the prod-sim seeders, and the runtime bootstrap.
//
// pegasusX runs as a single-supplier tenant: every seeder and the backend boot
// path resolve the same supplier identity so that seeded topology, demo logins,
// and dev JWTs all scope to one coherent tenant. EnsureSupplier is idempotent
// and returns a deterministic SupplierID (DefaultSupplierID) so re-running any
// seeder never forks the tenant and cmd/mint-dev-jwt tokens resolve to the
// seeded scope.
package seed

import (
	"context"
	"log/slog"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
)

// DefaultSupplierID is the deterministic single-tenant seed supplier identity.
// It is referenced by cmd/mint-dev-jwt and the SSMR smoke artifacts so that a
// dev-minted JWT resolves to the seeded supplier scope without a database read.
const DefaultSupplierID = "sup_61d822c6ab9714ca11f20db9"

// Supplier is the minimal seed identity written to the Suppliers table.
type Supplier struct {
	SupplierID  string
	Name        string
	CountryCode string
	Currency    string
}

// Repository persists the seed supplier row. Implementations perform an
// idempotent upsert (InsertOrUpdate) keyed on SupplierID.
type Repository interface {
	UpsertSupplier(ctx context.Context, s Supplier) error
}

// EnsureSupplier returns the deterministic single-tenant seed supplier and, when
// a repository is provided, idempotently upserts its row. A nil repository is
// tolerated (e.g. in-memory boot with no Spanner client): the supplier identity
// is still returned so downstream wiring keys off a stable SupplierID.
func EnsureSupplier(ctx context.Context, repo Repository, name, country, currency string, logger *slog.Logger) (Supplier, error) {
	s := Supplier{
		SupplierID:  DefaultSupplierID,
		Name:        strings.TrimSpace(name),
		CountryCode: strings.TrimSpace(country),
		Currency:    strings.TrimSpace(currency),
	}

	if repo == nil {
		if logger != nil {
			logger.Warn("seed supplier: no repository provided; skipping upsert", "supplier_id", s.SupplierID)
		}
		return s, nil
	}

	if err := repo.UpsertSupplier(ctx, s); err != nil {
		return Supplier{}, err
	}

	if logger != nil {
		logger.Info("seed supplier ensured", "supplier_id", s.SupplierID, "name", s.Name, "country", s.CountryCode, "currency", s.Currency)
	}
	return s, nil
}

// SeedUzbekistanHolidays idempotently seeds the 2025-2035 Uzbekistan Muftiate
// and national seasonal holiday windows into SeasonalTemplateOverrides.
// When client is nil (memory boot), it is a no-op returning (0, nil).
func SeedUzbekistanHolidays(ctx context.Context, client *spanner.Client, supplierID string, logger *slog.Logger) (int, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		sid = DefaultSupplierID
	}
	n, err := seasonalcore.SeedUzbekistanHolidays(ctx, client, sid)
	if err != nil {
		return 0, err
	}
	if logger != nil {
		logger.Info("seeded Uzbekistan 2025-2035 seasonal holidays", "supplier_id", sid, "count", n)
	}
	return n, nil
}
