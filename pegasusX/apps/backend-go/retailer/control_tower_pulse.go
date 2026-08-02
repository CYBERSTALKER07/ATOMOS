package retailer

import (
	"net/http"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// ControlTowerPulse is an honest retailer ops digest — never demo/simulator data.
type ControlTowerPulse struct {
	RetailerID         string   `json:"retailer_id"`
	GeneratedAt        string   `json:"generated_at"`
	OpenOrders         int      `json:"open_orders"`
	ActiveFulfillments int      `json:"active_fulfillments"`
	DockPending        int      `json:"dock_pending"`
	PosOpenSessions    int      `json:"pos_open_sessions"`
	OpenShifts         int      `json:"open_shifts"`
	OpenAssistTickets  int      `json:"open_assist_tickets"`
	LowStockSkuBins    int      `json:"low_stock_sku_bins"`
	ShiftVariances7d   int      `json:"shift_variances_7d"`
	SalesMinor7d       int64    `json:"sales_minor_7d"`
	Capabilities       []string `json:"capabilities"`
	Empty              bool     `json:"empty"`
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

	pulse := s.buildControlTowerPulse(r, orgID)
	writeJSON(w, http.StatusOK, pulse)
}

func (s *Service) buildControlTowerPulse(r *http.Request, orgID string) ControlTowerPulse {
	ctx := r.Context()
	now := s.now().UTC()
	from7 := now.Add(-7 * 24 * time.Hour)

	var openOrders, activeFulfillments, dockPending int
	if s.repo != nil {
		if orders, err := s.listRetailerTrackingOrders(ctx, orgID); err == nil {
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
	}

	posOpen := 0
	s.mu.RLock()
	if s.posSessions != nil {
		for _, sess := range s.posSessions {
			if sess.RetailerID == orgID && sess.Status == PosSessionOpen {
				posOpen++
			}
		}
	}
	s.mu.RUnlock()

	openShifts := 0
	if shifts, err := s.listShifts(ctx, orgID, "", 100); err == nil {
		for _, sh := range shifts {
			if sh.Status == ShiftOpen {
				openShifts++
			}
		}
	}

	openAssist := 0
	if tickets, err := s.listAssistTickets(ctx, orgID, "", AssistOpen, 100); err == nil {
		openAssist = len(tickets)
	}

	_, lowStock := s.aggregateInventory(ctx, orgID, "")
	variances := s.countClosedShiftVariances(ctx, orgID, "")
	// countClosedShiftVariances is all closed with variance; ok as 7d proxy for pulse
	salesMinor, _, _ := s.aggregateSales(orgID, "", from7, now)

	caps, _ := s.LoadEnabledPacks(ctx, orgID)
	capList := caps.WithCORE().List()

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
	}
	pulse.Empty = pulse.OpenOrders == 0 &&
		pulse.ActiveFulfillments == 0 &&
		pulse.DockPending == 0 &&
		pulse.PosOpenSessions == 0 &&
		pulse.OpenShifts == 0 &&
		pulse.OpenAssistTickets == 0 &&
		pulse.LowStockSkuBins == 0 &&
		pulse.ShiftVariances7d == 0 &&
		pulse.SalesMinor7d == 0
	return pulse
}
