package retailerroutes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
)

func issueMandateRouteToken(t *testing.T, orgID, userID, staffRole string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:        userID,
		Role:           auth.RoleRetailer,
		RetailerOrgID:  orgID,
		RetailerUserID: userID,
		RetailerRole:   staffRole,
	}, auth.IssueOptions{Secret: "retailer-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestRetailerRoutes_MemberMandateEndpoints(t *testing.T) {
	t.Parallel()
	var seq atomic.Int64
	svc := retailer.NewService(retailer.ServiceConfig{
		JWTSecret: "retailer-test-secret",
		JWTIssuer: "test",
		Now:       func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) },
		NewID:     func() string { return fmt.Sprintf("r-mandate-%d", seq.Add(1)) },
	})
	owner, err := svc.EnsureOwnerUser(t.Context(), retailer.Retailer{RetailerID: "ret-route-1", Phone: "+998901112233", Name: "Route Shop"})
	if err != nil {
		t.Fatal(err)
	}

	router := retailerRouter(svc)
	ownerTok := issueMandateRouteToken(t, "ret-route-1", owner.UserID, "OWNER")

	// Create cashier via POST /v1/retailer/org/members.
	reqCreate := httptest.NewRequest(http.MethodPost, "/v1/retailer/org/members", bytes.NewBufferString(`{"name":"Cashier","phone":"+998901112244","password":"pass1234","retailer_role":"CASHIER"}`))
	reqCreate.Header.Set("Authorization", "Bearer "+ownerTok)
	rrCreate := httptest.NewRecorder()
	router.ServeHTTP(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("create member status = %d, want 201 (%s)", rrCreate.Code, rrCreate.Body.String())
	}
	var membersResp struct {
		Items []retailer.OrgMemberDTO `json:"items"`
	}
	if err := json.Unmarshal(rrCreate.Body.Bytes(), &membersResp); err != nil {
		t.Fatal(err)
	}
	var member retailer.OrgMemberDTO
	for _, it := range membersResp.Items {
		if it.RetailerRole == "CASHIER" {
			member = it
			break
		}
	}
	if member.UserID == "" {
		t.Fatalf("cashier missing in create response: %s", rrCreate.Body.String())
	}

	cashierTok := issueMandateRouteToken(t, "ret-route-1", member.UserID, "CASHIER")

	// Unauthenticated -> 401.
	rrUnauth := httptest.NewRecorder()
	router.ServeHTTP(rrUnauth, httptest.NewRequest(http.MethodGet, "/v1/retailer/org/members/"+member.UserID+"/mandate", nil))
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth mandate status = %d, want 401", rrUnauth.Code)
	}

	// Owner PATCH -> 200 OK.
	reqPatch := httptest.NewRequest(http.MethodPatch, "/v1/retailer/org/members/"+member.UserID+"/mandate", bytes.NewBufferString(`{"can_use_credit":true,"daily_credit_cap_minor":30000,"max_single_order_minor":15000}`))
	reqPatch.Header.Set("Authorization", "Bearer "+ownerTok)
	rrPatch := httptest.NewRecorder()
	router.ServeHTTP(rrPatch, reqPatch)
	if rrPatch.Code != http.StatusOK {
		t.Fatalf("PATCH mandate status = %d, want 200 (%s)", rrPatch.Code, rrPatch.Body.String())
	}

	// Cashier GET own mandate -> 200 OK reflecting PATCH.
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/retailer/org/members/"+member.UserID+"/mandate", nil)
	reqGet.Header.Set("Authorization", "Bearer "+cashierTok)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("GET mandate status = %d, want 200 (%s)", rrGet.Code, rrGet.Body.String())
	}
	var got retailer.OrgMemberMandateResponse
	if err := json.Unmarshal(rrGet.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Configured || !got.Mandate.CanUseCredit || got.Mandate.DailyCreditCapMinor != 30000 || got.Mandate.MaxSingleOrderMinor != 15000 {
		t.Fatalf("unexpected GET mandate response: %+v", got)
	}
}
