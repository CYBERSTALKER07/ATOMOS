package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
)

const sellerMandatesPackID = "SELLER_MANDATES"

// SellerMandate mirrors order.SellerMandate so retailer and order share one contract.
type SellerMandate = order.SellerMandate

// SellerDailyUsageSnapshot captures a staff member's spend for a UTC usage date.
type SellerDailyUsageSnapshot struct {
	RetailerID       string `json:"retailer_id"`
	UserID           string `json:"user_id"`
	UsageDate        string `json:"usage_date"`
	SpentCreditMinor int64  `json:"spent_credit_minor"`
	SpentCashMinor   int64  `json:"spent_cash_minor"`
	OrderCount       int64  `json:"order_count"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// OrgMemberMandateResponse is returned by GET and PATCH /v1/retailer/org/members/{userID}/mandate.
type OrgMemberMandateResponse struct {
	UserID       string                   `json:"user_id"`
	RetailerID   string                   `json:"retailer_id"`
	Role         string                   `json:"role"`
	RetailerRole string                   `json:"retailer_role"`
	IsOwner      bool                     `json:"is_owner"`
	Configured   bool                     `json:"configured"`
	Mandate      SellerMandate            `json:"mandate"`
	DailyUsage   SellerDailyUsageSnapshot `json:"daily_usage"`
}

// MandatePatchRequest updates one or more SellerMandate fields on a staff member.
type MandatePatchRequest struct {
	CanReceiveGoods     *bool                `json:"can_receive_goods"`
	CanPayCash          *bool                `json:"can_pay_cash"`
	CanUseCredit        *bool                `json:"can_use_credit"`
	DailyCreditCapMinor *int64               `json:"daily_credit_cap_minor"`
	DailyCashCapMinor   *int64               `json:"daily_cash_cap_minor"`
	MaxSingleOrderMinor *int64               `json:"max_single_order_minor"`
	Mandate             *MandatePatchRequest `json:"mandate,omitempty"`
}

type mandatePackDocument struct {
	Mandates map[string]SellerMandate `json:"mandates"`
}

var (
	errMandateMemberNotFound = errors.New("mandate_member_not_found")
	errMandateOwnerLocked    = errors.New("retailer_org_member_owner_locked")
	errInvalidMandateCap     = errors.New("invalid_mandate_cap")
)

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func normalizeRetailerRoleString(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}

// ResolveSellerMandate implements order.MandateReader for cross-package checkout gating.
func (s *Service) ResolveSellerMandate(ctx context.Context, retailerID, userID, fallbackRole string) (order.MandateLookupResult, error) {
	if s == nil {
		role := normalizeRetailerRoleString(fallbackRole)
		isOwner := role == "OWNER"
		return order.MandateLookupResult{
			Mandate: order.DefaultSellerMandateForRole(role, isOwner),
			Role:    role,
			IsOwner: isOwner,
			Found:   false,
		}, nil
	}
	retailerID = strings.TrimSpace(retailerID)
	userID = strings.TrimSpace(userID)
	role := normalizeRetailerRoleString(fallbackRole)
	isOwner := role == "OWNER" || (userID != "" && userID == retailerID)

	if userID != "" && userID != retailerID {
		u, found, err := s.findRetailerUserByID(ctx, userID)
		if err != nil {
			return order.MandateLookupResult{}, fmt.Errorf("resolve retailer user for mandate: %w", err)
		}
		if found {
			if u.RetailerID != retailerID {
				return order.MandateLookupResult{}, errMandateMemberNotFound
			}
			if r := normalizeRetailerRoleString(u.RetailerRole); r != "" {
				role = r
			}
			isOwner = u.IsOwner || role == "OWNER"
		}
	}

	if isOwner {
		return order.MandateLookupResult{
			Mandate: order.DefaultSellerMandateForRole("OWNER", true),
			Role:    "OWNER",
			IsOwner: true,
			Found:   true,
		}, nil
	}

	explicit, configured, err := s.readExplicitMandate(ctx, retailerID, userID)
	if err != nil {
		return order.MandateLookupResult{}, err
	}
	if configured {
		return order.MandateLookupResult{
			Mandate: explicit,
			Role:    role,
			IsOwner: false,
			Found:   true,
		}, nil
	}

	return order.MandateLookupResult{
		Mandate: order.DefaultSellerMandateForRole(role, false),
		Role:    role,
		IsOwner: false,
		Found:   false,
	}, nil
}

func (s *Service) readExplicitMandate(ctx context.Context, retailerID, userID string) (SellerMandate, bool, error) {
	if s == nil || retailerID == "" || userID == "" {
		return SellerMandate{}, false, nil
	}
	if s.spannerClient != nil {
		row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerCapabilityPacks",
			spanner.Key{retailerID, sellerMandatesPackID}, []string{"ConfigJson"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return SellerMandate{}, false, nil
			}
			return SellerMandate{}, false, fmt.Errorf("read seller mandate pack: %w", err)
		}
		var raw spanner.NullString
		if err := row.Columns(&raw); err != nil {
			return SellerMandate{}, false, fmt.Errorf("scan seller mandate pack: %w", err)
		}
		if !raw.Valid || strings.TrimSpace(raw.StringVal) == "" {
			return SellerMandate{}, false, nil
		}
		var doc mandatePackDocument
		if err := json.Unmarshal([]byte(raw.StringVal), &doc); err != nil {
			return SellerMandate{}, false, fmt.Errorf("decode seller mandate pack: %w", err)
		}
		if doc.Mandates == nil {
			return SellerMandate{}, false, nil
		}
		m, ok := doc.Mandates[userID]
		return m, ok, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.mandatesByRetailer == nil {
		return SellerMandate{}, false, nil
	}
	byUser := s.mandatesByRetailer[retailerID]
	if byUser == nil {
		return SellerMandate{}, false, nil
	}
	m, ok := byUser[userID]
	return m, ok, nil
}

// GetSellerDailyUsage returns today's UTC usage totals for (retailerID, userID).
func (s *Service) GetSellerDailyUsage(ctx context.Context, retailerID, userID string) (SellerDailyUsageSnapshot, error) {
	today := civil.DateOf(s.currentTime())
	out := SellerDailyUsageSnapshot{
		RetailerID: retailerID,
		UserID:     userID,
		UsageDate:  today.String(),
	}
	if s == nil || s.spannerClient == nil || retailerID == "" || userID == "" {
		return out, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerSellerDailyUsage",
		spanner.Key{retailerID, userID, today},
		[]string{"SpentCreditMinor", "SpentCashMinor", "OrderCount", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return out, nil
		}
		return out, fmt.Errorf("read seller daily usage: %w", err)
	}
	var creditMinor, cashMinor, orderCount int64
	var updatedAt time.Time
	if err := row.Columns(&creditMinor, &cashMinor, &orderCount, &updatedAt); err != nil {
		return out, fmt.Errorf("scan seller daily usage: %w", err)
	}
	out.SpentCreditMinor = creditMinor
	out.SpentCashMinor = cashMinor
	out.OrderCount = orderCount
	if !updatedAt.IsZero() {
		out.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	}
	return out, nil
}

// GetSellerMandate returns the effective SellerMandate and today's daily usage for a staff member.
func (s *Service) GetSellerMandate(ctx context.Context, retailerID, userID string) (OrgMemberMandateResponse, error) {
	if s == nil {
		return OrgMemberMandateResponse{}, fmt.Errorf("nil service")
	}
	retailerID = strings.TrimSpace(retailerID)
	userID = strings.TrimSpace(userID)
	target, found, err := s.findRetailerUserByID(ctx, userID)
	if err != nil {
		return OrgMemberMandateResponse{}, fmt.Errorf("load org member: %w", err)
	}
	if !found || target.RetailerID != retailerID {
		return OrgMemberMandateResponse{}, errMandateMemberNotFound
	}
	resolved, err := s.ResolveSellerMandate(ctx, retailerID, userID, target.RetailerRole)
	if err != nil {
		return OrgMemberMandateResponse{}, err
	}
	usage, err := s.GetSellerDailyUsage(ctx, retailerID, userID)
	if err != nil {
		return OrgMemberMandateResponse{}, err
	}
	role := normalizeRetailerRoleString(target.RetailerRole)
	if role == "" {
		role = resolved.Role
	}
	isOwner := target.IsOwner || role == "OWNER"
	return OrgMemberMandateResponse{
		UserID:       target.UserID,
		RetailerID:   target.RetailerID,
		Role:         role,
		RetailerRole: role,
		IsOwner:      isOwner,
		Configured:   resolved.Found && !isOwner,
		Mandate:      resolved.Mandate,
		DailyUsage:   usage,
	}, nil
}

// UpdateSellerMandate updates and persists the SellerMandate for a non-owner org member.
func (s *Service) UpdateSellerMandate(ctx context.Context, retailerID, userID, actorUserID string, req MandatePatchRequest) (OrgMemberMandateResponse, error) {
	if s == nil {
		return OrgMemberMandateResponse{}, fmt.Errorf("nil service")
	}
	retailerID = strings.TrimSpace(retailerID)
	userID = strings.TrimSpace(userID)
	actorUserID = strings.TrimSpace(actorUserID)

	if (req.DailyCreditCapMinor != nil && *req.DailyCreditCapMinor < 0) ||
		(req.DailyCashCapMinor != nil && *req.DailyCashCapMinor < 0) ||
		(req.MaxSingleOrderMinor != nil && *req.MaxSingleOrderMinor < 0) {
		return OrgMemberMandateResponse{}, errInvalidMandateCap
	}

	target, found, err := s.findRetailerUserByID(ctx, userID)
	if err != nil {
		return OrgMemberMandateResponse{}, fmt.Errorf("load org member: %w", err)
	}
	if !found || target.RetailerID != retailerID {
		return OrgMemberMandateResponse{}, errMandateMemberNotFound
	}
	role := normalizeRetailerRoleString(target.RetailerRole)
	if target.IsOwner || role == "OWNER" {
		return OrgMemberMandateResponse{}, errMandateOwnerLocked
	}

	next := order.DefaultSellerMandateForRole(role, false)
	if existing, ok, err := s.readExplicitMandate(ctx, retailerID, userID); err != nil {
		return OrgMemberMandateResponse{}, err
	} else if ok {
		next = existing
	}

	if req.CanReceiveGoods != nil {
		next.CanReceiveGoods = *req.CanReceiveGoods
	}
	if req.CanPayCash != nil {
		next.CanPayCash = *req.CanPayCash
	}
	if req.CanUseCredit != nil {
		next.CanUseCredit = *req.CanUseCredit
	}
	if req.DailyCreditCapMinor != nil {
		next.DailyCreditCapMinor = *req.DailyCreditCapMinor
	}
	if req.DailyCashCapMinor != nil {
		next.DailyCashCapMinor = *req.DailyCashCapMinor
	}
	if req.MaxSingleOrderMinor != nil {
		next.MaxSingleOrderMinor = *req.MaxSingleOrderMinor
	}
	next.UpdatedAt = s.currentTime().Format(time.RFC3339Nano)
	next.UpdatedBy = actorUserID

	if s.spannerClient != nil {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			doc := mandatePackDocument{Mandates: make(map[string]SellerMandate)}
			row, rErr := txn.ReadRow(txCtx, "RetailerCapabilityPacks",
				spanner.Key{retailerID, sellerMandatesPackID}, []string{"ConfigJson"})
			if rErr == nil {
				var raw spanner.NullString
				if cErr := row.Columns(&raw); cErr != nil {
					return cErr
				}
				if raw.Valid && strings.TrimSpace(raw.StringVal) != "" {
					if jErr := json.Unmarshal([]byte(raw.StringVal), &doc); jErr != nil {
						return fmt.Errorf("decode existing seller mandate pack: %w", jErr)
					}
					if doc.Mandates == nil {
						doc.Mandates = make(map[string]SellerMandate)
					}
				}
			} else if spanner.ErrCode(rErr) != codes.NotFound {
				return rErr
			}

			doc.Mandates[userID] = next
			cfgBytes, mErr := json.Marshal(doc)
			if mErr != nil {
				return mErr
			}

			muts := []*spanner.Mutation{
				spanner.InsertOrUpdateMap("RetailerCapabilityPacks", map[string]any{
					"RetailerId": retailerID,
					"PackId":     sellerMandatesPackID,
					"Enabled":    false,
					"ConfigJson": string(cfgBytes),
					"UpdatedAt":  spanner.CommitTimestamp,
				}),
			}
			buf := &spannerTxnBuffer{}
			if aErr := outbox.WriteAudit(txCtx, buf, "", actorUserID, "RETAILER_OWNER", "SELLER_MANDATE_UPDATED", "RetailerUser", userID, map[string]any{
				"retailer_id": retailerID,
				"mandate":     next,
			}); aErr != nil {
				return aErr
			}
			for _, a := range buf.audits {
				muts = append(muts, spanner.InsertMap("AuditLog", a.AuditRowMap()))
			}
			return txn.BufferWrite(muts)
		})
		if err != nil {
			return OrgMemberMandateResponse{}, fmt.Errorf("persist seller mandate: %w", err)
		}
	} else {
		s.mu.Lock()
		if s.mandatesByRetailer == nil {
			s.mandatesByRetailer = make(map[string]map[string]SellerMandate)
		}
		if s.mandatesByRetailer[retailerID] == nil {
			s.mandatesByRetailer[retailerID] = make(map[string]SellerMandate)
		}
		s.mandatesByRetailer[retailerID][userID] = next
		s.mu.Unlock()
	}

	usage, err := s.GetSellerDailyUsage(ctx, retailerID, userID)
	if err != nil {
		return OrgMemberMandateResponse{}, err
	}
	return OrgMemberMandateResponse{
		UserID:       target.UserID,
		RetailerID:   target.RetailerID,
		Role:         role,
		RetailerRole: role,
		IsOwner:      false,
		Configured:   true,
		Mandate:      next,
		DailyUsage:   usage,
	}, nil
}

// HandleOrgMemberMandate serves GET and PATCH/PUT /v1/retailer/org/members/{userID}/mandate.
func (s *Service) HandleOrgMemberMandate(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	targetUserID := strings.TrimSpace(chi.URLParam(r, "userID"))
	if targetUserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id_required"})
		return
	}
	actorUserID := auth.ResolveRetailerUserID(claims)

	switch r.Method {
	case http.MethodGet:
		if !auth.HasRetailerPerm(claims, auth.PermStaffManage) && actorUserID != targetUserID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
			return
		}
		resp, err := s.GetSellerMandate(r.Context(), orgID, targetUserID)
		if err != nil {
			writeMandateError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPatch, http.MethodPut:
		if !auth.HasRetailerPerm(claims, auth.PermStaffManage) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStaffManage})
			return
		}
		body, okBody := readLimitedBody(w, r, 64*1024)
		if !okBody {
			return
		}
		if s.guardIdempotency(w, r, body) {
			return
		}
		idemCommitted := false
		defer func() {
			if !idemCommitted {
				s.releaseIdempotency(r.Context(), r)
			}
		}()

		var req MandatePatchRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		if req.Mandate != nil {
			req = *req.Mandate
		}
		if actorUserID == "" {
			actorUserID = orgID
		}
		resp, err := s.UpdateSellerMandate(r.Context(), orgID, targetUserID, actorUserID, req)
		if err != nil {
			writeMandateError(w, err)
			return
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encode_mandate_failed"})
			return
		}
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		idemCommitted = true
		writeJSONBytes(w, http.StatusOK, respBytes)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func writeMandateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errMandateMemberNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "org_member_not_found"})
	case errors.Is(err, errMandateOwnerLocked):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "retailer_org_member_owner_locked"})
	case errors.Is(err, errInvalidMandateCap):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_mandate_cap"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "seller_mandate_failed"})
	}
}

// enforceMandateReceive checks SellerMandate.CanReceiveGoods for non-owner staff before
// allowing dock/stock receiving actions.
func (s *Service) enforceMandateReceive(w http.ResponseWriter, r *http.Request, orgID string, claims auth.Claims) bool {
	userID := strings.TrimSpace(auth.ResolveRetailerUserID(claims))
	if userID == "" || userID == orgID {
		return true
	}
	res, err := s.ResolveSellerMandate(r.Context(), orgID, userID, auth.EffectiveRetailerRole(claims))
	if err != nil {
		if errors.Is(err, errMandateMemberNotFound) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return false
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mandate_lookup_failed"})
		return false
	}
	if !res.IsOwner && !res.Mandate.CanReceiveGoods {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"code":  "mandate_receive_forbidden",
			"error": "mandate_receive_forbidden",
		})
		return false
	}
	return true
}
