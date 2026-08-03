package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runMultiOrgAuthE2E covers Wave C1.4 markers:
//
//	PX_E2E_SINGLE_ORG_LOGIN_UNCHANGED_OK — full JWT business access (flag off default)
//	PX_E2E_PENDING_ORG_REJECT_OK         — intermediate token cannot call business routes
//	PX_E2E_MULTI_ORG_PICKER_OK           — login → pending_org_select → select-org (flag on)
//	PX_E2E_ORG_SWITCH_OK                 — switch-org re-issues full JWT for other org
//
// When MULTI_ORG_LOGIN_ENABLED is not true, multi-org picker/switch print SKIPPED
// so CORE gate still passes with only the single-org + middleware markers required.
func runMultiOrgAuthE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID string) error {
	sharedPhone := envOr("SSMR_MULTI_ORG_PHONE", "+998901000288")
	password := envOr("SSMR_MULTI_ORG_PASSWORD", "MultiOrg!234")
	ownerAPhone := envOr("SSMR_MULTI_ORG_OWNER_A", "+998901000281")
	ownerBPhone := envOr("SSMR_MULTI_ORG_OWNER_B", "+998901000282")

	orgA, _, err := registerRetailerWithPhone(ctx, client, base, cfg, ownerAPhone)
	if err != nil {
		return fmt.Errorf("register org A: %w", err)
	}
	orgB, _, err := registerRetailerWithPhone(ctx, client, base, cfg, ownerBPhone)
	if err != nil {
		return fmt.Errorf("register org B: %w", err)
	}

	tokA, err := issueRetailerOwnerJWT(cfg, supplierID, orgA)
	if err != nil {
		return err
	}
	tokB, err := issueRetailerOwnerJWT(cfg, supplierID, orgB)
	if err != nil {
		return err
	}

	if err := createOrgStaffMember(ctx, client, base, tokA, "Multi Staff A", sharedPhone, password, "CASHIER"); err != nil {
		return fmt.Errorf("staff org A: %w", err)
	}
	if err := createOrgStaffMember(ctx, client, base, tokB, "Multi Staff B", sharedPhone, password, "MANAGER"); err != nil {
		return fmt.Errorf("staff org B: %w", err)
	}

	// --- Always: intermediate token rejected on business routes ---
	pendingTok, err := auth.Issue(auth.Claims{
		Subject:     "pending-user",
		Role:        auth.RoleRetailer,
		TokenUse:    auth.TokenUsePendingOrgSelect,
		PhoneNumber: sharedPhone,
		SupplierID:  supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 7 * time.Minute})
	if err != nil {
		return fmt.Errorf("issue pending jwt: %w", err)
	}
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/me", nil, pendingTok, "")
	if err != nil {
		return fmt.Errorf("pending /me: %w", err)
	}
	if status != http.StatusForbidden {
		return fmt.Errorf("pending /me status %d want 403 body %s", status, string(body))
	}
	if !strings.Contains(string(body), "ORG_SELECT_REQUIRED") {
		return fmt.Errorf("pending /me body missing ORG_SELECT_REQUIRED: %s", string(body))
	}
	fmt.Println("PX_E2E_PENDING_ORG_REJECT_OK")

	// --- Always: full JWT path (single-org style) can access business routes ---
	fullTok, err := issueRetailerOwnerJWT(cfg, supplierID, orgA)
	if err != nil {
		return err
	}
	status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/me", nil, fullTok, "")
	if err != nil {
		return fmt.Errorf("full /me: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("full /me status %d body %s", status, string(body))
	}

	// Login with shared phone when multi-org is OFF should still return a usable full token
	// (first matching user) — CORE regression.
	loginStatus, loginBody, err := postRetailerLogin(ctx, client, base, sharedPhone, password)
	if err != nil {
		return fmt.Errorf("login shared phone: %w", err)
	}
	if loginStatus != http.StatusOK {
		return fmt.Errorf("login shared phone status %d body %s", loginStatus, string(loginBody))
	}
	var loginResp struct {
		Token     string `json:"token"`
		TokenType string `json:"token_type"`
	}
	if err := json.Unmarshal(loginBody, &loginResp); err != nil {
		return fmt.Errorf("decode login: %w", err)
	}
	if strings.TrimSpace(loginResp.Token) == "" {
		return fmt.Errorf("login missing token: %s", string(loginBody))
	}

	multiOn := multiOrgLoginEnvEnabled()
	if multiOn {
		if loginResp.TokenType != "pending_org_select" {
			return fmt.Errorf("flag on: want pending_org_select got %q body %s", loginResp.TokenType, string(loginBody))
		}
		// select-org → full
		selStatus, selBody, err := postSelectOrg(ctx, client, base, loginResp.Token, orgA)
		if err != nil {
			return fmt.Errorf("select-org: %w", err)
		}
		if selStatus != http.StatusOK {
			return fmt.Errorf("select-org status %d body %s", selStatus, string(selBody))
		}
		var selResp struct {
			Token         string `json:"token"`
			TokenType     string `json:"token_type"`
			RetailerOrgID string `json:"retailer_org_id"`
			RetailerID    string `json:"retailer_id"`
		}
		if err := json.Unmarshal(selBody, &selResp); err != nil {
			return fmt.Errorf("decode select-org: %w", err)
		}
		if selResp.TokenType != "full" && selResp.TokenType != "" {
			// backend sets full; empty also acceptable if retailer_id set
			if selResp.RetailerID == "" && selResp.RetailerOrgID == "" {
				return fmt.Errorf("select-org incomplete: %s", string(selBody))
			}
		}
		claims, err := auth.Parse(selResp.Token, cfg.JWTSecret)
		if err != nil {
			return fmt.Errorf("parse select token: %w", err)
		}
		if auth.IsPendingOrgSelect(claims) {
			return fmt.Errorf("select-org still pending")
		}
		gotOrg := auth.ResolveRetailerOrgID(claims)
		if gotOrg != orgA {
			return fmt.Errorf("select-org org=%s want %s", gotOrg, orgA)
		}
		fmt.Println("PX_E2E_MULTI_ORG_PICKER_OK")

		// switch-org → org B
		swStatus, swBody, err := postSwitchOrg(ctx, client, base, selResp.Token, orgB)
		if err != nil {
			return fmt.Errorf("switch-org: %w", err)
		}
		if swStatus != http.StatusOK {
			return fmt.Errorf("switch-org status %d body %s", swStatus, string(swBody))
		}
		var swResp struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(swBody, &swResp); err != nil {
			return fmt.Errorf("decode switch-org: %w", err)
		}
		swClaims, err := auth.Parse(swResp.Token, cfg.JWTSecret)
		if err != nil {
			return fmt.Errorf("parse switch token: %w", err)
		}
		if auth.ResolveRetailerOrgID(swClaims) != orgB {
			return fmt.Errorf("switch-org org=%s want %s", auth.ResolveRetailerOrgID(swClaims), orgB)
		}
		// Business route with switched token
		status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/me", nil, swResp.Token, "")
		if err != nil {
			return fmt.Errorf("switch /me: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("switch /me status %d body %s", status, string(body))
		}
		fmt.Println("PX_E2E_ORG_SWITCH_OK")
	} else {
		// Flag off: must not be intermediate
		if loginResp.TokenType == "pending_org_select" {
			return fmt.Errorf("flag off: login returned pending_org_select (MULTI_ORG_LOGIN_ENABLED should be off)")
		}
		claims, err := auth.Parse(loginResp.Token, cfg.JWTSecret)
		if err != nil {
			return fmt.Errorf("parse flag-off login: %w", err)
		}
		if auth.IsPendingOrgSelect(claims) {
			return fmt.Errorf("flag off: token is PendingOrgSelect")
		}
		status, body, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/retailer/me", nil, loginResp.Token, "")
		if err != nil {
			return fmt.Errorf("flag-off /me: %w", err)
		}
		if status != http.StatusOK {
			return fmt.Errorf("flag-off /me status %d body %s", status, string(body))
		}
		fmt.Println("PX_E2E_MULTI_ORG_PICKER_SKIPPED")
		fmt.Println("PX_E2E_ORG_SWITCH_SKIPPED")
	}

	fmt.Println("PX_E2E_SINGLE_ORG_LOGIN_UNCHANGED_OK")
	fmt.Println("PX_E2E_MULTI_ORG_AUTH_OK")
	return nil
}

func multiOrgLoginEnvEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("MULTI_ORG_LOGIN_ENABLED")))
	// Also allow explicit smoke override
	if v2 := strings.TrimSpace(strings.ToLower(os.Getenv("SSMR_MULTI_ORG_E2E"))); v2 == "1" || v2 == "true" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func issueRetailerOwnerJWT(cfg *bootstrap.Config, supplierID, orgID string) (string, error) {
	return auth.Issue(auth.Claims{
		Subject:        orgID + "-owner",
		Role:           auth.RoleRetailer,
		SupplierID:     supplierID,
		RetailerOrgID:  orgID,
		RetailerRole:   "OWNER",
		RetailerUserID: orgID + "-owner",
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 30 * time.Minute})
}

func createOrgStaffMember(ctx context.Context, client *http.Client, base, ownerToken, name, phone, password, role string) error {
	body, _ := json.Marshal(map[string]any{
		"name":          name,
		"phone":         phone,
		"password":      password,
		"retailer_role": role,
	})
	status, resp, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/retailer/org/members", body, ownerToken, "ssmr-multi-org-staff-"+phone+"-"+role)
	if err != nil {
		return err
	}
	// 201 created, or 409 if re-run
	if status != http.StatusCreated && status != http.StatusOK && status != http.StatusConflict {
		return fmt.Errorf("create member status %d body %s", status, string(resp))
	}
	return nil
}

func postRetailerLogin(ctx context.Context, client *http.Client, base, phone, password string) (int, []byte, error) {
	body, _ := json.Marshal(map[string]string{
		"phone":    phone,
		"password": password,
	})
	status, resp, _, err := clientPostRetry(ctx, client, base+"/v1/auth/retailer/login", body, "", "")
	return status, resp, err
}

func postSelectOrg(ctx context.Context, client *http.Client, base, pendingToken, retailerID string) (int, []byte, error) {
	body, _ := json.Marshal(map[string]string{"retailer_id": retailerID})
	status, resp, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/auth/retailer/select-org", body, pendingToken, "")
	return status, resp, err
}

func postSwitchOrg(ctx context.Context, client *http.Client, base, fullToken, retailerID string) (int, []byte, error) {
	body, _ := json.Marshal(map[string]string{"retailer_id": retailerID})
	status, resp, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/auth/retailer/switch-org", body, fullToken, "")
	return status, resp, err
}
