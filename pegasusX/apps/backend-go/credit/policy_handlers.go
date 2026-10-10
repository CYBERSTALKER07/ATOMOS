package credit

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func policyErrStatus(err error) (int, string) {
	switch {
	case errors.Is(err, ErrWarningAckRequired):
		return http.StatusBadRequest, "warning_ack_required"
	case errors.Is(err, ErrDisableRequiresSupport):
		return http.StatusForbidden, "credit_disable_requires_support"
	case errors.Is(err, ErrProgramDisabled):
		return http.StatusConflict, "credit_program_not_enabled"
	case errors.Is(err, ErrCreditNotEnabled):
		return http.StatusConflict, "credit_relationship_not_enabled"
	case errors.Is(err, ErrProfileNotFound):
		return http.StatusNotFound, "credit_profile_not_found"
	default:
		if err != nil && strings.Contains(err.Error(), "ticket_id_and_reason_required") {
			return http.StatusBadRequest, "ticket_id_and_reason_required"
		}
		return http.StatusInternalServerError, "internal"
	}
}

func supplierScope(w http.ResponseWriter, r *http.Request) (claims auth.Claims, supplierID string, ok bool) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return auth.Claims{}, "", false
	}
	// Supplier portal (ADMIN), warehouse finance, or warehouse admin — supplier-scoped.
	switch c.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		// ok
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return auth.Claims{}, "", false
	}
	sid := strings.TrimSpace(c.SupplierID)
	if sid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return auth.Claims{}, "", false
	}
	return c, sid, true
}

// HandleGetCreditProgram GET /v1/supplier/credit-program
func (s *PolicyService) HandleGetCreditProgram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	p, found, err := s.GetProgram(r.Context(), sid)
	if err != nil {
		slog.ErrorContext(r.Context(), "get credit program", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, SupplierCreditProgram{SupplierID: sid, ProgramEnabled: false, GlobalTermsDays: 30, Timezone: packCreditTimezone(r.Context(), sid)})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// HandleEnableCreditProgram POST /v1/supplier/credit-program
func (s *PolicyService) HandleEnableCreditProgram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	var body struct {
		WarningAck              bool   `json:"warning_ack"`
		WarningAckAt            string `json:"warning_ack_at"`
		GlobalTermsDays         int64  `json:"global_terms_days"`
		GlobalGraceDays         int64  `json:"global_grace_days"`
		GlobalDefaultLimitMinor int64  `json:"global_default_limit_minor"`
		Timezone                string `json:"timezone"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ackAt := s.now()
	if body.WarningAckAt != "" {
		if t, err := time.Parse(time.RFC3339, body.WarningAckAt); err == nil {
			ackAt = t
		}
	}
	defaults := &SupplierCreditProgram{
		GlobalTermsDays:         body.GlobalTermsDays,
		GlobalGraceDays:         body.GlobalGraceDays,
		GlobalDefaultLimitMinor: body.GlobalDefaultLimitMinor,
		Timezone:                body.Timezone,
	}
	p, err := s.EnableProgram(r.Context(), sid, claims.Subject, string(claims.Role), body.WarningAck, ackAt, defaults)
	if err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// HandlePatchCreditProgramDefaults GET|PATCH /v1/supplier/credit-program/defaults
func (s *PolicyService) HandleCreditProgramDefaults(w http.ResponseWriter, r *http.Request) {
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		p, found, err := s.GetProgram(r.Context(), sid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "credit_program_not_enabled"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"global_terms_days":          p.GlobalTermsDays,
			"global_grace_days":          p.GlobalGraceDays,
			"global_default_limit_minor": p.GlobalDefaultLimitMinor,
			"timezone":                   p.Timezone,
		})
		return
	}
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	var body struct {
		GlobalTermsDays         *int64  `json:"global_terms_days"`
		GlobalGraceDays         *int64  `json:"global_grace_days"`
		GlobalDefaultLimitMinor *int64  `json:"global_default_limit_minor"`
		Timezone                *string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	p, err := s.PatchProgramDefaults(r.Context(), sid, claims.Subject, string(claims.Role),
		body.GlobalTermsDays, body.GlobalGraceDays, body.GlobalDefaultLimitMinor, body.Timezone)
	if err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// HandleListCreditRelationships GET /v1/supplier/credit-relationships
func (s *PolicyService) HandleListCreditRelationships(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	list, err := s.ListRelationships(r.Context(), sid, 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	type row struct {
		RetailerPaymentTerms
		ProfileStatus        string `json:"profile_status,omitempty"`
		CurrentBalanceMinor  int64  `json:"current_balance_minor,omitempty"`
		AvailableCreditMinor int64  `json:"available_credit_minor,omitempty"`
		ReservedMinor        int64  `json:"reserved_minor,omitempty"`
	}
	out := make([]row, 0, len(list))
	for _, t := range list {
		rr := row{RetailerPaymentTerms: t}
		if s.credit != nil {
			if p, found, _ := s.credit.GetProfile(r.Context(), t.RetailerID, sid); found {
				rr.ProfileStatus = string(p.Status)
				rr.CurrentBalanceMinor = p.CurrentBalanceMinor
				rr.AvailableCreditMinor = p.Available()
				rr.ReservedMinor = p.ReservedMinor
			}
		}
		out = append(out, rr)
	}
	writeJSON(w, http.StatusOK, map[string]any{"relationships": out})
}

// HandleEnableCreditRelationship POST /v1/supplier/credit-relationships/{retailerId}/enable
func (s *PolicyService) HandleEnableCreditRelationship(w http.ResponseWriter, r *http.Request) {
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	rid := strings.TrimSpace(chi.URLParam(r, "retailerId"))
	if rid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	var body struct {
		WarningAck        bool   `json:"warning_ack"`
		WarningAckAt      string `json:"warning_ack_at"`
		TermsDays         int64  `json:"terms_days"`
		GracePeriodDays   int64  `json:"grace_period_days"`
		CreditLimitMinor  int64  `json:"credit_limit_minor"`
		UseGlobalDefaults bool   `json:"use_global_defaults"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ackAt := s.now()
	if body.WarningAckAt != "" {
		if t, err := time.Parse(time.RFC3339, body.WarningAckAt); err == nil {
			ackAt = t
		}
	}
	t, err := s.EnableRelationship(r.Context(), sid, rid, claims.Subject, string(claims.Role),
		body.WarningAck, ackAt, body.TermsDays, body.GracePeriodDays, body.CreditLimitMinor, body.UseGlobalDefaults)
	if err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// HandlePatchCreditRelationshipTerms PATCH /v1/supplier/credit-relationships/{retailerId}/terms
func (s *PolicyService) HandlePatchCreditRelationshipTerms(w http.ResponseWriter, r *http.Request) {
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	rid := strings.TrimSpace(chi.URLParam(r, "retailerId"))
	var body struct {
		TermsDays         *int64 `json:"terms_days"`
		GracePeriodDays   *int64 `json:"grace_period_days"`
		CreditLimitMinor  *int64 `json:"credit_limit_minor"`
		UseGlobalDefaults *bool  `json:"use_global_defaults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	t, err := s.PatchRelationshipTerms(r.Context(), sid, rid, claims.Subject, string(claims.Role),
		body.TermsDays, body.GracePeriodDays, body.CreditLimitMinor, body.UseGlobalDefaults)
	if err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// HandleHoldCreditRelationship POST .../hold
func (s *PolicyService) HandleHoldCreditRelationship(w http.ResponseWriter, r *http.Request) {
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	rid := chi.URLParam(r, "retailerId")
	if err := s.HoldRelationship(r.Context(), sid, rid, claims.Subject, string(claims.Role)); err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "FROZEN"})
}

// HandleUnholdCreditRelationship POST .../unhold
func (s *PolicyService) HandleUnholdCreditRelationship(w http.ResponseWriter, r *http.Request) {
	claims, sid, ok := supplierScope(w, r)
	if !ok {
		return
	}
	rid := chi.URLParam(r, "retailerId")
	if err := s.UnholdRelationship(r.Context(), sid, rid, claims.Subject, string(claims.Role)); err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ACTIVE"})
}

// HandleSelfServeDisableCreditRelationship POST .../disable → always 403
func (s *PolicyService) HandleSelfServeDisableCreditRelationship(w http.ResponseWriter, r *http.Request) {
	_, _, ok := supplierScope(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]string{
		"error":   "credit_disable_requires_support",
		"message": "Contact Pegaus support to disable credit. Temporary holds remain available.",
	})
}

// HandleListRetailerCreditRelationships GET /v1/retailer/credit-relationships
func (s *PolicyService) HandleListRetailerCreditRelationships(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var rid string
	if claims.Role == auth.RoleAdmin {
		rid = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
		if rid == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
			return
		}
	} else {
		rid = auth.ResolveRetailerOrgID(claims)
	}
	list, err := s.ListRetailerRelationships(r.Context(), rid, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	type row struct {
		RetailerPaymentTerms
		Resolved             ResolvedTerms `json:"resolved_terms"`
		ProfileStatus        string        `json:"profile_status,omitempty"`
		AvailableCreditMinor int64         `json:"available_credit_minor,omitempty"`
		CurrentBalanceMinor  int64         `json:"current_balance_minor,omitempty"`
		OnHold               bool          `json:"on_hold"`
	}
	out := make([]row, 0, len(list))
	for _, t := range list {
		if !t.CreditEnabled {
			continue
		}
		resolved, _ := s.ResolveTermsFor(r.Context(), t.RetailerID, t.SupplierID)
		rr := row{RetailerPaymentTerms: t, Resolved: resolved}
		if s.credit != nil {
			if p, found, _ := s.credit.GetProfile(r.Context(), t.RetailerID, t.SupplierID); found {
				rr.ProfileStatus = string(p.Status)
				rr.AvailableCreditMinor = p.Available()
				rr.CurrentBalanceMinor = p.CurrentBalanceMinor
				rr.OnHold = p.Status == StatusFrozen
			}
		}
		out = append(out, rr)
	}
	writeJSON(w, http.StatusOK, map[string]any{"relationships": out})
}

// HandleAdminDisableRelationship POST /v1/admin/credit-relationships/{supplierId}/{retailerId}/disable
func (s *PolicyService) HandleAdminDisableRelationship(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	sid := chi.URLParam(r, "supplierId")
	rid := chi.URLParam(r, "retailerId")
	var body struct {
		TicketID string `json:"ticket_id"`
		Reason   string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.AdminDisableRelationship(r.Context(), sid, rid, claims.Subject, string(claims.Role), body.TicketID, body.Reason); err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// HandleAdminDisableProgram POST /v1/admin/credit-program/{supplierId}/disable
func (s *PolicyService) HandleAdminDisableProgram(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	sid := chi.URLParam(r, "supplierId")
	var body struct {
		TicketID string `json:"ticket_id"`
		Reason   string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.AdminDisableProgram(r.Context(), sid, claims.Subject, string(claims.Role), body.TicketID, body.Reason); err != nil {
		st, code := policyErrStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}
