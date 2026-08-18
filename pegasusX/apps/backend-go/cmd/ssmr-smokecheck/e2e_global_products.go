package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/globalproducts"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runGlobalProductsSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	if !envTruthy("GLOBAL_PRODUCTS_ENABLED") {
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_GTIN_LINK_SKIPPED")
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE_SKIPPED")
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE_SKIPPED")
		return nil
	}
	_ = os.Setenv("GLOBAL_PRODUCTS_ENABLED", "true")

	client, err := openGlobalProductsSpanner(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	repo := globalproducts.NewSpannerRepository(client)
	svc := globalproducts.NewService(repo, nil)
	if err := svc.EnsureBootstrap(ctx); err != nil {
		return fmt.Errorf("uom bootstrap: %w", err)
	}

	gtin := "4006381333931"
	suffix := uuid.NewString()[:8]
	s1, s2 := "gp-sup-a-"+suffix, "gp-sup-b-"+suffix
	p1, p2 := "gp-prod-a-"+suffix, "gp-prod-b-"+suffix
	op := smokeOperatingCurrency(ctx, cfg.SeedSupplierCurrency)
	if op == "" {
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_GTIN_LINK_SKIPPED")
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE_SKIPPED")
		fmt.Println("PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE_SKIPPED")
		return nil
	}

	res1, err := svc.MatchAndLink(ctx, globalproducts.ProductInput{
		ProductID: p1, SupplierID: s1, Name: "Cola 0.5L", Brand: "ColaBrand",
		Barcode: gtin, PriceMinor: 5000, Currency: op, UnitsPerPack: 1,
	})
	if err != nil {
		return fmt.Errorf("gtin link a: %w", err)
	}
	res2, err := svc.MatchAndLink(ctx, globalproducts.ProductInput{
		ProductID: p2, SupplierID: s2, Name: "Cola Half Liter", Brand: "Other",
		Barcode: gtin, PriceMinor: 4800, Currency: op, UnitsPerPack: 1,
	})
	if err != nil {
		return fmt.Errorf("gtin link b: %w", err)
	}
	if res2.GlobalProductID == "" || res2.GlobalProductID != res1.GlobalProductID || res2.Method != globalproducts.MethodExactGTIN {
		return fmt.Errorf("gtin link mismatch: a=%+v b=%+v", res1, res2)
	}
	fmt.Println("PX_E2E_GLOBAL_PRODUCT_GTIN_LINK_OK")

	// Ambiguous fuzzy: seed two similar masters then match a third SKU without GTIN.
	keyBrand := "FuzzyBrand" + suffix
	_ = repo.UpsertGlobal(ctx, globalproducts.GlobalProduct{
		GlobalProductID: "gp-fz-a-" + suffix,
		Brand:           keyBrand,
		Name:            "Widget Blue",
		PackQty:         12,
		BaseUomID:       globalproducts.UomEachID,
		NormalizedKey:   globalproducts.BuildNormalizedKey(keyBrand, "Widget Blue", 12, "EACH"),
		Version:         1,
	})
	_ = repo.UpsertGlobal(ctx, globalproducts.GlobalProduct{
		GlobalProductID: "gp-fz-b-" + suffix,
		Brand:           keyBrand,
		Name:            "Widget Blue Extra",
		PackQty:         12,
		BaseUomID:       globalproducts.UomEachID,
		NormalizedKey:   globalproducts.BuildNormalizedKey(keyBrand, "Widget Blue Extra", 12, "EACH"),
		Version:         1,
	})
	fres, err := svc.MatchAndLink(ctx, globalproducts.ProductInput{
		ProductID: "gp-prod-fz-" + suffix, SupplierID: s1,
		Name: "Widget Blue", Brand: keyBrand, PriceMinor: 1000, Currency: op,
		UnitsPerPack: 12, UomCode: "EACH",
	})
	if err != nil {
		return fmt.Errorf("fuzzy queue: %w", err)
	}
	if !fres.Queued {
		return fmt.Errorf("expected fuzzy queue, got %+v", fres)
	}
	fmt.Println("PX_E2E_GLOBAL_PRODUCT_FUZZY_QUEUE_OK")

	offers, err := svc.ListOffers(ctx, res1.GlobalProductID)
	if err != nil {
		return fmt.Errorf("offers: %w", err)
	}
	if len(offers) < 2 {
		return fmt.Errorf("want >=2 offers for compare, got %d", len(offers))
	}
	fmt.Println("PX_E2E_GLOBAL_PRODUCT_OFFERS_COMPARE_OK")
	return nil
}

func openGlobalProductsSpanner(ctx context.Context, cfg *bootstrap.Config) (*spanner.Client, error) {
	project := strings.TrimSpace(cfg.SpannerProject)
	instance := strings.TrimSpace(cfg.SpannerInstance)
	database := strings.TrimSpace(cfg.SpannerDatabase)
	if project == "" {
		project = envOr("SPANNER_PROJECT", "pegasusx-local")
	}
	if instance == "" {
		instance = envOr("SPANNER_INSTANCE", "pegasusx-instance")
	}
	if database == "" {
		database = envOr("SPANNER_DATABASE", "pegasusx-db")
	}
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		return spanner.NewClient(ctx, db)
	}
	return spanner.NewClient(ctx, db,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
}
