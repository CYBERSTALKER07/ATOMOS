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

func (s *Service) listManifestWiresLocked(stateFilter, truckFilter, warehouseScope string) []manifest.Wire {
	orderByID := make(map[string]OrderRow, len(s.orders))
	for i := range s.orders {
		orderByID[s.orders[i].OrderID] = s.orders[i]
	}
	rows := append([]ManifestRow(nil), s.manifests...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt > rows[j].UpdatedAt })
	wire := make([]manifest.Wire, 0, len(rows))
	for i := range rows {
		m := rows[i]
		// B7 PL-P0-6: JWT warehouse scope filters list (empty row WH still visible).
		if !manifestMatchesWarehouseScope(m.WarehouseID, warehouseScope) {
			continue
		}
		orders := append([]ManifestOrder(nil), s.manifestOrders[m.ManifestID]...)
		payloadOrders := make([]manifest.PayloadOrderRow, 0, len(orders))
		for j := range orders {
			or := orderByID[orders[j].OrderID]
			payloadOrders = append(payloadOrders, manifest.PayloadOrderRow{
				OrderID:     orders[j].OrderID,
				State:       orders[j].State,
				Amount:      or.TotalMinor,
				RouteID:     or.RouteID,
				WarehouseID: m.WarehouseID,
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
		rows, err := s.portalLister.ListPortalManifests(r.Context(), s.resolveSupplierScope(r.Context()))
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
		_ = s.hydrateFromRepo(r.Context())
		whScope := s.resolveWarehouseScope(r.Context())
		s.mu.Lock()
		s.ensureManifestStateLocked()
		wire = s.listManifestWiresLocked(stateFilter, truckFilter, whScope)
		s.mu.Unlock()
	}
	s.attachInboundDriverLocations(r.Context(), wire)

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

	_ = s.hydrateFromRepo(r.Context())
	s.mu.Lock()
	s.ensureManifestStateLocked()
	idx := s.findManifestIndexLocked(manifestID)
	var row ManifestRow
	if idx >= 0 {
		row = s.manifests[idx]
	}
	wire, ok := s.manifestDetailWireLocked(manifestID)
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manifest_not_found"})
		return
	}
	// B7 PL-P0-6: detail for foreign warehouse when both sides set → 403 (not 404 leak).
	if err := s.assertManifestWarehouseScope(r.Context(), row); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_forbidden"})
		return
	}
	wire = s.enrichManifestWireExpectations(r.Context(), wire)
	wires := []manifest.Wire{wire}
	s.attachInboundDriverLocations(r.Context(), wires)
	writeJSON(w, http.StatusOK, wires[0])
}

// attachInboundDriverLocations fills thin inbound map coords on payload manifest wires.
func (s *Service) attachInboundDriverLocations(ctx context.Context, wires []manifest.Wire) {
	if s == nil || s.locations == nil || len(wires) == 0 {
		return
	}
	now := s.now()
	cache := make(map[string]struct {
		lat, lng float64
		live     bool
	}, len(wires))
	for i := range wires {
		driverID := strings.TrimSpace(wires[i].DriverID)
		if driverID == "" {
			continue
		}
		cached, ok := cache[driverID]
		if !ok {
			loc, found, err := s.locations.GetDriverLocation(ctx, driverID)
			if err != nil || !found {
				cache[driverID] = cached
				continue
			}
			lat, lng := loc.Lat, loc.Lng
			if lat == 0 {
				lat = loc.Latitude
			}
			if lng == 0 {
				lng = loc.Longitude
			}
			cached = struct {
				lat, lng float64
				live     bool
			}{lat: lat, lng: lng, live: loc.IsLive(now)}
			cache[driverID] = cached
		}
		if cached.lat == 0 && cached.lng == 0 {
			continue
		}
		lat, lng := cached.lat, cached.lng
		wires[i].DriverLat = &lat
		wires[i].DriverLng = &lng
		wires[i].LiveLocationAvailable = cached.live
	}
}
