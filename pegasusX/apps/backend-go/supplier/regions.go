package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type supplierRegionInput struct {
	RegionID    string `json:"region_id,omitempty"`
	Name        string `json:"name"`
	CountryCode string `json:"country_code,omitempty"`
}

type supplierRegion struct {
	RegionID    string `json:"region_id"`
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
}

// HandleSupplierRegions serves GET/PUT /v1/supplier/regions.
func (s *Service) HandleSupplierRegions(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(s.scopedSupplierID(r))
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "spanner_unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		rows, err := s.listSupplierRegions(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_regions_failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": rows})
	case http.MethodPut:
		var req struct {
			Items []supplierRegionInput `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if err := s.replaceSupplierRegions(r.Context(), sid, req.Items); err != nil {
			s.writePinError(w, err)
			return
		}
		rows, _ := s.listSupplierRegions(r.Context(), sid)
		writeJSON(w, http.StatusOK, map[string]any{"items": rows})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) listSupplierRegions(ctx context.Context, supplierID string) ([]supplierRegion, error) {
	iter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT RegionId, Name, CountryCode FROM SupplierRegions WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	})
	defer iter.Stop()
	out := make([]supplierRegion, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var item supplierRegion
		if err := row.Columns(&item.RegionID, &item.Name, &item.CountryCode); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) replaceSupplierRegions(ctx context.Context, supplierID string, in []supplierRegionInput) error {
	pack, err := auth.FiscalPackForSupplier(supplierID)
	if err != nil {
		return err
	}
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := txn.Update(ctx, spanner.Statement{
			SQL:    `DELETE FROM SupplierRegions WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supplierID},
		}); err != nil {
			return err
		}
		muts := make([]*spanner.Mutation, 0, len(in))
		for _, raw := range in {
			name := strings.TrimSpace(raw.Name)
			if name == "" {
				continue
			}
			country := auth.NormalizeCountryCode(raw.CountryCode)
			if country == "" {
				country = packCountry
			}
			if err := auth.AssertSameMarket(packCountry, country); err != nil {
				return err
			}
			id := strings.TrimSpace(raw.RegionID)
			if id == "" {
				id = uuid.NewString()
			}
			muts = append(muts, spanner.InsertMap("SupplierRegions", map[string]any{
				"SupplierId":  supplierID,
				"RegionId":    id,
				"CountryCode": country,
				"Name":        name,
				"CreatedAt":   now,
				"UpdatedAt":   now,
			}))
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateSupplier, supplierID, events.TopicMain, events.SupplierEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventSupplierUpdated, Timestamp: now.UTC().Format(time.RFC3339Nano)},
			SupplierID: supplierID,
			Action:     "supplier_regions_replaced",
		}); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		if len(muts) == 0 {
			return nil
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (s *Service) supplierRegionExists(ctx context.Context, supplierID, regionID string) bool {
	if s.portalSpanner == nil || strings.TrimSpace(regionID) == "" {
		return false
	}
	_, err := s.portalSpanner.Single().ReadRow(ctx, "SupplierRegions", spanner.Key{supplierID, regionID}, []string{"RegionId"})
	return err == nil
}
