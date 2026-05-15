package countrycfg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Region models the region identity table used for market-level routing.
type Region struct {
	RegionID     string     `json:"region_id"`
	RegionCode   string     `json:"region_code"`
	RegionName   string     `json:"region_name"`
	CountryCode  string     `json:"country_code"`
	Timezone     string     `json:"timezone"`
	CurrencyCode string     `json:"currency_code"`
	DistanceUnit string     `json:"distance_unit"`
	IsDefault    bool       `json:"is_default"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

// RegionalConfig holds region-scoped operational overrides.
type RegionalConfig struct {
	RegionID                           string   `json:"region_id"`
	PaymentGateways                    []string `json:"payment_gateways"`
	NotificationFallbackOrder          []string `json:"notification_fallback_order"`
	SMSProvider                        string   `json:"sms_provider,omitempty"`
	MapsProvider                       string   `json:"maps_provider,omitempty"`
	LLMProvider                        string   `json:"llm_provider,omitempty"`
	DefaultVUConversion                float64  `json:"default_vu_conversion"`
	BreachRadiusMeters                 float64  `json:"breach_radius_meters"`
	ShopClosedGraceMinutes             int64    `json:"shop_closed_grace_minutes"`
	ShopClosedEscalationMinutes        int64    `json:"shop_closed_escalation_minutes"`
	OfflineModeDurationMinutes         int64    `json:"offline_mode_duration_minutes"`
	CashCustodyAlertHours              int64    `json:"cash_custody_alert_hours"`
	DegressiveFeeGrowthThresholdAmount int64    `json:"degressive_fee_growth_threshold_amount"`
	DegressiveFeeScaleThresholdAmount  int64    `json:"degressive_fee_scale_threshold_amount"`
	DegressiveFeeCapAmount             int64    `json:"degressive_fee_cap_amount"`
}

// GetRegionByID fetches region metadata by stable region ID.
func (s *Service) GetRegionByID(ctx context.Context, regionID string) (*Region, error) {
	if strings.TrimSpace(regionID) == "" {
		return nil, nil
	}

	row, err := s.Spanner.Single().ReadRow(ctx, "Regions", spanner.Key{regionID}, []string{
		"RegionId", "RegionCode", "RegionName", "CountryCode", "Timezone", "CurrencyCode",
		"DistanceUnit", "IsDefault", "IsActive", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return nil, nil
	}

	var region Region
	var updatedAt spanner.NullTime
	if err := row.Columns(
		&region.RegionID,
		&region.RegionCode,
		&region.RegionName,
		&region.CountryCode,
		&region.Timezone,
		&region.CurrencyCode,
		&region.DistanceUnit,
		&region.IsDefault,
		&region.IsActive,
		&region.CreatedAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("parse region %s: %w", regionID, err)
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		region.UpdatedAt = &t
	}
	return &region, nil
}

// GetDefaultRegionByCountry resolves the active default region for country code.
func (s *Service) GetDefaultRegionByCountry(ctx context.Context, countryCode string) (*Region, error) {
	country := strings.ToUpper(strings.TrimSpace(countryCode))
	if country == "" {
		country = "UZ"
	}

	stmt := spanner.Statement{
		SQL: `SELECT RegionId
		      FROM Regions
		      WHERE CountryCode = @countryCode AND IsActive = true AND IsDefault = true
		      LIMIT 1`,
		Params: map[string]interface{}{"countryCode": country},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, nil
	}
	var regionID string
	if err := row.Columns(&regionID); err != nil {
		return nil, fmt.Errorf("parse default region for %s: %w", country, err)
	}
	return s.GetRegionByID(ctx, regionID)
}

// ResolveSupplierRegion resolves an explicit supplier region first, then falls
// back to the country default region.
func (s *Service) ResolveSupplierRegion(ctx context.Context, supplierID string) (*Region, error) {
	if strings.TrimSpace(supplierID) == "" {
		return nil, nil
	}

	row, err := s.Spanner.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"RegionId", "CountryCode"})
	if err != nil {
		return nil, nil
	}

	var regionID spanner.NullString
	var countryCode spanner.NullString
	if err := row.Columns(&regionID, &countryCode); err != nil {
		return nil, fmt.Errorf("parse supplier region scope for %s: %w", supplierID, err)
	}

	if regionID.Valid && strings.TrimSpace(regionID.StringVal) != "" {
		region, getErr := s.GetRegionByID(ctx, regionID.StringVal)
		if getErr != nil {
			return nil, getErr
		}
		if region != nil {
			return region, nil
		}
	}

	country := "UZ"
	if countryCode.Valid {
		country = countryCode.StringVal
	}
	return s.GetDefaultRegionByCountry(ctx, country)
}

// GetRegionalConfig returns region-level overrides if present.
func (s *Service) GetRegionalConfig(ctx context.Context, regionID string) (*RegionalConfig, error) {
	if strings.TrimSpace(regionID) == "" {
		return nil, nil
	}

	row, err := s.Spanner.Single().ReadRow(ctx, "RegionalConfigs", spanner.Key{regionID}, []string{
		"RegionId", "PaymentGateways", "NotificationFallbackOrder", "SMSProvider", "MapsProvider", "LLMProvider",
		"DefaultVUConversion", "BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
		"OfflineModeDurationMinutes", "CashCustodyAlertHours",
		"DegressiveFeeGrowthThresholdAmount", "DegressiveFeeScaleThresholdAmount", "DegressiveFeeCapAmount",
	})
	if err != nil {
		lowerErr := strings.ToLower(err.Error())
		if strings.Contains(lowerErr, "degressivefee") && strings.Contains(lowerErr, "column") {
			return s.getRegionalConfigLegacy(ctx, regionID)
		}
		return nil, nil
	}

	return parseRegionalConfigRow(row, regionID, true)
}

func (s *Service) getRegionalConfigLegacy(ctx context.Context, regionID string) (*RegionalConfig, error) {
	row, err := s.Spanner.Single().ReadRow(ctx, "RegionalConfigs", spanner.Key{regionID}, []string{
		"RegionId", "PaymentGateways", "NotificationFallbackOrder", "SMSProvider", "MapsProvider", "LLMProvider",
		"DefaultVUConversion", "BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes",
		"OfflineModeDurationMinutes", "CashCustodyAlertHours",
	})
	if err != nil {
		return nil, nil
	}

	return parseRegionalConfigRow(row, regionID, false)
}

func parseRegionalConfigRow(row *spanner.Row, regionID string, includeDegressiveFeeFields bool) (*RegionalConfig, error) {
	cfg := &RegionalConfig{}
	var paymentGatewaysJSON spanner.NullString
	var notificationJSON spanner.NullString
	var smsProvider spanner.NullString
	var mapsProvider spanner.NullString
	var llmProvider spanner.NullString

	if includeDegressiveFeeFields {
		var growthThreshold spanner.NullInt64
		var scaleThreshold spanner.NullInt64
		var capAmount spanner.NullInt64
		if err := row.Columns(
			&cfg.RegionID,
			&paymentGatewaysJSON,
			&notificationJSON,
			&smsProvider,
			&mapsProvider,
			&llmProvider,
			&cfg.DefaultVUConversion,
			&cfg.BreachRadiusMeters,
			&cfg.ShopClosedGraceMinutes,
			&cfg.ShopClosedEscalationMinutes,
			&cfg.OfflineModeDurationMinutes,
			&cfg.CashCustodyAlertHours,
			&growthThreshold,
			&scaleThreshold,
			&capAmount,
		); err != nil {
			return nil, fmt.Errorf("parse regional config %s: %w", regionID, err)
		}
		if growthThreshold.Valid {
			cfg.DegressiveFeeGrowthThresholdAmount = growthThreshold.Int64
		}
		if scaleThreshold.Valid {
			cfg.DegressiveFeeScaleThresholdAmount = scaleThreshold.Int64
		}
		if capAmount.Valid {
			cfg.DegressiveFeeCapAmount = capAmount.Int64
		}
	} else {
		if err := row.Columns(
			&cfg.RegionID,
			&paymentGatewaysJSON,
			&notificationJSON,
			&smsProvider,
			&mapsProvider,
			&llmProvider,
			&cfg.DefaultVUConversion,
			&cfg.BreachRadiusMeters,
			&cfg.ShopClosedGraceMinutes,
			&cfg.ShopClosedEscalationMinutes,
			&cfg.OfflineModeDurationMinutes,
			&cfg.CashCustodyAlertHours,
		); err != nil {
			return nil, fmt.Errorf("parse regional config %s: %w", regionID, err)
		}
	}

	if paymentGatewaysJSON.Valid {
		_ = json.Unmarshal([]byte(paymentGatewaysJSON.StringVal), &cfg.PaymentGateways)
	}
	if notificationJSON.Valid {
		_ = json.Unmarshal([]byte(notificationJSON.StringVal), &cfg.NotificationFallbackOrder)
	}
	if smsProvider.Valid {
		cfg.SMSProvider = smsProvider.StringVal
	}
	if mapsProvider.Valid {
		cfg.MapsProvider = mapsProvider.StringVal
	}
	if llmProvider.Valid {
		cfg.LLMProvider = llmProvider.StringVal
	}

	return cfg, nil
}

// SeedDefaultRegions inserts baseline region and config rows when absent.
// This keeps rollout additive: existing country-level behavior remains active
// while region-scoped lookups gain a deterministic fallback target.
func SeedDefaultRegions(ctx context.Context, client *spanner.Client) {
	if client == nil {
		return
	}

	paymentGatewaysJSON, _ := json.Marshal([]string{"GLOBAL_PAY", "CASH"})
	notifJSON, _ := json.Marshal([]string{"FCM", "TELEGRAM"})

	seed := []struct {
		RegionID     string
		RegionCode   string
		RegionName   string
		CountryCode  string
		Timezone     string
		CurrencyCode string
		DistanceUnit string
		IsDefault    bool
	}{
		{
			RegionID:     "region-uz-default",
			RegionCode:   "UZ-DEFAULT",
			RegionName:   "Uzbekistan Default Region",
			CountryCode:  "UZ",
			Timezone:     "Asia/Tashkent",
			CurrencyCode: "UZS",
			DistanceUnit: "km",
			IsDefault:    true,
		},
	}

	for _, region := range seed {
		growthThresholdAmount, scaleThresholdAmount, capAmount := defaultRegionalDegressiveFeeThresholds(region.CurrencyCode)
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			mutations := make([]*spanner.Mutation, 0, 2)

			_, regionReadErr := txn.ReadRow(ctx, "Regions", spanner.Key{region.RegionID}, []string{"RegionId"})
			if regionReadErr != nil {
				mutations = append(mutations, spanner.Insert("Regions",
					[]string{"RegionId", "RegionCode", "RegionName", "CountryCode", "Timezone", "CurrencyCode", "DistanceUnit", "IsDefault", "IsActive", "CreatedAt"},
					[]interface{}{region.RegionID, region.RegionCode, region.RegionName, region.CountryCode, region.Timezone, region.CurrencyCode, region.DistanceUnit, region.IsDefault, true, spanner.CommitTimestamp},
				))
			}

			configRow, configReadErr := txn.ReadRow(ctx, "RegionalConfigs", spanner.Key{region.RegionID}, []string{
				"RegionId", "DegressiveFeeGrowthThresholdAmount", "DegressiveFeeScaleThresholdAmount", "DegressiveFeeCapAmount",
			})
			if configReadErr == nil {
				var existingRegionID string
				var existingGrowthThreshold spanner.NullInt64
				var existingScaleThreshold spanner.NullInt64
				var existingCapAmount spanner.NullInt64
				if err := configRow.Columns(&existingRegionID, &existingGrowthThreshold, &existingScaleThreshold, &existingCapAmount); err != nil {
					return err
				}
				if !existingGrowthThreshold.Valid || !existingScaleThreshold.Valid || !existingCapAmount.Valid {
					mutations = append(mutations, spanner.InsertOrUpdate("RegionalConfigs",
						[]string{"RegionId", "DegressiveFeeGrowthThresholdAmount", "DegressiveFeeScaleThresholdAmount", "DegressiveFeeCapAmount"},
						[]interface{}{region.RegionID, growthThresholdAmount, scaleThresholdAmount, capAmount},
					))
				}
			} else {
				mutations = append(mutations, spanner.Insert("RegionalConfigs",
					[]string{"RegionId", "PaymentGateways", "NotificationFallbackOrder", "MapsProvider", "LLMProvider", "DefaultVUConversion", "BreachRadiusMeters", "ShopClosedGraceMinutes", "ShopClosedEscalationMinutes", "OfflineModeDurationMinutes", "CashCustodyAlertHours", "DegressiveFeeGrowthThresholdAmount", "DegressiveFeeScaleThresholdAmount", "DegressiveFeeCapAmount", "CreatedAt"},
					[]interface{}{region.RegionID, string(paymentGatewaysJSON), string(notifJSON), "GOOGLE", "GEMINI", 1.0, 100.0, int64(5), int64(3), int64(30), int64(4), growthThresholdAmount, scaleThresholdAmount, capAmount, spanner.CommitTimestamp},
				))
			}

			if len(mutations) == 0 {
				return nil
			}
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			log.Printf("[CountryCfg] Seed region %s failed: %v", region.RegionCode, err)
		} else {
			log.Printf("[CountryCfg] Seed region %s: OK", region.RegionCode)
		}
	}
}

func defaultRegionalDegressiveFeeThresholds(currencyCode string) (int64, int64, int64) {
	switch strings.ToUpper(strings.TrimSpace(currencyCode)) {
	case "UZS":
		return 5_000_000, 20_000_000, 1_000_000
	default:
		return 50_000, 250_000, 25_000
	}
}
