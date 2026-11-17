package credit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestCredit_UpsertAndEnablePreserveCurrentBalance_Emulator(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	retailerID := fmt.Sprintf("ret-bal-pres-%d", suffix)
	supplierID := fmt.Sprintf("sup-bal-pres-%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)

	// Seed active profile carrying an unpaid CurrentBalanceMinor = 75,000.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":           retailerID,
			"SupplierId":           supplierID,
			"CreditLimitMinor":     int64(200000),
			"CurrentBalanceMinor":  int64(75000),
			"ReservedMinor":        int64(0),
			"AvailableCreditMinor": int64(125000),
			"Status":               string(StatusActive),
			"RiskScore":            int64(80),
			"DelinquencyCount":     int64(0),
			"Version":              int64(1),
			"LastEvaluatedAt":      now,
			"CreatedAt":            now,
			"UpdatedAt":            now,
		}),
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	repo := NewSpannerRepository(client)
	// Raise credit limit via UpsertProfile passing zeroed CurrentBalanceMinor -> must preserve 75,000.
	err := repo.UpsertProfile(ctx, Profile{
		RetailerID:          retailerID,
		SupplierID:          supplierID,
		CreditLimitMinor:    300000,
		CurrentBalanceMinor: 0,
		Status:              StatusActive,
		RiskScore:           80,
		LastEvaluatedAt:     now,
	}, nil)
	if err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}

	got, found, err := repo.GetProfile(ctx, retailerID, supplierID)
	if err != nil || !found {
		t.Fatalf("GetProfile found=%v err=%v", found, err)
	}
	if got.CurrentBalanceMinor != 75000 {
		t.Fatalf("CurrentBalanceMinor=%d, want 75000 preserved", got.CurrentBalanceMinor)
	}
	if got.AvailableCreditMinor != 225000 {
		t.Fatalf("AvailableCreditMinor=%d, want 225000 (300000 - 75000)", got.AvailableCreditMinor)
	}
}
