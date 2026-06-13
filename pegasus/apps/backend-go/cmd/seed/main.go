package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
)

// CanonicalMatrix defines the immutable test state
var CanonicalMatrix = struct {
	Retailers []map[string]interface{}
	Suppliers []map[string]interface{}
	Products  []map[string]interface{}
}{
	Retailers: []map[string]interface{}{
		{"RetailerId": "RET-TASH-CENTRAL", "Name": "Tashkent Center Mart", "H3Index": "873112b29ffffff", "AccessType": "PUBLIC"},
		{"RetailerId": "RET-SAM-REGISTAN", "Name": "Samarkand Plaza", "H3Index": "873112b20ffffff", "AccessType": "PUBLIC"},
	},
	Suppliers: []map[string]interface{}{
		{"SupplierId": "SUP-BEV-001", "Name": "Global Beverages Ltd", "CreatedAt": spanner.CommitTimestamp},
		{"SupplierId": "SUP-FROZEN-002", "Name": "Arctic Distribution", "CreatedAt": spanner.CommitTimestamp},
	},
	Products: []map[string]interface{}{
		{"SkuId": "FANTA-CAN-24", "SupplierId": "SUP-BEV-001", "Name": "Fanta 24-can", "SellByBlock": true, "UnitsPerBlock": int64(24), "BasePrice": int64(120000), "VolumetricUnit": 0.05, "MinimumOrderQty": int64(1), "StepSize": int64(1), "IsActive": true, "CreatedAt": spanner.CommitTimestamp},
		{"SkuId": "COKE-CAN-24", "SupplierId": "SUP-BEV-001", "Name": "Coke 24-can", "SellByBlock": true, "UnitsPerBlock": int64(24), "BasePrice": int64(115000), "VolumetricUnit": 0.05, "MinimumOrderQty": int64(1), "StepSize": int64(1), "IsActive": true, "CreatedAt": spanner.CommitTimestamp},
		{"SkuId": "ICE-CREAM-BOX", "SupplierId": "SUP-FROZEN-002", "Name": "Ice Cream Box", "SellByBlock": true, "UnitsPerBlock": int64(1), "BasePrice": int64(450000), "VolumetricUnit": 0.12, "MinimumOrderQty": int64(1), "StepSize": int64(1), "IsActive": true, "CreatedAt": spanner.CommitTimestamp},
	},
}

func main() {
	ctx := context.Background()
	db := "projects/v-o-i-d/instances/dev-instance/databases/atomos"
	client, err := spanner.NewClient(ctx, db, option.WithEndpoint("localhost:9010"), option.WithoutAuthentication())
	if err != nil {
		log.Fatalf("Failed to connect to Spanner: %v", err)
	}
	defer client.Close()

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Seed Retailers
		for _, r := range CanonicalMatrix.Retailers {
			m := spanner.InsertOrUpdateMap("Retailers", r)
			if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
				return err
			}
		}

		// 2. Seed Suppliers
		for _, s := range CanonicalMatrix.Suppliers {
			m := spanner.InsertOrUpdateMap("Suppliers", s)
			if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
				return err
			}
		}

		// 3. Seed Products (SKUs)
		for _, p := range CanonicalMatrix.Products {
			m := spanner.InsertOrUpdateMap("SupplierProducts", p)
			if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	fmt.Println("V.O.I.D. Canonical State Seeded Successfully.")
}
