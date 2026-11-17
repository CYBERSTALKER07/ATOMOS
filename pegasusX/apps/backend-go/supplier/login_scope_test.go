package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"golang.org/x/crypto/bcrypt"
)

type loginScopeRepo struct {
	pricingTestRepo
	authByPhone map[string]SupplierAuthRecord
}

func (r *loginScopeRepo) GetAuthByPhone(_ context.Context, phone string) (SupplierAuthRecord, bool, error) {
	rec, ok := r.authByPhone[strings.TrimSpace(phone)]
	return rec, ok, nil
}

func TestHandleLogin_SupportsApprovedWarehouseAdminAndPreservesScope(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Passw0rd!2026"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	repo := &loginScopeRepo{
		authByPhone: map[string]SupplierAuthRecord{
			"+998901234567": {
				UserID:              "usr-wh-admin-1",
				SupplierID:          "sup-tenant-1",
				Phone:               "+998901234567",
				PasswordHash:        string(hash),
				SupplierRole:        auth.RoleWarehouseAdmin,
				AssignedWarehouseID: "wh-main-1",
				MarketCode:          "UZ",
				HomeCell:            "cell-uz",
				IsRegistered:        true,
				IsConfigured:        true,
			},
			"+998909998877": {
				UserID:       "usr-driver-1",
				SupplierID:   "sup-tenant-1",
				Phone:        "+998909998877",
				PasswordHash: string(hash),
				SupplierRole: auth.RoleDriver,
				IsRegistered: true,
				IsConfigured: true,
			},
		},
	}

	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-seed",
		Country:    "UZ",
		Currency:   "UZS",
		JWTSecret:  "test-jwt-secret-key-2026",
		JWTIssuer:  "pegasusx-test",
		JWTTTL:     time.Hour,
	})
	svc.SetINNLookup(innlookup.NewService(innlookup.Config{
		JoinRequests: innlookup.NewMemoryStore(),
	}))

	// 1. WAREHOUSE_ADMIN logs in cleanly and receives scoped claims.
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/supplier/login", strings.NewReader(`{"phone":"+998901234567","password":"Passw0rd!2026"}`))
	rr := httptest.NewRecorder()
	svc.HandleLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.SupplierID != "sup-tenant-1" || resp.UserID != "usr-wh-admin-1" || resp.SupplierRole != string(auth.RoleWarehouseAdmin) {
		t.Fatalf("unexpected login response: %+v", resp)
	}

	claims, err := auth.Parse(resp.Token, "test-jwt-secret-key-2026")
	if err != nil {
		t.Fatalf("parse login token: %v", err)
	}
	if claims.Subject != "usr-wh-admin-1" || claims.Role != auth.RoleAdmin || claims.SupplierRole != auth.RoleWarehouseAdmin ||
		claims.SupplierID != "sup-tenant-1" || claims.HomeNodeType != auth.HomeNodeWarehouse || claims.HomeNodeID != "wh-main-1" ||
		claims.MarketCode != "UZ" || claims.HomeCell != "cell-uz" {
		t.Fatalf("unexpected JWT claims: %+v", claims)
	}

	// 2. Refresh preserves derivative SupplierRole and tenant scope.
	refReq := httptest.NewRequest(http.MethodPost, "/v1/auth/supplier/refresh", strings.NewReader(`{"refresh_token":"`+resp.RefreshToken+`"}`))
	refRR := httptest.NewRecorder()
	svc.HandleSupplierRefresh(refRR, refReq)
	if refRR.Code != http.StatusOK {
		t.Fatalf("refresh status=%d want=200 body=%s", refRR.Code, refRR.Body.String())
	}
	var refBody map[string]string
	if err := json.Unmarshal(refRR.Body.Bytes(), &refBody); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	refClaims, err := auth.Parse(refBody["token"], "test-jwt-secret-key-2026")
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if refClaims.SupplierRole != auth.RoleWarehouseAdmin || refClaims.SupplierID != "sup-tenant-1" || refClaims.Subject != "usr-wh-admin-1" {
		t.Fatalf("refresh lost scope: %+v", refClaims)
	}

	// 3. Derivative WAREHOUSE_ADMIN cannot approve org-wide join requests.
	r := chi.NewRouter()
	r.Post("/v1/supplier/join-requests/{requestId}/approve", svc.HandleApproveJoinRequest)
	approveReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/join-requests/jr-1/approve", strings.NewReader(`{}`))
	approveReq = approveReq.WithContext(auth.WithClaims(approveReq.Context(), claims))
	approveRR := httptest.NewRecorder()
	r.ServeHTTP(approveRR, approveReq)
	if approveRR.Code != http.StatusForbidden {
		t.Fatalf("warehouse_admin approve join request status=%d want=403", approveRR.Code)
	}

	// 4. Non-portal DRIVER role is rejected with 403 portal_role_forbidden.
	drvReq := httptest.NewRequest(http.MethodPost, "/v1/auth/supplier/login", strings.NewReader(`{"phone":"+998909998877","password":"Passw0rd!2026"}`))
	drvRR := httptest.NewRecorder()
	svc.HandleLogin(drvRR, drvReq)
	if drvRR.Code != http.StatusForbidden {
		t.Fatalf("driver login status=%d want=403 body=%s", drvRR.Code, drvRR.Body.String())
	}
}
