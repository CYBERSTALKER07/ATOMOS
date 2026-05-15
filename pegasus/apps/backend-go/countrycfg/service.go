package countrycfg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// CountryConfig mirrors the CountryConfigs Spanner table.
type CountryConfig struct {
	CountryCode                 string   `json:"country_code"`
	CountryName                 string   `json:"country_name"`
	Timezone                    string   `json:"timezone"`
	CurrencyCode                string   `json:"currency_code"`
	CurrencyDecimalPlaces       int64    `json:"currency_decimal_places"`
	DistanceUnit                string   `json:"distance_unit"`
	DefaultVUConversion         float64  `json:"default_vu_conversion"`
	MapsProvider                string   `json:"maps_provider"`
	LLMProvider                 string   `json:"llm_provider"`
	PaymentGateways             []string `json:"payment_gateways"`
	SMSProvider                 string   `json:"sms_provider"`
	NotificationFallbackOrder   []string `json:"notification_fallback_order"`
	LegalRetentionDays          int64    `json:"legal_retention_days"`
	GridSystem                  string   `json:"grid_system"`
	BreachRadiusMeters          float64  `json:"breach_radius_meters"`
	ShopClosedGraceMinutes      int64    `json:"shop_closed_grace_minutes"`
	ShopClosedEscalationMinutes int64    `json:"shop_closed_escalation_minutes"`
	OfflineModeDurationMinutes  int64    `json:"offline_mode_duration_minutes"`
	CashCustodyAlertHours       int64    `json:"cash_custody_alert_hours"`
}

// SupplierOverride mirrors the SupplierCountryOverrides Spanner table.
// All fields are pointers — nil means "use country default".
type SupplierOverride struct {
	SupplierId                  string   `json:"supplier_id"`
	CountryCode                 string   `json:"country_code"`
	BreachRadiusMeters          *float64 `json:"breach_radius_meters"`
	ShopClosedGraceMinutes      *int64   `json:"shop_closed_grace_minutes"`
	ShopClosedEscalationMinutes *int64   `json:"shop_closed_escalation_minutes"`
	OfflineModeDurationMinutes  *int64   `json:"offline_mode_duration_minutes"`
	CashCustodyAlertHours       *int64   `json:"cash_custody_alert_hours"`
	PaymentGateways             []string `json:"payment_gateways"`
	NotificationFallbackOrder   []string `json:"notification_fallback_order"`
	SMSProvider                 *string  `json:"sms_provider"`
	MapsProvider                *string  `json:"maps_provider"`
	LLMProvider                 *string  `json:"llm_provider"`
	Reason                      string   `json:"reason,omitempty"`
	UpdatedBy                   string   `json:"updated_by,omitempty"`
	UpdatedByType               string   `json:"updated_by_type,omitempty"`
}

const (
	paymentGatewaySourceCountryDefault   = "COUNTRY_DEFAULT"
	paymentGatewaySourceRegionalDefault  = "REGIONAL_DEFAULT"
	paymentGatewaySourceSupplierOverride = "SUPPLIER_OVERRIDE"
	supplierOverrideReasonDefault        = "SUPPLIER_SELF_SERVICE_OVERRIDE"
	supplierOverrideDeleteReason         = "SUPPLIER_REVERT_TO_POLICY"
)

var paymentGatewayProviderOrder = []string{"GLOBAL_PAY", "ADYEN", "AIRWALLEX", "CASH"}

// SupplierPaymentGatewayPolicy describes the region/country policy and runtime
// gateway readiness that apply to a supplier checkout path.
type SupplierPaymentGatewayPolicy struct {
	SupplierId         string   `json:"supplier_id"`
	CountryCode        string   `json:"country_code"`
	RegionId           string   `json:"region_id,omitempty"`
	Source             string   `json:"source"`
	AllowedGateways    []string `json:"allowed_gateways,omitempty"`
	ConfiguredGateways []string `json:"configured_gateways,omitempty"`
	RequestedGateways  []string `json:"requested_gateways,omitempty"`
	EffectiveGateways  []string `json:"effective_gateways,omitempty"`
	ValidationError    string   `json:"validation_error,omitempty"`
}

// PaymentGatewayPolicyError reports a supplier policy/credential mismatch.
type PaymentGatewayPolicyError struct {
	Message string
	Policy  *SupplierPaymentGatewayPolicy
}

func (e *PaymentGatewayPolicyError) Error() string {
	if e == nil {
		return "payment gateway policy error"
	}
	return e.Message
}

type cacheEntry struct {
	config    *CountryConfig
	expiresAt time.Time
}

type overrideCacheEntry struct {
	override  *SupplierOverride
	expiresAt time.Time
}

// Service provides config-driven country parameters with 5min in-memory cache.
type Service struct {
	Spanner       *spanner.Client
	cache         sync.Map // string → *cacheEntry
	overrideCache sync.Map // "supplierID:countryCode" → *overrideCacheEntry
	cacheTTL      time.Duration
	invalidator   *cache.Cache
	hookOnce      sync.Once
}

// NewService creates a config service with default 5min cache TTL.
func NewService(client *spanner.Client) *Service {
	return &Service{
		Spanner:  client,
		cacheTTL: 5 * time.Minute,
	}
}

func normalizePaymentGateway(gateway string) string {
	switch strings.ToUpper(strings.TrimSpace(gateway)) {
	case "GLOBAL_PAY":
		return "GLOBAL_PAY"
	case "ADYEN":
		return "ADYEN"
	case "AIRWALLEX":
		return "AIRWALLEX"
	case "CASH":
		return "CASH"
	default:
		return ""
	}
}

func defaultPolicyGatewaysForCountry(countryCode string) []string {
	normalizedCountry := strings.ToUpper(strings.TrimSpace(countryCode))
	if normalizedCountry == "" || normalizedCountry == "UZ" {
		return []string{"GLOBAL_PAY", "CASH"}
	}

	switch normalizePaymentGateway(ResolveDefaultGatewayForCountry(normalizedCountry)) {
	case "ADYEN":
		return []string{"ADYEN", "CASH"}
	case "AIRWALLEX":
		return []string{"AIRWALLEX", "CASH"}
	default:
		return []string{"GLOBAL_PAY", "CASH"}
	}
}

func normalizePaymentGateways(gateways []string) []string {
	if len(gateways) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(gateways))
	normalized := make([]string, 0, len(gateways))
	for _, gateway := range gateways {
		normalizedGateway := normalizePaymentGateway(gateway)
		if normalizedGateway == "" {
			continue
		}
		if _, ok := seen[normalizedGateway]; ok {
			continue
		}
		seen[normalizedGateway] = struct{}{}
		normalized = append(normalized, normalizedGateway)
	}

	return normalized
}

func filterPaymentGatewaysByActive(gateways, activeGateways []string) []string {
	normalizedGateways := normalizePaymentGateways(gateways)
	if len(normalizedGateways) == 0 {
		return nil
	}

	activeSet := make(map[string]struct{}, len(activeGateways))
	for _, gateway := range activeGateways {
		normalizedGateway := normalizePaymentGateway(gateway)
		if normalizedGateway == "" {
			continue
		}
		activeSet[normalizedGateway] = struct{}{}
	}

	filtered := make([]string, 0, len(normalizedGateways))
	for _, gateway := range normalizedGateways {
		if gateway == "CASH" {
			filtered = append(filtered, gateway)
			continue
		}
		if _, ok := activeSet[gateway]; ok {
			filtered = append(filtered, gateway)
		}
	}

	return filtered
}

func isPaymentGatewaySubset(allowedGateways, requestedGateways []string) bool {
	if len(requestedGateways) == 0 {
		return true
	}

	allowedSet := make(map[string]struct{}, len(allowedGateways))
	for _, gateway := range normalizePaymentGateways(allowedGateways) {
		allowedSet[gateway] = struct{}{}
	}

	for _, gateway := range normalizePaymentGateways(requestedGateways) {
		if _, ok := allowedSet[gateway]; !ok {
			return false
		}
	}

	return true
}

func normalizeSupplierOverrideActorType(actorRole string) string {
	switch strings.ToUpper(strings.TrimSpace(actorRole)) {
	case "ADMIN", "SUPPLIER":
		return "SUPPLIER"
	case "INTERNAL":
		return "INTERNAL"
	case "SYSTEM":
		return "SYSTEM"
	default:
		return strings.ToUpper(strings.TrimSpace(actorRole))
	}
}

// AttachInvalidation wires the in-process country-config caches into the shared
// Redis invalidation spine so edits evict stale entries across pods.
func (s *Service) AttachInvalidation(c *cache.Cache) {
	if s == nil || c == nil {
		return
	}
	s.invalidator = c
	s.hookOnce.Do(func() {
		c.OnInvalidate(s.handleInvalidation)
	})
}

// defaultUZ returns the hardcoded UZ fallback — used when Spanner has no row.
func defaultUZ() *CountryConfig {
	return &CountryConfig{
		CountryCode:                 "UZ",
		CountryName:                 "Uzbekistan",
		Timezone:                    "Asia/Tashkent",
		CurrencyCode:                "UZS",
		CurrencyDecimalPlaces:       0,
		DistanceUnit:                "km",
		DefaultVUConversion:         1.0,
		MapsProvider:                "GOOGLE",
		LLMProvider:                 "GEMINI",
		PaymentGateways:             []string{"GLOBAL_PAY", "CASH"},
		SMSProvider:                 "ESKIZ",
		NotificationFallbackOrder:   []string{"FCM", "TELEGRAM"},
		LegalRetentionDays:          365,
		GridSystem:                  "H3",
		BreachRadiusMeters:          100.0,
		ShopClosedGraceMinutes:      5,
		ShopClosedEscalationMinutes: 3,
		OfflineModeDurationMinutes:  30,
		CashCustodyAlertHours:       4,
	}
}

func cloneCountryConfig(cfg *CountryConfig) *CountryConfig {
	if cfg == nil {
		return nil
	}

	cloned := *cfg
	cloned.PaymentGateways = append([]string(nil), cfg.PaymentGateways...)
	cloned.NotificationFallbackOrder = append([]string(nil), cfg.NotificationFallbackOrder...)
	return &cloned
}

func mergeCountryConfigForUpsert(existing *CountryConfig, incoming *CountryConfig) *CountryConfig {
	base := cloneCountryConfig(defaultUZ())
	if existing != nil {
		base = cloneCountryConfig(existing)
	}
	if incoming == nil {
		return base
	}

	merged := cloneCountryConfig(base)
	if incoming.CountryCode != "" {
		merged.CountryCode = incoming.CountryCode
	}
	if incoming.CountryName != "" {
		merged.CountryName = incoming.CountryName
	}
	if incoming.Timezone != "" {
		merged.Timezone = incoming.Timezone
	}
	if incoming.CurrencyCode != "" {
		merged.CurrencyCode = incoming.CurrencyCode
	}
	if incoming.CurrencyDecimalPlaces != 0 {
		merged.CurrencyDecimalPlaces = incoming.CurrencyDecimalPlaces
	}
	if incoming.DistanceUnit != "" {
		merged.DistanceUnit = incoming.DistanceUnit
	}
	if incoming.DefaultVUConversion != 0 {
		merged.DefaultVUConversion = incoming.DefaultVUConversion
	}
	if incoming.MapsProvider != "" {
		merged.MapsProvider = incoming.MapsProvider
	}
	if incoming.LLMProvider != "" {
		merged.LLMProvider = incoming.LLMProvider
	}
	if len(incoming.PaymentGateways) > 0 {
		merged.PaymentGateways = append([]string(nil), incoming.PaymentGateways...)
	}
	if incoming.SMSProvider != "" {
		merged.SMSProvider = incoming.SMSProvider
	}
	if len(incoming.NotificationFallbackOrder) > 0 {
		merged.NotificationFallbackOrder = append([]string(nil), incoming.NotificationFallbackOrder...)
	}
	if incoming.LegalRetentionDays != 0 {
		merged.LegalRetentionDays = incoming.LegalRetentionDays
	}
	if incoming.GridSystem != "" {
		merged.GridSystem = incoming.GridSystem
	}
	if incoming.BreachRadiusMeters != 0 {
		merged.BreachRadiusMeters = incoming.BreachRadiusMeters
	}
	if incoming.ShopClosedGraceMinutes != 0 {
		merged.ShopClosedGraceMinutes = incoming.ShopClosedGraceMinutes
	}
	if incoming.ShopClosedEscalationMinutes != 0 {
		merged.ShopClosedEscalationMinutes = incoming.ShopClosedEscalationMinutes
	}
	if incoming.OfflineModeDurationMinutes != 0 {
		merged.OfflineModeDurationMinutes = incoming.OfflineModeDurationMinutes
	}
	if incoming.CashCustodyAlertHours != 0 {
		merged.CashCustodyAlertHours = incoming.CashCustodyAlertHours
	}
	return merged
}

// GetConfig returns the config for a country code, using 5min cache.
// Falls back to UZ defaults if the country is not found.
func (s *Service) GetConfig(ctx context.Context, countryCode string) *CountryConfig {
	if countryCode == "" {
		countryCode = "UZ"
	}

	// Check cache
	if val, ok := s.cache.Load(countryCode); ok {
		entry := val.(*cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.config
		}
		s.cache.Delete(countryCode)
	}

	// Read from Spanner
	cfg := s.readCountryConfig(ctx, countryCode)
	if cfg == nil {
		cfg = defaultUZ()
	}

	s.cache.Store(countryCode, &cacheEntry{
		config:    cfg,
		expiresAt: time.Now().Add(s.cacheTTL),
	})

	return cfg
}

// GetEffectiveConfig returns the merged config: SupplierOverride ?? CountryConfig ?? UZ fallback.
func (s *Service) GetEffectiveConfig(ctx context.Context, supplierID, countryCode string) *CountryConfig {
	base := s.GetConfig(ctx, countryCode)
	if supplierID == "" {
		return base
	}

	override := s.getSupplierOverride(ctx, supplierID, countryCode)
	if override == nil {
		return base
	}

	return mergeOverride(base, override)
}

// ResolveSupplierCountryCode returns the supplier's country code with a UZ fallback.
func (s *Service) ResolveSupplierCountryCode(ctx context.Context, supplierID string) string {
	if strings.TrimSpace(supplierID) == "" {
		return "UZ"
	}

	row, err := s.Spanner.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"CountryCode"})
	if err != nil {
		return "UZ"
	}

	var countryCode spanner.NullString
	if err := row.Columns(&countryCode); err != nil || !countryCode.Valid {
		return "UZ"
	}

	resolved := strings.ToUpper(strings.TrimSpace(countryCode.StringVal))
	if resolved == "" {
		return "UZ"
	}

	return resolved
}

// ResolveSupplierCheckoutGateways returns the effective supplier gateway list
// after region/country policy, override exceptions, and runtime readiness are applied.
func (s *Service) ResolveSupplierCheckoutGateways(ctx context.Context, supplierID string) ([]string, error) {
	policy, err := s.ResolveSupplierPaymentGatewayPolicy(ctx, supplierID, "")
	if err != nil {
		return nil, err
	}
	return append([]string(nil), policy.EffectiveGateways...), nil
}

// ResolveSupplierPaymentGatewayPolicy resolves the effective supplier gateway
// policy and the currently credential-ready gateways for checkout.
func (s *Service) ResolveSupplierPaymentGatewayPolicy(ctx context.Context, supplierID, countryCode string) (*SupplierPaymentGatewayPolicy, error) {
	return s.resolveSupplierPaymentGatewayPolicy(ctx, supplierID, countryCode, nil)
}

// ValidateSupplierOverride checks whether the requested supplier override stays
// within region/country policy and runtime credential readiness.
func (s *Service) ValidateSupplierOverride(ctx context.Context, o *SupplierOverride) (*SupplierPaymentGatewayPolicy, error) {
	if o == nil {
		return nil, &PaymentGatewayPolicyError{Message: "supplier override is required"}
	}
	return s.resolveSupplierPaymentGatewayPolicy(ctx, o.SupplierId, o.CountryCode, o)
}

func (s *Service) resolveSupplierPaymentGatewayPolicy(ctx context.Context, supplierID, countryCode string, override *SupplierOverride) (*SupplierPaymentGatewayPolicy, error) {
	trimmedSupplierID := strings.TrimSpace(supplierID)
	if trimmedSupplierID == "" {
		return nil, &PaymentGatewayPolicyError{Message: "supplier_id is required for payment gateway resolution"}
	}

	resolvedCountryCode := strings.ToUpper(strings.TrimSpace(countryCode))
	if resolvedCountryCode == "" {
		if s != nil && s.Spanner != nil {
			resolvedCountryCode = s.ResolveSupplierCountryCode(ctx, trimmedSupplierID)
		} else {
			resolvedCountryCode = "UZ"
		}
	}

	allowedGateways := defaultPolicyGatewaysForCountry(resolvedCountryCode)
	policy := &SupplierPaymentGatewayPolicy{
		SupplierId:  trimmedSupplierID,
		CountryCode: resolvedCountryCode,
		Source:      paymentGatewaySourceCountryDefault,
	}

	if s != nil {
		if s.Spanner != nil {
			if cfg := s.GetConfig(ctx, resolvedCountryCode); cfg != nil {
				if strings.EqualFold(strings.TrimSpace(cfg.CountryCode), resolvedCountryCode) && len(cfg.PaymentGateways) > 0 {
					allowedGateways = cfg.PaymentGateways
				}
			}
		}
		if s.Spanner != nil {
			region, err := s.ResolveSupplierRegion(ctx, trimmedSupplierID)
			if err != nil {
				return nil, fmt.Errorf("resolve supplier %s region: %w", trimmedSupplierID, err)
			}
			if region != nil {
				policy.RegionId = region.RegionID
				regionalConfig, err := s.GetRegionalConfig(ctx, region.RegionID)
				if err != nil {
					return nil, fmt.Errorf("read regional config %s: %w", region.RegionID, err)
				}
				if regionalConfig != nil && len(regionalConfig.PaymentGateways) > 0 {
					allowedGateways = regionalConfig.PaymentGateways
					policy.Source = paymentGatewaySourceRegionalDefault
				}
			}
		}
	}

	allowedGateways = normalizePaymentGateways(allowedGateways)
	policy.AllowedGateways = append([]string(nil), allowedGateways...)
	if len(allowedGateways) == 0 {
		policy.ValidationError = fmt.Sprintf("supplier %s has no policy-allowed payment gateways", trimmedSupplierID)
		return policy, &PaymentGatewayPolicyError{Message: policy.ValidationError, Policy: policy}
	}

	var activeGateways []string
	if s != nil && s.Spanner != nil {
		var err error
		activeGateways, err = s.listActiveGatewayNames(ctx, trimmedSupplierID)
		if err != nil {
			return nil, fmt.Errorf("list active gateway names for supplier %s: %w", trimmedSupplierID, err)
		}
	}

	configuredGateways := append([]string(nil), allowedGateways...)
	if s != nil && s.Spanner != nil {
		configuredGateways = filterPaymentGatewaysByActive(allowedGateways, activeGateways)
	}
	policy.ConfiguredGateways = append([]string(nil), configuredGateways...)

	requestedOverride := override
	if requestedOverride == nil && s != nil && s.Spanner != nil {
		storedOverride, err := s.GetSupplierOverride(ctx, trimmedSupplierID, resolvedCountryCode)
		if err != nil {
			return nil, fmt.Errorf("read supplier override %s/%s: %w", trimmedSupplierID, resolvedCountryCode, err)
		}
		requestedOverride = storedOverride
	}

	effectiveGateways := append([]string(nil), configuredGateways...)
	if requestedOverride != nil {
		requestedGateways := normalizePaymentGateways(requestedOverride.PaymentGateways)
		if len(requestedGateways) > 0 {
			policy.Source = paymentGatewaySourceSupplierOverride
			policy.RequestedGateways = append([]string(nil), requestedGateways...)
			if !isPaymentGatewaySubset(allowedGateways, requestedGateways) {
				policy.ValidationError = fmt.Sprintf("payment_gateways must stay within the policy-allowed set: %s", strings.Join(allowedGateways, ", "))
				return policy, &PaymentGatewayPolicyError{Message: policy.ValidationError, Policy: policy}
			}

			effectiveGateways = append([]string(nil), requestedGateways...)
			if s != nil && s.Spanner != nil {
				effectiveGateways = filterPaymentGatewaysByActive(requestedGateways, activeGateways)
			}
		}
	}

	if len(effectiveGateways) == 0 {
		if len(policy.RequestedGateways) > 0 {
			policy.ValidationError = "payment_gateways override has no credential-ready gateways"
		} else {
			policy.ValidationError = fmt.Sprintf("supplier %s has no credential-ready payment gateways", trimmedSupplierID)
		}
		return policy, &PaymentGatewayPolicyError{Message: policy.ValidationError, Policy: policy}
	}

	policy.EffectiveGateways = append([]string(nil), effectiveGateways...)
	return policy, nil
}

func (s *Service) listActiveGatewayNames(ctx context.Context, supplierID string) ([]string, error) {
	if s == nil || s.Spanner == nil || strings.TrimSpace(supplierID) == "" {
		return nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT GatewayName
		      FROM SupplierPaymentConfigs
		      WHERE SupplierId = @sid AND IsActive = TRUE`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	activeSet := make(map[string]struct{}, len(paymentGatewayProviderOrder))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var gatewayName string
		if err := row.Columns(&gatewayName); err != nil {
			return nil, err
		}
		normalizedGateway := normalizePaymentGateway(gatewayName)
		if normalizedGateway == "" {
			continue
		}
		activeSet[normalizedGateway] = struct{}{}
	}

	activeGateways := make([]string, 0, len(activeSet))
	for _, gateway := range paymentGatewayProviderOrder {
		if _, ok := activeSet[gateway]; ok {
			activeGateways = append(activeGateways, gateway)
		}
	}

	return activeGateways, nil
}

// InvalidateCache removes cached entries for a country (called on admin updates).
func (s *Service) InvalidateCache(countryCode string) {
	s.cache.Delete(countryCode)
	// Also clear any supplier overrides for this country
	s.overrideCache.Range(func(key, _ interface{}) bool {
		k := key.(string)
		if len(k) > 3 && k[len(k)-2:] == countryCode {
			s.overrideCache.Delete(key)
		}
		return true
	})
}

func (s *Service) invalidateCountryConfig(ctx context.Context, countryCode string) {
	s.InvalidateCache(countryCode)
	if s.invalidator != nil {
		s.invalidator.Invalidate(ctx, cache.CountryConfigCacheKey(countryCode))
	}
}

func (s *Service) invalidateSupplierOverride(ctx context.Context, supplierID, countryCode string) {
	s.overrideCache.Delete(supplierID + ":" + countryCode)
	if s.invalidator != nil {
		s.invalidator.Invalidate(ctx, cache.CountryOverrideCacheKey(supplierID, countryCode))
	}
}

func (s *Service) handleInvalidation(keys []string) {
	for _, key := range keys {
		switch {
		case strings.HasPrefix(key, cache.PrefixCountryConfigCache):
			countryCode := strings.TrimSpace(strings.TrimPrefix(key, cache.PrefixCountryConfigCache))
			if countryCode != "" {
				s.InvalidateCache(countryCode)
			}
		case strings.HasPrefix(key, cache.PrefixCountryOverrideCache):
			parts := strings.Split(strings.TrimPrefix(key, cache.PrefixCountryOverrideCache), ":")
			if len(parts) != 2 {
				continue
			}
			supplierID := strings.TrimSpace(parts[0])
			countryCode := strings.TrimSpace(parts[1])
			if supplierID == "" || countryCode == "" {
				continue
			}
			s.overrideCache.Delete(supplierID + ":" + countryCode)
		}
	}
}

func (s *Service) readCountryConfig(ctx context.Context, countryCode string) *CountryConfig {
	row, err := s.Spanner.Single().ReadRow(ctx, "CountryConfigs",
		spanner.Key{countryCode},
		[]string{
			"CountryCode", "CountryName", "Timezone", "CurrencyCode",
			"CurrencyDecimalPlaces", "DistanceUnit", "DefaultVUConversion",
			"MapsProvider", "LLMProvider", "GlobalPayntGateways", "SMSProvider",
			"NotificationFallbackOrder", "LegalRetentionDays", "GridSystem",
			"BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
			"OfflineModeDurationMinutes", "CashCustodyAlertHours",
		})
	if err != nil {
		log.Printf("[CountryCfg] No config for %s, using defaults: %v", countryCode, err)
		return nil
	}

	cfg := &CountryConfig{}
	var paymentGatewaysJSON, notifOrderJSON, smsProvider spanner.NullString

	if err := row.Columns(
		&cfg.CountryCode, &cfg.CountryName, &cfg.Timezone, &cfg.CurrencyCode,
		&cfg.CurrencyDecimalPlaces, &cfg.DistanceUnit, &cfg.DefaultVUConversion,
		&cfg.MapsProvider, &cfg.LLMProvider, &paymentGatewaysJSON, &smsProvider,
		&notifOrderJSON, &cfg.LegalRetentionDays, &cfg.GridSystem,
		&cfg.BreachRadiusMeters, &cfg.ShopClosedGraceMinutes, &cfg.ShopClosedEscalationMinutes,
		&cfg.OfflineModeDurationMinutes, &cfg.CashCustodyAlertHours,
	); err != nil {
		log.Printf("[CountryCfg] Error scanning row for %s: %v", countryCode, err)
		return nil
	}

	if paymentGatewaysJSON.Valid {
		_ = json.Unmarshal([]byte(paymentGatewaysJSON.StringVal), &cfg.PaymentGateways)
	}
	if notifOrderJSON.Valid {
		_ = json.Unmarshal([]byte(notifOrderJSON.StringVal), &cfg.NotificationFallbackOrder)
	}
	if smsProvider.Valid {
		cfg.SMSProvider = smsProvider.StringVal
	}

	return cfg
}

func (s *Service) getSupplierOverride(ctx context.Context, supplierID, countryCode string) *SupplierOverride {
	cacheKey := supplierID + ":" + countryCode

	if val, ok := s.overrideCache.Load(cacheKey); ok {
		entry := val.(*overrideCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.override
		}
		s.overrideCache.Delete(cacheKey)
	}

	override := s.readSupplierOverride(ctx, supplierID, countryCode)
	s.overrideCache.Store(cacheKey, &overrideCacheEntry{
		override:  override,
		expiresAt: time.Now().Add(s.cacheTTL),
	})

	return override
}

func (s *Service) readSupplierOverride(ctx context.Context, supplierID, countryCode string) *SupplierOverride {
	row, err := s.Spanner.Single().ReadRow(ctx, "SupplierCountryOverrides",
		spanner.Key{supplierID, countryCode},
		[]string{
			"SupplierId", "CountryCode", "BreachRadiusMeters",
			"ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
			"OfflineModeDurationMinutes", "CashCustodyAlertHours",
			"GlobalPayntGateways", "NotificationFallbackOrder",
			"SMSProvider", "MapsProvider", "LLMProvider",
			"OverrideReason", "UpdatedBy", "UpdatedByType",
		})
	if err != nil {
		return nil
	}

	o := &SupplierOverride{}
	var breach spanner.NullFloat64
	var shopGrace, shopEsc, offlineDur, cashAlert spanner.NullInt64
	var payGW, notifOrder, sms, maps, llm, reason, updatedBy, updatedByType spanner.NullString

	if err := row.Columns(
		&o.SupplierId, &o.CountryCode, &breach,
		&shopGrace, &shopEsc, &offlineDur, &cashAlert,
		&payGW, &notifOrder, &sms, &maps, &llm,
		&reason, &updatedBy, &updatedByType,
	); err != nil {
		log.Printf("[CountryCfg] Error scanning supplier override for %s/%s: %v", supplierID, countryCode, err)
		return nil
	}

	if breach.Valid {
		o.BreachRadiusMeters = &breach.Float64
	}
	if shopGrace.Valid {
		o.ShopClosedGraceMinutes = &shopGrace.Int64
	}
	if shopEsc.Valid {
		o.ShopClosedEscalationMinutes = &shopEsc.Int64
	}
	if offlineDur.Valid {
		o.OfflineModeDurationMinutes = &offlineDur.Int64
	}
	if cashAlert.Valid {
		o.CashCustodyAlertHours = &cashAlert.Int64
	}
	if payGW.Valid {
		_ = json.Unmarshal([]byte(payGW.StringVal), &o.PaymentGateways)
	}
	if notifOrder.Valid {
		_ = json.Unmarshal([]byte(notifOrder.StringVal), &o.NotificationFallbackOrder)
	}
	if sms.Valid {
		o.SMSProvider = &sms.StringVal
	}
	if maps.Valid {
		o.MapsProvider = &maps.StringVal
	}
	if llm.Valid {
		o.LLMProvider = &llm.StringVal
	}
	if reason.Valid {
		o.Reason = strings.TrimSpace(reason.StringVal)
	}
	if updatedBy.Valid {
		o.UpdatedBy = strings.TrimSpace(updatedBy.StringVal)
	}
	if updatedByType.Valid {
		o.UpdatedByType = strings.TrimSpace(updatedByType.StringVal)
	}

	return o
}

// GetSupplierOverride returns the supplier's override for the given country, or nil if none set.
func (s *Service) GetSupplierOverride(ctx context.Context, supplierID, countryCode string) (*SupplierOverride, error) {
	o := s.getSupplierOverride(ctx, supplierID, countryCode)
	return o, nil
}

// ListSupplierOverrides returns all country overrides set by the given supplier.
func (s *Service) ListSupplierOverrides(ctx context.Context, supplierID string) ([]*SupplierOverride, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT SupplierId, CountryCode FROM SupplierCountryOverrides WHERE SupplierId = @sid`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var result []*SupplierOverride
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list supplier overrides: %w", err)
		}
		var sid, cc string
		if err := row.Columns(&sid, &cc); err != nil {
			continue
		}
		o := s.readSupplierOverride(ctx, sid, cc)
		if o != nil {
			result = append(result, o)
		}
	}
	return result, nil
}

func (s *Service) readSupplierOverrideForWrite(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, countryCode string) (*SupplierOverride, spanner.NullTime, error) {
	row, err := txn.ReadRow(ctx, "SupplierCountryOverrides",
		spanner.Key{supplierID, countryCode},
		[]string{
			"SupplierId", "CountryCode", "BreachRadiusMeters",
			"ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
			"OfflineModeDurationMinutes", "CashCustodyAlertHours",
			"GlobalPayntGateways", "NotificationFallbackOrder",
			"SMSProvider", "MapsProvider", "LLMProvider",
			"OverrideReason", "UpdatedBy", "UpdatedByType", "CreatedAt",
		})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return nil, spanner.NullTime{}, nil
		}
		return nil, spanner.NullTime{}, err
	}

	o := &SupplierOverride{}
	var breach spanner.NullFloat64
	var shopGrace, shopEsc, offlineDur, cashAlert spanner.NullInt64
	var payGW, notifOrder, sms, maps, llm, reason, updatedBy, updatedByType spanner.NullString
	var createdAt spanner.NullTime

	if err := row.Columns(
		&o.SupplierId, &o.CountryCode, &breach,
		&shopGrace, &shopEsc, &offlineDur, &cashAlert,
		&payGW, &notifOrder, &sms, &maps, &llm,
		&reason, &updatedBy, &updatedByType, &createdAt,
	); err != nil {
		return nil, spanner.NullTime{}, err
	}

	if breach.Valid {
		o.BreachRadiusMeters = &breach.Float64
	}
	if shopGrace.Valid {
		o.ShopClosedGraceMinutes = &shopGrace.Int64
	}
	if shopEsc.Valid {
		o.ShopClosedEscalationMinutes = &shopEsc.Int64
	}
	if offlineDur.Valid {
		o.OfflineModeDurationMinutes = &offlineDur.Int64
	}
	if cashAlert.Valid {
		o.CashCustodyAlertHours = &cashAlert.Int64
	}
	if payGW.Valid {
		_ = json.Unmarshal([]byte(payGW.StringVal), &o.PaymentGateways)
	}
	if notifOrder.Valid {
		_ = json.Unmarshal([]byte(notifOrder.StringVal), &o.NotificationFallbackOrder)
	}
	if sms.Valid {
		o.SMSProvider = &sms.StringVal
	}
	if maps.Valid {
		o.MapsProvider = &maps.StringVal
	}
	if llm.Valid {
		o.LLMProvider = &llm.StringVal
	}
	if reason.Valid {
		o.Reason = strings.TrimSpace(reason.StringVal)
	}
	if updatedBy.Valid {
		o.UpdatedBy = strings.TrimSpace(updatedBy.StringVal)
	}
	if updatedByType.Valid {
		o.UpdatedByType = strings.TrimSpace(updatedByType.StringVal)
	}

	return o, createdAt, nil
}

// UpsertSupplierOverride creates or replaces the supplier's country override row.
func (s *Service) UpsertSupplierOverride(ctx context.Context, o *SupplierOverride, actorID, actorRole string) (*SupplierPaymentGatewayPolicy, error) {
	if o == nil || o.SupplierId == "" || o.CountryCode == "" {
		return nil, fmt.Errorf("supplier_id and country_code are required")
	}

	o.SupplierId = strings.TrimSpace(o.SupplierId)
	o.CountryCode = strings.ToUpper(strings.TrimSpace(o.CountryCode))
	o.Reason = strings.TrimSpace(o.Reason)
	if o.Reason == "" {
		o.Reason = supplierOverrideReasonDefault
	}
	o.UpdatedBy = strings.TrimSpace(actorID)
	o.UpdatedByType = normalizeSupplierOverrideActorType(actorRole)

	policy, err := s.ValidateSupplierOverride(ctx, o)
	if err != nil {
		return policy, err
	}

	var payGWJSON, notifOrderJSON []byte
	if len(o.PaymentGateways) > 0 {
		payGWJSON, _ = json.Marshal(o.PaymentGateways)
	}
	if len(o.NotificationFallbackOrder) > 0 {
		notifOrderJSON, _ = json.Marshal(o.NotificationFallbackOrder)
	}

	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		before, createdAt, err := s.readSupplierOverrideForWrite(ctx, txn, o.SupplierId, o.CountryCode)
		if err != nil {
			return fmt.Errorf("read supplier override before write: %w", err)
		}

		metadata, err := json.Marshal(map[string]interface{}{
			"source":                 "SUPPLIER_SELF_SERVICE",
			"payment_gateway_policy": policy,
			"before":                 before,
			"after":                  o,
		})
		if err != nil {
			return fmt.Errorf("encode supplier override audit metadata: %w", err)
		}

		cols := []string{
			"SupplierId", "CountryCode",
			"BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
			"OfflineModeDurationMinutes", "CashCustodyAlertHours",
			"GlobalPayntGateways", "NotificationFallbackOrder",
			"SMSProvider", "MapsProvider", "LLMProvider",
			"OverrideReason", "UpdatedBy", "UpdatedByType",
			"UpdatedAt",
		}

		toNullFloat64 := func(p *float64) spanner.NullFloat64 {
			if p == nil {
				return spanner.NullFloat64{}
			}
			return spanner.NullFloat64{Float64: *p, Valid: true}
		}
		toNullInt64 := func(p *int64) spanner.NullInt64 {
			if p == nil {
				return spanner.NullInt64{}
			}
			return spanner.NullInt64{Int64: *p, Valid: true}
		}
		toNullString := func(p *string) spanner.NullString {
			if p == nil {
				return spanner.NullString{}
			}
			return spanner.NullString{StringVal: *p, Valid: true}
		}
		toNullJSON := func(b []byte) spanner.NullString {
			if len(b) == 0 {
				return spanner.NullString{}
			}
			return spanner.NullString{StringVal: string(b), Valid: true}
		}

		vals := []interface{}{
			o.SupplierId, o.CountryCode,
			toNullFloat64(o.BreachRadiusMeters),
			toNullInt64(o.ShopClosedGraceMinutes),
			toNullInt64(o.ShopClosedEscalationMinutes),
			toNullInt64(o.OfflineModeDurationMinutes),
			toNullInt64(o.CashCustodyAlertHours),
			toNullJSON(payGWJSON),
			toNullJSON(notifOrderJSON),
			toNullString(o.SMSProvider),
			toNullString(o.MapsProvider),
			toNullString(o.LLMProvider),
			toNullString(&o.Reason),
			toNullString(&o.UpdatedBy),
			toNullString(&o.UpdatedByType),
			spanner.CommitTimestamp,
		}

		// Set CreatedAt only on first insert.
		if !createdAt.Valid {
			cols = append(cols, "CreatedAt")
			vals = append(vals, spanner.CommitTimestamp)
		} else {
			cols = append(cols, "CreatedAt")
			vals = append(vals, createdAt.Time)
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdate("SupplierCountryOverrides", cols, vals),
			spanner.Insert("AuditLog",
				[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
				[]interface{}{
					uuid.NewString(),
					o.UpdatedBy,
					o.UpdatedByType,
					"SUPPLIER_COUNTRY_OVERRIDE_UPSERTED",
					"SUPPLIER_COUNTRY_OVERRIDE",
					fmt.Sprintf("%s:%s", o.SupplierId, o.CountryCode),
					string(metadata),
					spanner.CommitTimestamp,
				},
			),
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return policy, fmt.Errorf("upsert supplier override %s/%s: %w", o.SupplierId, o.CountryCode, err)
	}

	s.invalidateSupplierOverride(ctx, o.SupplierId, o.CountryCode)
	log.Printf("[CountryCfg] Upserted override %s/%s", o.SupplierId, o.CountryCode)
	return policy, nil
}

// DeleteSupplierOverride removes the supplier's override for a country (reverts to platform defaults).
func (s *Service) DeleteSupplierOverride(ctx context.Context, supplierID, countryCode, actorID, actorRole string) error {
	if supplierID == "" || countryCode == "" {
		return fmt.Errorf("supplier_id and country_code are required")
	}

	trimmedSupplierID := strings.TrimSpace(supplierID)
	normalizedCountryCode := strings.ToUpper(strings.TrimSpace(countryCode))
	actor := strings.TrimSpace(actorID)
	actorType := normalizeSupplierOverrideActorType(actorRole)

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		before, _, err := s.readSupplierOverrideForWrite(ctx, txn, trimmedSupplierID, normalizedCountryCode)
		if err != nil {
			return fmt.Errorf("read supplier override before delete: %w", err)
		}

		metadata, err := json.Marshal(map[string]interface{}{
			"source":       "SUPPLIER_SELF_SERVICE",
			"reason":       supplierOverrideDeleteReason,
			"before":       before,
			"after":        nil,
			"country_code": normalizedCountryCode,
		})
		if err != nil {
			return fmt.Errorf("encode supplier override delete audit metadata: %w", err)
		}

		mutations := []*spanner.Mutation{
			spanner.Delete("SupplierCountryOverrides", spanner.Key{trimmedSupplierID, normalizedCountryCode}),
			spanner.Insert("AuditLog",
				[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
				[]interface{}{
					uuid.NewString(),
					actor,
					actorType,
					"SUPPLIER_COUNTRY_OVERRIDE_DELETED",
					"SUPPLIER_COUNTRY_OVERRIDE",
					fmt.Sprintf("%s:%s", trimmedSupplierID, normalizedCountryCode),
					string(metadata),
					spanner.CommitTimestamp,
				},
			),
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("delete supplier override %s/%s: %w", trimmedSupplierID, normalizedCountryCode, err)
	}

	s.invalidateSupplierOverride(ctx, trimmedSupplierID, normalizedCountryCode)
	log.Printf("[CountryCfg] Deleted override %s/%s", trimmedSupplierID, normalizedCountryCode)
	return nil
}

// mergeOverride applies supplier-specific overrides on top of the country config.
func mergeOverride(base *CountryConfig, o *SupplierOverride) *CountryConfig {
	merged := *base // shallow copy
	if o.BreachRadiusMeters != nil {
		merged.BreachRadiusMeters = *o.BreachRadiusMeters
	}
	if o.ShopClosedGraceMinutes != nil {
		merged.ShopClosedGraceMinutes = *o.ShopClosedGraceMinutes
	}
	if o.ShopClosedEscalationMinutes != nil {
		merged.ShopClosedEscalationMinutes = *o.ShopClosedEscalationMinutes
	}
	if o.OfflineModeDurationMinutes != nil {
		merged.OfflineModeDurationMinutes = *o.OfflineModeDurationMinutes
	}
	if o.CashCustodyAlertHours != nil {
		merged.CashCustodyAlertHours = *o.CashCustodyAlertHours
	}
	if len(o.PaymentGateways) > 0 {
		merged.PaymentGateways = o.PaymentGateways
	}
	if len(o.NotificationFallbackOrder) > 0 {
		merged.NotificationFallbackOrder = o.NotificationFallbackOrder
	}
	if o.SMSProvider != nil {
		merged.SMSProvider = *o.SMSProvider
	}
	if o.MapsProvider != nil {
		merged.MapsProvider = *o.MapsProvider
	}
	if o.LLMProvider != nil {
		merged.LLMProvider = *o.LLMProvider
	}
	return &merged
}

// SeedDefaultConfigs inserts default country configs if they don't exist.
// Called during backend boot.
func SeedDefaultConfigs(ctx context.Context, client *spanner.Client) {
	configs := []CountryConfig{
		*defaultUZ(),
		{
			CountryCode: "KZ", CountryName: "Kazakhstan",
			Timezone: "Asia/Almaty", CurrencyCode: "KZT", CurrencyDecimalPlaces: 0,
			DistanceUnit: "km", DefaultVUConversion: 1.0,
			MapsProvider: "GOOGLE", LLMProvider: "GEMINI",
			PaymentGateways: []string{"KASPI"}, SMSProvider: "TWILIO",
			NotificationFallbackOrder: []string{"FCM", "SMS"},
			LegalRetentionDays:        365, GridSystem: "H3",
			BreachRadiusMeters: 100.0, ShopClosedGraceMinutes: 5,
			ShopClosedEscalationMinutes: 3, OfflineModeDurationMinutes: 30,
			CashCustodyAlertHours: 4,
		},
	}

	for _, cfg := range configs {
		payGW, _ := json.Marshal(cfg.PaymentGateways)
		notif, _ := json.Marshal(cfg.NotificationFallbackOrder)

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Check if exists
			_, readErr := txn.ReadRow(ctx, "CountryConfigs", spanner.Key{cfg.CountryCode}, []string{"CountryCode"})
			if readErr == nil {
				return nil // already exists
			}

			m, err := spanner.InsertStruct("CountryConfigs", &struct {
				CountryCode                 string    `spanner:"CountryCode"`
				CountryName                 string    `spanner:"CountryName"`
				Timezone                    string    `spanner:"Timezone"`
				CurrencyCode                string    `spanner:"CurrencyCode"`
				CurrencyDecimalPlaces       int64     `spanner:"CurrencyDecimalPlaces"`
				DistanceUnit                string    `spanner:"DistanceUnit"`
				DefaultVUConversion         float64   `spanner:"DefaultVUConversion"`
				MapsProvider                string    `spanner:"MapsProvider"`
				LLMProvider                 string    `spanner:"LLMProvider"`
				PaymentGateways             string    `spanner:"GlobalPayntGateways"`
				SMSProvider                 string    `spanner:"SMSProvider"`
				NotificationFallbackOrder   string    `spanner:"NotificationFallbackOrder"`
				LegalRetentionDays          int64     `spanner:"LegalRetentionDays"`
				GridSystem                  string    `spanner:"GridSystem"`
				BreachRadiusMeters          float64   `spanner:"BreachRadiusMeters"`
				ShopClosedGraceMinutes      int64     `spanner:"ShopClosedGraceMinutes"`
				ShopClosedEscalationMinutes int64     `spanner:"ShopClosedEscalationMinutes"`
				OfflineModeDurationMinutes  int64     `spanner:"OfflineModeDurationMinutes"`
				CashCustodyAlertHours       int64     `spanner:"CashCustodyAlertHours"`
				IsActive                    bool      `spanner:"IsActive"`
				CreatedAt                   time.Time `spanner:"CreatedAt"`
			}{
				CountryCode: cfg.CountryCode, CountryName: cfg.CountryName,
				Timezone: cfg.Timezone, CurrencyCode: cfg.CurrencyCode,
				CurrencyDecimalPlaces: cfg.CurrencyDecimalPlaces,
				DistanceUnit:          cfg.DistanceUnit, DefaultVUConversion: cfg.DefaultVUConversion,
				MapsProvider: cfg.MapsProvider, LLMProvider: cfg.LLMProvider,
				PaymentGateways: string(payGW), SMSProvider: cfg.SMSProvider,
				NotificationFallbackOrder: string(notif),
				LegalRetentionDays:        cfg.LegalRetentionDays, GridSystem: cfg.GridSystem,
				BreachRadiusMeters:          cfg.BreachRadiusMeters,
				ShopClosedGraceMinutes:      cfg.ShopClosedGraceMinutes,
				ShopClosedEscalationMinutes: cfg.ShopClosedEscalationMinutes,
				OfflineModeDurationMinutes:  cfg.OfflineModeDurationMinutes,
				CashCustodyAlertHours:       cfg.CashCustodyAlertHours,
				IsActive:                    true, CreatedAt: spanner.CommitTimestamp,
			})
			if err != nil {
				return err
			}
			txn.BufferWrite([]*spanner.Mutation{m})
			return nil
		})
		if err != nil {
			log.Printf("[CountryCfg] Seed %s failed: %v", cfg.CountryCode, err)
		} else {
			log.Printf("[CountryCfg] Seed %s: OK", cfg.CountryCode)
		}
	}
}

// ListAllConfigs returns all active country configurations.
func (s *Service) ListAllConfigs(ctx context.Context) ([]*CountryConfig, error) {
	stmt := spanner.Statement{SQL: "SELECT CountryCode FROM CountryConfigs WHERE IsActive = true"}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var configs []*CountryConfig
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var code string
		if err := row.Columns(&code); err != nil {
			continue
		}
		configs = append(configs, s.GetConfig(ctx, code))
	}
	return configs, nil
}

// UpsertConfig creates or updates a country config row and invalidates cache.
func (s *Service) UpsertConfig(ctx context.Context, cfg *CountryConfig) error {
	if cfg == nil || cfg.CountryCode == "" {
		return fmt.Errorf("country code is required")
	}

	merged := mergeCountryConfigForUpsert(s.readCountryConfig(ctx, cfg.CountryCode), cfg)
	paymentGatewaysJSON, _ := json.Marshal(merged.PaymentGateways)
	notifOrderJSON, _ := json.Marshal(merged.NotificationFallbackOrder)

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt := interface{}(spanner.CommitTimestamp)
		row, readErr := txn.ReadRow(ctx, "CountryConfigs", spanner.Key{merged.CountryCode}, []string{"CreatedAt"})
		if readErr == nil {
			var existingCreatedAt time.Time
			if colErr := row.Columns(&existingCreatedAt); colErr == nil {
				createdAt = existingCreatedAt
			}
		}

		m := spanner.InsertOrUpdate("CountryConfigs",
			[]string{
				"CountryCode", "CountryName", "Timezone", "CurrencyCode", "CurrencyDecimalPlaces",
				"DistanceUnit", "DefaultVUConversion", "MapsProvider", "LLMProvider",
				"GlobalPayntGateways", "SMSProvider", "NotificationFallbackOrder", "LegalRetentionDays",
				"GridSystem", "BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
				"OfflineModeDurationMinutes", "CashCustodyAlertHours", "IsActive", "CreatedAt", "UpdatedAt",
			},
			[]interface{}{
				merged.CountryCode, merged.CountryName, merged.Timezone, merged.CurrencyCode, merged.CurrencyDecimalPlaces,
				merged.DistanceUnit, merged.DefaultVUConversion, merged.MapsProvider, merged.LLMProvider,
				string(paymentGatewaysJSON), merged.SMSProvider, string(notifOrderJSON), merged.LegalRetentionDays,
				merged.GridSystem, merged.BreachRadiusMeters, merged.ShopClosedGraceMinutes, merged.ShopClosedEscalationMinutes,
				merged.OfflineModeDurationMinutes, merged.CashCustodyAlertHours, true, createdAt, spanner.CommitTimestamp,
			},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return err
	}

	s.invalidateCountryConfig(ctx, merged.CountryCode)
	return nil
}
