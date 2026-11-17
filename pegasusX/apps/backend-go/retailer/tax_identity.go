package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type taxIdentityRequest struct {
	TaxID             string `json:"tax_id,omitempty"`
	INN               string `json:"inn,omitempty"`
	InformalShop      bool   `json:"informal_shop,omitempty"`
	LegalEntityStatus string `json:"legal_entity_status,omitempty"`
	HomeCell          string `json:"home_cell,omitempty"`
	MarketCode        string `json:"market_code,omitempty"`
	RoleSought        string `json:"role_sought,omitempty"`
	LegalName         string `json:"legal_name,omitempty"`
}

type joinRequestConflictPayload struct {
	Error          string `json:"error"`
	RequestID      string `json:"request_id"`
	TargetTenantID string `json:"target_tenant_id"`
	LegalName      string `json:"legal_name"`
	NormalizedINN  string `json:"normalized_inn"`
}

func (s *Service) resolveAndApplyTaxIdentity(ctx context.Context, ret *Retailer, req taxIdentityRequest, requesterName string) (status int, conflict *joinRequestConflictPayload, errCode string) {
	rawTaxID := strings.TrimSpace(req.TaxID)
	if rawTaxID == "" {
		rawTaxID = strings.TrimSpace(req.INN)
	}
	informal := req.InformalShop || strings.EqualFold(strings.TrimSpace(req.LegalEntityStatus), LegalEntityInformalPhone)
	if informal && rawTaxID == "" {
		ret.TaxID = ""
		ret.NormalizedTaxID = ""
		ret.InnVerifiedAt = nil
		ret.LegalEntityStatus = LegalEntityInformalPhone
		if strings.TrimSpace(req.HomeCell) != "" {
			ret.HomeCell = strings.TrimSpace(req.HomeCell)
		}
		return 0, nil, ""
	}
	if rawTaxID == "" {
		if strings.TrimSpace(ret.LegalEntityStatus) == "" {
			ret.LegalEntityStatus = LegalEntityInformalPhone
		}
		if strings.TrimSpace(req.HomeCell) != "" && strings.TrimSpace(ret.HomeCell) == "" {
			ret.HomeCell = strings.TrimSpace(req.HomeCell)
		}
		return 0, nil, ""
	}

	norm, _, err := innlookup.NormalizeTaxID(rawTaxID)
	if err != nil {
		return http.StatusBadRequest, nil, err.Error()
	}
	roleSought, err := innlookup.NormalizeRoleSought(req.RoleSought)
	if err != nil {
		return http.StatusBadRequest, nil, err.Error()
	}
	cell, err := innlookup.ResolveHomeCell(firstNonEmpty(req.HomeCell, ret.HomeCell), firstNonEmpty(req.MarketCode, ret.CountryCode))
	if err != nil {
		if errors.Is(err, innlookup.ErrCrossCellDenied) {
			return http.StatusForbidden, nil, innlookup.ErrCrossCellDenied.Error()
		}
		return http.StatusBadRequest, nil, err.Error()
	}
	innSvc := s.INNLookup()
	if innSvc == nil {
		return http.StatusServiceUnavailable, nil, innlookup.ErrRegistryUnavailable.Error()
	}

	res, err := innSvc.Lookup(ctx, innlookup.LookupOptions{
		TaxID:      norm,
		HomeCell:   cell,
		MarketCode: firstNonEmpty(req.MarketCode, ret.CountryCode),
		TenantType: innlookup.TenantTypeRetailer,
	})
	if err != nil {
		switch {
		case errors.Is(err, innlookup.ErrInvalidTaxID):
			return http.StatusBadRequest, nil, err.Error()
		case errors.Is(err, innlookup.ErrCrossCellDenied):
			return http.StatusForbidden, nil, err.Error()
		case errors.Is(err, innlookup.ErrRateLimited):
			return http.StatusTooManyRequests, nil, err.Error()
		default:
			return http.StatusServiceUnavailable, nil, innlookup.ErrRegistryUnavailable.Error()
		}
	}
	if res.ExistingTenantID != "" && res.ExistingTenantID != ret.RetailerID {
		now := s.now()
		reqName := firstNonEmpty(requesterName, ret.Name, req.LegalName, ret.Phone)
		jr, jerr := innSvc.CreateJoinRequest(ctx, innlookup.JoinRequest{
			HomeCell:         cell,
			TargetTenantType: innlookup.TenantTypeRetailer,
			TargetTenantID:   res.ExistingTenantID,
			NormalizedINN:    res.NormalizedINN,
			RequesterPhone:   ret.Phone,
			RequesterName:    reqName,
			RoleSought:       roleSought,
			CreatedAt:        now,
		}, func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(ctx, txn, events.AggregateRetailer, res.ExistingTenantID, events.TopicMain, map[string]any{
				"type":               "JOIN_REQUEST_CREATED",
				"timestamp":          now.Format(time.RFC3339Nano),
				"retailer_id":        res.ExistingTenantID,
				"target_tenant_type": innlookup.TenantTypeRetailer,
				"target_tenant_id":   res.ExistingTenantID,
				"normalized_inn":     res.NormalizedINN,
				"requester_phone":    ret.Phone,
				"requester_name":     reqName,
				"role_sought":        roleSought,
			})
		})
		if jerr != nil {
			return http.StatusInternalServerError, nil, "create_join_request_failed"
		}
		if s.migrationNotifier != nil {
			_ = s.migrationNotifier(
				ctx,
				res.ExistingTenantID,
				string(auth.RoleRetailer),
				"JOIN_REQUEST_CREATED",
				"New join request for matching INN",
				reqName+" ("+ret.Phone+") requested access to organization INN "+res.NormalizedINN,
				"/setup/tax?requestId="+jr.RequestID,
			)
		}
		return http.StatusConflict, &joinRequestConflictPayload{
			Error:          innlookup.ErrDuplicateTenant.Error(),
			RequestID:      jr.RequestID,
			TargetTenantID: res.ExistingTenantID,
			LegalName:      firstNonEmpty(res.ExistingLegalName, res.LegalName),
			NormalizedINN:  res.NormalizedINN,
		}, ""
	}
	if res.Status == innlookup.StatusNotFound {
		return http.StatusUnprocessableEntity, nil, innlookup.ErrTaxIDNotFound.Error()
	}
	if res.Status != innlookup.StatusFound {
		return http.StatusServiceUnavailable, nil, innlookup.ErrRegistryUnavailable.Error()
	}

	now := s.now()
	ret.TaxID = rawTaxID
	ret.NormalizedTaxID = res.NormalizedINN
	ret.HomeCell = cell
	ret.LegalEntityStatus = LegalEntityVerified
	ret.InnVerifiedAt = &now
	if strings.TrimSpace(ret.Name) == "" && strings.TrimSpace(res.LegalName) != "" {
		ret.Name = strings.TrimSpace(res.LegalName)
	}
	return 0, nil, ""
}

// HandleTaxIdentity serves PATCH /v1/retailer/tax-identity for dual-mode legal/informal store setup.
func (s *Service) HandleTaxIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 32*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req taxIdentityRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if !req.InformalShop &&
		!strings.EqualFold(strings.TrimSpace(req.LegalEntityStatus), LegalEntityInformalPhone) &&
		strings.TrimSpace(req.TaxID) == "" &&
		strings.TrimSpace(req.INN) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tax_id_or_informal_mode_required"})
		return
	}

	ret, found, err := s.repo.GetRetailer(r.Context(), rid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_retailer_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "retailer_not_found"})
		return
	}
	if strings.TrimSpace(ret.SupplierID) == "" {
		ret.SupplierID = s.resolveSupplierScope(r.Context())
	}

	status, conflict, errCode := s.resolveAndApplyTaxIdentity(r.Context(), &ret, req, req.LegalName)
	if conflict != nil {
		writeJSON(w, status, conflict)
		return
	}
	if errCode != "" {
		writeJSON(w, status, map[string]string{"error": errCode})
		return
	}
	ret.UpdatedAt = s.now()
	if err := s.repo.UpdateRetailer(r.Context(), ret, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateRetailer, ret.RetailerID, events.TopicMain, events.RetailerEvent{
			BaseEvent:   events.BaseEvent{Type: events.EventRetailerRegistered, Timestamp: ret.UpdatedAt.Format(time.RFC3339Nano)},
			RetailerID:  ret.RetailerID,
			SupplierID:  ret.SupplierID,
			Phone:       ret.Phone,
			Name:        ret.Name,
			CountryCode: ret.CountryCode,
			H3Cell:      ret.H3Cell,
		})
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_retailer_failed"})
		return
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), retailerByPhoneKey(ret.Phone), "retailer:id:"+ret.RetailerID)
	}

	profile := retailerProfileDTO(ret, s.resolveSupplierScope(r.Context()))
	respBytes, _ := json.Marshal(profile)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleListJoinRequests serves GET /v1/retailer/join-requests.
func (s *Service) HandleListJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	innSvc := s.INNLookup()
	if innSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "join_requests_unavailable"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := innSvc.ListJoinRequests(r.Context(), innlookup.TenantTypeRetailer, rid, status, 100)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	if items == nil {
		items = []innlookup.JoinRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": rid,
		"requests":    items,
	})
}

// HandleApproveJoinRequest serves POST /v1/retailer/join-requests/{requestId}/approve.
func (s *Service) HandleApproveJoinRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDecideRetailerJoinRequest(w, r, innlookup.JoinStatusApproved)
}

// HandleRejectJoinRequest serves POST /v1/retailer/join-requests/{requestId}/reject.
func (s *Service) HandleRejectJoinRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDecideRetailerJoinRequest(w, r, innlookup.JoinStatusRejected)
}

func (s *Service) handleDecideRetailerJoinRequest(w http.ResponseWriter, r *http.Request, decision string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	rid, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	innSvc := s.INNLookup()
	if innSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "join_requests_unavailable"})
		return
	}
	requestID := strings.TrimSpace(chi.URLParam(r, "requestId"))
	if requestID == "" {
		requestID = strings.TrimSpace(chi.URLParam(r, "requestID"))
	}
	if requestID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request_id_required"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_failed"})
		return
	}
	defer r.Body.Close()
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}
	actor := rid
	if claims, ok := auth.FromContext(r.Context()); ok && strings.TrimSpace(claims.Subject) != "" {
		actor = claims.Subject
	}
	now := s.now()
	decided, err := innSvc.DecideJoinRequest(
		r.Context(),
		innlookup.TenantTypeRetailer,
		rid,
		requestID,
		decision,
		actor,
		strings.TrimSpace(body.Reason),
		func(txn outbox.TxnBuffer) error {
			return outbox.EmitJSON(r.Context(), txn, events.AggregateRetailer, rid, events.TopicMain, map[string]any{
				"type":               "JOIN_REQUEST_RESOLVED",
				"timestamp":          now.Format(time.RFC3339Nano),
				"retailer_id":        rid,
				"target_tenant_type": innlookup.TenantTypeRetailer,
				"target_tenant_id":   rid,
				"request_id":         requestID,
				"status":             decision,
				"decided_by":         actor,
				"decision_reason":    strings.TrimSpace(body.Reason),
			})
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, innlookup.ErrJoinRequestNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, innlookup.ErrJoinRequestDecided):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": rid,
		"request":     decided,
	})
}
