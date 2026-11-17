// Package innlookup implements Uzbekistan 9-digit INN (YURIDIK) and 14-digit
// PINFL/JSHSHIR normalization, taxpayer registry verification, 24h Spanner cache,
// rate limiting, duplicate-INN detection, and JoinRequests persistence.
package innlookup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Lookup result statuses persisted in InnRegistryCache and returned to clients.
const (
	StatusFound       = "FOUND"
	StatusNotFound    = "NOT_FOUND"
	StatusUnavailable = "UNAVAILABLE"

	EntityTypeYuridik = "YURIDIK_9D"
	EntityTypePinfl   = "PINFL_14D"

	TenantTypeSupplier = "SUPPLIER"
	TenantTypeRetailer = "RETAILER"

	JoinStatusPending  = "PENDING"
	JoinStatusApproved = "APPROVED"
	JoinStatusRejected = "REJECTED"

	DefaultCacheTTL           = 24 * time.Hour
	DefaultRateLimitPerMinute = 30
	DefaultRateLimitWindow    = time.Minute
)

// Sentinel errors returned by innlookup service and stores.
var (
	ErrInvalidTaxID        = errors.New("invalid_tax_id")
	ErrRegistryUnavailable = errors.New("inn_registry_unavailable")
	ErrTaxIDNotFound       = errors.New("inn_not_found")
	ErrRateLimited         = errors.New("inn_lookup_rate_limited")
	ErrJoinRequestNotFound = errors.New("join_request_not_found")
	ErrJoinRequestDecided  = errors.New("join_request_already_decided")
	ErrCrossCellDenied     = errors.New("cross_cell_tenant_mismatch")
	ErrInvalidRoleSought   = errors.New("invalid_role_sought")
	ErrDuplicateTenant     = errors.New("join_request_pending")
)

// LookupResult is the normalized taxpayer verification response.
type LookupResult struct {
	Status               string          `json:"status"`
	NormalizedINN        string          `json:"normalized_inn"`
	EntityType           string          `json:"entity_type,omitempty"`
	LegalName            string          `json:"legal_name,omitempty"`
	Address              string          `json:"address,omitempty"`
	RegistryState        string          `json:"registry_state,omitempty"`
	RawJSON              json.RawMessage `json:"raw_json,omitempty"`
	Cached               bool            `json:"cached"`
	FetchedAt            time.Time       `json:"fetched_at,omitempty"`
	ExpiresAt            time.Time       `json:"expires_at,omitempty"`
	ExistingTenantID     string          `json:"existing_tenant_id,omitempty"`
	ExistingTenantType   string          `json:"existing_tenant_type,omitempty"`
	ExistingLegalName    string          `json:"existing_legal_name,omitempty"`
	ExistingJoinEligible bool            `json:"existing_join_eligible"`
}

// TenantMatch identifies an existing Supplier or Retailer owning a normalized INN in a HomeCell.
type TenantMatch struct {
	HomeCell      string `json:"home_cell"`
	TenantType    string `json:"tenant_type"`
	TenantID      string `json:"tenant_id"`
	LegalName     string `json:"legal_name"`
	NormalizedINN string `json:"normalized_inn"`
}

// JoinRequest represents a pending/decided team access request against a duplicate INN.
type JoinRequest struct {
	RequestID             string     `json:"request_id"`
	HomeCell              string     `json:"home_cell"`
	TargetTenantType      string     `json:"target_tenant_type"`
	TargetTenantID        string     `json:"target_tenant_id"`
	NormalizedINN         string     `json:"normalized_inn"`
	RequesterPhone        string     `json:"requester_phone"`
	RequesterName         string     `json:"requester_name,omitempty"`
	RequesterEmail        string     `json:"requester_email,omitempty"`
	RoleSought            string     `json:"role_sought"`
	RequesterPasswordHash string     `json:"-"`
	Status                string     `json:"status"`
	DecidedBy             string     `json:"decided_by,omitempty"`
	DecidedAt             *time.Time `json:"decided_at,omitempty"`
	DecisionReason        string     `json:"decision_reason,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

// LookupOptions parameterizes an INN/PINFL verification call.
type LookupOptions struct {
	TaxID      string
	HomeCell   string
	MarketCode string
	TenantType string
	ClientKey  string
}

// LookupAdapter fetches taxpayer status from an external registry source.
// Implementations MUST fail closed (StatusUnavailable / ErrRegistryUnavailable)
// when upstream transport or auth is unavailable rather than fabricating StatusFound.
type LookupAdapter interface {
	Lookup(ctx context.Context, homeCell, normalizedINN string) (LookupResult, error)
}

// CacheStore caches FOUND / NOT_FOUND taxpayer lookups (typically 24h in Spanner).
type CacheStore interface {
	GetCachedINN(ctx context.Context, homeCell, normalizedINN string, now time.Time) (LookupResult, bool, error)
	PutCachedINN(ctx context.Context, homeCell string, res LookupResult) error
}

// TenantDirectory locates existing legal entities owning a normalized INN inside a cell.
type TenantDirectory interface {
	FindTenantByNormalizedINN(ctx context.Context, homeCell, tenantType, normalizedINN string) (TenantMatch, bool, error)
}

// JoinRequestStore manages duplicate-INN JoinRequests and approval membership writes.
type JoinRequestStore interface {
	CreateJoinRequest(ctx context.Context, req JoinRequest, emit func(outbox.TxnBuffer) error) (JoinRequest, error)
	ListJoinRequests(ctx context.Context, targetTenantType, targetTenantID, status string, limit int) ([]JoinRequest, error)
	GetJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID string) (JoinRequest, bool, error)
	DecideJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID, decisionStatus, decidedBy, reason string, decidedAt time.Time, emit func(outbox.TxnBuffer) error) (JoinRequest, error)
}

// Config configures the innlookup Service.
type Config struct {
	Adapter            LookupAdapter
	Cache              CacheStore
	Tenants            TenantDirectory
	JoinRequests       JoinRequestStore
	CacheTTL           time.Duration
	RateLimitPerMinute int
	RateLimitWindow    time.Duration
	Now                func() time.Time
}

// Service coordinates INN normalization, caching, rate limiting, duplicate checks, and join requests.
type Service struct {
	adapter     LookupAdapter
	cache       CacheStore
	tenants     TenantDirectory
	joinStore   JoinRequestStore
	cacheTTL    time.Duration
	rateLimiter *windowRateLimiter
	now         func() time.Time
}

// NewService constructs a Service with safe defaults.
// When Adapter is nil, lookups fail closed (ErrRegistryUnavailable) after cache check.
func NewService(cfg Config) *Service {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	limit := cfg.RateLimitPerMinute
	if limit <= 0 {
		limit = DefaultRateLimitPerMinute
	}
	window := cfg.RateLimitWindow
	if window <= 0 {
		window = DefaultRateLimitWindow
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	mem := NewMemoryStore()
	cache := cfg.Cache
	if cache == nil {
		cache = mem
	}
	joinStore := cfg.JoinRequests
	if joinStore == nil {
		joinStore = mem
	}
	tenants := cfg.Tenants
	if tenants == nil && cfg.Cache == nil && cfg.JoinRequests == nil {
		tenants = mem
	}
	return &Service{
		adapter:     cfg.Adapter,
		cache:       cache,
		tenants:     tenants,
		joinStore:   joinStore,
		cacheTTL:    ttl,
		rateLimiter: newWindowRateLimiter(limit, window),
		now:         nowFn,
	}
}

// NormalizeTaxID validates and normalizes an Uzbekistan 9-digit legal entity INN/STIR
// or 14-digit sole-proprietor/individual PINFL (JSHSHIR).
func NormalizeTaxID(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", ErrInvalidTaxID
	}
	upper := strings.ToUpper(trimmed)
	for _, prefix := range []string{"PINFL", "JSHSHIR", "ПИНФЛ", "STIR", "СТИР", "ИНН", "INN", "TIN", "NO.", "№"} {
		if strings.HasPrefix(upper, prefix) {
			trimmed = strings.TrimSpace(trimmed[len(prefix):])
			break
		}
	}
	var b strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == ':' || r == '.':
			continue
		default:
			return "", "", ErrInvalidTaxID
		}
	}
	digits := b.String()
	if strings.Trim(digits, "0") == "" {
		return "", "", ErrInvalidTaxID
	}
	switch len(digits) {
	case 9:
		return digits, EntityTypeYuridik, nil
	case 14:
		return digits, EntityTypePinfl, nil
	default:
		return "", "", ErrInvalidTaxID
	}
}

// NormalizeRoleSought normalizes an onboarding join-request role onto valid supplier/retailer roles.
func NormalizeRoleSought(raw string) (string, error) {
	role := strings.ToUpper(strings.TrimSpace(raw))
	if role == "" {
		return string(auth.RoleAdmin), nil
	}
	switch auth.Role(role) {
	case auth.RoleAdmin,
		auth.RoleWarehouseAdmin,
		auth.RoleWarehouse,
		auth.RoleFactoryAdmin,
		auth.RoleFactory,
		auth.RoleDriver,
		auth.RolePayload,
		auth.RoleRetailer:
		return role, nil
	default:
		return "", ErrInvalidRoleSought
	}
}

// ResolveHomeCell determines the authoritative cell for INN lookups/checks.
func ResolveHomeCell(homeCell, marketCode string) (string, error) {
	cell := strings.TrimSpace(homeCell)
	mc := strings.ToUpper(strings.TrimSpace(marketCode))
	if mc != "" {
		pack, ok := auth.ResolveShippedMarketPack(mc)
		if !ok {
			return "", fmt.Errorf("invalid_or_unshipped_market: %s", mc)
		}
		if cell != "" && cell != pack.HomeCell {
			return "", ErrCrossCellDenied
		}
		return pack.HomeCell, nil
	}
	if cell == "" {
		return auth.DefaultHomeCell, nil
	}
	return cell, nil
}

// Lookup verifies raw/normalized tax ID against 24h Spanner cache and the upstream registry adapter,
// and detects existing tenants owning that INN in the target HomeCell.
func (s *Service) Lookup(ctx context.Context, opts LookupOptions) (LookupResult, error) {
	if s == nil {
		return LookupResult{Status: StatusUnavailable}, ErrRegistryUnavailable
	}
	normalized, entityType, err := NormalizeTaxID(opts.TaxID)
	if err != nil {
		return LookupResult{}, err
	}
	cell, err := ResolveHomeCell(opts.HomeCell, opts.MarketCode)
	if err != nil {
		return LookupResult{}, err
	}
	now := s.now()
	tenantType := strings.ToUpper(strings.TrimSpace(opts.TenantType))

	attachDuplicate := func(res LookupResult) LookupResult {
		if res.EntityType == "" {
			res.EntityType = entityType
		}
		if s.tenants == nil || tenantType == "" {
			return res
		}
		if match, found, merr := s.tenants.FindTenantByNormalizedINN(ctx, cell, tenantType, normalized); merr == nil && found {
			res.ExistingTenantID = match.TenantID
			res.ExistingTenantType = match.TenantType
			res.ExistingLegalName = match.LegalName
			res.ExistingJoinEligible = true
		}
		return res
	}

	if s.cache != nil {
		if cached, hit, cerr := s.cache.GetCachedINN(ctx, cell, normalized, now); cerr == nil && hit {
			cached.Cached = true
			return attachDuplicate(cached), nil
		}
	}

	limitKey := strings.TrimSpace(opts.ClientKey)
	if limitKey == "" {
		limitKey = "global"
	}
	if s.rateLimiter != nil && !s.rateLimiter.Allow(limitKey, now) {
		return LookupResult{}, ErrRateLimited
	}

	if s.adapter == nil {
		return LookupResult{
			Status:        StatusUnavailable,
			NormalizedINN: normalized,
			EntityType:    entityType,
		}, ErrRegistryUnavailable
	}

	res, lerr := s.adapter.Lookup(ctx, cell, normalized)
	if lerr != nil || res.Status == "" || res.Status == StatusUnavailable {
		return LookupResult{
			Status:        StatusUnavailable,
			NormalizedINN: normalized,
			EntityType:    entityType,
		}, ErrRegistryUnavailable
	}
	res.NormalizedINN = normalized
	if res.EntityType == "" {
		res.EntityType = entityType
	}
	res.Cached = false
	if res.FetchedAt.IsZero() {
		res.FetchedAt = now
	}
	if res.ExpiresAt.IsZero() {
		res.ExpiresAt = res.FetchedAt.Add(s.cacheTTL)
	}
	if s.cache != nil {
		_ = s.cache.PutCachedINN(ctx, cell, res)
	}
	return attachDuplicate(res), nil
}

// FindTenantByNormalizedINN delegates cell-scoped duplicate-INN detection.
func (s *Service) FindTenantByNormalizedINN(ctx context.Context, homeCell, tenantType, normalizedINN string) (TenantMatch, bool, error) {
	if s == nil || s.tenants == nil {
		return TenantMatch{}, false, nil
	}
	return s.tenants.FindTenantByNormalizedINN(ctx, homeCell, tenantType, normalizedINN)
}

// CreateJoinRequest persists a new PENDING JoinRequest with optional transactional outbox emission.
func (s *Service) CreateJoinRequest(ctx context.Context, req JoinRequest, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	if s == nil || s.joinStore == nil {
		return JoinRequest{}, ErrRegistryUnavailable
	}
	return s.joinStore.CreateJoinRequest(ctx, req, emit)
}

// ListJoinRequests lists join requests for a supplier or retailer tenant.
func (s *Service) ListJoinRequests(ctx context.Context, targetTenantType, targetTenantID, status string, limit int) ([]JoinRequest, error) {
	if s == nil || s.joinStore == nil {
		return nil, ErrRegistryUnavailable
	}
	return s.joinStore.ListJoinRequests(ctx, targetTenantType, targetTenantID, status, limit)
}

// GetJoinRequest fetches a single join request within tenant scope.
func (s *Service) GetJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID string) (JoinRequest, bool, error) {
	if s == nil || s.joinStore == nil {
		return JoinRequest{}, false, ErrRegistryUnavailable
	}
	return s.joinStore.GetJoinRequest(ctx, targetTenantType, targetTenantID, requestID)
}

// DecideJoinRequest approves or rejects a join request atomically.
func (s *Service) DecideJoinRequest(ctx context.Context, targetTenantType, targetTenantID, requestID, decisionStatus, decidedBy, reason string, emit func(outbox.TxnBuffer) error) (JoinRequest, error) {
	if s == nil || s.joinStore == nil {
		return JoinRequest{}, ErrRegistryUnavailable
	}
	return s.joinStore.DecideJoinRequest(ctx, targetTenantType, targetTenantID, requestID, decisionStatus, decidedBy, reason, s.now(), emit)
}

type lookupHTTPRequest struct {
	TaxID      string `json:"tax_id"`
	INN        string `json:"inn"`
	HomeCell   string `json:"home_cell"`
	MarketCode string `json:"market_code"`
	TenantType string `json:"tenant_type"`
}

// HandleLookup serves POST /v1/platform/inn/lookup and GET /v1/onboarding/tax-id/{inn}.
func (s *Service) HandleLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		web.JSONError(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	var opts LookupOptions
	if r.Method == http.MethodGet {
		inn := strings.TrimSpace(chi.URLParam(r, "inn"))
		if inn == "" {
			inn = strings.TrimSpace(r.URL.Query().Get("tax_id"))
		}
		if inn == "" {
			inn = strings.TrimSpace(r.URL.Query().Get("inn"))
		}
		opts = LookupOptions{
			TaxID:      inn,
			HomeCell:   r.URL.Query().Get("home_cell"),
			MarketCode: r.URL.Query().Get("market_code"),
			TenantType: r.URL.Query().Get("tenant_type"),
		}
	} else {
		body, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		if err != nil {
			web.JSONError(w, "read_body_failed", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		var req lookupHTTPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			web.JSONError(w, "invalid_json", http.StatusBadRequest)
			return
		}
		taxID := strings.TrimSpace(req.TaxID)
		if taxID == "" {
			taxID = strings.TrimSpace(req.INN)
		}
		opts = LookupOptions{
			TaxID:      taxID,
			HomeCell:   req.HomeCell,
			MarketCode: req.MarketCode,
			TenantType: req.TenantType,
		}
	}

	if claims, ok := auth.FromContext(r.Context()); ok && strings.TrimSpace(claims.HomeCell) != "" {
		if strings.TrimSpace(opts.HomeCell) != "" && strings.TrimSpace(opts.HomeCell) != claims.HomeCell {
			web.JSONError(w, ErrCrossCellDenied.Error(), http.StatusForbidden)
			return
		}
		opts.HomeCell = claims.HomeCell
	}
	opts.ClientKey = clientRateLimitKey(r)

	res, err := s.Lookup(r.Context(), opts)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTaxID):
			web.JSONError(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrCrossCellDenied):
			web.JSONError(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, ErrRateLimited):
			web.JSONError(w, err.Error(), http.StatusTooManyRequests)
		case errors.Is(err, ErrRegistryUnavailable):
			web.JSONResponse(w, http.StatusServiceUnavailable, map[string]any{
				"error":          err.Error(),
				"status":         StatusUnavailable,
				"normalized_inn": res.NormalizedINN,
				"entity_type":    res.EntityType,
			})
		default:
			web.JSONError(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	web.JSONResponse(w, http.StatusOK, res)
}

func clientRateLimitKey(r *http.Request) string {
	if claims, ok := auth.FromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
		return "sub:" + strings.TrimSpace(claims.Subject)
	}
	if ip := auth.ResolveTrustedClientIP(r); ip != "" {
		return "ip:" + ip
	}
	return "global"
}

type rateWindowEntry struct {
	count   int
	resetAt time.Time
}

type windowRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateWindowEntry
}

func newWindowRateLimiter(limit int, window time.Duration) *windowRateLimiter {
	return &windowRateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateWindowEntry),
	}
}

func (l *windowRateLimiter) Allow(key string, now time.Time) bool {
	if l == nil || l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) > 2048 {
		for k, e := range l.entries {
			if !now.Before(e.resetAt) {
				delete(l.entries, k)
			}
		}
	}
	entry, ok := l.entries[key]
	if !ok || !now.Before(entry.resetAt) {
		l.entries[key] = rateWindowEntry{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}
