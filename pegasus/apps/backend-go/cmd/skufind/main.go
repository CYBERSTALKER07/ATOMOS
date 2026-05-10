package main

import (
	"context"
	"fmt"
	"log"
	
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db")
	if err != nil { log.Fatal(err) }
	
	iter := client.Single().Query(ctx, spanner.Statement{SQL: "SELECT SupplierId, SkuId, BasePrice FROM SupplierProducts LIMIT 1"})
	defer iter.Stop()
	
	var supp, sku string
	var price int64
	for {
		row, err := iter.Next()
		if err == iterator.Done { break }
		if err != nil { log.Fatal(err) }
		if err := row.Columns(&supp, &sku, &price); err != nil { log.Fatal(err) }
		fmt.Printf("Supplier: %s, SkuId: %s, BasePrice: %d\n", supp, sku, price)
		break
	}
}
