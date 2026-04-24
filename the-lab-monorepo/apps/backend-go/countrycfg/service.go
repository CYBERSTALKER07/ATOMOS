package countrycfg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
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
	GlobalPayntGateways             []string `json:"global_paynt_gateways"`
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
	GlobalPayntGateways             []string `json:"global_paynt_gateways"`
	NotificationFallbackOrder   []string `json:"notification_fallback_order"`
	SMSProvider                 *string  `json:"sms_provider"`
	MapsProvider                *string  `json:"maps_provider"`
	LLMProvider                 *string  `json:"llm_provider"`
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
}

// NewService creates a config service with default 5min cache TTL.
func NewService(client *spanner.Client) *Service {
	return &Service{
		Spanner:  client,
		cacheTTL: 5 * time.Minute,
	}
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
		GlobalPayntGateways:             []string{"GLOBAL_PAY", "CASH"},
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
	var global_payntGatewaysJSON, notifOrderJSON, smsProvider spanner.NullString

	if err := row.Columns(
		&cfg.CountryCode, &cfg.CountryName, &cfg.Timezone, &cfg.CurrencyCode,
		&cfg.CurrencyDecimalPlaces, &cfg.DistanceUnit, &cfg.DefaultVUConversion,
		&cfg.MapsProvider, &cfg.LLMProvider, &global_payntGatewaysJSON, &smsProvider,
		&notifOrderJSON, &cfg.LegalRetentionDays, &cfg.GridSystem,
		&cfg.BreachRadiusMeters, &cfg.ShopClosedGraceMinutes, &cfg.ShopClosedEscalationMinutes,
		&cfg.OfflineModeDurationMinutes, &cfg.CashCustodyAlertHours,
	); err != nil {
		log.Printf("[CountryCfg] Error scanning row for %s: %v", countryCode, err)
		return nil
	}

	if global_payntGatewaysJSON.Valid {
		_ = json.Unmarshal([]byte(global_payntGatewaysJSON.StringVal), &cfg.GlobalPayntGateways)
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
		})
	if err != nil {
		return nil
	}

	o := &SupplierOverride{}
	var breach spanner.NullFloat64
	var shopGrace, shopEsc, offlineDur, cashAlert spanner.NullInt64
	var payGW, notifOrder, sms, maps, llm spanner.NullString

	if err := row.Columns(
		&o.SupplierId, &o.CountryCode, &breach,
		&shopGrace, &shopEsc, &offlineDur, &cashAlert,
		&payGW, &notifOrder, &sms, &maps, &llm,
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
		_ = json.Unmarshal([]byte(payGW.StringVal), &o.GlobalPayntGateways)
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

// UpsertSupplierOverride creates or replaces the supplier's country override row.
func (s *Service) UpsertSupplierOverride(ctx context.Context, o *SupplierOverride) error {
	if o == nil || o.SupplierId == "" || o.CountryCode == "" {
		return fmt.Errorf("supplier_id and country_code are required")
	}

	var payGWJSON, notifOrderJSON []byte
	if len(o.GlobalPayntGateways) > 0 {
		payGWJSON, _ = json.Marshal(o.GlobalPayntGateways)
	}
	if len(o.NotificationFallbackOrder) > 0 {
		notifOrderJSON, _ = json.Marshal(o.NotificationFallbackOrder)
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cols := []string{
			"SupplierId", "CountryCode",
			"BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
			"OfflineModeDurationMinutes", "CashCustodyAlertHours",
			"GlobalPayntGateways", "NotificationFallbackOrder",
			"SMSProvider", "MapsProvider", "LLMProvider",
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
			spanner.CommitTimestamp,
		}

		// Set CreatedAt only on first insert.
		existRow, readErr := txn.ReadRow(ctx, "SupplierCountryOverrides",
			spanner.Key{o.SupplierId, o.CountryCode}, []string{"CreatedAt"})
		if readErr != nil {
			cols = append(cols, "CreatedAt")
			vals = append(vals, spanner.CommitTimestamp)
		} else {
			var createdAt spanner.NullTime
			_ = existRow.Columns(&createdAt)
			cols = append(cols, "CreatedAt")
			vals = append(vals, createdAt.Time)
		}

		m := spanner.InsertOrUpdate("SupplierCountryOverrides", cols, vals)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("upsert supplier override %s/%s: %w", o.SupplierId, o.CountryCode, err)
	}

	// Invalidate cached entry.
	cacheKey := o.SupplierId + ":" + o.CountryCode
	s.overrideCache.Delete(cacheKey)
	log.Printf("[CountryCfg] Upserted override %s/%s", o.SupplierId, o.CountryCode)
	return nil
}

// DeleteSupplierOverride removes the supplier's override for a country (reverts to platform defaults).
func (s *Service) DeleteSupplierOverride(ctx context.Context, supplierID, countryCode string) error {
	if supplierID == "" || countryCode == "" {
		return fmt.Errorf("supplier_id and country_code are required")
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Delete("SupplierCountryOverrides", spanner.Key{supplierID, countryCode})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("delete supplier override %s/%s: %w", supplierID, countryCode, err)
	}

	cacheKey := supplierID + ":" + countryCode
	s.overrideCache.Delete(cacheKey)
	log.Printf("[CountryCfg] Deleted override %s/%s", supplierID, countryCode)
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
	if len(o.GlobalPayntGateways) > 0 {
		merged.GlobalPayntGateways = o.GlobalPayntGateways
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
			GlobalPayntGateways: []string{"KASPI"}, SMSProvider: "TWILIO",
			NotificationFallbackOrder: []string{"FCM", "SMS"},
			LegalRetentionDays:        365, GridSystem: "H3",
			BreachRadiusMeters: 100.0, ShopClosedGraceMinutes: 5,
			ShopClosedEscalationMinutes: 3, OfflineModeDurationMinutes: 30,
			CashCustodyAlertHours: 4,
		},
	}

	for _, cfg := range configs {
		payGW, _ := json.Marshal(cfg.GlobalPayntGateways)
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
				GlobalPayntGateways             string    `spanner:"GlobalPayntGateways"`
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
				GlobalPayntGateways: string(payGW), SMSProvider: cfg.SMSProvider,
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

	base := defaultUZ()
	if cfg.CountryName == "" {
		cfg.CountryName = base.CountryName
	}
	if cfg.Timezone == "" {
		cfg.Timezone = base.Timezone
	}
	if cfg.CurrencyCode == "" {
		cfg.CurrencyCode = base.CurrencyCode
	}
	if cfg.DistanceUnit == "" {
		cfg.DistanceUnit = base.DistanceUnit
	}
	if cfg.MapsProvider == "" {
		cfg.MapsProvider = base.MapsProvider
	}
	if cfg.LLMProvider == "" {
		cfg.LLMProvider = base.LLMProvider
	}
	if len(cfg.GlobalPayntGateways) == 0 {
		cfg.GlobalPayntGateways = base.GlobalPayntGateways
	}
	if len(cfg.NotificationFallbackOrder) == 0 {
		cfg.NotificationFallbackOrder = base.NotificationFallbackOrder
	}
	if cfg.SMSProvider == "" {
		cfg.SMSProvider = base.SMSProvider
	}
	if cfg.CurrencyDecimalPlaces == 0 {
		cfg.CurrencyDecimalPlaces = base.CurrencyDecimalPlaces
	}
	if cfg.DefaultVUConversion == 0 {
		cfg.DefaultVUConversion = base.DefaultVUConversion
	}
	if cfg.LegalRetentionDays == 0 {
		cfg.LegalRetentionDays = base.LegalRetentionDays
	}
	if cfg.GridSystem == "" {
		cfg.GridSystem = base.GridSystem
	}
	if cfg.BreachRadiusMeters == 0 {
		cfg.BreachRadiusMeters = base.BreachRadiusMeters
	}
	if cfg.ShopClosedGraceMinutes == 0 {
		cfg.ShopClosedGraceMinutes = base.ShopClosedGraceMinutes
	}
	if cfg.ShopClosedEscalationMinutes == 0 {
		cfg.ShopClosedEscalationMinutes = base.ShopClosedEscalationMinutes
	}
	if cfg.OfflineModeDurationMinutes == 0 {
		cfg.OfflineModeDurationMinutes = base.OfflineModeDurationMinutes
	}
	if cfg.CashCustodyAlertHours == 0 {
		cfg.CashCustodyAlertHours = base.CashCustodyAlertHours
	}

	global_payntGatewaysJSON, _ := json.Marshal(cfg.GlobalPayntGateways)
	notifOrderJSON, _ := json.Marshal(cfg.NotificationFallbackOrder)

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt := interface{}(spanner.CommitTimestamp)
		row, readErr := txn.ReadRow(ctx, "CountryConfigs", spanner.Key{cfg.CountryCode}, []string{"CreatedAt"})
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
				cfg.CountryCode, cfg.CountryName, cfg.Timezone, cfg.CurrencyCode, cfg.CurrencyDecimalPlaces,
				cfg.DistanceUnit, cfg.DefaultVUConversion, cfg.MapsProvider, cfg.LLMProvider,
				string(global_payntGatewaysJSON), cfg.SMSProvider, string(notifOrderJSON), cfg.LegalRetentionDays,
				cfg.GridSystem, cfg.BreachRadiusMeters, cfg.ShopClosedGraceMinutes, cfg.ShopClosedEscalationMinutes,
				cfg.OfflineModeDurationMinutes, cfg.CashCustodyAlertHours, true, createdAt, spanner.CommitTimestamp,
			},
		)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return err
	}

	s.InvalidateCache(cfg.CountryCode)
	return nil
}
