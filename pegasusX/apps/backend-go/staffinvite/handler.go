package staffinvite

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handler mints staff invites for supplier ADMIN.
type Handler struct {
	Secret         string
	SeedSupplierID string
	Now            func() time.Time
	NodeOwned      NodeOwnedFunc
}

// HandleCreate is POST /v1/supplier/staff-invites (ADMIN cookie).
func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrInviteSecretMissing.Error()})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrInviteRequired.Error()})
		return
	}
	if err := GuardSeed(h.SeedSupplierID, sid); err != nil {
		WriteError(w, err)
		return
	}

	var req struct {
		Role   string `json:"role"`
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	role := NormalizeRole(req.Role)
	nid := strings.TrimSpace(req.NodeID)
	if role != RoleFactory && role != RoleWarehouse {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrInviteRoleMismatch.Error()})
		return
	}
	if nid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrNodeRequired.Error()})
		return
	}
	if h.NodeOwned != nil {
		ok, err := h.NodeOwned(r.Context(), sid, role, nid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "node_lookup_failed"})
			return
		}
		if !ok {
			WriteError(w, ErrNodeNotOwned)
			return
		}
	}
	now := time.Now().UTC()
	if h.Now != nil {
		now = h.Now()
	}
	token, exp, err := Mint(h.Secret, role, sid, nid, 7*24*time.Hour, now)
	if err != nil {
		WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"role":        role,
		"supplier_id": sid,
		"node_id":     nid,
		"token":       token,
		"expires_at":  exp.UTC().Format(time.RFC3339),
	})
}

// WriteError maps T5 sentinels to HTTP status.
func WriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInviteRequired), errors.Is(err, ErrInviteInvalid),
		errors.Is(err, ErrInviteExpired), errors.Is(err, ErrInviteRoleMismatch),
		errors.Is(err, ErrNodeRequired), errors.Is(err, ErrPasswordRequired):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrSeedStaffForbidden), errors.Is(err, ErrNodeNotOwned):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInviteSecretMissing):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
