package auth

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultSmokeSKU         = "SSMR-SKU-1"
	defaultSmokeInventoryQOH = int64(10000)
)

// EnsureDemoScopeLinks upserts SSMR demo warehouse/factory rows with factory linkage,
// smoke supplier login, and catalog inventory so FACTORY_ADMIN scope resolution and
// retailer order create succeed in the isolated sandbox.
func EnsureDemoScopeLinks(ctx context.Context, client *spanner.Client, supplierID string) error {
	if client == nil || strings.TrimSpace(supplierID) == "" {
		return nil
	}

	warehouseID := demoWarehouseID()
	factoryID := demoFactoryID()
	centerLat, centerLng := deliveryZoneCenter()
	coverageKm := deliveryZoneRadiusKm()
	sku := smokeSKU()

	password := strings.TrimSpace(os.Getenv("SSMR_SMOKE_SUPPLIER_PASSWORD"))
	if password == "" {
		password = "SmokeTest!234"
	}
	phone := strings.TrimSpace(os.Getenv("SSMR_SMOKE_SUPPLIER_PHONE"))
	if phone == "" {
		phone = "+998901000001"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash smoke supplier password: %w", err)
	}

	factoryPhone := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PHONE"))
	if factoryPhone == "" {
		factoryPhone = "+998901000099"
	}
	factoryPIN := strings.TrimSpace(os.Getenv("FACTORY_DEMO_PIN"))
	if factoryPIN == "" {
		factoryPIN = "1234"
	}
	factoryPINHash, err := bcrypt.GenerateFromPassword([]byte(factoryPIN), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash factory demo pin: %w", err)
	}

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		ts := spanner.CommitTimestamp
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("FactoryInternalTransfers", map[string]any{
				"TransferId":    "ssmr-wh-transfer-receive",
				"FactoryId":     factoryID,
				"SupplierId":    supplierID,
				"WarehouseId":   warehouseID,
				"State":         "IN_TRANSIT",
				"TotalVolumeVU": 12.0,
				"CreatedAt":     ts,
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("Factories", map[string]any{
				"FactoryId":  factoryID,
				"SupplierId": supplierID,
				"Name":       "SSMR Demo Factory",
				"Lat":        centerLat,
				"Lng":        centerLng + 0.01,
				"IsActive":   true,
				"CreatedAt":  ts,
				"UpdatedAt":  ts,
			}),
			spanner.InsertOrUpdateMap("Warehouses", map[string]any{
				"WarehouseId":      warehouseID,
				"SupplierId":       supplierID,
				"Name":             "SSMR Demo Warehouse",
				"Lat":              centerLat,
				"Lng":              centerLng,
				"CoverageRadiusKm": coverageKm,
				"PrimaryFactoryId": factoryID,
				"TransferMode":     "TRUCK",
				"IsActive":         true,
				"IsOnShift":        true,
				"CreatedAt":        ts,
				"UpdatedAt":        ts,
			}),
			spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
				"UserId":       "ssmr-smoke-supplier-admin",
				"SupplierId":   supplierID,
				"Phone":        phone,
				"Name":         "SSMR Smoke Supplier Admin",
				"PasswordHash": string(passwordHash),
				"SupplierRole": "ADMIN",
				"IsActive":     true,
				"CreatedAt":    ts,
				"UpdatedAt":    ts,
			}),
			spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
				"UserId":            "ssmr-smoke-factory-admin",
				"SupplierId":        supplierID,
				"Phone":             factoryPhone,
				"Name":              "SSMR Demo Factory Admin",
				"PasswordHash":      string(factoryPINHash),
				"SupplierRole":      "FACTORY_ADMIN",
				"AssignedFactoryId": factoryID,
				"IsActive":          true,
				"CreatedAt":         ts,
				"UpdatedAt":         ts,
			}),
			spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
				"SupplierId":       supplierID,
				"WarehouseId":      warehouseID,
				"ProductId":        sku,
				"QuantityOnHand":   defaultSmokeInventoryQOH,
				"QuantityReserved": int64(0),
				"UpdatedAt":        ts,
			}),
			spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
				"CategoryId": "ssmr-demo-category",
				"SupplierId": supplierID,
				"Name":       "SSMR Demo",
				"SortOrder":  int64(0),
				"CreatedAt":  ts,
				"UpdatedAt":  ts,
			}),
			spanner.InsertOrUpdateMap("Products", map[string]any{
				"ProductId":     sku,
				"SupplierId":    supplierID,
				"CategoryId":    "ssmr-demo-category",
				"Name":          "SSMR Demo SKU",
				"PriceMinor":    int64(50000),
				"Currency":      "UZS",
				"StockQuantity": defaultSmokeInventoryQOH,
				"Unit":          "UNIT",
				"UnitVolumeVU":  1.0,
				"IsActive":      true,
				"Version":       int64(1),
				"CreatedAt":     ts,
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("Drivers", map[string]any{
				"DriverId":     "drv_payload_1",
				"SupplierId":   supplierID,
				"Name":         "SSMR Payload Demo Driver",
				"Phone":        "+998901110033",
				"HomeNodeType": "WAREHOUSE",
				"HomeNodeId":   warehouseID,
				"IsActive":     true,
				"CreatedAt":    ts,
				"UpdatedAt":    ts,
			}),
			spanner.InsertOrUpdateMap("Vehicles", map[string]any{
				"VehicleId":    "veh_payload_1",
				"SupplierId":   supplierID,
				"LicensePlate": "01P111AA",
				"HomeNodeType": "WAREHOUSE",
				"HomeNodeId":   warehouseID,
				"IsActive":     true,
				"CreatedAt":    ts,
				"UpdatedAt":    ts,
			}),
			spanner.InsertOrUpdateMap("SupplierTruckManifests", map[string]any{
				"ManifestId":    "mf_payload_1",
				"SupplierId":    supplierID,
				"WarehouseId":   warehouseID,
				"RouteId":       "route_veh_payload_1",
				"TruckId":       "veh_payload_1",
				"DriverId":      "drv_payload_1",
				"State":         "DRAFT",
				"TotalVolumeVU": 75.0,
				"MaxVolumeVU":   140.0,
				"StopCount":     int64(2),
				"CreatedAt":     ts,
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("ManifestOrders", map[string]any{
				"ManifestId":    "mf_payload_1",
				"OrderId":       "ord_payload_1",
				"SequenceIndex": int64(1),
				"LoadingOrder":  int64(1),
				"VolumeVU":      41.0,
				"State":         "ASSIGNED",
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("ManifestOrders", map[string]any{
				"ManifestId":    "mf_payload_1",
				"OrderId":       "ord_payload_2",
				"SequenceIndex": int64(2),
				"LoadingOrder":  int64(2),
				"VolumeVU":      34.0,
				"State":         "ASSIGNED",
				"UpdatedAt":     ts,
			}),
			spanner.InsertOrUpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId":         "ins_wh_1",
				"WarehouseId":       warehouseID,
				"ProductId":         sku,
				"SupplierId":        supplierID,
				"CurrentStock":      int64(12),
				"DailyBurnRate":     4.5,
				"TimeToEmptyDays":   3.0,
				"SuggestedQuantity": int64(48),
				"UrgencyLevel":      "WARNING",
				"ReasonCode":        "LOW_STOCK",
				"Status":            "PENDING",
				"TargetFactoryId":   factoryID,
				"CreatedAt":         ts,
			}),
			spanner.InsertOrUpdateMap("ReplenishmentInsights", map[string]any{
				"InsightId":         "ins_wh_2",
				"WarehouseId":       warehouseID,
				"ProductId":         sku,
				"SupplierId":        supplierID,
				"CurrentStock":      int64(8),
				"DailyBurnRate":     5.0,
				"TimeToEmptyDays":   2.0,
				"SuggestedQuantity": int64(36),
				"UrgencyLevel":      "CRITICAL",
				"ReasonCode":        "HIGH_VELOCITY",
				"Status":            "PENDING",
				"TargetFactoryId":   factoryID,
				"CreatedAt":         ts,
			}),
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("ensure demo scope links: %w", err)
	}
	return nil
}

func demoWarehouseID() string {
	if id := strings.TrimSpace(os.Getenv("WAREHOUSE_DEMO_ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(os.Getenv("SSMR_SMOKE_WAREHOUSE_ID")); id != "" {
		return id
	}
	return "wh-demo-1"
}

func demoFactoryID() string {
	if id := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID")); id != "" {
		return id
	}
	return "factory-demo-1"
}

func smokeSKU() string {
	if sku := strings.TrimSpace(os.Getenv("SSMR_SMOKE_SKU")); sku != "" {
		return sku
	}
	return defaultSmokeSKU
}

func deliveryZoneCenter() (float64, float64) {
	lat := envFloat("DELIVERY_ZONE_CENTER_LAT", 41.2995)
	lng := envFloat("DELIVERY_ZONE_CENTER_LNG", 69.2401)
	return lat, lng
}

func deliveryZoneRadiusKm() float64 {
	radius := envFloat("DELIVERY_ZONE_RADIUS_KM", 20)
	if radius <= 0 {
		return 20
	}
	return radius
}

func envFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
