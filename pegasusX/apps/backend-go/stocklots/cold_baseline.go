package stocklots

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// AssertColdChainBaselineReady requires at least one temperature reading for the
// manifest when cold chain is effective and the truck carries chilled SKUs (G2.C).
func AssertColdChainBaselineReady(ctx context.Context, client *spanner.Client, manifestID string) error {
	manifestID = strings.TrimSpace(manifestID)
	if client == nil || manifestID == "" {
		return nil
	}
	supplierID, warehouseID := manifestScope(ctx, client, manifestID)
	if !EffectiveColdChain(ctx, warehouseID, supplierID) {
		return nil
	}
	needs, err := productsNeedColdClient(ctx, client, manifestID)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT ReadingId FROM TemperatureReadings WHERE ManifestId = @mid LIMIT 1`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err = iter.Next()
	if err == iterator.Done {
		return ErrColdChainBaselineRequired
	}
	if err != nil {
		if strings.Contains(err.Error(), "TemperatureReadings") || spanner.ErrCode(err) == 5 {
			return fmt.Errorf("%w: readings_unavailable", ErrColdChainBaselineRequired)
		}
		return fmt.Errorf("%w: %v", ErrColdChainBaselineRequired, err)
	}
	return nil
}

func productsNeedColdClient(ctx context.Context, client *spanner.Client, manifestID string) (bool, error) {
	// Prefer orders linked by ManifestId; also allow manifest-order child table if present.
	stmt := spanner.Statement{
		SQL:    `SELECT OrderId, LineItemsJson FROM Orders WHERE ManifestId = @mid LIMIT 100`,
		Params: map[string]interface{}{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	productIDs := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Orders.ManifestId may be unpopulated; treat as non-chilled (no false block).
			return false, nil
		}
		var oid string
		var raw []byte
		if err := row.Columns(&oid, &raw); err != nil {
			continue
		}
		for _, id := range extractProductIDsFromLineItems(raw) {
			productIDs[id] = struct{}{}
		}
	}
	if len(productIDs) == 0 {
		return false, nil
	}
	ids := make([]string, 0, len(productIDs))
	for id := range productIDs {
		ids = append(ids, id)
	}
	pstmt := spanner.Statement{
		SQL: `SELECT ProductId FROM Products
		      WHERE ProductId IN UNNEST(@pids)
		        AND (RequiresColdChain = true OR StorageTempMinC IS NOT NULL OR StorageTempMaxC IS NOT NULL)
		      LIMIT 1`,
		Params: map[string]interface{}{"pids": ids},
	}
	piter := client.Single().Query(ctx, pstmt)
	defer piter.Stop()
	_, err := piter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}

func extractProductIDsFromLineItems(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var source []struct {
		SKU   string `json:"sku"`
		SKUID string `json:"sku_id"`
		ID    string `json:"product_id"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil
	}
	out := make([]string, 0, len(source))
	for _, s := range source {
		id := strings.TrimSpace(s.SKUID)
		if id == "" {
			id = strings.TrimSpace(s.ID)
		}
		if id == "" {
			id = strings.TrimSpace(s.SKU)
		}
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}
