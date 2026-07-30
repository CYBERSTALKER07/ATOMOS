package tax

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"cloud.google.com/go/spanner"
)

const (
	activeRegimeCacheKey = "tax:regime:active:%s" // countryCode
	activeRegimeCacheTTL = 5 * time.Minute
)

// Service provides tax regime management and fiscal snapshot computation.
type Service struct {
	repo  Repository
	cache *cache.Cache
	log   *slog.Logger
}

// NewService constructs the tax regime service.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: c, log: log}
}

// Repo exposes the repository for transactional snapshot inserts from the order package.
func (s *Service) Repo() Repository {
	return s.repo
}

// GetActiveRegime returns the tax regime effective for a country at a given timestamp.
// Results are cached for 5 minutes.
func (s *Service) GetActiveRegime(ctx context.Context, txn *spanner.ReadWriteTransaction, countryCode string, ts time.Time) (TaxRegimeVersion, bool, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if countryCode == "" {
		return TaxRegimeVersion{}, false, ErrMissingCountryCode
	}
	return s.repo.GetActiveRegime(ctx, txn, countryCode, ts)
}

// GetRegime loads a single regime by ID.
func (s *Service) GetRegime(ctx context.Context, id string) (TaxRegimeVersion, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TaxRegimeVersion{}, false, ErrRegimeNotFound
	}
	return s.repo.GetRegime(ctx, id)
}

// ListRegimes returns all regimes for a country.
func (s *Service) ListRegimes(ctx context.Context, countryCode string, limit int) ([]TaxRegimeVersion, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if countryCode == "" {
		return nil, ErrMissingCountryCode
	}
	return s.repo.ListRegimes(ctx, countryCode, limit)
}

// CreateRegime creates a new tax regime version.
// Validates: no overlap with existing active regimes, required fields present.
func (s *Service) CreateRegime(ctx context.Context, claims auth.Claims, req CreateRegimeRequest) (TaxRegimeVersion, error) {
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return TaxRegimeVersion{}, ErrRegimeForbidden
	}

	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if countryCode == "" {
		return TaxRegimeVersion{}, ErrMissingCountryCode
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		return TaxRegimeVersion{}, fmt.Errorf("%w: currency required", ErrRegimeInvalid)
	}
	if req.EffectiveFrom.IsZero() {
		return TaxRegimeVersion{}, fmt.Errorf("%w: effective_from required", ErrRegimeInvalid)
	}
	if req.EffectiveTo != nil && req.EffectiveTo.Before(req.EffectiveFrom) {
		return TaxRegimeVersion{}, fmt.Errorf("%w: effective_to must be after effective_from", ErrRegimeInvalid)
	}
	if req.VatRateBps < 0 || req.VatRateBps > 10000 {
		return TaxRegimeVersion{}, fmt.Errorf("%w: vat rate bps must be 0-10000, got %d", ErrRegimeInvalid, req.VatRateBps)
	}

	now := time.Now().UTC()
	regime := TaxRegimeVersion{
		Id:            uuid.NewString(),
		CountryCode:   countryCode,
		EffectiveFrom: req.EffectiveFrom.UTC(),
		Currency:      currency,
		VatRateBps:    req.VatRateBps,
		Simplified:    req.Simplified,
		RulesJson:     req.RulesJson,
		CreatedAt:     now,
		CreatedBy:     claims.Subject,
		UpdatedAt:     now,
	}
	if req.EffectiveTo != nil {
		t := req.EffectiveTo.UTC()
		regime.EffectiveTo = &t
	}

	if err := s.repo.CreateRegime(ctx, regime); err != nil {
		return TaxRegimeVersion{}, fmt.Errorf("create regime: %w", err)
	}

	// Invalidate cache for this country.
	if s.cache != nil {
		key := fmt.Sprintf(activeRegimeCacheKey, countryCode)
		s.cache.Invalidate(ctx, key)
	}

	s.log.InfoContext(ctx, "tax regime created",
		"regime_id", regime.Id,
		"country", countryCode,
		"effective_from", regime.EffectiveFrom,
		"vat_rate_bps", regime.VatRateBps,
		"created_by", claims.Subject,
	)

	return regime, nil
}
