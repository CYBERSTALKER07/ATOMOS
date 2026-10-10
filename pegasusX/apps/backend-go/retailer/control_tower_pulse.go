package retailer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// ControlTowerPulse is an honest retailer ops digest — never demo/simulator data.
type ControlTowerPulse struct {
	RetailerID         string                       `json:"retailer_id"`
	GeneratedAt        string                       `json:"generated_at"`
	OpenOrders         int                          `json:"open_orders"`
	ActiveFulfillments int                          `json:"active_fulfillments"`
	DockPending        int                          `json:"dock_pending"`
	PosOpenSessions    int                          `json:"pos_open_sessions"`
	OpenShifts         int                          `json:"open_shifts"`
	OpenAssistTickets  int                          `json:"open_assist_tickets"`
	LowStockSkuBins    int                          `json:"low_stock_sku_bins"`
	ShiftVariances7d   int                          `json:"shift_variances_7d"`
	SalesMinor7d       int64                        `json:"sales_minor_7d"`
	Capabilities       []string                     `json:"capabilities"`
	Empty              bool                         `json:"empty"`
	Source             string                       `json:"source"`
	OrdersByStatus     map[string]int               `json:"orders_by_status"`
	OrdersBySupplier   []RetailerSupplierOrderFacet `json:"orders_by_supplier"`
	Loyalty            RetailerPulseLoyalty         `json:"loyalty"`
}

// HandleControlTowerPulse serves GET /v1/retailer/control-tower/pulse
func (s *Service) HandleControlTowerPulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// Any authenticated retailer staff may view ops pulse; prefer reports.view but allow CORE roles with stock.view too.
	if !auth.HasRetailerPerm(claims, auth.PermReportsView) &&
		!auth.HasRetailerPerm(claims, auth.PermStockView) &&
		!auth.HasRetailerPerm(claims, auth.PermOrderPlace) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	pulse, err := s.buildControlTowerPulse(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "control_tower_pulse_failed"})
		return
	}
	writeJSON(w, http.StatusOK, pulse)
}

func (s *Service) buildControlTowerPulse(ctx context.Context, orgID string) (ControlTowerPulse, error) {
	now := s.now().UTC()
	from7 := now.Add(-7 * 24 * time.Hour)

	var openOrders, activeFulfillments, dockPending int
	if s.repo != nil {
		orders, err := s.listRetailerTrackingOrders(ctx, orgID)
		if err != nil {
			return ControlTowerPulse{}, fmt.Errorf("control tower tracking: %w", err)
		}
		for _, o := range orders {
			st := o.Status
			if st != "COMPLETED" && st != "CANCELLED" && st != "DELIVERED" {
				openOrders++
			}
			switch st {
			case "ASSIGNED", "IN_TRANSIT", "ARRIVED", "OUT_FOR_DELIVERY", "LOADED":
				activeFulfillments++
			case "ARRIVED_SHOP_CLOSED", "AT_DOORSTEP", "PENDING_RECEIVE":
				dockPending++
			}
		}
	}

	posOpen, err := s.countOpenPosSessions(ctx, orgID)
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower pos: %w", err)
	}

	shifts, err := s.listShifts(ctx, orgID, "", 100)
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower shifts: %w", err)
	}
	openShifts := 0
	variances := 0
	for _, sh := range shifts {
		if sh.Status == ShiftOpen {
			openShifts++
		}
		if sh.Status == ShiftClosed && sh.VarianceMinor != nil && abs64(*sh.VarianceMinor) > 0 {
			variances++
		}
	}

	tickets, err := s.listAssistTickets(ctx, orgID, "", AssistOpen, 100)
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower assist: %w", err)
	}
	openAssist := len(tickets)

	balances, err := s.listStockBalances(ctx, orgID, "")
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower stock: %w", err)
	}
	lowStock := 0
	for _, b := range balances {
		if b.OnHand > 0 && b.OnHand <= 5 {
			lowStock++
		}
	}

	salesMinor, _, _, err := s.aggregateSales(ctx, orgID, "", from7, now)
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower sales: %w", err)
	}

	caps, err := s.LoadEnabledPacks(ctx, orgID)
	if err != nil {
		return ControlTowerPulse{}, fmt.Errorf("control tower packs: %w", err)
	}
	capList := caps.WithCORE().List()

	orderRows, orderSource, err := s.loadRetailerDashboardOrders(ctx, orgID)
	if err != nil {
		return ControlTowerPulse{}, err
	}
	ordersByStatus, ordersBySupplier, rollupOpen := applyRetailerOrderRows(orderRows)
	if orderSource != "empty" {
		openOrders = rollupOpen
	}

	pulse := ControlTowerPulse{
		RetailerID:         orgID,
		GeneratedAt:        now.Format(time.RFC3339Nano),
		OpenOrders:         openOrders,
		ActiveFulfillments: activeFulfillments,
		DockPending:        dockPending,
		PosOpenSessions:    posOpen,
		OpenShifts:         openShifts,
		OpenAssistTickets:  openAssist,
		LowStockSkuBins:    lowStock,
		ShiftVariances7d:   variances,
		SalesMinor7d:       salesMinor,
		Capabilities:       capList,
		Source:             orderSource,
		OrdersByStatus:     ordersByStatus,
		OrdersBySupplier:   ordersBySupplier,
		Loyalty:            RetailerPulseLoyalty{Enrolled: false},
	}
	pulse.Empty = pulse.OpenOrders == 0 &&
		pulse.ActiveFulfillments == 0 &&
		pulse.DockPending == 0 &&
		pulse.PosOpenSessions == 0 &&
		pulse.OpenShifts == 0 &&
		pulse.OpenAssistTickets == 0 &&
		pulse.LowStockSkuBins == 0 &&
		pulse.ShiftVariances7d == 0 &&
		pulse.SalesMinor7d == 0 &&
		rollupOpen == 0
	return pulse, nil
}

// countOpenPosSessions prefers Spanner (multi-pod honest). Memory only when Spanner is unset.
func (s *Service) countOpenPosSessions(ctx context.Context, orgID string) (int, error) {
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(1) FROM RetailerPosSessions
				WHERE RetailerId = @rid AND Status = @st`,
			Params: map[string]any{"rid": orgID, "st": PosSessionOpen},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		var n int64
		if err := row.Columns(&n); err != nil {
			return 0, err
		}
		return int(n), nil
	}
	n := 0
	s.mu.RLock()
	if s.posSessions != nil {
		for _, sess := range s.posSessions {
			if sess.RetailerID == orgID && sess.Status == PosSessionOpen {
				n++
			}
		}
	}
	s.mu.RUnlock()
	return n, nil
}
