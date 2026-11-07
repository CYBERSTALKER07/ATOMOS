package warehouse

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

type warehouseOpsCoverageResponse struct {
	WarehouseID string                 `json:"warehouse_id"`
	Mode        string                 `json:"mode"`
	Cities      []order.CoverageCity   `json:"cities"`
	Pins        []proximity.ServicePin `json:"pins"`
	CountryCode string                 `json:"country_code"`
}

type warehouseOpsSupplyFactoryResponse struct {
	WarehouseID  string `json:"warehouse_id"`
	FactoryID    string `json:"factory_id,omitempty"`
	TransferMode string `json:"transfer_mode,omitempty"`
	Source       string `json:"source"`
	CountryCode  string `json:"country_code"`
}

// HandleOpsCoverage serves GET /v1/warehouse/ops/coverage (GS-R). View only — warehouse cannot re-pin.
func (s *Service) HandleOpsCoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	warehouseID, ok := s.scopedWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	pack, err := auth.CheckoutPackFromContext(r.Context())
	if err != nil {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	country, err := auth.PackCountryCode(pack)
	if err != nil {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	resp := warehouseOpsCoverageResponse{
		WarehouseID: warehouseID,
		Mode:        proximity.CoverageModeCountryClosest,
		Cities:      []order.CoverageCity{},
		Pins:        []proximity.ServicePin{},
		CountryCode: country,
	}
	if s.spannerClient != nil {
		loaded, loadErr := s.loadOpsCoverage(r.Context(), warehouseID)
		if loadErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_coverage_failed"})
			return
		}
		resp.Mode = loaded.Mode
		resp.Cities = loaded.Cities
		resp.Pins = loaded.Pins
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleOpsSupplyFactory serves GET /v1/warehouse/ops/supply-factory (GS-R). Engine only — not a picker.
func (s *Service) HandleOpsSupplyFactory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	warehouseID, ok := s.scopedWarehouseID(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "warehouse_id_required"})
		return
	}
	pack, err := auth.CheckoutPackFromContext(r.Context())
	if err != nil {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	country, err := auth.PackCountryCode(pack)
	if err != nil {
		status, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	resolve := s.resolveSupplyContext
	if resolve == nil {
		if s.spannerClient == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "factory_lookup_unavailable"})
			return
		}
		resolve = s.resolveWarehouseSupplyContext
	}
	supply, err := resolve(r.Context(), warehouseID)
	if err != nil {
		if writeMarketLaw(w, err) {
			return
		}
		if errors.Is(err, proximity.ErrFactoryUnassigned) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": proximity.ErrFactoryUnassigned.Error()})
			return
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, warehouseOpsSupplyFactoryResponse{
		WarehouseID:  warehouseID,
		FactoryID:    supply.FactoryID,
		TransferMode: supply.TransferMode,
		Source:       "engine",
		CountryCode:  country,
	})
}

func (s *Service) loadOpsCoverage(ctx context.Context, warehouseID string) (warehouseOpsCoverageResponse, error) {
	supplierID := s.resolveSupplierScope(ctx)
	pins, err := s.listOpsPins(ctx, supplierID, warehouseID)
	if err != nil {
		return warehouseOpsCoverageResponse{}, err
	}
	cities := s.listOpsCoverageCities(ctx, warehouseID)
	cells := s.listOpsCoverageCells(ctx, supplierID, warehouseID)
	mode := proximity.EffectiveCoverageMode(proximity.WarehouseCandidate{
		WarehouseID:   warehouseID,
		CoverageCells: cells,
	}, pins)
	return warehouseOpsCoverageResponse{
		WarehouseID: warehouseID,
		Mode:        mode,
		Cities:      cities,
		Pins:        pins,
	}, nil
}

func (s *Service) listOpsPins(ctx context.Context, supplierID, warehouseID string) ([]proximity.ServicePin, error) {
	if s.spannerClient == nil {
		return []proximity.ServicePin{}, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, TargetType, TargetId, Priority
		      FROM ServicePins WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	out := make([]proximity.ServicePin, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var pin proximity.ServicePin
		if err := row.Columns(&pin.WarehouseID, &pin.TargetType, &pin.TargetID, &pin.Priority); err != nil {
			continue
		}
		out = append(out, pin)
	}
	return out, nil
}

func (s *Service) listOpsCoverageCities(ctx context.Context, warehouseID string) []order.CoverageCity {
	if s.spannerClient == nil {
		return []order.CoverageCity{}
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT CityName, Lat, Lng FROM WarehouseCoverageCities WHERE WarehouseId = @wid ORDER BY CityName`,
		Params: map[string]any{"wid": warehouseID},
	})
	defer iter.Stop()
	out := make([]order.CoverageCity, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var city order.CoverageCity
		if err := row.Columns(&city.Name, &city.Lat, &city.Lng); err != nil {
			continue
		}
		out = append(out, city)
	}
	return out
}

func (s *Service) listOpsCoverageCells(ctx context.Context, supplierID, warehouseID string) []string {
	if s.spannerClient == nil {
		return nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT H3Cell FROM WarehouseCoverageCells WHERE SupplierId = @sid AND WarehouseId = @wid`,
		Params: map[string]any{"sid": supplierID, "wid": warehouseID},
	})
	defer iter.Stop()
	out := make([]string, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var cell string
		if err := row.Columns(&cell); err == nil && strings.TrimSpace(cell) != "" {
			out = append(out, cell)
		}
	}
	return out
}
