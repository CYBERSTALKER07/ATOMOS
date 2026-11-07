package order

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
)

func TestStampTaxRegimeTxn_PackCountryNotCurrency(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	svc.SetTaxService(tax.NewService(tax.NewMemoryRepository(), nil, slog.Default()))
	err := svc.stampTaxRegimeTxn(context.Background(), nil, &Order{
		OrderID:    "o1",
		SupplierID: "s1",
		Currency:   "KZT",
		LineItems:  []LineItem{{SKU: "a", Quantity: 1, UnitPrice: 100}},
	})
	if err == nil || !strings.Contains(err.Error(), "country UZ") {
		t.Fatalf("err=%v want missing regime for pack UZ (not KZ from KZT)", err)
	}
}

func TestStampTaxRegimeTxn_PlannedFailsClosed(t *testing.T) {
	svc := NewService(ServiceConfig{})
	svc.SetTaxService(tax.NewService(tax.NewMemoryRepository(), nil, slog.Default()))
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "KZ", SupplierID: "s1"})
	err := svc.stampTaxRegimeTxn(ctx, nil, &Order{
		OrderID:    "o1",
		SupplierID: "s1",
		Currency:   "KZT",
		LineItems:  []LineItem{{SKU: "a", Quantity: 1, UnitPrice: 100}},
	})
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestStampTaxRegimeTxn_NilTaxServiceSkips(t *testing.T) {
	svc := NewService(ServiceConfig{})
	if err := svc.stampTaxRegimeTxn(context.Background(), nil, &Order{Currency: "KZT"}); err != nil {
		t.Fatalf("nil tax svc must skip: %v", err)
	}
}

type mockTaxRepo struct {
	tax.MemoryRepository
	regime    tax.TaxRegimeVersion
	snapshots []tax.OrderLineFiscalSnapshot
}

func (m *mockTaxRepo) GetActiveRegime(ctx context.Context, txn *spanner.ReadWriteTransaction, countryCode string, ts time.Time) (tax.TaxRegimeVersion, bool, error) {
	if countryCode == m.regime.CountryCode {
		return m.regime, true, nil
	}
	return tax.TaxRegimeVersion{}, false, nil
}

func (m *mockTaxRepo) InsertLineSnapshot(ctx context.Context, txn *spanner.ReadWriteTransaction, snapshot tax.OrderLineFiscalSnapshot) error {
	m.snapshots = append(m.snapshots, snapshot)
	return nil
}

func TestStampTaxRegimeTxn_LineItemSpecificVatAndHalfUpRounding(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	repo := &mockTaxRepo{
		regime: tax.TaxRegimeVersion{
			Id:          "reg_uz_2026",
			CountryCode: "UZ",
			VatRatesBps: []int64{1200}, // standard 12%
		},
	}
	svc := NewService(ServiceConfig{})
	svc.SetTaxService(tax.NewService(repo, nil, slog.Default()))

	zeroRate := int64(0)
	orderRecord := &Order{
		OrderID:    "ord_fiscal_round_1",
		SupplierID: "sup_1",
		Currency:   "UZS",
		LineItems: []LineItem{
			// Line 1: Standard, Net = 1255, Rate = 1200 bps -> Vat = (1255*1200 + 5000)/10000 = 151
			{SKU: "SKU-STD-ROUNDUP", Quantity: 1, UnitPrice: 1255},
			// Line 2: Standard, Net = 1254, Rate = 1200 bps -> Vat = (1254*1200 + 5000)/10000 = 150
			{SKU: "SKU-STD-ROUNDDOWN", Quantity: 1, UnitPrice: 1254},
			// Line 3: Exempt category -> Rate = 0, Vat = 0
			{SKU: "SKU-EXEMPT-BREAD", Quantity: 1, UnitPrice: 50000, TaxCategory: "EXEMPT"},
			// Line 4: Explicit zero rate -> Rate = 0, Vat = 0
			{SKU: "SKU-ZERO-EXPORT", Quantity: 1, UnitPrice: 80000, VatRateBps: &zeroRate},
		},
	}

	ctx := context.Background()
	if err := svc.stampTaxRegimeTxn(ctx, nil, orderRecord); err != nil {
		t.Fatalf("stampTaxRegimeTxn failed: %v", err)
	}

	if len(repo.snapshots) != 4 {
		t.Fatalf("expected 4 snapshots, got %d", len(repo.snapshots))
	}

	// Line 1:
	s1 := repo.snapshots[0]
	if s1.VatRateBps != 1200 || s1.NetMinor != 1255 || s1.VatMinor != 151 || s1.GrossMinor != 1406 {
		t.Fatalf("line 1 mismatch: vatRate=%d net=%d vat=%d gross=%d", s1.VatRateBps, s1.NetMinor, s1.VatMinor, s1.GrossMinor)
	}

	// Line 2:
	s2 := repo.snapshots[1]
	if s2.VatRateBps != 1200 || s2.NetMinor != 1254 || s2.VatMinor != 150 || s2.GrossMinor != 1404 {
		t.Fatalf("line 2 mismatch: vatRate=%d net=%d vat=%d gross=%d", s2.VatRateBps, s2.NetMinor, s2.VatMinor, s2.GrossMinor)
	}

	// Line 3:
	s3 := repo.snapshots[2]
	if s3.VatRateBps != 0 || s3.NetMinor != 50000 || s3.VatMinor != 0 || s3.GrossMinor != 50000 {
		t.Fatalf("line 3 mismatch: vatRate=%d net=%d vat=%d gross=%d", s3.VatRateBps, s3.NetMinor, s3.VatMinor, s3.GrossMinor)
	}

	// Line 4:
	s4 := repo.snapshots[3]
	if s4.VatRateBps != 0 || s4.NetMinor != 80000 || s4.VatMinor != 0 || s4.GrossMinor != 80000 {
		t.Fatalf("line 4 mismatch: vatRate=%d net=%d vat=%d gross=%d", s4.VatRateBps, s4.NetMinor, s4.VatMinor, s4.GrossMinor)
	}
}
