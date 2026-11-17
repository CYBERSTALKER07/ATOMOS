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
// manifest when cold chain is effective and the truck carries chilled/frozen SKUs (G2.C).
func AssertColdChainBaselineReady(ctx context.Context, client *spanner.Client, manifestID string) error {
	manifestID = strings.TrimSpace(manifestID)
	supplierID, warehouseID := "", ""
	if client != nil && manifestID != "" {
		var err error
		supplierID, warehouseID, err = manifestScope(ctx, client, manifestID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrColdChainBaselineRequired, err)
		}
	}
	if !EffectiveColdChain(ctx, warehouseID, supplierID) {
		return nil
	}
	if manifestID == "" {
		return ErrColdChainBaselineRequired
	}
	if client == nil {
		if memoryBlocked() == nil && hasColdChainReadingMemory(manifestID) {
			return nil
		}
		return fmt.Errorf("%w: spanner unavailable", ErrColdChainBaselineRequired)
	}
	needs, err := productsNeedColdClient(ctx, client, manifestID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrColdChainBaselineRequired, err)
	}
	if !needs {
		return nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT ReadingId FROM TemperatureReadings WHERE ManifestId = @mid LIMIT 1`,
		Params: map[string]any{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err = iter.Next()
	if err == iterator.Done {
		return ErrColdChainBaselineRequired
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrColdChainBaselineRequired, err)
	}
	return nil
}

func productsNeedColdClient(ctx context.Context, client *spanner.Client, manifestID string) (bool, error) {
	productIDs, err := collectManifestProductIDs(ctx, client, manifestID)
	if err != nil {
		return false, err
	}
	if len(productIDs) == 0 {
		return false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ProductId FROM Products
		      WHERE ProductId IN UNNEST(@pids)
		        AND (RequiresColdChain = true OR IsPerishable = true OR HandlingClass IN ('FROZEN', 'CHILLED', 'COLD', 'PERISHABLE'))
		      LIMIT 1`,
		Params: map[string]any{"pids": productIDs},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err = iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func collectManifestProductIDs(ctx context.Context, client *spanner.Client, manifestID string) ([]string, error) {
	seen := make(map[string]struct{})
	queries := []spanner.Statement{
		{
			SQL: `SELECT o.LineItemsJson
			      FROM ManifestOrders mo
			      JOIN Orders o ON o.OrderId = mo.OrderId
			      WHERE mo.ManifestId = @mid
			      LIMIT 200`,
			Params: map[string]any{"mid": manifestID},
		},
		{
			SQL:    `SELECT LineItemsJson FROM Orders WHERE ManifestId = @mid LIMIT 200`,
			Params: map[string]any{"mid": manifestID},
		},
	}
	for _, stmt := range queries {
		iter := client.Single().Query(ctx, stmt)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return nil, err
			}
			var raw []byte
			if err := row.Columns(&raw); err != nil {
				continue
			}
			for _, id := range extractProductIDsFromLineItems(raw) {
				seen[id] = struct{}{}
			}
		}
		iter.Stop()
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
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
