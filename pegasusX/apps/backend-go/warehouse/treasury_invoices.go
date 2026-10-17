package warehouse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const warehouseTreasuryInvoiceLimit = 200

type treasuryInvoiceRow struct {
	InvoiceID      string
	OrderID        string
	RetailerID     string
	Status         string
	PrincipalMinor int64
	BalanceMinor   int64
	Currency       string
	DueAt          time.Time
}

func (s *Service) loadWarehouseTreasuryInvoices(ctx context.Context, warehouseID, supplierID string) ([]map[string]any, error) {
	if s == nil || s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	supplierID = strings.TrimSpace(supplierID)
	if warehouseID == "" {
		return nil, fmt.Errorf("warehouse_id required")
	}
	if supplierID == "" {
		return nil, fmt.Errorf("supplier_id required")
	}

	readCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	stmt := spanner.Statement{
		SQL: `SELECT i.InvoiceId, i.OrderId, i.RetailerId, i.Status, i.PrincipalMinor,
		             i.BalanceMinor, i.Currency, i.DueAt
		      FROM ArInvoices@{FORCE_INDEX=Idx_ArInvoices_BySupplierDue} i
		      JOIN Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated} o
		        ON o.OrderId = i.OrderId
		      WHERE i.SupplierId = @sid AND o.WarehouseId = @wh
		      ORDER BY i.DueAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"sid": supplierID,
			"wh":  warehouseID,
			"lim": int64(warehouseTreasuryInvoiceLimit),
		},
	}
	iter := s.spannerClient.Single().WithTimestampBound(spanner.MaxStaleness(15*time.Second)).Query(readCtx, stmt)
	defer iter.Stop()

	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("treasury invoices query: %w", err)
		}
		var rec treasuryInvoiceRow
		if err := row.Columns(
			&rec.InvoiceID, &rec.OrderID, &rec.RetailerID, &rec.Status,
			&rec.PrincipalMinor, &rec.BalanceMinor, &rec.Currency, &rec.DueAt,
		); err != nil {
			return nil, fmt.Errorf("treasury invoices scan: %w", err)
		}
		out = append(out, map[string]any{
			"invoice_id":      rec.InvoiceID,
			"order_id":        rec.OrderID,
			"retailer_id":     rec.RetailerID,
			"status":          rec.Status,
			"principal_minor": rec.PrincipalMinor,
			"balance_minor":   rec.BalanceMinor,
			"currency":        rec.Currency,
			"due_at":          rec.DueAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func writeTreasuryInvoicesUnavailable(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "invoices_unavailable",
		"message": err.Error(),
	})
}
