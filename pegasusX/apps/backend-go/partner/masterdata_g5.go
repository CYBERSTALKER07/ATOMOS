package partner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PartyUpsertItem is one trading-party master row (G5.C).
type PartyUpsertItem struct {
	ExternalID string `json:"external_id"`
	Role       string `json:"role"` // RETAILER|SUPPLIER|WAREHOUSE|CARRIER
	LegalName  string `json:"legal_name"`
	GLN        string `json:"gln"`
	Version    int64  `json:"version"`
}

// PartyUpsertResult is per-row outcome.
type PartyUpsertResult struct {
	ExternalID string `json:"external_id"`
	Action     string `json:"action"` // created|updated|conflict|error
	Error      string `json:"error,omitempty"`
}

// PlantUpsertItem maps external plant to warehouse.
type PlantUpsertItem struct {
	ExternalPlantID string `json:"external_plant_id"`
	WarehouseID     string `json:"warehouse_id"`
}

// PlantUpsertResult is per-row plant map outcome.
type PlantUpsertResult struct {
	ExternalPlantID string `json:"external_plant_id"`
	Action          string `json:"action"`
	Error           string `json:"error,omitempty"`
}

// MasterDataDLQRow is a failed sync row.
type MasterDataDLQRow struct {
	DlqID      string    `json:"dlq_id"`
	TenantType string    `json:"tenant_type"`
	TenantID   string    `json:"tenant_id"`
	EntityType string    `json:"entity_type"`
	ExternalID string    `json:"external_id"`
	Reason     string    `json:"reason"`
	PayloadHash string   `json:"payload_hash,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type partyKey struct {
	tt, tid, ext string
}

type plantKey struct {
	tt, tid, ext string
}

// In-memory master-data stores (Spanner optional later; G5 memory path for tests).
var (
	mdMu      sync.RWMutex
	mdParties = map[partyKey]PartyUpsertItem{}
	mdPlants  = map[plantKey]PlantUpsertItem{}
	mdDLQ     []MasterDataDLQRow
)

// UpsertParties applies party rows with version conflict detection.
func (s *Service) UpsertParties(ctx context.Context, p Principal, items []PartyUpsertItem) ([]PartyUpsertResult, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items_required")
	}
	if len(items) > maxMasterDataBatch {
		return nil, fmt.Errorf("batch_too_large")
	}
	out := make([]PartyUpsertResult, 0, len(items))
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalID)
		res := PartyUpsertResult{ExternalID: ext}
		if ext == "" {
			res.Action = "error"
			res.Error = "external_id_required"
			out = append(out, res)
			_ = s.pushMasterDLQ(p, "party", ext, res.Error, it)
			continue
		}
		role := strings.ToUpper(strings.TrimSpace(it.Role))
		if role == "" {
			role = "RETAILER"
		}
		it.Role = role
		it.GLN = strings.TrimSpace(it.GLN)
		k := partyKey{p.TenantType, p.TenantID, ext}
		mdMu.Lock()
		prev, exists := mdParties[k]
		if exists && it.Version > 0 && prev.Version > it.Version {
			mdMu.Unlock()
			res.Action = "conflict"
			res.Error = "stale_version"
			out = append(out, res)
			_ = s.pushMasterDLQ(p, "party", ext, res.Error, it)
			continue
		}
		if exists {
			res.Action = "updated"
		} else {
			res.Action = "created"
		}
		if it.Version == 0 {
			it.Version = prev.Version + 1
			if it.Version == 0 {
				it.Version = 1
			}
		}
		mdParties[k] = it
		mdMu.Unlock()
		out = append(out, res)
	}
	return out, nil
}

// UpsertPlants maps external plants to warehouse ids.
func (s *Service) UpsertPlants(ctx context.Context, p Principal, items []PlantUpsertItem) ([]PlantUpsertResult, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items_required")
	}
	out := make([]PlantUpsertResult, 0, len(items))
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalPlantID)
		wh := strings.TrimSpace(it.WarehouseID)
		res := PlantUpsertResult{ExternalPlantID: ext}
		if ext == "" || wh == "" {
			res.Action = "error"
			res.Error = "plant_and_warehouse_required"
			out = append(out, res)
			_ = s.pushMasterDLQ(p, "plant", ext, res.Error, it)
			continue
		}
		k := plantKey{p.TenantType, p.TenantID, ext}
		mdMu.Lock()
		_, exists := mdPlants[k]
		mdPlants[k] = PlantUpsertItem{ExternalPlantID: ext, WarehouseID: wh}
		mdMu.Unlock()
		if exists {
			res.Action = "updated"
		} else {
			res.Action = "created"
		}
		out = append(out, res)
	}
	return out, nil
}

// ResolvePlantWarehouse returns mapped warehouse for external plant id.
func ResolvePlantWarehouse(tenantType, tenantID, externalPlantID string) (string, bool) {
	mdMu.RLock()
	defer mdMu.RUnlock()
	it, ok := mdPlants[plantKey{tenantType, tenantID, strings.TrimSpace(externalPlantID)}]
	return it.WarehouseID, ok
}

func (s *Service) pushMasterDLQ(p Principal, entity, ext, reason string, payload any) error {
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	row := MasterDataDLQRow{
		DlqID:       uuid.NewString(),
		TenantType:  p.TenantType,
		TenantID:    p.TenantID,
		EntityType:  entity,
		ExternalID:  ext,
		Reason:      reason,
		PayloadHash: hex.EncodeToString(sum[:]),
		CreatedAt:   time.Now().UTC(),
	}
	mdMu.Lock()
	mdDLQ = append(mdDLQ, row)
	if len(mdDLQ) > 500 {
		mdDLQ = mdDLQ[len(mdDLQ)-500:]
	}
	mdMu.Unlock()
	return nil
}

// ListMasterDataDLQ returns recent failed sync rows for tenant.
func (s *Service) ListMasterDataDLQ(ctx context.Context, p Principal, limit int) ([]MasterDataDLQRow, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	mdMu.RLock()
	defer mdMu.RUnlock()
	out := make([]MasterDataDLQRow, 0, limit)
	for i := len(mdDLQ) - 1; i >= 0 && len(out) < limit; i-- {
		row := mdDLQ[i]
		if row.TenantType == p.TenantType && row.TenantID == p.TenantID {
			out = append(out, row)
		}
	}
	return out, nil
}

// HandleUpsertParties PUT /partner/v1/masterdata/parties
func (h *Handlers) HandleUpsertParties(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []PartyUpsertItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	results, err := h.Svc.UpsertParties(r.Context(), p, req.Items)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "count": len(results)})
}

// HandleUpsertPlants PUT /partner/v1/masterdata/plants
func (h *Handlers) HandleUpsertPlants(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Items []PlantUpsertItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	results, err := h.Svc.UpsertPlants(r.Context(), p, req.Items)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "count": len(results)})
}

// HandleListMasterDataDLQ GET /partner/v1/masterdata/dlq
func (h *Handlers) HandleListMasterDataDLQ(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.Svc.ListMasterDataDLQ(r.Context(), p, 50)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": rows, "count": len(rows)})
}

// ValidGTIN reports whether s looks like EAN-8/13/14 (digits only, length).
func ValidGTIN(s string) bool {
	s = strings.TrimSpace(s)
	if n := len(s); n != 8 && n != 12 && n != 13 && n != 14 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
