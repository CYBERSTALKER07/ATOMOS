package order

// ─────────────────────────────────────────────────────────────────────────────
// AI Pre-Orderer — The Empathy Engine
//
// Architecture:
//   1. Reads the retailer's last 10 COMPLETED orders from Spanner.
//   2. Sends order history as a structured prompt to Gemini 2.0 Flash via REST.
//   3. Gemini responds with a JSON prediction: { predicted_amount, trigger_date }.
//   4. SavePrediction() writes the result to Spanner's AIPredictions table.
//   5. StartAwakener() (cron.go) polls every minute and fires WAITING predictions
//      that have passed their TriggerDate via CreateOrder().
//
// Security: GEMINI_API_KEY is read from the environment — never hardcoded.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Spanner History Queries ───────────────────────────────────────────────

type orderHistoryRow struct {
	OrderID        string `json:"order_id"`
	Amount         int64  `json:"amount"`
	PaymentGateway string `json:"payment_gateway"`
	State          string `json:"state"`
}

func (s *OrderService) getOrderHistory(ctx context.Context, retailerID string) ([]orderHistoryRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, Amount, PaymentGateway, State
		      FROM Orders
		      WHERE RetailerId = @retailerId AND State = 'COMPLETED'
		      ORDER BY OrderId DESC
		      LIMIT 10`,
		Params: map[string]interface{}{
			"retailerId": retailerID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []orderHistoryRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("history query failed: %v", err)
		}
		var r orderHistoryRow
		if err := row.Columns(&r.OrderID, &r.Amount, &r.PaymentGateway, &r.State); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// lineItemHistoryRow is a single SKU purchase record for the Gemini prompt.
type lineItemHistoryRow struct {
	SkuID     string `json:"sku_id"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	OrderDate string `json:"order_date"`
}

// getLineItemHistory fetches recent SKU-level purchase history for a retailer.
func (s *OrderService) getLineItemHistory(ctx context.Context, retailerID string) ([]lineItemHistoryRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT oli.SkuId, oli.Quantity, oli.UnitPrice, o.CreatedAt
		      FROM OrderLineItems oli
		      JOIN Orders o ON o.OrderId = oli.OrderId
		      WHERE o.RetailerId = @rid AND o.State = 'COMPLETED'
		      ORDER BY o.CreatedAt DESC
		      LIMIT 200`,
		Params: map[string]interface{}{"rid": retailerID},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []lineItemHistoryRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line item history query failed: %v", err)
		}
		var r lineItemHistoryRow
		var createdAt time.Time
		if err := row.Columns(&r.SkuID, &r.Quantity, &r.UnitPrice, &createdAt); err != nil {
			return nil, err
		}
		r.OrderDate = createdAt.Format(time.RFC3339)
		rows = append(rows, r)
	}
	return rows, nil
}

// ─── Gemini REST Request/Response Structs ─────────────────────────────────

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

// ─── AI Prediction Output (parsed from Gemini JSON) ───────────────────────

type AIPredictionResult struct {
	RetailerID       string           `json:"retailer_id"`
	PredictedAmount  int64            `json:"predicted_amount"`
	TriggerDate      string           `json:"trigger_date"` // RFC3339
	ReasoningSummary string           `json:"reasoning_summary"`
	Items            []PredictionItem `json:"items,omitempty"`
}

// ─── Core: GeneratePreorder ────────────────────────────────────────────────

// GeneratePreorder reads a retailer's order and line-item history, sends it
// to Gemini 2.0 Flash for SKU-level prediction, and persists via
// SavePredictionWithItems(). Falls back to aggregate-only if SKU data is
// unavailable.
func (s *OrderService) GeneratePreorder(ctx context.Context, retailerID string) (*AIPredictionResult, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set — cannot invoke Empathy Engine")
	}

	// 1. Fetch order history from Spanner
	history, err := s.getOrderHistory(ctx, retailerID)
	if err != nil {
		return nil, fmt.Errorf("history fetch failed: %v", err)
	}
	if len(history) == 0 {
		// Cold start (A-1): No order history → try neighborhood heuristic
		return s.coldStartPrediction(ctx, retailerID)
	}

	// 2. Fetch line-item history for SKU-level prompt
	lineItems, lineErr := s.getLineItemHistory(ctx, retailerID)
	hasLineItems := lineErr == nil && len(lineItems) > 0

	// 3. Build structured prompt — SKU-level when line items available
	historyJSON, _ := json.MarshalIndent(history, "", "  ")

	var prompt string
	if hasLineItems {
		lineItemsJSON, _ := json.MarshalIndent(lineItems, "", "  ")
		prompt = fmt.Sprintf(`You are an AI supply chain analyst for The Lab Industries, a Coca-Cola B2B distributor in Uzbekistan.

Analyze the following completed order history AND line-item purchase history for retailer "%s" and predict their next reorder at SKU level.

ORDER HISTORY (last %d completed orders):
%s

LINE-ITEM HISTORY (last %d SKU purchases):
%s

Respond ONLY with a valid JSON object in this exact format (no markdown, no explanation):
{
  "predicted_amount": <integer, total order value in minor currency units>,
  "trigger_date": "<RFC3339 timestamp, when to auto-dispatch, typically 7-14 days from now>",
  "reasoning_summary": "<1 sentence explaining the prediction>",
  "items": [
    {"sku_id": "<string>", "quantity": <integer>, "price": <integer, unit price>}
  ]
}

Include the top SKUs by frequency/recency. The items array must have at least 1 entry. predicted_amount should equal the sum of (quantity * price) across items.`,
			retailerID, len(history), string(historyJSON), len(lineItems), string(lineItemsJSON))
	} else {
		prompt = fmt.Sprintf(`You are an AI supply chain analyst for The Lab Industries, a Coca-Cola B2B distributor in Uzbekistan.

Analyze the following completed order history for retailer "%s" and predict their next reorder.

ORDER HISTORY (last %d completed orders):
%s

Respond ONLY with a valid JSON object in this exact format (no markdown, no explanation):
{
  "predicted_amount": <integer, based on their average order value trend>,
  "trigger_date": "<RFC3339 timestamp, when to auto-dispatch the next order, typically 7-14 days from now>",
  "reasoning_summary": "<1 sentence explaining the prediction>"
}`,
			retailerID, len(history), string(historyJSON))
	}

	// 4. Call Gemini 2.0 Flash REST API
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini request: %v", err)
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s",
		apiKey,
	)

	httpResp, err := http.Post(endpoint, "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %v", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gemini response: %v", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini returned HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	// 5. Parse Gemini response
	var gemResp geminiResponse
	if err := json.Unmarshal(body, &gemResp); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %v", err)
	}
	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini returned no candidates")
	}

	rawText := gemResp.Candidates[0].Content.Parts[0].Text
	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	var prediction struct {
		PredictedAmount  int64  `json:"predicted_amount"`
		TriggerDate      string `json:"trigger_date"`
		ReasoningSummary string `json:"reasoning_summary"`
		Items            []struct {
			SkuID    string `json:"sku_id"`
			Quantity int64  `json:"quantity"`
			Price    int64  `json:"price"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(rawText), &prediction); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini JSON prediction: %v\nRaw: %s", err, rawText)
	}

	fmt.Printf("[EMPATHY ENGINE] Prediction for %s: %d on %s (%d items) — %s\n",
		retailerID, prediction.PredictedAmount, prediction.TriggerDate,
		len(prediction.Items), prediction.ReasoningSummary)

	// 6. SKU Guard (A-2): Validate predicted SKUs against SupplierProducts catalog
	var predItems []PredictionItem
	if len(prediction.Items) > 0 {
		validSKUs, err := s.validatePredictedSKUs(ctx, prediction.Items)
		if err != nil {
			log.Printf("[SKU_GUARD] Validation query failed for %s: %v — keeping all items", retailerID, err)
			for _, item := range prediction.Items {
				predItems = append(predItems, PredictionItem{
					SkuID:    item.SkuID,
					Quantity: item.Quantity,
					Price:    item.Price,
				})
			}
		} else if len(validSKUs) == 0 {
			// ALL SKUs hallucinated — fall back to cold start
			log.Printf("[SKU_GUARD] All %d predicted SKUs are hallucinated for %s — falling back to cold start", len(prediction.Items), retailerID)
			return s.coldStartPrediction(ctx, retailerID)
		} else {
			predItems = validSKUs
			if len(validSKUs) < len(prediction.Items) {
				log.Printf("[SKU_GUARD] Filtered %d hallucinated SKUs for %s (%d → %d)",
					len(prediction.Items)-len(validSKUs), retailerID, len(prediction.Items), len(validSKUs))
			}
		}
	}

	if len(predItems) > 0 {
		if err := s.SavePredictionWithItems(ctx, retailerID, prediction.PredictedAmount, prediction.TriggerDate, predItems, "WAITING", ""); err != nil {
			return nil, fmt.Errorf("failed to save prediction with items: %v", err)
		}
	} else {
		if err := s.SavePrediction(ctx, retailerID, prediction.PredictedAmount, prediction.TriggerDate, ""); err != nil {
			return nil, fmt.Errorf("failed to save prediction to Spanner: %v", err)
		}
	}

	return &AIPredictionResult{
		RetailerID:       retailerID,
		PredictedAmount:  prediction.PredictedAmount,
		TriggerDate:      prediction.TriggerDate,
		ReasoningSummary: prediction.ReasoningSummary,
		Items:            predItems,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cold Start Heuristic (A-1)
// ═══════════════════════════════════════════════════════════════════════════════

// coldStartPrediction generates a prediction for a retailer with zero order history
// by querying neighborhood peers (same supplier, same H3 cell) for their top SKUs.
// Falls back to supplier's globally top-selling SKUs if no neighborhood data exists.
func (s *OrderService) coldStartPrediction(ctx context.Context, retailerID string) (*AIPredictionResult, error) {
	// Try neighborhood heuristic: find top SKUs ordered by other retailers in the same H3 cell
	stmt := spanner.Statement{
		SQL: `SELECT oi.SkuId, SUM(oi.Quantity) AS TotalQty, AVG(oi.UnitPrice) AS AvgPrice
		      FROM OrderItems oi
		      JOIN Orders o ON oi.OrderId = o.OrderId
		      JOIN Retailers r ON o.RetailerId = r.RetailerId
		      WHERE o.State = 'COMPLETED'
		        AND r.H3Index = (SELECT H3Index FROM Retailers WHERE RetailerId = @retailerID)
		        AND o.RetailerId != @retailerID
		      GROUP BY oi.SkuId
		      ORDER BY TotalQty DESC
		      LIMIT 10`,
		Params: map[string]interface{}{"retailerID": retailerID},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var items []PredictionItem
	var totalAmount int64
	source := "NEIGHBORHOOD_HEURISTIC"

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[COLD_START] Neighborhood query failed for %s: %v", retailerID, err)
			break
		}

		var skuID string
		var qty int64
		var avgPrice float64
		if err := row.Columns(&skuID, &qty, &avgPrice); err != nil {
			continue
		}

		// Normalize: cap quantity at median (half of neighborhood total)
		normQty := qty / 2
		if normQty < 1 {
			normQty = 1
		}
		price := int64(avgPrice)
		items = append(items, PredictionItem{
			SkuID:    skuID,
			Quantity: normQty,
			Price:    price,
		})
		totalAmount += normQty * price
	}

	// If no neighborhood data, fall back to supplier default (top 10 global SKUs)
	if len(items) == 0 {
		source = "SUPPLIER_DEFAULT"
		fallbackStmt := spanner.Statement{
			SQL: `SELECT oi.SkuId, SUM(oi.Quantity) AS TotalQty, AVG(oi.UnitPrice) AS AvgPrice
			      FROM OrderItems oi
			      JOIN Orders o ON oi.OrderId = o.OrderId
			      WHERE o.State = 'COMPLETED'
			      GROUP BY oi.SkuId
			      ORDER BY TotalQty DESC
			      LIMIT 10`,
		}

		fbIter := s.Client.Single().Query(ctx, fallbackStmt)
		defer fbIter.Stop()

		for {
			row, err := fbIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("cold start fallback query failed: %v", err)
			}

			var skuID string
			var qty int64
			var avgPrice float64
			if err := row.Columns(&skuID, &qty, &avgPrice); err != nil {
				continue
			}

			normQty := qty / 4 // More conservative for global fallback
			if normQty < 1 {
				normQty = 1
			}
			price := int64(avgPrice)
			items = append(items, PredictionItem{
				SkuID:    skuID,
				Quantity: normQty,
				Price:    price,
			})
			totalAmount += normQty * price
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("cold start: no neighborhood or global data available for retailer %s", retailerID)
	}

	triggerDate := time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	reasoning := fmt.Sprintf("Cold start via %s — %d SKUs", source, len(items))

	if err := s.SavePredictionWithItems(ctx, retailerID, totalAmount, triggerDate, items, "WAITING", source); err != nil {
		return nil, fmt.Errorf("cold start save failed: %v", err)
	}

	log.Printf("[COLD_START] Generated %s prediction for %s: %d, %d items", source, retailerID, totalAmount, len(items))

	return &AIPredictionResult{
		RetailerID:       retailerID,
		PredictedAmount:  totalAmount,
		TriggerDate:      triggerDate,
		ReasoningSummary: reasoning,
		Items:            items,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// SKU Hallucination Guard (A-2)
// ═══════════════════════════════════════════════════════════════════════════════

// validatePredictedSKUs batch-checks predicted SKU IDs against the SupplierProducts
// catalog. Returns only items with valid, active SKUs. Hallucinated SKUs are logged.
func (s *OrderService) validatePredictedSKUs(ctx context.Context, items []struct {
	SkuID    string `json:"sku_id"`
	Quantity int64  `json:"quantity"`
	Price    int64  `json:"price"`
}) ([]PredictionItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// Collect unique SKU IDs
	skuIDs := make([]string, 0, len(items))
	for _, item := range items {
		skuIDs = append(skuIDs, item.SkuID)
	}

	// Batch query: which of these SKUs actually exist and are active?
	stmt := spanner.Statement{
		SQL: `SELECT ProductId FROM SupplierProducts
		      WHERE ProductId IN UNNEST(@skuIDs)
		        AND IsActive = TRUE`,
		Params: map[string]interface{}{"skuIDs": skuIDs},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	validSet := make(map[string]bool)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("sku validation query failed: %w", err)
		}
		var pid string
		if err := row.Columns(&pid); err == nil {
			validSet[pid] = true
		}
	}

	// Filter: keep only items with valid SKUs
	var valid []PredictionItem
	for _, item := range items {
		if validSet[item.SkuID] {
			valid = append(valid, PredictionItem{
				SkuID:    item.SkuID,
				Quantity: item.Quantity,
				Price:    item.Price,
			})
		} else {
			log.Printf("[SKU_GUARD] Hallucinated SKU filtered: %s", item.SkuID)
		}
	}

	return valid, nil
}
