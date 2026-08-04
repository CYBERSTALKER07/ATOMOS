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
