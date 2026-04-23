package main
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
	for _, arg := range os.Args {
		if arg == "-clear" {
			clearFlag = true
		}
	}

	if clearFlag {
		log.Println("[SIMULATE] -clear flag provided. Truncating tables...")
		truncateTables(ctx, spannerClient)
	}

	log.Println("[SIMULATE] Starting deterministic injection...")
	seedSimulation(ctx, spannerClient)
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

func seedSimulation(ctx context.Context, client *spanner.Client) {
	rng := rand.New(rand.NewSource(42)) // Deterministic random seed

	supplierId1 := "SUP-SIM-001"
	supplierId2 := "SUP-SIM-002"
	wh1 := "WH-SIM-001"
	fc1 := "FAC-SIM-001"
	
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}

		for _, sId := range []string{supplierId1, supplierId2} {
			mutations = append(mutations, spanner.Insert("Suppliers",
				[]string{"SupplierId", "Name", "Category", "Phone", "Email", "PasswordHash",
					"TaxId", "ContactPerson", "CompanyRegNumber", "BillingAddress",
					"IsConfigured", "OperatingCategories", "BankName", "AccountNumber",
					"CardNumber", "PaymentGateway", "CreatedAt"},
				[]interface{}{sId, fmt.Sprintf("Simulator Supplier %s", sId), "FMCG", "+998900000000",
					fmt.Sprintf("%s@simulator.local", strings.ToLower(sId)), "sim_hash",
					"SIM-TAX-123", "John Sim", "REG-1234", "Tashkent SIM",
					true, []string{"CAT-1"}, "SimBank", "ACC123", "CARD123", "PAYME", spanner.CommitTimestamp}))
		}

		mutations = append(mutations, spanner.Insert("Warehouses", 
			[]string{"WarehouseId", "SupplierId", "Name", "LocationWKT", "CapacityVU", "SafetyStockDays", "CreatedAt"},
			[]interface{}{wh1, supplierId1, "Sim Warehouse 1", "POINT(69.2401 41.2995)", int64(1000), int64(3), spanner.CommitTimestamp}))
		mutations = append(mutations, spanner.Insert("Factories", 
			[]string{"FactoryId", "SupplierId", "Name", "LocationWKT", "ProductionCapacityVU", "CreatedAt"},
			[]interface{}{fc1, supplierId1, "Sim Factory 1", "POINT(69.3000 41.3000)", int64(5000), spanner.CommitTimestamp}))
		mutations = append(mutations, spanner.Insert("SupplierProducts",
			[]string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "SellByBlock", "UnitsPerBlock", "BasePrice", "PalletFootprint", "IsActive", "CategoryId", "CreatedAt"},
			[]interface{}{"SKU-SIM-001", supplierId1, "Sim Product A", "Desc", "url", true, int64(10), int64(5000), int64(10), true, "CAT-1", spanner.CommitTimestamp}))

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Fatalf("Failed generating suppliers/nodes: %v", err)
	}

	log.Println("[SIMULATE] Generated Target Suppliers, Factories, and Warehouses.")

	allRetailers := GenerateGeoClusters(120, rng)
	
	// Spanner mutation limit is 20k, we batch manually if needed, but 120 retailers is fine.
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}
		for _, r := range allRetailers {
			mutations = append(mutations, spanner.Insert("Retailers",
				[]string{"RetailerId", "Name", "Phone", "ShopName", "ShopLocation", "H3Cell", "TaxIdentificationNumber", "Status", "PasswordHash", "CreatedAt"},
				[]interface{}{r.ID, r.Name, r.Phone, r.ShopName, r.LocationWKT, r.H3Cell, "TAX-SIM-" + r.ID, "VERIFIED", "sim_hash", spanner.CommitTimestamp}))
			
			mutations = append(mutations, spanner.Insert("RetailerGlobalSettings",
				[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
				[]interface{}{r.ID, false, spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("RetailerSuppliers",
				[]string{"RetailerId", "SupplierId", "AddedAt"},
				[]interface{}{r.ID, supplierId1, spanner.CommitTimestamp}))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Fatalf("Failed inserting retailers: %v", err)
	}
	
	log.Printf("[SIMULATE] Inserted %d geo-clustered Retailers.\n", len(allRetailers))

	InjectAIAndOrders(ctx, client, allRetailers, supplierId1, rng)
	log.Println("[SIMULATE] Injected AI states, predictions, and TrackingCodes for simulation.")
}

func nilStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
