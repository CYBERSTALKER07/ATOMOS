package partner

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleGetEdiProfile GET /partner/v1/edi/profile and JWT supplier alias.
func (h *Handlers) HandleGetEdiProfile(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.resolvePartnerTenant(r)
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	prof := ResolveEdiProfile(r.Context(), h.ediProfiles(), tt, tid)
	writeJSON(w, http.StatusOK, prof)
}

// HandlePutEdiProfile PUT /partner/v1/edi/profile
func (h *Handlers) HandlePutEdiProfile(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.resolvePartnerTenant(r)
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body EdiProfile
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	body.TenantType = tt
	body.TenantID = tid
	if body.PackName == "" {
		body.PackName = EdiPackEdifactLiteV1
	}
	out := make([]string, 0, len(body.EnabledDocTypes))
	for _, d := range body.EnabledDocTypes {
		if t := strings.ToUpper(strings.TrimSpace(d)); t != "" {
			out = append(out, t)
		}
	}
	body.EnabledDocTypes = out
	repo := h.ediProfiles()
	if repo == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "profiles_unavailable")
		return
	}
	if err := repo.Upsert(r.Context(), body); err != nil {
		writePartnerError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handlers) ediProfiles() EdiProfileRepository {
	if h == nil || h.Svc == nil {
		return nil
	}
	return h.Svc.ediProfiles
}

func (h *Handlers) resolvePartnerTenant(r *http.Request) (tenantType, tenantID string, ok bool) {
	if p, pok := PrincipalFromContext(r.Context()); pok && p.TenantID != "" {
		return p.TenantType, p.TenantID, true
	}
	claims, cok := auth.FromContext(r.Context())
	if !cok {
		return "", "", false
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		sid = auth.PreferTenantSupplierID(r.Context(), "")
	}
	if sid == "" {
		return "", "", false
	}
	return TenantSupplier, sid, true
}
