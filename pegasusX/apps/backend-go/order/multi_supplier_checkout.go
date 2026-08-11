package order

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	parentStatusPending   = "PENDING"
	parentStatusPartial   = "PARTIAL"
	parentStatusComplete  = "COMPLETE"
	parentStatusCancelled = "CANCELLED"
)

// MultiSupplierCheckoutEnabled gates ParentOrders + per-supplier Create split.
// Default on when PEGASUSX_ENV=ssmr; elsewhere require MULTI_SUPPLIER_CHECKOUT_ENABLED=true.
func MultiSupplierCheckoutEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("MULTI_SUPPLIER_CHECKOUT_ENABLED"))
	if raw == "" {
		return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "ssmr")
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// SupplierCheckoutFailure is one leg's failure in a multi-supplier checkout attempt.
type SupplierCheckoutFailure struct {
	SupplierID string `json:"supplier_id"`
	Error      string `json:"error"`
}

// MultiSupplierCheckoutError is returned when the all-or-nothing split aborts.
type MultiSupplierCheckoutError struct {
	Failures []SupplierCheckoutFailure `json:"supplier_errors"`
	Message  string                    `json:"error"`
}

func (e *MultiSupplierCheckoutError) Error() string {
	if e == nil {
		return "multi_supplier_checkout_failed"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if len(e.Failures) > 0 {
		return fmt.Sprintf("multi_supplier_checkout_failed: supplier %s: %s", e.Failures[0].SupplierID, e.Failures[0].Error)
	}
	return "multi_supplier_checkout_failed"
}

type checkoutLineGroup struct {
	SupplierID string
	Lines      []LineItem
	ItemCount  int
}

func (s *Service) groupCheckoutLinesBySupplier(
	ctx context.Context,
	reqItems []UnifiedCheckoutLineItem,
	lineItems []LineItem,
) ([]checkoutLineGroup, error) {
	if len(lineItems) == 0 {
		return nil, fmt.Errorf("items must not be empty")
	}

	hintBySKU := make(map[string]string, len(reqItems))
	for _, it := range reqItems {
		sku := strings.TrimSpace(it.SkuID)
		if sku == "" {
			continue
		}
		if sid := strings.TrimSpace(it.SupplierID); sid != "" {
			hintBySKU[sku] = sid
		}
	}

	catalogBySKU, err := s.loadProductSupplierIDs(ctx, lineItems)
	if err != nil {
		return nil, err
	}

	fallback := strings.TrimSpace(s.resolveSupplierScope(ctx))
	grouped := make(map[string][]LineItem)
	orderKeys := make([]string, 0)
	for i, line := range lineItems {
		sku := strings.TrimSpace(line.SKU)
		sid := strings.TrimSpace(catalogBySKU[sku])
		if sid == "" {
			sid = strings.TrimSpace(hintBySKU[sku])
		}
		if sid == "" {
			sid = fallback
		}
		if sid == "" {
			return nil, fmt.Errorf("items[%d] missing supplier_id for sku %s", i, sku)
		}
		if _, ok := grouped[sid]; !ok {
			orderKeys = append(orderKeys, sid)
		}
		grouped[sid] = append(grouped[sid], line)
	}
	sort.Strings(orderKeys)

	out := make([]checkoutLineGroup, 0, len(orderKeys))
	for _, sid := range orderKeys {
		lines := grouped[sid]
		out = append(out, checkoutLineGroup{
			SupplierID: sid,
			Lines:      lines,
			ItemCount:  len(lines),
		})
	}
	return out, nil
}

func (s *Service) loadProductSupplierIDs(ctx context.Context, lines []LineItem) (map[string]string, error) {
	out := make(map[string]string)
	if s.spannerClient == nil || len(lines) == 0 {
		return out, nil
	}
	keys := make([]spanner.Key, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, li := range lines {
		sku := strings.TrimSpace(li.SKU)
		if sku == "" {
			continue
		}
		if _, ok := seen[sku]; ok {
			continue
		}
		seen[sku] = struct{}{}
		keys = append(keys, spanner.Key{sku})
	}
	if len(keys) == 0 {
		return out, nil
	}
	iter := s.spannerClient.Single().Read(ctx, "Products", spanner.KeySetFromKeys(keys...), []string{"ProductId", "SupplierId"})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("load product supplier ids: %w", err)
		}
		var productID, supplierID string
		if err := row.Columns(&productID, &supplierID); err != nil {
			return nil, fmt.Errorf("scan product supplier id: %w", err)
		}
		out[productID] = strings.TrimSpace(supplierID)
	}
	return out, nil
}

func (s *Service) insertParentOrder(ctx context.Context, parentID, retailerID, currency string, childCount int) error {
	if s.spannerClient == nil {
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ParentOrders", map[string]any{
			"ParentOrderId": parentID,
			"RetailerId":    retailerID,
			"Status":        parentStatusPending,
			"Currency":      currency,
			"TotalMinor":    int64(0),
			"ChildCount":    int64(childCount),
			"CreatedAt":     spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		return fmt.Errorf("insert parent order: %w", err)
	}
	return nil
}

func (s *Service) updateParentOrderTotals(ctx context.Context, parentID, status, currency string, totalMinor int64, childCount int) error {
	if s.spannerClient == nil {
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("ParentOrders", map[string]any{
			"ParentOrderId": parentID,
			"Status":        status,
			"Currency":      currency,
			"TotalMinor":    totalMinor,
			"ChildCount":    int64(childCount),
			"UpdatedAt":     spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		return fmt.Errorf("update parent order: %w", err)
	}
	return nil
}

func (s *Service) compensateParentCheckout(ctx context.Context, parentID string, created []Order) {
	for i := range created {
		o := created[i]
		if s.repo != nil {
			if full, ok, err := s.repo.GetOrder(ctx, o.OrderID); err == nil && ok {
				o = full
			}
		}
		if _, err := s.cancelOrderWithReason(ctx, &o, "system", "SYSTEM", "multi_supplier_checkout_abort", ""); err != nil {
			s.log.Warn("compensate cancel failed", "order_id", o.OrderID, "parent_order_id", parentID, "err", err)
		}
	}
	_ = s.updateParentOrderTotals(ctx, parentID, parentStatusCancelled, s.currency, 0, len(created))
}

func createCtxForSupplier(ctx context.Context, supplierID string) context.Context {
	return auth.WithTenant(ctx, auth.TenantContext{
		SupplierID: strings.TrimSpace(supplierID),
		Source:     "multi_supplier_checkout",
	})
}
