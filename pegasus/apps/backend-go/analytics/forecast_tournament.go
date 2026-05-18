package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/hotspot"
	"backend-go/proximity"
)

type ForecastTournamentRequest struct {
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	WarehouseID   string `json:"warehouse_id,omitempty"`
	MinSampleSize int64  `json:"min_sample_size,omitempty"`
}

type ForecastTournamentVariantScore struct {
	VariantKey           string  `json:"variant_key"`
	Label                string  `json:"label"`
	Predictions          int64   `json:"predictions"`
	Fired                int64   `json:"fired"`
	Rejected             int64   `json:"rejected"`
	Waiting              int64   `json:"waiting"`
	Dormant              int64   `json:"dormant"`
	ConversionRate       float64 `json:"conversion_rate"`
	Score                float64 `json:"score"`
	ChampionEligible     bool    `json:"champion_eligible"`
	AvgPredictedAmount   int64   `json:"avg_predicted_amount"`
	TotalPredictedAmount int64   `json:"total_predicted_amount"`
}

type ForecastTournamentResult struct {
	WindowFrom       string                           `json:"window_from"`
	WindowTo         string                           `json:"window_to"`
	ScopeSupplierID  string                           `json:"scope_supplier_id"`
	ScopeWarehouseID string                           `json:"scope_warehouse_id,omitempty"`
	MinSampleSize    int64                            `json:"min_sample_size"`
	ChampionVariant  string                           `json:"champion_variant,omitempty"`
	ChampionScore    float64                          `json:"champion_score"`
	TotalPredictions int64                            `json:"total_predictions"`
	Variants         []ForecastTournamentVariantScore `json:"variants"`
	DataSources      []string                         `json:"data_sources"`
	GeneratedAt      string                           `json:"generated_at"`
}

type normalizedForecastTournamentRequest struct {
	From          time.Time
	To            time.Time
	WarehouseID   string
	MinSampleSize int64
}

const (
	defaultForecastTournamentWindowDays = 30
	defaultForecastTournamentSampleSize = int64(10)
	maxForecastTournamentSampleSize     = int64(500)
)

// HandleForecastTournament exposes supplier-scoped forecast variant scoring.
//
// POST /v1/supplier/analytics/forecast/tournament
func HandleForecastTournament(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ws := extractScope(r)
		if claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req ForecastTournamentRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		normalized, err := normalizeForecastTournamentRequest(req, ws)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"invalid_request","message":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		readClient := getReadClient(r.Context(), client, readRouter, ws)
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()

		variants, err := queryForecastTournamentVariants(ctx, readClient, claims.ResolveSupplierID(), normalized)
		if err != nil {
			http.Error(w, `{"error":"forecast_tournament_failed"}`, http.StatusInternalServerError)
			return
		}

		championVariant, championScore := pickChampionVariant(variants)

		totalPredictions := int64(0)
		for _, variant := range variants {
			totalPredictions += variant.Predictions
		}

		result := ForecastTournamentResult{
			WindowFrom:       normalized.From.Format(time.RFC3339),
			WindowTo:         normalized.To.Format(time.RFC3339),
			ScopeSupplierID:  claims.ResolveSupplierID(),
			ScopeWarehouseID: forecastTournamentScopeWarehouseID(normalized.WarehouseID, ws),
			MinSampleSize:    normalized.MinSampleSize,
			ChampionVariant:  championVariant,
			ChampionScore:    championScore,
			TotalPredictions: totalPredictions,
			Variants:         variants,
			DataSources:      []string{"AIPredictions", "AIPredictionItems", "SupplierRetailerClients"},
			GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		}

		writeJSON(w, result)
	}
}

func normalizeForecastTournamentRequest(
	raw ForecastTournamentRequest,
	ws *auth.WarehouseScope,
) (normalizedForecastTournamentRequest, error) {
	normalized := normalizedForecastTournamentRequest{
		WarehouseID:   strings.TrimSpace(raw.WarehouseID),
		MinSampleSize: raw.MinSampleSize,
	}

	now := time.Now().UTC()
	normalized.From = now.AddDate(0, 0, -defaultForecastTournamentWindowDays)
	normalized.To = now

	if strings.TrimSpace(raw.From) != "" {
		from, err := parseGraphQueryTime(raw.From, false)
		if err != nil {
			return normalizedForecastTournamentRequest{}, fmt.Errorf("invalid from timestamp")
		}
		normalized.From = from
	}

	if strings.TrimSpace(raw.To) != "" {
		to, err := parseGraphQueryTime(raw.To, true)
		if err != nil {
			return normalizedForecastTournamentRequest{}, fmt.Errorf("invalid to timestamp")
		}
		normalized.To = to
	}

	if normalized.From.After(normalized.To) {
		normalized.From, normalized.To = normalized.To, normalized.From
	}

	maxWindow := time.Duration(maxRangeDays) * 24 * time.Hour
	if normalized.To.Sub(normalized.From) > maxWindow {
		normalized.From = normalized.To.Add(-maxWindow)
	}

	if normalized.MinSampleSize <= 0 {
		normalized.MinSampleSize = defaultForecastTournamentSampleSize
	}
	if normalized.MinSampleSize > maxForecastTournamentSampleSize {
		return normalizedForecastTournamentRequest{}, fmt.Errorf("min_sample_size exceeds max %d", maxForecastTournamentSampleSize)
	}

	if ws != nil && ws.WarehouseID != "" {
		normalized.WarehouseID = ws.WarehouseID
	}

	return normalized, nil
}

func queryForecastTournamentVariants(
	ctx context.Context,
	client *spanner.Client,
	supplierID string,
	input normalizedForecastTournamentRequest,
) ([]ForecastTournamentVariantScore, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner client unavailable")
	}

	shards := hotspot.AllShards()
	stmt := spanner.Statement{
		SQL: `
			WITH scoped_predictions AS (
				SELECT p.PredictionId, p.PredictedAmount, p.Status, p.WarehouseId
				FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_ByTriggerShardStatusDate} p
				JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
				WHERE p.TriggerShard IN UNNEST(@shards)
				  AND p.TriggerDate >= @from_ts
				  AND p.TriggerDate <= @to_ts
				UNION ALL
				SELECT p.PredictionId, p.PredictedAmount, p.Status, p.WarehouseId
				FROM AIPredictions p
				JOIN SupplierRetailerClients src ON src.RetailerId = p.RetailerId AND src.SupplierId = @sid
				WHERE p.TriggerShard IS NULL
				  AND p.TriggerDate >= @from_ts
				  AND p.TriggerDate <= @to_ts
			),
			item_presence AS (
				SELECT pi.PredictionId, TRUE AS HasItems
				FROM AIPredictionItems pi
				JOIN scoped_predictions sp ON sp.PredictionId = pi.PredictionId
				GROUP BY pi.PredictionId
			),
			classified AS (
				SELECT
					CASE
						WHEN sp.WarehouseId IN ('NEIGHBORHOOD_HEURISTIC', 'SUPPLIER_DEFAULT') THEN sp.WarehouseId
						WHEN COALESCE(ip.HasItems, FALSE) THEN 'SKU_MEDIAN_V3'
						ELSE 'LEGACY_AGGREGATE'
					END AS VariantKey,
					sp.Status,
					sp.PredictedAmount
				FROM scoped_predictions sp
				LEFT JOIN item_presence ip ON ip.PredictionId = sp.PredictionId
				WHERE (@warehouse_filter = '' OR sp.WarehouseId = @warehouse_filter)
			)
			SELECT
				VariantKey,
				COUNT(*) AS Predictions,
				COUNTIF(Status = 'FIRED') AS Fired,
				COUNTIF(Status = 'REJECTED') AS Rejected,
				COUNTIF(Status = 'WAITING') AS Waiting,
				COUNTIF(Status = 'DORMANT') AS Dormant,
				COALESCE(AVG(PredictedAmount), 0) AS AvgPredictedAmount,
				COALESCE(SUM(PredictedAmount), 0) AS TotalPredictedAmount
			FROM classified
			GROUP BY VariantKey
			ORDER BY Predictions DESC`,
		Params: map[string]interface{}{
			"sid":              supplierID,
			"shards":           shards,
			"from_ts":          input.From,
			"to_ts":            input.To,
			"warehouse_filter": input.WarehouseID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	variants := make([]ForecastTournamentVariantScore, 0, 4)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var variantKey spanner.NullString
		var predictions, fired, rejected, waiting, dormant spanner.NullInt64
		var avgPredictedAmount spanner.NullFloat64
		var totalPredictedAmount spanner.NullInt64
		if err := row.Columns(
			&variantKey,
			&predictions,
			&fired,
			&rejected,
			&waiting,
			&dormant,
			&avgPredictedAmount,
			&totalPredictedAmount,
		); err != nil {
			continue
		}

		variant := ForecastTournamentVariantScore{
			VariantKey:           variantKey.StringVal,
			Label:                forecastVariantLabel(variantKey.StringVal),
			Predictions:          predictions.Int64,
			Fired:                fired.Int64,
			Rejected:             rejected.Int64,
			Waiting:              waiting.Int64,
			Dormant:              dormant.Int64,
			AvgPredictedAmount:   int64(math.Round(avgPredictedAmount.Float64)),
			TotalPredictedAmount: totalPredictedAmount.Int64,
			ChampionEligible:     predictions.Int64 >= input.MinSampleSize,
		}

		settled := variant.Fired + variant.Rejected
		if settled > 0 {
			variant.ConversionRate = float64(variant.Fired) / float64(settled)
		}
		variant.ConversionRate = roundTo4(variant.ConversionRate)
		variant.Score = variant.ConversionRate

		variants = append(variants, variant)
	}

	sort.SliceStable(variants, func(i, j int) bool {
		if variants[i].Score == variants[j].Score {
			return variants[i].Predictions > variants[j].Predictions
		}
		return variants[i].Score > variants[j].Score
	})

	if variants == nil {
		variants = []ForecastTournamentVariantScore{}
	}

	return variants, nil
}

func pickChampionVariant(variants []ForecastTournamentVariantScore) (string, float64) {
	for _, variant := range variants {
		if variant.ChampionEligible {
			return variant.VariantKey, roundTo4(variant.Score)
		}
	}
	return "", 0
}

func forecastVariantLabel(variantKey string) string {
	switch variantKey {
	case "SKU_MEDIAN_V3":
		return "SKU Median v3"
	case "LEGACY_AGGREGATE":
		return "Legacy Aggregate"
	case "NEIGHBORHOOD_HEURISTIC":
		return "Cold Start Neighborhood"
	case "SUPPLIER_DEFAULT":
		return "Cold Start Supplier Default"
	default:
		return firstNonEmpty(variantKey, "Unknown")
	}
}

func forecastTournamentScopeWarehouseID(requestWarehouseID string, ws *auth.WarehouseScope) string {
	if ws != nil && ws.WarehouseID != "" {
		return ws.WarehouseID
	}
	return strings.TrimSpace(requestWarehouseID)
}

func roundTo4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
