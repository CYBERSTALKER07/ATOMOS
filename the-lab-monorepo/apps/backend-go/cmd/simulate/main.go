package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config represents the basic config for Spanner connections.
type Config struct {
	SpannerProject  string
	SpannerInstance string
	SpannerDatabase string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("[SIMULATE] Initializing Hyper-Scale Geo & AI Ecosystem Simulation...")

	// 1. Spanner Connection Initialization
	emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorAddr == "" {
		emulatorAddr = "localhost:9010"
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorAddr)
	}

	cfg := Config{
		SpannerProject:  "void-logistics",
		SpannerInstance: "void-local",
		SpannerDatabase: "void-db",
	}

	opts := []option.ClientOption{
		option.WithEndpoint(emulatorAddr),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}

	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)

	ctx := context.Background()
	spannerClient, err := spanner.NewClient(ctx, dbName, opts...)
	if err != nil {
		log.Fatalf("[SIMULATE] Spanner client failed: %v", err)
	}
	defer spannerClient.Close()

	// Check flags
	clearFlag := false
	numSuppliers := 5
	numWarehouses := 2
	numRetailers := 100

	for _, arg := range os.Args {
		if arg == "-clear" {
			clearFlag = true
		}
		if strings.HasPrefix(arg, "-suppliers=") {
			fmt.Sscanf(arg, "-suppliers=%d", &numSuppliers)
		}
		if strings.HasPrefix(arg, "-warehouses=") {
			fmt.Sscanf(arg, "-warehouses=%d", &numWarehouses)
		}
		if strings.HasPrefix(arg, "-retailers=") {
			fmt.Sscanf(arg, "-retailers=%d", &numRetailers)
		}
	}

	if clearFlag {
		log.Println("[SIMULATE] -clear flag provided. Truncating tables...")
		truncateTables(ctx, spannerClient)
	}

	log.Printf("[SIMULATE] Starting deterministic injection... Suppliers: %d, Warehouses: %d, Retailers: %d\n", numSuppliers, numWarehouses, numRetailers)
	seedSimulation(ctx, spannerClient, numSuppliers, numWarehouses, numRetailers)
	log.Println("[SIMULATE] Operation COMPLETED successfully. You can now use your physical Android device.")
}

func truncateTables(ctx context.Context, client *spanner.Client) {
	tables := []string{
		"AIPredictionItems", "AIPredictions", "OrderLineItems", "Orders",
		"RetailerVariantSettings", "RetailerProductSettings", "RetailerCategorySettings",
		"RetailerSupplierSettings", "RetailerGlobalSettings", "RetailerSuppliers", "Retailers",
		"SupplierProducts", "SupplierInventory", "Warehouses", "Factories", "Suppliers",
	}

	for _, t := range tables {
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			stmt := spanner.Statement{SQL: fmt.Sprintf("DELETE FROM %s WHERE TRUE", t)}
			count, err := txn.Update(ctx, stmt)
			if err == nil {
				log.Printf("[TRUNCATE] %s → %d rows deleted", t, count)
			}
			return err
		})
		if err != nil {
			log.Printf("[TRUNCATE] Failed on %s: %v", t, err)
		}
	}
}

func seedSimulation(ctx context.Context, client *spanner.Client, numSuppliers, numWarehouses, numRetailers int) {
	rng := rand.New(rand.NewSource(42)) // Deterministic random seed

	var allSupplierIds []string

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}

		for s := 1; s <= numSuppliers; s++ {
			sId := fmt.Sprintf("SUP-SIM-%03d", s)
			allSupplierIds = append(allSupplierIds, sId)

			mutations = append(mutations, spanner.Insert("Suppliers",
				[]string{"SupplierId", "Name", "Category", "Phone", "Email", "PasswordHash",
					"TaxId", "ContactPerson", "CompanyRegNumber", "BillingAddress",
					"IsConfigured", "OperatingCategories", "BankName", "AccountNumber",
					"CardNumber", "GlobalPayntGateway", "CreatedAt"},
				[]interface{}{sId, fmt.Sprintf("Simulator Supplier %d", s), "FMCG", fmt.Sprintf("+9989000000%02d", s),
					fmt.Sprintf("supplier%d@simulator.local", s), "sim_hash",
					"SIM-TAX-123", "John Sim", "REG-1234", "Tashkent SIM",
					true, []string{"CAT-1"}, "SimBank", "ACC123", "CARD123", "GLOBAL_PAY", spanner.CommitTimestamp}))

			for w := 1; w <= numWarehouses; w++ {
				whId := fmt.Sprintf("WH-SIM-%03d-%03d", s, w)
				mutations = append(mutations, spanner.Insert("Warehouses",
					[]string{"WarehouseId", "SupplierId", "Name", "LocationWKT", "CapacityVU", "SafetyStockDays", "CreatedAt"},
					[]interface{}{whId, sId, fmt.Sprintf("Sim Warehouse %d for Supp %d", w, s), "POINT(69.2401 41.2995)", int64(1000), int64(3), spanner.CommitTimestamp}))
			}

			fcId := fmt.Sprintf("FAC-SIM-%03d", s)
			mutations = append(mutations, spanner.Insert("Factories",
				[]string{"FactoryId", "SupplierId", "Name", "LocationWKT", "ProductionCapacityVU", "CreatedAt"},
				[]interface{}{fcId, sId, fmt.Sprintf("Sim Factory for Supp %d", s), "POINT(69.3000 41.3000)", int64(5000), spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("SupplierProducts",
				[]string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "SellByBlock", "UnitsPerBlock", "BasePrice", "PalletFootprint", "IsActive", "CategoryId", "CreatedAt"},
				[]interface{}{fmt.Sprintf("SKU-SIM-%03d", s), sId, "Sim Product A", "Desc", "url", true, int64(10), int64(5000), int64(10), true, "CAT-1", spanner.CommitTimestamp}))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Fatalf("Failed generating suppliers/nodes: %v", err)
	}

	log.Printf("[SIMULATE] Generated %d Suppliers with %d Warehouses each.\n", numSuppliers, numWarehouses)

	allRetailers := GenerateGeoClusters(numRetailers, rng)

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}
		for _, r := range allRetailers {
			mutations = append(mutations, spanner.Insert("Retailers",
				[]string{"RetailerId", "Name", "Phone", "ShopName", "ShopLocation", "H3Cell", "TaxIdentificationNumber", "Status", "PasswordHash", "CreatedAt"},
				[]interface{}{r.ID, r.Name, r.Phone, r.ShopName, r.LocationWKT, r.H3Cell, "TAX-SIM-" + r.ID, "VERIFIED", "sim_hash", spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("RetailerGlobalSettings",
				[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
				[]interface{}{r.ID, false, spanner.CommitTimestamp}))

			for _, sId := range allSupplierIds {
				mutations = append(mutations, spanner.Insert("RetailerSuppliers",
					[]string{"RetailerId", "SupplierId", "AddedAt"},
					[]interface{}{r.ID, sId, spanner.CommitTimestamp}))
			}
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Fatalf("Failed inserting retailers: %v", err)
	}

	log.Printf("[SIMULATE] Inserted %d geo-clustered Retailers.\n", len(allRetailers))

	InjectAIAndOrders(ctx, client, allRetailers, allSupplierIds, rng)
	log.Println("[SIMULATE] Injected AI states, predictions, and TrackingCodes for simulation.")
}

func nilStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
