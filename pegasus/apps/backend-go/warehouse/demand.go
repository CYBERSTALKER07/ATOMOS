package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/proximity"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// DemandBreakdown represents the 4-source demand analysis for a product.
type DemandBreakdown struct {
	ProductID         string  `json:"product_id"`
	ProductName       string  `json:"product_name"`
	CurrentStock      int64   `json:"current_stock"`
	IncomingOrders    int64   `json:"incoming_orders"` // confirmed retailer orders in pipeline
	AIPredicted       int64   `json:"ai_predicted"`    // AI preorder predictions
	PreOrders         int64   `json:"pre_orders"`      // scheduled orders with future delivery dates
	BurnRateDaily     float64 `json:"burn_rate_daily"` // avg daily consumption from last 30 days
	RecommendedQty    int64   `json:"recommended_qty"` // recommended restock quantity
	DaysUntilStockout float64 `json:"days_until_stockout"`
	UnitVolumeVU      float64 `json:"unit_volume_vu"`
	Priority          string  `json:"priority"` // CRITICAL | URGENT | NORMAL
}

// DemandForecastResponse wraps the full demand analysis for a warehouse.
type DemandForecastResponse struct {
	WarehouseID   string            `json:"warehouse_id"`
	WarehouseName string            `json:"warehouse_name"`
	AnalyzedAt    time.Time         `json:"analyzed_at"`
	ForecastDays  int               `json:"forecast_days"` // 7 or 30
	Products      []DemandBreakdown `json:"products"`
	TotalVolumeVU float64           `json:"total_volume_vu"`
	Summary       struct {
		CriticalCount int `json:"critical_count"`
		UrgentCount   int `json:"urgent_count"`
		NormalCount   int `json:"normal_count"`
	} `json:"summary"`
}

// HandleDemandForecast returns AI-powered demand analysis per product for a warehouse.
// GET /v1/warehouse/demand/forecast?days=7|30
func HandleDemandForecast(spannerClient *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		warehouseID := claims.WarehouseID
		if warehouseID == "" {
			warehouseID = r.URL.Query().Get("warehouse_id")
		}
		if warehouseID == "" {
			http.Error(w, `{"error":"warehouse_id required"}`, http.StatusBadRequest)
			return
		}

		forecastDays := 7
		if r.URL.Query().Get("days") == "30" {
			forecastDays = 30
		}

		// Resolve supplier from warehouse
		supplierID, warehouseName, whLat, whLng, err := resolveWarehouseSupplier(r.Context(), spannerClient, warehouseID)
		if err != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}

		readClient := proximity.ReadClientForRetailer(spannerClient, readRouter, whLat, whLng)
		products, err := computeDemandForecast(r.Context(), readClient, supplierID, warehouseID, forecastDays)
		if err != nil {
			log.Printf("[DEMAND FORECAST] computation error for warehouse %s: %v", warehouseID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		resp := DemandForecastResponse{
			WarehouseID:   warehouseID,
			WarehouseName: warehouseName,
			AnalyzedAt:    time.Now().UTC(),
			ForecastDays:  forecastDays,
			Products:      products,
		}

		for _, p := range products {
			resp.TotalVolumeVU += float64(p.RecommendedQty) * p.UnitVolumeVU
			switch p.Priority {
			case "CRITICAL":
				resp.Summary.CriticalCount++
			case "URGENT":
				resp.Summary.UrgentCount++
			default:
				resp.Summary.NormalCount++
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func resolveWarehouseSupplier(ctx context.Context, client *spanner.Client, warehouseID string) (string, string, float64, float64, error) {
	row, err := client.Single().ReadRow(ctx, "Warehouses",
		spanner.Key{warehouseID}, []string{"SupplierId", "Name", "Lat", "Lng"})
	if err != nil {
		return "", "", 0, 0, err
	}
	var supplierID, name string
	var lat, lng spanner.NullFloat64
	if err := row.Columns(&supplierID, &name, &lat, &lng); err != nil {
		return "", "", 0, 0, err
	}
	return supplierID, name, lat.Float64, lng.Float64, nil
}

func computeDemandForecast(ctx context.Context, client *spanner.Client, supplierID, warehouseID string, forecastDays int) ([]DemandBreakdown, error) {
	// Step 1: Get all active products for this supplier
	products, err := getActiveProducts(ctx, client, supplierID)
	if err != nil {
		return nil, fmt.Errorf("fetch products: %w", err)
	}

	// Step 2: Aggregate incoming orders per product (PENDING, PENDING_REVIEW, LOADED, DISPATCHED)
	incomingMap, err := getIncomingOrderQuantities(ctx, client, supplierID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("incoming orders: %w", err)
	}

	// Step 3: Aggregate AI predictions per product
	aiPredMap, err := getAIPredictionQuantities(ctx, client, supplierID, forecastDays)
	if err != nil {
		return nil, fmt.Errorf("ai predictions: %w", err)
	}

	// Step 4: Aggregate scheduled pre-orders per product
	preOrderMap, err := getPreOrderQuantities(ctx, client, supplierID, warehouseID, forecastDays)
	if err != nil {
		return nil, fmt.Errorf("pre-orders: %w", err)
	}

	// Step 5: Calculate burn rate from last 30 days of completed orders
	burnMap, err := getBurnRates(ctx, client, supplierID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("burn rates: %w", err)
	}

	// Step 6: Get current stock levels from replenishment insights
	stockMap, err := getCurrentStock(ctx, client, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("stock levels: %w", err)
	}

	var results []DemandBreakdown
	for _, p := range products {
		incoming := incomingMap[p.ProductID]
		aiPred := aiPredMap[p.ProductID]
		preOrders := preOrderMap[p.ProductID]
		burnRate := burnMap[p.ProductID]
		stock := stockMap[p.ProductID]

		// Total demand = incoming + AI predicted + pre-orders + (burn_rate * forecast_days)
		totalDemand := incoming + aiPred + preOrders + int64(math.Ceil(burnRate*float64(forecastDays)))

		// Recommended qty = total demand - current stock (with 20% safety buffer)
		deficit := totalDemand - stock
		recommendedQty := int64(0)
		if deficit > 0 {
			recommendedQty = int64(math.Ceil(float64(deficit) * 1.20)) // 20% safety buffer
		}

		daysUntilStockout := float64(0)
		if burnRate > 0 {
			daysUntilStockout = float64(stock) / burnRate
		} else if stock > 0 {
			daysUntilStockout = 999 // effectively no consumption
		}

		priority := "NORMAL"
		if daysUntilStockout < 2 {
			priority = "CRITICAL"
		} else if daysUntilStockout < 5 {
			priority = "URGENT"
		}

		if recommendedQty == 0 && priority == "NORMAL" {
			continue // No restock needed — skip from recommendation
		}

		results = append(results, DemandBreakdown{
			ProductID:         p.ProductID,
			ProductName:       p.ProductName,
			CurrentStock:      stock,
			IncomingOrders:    incoming,
			AIPredicted:       aiPred,
			PreOrders:         preOrders,
			BurnRateDaily:     math.Round(burnRate*100) / 100,
			RecommendedQty:    recommendedQty,
			DaysUntilStockout: math.Round(daysUntilStockout*10) / 10,
			UnitVolumeVU:      p.UnitVolumeVU,
			Priority:          priority,
		})
	}

	return results, nil
}

type productInfo struct {
	ProductID    string
	ProductName  string
	UnitVolumeVU float64
}

func getActiveProducts(ctx context.Context, client *spanner.Client, supplierID string) ([]productInfo, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, Name, VolumeUnits
		      FROM SupplierProducts
		      WHERE SupplierId = @sid AND IsActive = TRUE`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	var results []productInfo
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p productInfo
		var vol spanner.NullFloat64
		if err := row.Columns(&p.ProductID, &p.ProductName, &vol); err != nil {
			return nil, err
		}
		if vol.Valid {
			p.UnitVolumeVU = vol.Float64
		}
		results = append(results, p)
	}
	return results, nil
}

func getIncomingOrderQuantities(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT oi.ProductId, SUM(oi.Quantity) AS qty
		      FROM OrderItems oi
		      JOIN Orders o ON oi.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @wid
		        AND o.State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'DISPATCHED', 'IN_TRANSIT')
		      GROUP BY oi.ProductId`,
		Params: map[string]interface{}{"sid": supplierID, "wid": warehouseID},
	}
	return queryProductQuantityMap(ctx, client, stmt)
}

func getAIPredictionQuantities(ctx context.Context, client *spanner.Client, supplierID string, forecastDays int) (map[string]int64, error) {
	cutoff := time.Now().Add(time.Duration(forecastDays) * 24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT api.ProductId, SUM(api.PredictedQuantity) AS qty
		      FROM AIPredictionItems api
		      JOIN AIPredictions ap ON api.PredictionId = ap.PredictionId
		      WHERE ap.SupplierId = @sid
		        AND ap.Status IN ('PENDING', 'CONFIRMED')
		        AND ap.TriggerDate <= @cutoff
		      GROUP BY api.ProductId`,
		Params: map[string]interface{}{"sid": supplierID, "cutoff": cutoff},
	}
	return queryProductQuantityMap(ctx, client, stmt)
}

func getPreOrderQuantities(ctx context.Context, client *spanner.Client, supplierID, warehouseID string, forecastDays int) (map[string]int64, error) {
	cutoff := time.Now().Add(time.Duration(forecastDays) * 24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT oi.ProductId, SUM(oi.Quantity) AS qty
		      FROM OrderItems oi
		      JOIN Orders o ON oi.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @wid
		        AND o.State = 'SCHEDULED'
		        AND o.RequestedDeliveryDate <= @cutoff
		      GROUP BY oi.ProductId`,
		Params: map[string]interface{}{"sid": supplierID, "wid": warehouseID, "cutoff": cutoff},
	}
	return queryProductQuantityMap(ctx, client, stmt)
}

func getBurnRates(ctx context.Context, client *spanner.Client, supplierID, warehouseID string) (map[string]float64, error) {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT oi.ProductId, SUM(oi.Quantity) AS total
		      FROM OrderItems oi
		      JOIN Orders o ON oi.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @wid
		        AND o.State = 'COMPLETED'
		        AND o.CompletedAt >= @since
		      GROUP BY oi.ProductId`,
		Params: map[string]interface{}{"sid": supplierID, "wid": warehouseID, "since": thirtyDaysAgo},
	}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	result := make(map[string]float64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var total int64
		if err := row.Columns(&productID, &total); err != nil {
			return nil, err
		}
		result[productID] = float64(total) / 30.0
	}
	return result, nil
}

func getCurrentStock(ctx context.Context, client *spanner.Client, warehouseID string) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, CurrentLevel
		      FROM ReplenishmentInsights
		      WHERE WarehouseId = @wid`,
		Params: map[string]interface{}{"wid": warehouseID},
	}
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	result := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var level int64
		if err := row.Columns(&productID, &level); err != nil {
			return nil, err
		}
		result[productID] = level
	}
	return result, nil
}

func queryProductQuantityMap(ctx context.Context, client *spanner.Client, stmt spanner.Statement) (map[string]int64, error) {
	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	result := make(map[string]int64)
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
		result[productID] = qty
	}
	return result, nil
}
