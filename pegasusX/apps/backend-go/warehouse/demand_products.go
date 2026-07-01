package warehouse

import (
	"context"
	"math"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

type demandForecastSources struct {
	IncomingOrders int64   `json:"incoming_orders"`
	AIPrediction   int64   `json:"ai_prediction"`
	PreOrders      int64   `json:"pre_orders"`
	BurnRate       float64 `json:"burn_rate"`
}

type demandForecastProduct struct {
	ProductID         string                `json:"product_id"`
	ProductName       string                `json:"product_name"`
	CurrentStock      int64                 `json:"current_stock"`
	RecommendedQty    int64                 `json:"recommended_qty"`
	DaysUntilStockout float64               `json:"days_until_stockout"`
	Priority          string                `json:"priority"`
	Unit              string                `json:"unit"`
	Sources           demandForecastSources `json:"sources"`
}

func (s *Service) productDemandForecast(ctx context.Context, warehouseID string, forecastDays int) []demandForecastProduct {
	if rows, err := s.productDemandFromSpanner(ctx, warehouseID, forecastDays); err == nil && len(rows) > 0 {
		return rows
	}
	return s.productDemandFromScaffold(warehouseID, forecastDays)
}

func (s *Service) productDemandFromScaffold(warehouseID string, forecastDays int) []demandForecastProduct {
	s.ensurePortalSeed()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureReplenishmentInsightsLocked(warehouseID)

	insightByProduct := make(map[string]replenishmentInsight, len(s.insights))
	for _, row := range s.insights {
		if warehouseID != "" && row.WarehouseID != warehouseID {
			continue
		}
		insightByProduct[row.ProductID] = row
	}

	pendingByProduct := make(map[string]int64)
	for _, order := range s.orders {
		if strings.ToUpper(order.Status) != "PENDING" && strings.ToUpper(order.Status) != "LOADED" {
			continue
		}
		pendingByProduct["prod-1"] += 1
	}

	seen := make(map[string]struct{})
	var out []demandForecastProduct

	appendRow := func(productID, name string, stock int64) {
		if _, ok := seen[productID]; ok {
			return
		}
		seen[productID] = struct{}{}

		incoming := pendingByProduct[productID]
		insight, hasInsight := insightByProduct[productID]
		burn := 0.0
		aiPred := int64(0)
		preOrders := int64(0)
		recommended := int64(0)
		daysOut := 0.0
		priority := "NORMAL"

		if hasInsight {
			stock = insight.CurrentStock
			burn = insight.AvgDailyVelocity
			recommended = insight.ReorderQuantity
			daysOut = float64(insight.DaysUntilStockout)
			switch strings.ToUpper(insight.Urgency) {
			case "CRITICAL", "HIGH":
				priority = "CRITICAL"
			case "URGENT", "MEDIUM":
				priority = "URGENT"
			}
		} else if burn <= 0 && stock > 0 {
			daysOut = 999
		}

		if recommended == 0 && burn > 0 {
			deficit := incoming + aiPred + preOrders + int64(math.Ceil(burn*float64(forecastDays))) - stock
			if deficit > 0 {
				recommended = int64(math.Ceil(float64(deficit) * 1.2))
			}
			if daysOut == 0 {
				daysOut = float64(stock) / burn
			}
		}
		if priority == "NORMAL" {
			if daysOut < 2 {
				priority = "CRITICAL"
			} else if daysOut < 5 {
				priority = "URGENT"
			}
		}

		if recommended == 0 && priority == "NORMAL" {
			return
		}

		out = append(out, demandForecastProduct{
			ProductID:         productID,
			ProductName:       name,
			CurrentStock:      stock,
			RecommendedQty:    recommended,
			DaysUntilStockout: math.Round(daysOut*10) / 10,
			Priority:          priority,
			Unit:              "VU",
			Sources: demandForecastSources{
				IncomingOrders: incoming,
				AIPrediction:   aiPred,
				PreOrders:      preOrders,
				BurnRate:       math.Round(burn*100) / 100,
			},
		})
	}

	inventoryList, _ := s.repo.GetInventoryList(context.Background(), warehouseID, InventoryListOptions{})
	for sku, row := range inventoryList {
		appendRow(sku, row.ProductName, int64(row.Quantity))
	}
	for _, product := range s.products {
		if !product.IsActive {
			continue
		}
		stock := int64(0)
		if inv, ok := inventoryList[product.ProductID]; ok {
			stock = int64(inv.Quantity)
		}
		appendRow(product.ProductID, product.Name, stock)
	}

	for _, insight := range s.insights {
		if warehouseID != "" && insight.WarehouseID != warehouseID {
			continue
		}
		appendRow(insight.ProductID, insight.ProductName, insight.CurrentStock)
	}

	return out
}

func (s *Service) productDemandFromSpanner(ctx context.Context, warehouseID string, forecastDays int) ([]demandForecastProduct, error) {
	if s == nil || s.spannerClient == nil || strings.TrimSpace(warehouseID) == "" {
		return nil, spanner.ErrRowNotFound
	}
	supplierID := strings.TrimSpace(s.supplierID)
	if supplierID == "" {
		return nil, spanner.ErrRowNotFound
	}

	stockMap, err := s.inventoryLevelsByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	insightByProduct := s.replenishmentInsightsByProduct(ctx, warehouseID)
	baselineByProduct := s.demandBaselineByProduct(ctx, supplierID, warehouseID, forecastDays)
	products, err := s.activeProductsBySupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}

	var out []demandForecastProduct
	for _, product := range products {
		stock := stockMap[product.ProductID]
		burn := 0.0
		daysOut := 999.0
		recommended := int64(0)
		priority := "NORMAL"

		if insight, ok := insightByProduct[product.ProductID]; ok {
			stock = insight.CurrentStock
			burn = insight.AvgDailyVelocity
			recommended = insight.ReorderQuantity
			daysOut = float64(insight.DaysUntilStockout)
			switch strings.ToUpper(insight.Urgency) {
			case "CRITICAL", "HIGH":
				priority = "CRITICAL"
			case "URGENT", "WARNING":
				priority = "URGENT"
			}
		} else if baseline, ok := baselineByProduct[product.ProductID]; ok && baseline > 0 {
			burn = float64(baseline) / float64(forecastDays)
			recommended = baseline
			if stock > 0 && burn > 0 {
				daysOut = float64(stock) / burn
			}
		} else if stock > 0 {
			burn = 1.0
			daysOut = float64(stock) / burn
		}
		if recommended == 0 && burn > 0 {
			deficit := int64(math.Ceil(burn*float64(forecastDays))) - stock
			if deficit > 0 {
				recommended = int64(math.Ceil(float64(deficit) * 1.2))
			}
		}
		if priority == "NORMAL" {
			if daysOut < 2 {
				priority = "CRITICAL"
			} else if daysOut < 5 {
				priority = "URGENT"
			}
		}
		if recommended == 0 && priority == "NORMAL" {
			continue
		}
		out = append(out, demandForecastProduct{
			ProductID:         product.ProductID,
			ProductName:       product.Name,
			CurrentStock:      stock,
			RecommendedQty:    recommended,
			DaysUntilStockout: math.Round(daysOut*10) / 10,
			Priority:          priority,
			Unit:              "VU",
			Sources: demandForecastSources{
				BurnRate:     burn,
				AIPrediction: baselineByProduct[product.ProductID],
			},
		})
	}
	return out, nil
}

type catalogProduct struct {
	ProductID string
	Name      string
}

func (s *Service) activeProductsBySupplier(ctx context.Context, supplierID string) ([]catalogProduct, error) {
	const pageSize = 1000
	var all []catalogProduct
	offset := 0
	for {
		batch, err := s.activeProductsBySupplierPage(ctx, supplierID, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
		offset += pageSize
	}
	return all, nil
}

func (s *Service) activeProductsBySupplierPage(ctx context.Context, supplierID string, limit, offset int) ([]catalogProduct, error) {
	if limit <= 0 {
		limit = 1000
	}
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, Name FROM Products@{FORCE_INDEX=Idx_Products_BySupplierActive}
		      WHERE SupplierId = @sid AND IsActive = TRUE
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim OFFSET @off`,
		Params: map[string]any{
			"sid": supplierID,
			"lim": int64(limit),
			"off": int64(offset),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []catalogProduct
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var item catalogProduct
		if err := row.Columns(&item.ProductID, &item.Name); err != nil {
			return nil, err
		}
		rows = append(rows, item)
	}
	return rows, nil
}

func (s *Service) inventoryLevelsByWarehouse(ctx context.Context, warehouseID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, QuantityOnHand FROM InventoryLevels
		      WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()

	out := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var qty int64
		if err := row.Columns(&productID, &qty); err != nil {
			return nil, err
		}
		out[productID] = qty
	}
	return out, nil
}
