package bootstrap

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"google.golang.org/api/iterator"
)

type migrationARAdapter struct {
	arSvc   *ar.Service
	spanner *spanner.Client
}

func (a *migrationARAdapter) ListOpeningBalances(ctx context.Context, retailerID string) ([]retailer.MigrationOpeningInvoice, error) {
	if a == nil || a.arSvc == nil {
		return nil, ar.ErrRepoUnavailable
	}
	invs, err := a.arSvc.ListMigrationOpeningBalances(ctx, retailerID)
	if err != nil {
		return nil, err
	}
	supplierNames := a.lookupSupplierNames(ctx, invs)
	out := make([]retailer.MigrationOpeningInvoice, 0, len(invs))
	for _, inv := range invs {
		out = append(out, mapARInvoiceToOpeningDTO(inv, supplierNames[inv.SupplierID]))
	}
	return out, nil
}

func (a *migrationARAdapter) AcknowledgeOpeningInvoice(ctx context.Context, retailerID, invoiceID string) (retailer.MigrationOpeningInvoice, error) {
	if a == nil || a.arSvc == nil {
		return retailer.MigrationOpeningInvoice{}, ar.ErrRepoUnavailable
	}
	inv, err := a.arSvc.AcknowledgeMigrationInvoice(ctx, retailerID, invoiceID)
	if err != nil {
		return retailer.MigrationOpeningInvoice{}, err
	}
	names := a.lookupSupplierNames(ctx, []ar.Invoice{inv})
	return mapARInvoiceToOpeningDTO(inv, names[inv.SupplierID]), nil
}

func (a *migrationARAdapter) DisputeOpeningInvoice(ctx context.Context, retailerID, invoiceID, reason string) (retailer.MigrationOpeningInvoice, error) {
	if a == nil || a.arSvc == nil {
		return retailer.MigrationOpeningInvoice{}, ar.ErrRepoUnavailable
	}
	inv, err := a.arSvc.DisputeMigrationInvoice(ctx, retailerID, invoiceID, reason)
	if err != nil {
		return retailer.MigrationOpeningInvoice{}, err
	}
	names := a.lookupSupplierNames(ctx, []ar.Invoice{inv})
	return mapARInvoiceToOpeningDTO(inv, names[inv.SupplierID]), nil
}

func (a *migrationARAdapter) lookupSupplierNames(ctx context.Context, invs []ar.Invoice) map[string]string {
	out := make(map[string]string)
	if a == nil || a.spanner == nil || len(invs) == 0 {
		return out
	}
	seen := make(map[string]struct{}, len(invs))
	ids := make([]string, 0, len(invs))
	for _, inv := range invs {
		sid := strings.TrimSpace(inv.SupplierID)
		if sid == "" {
			continue
		}
		if _, ok := seen[sid]; ok {
			continue
		}
		seen[sid] = struct{}{}
		ids = append(ids, sid)
	}
	if len(ids) == 0 {
		return out
	}
	iter := a.spanner.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(ctx, spanner.Statement{
		SQL:    `SELECT SupplierId, Name FROM Suppliers WHERE SupplierId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": ids},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done || err != nil {
			break
		}
		var id, name string
		if err := row.Columns(&id, &name); err == nil {
			out[id] = name
		}
	}
	return out
}

func mapARInvoiceToOpeningDTO(inv ar.Invoice, supplierName string) retailer.MigrationOpeningInvoice {
	dto := retailer.MigrationOpeningInvoice{
		InvoiceID:      inv.InvoiceID,
		SupplierID:     inv.SupplierID,
		SupplierName:   supplierName,
		RetailerID:     inv.RetailerID,
		OrderID:        inv.OrderID,
		ExternalID:     inv.ExternalID,
		PrincipalMinor: inv.PrincipalMinor,
		BalanceMinor:   inv.BalanceMinor,
		Currency:       inv.Currency,
		Status:         inv.Status,
		Source:         inv.Source,
		DunningFrozen:  inv.IsMigrationDunningFrozen(),
		DisputeReason:  inv.DisputeReason,
	}
	if !inv.DueAt.IsZero() {
		dto.DueAt = inv.DueAt.UTC().Format(time.RFC3339)
	}
	if inv.AcknowledgedAt != nil && !inv.AcknowledgedAt.IsZero() {
		dto.AcknowledgedAt = inv.AcknowledgedAt.UTC().Format(time.RFC3339)
	}
	if inv.DisputedAt != nil && !inv.DisputedAt.IsZero() {
		dto.DisputedAt = inv.DisputedAt.UTC().Format(time.RFC3339)
	}
	if !inv.UpdatedAt.IsZero() {
		dto.UpdatedAt = inv.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return dto
}
