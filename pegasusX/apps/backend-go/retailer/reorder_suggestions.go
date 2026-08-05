package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"google.golang.org/api/iterator"
)

// RetailerReorderSuggestion is the retailer-org view of OPEN reorder rows (L3.4).
type RetailerReorderSuggestion struct {
	SKU              string   `json:"sku"`
	SuggestedQty     int64    `json:"suggested_qty"`
	AdjustedDemand   float64  `json:"adjusted_demand_per_day"`
	CurrentStock     int64    `json:"current_stock"`
	InFlightQty      int64    `json:"in_flight_qty"`
	SafetyStock      float64  `json:"safety_stock,omitempty"`
	Status           string   `json:"status"`
	Sources          []string `json:"sources,omitempty"`
	SellThroughVel   float64  `json:"sell_through_velocity,omitempty"`
	BaseDemand       float64  `json:"base_demand_per_day,omitempty"`
	SuggestedByDate  string   `json:"suggested_by_date,omitempty"`
	ComputedAt       string   `json:"computed_at,omitempty"`
}

// SeedReorderSuggestions injects OPEN suggestions for unit tests (memory mode).
func (s *Service) SeedReorderSuggestions(orgID string, items []RetailerReorderSuggestion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reorderSuggestionSeed == nil {
		s.reorderSuggestionSeed = map[string][]RetailerReorderSuggestion{}
	}
	s.reorderSuggestionSeed[orgID] = append([]RetailerReorderSuggestion(nil), items...)
}

// HandleRetailerReorderSuggestions serves GET /v1/retailer/reorder-suggestions
func (s *Service) HandleRetailerReorderSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermOrderPlace) &&
		!auth.HasRetailerPerm(claims, auth.PermStockView) &&
		!auth.HasRetailerPerm(claims, auth.PermReportsView)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sourceFilter, srcErr := replenishment.ParseDemandSourcesQuery(r.URL.Query())
	if srcErr != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":   "invalid_source",
			"allowed": replenishment.SourceStorePOS + "," + replenishment.SourceWholesaleHistory,
		})
		return
	}
	items, err := s.listRetailerReorderSuggestions(r.Context(), orgID, sourceFilter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed", "detail": err.Error()})
		return
	}
	if items == nil {
		items = []RetailerReorderSuggestion{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) listRetailerReorderSuggestions(ctx context.Context, orgID string, sourceFilter []string) ([]RetailerReorderSuggestion, error) {
	// Memory / test seed first
	s.mu.RLock()
	if s.reorderSuggestionSeed != nil {
		if seed := s.reorderSuggestionSeed[orgID]; len(seed) > 0 {
			out := make([]RetailerReorderSuggestion, 0, len(seed))
			for _, item := range seed {
				if replenishment.SourcesMatchAny(item.Sources, sourceFilter) {
					out = append(out, item)
				}
			}
			s.mu.RUnlock()
			return out, nil
		}
	}
	s.mu.RUnlock()

	if s.spannerClient == nil {
		return []RetailerReorderSuggestion{}, nil
	}

	sql := `SELECT Sku, SuggestedQty, AdjustedDemand, CurrentStock, InFlightQty,
			COALESCE(SafetyStock, 0), Status,
			COALESCE(SourcesJson, ''), COALESCE(SellThroughVel, 0), COALESCE(BaseDemand, 0),
			CAST(SuggestedByDate AS STRING), CAST(ComputedAt AS STRING)
			FROM ReorderSuggestions
			WHERE RetailerId = @rid AND Status = @st AND SuggestedQty > 0`
	params := map[string]any{"rid": orgID, "st": "OPEN"}
	if len(sourceFilter) > 0 {
		parts := make([]string, 0, len(sourceFilter))
		for i, src := range sourceFilter {
			key := "srcN" + strconv.Itoa(i)
			parts = append(parts, "SourcesJson LIKE @"+key)
			params[key] = `%"%` + src + `"%`
		}
		sql += " AND (" + strings.Join(parts, " OR ") + ")"
	}
	sql += " ORDER BY SuggestedQty DESC LIMIT 100"
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []RetailerReorderSuggestion
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Pre-migration schema without sources columns
			return s.listRetailerReorderSuggestionsLegacy(ctx, orgID)
		}
		var item RetailerReorderSuggestion
		var sourcesJSON string
		if err := row.Columns(
			&item.SKU, &item.SuggestedQty, &item.AdjustedDemand, &item.CurrentStock, &item.InFlightQty,
			&item.SafetyStock, &item.Status,
			&sourcesJSON, &item.SellThroughVel, &item.BaseDemand, &item.SuggestedByDate, &item.ComputedAt,
		); err != nil {
			return nil, err
		}
		item.Sources = decodeSourcesJSON(sourcesJSON)
		if strings.HasPrefix(strings.ToLower(item.SKU), "local:") {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) listRetailerReorderSuggestionsLegacy(ctx context.Context, orgID string) ([]RetailerReorderSuggestion, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Sku, SuggestedQty, AdjustedDemand, CurrentStock, InFlightQty, Status
			FROM ReorderSuggestions
			WHERE RetailerId = @rid AND Status = @st AND SuggestedQty > 0
			ORDER BY SuggestedQty DESC LIMIT 100`,
		Params: map[string]any{"rid": orgID, "st": "OPEN"},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []RetailerReorderSuggestion
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var item RetailerReorderSuggestion
		if err := row.Columns(&item.SKU, &item.SuggestedQty, &item.AdjustedDemand, &item.CurrentStock, &item.InFlightQty, &item.Status); err != nil {
			return nil, err
		}
		item.Sources = []string{"WHOLESALE_HISTORY"}
		if strings.HasPrefix(strings.ToLower(item.SKU), "local:") {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func decodeSourcesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
