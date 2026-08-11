package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// BroadcastTemplateWire is a depot or network broadcast starter message.
type BroadcastTemplateWire struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	DefaultRole string `json:"default_role"`
	Scope       string `json:"scope"`
	Source      string `json:"source"`
	WarehouseID string `json:"warehouse_id,omitempty"`
}

type broadcastTemplatesResponse struct {
	Templates []BroadcastTemplateWire `json:"templates"`
}

var builtinWarehouseBroadcastTemplates = []BroadcastTemplateWire{
	{
		ID: "wh_yard_hold", Category: "operations", Scope: "warehouse", Source: "builtin",
		Title: "Yard congestion advisory",
		Body:  "Loading bay congestion at this depot. Drivers: expect queue delays at check-in.",
		DefaultRole: "DRIVER",
	},
	{
		ID: "wh_gate_delay", Category: "operations", Scope: "warehouse", Source: "builtin",
		Title: "Gate delay notice",
		Body:  "Inbound gate processing is slower than usual. Drivers: allow extra time at arrival.",
		DefaultRole: "DRIVER",
	},
	{
		ID: "wh_receiving_hours", Category: "operations", Scope: "warehouse", Source: "builtin",
		Title: "Receiving hours update",
		Body:  "This depot will operate on reduced receiving hours on {date}. Confirm your delivery window.",
		DefaultRole: "RETAILER",
	},
	{
		ID: "wh_check_in_slow", Category: "operations", Scope: "warehouse", Source: "builtin",
		Title: "Slow check-in advisory",
		Body:  "Check-in is taking longer than usual at this warehouse. Reason: {reason}",
		DefaultRole: "DRIVER",
	},
}

type customBroadcastTemplateRow struct {
	TemplateID  string
	WarehouseID string
	SupplierID  string
	Title       string
	Body        string
	DefaultRole string
	Category    string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HandleWarehouseBroadcastTemplates serves GET/POST /v1/warehouse/ops/broadcast/templates.
func (s *Service) HandleWarehouseBroadcastTemplates(w http.ResponseWriter, r *http.Request) {
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleListBroadcastTemplates(w, r, whID)
	case http.MethodPost:
		s.handleCreateBroadcastTemplate(w, r, whID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleWarehouseBroadcastTemplateDelete serves DELETE /v1/warehouse/ops/broadcast/templates/{id}.
func (s *Service) HandleWarehouseBroadcastTemplateDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}
	templateID := strings.TrimSpace(chi.URLParam(r, "id"))
	if templateID == "" || strings.HasPrefix(templateID, "wh_") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot_delete_builtin_template"})
		return
	}
	body, ok := readMutationBody(w, r, 4*1024)
	if !ok {
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	if err := s.deleteCustomBroadcastTemplate(r.Context(), whID, templateID); err != nil {
		if strings.Contains(err.Error(), "not_found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete_failed"})
		return
	}
	resp := map[string]string{"status": "deleted", "template_id": templateID}
	respBytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
}

// HandleWarehouseBroadcast serves POST /v1/warehouse/ops/broadcast (depot-scoped WS fan-out).
func (s *Service) HandleWarehouseBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if whID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "warehouse_scope_required"})
		return
	}

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title_and_body_required"})
		return
	}

	supplierID, err := s.resolveWarehouseSupplierID(r.Context(), whID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}

	targetRole := strings.ToUpper(strings.TrimSpace(req.Role))
	if targetRole == "" {
		targetRole = "DRIVER"
	}
	payload, _ := json.Marshal(map[string]any{
		"type":         "WAREHOUSE_BROADCAST",
		"title":        req.Title,
		"body":         req.Body,
		"target_role":  targetRole,
		"warehouse_id": whID,
		"supplier_id":  supplierID,
		"timestamp":    s.now().UTC().Format(time.RFC3339Nano),
	})
	s.broadcastDepotMessage(r.Context(), whID, supplierID, targetRole, payload)

	resp := map[string]any{
		"status":       "broadcast_sent",
		"warehouse_id": whID,
		"supplier_id":  supplierID,
	}
	respBytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	idemCommitted = true
}

func (s *Service) handleListBroadcastTemplates(w http.ResponseWriter, r *http.Request, warehouseID string) {
	custom, err := s.listCustomBroadcastTemplates(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_templates_failed"})
		return
	}
	out := make([]BroadcastTemplateWire, 0, len(builtinWarehouseBroadcastTemplates)+len(custom))
	out = append(out, builtinWarehouseBroadcastTemplates...)
	for _, row := range custom {
		out = append(out, row.toWire())
	}
	writeJSON(w, http.StatusOK, broadcastTemplatesResponse{Templates: out})
}

func (s *Service) handleCreateBroadcastTemplate(w http.ResponseWriter, r *http.Request, warehouseID string) {
	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), key)
		}
	}()

	var req struct {
		Title       string `json:"title"`
		Body        string `json:"body"`
		DefaultRole string `json:"default_role"`
		Category    string `json:"category"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title_and_body_required"})
		return
	}
	role := strings.ToUpper(strings.TrimSpace(req.DefaultRole))
	if role == "" {
		role = "DRIVER"
	}
	switch role {
	case "DRIVER", "RETAILER", "ALL":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_target_role"})
		return
	}

	supplierID, err := s.resolveWarehouseSupplierID(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "warehouse_not_found"})
		return
	}
	actorID := ""
	if claims, ok := auth.FromContext(r.Context()); ok {
		actorID = strings.TrimSpace(claims.Subject)
	}

	row, err := s.insertCustomBroadcastTemplate(r.Context(), customBroadcastTemplateRow{
		TemplateID:  uuid.NewString(),
		WarehouseID: warehouseID,
		SupplierID:  supplierID,
		Title:       req.Title,
		Body:        req.Body,
		DefaultRole: role,
		Category:    strings.TrimSpace(req.Category),
		CreatedBy:   actorID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_template_failed"})
		return
	}
	wire := row.toWire()
	respBytes, _ := json.Marshal(wire)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
	s.storeMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	idemCommitted = true
}

func (row customBroadcastTemplateRow) toWire() BroadcastTemplateWire {
	category := strings.TrimSpace(row.Category)
	if category == "" {
		category = "custom"
	}
	return BroadcastTemplateWire{
		ID:          row.TemplateID,
		Category:    category,
		Title:       row.Title,
		Body:        row.Body,
		DefaultRole: row.DefaultRole,
		Scope:       "warehouse",
		Source:      "custom",
		WarehouseID: row.WarehouseID,
	}
}

func (s *Service) resolveWarehouseSupplierID(ctx context.Context, warehouseID string) (string, error) {
	if s.spannerClient == nil {
		return strings.TrimSpace(s.seedSupplierID), nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
	if err != nil {
		return "", err
	}
	var supplierID string
	if err := row.Columns(&supplierID); err != nil {
		return "", err
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return "", errors.New("warehouse_not_found")
	}
	return supplierID, nil
}

func (s *Service) listCustomBroadcastTemplates(ctx context.Context, warehouseID string) ([]customBroadcastTemplateRow, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return append([]customBroadcastTemplateRow(nil), s.broadcastTemplatesMem[warehouseID]...), nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT TemplateId, WarehouseId, SupplierId, Title, Body, DefaultRole,
		             COALESCE(Category, ''), COALESCE(CreatedBy, ''), CreatedAt, UpdatedAt
		      FROM WarehouseBroadcastTemplates
		      WHERE WarehouseId = @wid
		      ORDER BY UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []customBroadcastTemplateRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var item customBroadcastTemplateRow
		if err := row.Columns(
			&item.TemplateID, &item.WarehouseID, &item.SupplierID, &item.Title, &item.Body,
			&item.DefaultRole, &item.Category, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			continue
		}
		out = append(out, item)
	}
}

func (s *Service) insertCustomBroadcastTemplate(ctx context.Context, row customBroadcastTemplateRow) (customBroadcastTemplateRow, error) {
	now := s.now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.broadcastTemplatesMem == nil {
			s.broadcastTemplatesMem = map[string][]customBroadcastTemplateRow{}
		}
		s.broadcastTemplatesMem[row.WarehouseID] = append(s.broadcastTemplatesMem[row.WarehouseID], row)
		return row, nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("WarehouseBroadcastTemplates", map[string]any{
				"WarehouseId": row.WarehouseID,
				"TemplateId":  row.TemplateID,
				"SupplierId":  row.SupplierID,
				"Title":       row.Title,
				"Body":        row.Body,
				"DefaultRole": row.DefaultRole,
				"Category":    nullString(row.Category),
				"CreatedBy":   nullString(row.CreatedBy),
				"CreatedAt":   spanner.CommitTimestamp,
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		})
	})
	return row, err
}

func (s *Service) deleteCustomBroadcastTemplate(ctx context.Context, warehouseID, templateID string) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		rows := s.broadcastTemplatesMem[warehouseID]
		next := rows[:0]
		found := false
		for _, row := range rows {
			if row.TemplateID == templateID {
				found = true
				continue
			}
			next = append(next, row)
		}
		if !found {
			return errTemplateNotFound
		}
		s.broadcastTemplatesMem[warehouseID] = next
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.ReadRow(ctx, "WarehouseBroadcastTemplates", spanner.Key{warehouseID, templateID}, []string{"TemplateId"})
		if err != nil {
			return errTemplateNotFound
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("WarehouseBroadcastTemplates", spanner.Key{warehouseID, templateID}),
		})
	})
	return err
}

func (s *Service) broadcastDepotMessage(ctx context.Context, warehouseID, supplierID, targetRole string, payload []byte) {
	if len(payload) == 0 {
		return
	}
	role := strings.ToUpper(strings.TrimSpace(targetRole))
	if s.warehouseHub != nil {
		s.warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, payload)
	}
	fanAll := role == "ALL"
	if fanAll || role == "DRIVER" {
		s.fanDepotDrivers(ctx, warehouseID, payload)
	}
	if fanAll || role == "RETAILER" {
		s.fanDepotRetailers(ctx, warehouseID, supplierID, payload)
	}
}

func (s *Service) fanDepotDrivers(ctx context.Context, warehouseID string, payload []byte) {
	if s.driverHub == nil {
		return
	}
	seen := map[string]struct{}{}
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT DriverId FROM Drivers@{FORCE_INDEX=Idx_Drivers_ByHomeNode}
			      WHERE HomeNodeType = 'WAREHOUSE' AND HomeNodeId = @wid`,
			Params: map[string]any{"wid": warehouseID},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var driverID string
			if err := row.Columns(&driverID); err != nil {
				continue
			}
			driverID = strings.TrimSpace(driverID)
			if driverID == "" {
				continue
			}
			seen[driverID] = struct{}{}
			s.driverHub.Broadcast(ctx, "driver:"+driverID, payload)
		}
	}
	if s.opsDrivers != nil {
		drivers, err := s.opsDrivers(ctx, warehouseID)
		if err == nil {
			for _, d := range drivers {
				driverID := strings.TrimSpace(d.DriverID)
				if driverID == "" {
					continue
				}
				if _, ok := seen[driverID]; ok {
					continue
				}
				seen[driverID] = struct{}{}
				s.driverHub.Broadcast(ctx, "driver:"+driverID, payload)
			}
		}
	}
}

func (s *Service) fanDepotRetailers(ctx context.Context, warehouseID, supplierID string, payload []byte) {
	if s.retailerHub == nil || s.spannerClient == nil {
		return
	}
	since := s.now().UTC().Add(-30 * 24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT RetailerId
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByWarehouseCreated}
		      WHERE WarehouseId = @wid AND UpdatedAt >= @since AND RetailerId IS NOT NULL
		      LIMIT 200`,
		Params: map[string]any{"wid": warehouseID, "since": since},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	seen := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		var retailerID string
		if err := row.Columns(&retailerID); err != nil {
			continue
		}
		retailerID = strings.TrimSpace(retailerID)
		if retailerID == "" {
			continue
		}
		if _, ok := seen[retailerID]; ok {
			continue
		}
		seen[retailerID] = struct{}{}
		s.retailerHub.Broadcast(ctx, "retailer:"+retailerID, payload)
	}
}

func nullString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

var errTemplateNotFound = errors.New("template_not_found")
