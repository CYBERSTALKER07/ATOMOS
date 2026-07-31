package replenishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	SuggestionStatusOpen      = "OPEN"
	SuggestionStatusDismissed = "DISMISSED"
	SuggestionStatusConverted = "CONVERTED"
)

// SuggestionRow is the wire shape for reorder suggestions.
type SuggestionRow struct {
	RetailerID      string  `json:"retailer_id"`
	RetailerName    string  `json:"retailer_name,omitempty"`
	Sku             string  `json:"sku"`
	SkuName         string  `json:"sku_name,omitempty"`
	SuggestedQty    int64   `json:"suggested_qty"`
	AdjustedDemand  float64 `json:"adjusted_demand_per_day"`
	CurrentStock    int64   `json:"current_stock"`
	InFlightQty     int64   `json:"in_flight_qty"`
	SuggestedByDate string  `json:"suggested_by_date"`
	Status          string  `json:"status"`
	ComputedAt      string  `json:"computed_at"`
}

// SuggestionsAPI serves supplier-portal reorder suggestion endpoints.
type SuggestionsAPI struct {
	Spanner     *spanner.Client
	Orders      *order.Service
	SupplierID  func(r *http.Request) string
}

func NewSuggestionsAPI(client *spanner.Client, orders *order.Service, supplierID func(r *http.Request) string) *SuggestionsAPI {
	return &SuggestionsAPI{
		Spanner:    client,
		Orders:     orders,
		SupplierID: supplierID,
	}
}

// HandleList serves GET /v1/replenishment/suggestions.
func (a *SuggestionsAPI) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if a.Spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"suggestions": []SuggestionRow{}})
		return
	}
	sid := strings.TrimSpace(a.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	retailerID := strings.TrimSpace(r.URL.Query().Get("retailerId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = SuggestionStatusOpen
	}
	skuSearch := strings.TrimSpace(r.URL.Query().Get("sku"))

	rows, err := a.listSuggestions(r.Context(), sid, retailerID, status, skuSearch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_suggestions_failed"})
		return
	}
	if rows == nil {
		rows = []SuggestionRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": rows})
}

type dismissRequest struct {
	RetailerID string `json:"retailer_id"`
	Sku        string `json:"sku"`
}

// HandleDismiss serves POST /v1/replenishment/suggestions/dismiss.
func (a *SuggestionsAPI) HandleDismiss(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if a.Spanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}
	sid := strings.TrimSpace(a.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req dismissRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.RetailerID = strings.TrimSpace(req.RetailerID)
	req.Sku = strings.TrimSpace(req.Sku)
	if req.RetailerID == "" || req.Sku == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_and_sku_required"})
		return
	}
	if !a.retailerLinkedToSupplier(r.Context(), sid, req.RetailerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_not_linked"})
		return
	}

	_, err := a.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ReorderSuggestions", map[string]any{
				"RetailerId": req.RetailerID,
				"Sku":        req.Sku,
				"Status":     SuggestionStatusDismissed,
			}),
		})
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dismiss_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createDraftRequest struct {
	RetailerID string `json:"retailer_id"`
	Sku        string `json:"sku"`
}

type bulkCreateDraftsRequest struct {
	Items []createDraftRequest `json:"items"`
}

// HandleCreateDraft serves POST /v1/replenishment/suggestions/create-draft.
func (a *SuggestionsAPI) HandleCreateDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if a.Spanner == nil || a.Orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_capture_unavailable"})
		return
	}
	sid := strings.TrimSpace(a.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := a.createDraftFromSuggestion(r.Context(), sid, req.RetailerID, req.Sku)
	if err != nil {
		switch {
		case errors.Is(err, errSuggestionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "suggestion_not_found"})
		case errors.Is(err, errRetailerNotLinked):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_not_linked"})
		case errors.Is(err, errRetailerLocationMissing):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "retailer_location_missing"})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// HandleBulkCreateDrafts serves POST /v1/replenishment/suggestions/create-drafts.
func (a *SuggestionsAPI) HandleBulkCreateDrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if a.Spanner == nil || a.Orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_capture_unavailable"})
		return
	}
	sid := strings.TrimSpace(a.SupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req bulkCreateDraftsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items_required"})
		return
	}

	type resultRow struct {
		RetailerID string `json:"retailer_id"`
		Sku        string `json:"sku"`
		OrderID    string `json:"order_id,omitempty"`
		Error      string `json:"error,omitempty"`
	}
	out := make([]resultRow, 0, len(req.Items))
	for _, item := range req.Items {
		resp, err := a.createDraftFromSuggestion(r.Context(), sid, item.RetailerID, item.Sku)
		if err != nil {
			out = append(out, resultRow{
				RetailerID: strings.TrimSpace(item.RetailerID),
				Sku:        strings.TrimSpace(item.Sku),
				Error:      err.Error(),
			})
			continue
		}
		out = append(out, resultRow{
			RetailerID: strings.TrimSpace(item.RetailerID),
			Sku:        strings.TrimSpace(item.Sku),
			OrderID:    resp.OrderID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

var (
	errSuggestionNotFound       = errors.New("suggestion_not_found")
	errRetailerNotLinked        = errors.New("retailer_not_linked")
	errRetailerLocationMissing  = errors.New("retailer_location_missing")
)

func (a *SuggestionsAPI) listSuggestions(ctx context.Context, supplierID, retailerID, status, skuSearch string) ([]SuggestionRow, error) {
	sql := `SELECT rs.RetailerId, rs.Sku, rs.SuggestedQty, rs.AdjustedDemand, rs.CurrentStock,
	               rs.InFlightQty, rs.SuggestedByDate, rs.ComputedAt, rs.Status,
	               r.Name AS RetailerName, p.Name AS SkuName
	        FROM ReorderSuggestions rs
	        INNER JOIN (
	          SELECT DISTINCT RetailerId FROM Orders WHERE SupplierId = @supplierId
	        ) linked ON linked.RetailerId = rs.RetailerId
	        LEFT JOIN Retailers r ON r.RetailerId = rs.RetailerId
	        LEFT JOIN Products p ON p.ProductId = rs.Sku
	        WHERE rs.Status = @status`
	params := map[string]interface{}{
		"supplierId": supplierID,
		"status":     status,
	}
	if retailerID != "" {
		sql += " AND rs.RetailerId = @retailerId"
		params["retailerId"] = retailerID
	}
	if skuSearch != "" {
		sql += " AND (rs.Sku LIKE @sku OR p.Name LIKE @sku)"
		params["sku"] = "%" + skuSearch + "%"
	}
	sql += " ORDER BY rs.ComputedAt DESC LIMIT 500"

	iter := a.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var rows []SuggestionRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sr SuggestionRow
		var suggestedBy civil.Date
		var computedAt time.Time
		var retailerName, skuName spanner.NullString
		if err := row.Columns(
			&sr.RetailerID, &sr.Sku, &sr.SuggestedQty, &sr.AdjustedDemand, &sr.CurrentStock,
			&sr.InFlightQty, &suggestedBy, &computedAt, &sr.Status, &retailerName, &skuName,
		); err != nil {
			return nil, err
		}
		sr.SuggestedByDate = suggestedBy.String()
		sr.ComputedAt = computedAt.UTC().Format(time.RFC3339Nano)
		if retailerName.Valid {
			sr.RetailerName = retailerName.StringVal
		}
		if skuName.Valid {
			sr.SkuName = skuName.StringVal
		}
		rows = append(rows, sr)
	}
	return rows, nil
}

func (a *SuggestionsAPI) retailerLinkedToSupplier(ctx context.Context, supplierID, retailerID string) bool {
	retailerID = strings.TrimSpace(retailerID)
	if retailerID == "" {
		return false
	}
	iter := a.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT 1 FROM Orders WHERE SupplierId = @supplierId AND RetailerId = @retailerId LIMIT 1`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"retailerId": retailerID,
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	return err != iterator.Done && err == nil
}

func (a *SuggestionsAPI) createDraftFromSuggestion(ctx context.Context, supplierID, retailerID, sku string) (order.CreateResponse, error) {
	retailerID = strings.TrimSpace(retailerID)
	sku = strings.TrimSpace(sku)
	if retailerID == "" || sku == "" {
		return order.CreateResponse{}, fmt.Errorf("retailer_id_and_sku_required")
	}
	if !a.retailerLinkedToSupplier(ctx, supplierID, retailerID) {
		return order.CreateResponse{}, errRetailerNotLinked
	}

	suggestion, err := a.loadSuggestion(ctx, retailerID, sku)
	if err != nil {
		return order.CreateResponse{}, err
	}
	if suggestion.SuggestedQty <= 0 {
		return order.CreateResponse{}, fmt.Errorf("suggested_qty_zero")
	}

	lat, lng, h3, err := a.loadRetailerLocation(ctx, retailerID)
	if err != nil {
		return order.CreateResponse{}, err
	}

	deliveryDate := suggestion.SuggestedByDate.In(time.UTC)
	createReq := order.CreateRequest{
		LineItems: []order.LineItem{{
			SKU:      sku,
			Quantity: suggestion.SuggestedQty,
		}},
		H3Cell:                h3,
		Lat:                   lat,
		Lng:                   lng,
		DeliveryMode:          order.DeliveryModeScheduled,
		RequestedDeliveryDate: deliveryDate.Format(time.RFC3339),
	}

	resp, err := a.Orders.Create(ctx, retailerID, createReq)
	if err != nil {
		return order.CreateResponse{}, err
	}

	_, updateErr := a.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ReorderSuggestions", map[string]any{
				"RetailerId": retailerID,
				"Sku":        sku,
				"Status":     SuggestionStatusConverted,
			}),
		})
	})
	if updateErr != nil {
		return resp, fmt.Errorf("order_created_but_status_update_failed: %w", updateErr)
	}
	return resp, nil
}

func (a *SuggestionsAPI) loadSuggestion(ctx context.Context, retailerID, sku string) (ReorderSuggestion, error) {
	row, err := a.Spanner.Single().ReadRow(ctx, "ReorderSuggestions", spanner.Key{retailerID, sku},
		[]string{"RetailerId", "Sku", "SuggestedQty", "AdjustedDemand", "CurrentStock", "InFlightQty", "SafetyStock", "SuggestedByDate", "ComputedAt", "Status"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ReorderSuggestion{}, errSuggestionNotFound
		}
		return ReorderSuggestion{}, err
	}
	var s ReorderSuggestion
	var suggestedBy civil.Date
	if err := row.Columns(&s.RetailerId, &s.Sku, &s.SuggestedQty, &s.AdjustedDemand, &s.CurrentStock, &s.InFlightQty, &s.SafetyStock, &suggestedBy, &s.ComputedAt, &s.Status); err != nil {
		return ReorderSuggestion{}, err
	}
	s.SuggestedByDate = suggestedBy
	return s, nil
}

func (a *SuggestionsAPI) loadRetailerLocation(ctx context.Context, retailerID string) (lat, lng float64, h3 string, err error) {
	row, err := a.Spanner.Single().ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Lat", "Lng", "H3Cell"})
	if err != nil {
		return 0, 0, "", err
	}
	var latCol, lngCol spanner.NullFloat64
	var h3Col spanner.NullString
	if err := row.Columns(&latCol, &lngCol, &h3Col); err != nil {
		return 0, 0, "", err
	}
	if !latCol.Valid || !lngCol.Valid {
		return 0, 0, "", errRetailerLocationMissing
	}
	h3 = strings.TrimSpace(h3Col.StringVal)
	return latCol.Float64, lngCol.Float64, h3, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
