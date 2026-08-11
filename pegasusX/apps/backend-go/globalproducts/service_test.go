package globalproducts

import (
	"context"
	"testing"
)

func TestMatchAndLink_ExactGTIN(t *testing.T) {
	t.Setenv("GLOBAL_PRODUCTS_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	in1 := ProductInput{
		ProductID: "p1", SupplierID: "s1", Name: "Coca Cola 0.5L", Brand: "CocaCola",
		Barcode: "4006381333931", PriceMinor: 5000, Currency: "UZS", UnitsPerPack: 1,
	}
	res1, err := svc.MatchAndLink(context.Background(), in1)
	if err != nil {
		t.Fatal(err)
	}
	if !res1.Created || res1.GlobalProductID == "" {
		t.Fatalf("want created global, got %+v", res1)
	}

	in2 := ProductInput{
		ProductID: "p2", SupplierID: "s2", Name: "Coke 0.5", Brand: "Other",
		Barcode: "4006381333931", PriceMinor: 4800, Currency: "UZS", UnitsPerPack: 1,
	}
	res2, err := svc.MatchAndLink(context.Background(), in2)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Created || res2.Method != MethodExactGTIN {
		t.Fatalf("want exact link, got %+v", res2)
	}
	if res2.GlobalProductID != res1.GlobalProductID {
		t.Fatalf("gtin link mismatch %s vs %s", res1.GlobalProductID, res2.GlobalProductID)
	}
	offers, err := svc.ListOffers(context.Background(), res1.GlobalProductID)
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 2 {
		t.Fatalf("want 2 offers, got %d", len(offers))
	}
}

func TestMatchAndLink_FuzzyQueueAmbiguous(t *testing.T) {
	t.Setenv("GLOBAL_PRODUCTS_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Seed two similar globals manually so a third SKU is ambiguous.
	_ = repo.UpsertGlobal(context.Background(), GlobalProduct{
		GlobalProductID: "gp-a", Brand: "Acme", Name: "Widget Blue", PackQty: 12, BaseUomID: UomEachID,
		NormalizedKey: BuildNormalizedKey("Acme", "Widget Blue", 12, "EACH"),
	})
	_ = repo.UpsertGlobal(context.Background(), GlobalProduct{
		GlobalProductID: "gp-b", Brand: "Acme", Name: "Widget Blue Extra", PackQty: 12, BaseUomID: UomEachID,
		NormalizedKey: BuildNormalizedKey("Acme", "Widget Blue Extra", 12, "EACH"),
	})

	res, err := svc.MatchAndLink(context.Background(), ProductInput{
		ProductID: "p3", SupplierID: "s3", Name: "Widget Blue", Brand: "Acme",
		PriceMinor: 1000, Currency: "UZS", UnitsPerPack: 12, UomCode: "EACH",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Queued || len(res.QueueIDs) < 1 {
		t.Fatalf("want queued fuzzy, got %+v", res)
	}
	q, err := svc.ListMatchQueue(context.Background(), StatusPending, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(q) == 0 {
		t.Fatal("expected pending queue items")
	}
}

func TestMatchAndLink_InvalidGTINSkipsExact(t *testing.T) {
	t.Setenv("GLOBAL_PRODUCTS_ENABLED", "true")
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	res, err := svc.MatchAndLink(context.Background(), ProductInput{
		ProductID: "p4", SupplierID: "s4", Name: "Odd SKU", Brand: "Odd",
		Barcode: "not-a-gtin", PriceMinor: 100, Currency: "UZS", UnitsPerPack: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created {
		t.Fatalf("invalid gtin should still create master without exact path: %+v", res)
	}
}

func TestDecideFuzzy_SingleStrongAutoLinks(t *testing.T) {
	auto, queue := DecideFuzzy([]scoredCandidate{{GlobalProductID: "g1", Score: 0.9}})
	if auto == nil || queue != nil {
		t.Fatalf("auto=%v queue=%v", auto, queue)
	}
}

func TestEnabledFlag(t *testing.T) {
	t.Setenv("GLOBAL_PRODUCTS_ENABLED", "")
	if Enabled() {
		t.Fatal("want disabled")
	}
	t.Setenv("GLOBAL_PRODUCTS_ENABLED", "true")
	if !Enabled() {
		t.Fatal("want enabled")
	}
}
