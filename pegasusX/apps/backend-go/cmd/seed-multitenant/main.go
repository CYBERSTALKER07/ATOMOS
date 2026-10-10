// Package main seeds complete multi-tenant topologies into Cloud Spanner for Milestone 2.
// It idempotently provisions primary and secondary suppliers, warehouses, factories,
// localized product catalogs with cold-chain bounds, stock lots with FEFO shelf-life,
// vehicles, drivers with secure PINs, retailer accounts with credit limits, and full role matrices.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	PrimarySupplierID   = seed.DefaultSupplierID // "sup_61d822c6ab9714ca11f20db9"
	SecondarySupplierID = "sup_secondary_samarkand_dist"
	DefaultPIN          = "1234"
	DefaultPassword     = "SmokeTest!234"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	slog.Info("starting multi-tenant seed data provisioning",
		"primary_supplier_id", PrimarySupplierID,
		"secondary_supplier_id", SecondarySupplierID,
		"database", spannerDatabasePath(cfg),
	)

	if err := seedMultiTenantData(ctx, client, cfg); err != nil {
		slog.Error("seed multi-tenant data failed", "err", err)
		os.Exit(1)
	}

	slog.Info("multi-tenant seed data provisioned successfully")
}

func seedMultiTenantData(ctx context.Context, client *spanner.Client, cfg *bootstrap.Config) error {
	ts := spanner.CommitTimestamp
	now := time.Now().UTC()
	effectiveFrom := now.Add(-48 * time.Hour)

	pinHash, err := bcrypt.GenerateFromPassword([]byte(DefaultPIN), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash pin: %w", err)
	}
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	mutations := make([]*spanner.Mutation, 0, 100)

	// ==========================================
	// 1. PRIMARY SUPPLIER: Tashkent Beverage & FMCG Distribution LLC
	// ==========================================
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":   PrimarySupplierID,
			"Name":         "SSMR Smoke Supplier",
			"CountryCode":  "UZ",
			"Currency":     "UZS",
			"IsConfigured": true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("SupplierProfiles", map[string]any{
			"SupplierId":        PrimarySupplierID,
			"ContactName":       "Alisher Rustamov",
			"Email":             "admin@tashkentdist.uz",
			"Phone":             "+998901000001",
			"TaxId":             "998123456",
			"CompanyRegNumber":  "REG-TAS-998811",
			"FleetVehicleCount": int64(10),
			"FleetMaxVU":        int64(1500),
			"FactoryCount":      int64(2),
			"IsRegistered":      true,
			"RegisteredAt":      ts,
			"UpdatedAt":         ts,
		}),
	)

	// Primary Supplier Factories
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("Factories", map[string]any{
			"FactoryId":           "fact_tashkent_central_01",
			"SupplierId":          PrimarySupplierID,
			"Name":                "Tashkent Central Bottling & Production Plant",
			"Lat":                 41.3050,
			"Lng":                 69.2550,
			"H3Cell":              "8821812837fffff",
			"Address":             "100 Industrial Parkway, Tashkent",
			"IsActive":            true,
			"DailyOutputCapacity": int64(1500),
			"CountryCode":         "UZ",
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("Factories", map[string]any{
			"FactoryId":           "factory-demo-1",
			"SupplierId":          PrimarySupplierID,
			"Name":                "SSMR Demo Factory",
			"Lat":                 41.3095,
			"Lng":                 69.2501,
			"H3Cell":              "8821812837fffff",
			"Address":             "101 Demo Factory Rd, Tashkent",
			"IsActive":            true,
			"DailyOutputCapacity": int64(1000),
			"CountryCode":         "UZ",
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
	)

	// Primary Supplier Warehouses
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId":         "wh_tashkent_central_01",
			"SupplierId":          PrimarySupplierID,
			"Name":                "Tashkent Central Distribution Center",
			"Lat":                 41.2995,
			"Lng":                 69.2401,
			"H3Cell":              "8821812837fffff",
			"Address":             "123 Logistics Boulevard, Tashkent",
			"CoverageRadiusKm":    25.0,
			"PrimaryFactoryId":    "fact_tashkent_central_01",
			"TransferMode":        "TRUCK",
			"IsActive":            true,
			"IsOnShift":           true,
			"CountryCode":         "UZ",
			"AutoDispatchEnabled": false,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId":         "wh_tashkent_cold_02",
			"SupplierId":          PrimarySupplierID,
			"Name":                "Tashkent Cold Chain & Dairy Hub",
			"Lat":                 41.3110,
			"Lng":                 69.2790,
			"H3Cell":              "8821812837fffff",
			"Address":             "45 Cold Storage Way, Tashkent",
			"CoverageRadiusKm":    20.0,
			"PrimaryFactoryId":    "fact_tashkent_central_01",
			"TransferMode":        "TRUCK",
			"IsActive":            true,
			"IsOnShift":           true,
			"CountryCode":         "UZ",
			"AutoDispatchEnabled": false,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId":         "wh-demo-1",
			"SupplierId":          PrimarySupplierID,
			"Name":                "SSMR Demo Warehouse",
			"Lat":                 41.2995,
			"Lng":                 69.2401,
			"H3Cell":              "8821812837fffff",
			"Address":             "123 Main St, Tashkent",
			"CoverageRadiusKm":    15.0,
			"PrimaryFactoryId":    "factory-demo-1",
			"TransferMode":        "TRUCK",
			"IsActive":            true,
			"IsOnShift":           true,
			"CountryCode":         "UZ",
			"AutoDispatchEnabled": false,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
	)

	// Primary Supplier Product Categories
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_beverages",
			"SupplierId": PrimarySupplierID,
			"Name":       "Beverages & Drinks",
			"SortOrder":  int64(1),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_dairy",
			"SupplierId": PrimarySupplierID,
			"Name":       "Dairy & Cold Chain",
			"SortOrder":  int64(2),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_snacks",
			"SupplierId": PrimarySupplierID,
			"Name":       "Snacks & Confectionery",
			"SortOrder":  int64(3),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_frozen",
			"SupplierId": PrimarySupplierID,
			"Name":       "Frozen & Ice Cream",
			"SortOrder":  int64(4),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_bakery",
			"SupplierId": PrimarySupplierID,
			"Name":       "Bakery & Fresh",
			"SortOrder":  int64(5),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
	)

	// Primary Supplier Products
	mutations = append(mutations,
		// 1. Classic Cola 1.5L (Ambient Beverage)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_cola_1500ml",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_beverages",
			"Name":              "Classic Cola 1.5L",
			"Description":       "Refreshing carbonated soft drink 1.5L bottle",
			"Barcode":           "5901234123457",
			"PriceMinor":        int64(1400000), // 14,000 UZS
			"Currency":          "UZS",
			"StockQuantity":     int64(2000),
			"Unit":              "BOTTLE",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(6),
			"UnitVolumeVU":      1.5,
			"Stackable":         true,
			"MaxStackHeight":    int64(4),
			"HandlingClass":     "GENERAL",
			"RequiresColdChain": false,
			"IsHazardous":       false,
			"IsPerishable":      false,
			"MinShelfLifeDays":  int64(180),
			"StorageTempMinC":   5.0,
			"StorageTempMaxC":   25.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 2. Pure Mountain Mineral Water 1.0L
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_mineral_water_1000ml",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_beverages",
			"Name":              "Pure Mountain Mineral Water 1.0L",
			"Description":       "Natural still mineral water 1L bottle",
			"Barcode":           "5901234123464",
			"PriceMinor":        int64(600000), // 6,000 UZS
			"Currency":          "UZS",
			"StockQuantity":     int64(1500),
			"Unit":              "BOTTLE",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(12),
			"UnitVolumeVU":      1.0,
			"Stackable":         true,
			"MaxStackHeight":    int64(5),
			"HandlingClass":     "GENERAL",
			"RequiresColdChain": false,
			"IsHazardous":       false,
			"IsPerishable":      false,
			"MinShelfLifeDays":  int64(365),
			"StorageTempMinC":   5.0,
			"StorageTempMaxC":   25.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 3. Pasteurized Fresh Whole Milk 3.2% (Cold Chain)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_dairy_milk_1000ml",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_dairy",
			"Name":              "Fresh Whole Milk 3.2% 1.0L",
			"Description":       "Pasteurized fresh cow milk with 3.2% fat",
			"Barcode":           "5901234123471",
			"PriceMinor":        int64(1200000), // 12,000 UZS
			"StockQuantity":     int64(1000),
			"Currency":          "UZS",
			"Unit":              "CARTON",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(12),
			"UnitVolumeVU":      1.0,
			"Stackable":         true,
			"MaxStackHeight":    int64(3),
			"HandlingClass":     "COLD_CHAIN",
			"RequiresColdChain": true,
			"IsHazardous":       false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(14),
			"StorageTempMinC":   2.0,
			"StorageTempMaxC":   6.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 4. Artisanal Sweet Cream Butter 82.5% (Cold Chain)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_dairy_butter_200g",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_dairy",
			"Name":              "Artisanal Sweet Cream Butter 82.5% 200g",
			"Description":       "Premium dairy butter block 200g",
			"Barcode":           "5901234123488",
			"PriceMinor":        int64(2800000), // 28,000 UZS
			"StockQuantity":     int64(300),
			"Currency":          "UZS",
			"Unit":              "PACK",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(20),
			"UnitVolumeVU":      0.5,
			"Stackable":         true,
			"MaxStackHeight":    int64(4),
			"HandlingClass":     "COLD_CHAIN",
			"RequiresColdChain": true,
			"IsHazardous":       false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(45),
			"StorageTempMinC":   2.0,
			"StorageTempMaxC":   4.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 5. Plombir Vanilla Ice Cream 500g (Frozen Cold Chain)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_icecream_vanilla_500g",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_frozen",
			"Name":              "Premium Plombir Vanilla Ice Cream 500g",
			"Description":       "Deep frozen traditional cream plombir",
			"Barcode":           "5901234123495",
			"PriceMinor":        int64(2200000), // 22,000 UZS
			"StockQuantity":     int64(500),
			"Currency":          "UZS",
			"Unit":              "TUB",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(8),
			"UnitVolumeVU":      0.8,
			"Stackable":         true,
			"MaxStackHeight":    int64(4),
			"HandlingClass":     "COLD_CHAIN",
			"RequiresColdChain": true,
			"IsHazardous":       false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(90),
			"StorageTempMinC":   -22.0,
			"StorageTempMaxC":   -18.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 6. Potato Chips Salted 150g (General Snacks)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_potato_chips_150g",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_snacks",
			"Name":              "Crispy Salted Potato Chips 150g",
			"Description":       "Golden crispy potato chips party bag",
			"Barcode":           "5901234123501",
			"PriceMinor":        int64(1600000), // 16,000 UZS
			"StockQuantity":     int64(800),
			"Currency":          "UZS",
			"Unit":              "BAG",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(14),
			"UnitVolumeVU":      2.5,
			"Stackable":         true,
			"MaxStackHeight":    int64(3),
			"HandlingClass":     "GENERAL",
			"RequiresColdChain": false,
			"IsHazardous":       false,
			"IsPerishable":      false,
			"MinShelfLifeDays":  int64(120),
			"StorageTempMinC":   15.0,
			"StorageTempMaxC":   25.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		// 7. Fresh Butter Croissants 4-Pack (Perishable Bakery)
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_fresh_croissant_pack4",
			"SupplierId":        PrimarySupplierID,
			"CategoryId":        "cat_bakery",
			"Name":              "Fresh Butter Croissants 4-Pack",
			"Description":       "Freshly baked French-style butter croissants",
			"Barcode":           "5901234123518",
			"PriceMinor":        int64(1800000), // 18,000 UZS
			"StockQuantity":     int64(250),
			"Currency":          "UZS",
			"Unit":              "PACK",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(6),
			"UnitVolumeVU":      1.2,
			"Stackable":         false,
			"MaxStackHeight":    int64(1),
			"HandlingClass":     "PERISHABLE",
			"RequiresColdChain": false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(5),
			"StorageTempMinC":   15.0,
			"StorageTempMaxC":   22.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
	)

	// Price Lists & Price List Items
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("PriceLists", map[string]any{
			"PriceListId":   "pl_tashkent_standard",
			"SupplierId":    PrimarySupplierID,
			"Name":          "Tashkent Standard Wholesale Price List",
			"EffectiveFrom": effectiveFrom,
			"EffectiveTo":   nil,
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    "pl_tashkent_standard",
			"Sku":            "sku_cola_1500ml",
			"UnitPriceMinor": int64(1400000),
			"MinQty":         int64(1),
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    "pl_tashkent_standard",
			"Sku":            "sku_mineral_water_1000ml",
			"UnitPriceMinor": int64(600000),
			"MinQty":         int64(1),
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    "pl_tashkent_standard",
			"Sku":            "sku_dairy_milk_1000ml",
			"UnitPriceMinor": int64(1200000),
			"MinQty":         int64(1),
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    "pl_tashkent_standard",
			"Sku":            "sku_dairy_butter_200g",
			"UnitPriceMinor": int64(2800000),
			"MinQty":         int64(1),
		}),
		spanner.InsertOrUpdateMap("PriceListItems", map[string]any{
			"PriceListId":    "pl_tashkent_standard",
			"Sku":            "sku_icecream_vanilla_500g",
			"UnitPriceMinor": int64(2200000),
			"MinQty":         int64(1),
		}),
	)

	// Warehouse Locations (Bins)
	mutations = append(mutations,
		// Central WH ambient locations
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_central_01",
			"LocationId":   "LOC-AMB-A01",
			"Zone":         "AMBIENT",
			"Aisle":        "A",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(10),
			"MaxVolumeVU":  float64(200.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_central_01",
			"LocationId":   "LOC-AMB-A02",
			"Zone":         "AMBIENT",
			"Aisle":        "A",
			"Rack":         "02",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(20),
			"MaxVolumeVU":  float64(200.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_central_01",
			"LocationId":   "LOC-SNK-B01",
			"Zone":         "SNACKS",
			"Aisle":        "B",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(30),
			"MaxVolumeVU":  float64(150.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Cold Hub locations
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_cold_02",
			"LocationId":   "LOC-COLD-C01",
			"Zone":         "COLD_ROOM",
			"Aisle":        "C",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(10),
			"MaxVolumeVU":  float64(100.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_cold_02",
			"LocationId":   "LOC-COLD-C02",
			"Zone":         "COLD_ROOM",
			"Aisle":        "C",
			"Rack":         "02",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(20),
			"MaxVolumeVU":  float64(100.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_tashkent_cold_02",
			"LocationId":   "LOC-FRZ-D01",
			"Zone":         "FREEZER",
			"Aisle":        "D",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(30),
			"MaxVolumeVU":  float64(80.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
	)

	// Stock Lots with realistic FEFO shelf-life
	mfgAugust := civil.DateOf(now.AddDate(0, 0, -10))
	mfgJuly := civil.DateOf(now.AddDate(0, 0, -40))

	// FEFO Milk batch 1 (expires in 12 days) vs batch 2 (expires in 26 days)
	expMilk1 := civil.DateOf(now.AddDate(0, 0, 12))
	expMilk2 := civil.DateOf(now.AddDate(0, 0, 26))
	expButter := civil.DateOf(now.AddDate(0, 1, 15))
	expIceCream := civil.DateOf(now.AddDate(0, 3, 0))
	expCola := civil.DateOf(now.AddDate(0, 6, 0))
	expWater := civil.DateOf(now.AddDate(1, 0, 0))
	expChips := civil.DateOf(now.AddDate(0, 4, 0))
	expCroissant := civil.DateOf(now.AddDate(0, 0, 5))

	mutations = append(mutations,
		// Milk FEFO Early Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_milk_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_dairy_milk_1000ml",
			"LocationId":       "LOC-COLD-C01",
			"LotCode":          "BATCH-MLK-202608A",
			"ExpiryDate":       expMilk1,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(400),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Milk FEFO Later Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_milk_batch_02",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_dairy_milk_1000ml",
			"LocationId":       "LOC-COLD-C02",
			"LotCode":          "BATCH-MLK-202608B",
			"ExpiryDate":       expMilk2,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(600),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Butter Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_butter_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_dairy_butter_200g",
			"LocationId":       "LOC-COLD-C01",
			"LotCode":          "BATCH-BTR-202608",
			"ExpiryDate":       expButter,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(300),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Ice Cream Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_icecream_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_icecream_vanilla_500g",
			"LocationId":       "LOC-FRZ-D01",
			"LotCode":          "BATCH-ICE-202608",
			"ExpiryDate":       expIceCream,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(500),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Cola Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_cola_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_cola_1500ml",
			"LocationId":       "LOC-AMB-A01",
			"LotCode":          "BATCH-COL-202608",
			"ExpiryDate":       expCola,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(2000),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Water Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_water_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_mineral_water_1000ml",
			"LocationId":       "LOC-AMB-A02",
			"LotCode":          "BATCH-WTR-202608",
			"ExpiryDate":       expWater,
			"ManufacturedDate": mfgAugust,
			"QuantityOnHand":   int64(1500),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Chips Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_chips_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_potato_chips_150g",
			"LocationId":       "LOC-SNK-B01",
			"LotCode":          "BATCH-CHP-202607",
			"ExpiryDate":       expChips,
			"ManufacturedDate": mfgJuly,
			"QuantityOnHand":   int64(800),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		// Croissant Lot
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_croissant_batch_01",
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_fresh_croissant_pack4",
			"LocationId":       "LOC-AMB-A01",
			"LotCode":          "BATCH-BAK-202608",
			"ExpiryDate":       expCroissant,
			"ManufacturedDate": civil.DateOf(now),
			"QuantityOnHand":   int64(250),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
	)

	// Aggregated Inventory roll-up (SupplierInventoryV2)
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_cola_1500ml",
			"QuantityOnHand":   int64(2000),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_mineral_water_1000ml",
			"QuantityOnHand":   int64(1500),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_dairy_milk_1000ml",
			"QuantityOnHand":   int64(1000),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_dairy_butter_200g",
			"QuantityOnHand":   int64(300),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_cold_02",
			"ProductId":        "sku_icecream_vanilla_500g",
			"QuantityOnHand":   int64(500),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_potato_chips_150g",
			"QuantityOnHand":   int64(800),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       PrimarySupplierID,
			"WarehouseId":      "wh_tashkent_central_01",
			"ProductId":        "sku_fresh_croissant_pack4",
			"QuantityOnHand":   int64(250),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
	)

	// Delivery Vehicles for Primary Supplier
	mutations = append(mutations,
		// Class B: Standard Transit Van
		spanner.InsertOrUpdateMap("Vehicles", map[string]any{
			"VehicleId":    "veh_tashkent_van_01",
			"SupplierId":   PrimarySupplierID,
			"Label":        "Tashkent Transit Van 01",
			"LicensePlate": "01A777AA",
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_central_01",
			"VehicleClass": "CLASS_B",
			"MaxVolumeVU":  150.0,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Class B: Refrigerated Cold-Chain Van
		spanner.InsertOrUpdateMap("Vehicles", map[string]any{
			"VehicleId":    "veh_tashkent_reefer_02",
			"SupplierId":   PrimarySupplierID,
			"Label":        "Tashkent Reefer Van 02 (Cold Chain)",
			"LicensePlate": "01B888BB",
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_cold_02",
			"VehicleClass": "CLASS_B",
			"MaxVolumeVU":  150.0,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Class C: Large Box Truck
		spanner.InsertOrUpdateMap("Vehicles", map[string]any{
			"VehicleId":    "veh_tashkent_truck_03",
			"SupplierId":   PrimarySupplierID,
			"Label":        "Tashkent Heavy Box Truck 03",
			"LicensePlate": "01C999CC",
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_central_01",
			"VehicleClass": "CLASS_C",
			"MaxVolumeVU":  400.0,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Class A: Damas Minivan
		spanner.InsertOrUpdateMap("Vehicles", map[string]any{
			"VehicleId":    "veh_tashkent_damas_04",
			"SupplierId":   PrimarySupplierID,
			"Label":        "Tashkent Damas Express 04",
			"LicensePlate": "01D111DD",
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_central_01",
			"VehicleClass": "CLASS_A",
			"MaxVolumeVU":  50.0,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
	)

	// Drivers for Primary Supplier with secure PINs
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("Drivers", map[string]any{
			"DriverId":     "drv_tashkent_lead_01",
			"SupplierId":   PrimarySupplierID,
			"Name":         "Alisher Navoiy",
			"Phone":        "+998901112233",
			"PinHash":      string(pinHash),
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_central_01",
			"VehicleId":    "veh_tashkent_van_01",
			"IsActive":     true,
			"OnShift":      true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("Drivers", map[string]any{
			"DriverId":     "drv_tashkent_cold_02",
			"SupplierId":   PrimarySupplierID,
			"Name":         "Bobur Temur",
			"Phone":        "+998901112244",
			"PinHash":      string(pinHash),
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_cold_02",
			"VehicleId":    "veh_tashkent_reefer_02",
			"IsActive":     true,
			"OnShift":      true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("Drivers", map[string]any{
			"DriverId":     "drv_tashkent_express_03",
			"SupplierId":   PrimarySupplierID,
			"Name":         "Jasur Fayzullayev",
			"Phone":        "+998901112255",
			"PinHash":      string(pinHash),
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_tashkent_central_01",
			"VehicleId":    "veh_tashkent_damas_04",
			"IsActive":     true,
			"OnShift":      true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
	)

	// Supplier Users (Complete Role Matrix: ADMIN, WAREHOUSE_ADMIN, WAREHOUSE/Picker, PAYLOADER, DRIVER, FINANCE, FACTORY_ADMIN)
	mutations = append(mutations,
		// Admin
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":       "usr_sup_admin_01",
			"SupplierId":   PrimarySupplierID,
			"Name":         "SSMR Smoke Supplier Admin",
			"Phone":        "+998901000001",
			"Email":        "admin@tashkentdist.uz",
			"PasswordHash": string(pwdHash),
			"SupplierRole": "ADMIN",
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Warehouse Admin / Manager
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":              "usr_wh_mgr_01",
			"SupplierId":          PrimarySupplierID,
			"Name":                "ProdSim Warehouse Admin",
			"Phone":               "+998901234590",
			"Email":               "wh_admin@tashkentdist.uz",
			"PasswordHash":        string(pwdHash),
			"SupplierRole":        "WAREHOUSE_ADMIN",
			"AssignedWarehouseId": "wh_tashkent_central_01",
			"IsActive":            true,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		// Warehouse Picker / Staff
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":              "usr_wh_picker_01",
			"SupplierId":          PrimarySupplierID,
			"Name":                "Tashkent Central Picker",
			"Phone":               "+998901234591",
			"Email":               "picker@tashkentdist.uz",
			"PasswordHash":        string(pinHash),
			"SupplierRole":        "WAREHOUSE",
			"AssignedWarehouseId": "wh_tashkent_central_01",
			"IsActive":            true,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		// Payloader / Staging specialist
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":              "usr_wh_payloader_01",
			"SupplierId":          PrimarySupplierID,
			"Name":                "SSMR Demo Payloader",
			"Phone":               "+998901110022",
			"Email":               "payloader@tashkentdist.uz",
			"PasswordHash":        string(pwdHash),
			"SupplierRole":        "PAYLOADER",
			"AssignedWarehouseId": "wh_tashkent_central_01",
			"IsActive":            true,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		// Finance / Accountant & Cash Reconciler
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":       "usr_finance_mgr_01",
			"SupplierId":   PrimarySupplierID,
			"Name":         "Tashkent Finance Controller",
			"Phone":        "+998901118899",
			"Email":        "finance@tashkentdist.uz",
			"PasswordHash": string(pwdHash),
			"SupplierRole": "FINANCE",
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		// Factory Admin / Production Lead
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":            "usr_fact_admin_01",
			"SupplierId":        PrimarySupplierID,
			"Name":              "SSMR Demo Factory Admin",
			"Phone":             "+998901000099",
			"Email":             "factory_admin@tashkentdist.uz",
			"PasswordHash":      string(pwdHash),
			"SupplierRole":      "FACTORY_ADMIN",
			"AssignedFactoryId": "fact_tashkent_central_01",
			"IsActive":          true,
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
	)

	// ==========================================
	// 2. SECONDARY SUPPLIER: Samarkand Regional Logistics LLC
	// ==========================================
	mutations = append(mutations,
		spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":   SecondarySupplierID,
			"Name":         "Samarkand Regional Logistics LLC",
			"CountryCode":  "UZ",
			"Currency":     "UZS",
			"IsConfigured": true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("SupplierProfiles", map[string]any{
			"SupplierId":        SecondarySupplierID,
			"ContactName":       "Temur Samarkandiy",
			"Email":             "admin@samarkandlog.uz",
			"Phone":             "+998905550001",
			"TaxId":             "998654321",
			"CompanyRegNumber":  "REG-SAM-112233",
			"FleetVehicleCount": int64(5),
			"FleetMaxVU":        int64(750),
			"FactoryCount":      int64(1),
			"IsRegistered":      true,
			"RegisteredAt":      ts,
			"UpdatedAt":         ts,
		}),
		spanner.InsertOrUpdateMap("Factories", map[string]any{
			"FactoryId":           "fact_samarkand_bev_01",
			"SupplierId":          SecondarySupplierID,
			"Name":                "Samarkand Dairy & Bottling Plant",
			"Lat":                 39.6600,
			"Lng":                 66.9700,
			"H3Cell":              "882189a697fffff",
			"Address":             "12 Registan Industrial Rd, Samarkand",
			"IsActive":            true,
			"DailyOutputCapacity": int64(1000),
			"CountryCode":         "UZ",
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId":         "wh_samarkand_hub_01",
			"SupplierId":          SecondarySupplierID,
			"Name":                "Samarkand Regional Hub",
			"Lat":                 39.6542,
			"Lng":                 66.9597,
			"H3Cell":              "882189a697fffff",
			"Address":             "50 Silk Road Way, Samarkand",
			"CoverageRadiusKm":    30.0,
			"PrimaryFactoryId":    "fact_samarkand_bev_01",
			"TransferMode":        "TRUCK",
			"IsActive":            true,
			"IsOnShift":           true,
			"CountryCode":         "UZ",
			"AutoDispatchEnabled": false,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId": "cat_sec_samarkand_dairy",
			"SupplierId": SecondarySupplierID,
			"Name":       "Samarkand Regional Produce & Dairy",
			"SortOrder":  int64(1),
			"CreatedAt":  ts,
			"UpdatedAt":  ts,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_sec_samarkand_bread",
			"SupplierId":        SecondarySupplierID,
			"CategoryId":        "cat_sec_samarkand_dairy",
			"Name":              "Samarkand Traditional Non Bread",
			"Description":       "Traditional clay oven baked Samarkand bread",
			"Barcode":           "5901234990011",
			"PriceMinor":        int64(500000), // 5,000 UZS
			"StockQuantity":     int64(300),
			"Currency":          "UZS",
			"Unit":              "PIECE",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(10),
			"UnitVolumeVU":      1.0,
			"Stackable":         true,
			"MaxStackHeight":    int64(5),
			"HandlingClass":     "PERISHABLE",
			"RequiresColdChain": false,
			"IsHazardous":       false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(4),
			"StorageTempMinC":   15.0,
			"StorageTempMaxC":   25.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":         "sku_sec_samarkand_yogurt",
			"SupplierId":        SecondarySupplierID,
			"CategoryId":        "cat_sec_samarkand_dairy",
			"Name":              "Samarkand Bio Yogurt 500g",
			"Description":       "Natural fermented probiotic yogurt",
			"Barcode":           "5901234990028",
			"PriceMinor":        int64(900000), // 9,000 UZS
			"StockQuantity":     int64(450),
			"Currency":          "UZS",
			"Unit":              "JAR",
			"SaleUnit":          "UNIT",
			"UnitsPerPack":      int64(12),
			"UnitVolumeVU":      0.6,
			"Stackable":         true,
			"MaxStackHeight":    int64(4),
			"HandlingClass":     "COLD_CHAIN",
			"RequiresColdChain": true,
			"IsHazardous":       false,
			"IsPerishable":      true,
			"MinShelfLifeDays":  int64(20),
			"StorageTempMinC":   2.0,
			"StorageTempMaxC":   6.0,
			"IsActive":          true,
			"Version":           int64(1),
			"CreatedAt":         ts,
			"UpdatedAt":         ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_samarkand_hub_01",
			"LocationId":   "LOC-SAM-A01",
			"Zone":         "AMBIENT",
			"Aisle":        "A",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(10),
			"MaxVolumeVU":  float64(200.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("WarehouseLocations", map[string]any{
			"WarehouseId":  "wh_samarkand_hub_01",
			"LocationId":   "LOC-SAM-C01",
			"Zone":         "COLD_ROOM",
			"Aisle":        "C",
			"Rack":         "01",
			"Level":        "1",
			"Bin":          "01",
			"LocationType": "PICK",
			"PickSequence": int64(20),
			"MaxVolumeVU":  float64(150.0),
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_sec_bread_01",
			"SupplierId":       SecondarySupplierID,
			"WarehouseId":      "wh_samarkand_hub_01",
			"ProductId":        "sku_sec_samarkand_bread",
			"LocationId":       "LOC-SAM-A01",
			"LotCode":          "BATCH-SAM-BRD-01",
			"ExpiryDate":       civil.DateOf(now.AddDate(0, 0, 4)),
			"ManufacturedDate": civil.DateOf(now),
			"QuantityOnHand":   int64(300),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId":            "lot_sec_yogurt_01",
			"SupplierId":       SecondarySupplierID,
			"WarehouseId":      "wh_samarkand_hub_01",
			"ProductId":        "sku_sec_samarkand_yogurt",
			"LocationId":       "LOC-SAM-C01",
			"LotCode":          "BATCH-SAM-YOG-01",
			"ExpiryDate":       civil.DateOf(now.AddDate(0, 0, 20)),
			"ManufacturedDate": civil.DateOf(now.AddDate(0, 0, -2)),
			"QuantityOnHand":   int64(450),
			"QuantityReserved": int64(0),
			"ReceivedAt":       ts,
			"Status":           "AVAILABLE",
			"CreatedAt":        ts,
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       SecondarySupplierID,
			"WarehouseId":      "wh_samarkand_hub_01",
			"ProductId":        "sku_sec_samarkand_bread",
			"QuantityOnHand":   int64(300),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
			"SupplierId":       SecondarySupplierID,
			"WarehouseId":      "wh_samarkand_hub_01",
			"ProductId":        "sku_sec_samarkand_yogurt",
			"QuantityOnHand":   int64(450),
			"QuantityReserved": int64(0),
			"UpdatedAt":        ts,
		}),
		spanner.InsertOrUpdateMap("Vehicles", map[string]any{
			"VehicleId":    "veh_samarkand_van_01",
			"SupplierId":   SecondarySupplierID,
			"Label":        "Samarkand Delivery Van 01",
			"LicensePlate": "20A123AA",
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_samarkand_hub_01",
			"VehicleClass": "CLASS_B",
			"MaxVolumeVU":  150.0,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("Drivers", map[string]any{
			"DriverId":     "drv_samarkand_01",
			"SupplierId":   SecondarySupplierID,
			"Name":         "Jamshid Qodirov",
			"Phone":        "+998902223344",
			"PinHash":      string(pinHash),
			"HomeNodeType": "WAREHOUSE",
			"HomeNodeId":   "wh_samarkand_hub_01",
			"VehicleId":    "veh_samarkand_van_01",
			"IsActive":     true,
			"OnShift":      true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":       "usr_sec_admin_01",
			"SupplierId":   SecondarySupplierID,
			"Name":         "Samarkand Supplier Admin",
			"Phone":        "+998905550001",
			"Email":        "admin@samarkandlog.uz",
			"PasswordHash": string(pwdHash),
			"SupplierRole": "ADMIN",
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":              "usr_sec_wh_mgr_01",
			"SupplierId":          SecondarySupplierID,
			"Name":                "Samarkand Hub Manager",
			"Phone":               "+998905550002",
			"Email":               "wh_mgr@samarkandlog.uz",
			"PasswordHash":        string(pwdHash),
			"SupplierRole":        "WAREHOUSE_ADMIN",
			"AssignedWarehouseId": "wh_samarkand_hub_01",
			"IsActive":            true,
			"CreatedAt":           ts,
			"UpdatedAt":           ts,
		}),
		spanner.InsertOrUpdateMap("SupplierUsers", map[string]any{
			"UserId":       "usr_sec_finance_01",
			"SupplierId":   SecondarySupplierID,
			"Name":         "Samarkand Chief Accountant",
			"Phone":        "+998905550003",
			"Email":        "finance@samarkandlog.uz",
			"PasswordHash": string(pwdHash),
			"SupplierRole": "FINANCE",
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
	)

	// ==========================================
	// 3. RETAILERS & CREDIT PROFILES & MULTI-USER ROLES
	// ==========================================
	mutations = append(mutations,
		// Retailer 1: Korzinka Supermarket Chilonzor
		spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId":           "ret_korzinka_chilonzor_01",
			"Name":                 "Korzinka Supermarket Chilonzor",
			"Phone":                "+998909876501",
			"Email":                "chilonzor@korzinka.uz",
			"CountryCode":          "UZ",
			"Lat":                  41.2780,
			"Lng":                  69.2100,
			"H3Cell":               "8821812837fffff",
			"DeliveryAddress":      "Chilonzor District 9, Tashkent",
			"ReceivingWindowOpen":  "08:00",
			"ReceivingWindowClose": "18:00",
			"Timezone":             "Asia/Tashkent",
			"Gln":                  "4780000000018",
			"CreatedAt":            ts,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           "ret_korzinka_chilonzor_01",
			"SupplierId":           PrimarySupplierID,
			"CreditLimitMinor":     int64(500000000), // 5,000,000 UZS
			"CurrentBalanceMinor":  int64(0),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(500000000),
			"RiskScore":            int64(85),
			"DelinquencyCount":     int64(0),
			"Status":               "ACTIVE",
			"LastEvaluatedAt":      now,
			"Version":              int64(1),
			"CreatedAt":            ts,
			"UpdatedAt":            ts,
		}),
		// Retailer 1 Users (Role Matrix: OWNER, MANAGER, BUYER, CASHIER, RECEIVER)
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_owner_01",
			"RetailerId":   "ret_korzinka_chilonzor_01",
			"Phone":        "+998909876501",
			"Name":         "Sardor Korzinka Owner",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "OWNER",
			"IsOwner":      true,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_mgr_01",
			"RetailerId":   "ret_korzinka_chilonzor_01",
			"Phone":        "+998909876511",
			"Name":         "Nodir Store Manager",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "MANAGER",
			"IsOwner":      false,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_buyer_01",
			"RetailerId":   "ret_korzinka_chilonzor_01",
			"Phone":        "+998909876512",
			"Name":         "Dilnoza Purchasing Buyer",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "BUYER",
			"IsOwner":      false,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_cashier_01",
			"RetailerId":   "ret_korzinka_chilonzor_01",
			"Phone":        "+998909876513",
			"Name":         "Madina POS Cashier",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "CASHIER",
			"IsOwner":      false,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_receiver_01",
			"RetailerId":   "ret_korzinka_chilonzor_01",
			"Phone":        "+998909876514",
			"Name":         "Otabek Dock Receiver",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "RECEIVER",
			"IsOwner":      false,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),

		// Retailer 2: Makro Express Yunusabad
		spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId":           "ret_makro_yunusabad_02",
			"Name":                 "Makro Express Yunusabad",
			"Phone":                "+998909876502",
			"Email":                "yunusabad@makro.uz",
			"CountryCode":          "UZ",
			"Lat":                  41.3650,
			"Lng":                  69.2880,
			"H3Cell":               "8821812837fffff",
			"DeliveryAddress":      "Yunusabad 4, Tashkent",
			"ReceivingWindowOpen":  "07:00",
			"ReceivingWindowClose": "22:00",
			"Timezone":             "Asia/Tashkent",
			"Gln":                  "4780000000025",
			"CreatedAt":            ts,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           "ret_makro_yunusabad_02",
			"SupplierId":           PrimarySupplierID,
			"CreditLimitMinor":     int64(800000000), // 8,000,000 UZS
			"CurrentBalanceMinor":  int64(0),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(800000000),
			"RiskScore":            int64(92),
			"DelinquencyCount":     int64(0),
			"Status":               "ACTIVE",
			"LastEvaluatedAt":      now,
			"Version":              int64(1),
			"CreatedAt":            ts,
			"UpdatedAt":            ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_makro_owner",
			"RetailerId":   "ret_makro_yunusabad_02",
			"Phone":        "+998909876502",
			"Name":         "Ulugbek Makro Owner",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "OWNER",
			"IsOwner":      true,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),

		// Retailer 3: Registan Grocery Samarkand (Bound to Secondary Supplier)
		spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId":           "ret_samarkand_market_03",
			"Name":                 "Registan Grocery Samarkand",
			"Phone":                "+998909876503",
			"Email":                "registan@grocery.uz",
			"CountryCode":          "UZ",
			"Lat":                  39.6540,
			"Lng":                  66.9750,
			"H3Cell":               "882189a697fffff",
			"DeliveryAddress":      "Registan Street 12, Samarkand",
			"ReceivingWindowOpen":  "08:00",
			"ReceivingWindowClose": "20:00",
			"Timezone":             "Asia/Tashkent",
			"Gln":                  "4780000000032",
			"CreatedAt":            ts,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           "ret_samarkand_market_03",
			"SupplierId":           SecondarySupplierID,
			"CreditLimitMinor":     int64(300000000), // 3,000,000 UZS
			"CurrentBalanceMinor":  int64(0),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(300000000),
			"RiskScore":            int64(78),
			"DelinquencyCount":     int64(0),
			"Status":               "ACTIVE",
			"LastEvaluatedAt":      now,
			"Version":              int64(1),
			"CreatedAt":            ts,
			"UpdatedAt":            ts,
		}),
		spanner.InsertOrUpdateMap("RetailerUsers", map[string]any{
			"UserId":       "usr_ret_sam_owner",
			"RetailerId":   "ret_samarkand_market_03",
			"Phone":        "+998909876503",
			"Name":         "Shavkat Samarkand Owner",
			"PasswordHash": string(pwdHash),
			"RetailerRole": "OWNER",
			"IsOwner":      true,
			"IsActive":     true,
			"CreatedAt":    ts,
			"UpdatedAt":    ts,
		}),
	)

	// Apply all mutations in batches inside ReadWriteTransaction
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("apply multi-tenant mutations: %w", err)
	}

	// Ensure demo scope links for backward compatibility with existing smoke test expectations
	if err := auth.EnsureDemoScopeLinks(ctx, client, PrimarySupplierID); err != nil {
		return fmt.Errorf("ensure demo scope links: %w", err)
	}

	return nil
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	project := strings.TrimSpace(cfg.SpannerProject)
	if project == "" {
		project = "pegasusx-ssmr-local"
	}
	instance := strings.TrimSpace(cfg.SpannerInstance)
	if instance == "" {
		instance = "pegasusx-ssmr-instance"
	}
	database := strings.TrimSpace(cfg.SpannerDatabase)
	if database == "" {
		database = "pegasusx-ssmr-db"
	}
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
}

func spannerClientOptions(cfg *bootstrap.Config) []option.ClientOption {
	host := strings.TrimSpace(cfg.SpannerEmulatorHost)
	if host == "" {
		host = strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	}
	if host == "" {
		return nil
	}
	return []option.ClientOption{
		option.WithEndpoint(host),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
}
