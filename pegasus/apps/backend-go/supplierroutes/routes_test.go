package supplierroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_BillingSetupUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "billing-setup"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/billing/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "billing-setup" {
		t.Fatalf("idempotency guard header = %q, want billing-setup", got)
	}
}

func TestRegisterRoutes_ConfigureUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "configure"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/configure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "configure" {
		t.Fatalf("idempotency guard header = %q, want configure", got)
	}
}

func TestRegisterRoutes_PaymentConfigUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "payment-config"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/payment-config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "payment-config" {
		t.Fatalf("idempotency guard header = %q, want payment-config", got)
	}
}

func TestRegisterRoutes_GatewayOnboardingMutationsUseIdempotency(t *testing.T) {
	assertSupplierRouteIdempotency(t, "/v1/supplier/gateway-onboarding", http.MethodPost, "gateway-onboarding")
	assertSupplierRouteIdempotency(t, "/v1/supplier/gateway-onboarding", http.MethodDelete, "gateway-onboarding")
}

func TestRegisterRoutes_RecipientRegisterUsesIdempotency(t *testing.T) {
	assertSupplierRouteIdempotency(t, "/v1/supplier/payment/recipient/register", http.MethodPost, "recipient-register")
}

func TestRegisterRoutes_ProfileUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "profile"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "profile" {
		t.Fatalf("idempotency guard header = %q, want profile", got)
	}
}

func TestRegisterRoutes_ShiftUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "shift"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/shift", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "shift" {
		t.Fatalf("idempotency guard header = %q, want shift", got)
	}
}

func TestRegisterRoutes_OrgInviteUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "org-invite"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/org/members/invite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "org-invite" {
		t.Fatalf("idempotency guard header = %q, want org-invite", got)
	}
}

func TestRegisterRoutes_OrgMemberActionUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "org-member"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/org/members/member-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "org-member" {
		t.Fatalf("idempotency guard header = %q, want org-member", got)
	}
}

func TestRegisterRoutes_PayloaderCreateUsesIdempotency(t *testing.T) {
	assertSupplierRouteIdempotency(t, "/v1/supplier/staff/payloader", http.MethodPost, "payloader-create")
}

func TestRegisterRoutes_PayloaderRotatePinUsesIdempotency(t *testing.T) {
	assertSupplierRouteIdempotency(t, "/v1/supplier/staff/payloader/worker-1/rotate-pin", http.MethodPost, "payloader-rotate")
}

func TestRegisterRoutes_WarehousesUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "warehouses"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/warehouses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "warehouses" {
		t.Fatalf("idempotency guard header = %q, want warehouses", got)
	}
}

func TestRegisterRoutes_WarehouseActionUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "warehouse-action"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/warehouses/warehouse-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "warehouse-action" {
		t.Fatalf("idempotency guard header = %q, want warehouse-action", got)
	}
}

func TestRegisterRoutes_WarehouseStaffActionUsesIdempotency(t *testing.T) {
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "warehouse-staff-action"),
	})

	req := httptest.NewRequest(http.MethodTrace, "/v1/supplier/warehouse-staff/worker-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != "warehouse-staff-action" {
		t.Fatalf("idempotency guard header = %q, want warehouse-staff-action", got)
	}
}

func supplierTestToken(t *testing.T) string {
	t.Helper()

	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}
	return token
}

func assertSupplierRouteIdempotency(t *testing.T, path, method, marker string) {
	t.Helper()
	token := supplierTestToken(t)

	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:         passthroughMiddleware,
		Idempotency: stoppingMarkerMiddleware("X-Idempotency-Guard", marker),
	})

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("X-Idempotency-Guard"); got != marker {
		t.Fatalf("idempotency guard header = %q, want %s", got, marker)
	}
}

func passthroughMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}

func markerMiddleware(name, value string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(name, value)
			next(w, r)
		}
	}
}

func stoppingMarkerMiddleware(name, value string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(name, value)
			w.WriteHeader(http.StatusAccepted)
		}
	}
}
