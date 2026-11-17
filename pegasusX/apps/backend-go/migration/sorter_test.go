package migration

import (
	"fmt"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

func seqIDGen(prefix string) func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("%s-%d", prefix, n)
	}
}

func TestClassifyAndSplitBundle_CompositeAndDAGOrder(t *testing.T) {
	inputs := []ArtifactStreamsInput{
		{
			Artifact: Artifact{SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-1", ArtifactID: "art-1", FileName: "erp_dump.xlsx"},
			Streams: []parser.ParsedStream{
				{
					StreamName: "Tovar_Ostatok_Narx",
					Format:     parser.FormatXLSX,
					Encoding:   "utf-8",
					Headers:    []string{"Артикул", "Наименование товара", "Оптовая цена", "Склад", "Остаток на складе"},
					TotalRows:  120,
				},
				{
					StreamName: "Prodaji",
					Format:     parser.FormatXLSX,
					Encoding:   "utf-8",
					Headers:    []string{"Клиент", "ИНН", "Телефон", "Номер заказа", "Дата", "Дата доставки", "Сумма", "Статус"},
					TotalRows:  45,
					SampleRows: []parser.ParsedRow{
						{RowIndex: 0, Fields: map[string]any{"Клиент": "Baraka LLC", "ИНН": "301234567", "Телефон": "+998901234567", "Статус": "DELIVERED"}},
						{RowIndex: 1, Fields: map[string]any{"Клиент": "Nur Savdo", "ИНН": "307654321", "Телефон": "+998911112233", "Статус": "OPEN"}},
					},
				},
				{
					StreamName: "Nasiya_Qarzlar",
					Format:     parser.FormatXLSX,
					Encoding:   "utf-8",
					Headers:    []string{"Mijoz", "STIR", "Nasiya", "To'lov muddati"},
					TotalRows:  12,
				},
				{
					StreamName: "Omborlar",
					Format:     parser.FormatXLSX,
					Encoding:   "utf-8",
					Headers:    []string{"Ombor kodi", "Ombor nomi", "Manzil", "Shahar"},
					TotalRows:  3,
				},
			},
		},
	}

	dec, err := ClassifyAndSplitBundle(inputs, nil, nil, seqIDGen("sess"))
	if err != nil {
		t.Fatalf("ClassifyAndSplitBundle: %v", err)
	}
	if dec.BundleStatus != BundleStatusSorted || dec.Summary.NeedsReview {
		t.Fatalf("expected SORTED without review, got status=%s summary=%+v", dec.BundleStatus, dec.Summary)
	}

	gotPacks := make([]Pack, 0, len(dec.Sessions))
	for _, s := range dec.Sessions {
		gotPacks = append(gotPacks, s.Pack)
	}
	// Expected topological order across composite + pure sheets:
	// warehouses -> products_prices -> customers -> stock -> historical_orders -> nasiya_ar -> open_orders
	wantOrder := []Pack{
		PackWarehouses,
		PackProductsPrices,
		PackCustomers,
		PackStock,
		PackHistoricalOrders,
		PackNasiyaAR,
		PackOpenOrders,
	}
	if len(gotPacks) != len(wantOrder) {
		t.Fatalf("expected %d child sessions (%v), got %d (%v)", len(wantOrder), wantOrder, len(gotPacks), gotPacks)
	}
	for i := range wantOrder {
		if gotPacks[i] != wantOrder[i] {
			t.Fatalf("session %d expected pack %s, got %s (full: %v)", i, wantOrder[i], gotPacks[i], gotPacks)
		}
	}
}

func TestClassifyAndSplitBundle_ConflictsAndOverrides(t *testing.T) {
	inputs := []ArtifactStreamsInput{
		{
			Artifact: Artifact{SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-2", ArtifactID: "art-stock", FileName: "stock.csv"},
			Streams: []parser.ParsedStream{
				{
					StreamName:  "stock.csv",
					Format:      parser.FormatCSV,
					Encoding:    "cp1251",
					NeedsEncode: true,
					Headers:     []string{"SKU", "Warehouse", "Остаток"},
					TotalRows:   10,
				},
			},
		},
		{
			Artifact: Artifact{SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-2", ArtifactID: "art-lots", FileName: "lots.xlsx"},
			Streams: []parser.ParsedStream{
				{
					StreamName: "Batches",
					Format:     parser.FormatXLSX,
					Encoding:   "utf-8",
					Headers:    []string{"Артикул", "Партия", "Срок годности", "Количество"},
					TotalRows:  15,
				},
			},
		},
	}

	dec, err := ClassifyAndSplitBundle(inputs, nil, nil, seqIDGen("s"))
	if err != nil {
		t.Fatalf("ClassifyAndSplitBundle: %v", err)
	}
	if dec.BundleStatus != BundleStatusNeedsReview || !dec.Summary.ConflictingInventoryMode || !dec.Summary.NeedsReview {
		t.Fatalf("expected NEEDS_REVIEW with conflicting stock+lots + encoding ambiguity, got %+v", dec.Summary)
	}

	// Apply user override resolving inventory mode to PackLots and explicit cp1251 encoding.
	override := &BundleSortOverrideRequest{
		ResolvedInventoryMode: PackLots,
		Overrides: []StreamOverrideItem{
			{ArtifactID: "art-stock", StreamKey: "stock.csv", Encoding: "cp1251", Packs: []Pack{PackLots}, Confirmed: true},
		},
		ConfirmReview: true,
	}
	resolved, err := ClassifyAndSplitBundle(inputs, override, dec.Sessions, seqIDGen("s2"))
	if err != nil {
		t.Fatalf("ClassifyAndSplitBundle override: %v", err)
	}
	if resolved.BundleStatus != BundleStatusSorted || resolved.Summary.ConflictingInventoryMode {
		t.Fatalf("expected conflict cleared after override, got status=%s summary=%+v", resolved.BundleStatus, resolved.Summary)
	}
	for _, s := range resolved.Sessions {
		if s.Pack != PackLots {
			t.Fatalf("expected all sessions resolved to lots, got %s", s.Pack)
		}
	}
}
