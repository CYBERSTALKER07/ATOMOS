package payload

import (
	"context"
	"sync/atomic"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox" 
	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
	"google.golang.org/api/iterator"
)

// ShipUnit is one SSCC logistics unit on a supplier truck manifest.
type ShipUnit struct {
	ManifestID string    `json:"manifest_id"`
	ShipUnitID string    `json:"ship_unit_id"`
	SSCC       string    `json:"sscc"`
	OrderID    string    `json:"order_id"`
	Sequence   int64     `json:"sequence"`
	GTIN       string    `json:"gtin,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// EnsureShipUnitsForManifest mints SSCC rows for each ManifestOrders line (idempotent).
func (s *Service) EnsureShipUnitsForManifest(ctx context.Context, manifestID string) (int, error) {
	if s == nil || !gs1.LabelsEnabled() {
		return 0, nil
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return 0, fmt.Errorf("manifest_id_required")
	}
	prefix, err := s.loadGs1CompanyPrefix(ctx)
	if err != nil || strings.TrimSpace(prefix) == "" {
		if s.log != nil {
			s.log.Info("gs1 sscc skipped: no company prefix", "manifest_id", manifestID, "err", err)
		}
		return 0, nil
	}
	orderIDs := s.manifestOrderIDs(ctx, manifestID)
	if len(orderIDs) == 0 {
		return 0, nil
	}
	existing, _ := s.ListShipUnits(ctx, manifestID)
	have := map[string]bool{}
	for _, u := range existing {
		have[u.OrderID] = true
	}
	
	newUnits := []ShipUnit{}
	seq := int64(len(existing))
	for _, oid := range orderIDs {
		if have[oid] {
			continue
		}
		serial := ssccSerial(manifestID, oid, seq)
		sscc, err := gs1.GenerateSSCC(prefix, serial)
		if err != nil {
			return 0, err
		}
		unit := ShipUnit{
			ManifestID: manifestID,
			ShipUnitID: uuid.NewString(),
			SSCC:       sscc,
			OrderID:    oid,
			Sequence:   seq,
			CreatedAt:  s.now().UTC(),
		}
		newUnits = append(newUnits, unit)
		seq++
	}
	
	if len(newUnits) == 0 {
		return 0, nil
	}
	
	err = s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		if err := tx.SaveShipUnits(ctx, newUnits); err != nil {
			return err
		}
		// Fallback to in-memory
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.shipUnits == nil {
			s.shipUnits = map[string][]ShipUnit{}
		}
		s.shipUnits[manifestID] = append(s.shipUnits[manifestID], newUnits...)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateRoute, manifestID, events.TopicMain, map[string]any{
			"type":          "EventShipUnitsGenerated",
			"manifest_id":   manifestID,
			"units_count":   len(newUnits),
			"supplier_id":   s.resolveSupplierScope(ctx),
			"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	
	if err != nil {
		return 0, err
	}
	
	return len(newUnits), nil
}

func (s *Service) manifestOrderIDs(ctx context.Context, manifestID string) []string {
	s.mu.RLock()
	orders := s.manifestOrders[manifestID]
	s.mu.RUnlock()
	out := make([]string, 0, len(orders))
	for _, o := range orders {
		if o.OrderID == "" || o.State == "REMOVED_REASSIGNED" {
			continue
		}
		out = append(out, o.OrderID)
	}
	if len(out) > 0 {
		return out
	}
	client := s.spannerClient()
	if client == nil {
		return out
	}
	stmt := spanner.Statement{
		SQL:    `SELECT OrderId, State FROM ManifestOrders WHERE ManifestId = @mid`,
		Params: map[string]any{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		var oid, state string
		if err := row.Columns(&oid, &state); err != nil {
			continue
		}
		if oid != "" && state != "REMOVED_REASSIGNED" {
			out = append(out, oid)
		}
	}
	return out
}

func (s *Service) loadGs1CompanyPrefix(ctx context.Context) (string, error) {
	if override := strings.TrimSpace(os.Getenv("GS1_COMPANY_PREFIX")); override != "" {
		return override, nil
	}
	client := s.spannerClient()
	if client == nil {
		return "", nil
	}
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if sid == "" {
		return "", nil
	}
	row, err := client.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{sid}, []string{"Gs1CompanyPrefix"})
	if err != nil {
		return "", nil // soft: column missing or no profile
	}
	var prefix spanner.NullString
	if err := row.Columns(&prefix); err != nil {
		return "", nil
	}
	return strings.TrimSpace(prefix.StringVal), nil
}

func (s *Service) spannerClient() *spanner.Client {
	if store, ok := s.repo.(interface{ SpannerClient() *spanner.Client }); ok {
		return store.SpannerClient()
	}
	return nil
}

var ssccGlobalCounter uint64 = uint64(time.Now().UnixNano())

func ssccSerial(manifestID, orderID string, seq int64) uint64 {
	return atomic.AddUint64(&ssccGlobalCounter, 1)
}

func (s *Service) insertShipUnit(ctx context.Context, u ShipUnit) error {
	client := s.spannerClient()
	if client == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.shipUnits == nil {
			s.shipUnits = map[string][]ShipUnit{}
		}
		s.shipUnits[u.ManifestID] = append(s.shipUnits[u.ManifestID], u)
		return nil
	}
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ManifestShipUnits", map[string]any{
			"ManifestId": u.ManifestID,
			"ShipUnitId": u.ShipUnitID,
			"Sscc":       u.SSCC,
			"OrderId":    u.OrderID,
			"Sequence":   u.Sequence,
			"Gtin":       nullableStr(u.GTIN),
			"CreatedAt":  spanner.CommitTimestamp,
		}),
	})
	return err
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// ListShipUnits returns SSCC units for a manifest.
func (s *Service) ListShipUnits(ctx context.Context, manifestID string) ([]ShipUnit, error) {
	manifestID = strings.TrimSpace(manifestID)
	client := s.spannerClient()
	if client == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return append([]ShipUnit(nil), s.shipUnits[manifestID]...), nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, ShipUnitId, Sscc, OrderId, Sequence, Gtin, CreatedAt
			FROM ManifestShipUnits WHERE ManifestId = @mid ORDER BY Sequence`,
		Params: map[string]any{"mid": manifestID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ShipUnit, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var u ShipUnit
		var gtin spanner.NullString
		var created time.Time
		if err := row.Columns(&u.ManifestID, &u.ShipUnitID, &u.SSCC, &u.OrderID, &u.Sequence, &gtin, &created); err != nil {
			return nil, err
		}
		u.GTIN = gtin.StringVal
		u.CreatedAt = created.UTC()
		out = append(out, u)
	}
	return out, nil
}

// HandleListShipUnits GET /v1/payload/manifests/{manifestID}/ship-units
func (s *Service) HandleListShipUnits(w http.ResponseWriter, r *http.Request) {
	if !gs1.LabelsEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "gs1_labels_disabled"})
		return
	}
	mid := manifestIDParam(r)
	units, err := s.ListShipUnits(r.Context(), mid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"manifest_id": mid, "ship_units": units})
}

// HandleManifestLabels POST /v1/payload/manifests/{manifestID}/labels → text/plain ZPL
func (s *Service) HandleManifestLabels(w http.ResponseWriter, r *http.Request) {
	if !gs1.LabelsEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "gs1_labels_disabled"})
		return
	}
	mid := manifestIDParam(r)
	if mid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "manifest_id_required"})
		return
	}
	_, _ = s.EnsureShipUnitsForManifest(r.Context(), mid)
	units, err := s.ListShipUnits(r.Context(), mid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	var filterOrder string
	if r.Body != nil && r.ContentLength != 0 {
		var body struct {
			OrderID string `json:"order_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		filterOrder = strings.TrimSpace(body.OrderID)
	}
	fromGLN, toGLN := s.labelGLNs(r.Context(), mid)
	labels := make([]gs1.LabelData, 0, len(units))
	for _, u := range units {
		if filterOrder != "" && u.OrderID != filterOrder {
			continue
		}
		labels = append(labels, gs1.LabelData{
			SSCC: u.SSCC, GTIN: u.GTIN, OrderID: u.OrderID, ManifestID: mid,
			ShipFromGLN: fromGLN, ShipToGLN: toGLN,
		})
	}
	zpl, err := gs1.MultiLabelZPL(labels)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"manifest_%s.zpl\"", mid))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(zpl))
}

func (s *Service) labelGLNs(ctx context.Context, manifestID string) (fromGLN, toGLN string) {
	client := s.spannerClient()
	if client == nil {
		return "", ""
	}
	sid := strings.TrimSpace(s.resolveSupplierScope(ctx))
	if sid != "" {
		row, err := client.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{sid}, []string{"Gln"})
		if err == nil {
			var g spanner.NullString
			_ = row.Columns(&g)
			fromGLN = g.StringVal
		}
	}
	_ = manifestID
	return fromGLN, toGLN
}

func (s *Service) afterSealAssignSSCC(ctx context.Context, manifestID string) {
	if _, err := s.EnsureShipUnitsForManifest(ctx, manifestID); err != nil && s.log != nil {
		s.log.Warn("gs1 sscc assign failed", "manifest_id", manifestID, "err", err)
	}
}
