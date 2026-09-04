package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
)

// C1.2 multi-org auth: intermediate PendingOrgSelect → select-org → full JWT;
// switch-org re-issues full JWT. Flag MULTI_ORG_LOGIN_ENABLED defaults off.

const (
	defaultPendingOrgSelectTTLSec = 420 // 7 minutes
	maxPendingOrgSelectTTLSec     = 600 // 10 minutes
	minPendingOrgSelectTTLSec     = 300 // 5 minutes
)

// multiOrgLoginEnabled reads the feature flag (env or service override for tests).
func (s *Service) multiOrgLoginEnabled() bool {
	if s != nil && s.multiOrgLoginOverride != nil {
		return *s.multiOrgLoginOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("MULTI_ORG_LOGIN_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// multiOrgAllowlist: empty = all orgs eligible when flag on; else only listed retailer IDs.
func multiOrgAllowlist() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("MULTI_ORG_RETAILER_ALLOWLIST"))
	if raw == "" {
		return nil
	}
	out := map[string]struct{}{}
	for _, p := range strings.Split(raw, ",") {
		id := strings.TrimSpace(p)
		if id != "" {
			out[id] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func pendingOrgSelectTTL() time.Duration {
	sec := defaultPendingOrgSelectTTLSec
	if raw := strings.TrimSpace(os.Getenv("PENDING_ORG_SELECT_TTL_SEC")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			sec = n
		}
	}
	if sec < minPendingOrgSelectTTLSec {
		sec = minPendingOrgSelectTTLSec
	}
	if sec > maxPendingOrgSelectTTLSec {
		sec = maxPendingOrgSelectTTLSec
	}
	return time.Duration(sec) * time.Second
}

// shouldOfferOrgPicker is true when multi-org path should return intermediate token.
func shouldOfferOrgPicker(memberships []RetailerMembership) bool {
	if len(memberships) <= 1 {
		return false
	}
	allow := multiOrgAllowlist()
	if allow == nil {
		return true
	}
	// Only offer picker if at least two allowlisted memberships (or all active are allowlisted and >=2).
	n := 0
	for _, m := range memberships {
		if _, ok := allow[m.RetailerID]; ok {
			n++
		}
	}
	return n >= 2
}

// membershipDTO is the wire shape for multi-org picker lists.
type membershipDTO struct {
	UserID       string `json:"user_id"`
	RetailerID   string `json:"retailer_id"`
	RetailerRole string `json:"retailer_role"`
	Name         string `json:"name,omitempty"`
	Phone        string `json:"phone,omitempty"`
	IsActive     bool   `json:"is_active"`
}

func membershipToDTO(m RetailerMembership) membershipDTO {
	return membershipDTO{
		UserID:       m.UserID,
		RetailerID:   m.RetailerID,
		RetailerRole: m.RetailerRole,
		Name:         m.Name,
		Phone:        m.Phone,
		IsActive:     m.IsActive,
	}
}

// maybeWritePendingOrgSelect issues intermediate token when multi-org applies.
// Returns true if response was written.
func (s *Service) maybeWritePendingOrgSelect(w http.ResponseWriter, phone string, memberships []RetailerMembership) bool {
	if !s.multiOrgLoginEnabled() || !shouldOfferOrgPicker(memberships) {
		return false
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return true
	}
	// Prefer a stable subject: first membership user id (legacy per-org rows differ).
	sub := memberships[0].UserID
	if sub == "" {
		sub = phone
	}
	ttl := pendingOrgSelectTTL()
	claims := auth.Claims{
		Subject:     sub,
		Role:        auth.RoleRetailer,
		TokenUse:    auth.TokenUsePendingOrgSelect,
		PhoneNumber: phone,
		SupplierID:  s.seedSupplierID,
	}
	token, err := auth.Issue(claims, auth.IssueOptions{Secret: s.jwtSecret, Issuer: s.jwtIssuer, TTL: ttl})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return true
	}
	dtos := make([]membershipDTO, 0, len(memberships))
	for _, m := range memberships {
		dtos = append(dtos, membershipToDTO(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":            token,
		"token_type":       "pending_org_select",
		"memberships":      dtos,
		"expires_in_sec":   int(ttl.Seconds()),
		"membership_count": len(dtos),
	})
	return true
}

// resolveAuthenticatedMemberships validates password against all users for phone
// and returns active memberships for those that matched credentials.
func (s *Service) resolveAuthenticatedMemberships(ctx context.Context, phone, secret string) ([]RetailerMembership, *RetailerUser, error) {
	users, err := s.listRetailerUsersByPhone(ctx, phone)
	if err != nil {
		return nil, nil, err
	}
	if len(users) == 0 {
		return nil, nil, nil
	}
	var matched []RetailerUser
	for _, u := range users {
		if !u.IsActive {
			continue
		}
		if strings.TrimSpace(u.PasswordHash) == "" {
			// Owners without password: only allow when demo secret matches (legacy).
			if u.IsOwner && secretMatchesDemo(secret) {
				matched = append(matched, u)
			}
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(secret)); err == nil {
			matched = append(matched, u)
		}
	}
	if len(matched) == 0 {
		return nil, nil, nil
	}
	// Build membership list (union) for matched user ids / phones.
	ms, err := s.ListMembershipsByPhone(ctx, phone)
	if err != nil {
		return nil, nil, err
	}
	// Filter to orgs where a matched user belongs (by user id or retailer+phone).
	matchedUID := map[string]bool{}
	matchedOrg := map[string]bool{}
	for _, u := range matched {
		matchedUID[u.UserID] = true
		matchedOrg[u.RetailerID] = true
	}
	var out []RetailerMembership
	seen := map[string]bool{}
	for _, m := range ms {
		if !m.IsActive {
			continue
		}
		if !matchedUID[m.UserID] && !matchedOrg[m.RetailerID] {
			continue
		}
		key := m.RetailerID + "|" + m.UserID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, m)
	}
	// If membership dual-read empty, synthesize from matched users.
	if len(out) == 0 {
		for _, u := range matched {
			out = append(out, RetailerMembership{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
			})
		}
	}
	first := matched[0]
	return out, &first, nil
}

func secretMatchesDemo(secret string) bool {
	expect := demoRetailerSecret()
	return expect != "" && secret == expect
}

// HandleListMemberships serves GET /v1/auth/retailer/memberships
// Accepts PendingOrgSelect or full retailer JWT.
func (s *Service) HandleListMemberships(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	phone := strings.TrimSpace(claims.PhoneNumber)
	var ms []RetailerMembership
	var err error
	if phone != "" {
		ms, err = s.ListMembershipsByPhone(r.Context(), phone)
	} else {
		ms, err = s.ListMembershipsByUser(r.Context(), auth.ResolveRetailerUserID(claims))
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_memberships_failed"})
		return
	}
	dtos := make([]membershipDTO, 0, len(ms))
	for _, m := range ms {
		if m.IsActive {
			dtos = append(dtos, membershipToDTO(m))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"memberships": dtos})
}

// HandleSelectOrg serves POST /v1/auth/retailer/select-org
// body: { "retailer_id": "..." } — requires PendingOrgSelect intermediate token.
func (s *Service) HandleSelectOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !auth.IsPendingOrgSelect(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FULL_TOKEN_REQUIRED"})
		return
	}
	var req struct {
		RetailerID string `json:"retailer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	retailerID := strings.TrimSpace(req.RetailerID)
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}

	phone := strings.TrimSpace(claims.PhoneNumber)
	ms, err := s.ListMembershipsByPhone(r.Context(), phone)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_memberships_failed"})
		return
	}
	var chosen *RetailerMembership
	for i := range ms {
		if ms[i].RetailerID == retailerID {
			chosen = &ms[i]
			break
		}
	}
	if chosen == nil {
		// Fallback: user id scan for org
		if u, found, err := s.findRetailerUserByRetailerPhoneMemOrSpan(r.Context(), retailerID, phone); err == nil && found {
			m := RetailerMembership{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: u.IsActive, Phone: u.Phone, Name: u.Name,
			}
			chosen = &m
		}
	}
	if chosen == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "NOT_A_MEMBER"})
		return
	}
	if !chosen.IsActive {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "MEMBERSHIP_INACTIVE"})
		return
	}
	s.writeFullAuthForMembership(w, r.Context(), *chosen)
}

// HandleSwitchOrg serves POST /v1/auth/retailer/switch-org
// body: { "retailer_id": "..." } — requires full JWT (not intermediate).
func (s *Service) HandleSwitchOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.jwtSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if auth.IsPendingOrgSelect(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "ORG_SELECT_REQUIRED"})
		return
	}
	var req struct {
		RetailerID string `json:"retailer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	retailerID := strings.TrimSpace(req.RetailerID)
	if retailerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_required"})
		return
	}
	// Same org no-op re-issue still OK.
	phone := strings.TrimSpace(claims.PhoneNumber)
	uid := auth.ResolveRetailerUserID(claims)
	var chosen *RetailerMembership
	if phone != "" {
		ms, err := s.ListMembershipsByPhone(r.Context(), phone)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_memberships_failed"})
			return
		}
		for i := range ms {
			if ms[i].RetailerID == retailerID && ms[i].IsActive {
				chosen = &ms[i]
				break
			}
		}
	}
	if chosen == nil && uid != "" {
		ms, err := s.ListMembershipsByUser(r.Context(), uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_memberships_failed"})
			return
		}
		for i := range ms {
			if ms[i].RetailerID == retailerID && ms[i].IsActive {
				chosen = &ms[i]
				break
			}
		}
	}
	// Legacy: same person may have different UserIds per org — match by phone on org users.
	if chosen == nil && phone != "" {
		if u, found, err := s.findRetailerUserByRetailerPhoneMemOrSpan(r.Context(), retailerID, phone); err == nil && found && u.IsActive {
			m := RetailerMembership{
				UserID: u.UserID, RetailerID: u.RetailerID, RetailerRole: u.RetailerRole,
				IsActive: true, Phone: u.Phone, Name: u.Name,
			}
			chosen = &m
		}
	}
	if chosen == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "code": "NOT_A_MEMBER"})
		return
	}
	s.writeFullAuthForMembership(w, r.Context(), *chosen)
}

func (s *Service) writeFullAuthForMembership(w http.ResponseWriter, ctx context.Context, m RetailerMembership) {
	user := RetailerUser{
		UserID: m.UserID, RetailerID: m.RetailerID, Phone: m.Phone, Name: m.Name,
		RetailerRole: m.RetailerRole, IsActive: m.IsActive,
	}
	// Prefer durable user row when available.
	if u, ok, err := s.findRetailerUserByID(ctx, m.UserID); err == nil && ok {
		user = u
	}
	shop := Retailer{
		RetailerID: m.RetailerID,
		Phone:      m.Phone,
		Name:       m.Name,
		SupplierID: s.resolveSupplierScope(ctx),
	}
	if s.repo != nil {
		if r, ok, err := s.repo.GetRetailer(ctx, m.RetailerID); err == nil && ok {
			shop = r
		}
	}
	s.writeMobileAuthResponseForUser(w, shop, user)
}

// findRetailerUserByRetailerPhoneMemOrSpan works with or without Spanner.
func (s *Service) findRetailerUserByRetailerPhoneMemOrSpan(ctx context.Context, retailerID, phone string) (RetailerUser, bool, error) {
	retailerID = strings.TrimSpace(retailerID)
	phone = strings.TrimSpace(phone)
	if retailerID == "" || phone == "" {
		return RetailerUser{}, false, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if u, ok := s.ownerByRetailer[retailerID]; ok && u.Phone == phone {
			return u, true, nil
		}
		for _, u := range s.staffByRetailer[retailerID] {
			if u.Phone == phone {
				return u, true, nil
			}
		}
		return RetailerUser{}, false, nil
	}
	return s.findRetailerUserByRetailerPhone(ctx, retailerID, phone)
}
