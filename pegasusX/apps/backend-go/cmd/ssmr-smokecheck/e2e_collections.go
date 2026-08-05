package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runCollectionsDunningE2E seeds an overdue AR invoice, runs dunning once, asserts step + hold + delinquency.
func runCollectionsDunningE2E(
	ctx context.Context,
	client *http.Client,
	base, _ /* supplierCookie */, supplierID, retailerID string,
	cfg *bootstrap.Config,
) error {
	if !envTruthy("AR_INVOICES_ENABLED") || !envTruthy("AR_DUNNING_ENABLED") {
		fmt.Println("PX_E2E_COLLECTIONS_DUNNING_SKIPPED")
		return nil
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_COLLECTIONS_DUNNING_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	invoiceID := "ari-e2e-" + uuid.NewString()
	orderID := "ord-dunning-" + uuid.NewString()
	now := time.Now().UTC()
	dueAt := now.Add(-22 * 24 * time.Hour)
	_, err = spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": invoiceID, "SupplierId": supplierID, "RetailerId": retailerID, "OrderId": orderID,
			"Status": "OPEN", "PrincipalMinor": int64(50_000), "BalanceMinor": int64(50_000), "Currency": "UZS",
			"CreditLeaveAt": dueAt.Add(-30 * 24 * time.Hour), "DueAt": dueAt, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "AgingBucket": "1_30", "DunningStep": int64(0), "Version": int64(1),
			"CreatedAt": now, "UpdatedAt": now,
		}),
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Table not found") {
			fmt.Println("PX_E2E_COLLECTIONS_DUNNING_SKIPPED")
			return nil
		}
		return fmt.Errorf("seed ar invoice: %w", err)
	}

	adminToken, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-collections-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{
		Secret: cfg.JWTSecret,
		Issuer: cfg.JWTIssuer,
		TTL:    30 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("issue admin jwt: %w", err)
	}

	status, body, _, err := clientPost(ctx, client, base+"/v1/admin/ar/dunning/run-once", []byte("{}"), adminToken, "dunning-run-"+invoiceID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("dunning run-once status=%d body=%s", status, string(body))
	}

	row, err := spannerClient.Single().ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{"DunningStep", "AgingBucket"})
	if err != nil {
		return fmt.Errorf("read invoice after dunning: %w", err)
	}
	var step int64
	var bucket string
	if err := row.Columns(&step, &bucket); err != nil {
		return err
	}
	if step < 5 { // CREDIT_HOLD
		return fmt.Errorf("expected dunning step >= 5 (CREDIT_HOLD), got %d bucket=%s", step, bucket)
	}

	prof, err := spannerClient.Single().ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID},
		[]string{"DelinquencyCount", "Status"})
	if err != nil {
		return fmt.Errorf("read credit profile: %w", err)
	}
	var delinq int64
	var st string
	if err := prof.Columns(&delinq, &st); err != nil {
		return err
	}
	if delinq < 1 {
		return fmt.Errorf("expected DelinquencyCount >= 1, got %d", delinq)
	}
	if !strings.EqualFold(st, "FROZEN") {
		return fmt.Errorf("expected profile FROZEN after CREDIT_HOLD, got %s", st)
	}

	fmt.Println("PX_E2E_COLLECTIONS_DUNNING_OK")
	return nil
}
