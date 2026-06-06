package payload

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
)

// PortalManifestLister loads supplier-portal manifest projections (order aggregation).
type PortalManifestLister interface {
	ListPortalManifests(ctx context.Context, supplierID string) ([]manifest.PortalRow, error)
}

// SetPortalManifestLister wires the supplier portal projection for ADMIN manifest reads.
func (s *Service) SetPortalManifestLister(lister PortalManifestLister) {
	s.portalLister = lister
}

func (s *Service) listManifestWiresLocked(stateFilter, truckFilter string) []manifest.Wire {
	orderByID := make(map[string]OrderRow, len(s.orders))
	for i := range s.orders {
		orderByID[s.orders[i].OrderID] = s.orders[i]
	}
	rows := append([]ManifestRow(nil), s.manifests...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	wire := make([]manifest.Wire, 0, len(rows))
	for i := range rows {
		m := rows[i]
		orders := append([]ManifestOrder(nil), s.manifestOrders[m.ManifestID]...)
		payloadOrders := make([]manifest.PayloadOrderRow, 0, len(orders))
		for j := range orders {
			or := orderByID[orders[j].OrderID]
			payloadOrders = append(payloadOrders, manifest.PayloadOrderRow{
				OrderID:     orders[j].OrderID,
				State:       orders[j].State,
				Amount:      or.TotalMinor,
				RouteID:     or.RouteID,
				WarehouseID: "",
				RetailerID:  "",
			})
		}
		w := manifest.FromPayloadRow(manifest.PayloadRow{
			ManifestID:    m.ManifestID,
			VehicleID:     m.VehicleID,
			DriverID:      m.DriverID,
			State:         m.State,
			TotalVolumeVU: m.TotalVolumeVU,
			MaxVolumeVU:   m.MaxVolumeVU,
			StopCount:     m.StopCount,
			RegionCode:    "UZ-TAS",
			SealedAt:      m.SealedAt,
			CreatedAt:     m.CreatedAt,
			UpdatedAt:     m.UpdatedAt,
			OverflowCount: int(s.overflowCount[m.ManifestID]),
			Orders:        payloadOrders,
			DriverName:    m.DriverID,
			VehiclePlate:  plateForVehicleLocked(s, m.VehicleID),
		})
		if manifest.MatchesStateFilter(w, stateFilter) && manifest.MatchesTruckFilter(w, truckFilter) {
			wire = append(wire, w)
		}
	}
	return wire
}

func (s *Service) manifestDetailWireLocked(manifestID string) (manifest.Wire, bool) {
	idx := s.findManifestIndexLocked(manifestID)
	if idx < 0 {
		return manifest.Wire{}, false
	}
	m := s.manifests[idx]
	orderByID := make(map[string]OrderRow, len(s.orders))
	for i := range s.orders {
		orderByID[s.orders[i].OrderID] = s.orders[i]
	}
	orders := append([]ManifestOrder(nil), s.manifestOrders[manifestID]...)
	payloadOrders := make([]manifest.PayloadOrderRow, 0, len(orders))
	for j := range orders {
		or := orderByID[orders[j].OrderID]
		payloadOrders = append(payloadOrders, manifest.PayloadOrderRow{
			OrderID: orders[j].OrderID,
			State:   orders[j].State,
			Amount:  or.TotalMinor,
			RouteID: or.RouteID,
		})
	}
	return manifest.FromPayloadRow(manifest.PayloadRow{
		ManifestID:    m.ManifestID,
		VehicleID:     m.VehicleID,
		DriverID:      m.DriverID,
		State:         m.State,
		TotalVolumeVU: m.TotalVolumeVU,
		MaxVolumeVU:   m.MaxVolumeVU,
		StopCount:     m.StopCount,
		RegionCode:    "UZ-TAS",
		SealedAt:      m.SealedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		OverflowCount: int(s.overflowCount[manifestID]),
		Orders:        payloadOrders,
		DriverName:    m.DriverID,
		VehiclePlate:  plateForVehicleLocked(s, m.VehicleID),
	}), true
}

// HandleManifestsList serves GET /v1/payloader/manifests and GET /v1/supplier/manifests.
func (s *Service) HandleManifestsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	stateFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
	if stateFilter == "" {
		stateFilter = strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	}
	truckFilter := strings.TrimSpace(r.URL.Query().Get("truck_id"))
	if truckFilter == "" {
		truckFilter = strings.TrimSpace(r.URL.Query().Get("vehicle_id"))
	}

	claims, hasClaims := auth.FromContext(r.Context())
	usePortal := hasClaims && claims.Role == auth.RoleAdmin && truckFilter == "" && stateFilter == ""

	var wire []manifest.Wire
	if usePortal && s.portalLister != nil {
		rows, err := s.portalLister.ListPortalManifests(r.Context(), s.supplierID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_manifests_failed"})
			return
		}
		wire = make([]manifest.Wire, 0, len(rows))
		for i := range rows {
			w := manifest.FromPortalRow(rows[i])
			if manifest.MatchesStateFilter(w, stateFilter) && manifest.MatchesTruckFilter(w, truckFilter) {
				wire = append(wire, w)
			}
		}
	} else {
		s.mu.Lock()
		s.ensureDemoDataLocked()
		wire = s.listManifestWiresLocked(stateFilter, truckFilter)
		s.mu.Unlock()
	}

	writeJSON(w, http.StatusOK, map[string]any{"manifests": wire})
}

// HandleManifestDetail serves manifest detail for payloader and supplier paths.
func (s *Service) HandleManifestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	manifestID := manifestIDParam(r)
	if manifestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}

	s.mu.Lock()
	s.ensureDemoDataLocked()
	wire, ok := s.manifestDetailWireLocked(manifestID)
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, wire)
}
