// seed-demo-scope upserts SSMR demo catalog, inventory, and price-list rows for smoke/e2e.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"google.golang.org/api/iterator"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase,
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "spanner client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	supplierID, err := seedSupplierID(ctx, client, cfg.SeedSupplierName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed supplier: %v\n", err)
		os.Exit(1)
	}

	if err := auth.EnsureDemoScopeLinks(ctx, client, supplierID); err != nil {
		fmt.Fprintf(os.Stderr, "ensure demo scope: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("seed-demo-scope-ok supplier_id=" + supplierID)
}

func seedSupplierID(ctx context.Context, client *spanner.Client, name string) (string, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT SupplierId FROM Suppliers WHERE Name = @name LIMIT 1",
		Params: map[string]any{"name": name},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", fmt.Errorf("supplier %q not found", name)
	}
	if err != nil {
		return "", err
	}
	var id string
	if err := row.ColumnByName("SupplierId", &id); err != nil {
		return "", err
	}
	return id, nil
}
