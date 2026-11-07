package twin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// RouteException is an open operational exception on a route.
type RouteException struct {
	Type    string `json:"type"`
	OrderID string `json:"order_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// OpsRouteView enriches RouteTwinView for the live ops map.
type OpsRouteView struct {
	RouteTwinView
	DriverName   string           `json:"driver_name,omitempty"`
	DriverScore  int64            `json:"driver_score,omitempty"`
	Lateness     string           `json:"lateness"` // green | amber | red
	HasShopClosed bool            `json:"has_shop_closed"`
	Exceptions   []RouteException `json:"exceptions"`
}

// SupplierHTTPHandler serves supplier-scoped twin read APIs.
type SupplierHTTPHandler struct {
	repo       *SpannerRepository
	supplierID func(r *http.Request) string
	now        func() time.Time
}

func NewSupplierHTTPHandler(repo *SpannerRepository, supplierID func(r *http.Request) string) *SupplierHTTPHandler {
	return &SupplierHTTPHandler{
		repo:       repo,
		supplierID: supplierID,
		now:        time.Now,
	}
}

func (h *SupplierHTTPHandler) ListActiveRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := strings.TrimSpace(h.supplierID(r))
	if sid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.repo == nil {
		writeTwinJSON(w, []OpsRouteView{})
		return
	}

	zoneH3 := strings.TrimSpace(r.URL.Query().Get("zoneH3"))
	delayedOnly := strings.EqualFold(r.URL.Query().Get("delayedOnly"), "true")
	shopClosedOnly := strings.EqualFold(r.URL.Query().Get("shopClosedOnly"), "true")
	driverID := strings.TrimSpace(r.URL.Query().Get("driverId"))

	views, err := h.listActiveForSupplier(r.Context(), sid, zoneH3, delayedOnly, shopClosedOnly, driverID)
	if err != nil {
		http.Error(w, "list_active_routes_failed", http.StatusInternalServerError)
		return
	}
	if views == nil {
		views = []OpsRouteView{}
	}
	writeTwinJSON(w, views)
}

func (h *SupplierHTTPHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := strings.TrimSpace(h.supplierID(r))
	if sid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	routeID := strings.TrimSpace(chi.URLParam(r, "routeID"))
	if routeID == "" {
		routeID = routeIDFromPath(r.URL.Path)
	}
	if routeID == "" {
		http.Error(w, "route_id_required", http.StatusBadRequest)
		return
	}
	view, err := h.getRouteForSupplier(r.Context(), sid, routeID)
	if err != nil {
		http.Error(w, "get_route_failed", http.StatusInternalServerError)
		return
	}
	if view == nil {
		http.Error(w, "not_found", http.StatusNotFound)
		return
	}
	writeTwinJSON(w, view)
}

func (h *SupplierHTTPHandler) GetRouteInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := strings.TrimSpace(h.supplierID(r))
	if sid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	routeID := strings.TrimSpace(chi.URLParam(r, "routeID"))
	if routeID == "" {
		routeID = routeIDFromInventoryPath(r.URL.Path)
	}
	if routeID == "" {
		http.Error(w, "route_id_required", http.StatusBadRequest)
		return
	}
	ok, err := h.repo.routeBelongsToSupplier(r.Context(), routeID, sid)
	if err != nil {
		http.Error(w, "route_scope_failed", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "not_found", http.StatusNotFound)
		return
	}
	inv, err := h.repo.GetVehicleInventory(r.Context(), routeID)
	if err != nil {
		http.Error(w, "inventory_failed", http.StatusInternalServerError)
		return
	}
	if inv == nil {
		inv = []VehicleInventory{}
	}
	writeTwinJSON(w, inv)
}

func routeIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "routes" && i+1 < len(parts) {
			next := parts[i+1]
			if next != "active" {
				return next
			}
		}
	}
	return ""
}

func routeIDFromInventoryPath(path string) string {
	id := routeIDFromPath(path)
	if id == "" {
		return ""
	}
	return strings.TrimSuffix(id, "/inventory")
}

func writeTwinJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (h *SupplierHTTPHandler) listActiveForSupplier(ctx context.Context, supplierID, zoneH3 string, delayedOnly, shopClosedOnly bool, driverID string) ([]OpsRouteView, error) {
	routes, err := h.repo.ListActiveRouteTwinsForSupplier(ctx, supplierID, zoneH3, driverID)
	if err != nil {
		return nil, err
	}
	now := h.now()
	out := make([]OpsRouteView, 0, len(routes))
	for _, rt := range routes {
		view, err := h.enrichRoute(ctx, supplierID, rt, now)
		if err != nil {
			return nil, err
		}
		if delayedOnly && view.Lateness != "red" && view.Lateness != "amber" {
			continue
		}
		if shopClosedOnly && !view.HasShopClosed {
			continue
		}
		out = append(out, view)
	}
	return out, nil
}

func (h *SupplierHTTPHandler) getRouteForSupplier(ctx context.Context, supplierID, routeID string) (*OpsRouteView, error) {
	ok, err := h.repo.routeBelongsToSupplier(ctx, routeID, supplierID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	rt, err := h.repo.GetRouteTwin(ctx, routeID)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, nil
	}
	view, err := h.enrichRoute(ctx, supplierID, *rt, h.now())
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (h *SupplierHTTPHandler) enrichRoute(ctx context.Context, supplierID string, rt RouteTwinView, now time.Time) (OpsRouteView, error) {
	view := OpsRouteView{
		RouteTwinView: rt,
		Lateness:      classifyLateness(rt.Stops, now),
	}
	name, score, err := h.repo.driverProfile(ctx, rt.DriverID)
	if err != nil {
		return OpsRouteView{}, err
	}
	view.DriverName = name
	view.DriverScore = score
	exceptions, shopClosed, err := h.repo.routeExceptions(ctx, supplierID, rt.RouteID)
	if err != nil {
		return OpsRouteView{}, err
	}
	view.Exceptions = exceptions
	view.HasShopClosed = shopClosed
	return view, nil
}

func classifyLateness(stops []StopTwin, now time.Time) string {
	lateness := "green"
	for _, st := range stops {
		if strings.EqualFold(st.Status, "COMPLETED") || strings.EqualFold(st.Status, "CANCELLED") {
			continue
		}
		if st.WindowEnd != nil && now.After(*st.WindowEnd) {
			return "red"
		}
		if st.PredictedArrival != nil && st.WindowEnd != nil && st.PredictedArrival.After(*st.WindowEnd) {
			lateness = "red"
		} else if st.WindowStart != nil && now.After(*st.WindowStart) && lateness != "red" {
			lateness = "amber"
		}
	}
	return lateness
}

func (r *SpannerRepository) ListActiveRouteTwinsForSupplier(ctx context.Context, supplierID, zoneH3, driverID string) ([]RouteTwinView, error) {
	sql := `SELECT rt.RouteId, rt.SupplierId, rt.DriverId, rt.Status, rt.CurrentLat, rt.CurrentLng, rt.CurrentH3,
	               rt.LocationAt, rt.RemainingStops, rt.CapacityUsedWeight, rt.CapacityUsedVolume,
	               rt.LastEventAt, rt.UpdatedAt
	        FROM RouteTwins rt
	        INNER JOIN Drivers d ON d.DriverId = rt.DriverId AND d.SupplierId = @supplierId
	        WHERE rt.Status = 'ACTIVE'`
	params := map[string]interface{}{"supplierId": supplierID}
	if zoneH3 != "" {
		sql += " AND rt.CurrentH3 = @zoneH3"
		params["zoneH3"] = zoneH3
	}
	if driverID != "" {
		sql += " AND rt.DriverId = @driverId"
		params["driverId"] = driverID
	}

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var active []RouteTwin
	if err := iter.Do(func(row *spanner.Row) error {
		var rt RouteTwin
		if err := row.ToStruct(&rt); err != nil {
			return err
		}
		active = append(active, rt)
		return nil
	}); err != nil {
		return nil, err
	}

	var routes []RouteTwinView
	for _, rt := range active {
		full, err := r.GetRouteTwin(ctx, rt.RouteID)
		if err != nil {
			return nil, err
		}
		if full != nil {
			routes = append(routes, *full)
		}
	}
	if routes == nil {
		routes = []RouteTwinView{}
	}
	return routes, nil
}

func (r *SpannerRepository) routeBelongsToSupplier(ctx context.Context, routeID, supplierID string) (bool, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT 1
		      FROM RouteTwins rt
		      INNER JOIN Drivers d ON d.DriverId = rt.DriverId AND d.SupplierId = @supplierId
		      WHERE rt.RouteId = @routeId
		      LIMIT 1`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"routeId":    routeID,
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	return err == nil, err
}

func (r *SpannerRepository) driverProfile(ctx context.Context, driverID string) (name string, score int64, err error) {
	row, err := r.client.Single().ReadRow(ctx, "Drivers", spanner.Key{driverID}, []string{"Name"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return "", 0, nil
		}
		return "", 0, err
	}
	if err := row.Column(0, &name); err != nil {
		return "", 0, err
	}
	scoreRow, scoreErr := r.client.Single().ReadRow(ctx, "DriverScores", spanner.Key{driverID}, []string{"Score"})
	if scoreErr == nil {
		_ = scoreRow.Column(0, &score)
	}
	return name, score, nil
}

func (r *SpannerRepository) routeExceptions(ctx context.Context, supplierID, routeID string) ([]RouteException, bool, error) {
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId, Status FROM Orders
		      WHERE SupplierId = @supplierId AND RouteId = @routeId
		        AND Status IN ('SHOP_CLOSED_PENDING', 'ARRIVED_SHOP_CLOSED', 'CASH_DISPUTED')
		      LIMIT 20`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"routeId":    routeID,
		},
	})
	defer iter.Stop()

	var out []RouteException
	shopClosed := false
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, false, err
		}
		var orderID, status string
		if err := row.Columns(&orderID, &status); err != nil {
			return nil, false, err
		}
		excType := "ORDER"
		if strings.Contains(status, "SHOP_CLOSED") {
			excType = "SHOP_CLOSED"
			shopClosed = true
		} else if strings.Contains(status, "CASH") {
			excType = "CASH"
		}
		out = append(out, RouteException{
			Type:    excType,
			OrderID: orderID,
			Status:  status,
		})
	}

	cashIter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT cr.ReconciliationId, cr.Status, cr.DifferenceMinor
		      FROM CashReconciliations cr
		      WHERE cr.RouteId = @routeId AND cr.Status IN ('PENDING', 'DISPUTED')
		      LIMIT 10`,
		Params: map[string]interface{}{"routeId": routeID},
	})
	defer cashIter.Stop()
	for {
		row, err := cashIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, shopClosed, err
		}
		var reconID, status string
		var diff int64
		if err := row.Columns(&reconID, &status, &diff); err != nil {
			return nil, shopClosed, err
		}
		out = append(out, RouteException{
			Type:   "CASH_DISCREPANCY",
			Status: status,
			Detail: fmt.Sprintf("difference_minor=%d", diff),
		})
	}
	return out, shopClosed, nil
}
