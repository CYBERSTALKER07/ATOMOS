package supplierlogisticsroutes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"
	"backend-go/supplier"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
)

func TestManifestPathKind(t *testing.T) {
	tests := []struct {
		name string
		path string
		want manifestPathType
	}{
		{name: "root list", path: "/v1/supplier/manifests/", want: manifestPathList},
		{name: "detail", path: "/v1/supplier/manifests/manifest-1", want: manifestPathDetail},
		{name: "start loading", path: "/v1/supplier/manifests/manifest-1/start-loading", want: manifestPathStartLoading},
		{name: "seal", path: "/v1/supplier/manifests/manifest-1/seal", want: manifestPathSeal},
		{name: "inject order", path: "/v1/supplier/manifests/manifest-1/inject-order", want: manifestPathInjectOrder},
		{name: "unknown suffix falls back to list", path: "/v1/supplier/manifests/manifest-1/metrics", want: manifestPathList},
		{name: "missing id falls back to list", path: "/v1/supplier/manifests//seal", want: manifestPathList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestPathKind(tt.path); got != tt.want {
				t.Fatalf("manifestPathKind(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestRegisterRoutes_PayloadRoleAccess(t *testing.T) {
	identity := func(next http.HandlerFunc) http.HandlerFunc { return next }
	router := chi.NewRouter()

	RegisterRoutes(router, Deps{
		Spanner:     &spanner.Client{},
		ManifestSvc: &supplier.ManifestService{},
		Log:         identity,
	})

	tests := []struct {
		name       string
		method     string
		path       string
		role       string
		wantStatus int
	}{
		{
			name:       "payloader can reach manifest list route",
			method:     http.MethodPost,
			path:       "/v1/supplier/manifests",
			role:       "PAYLOADER",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "driver blocked from manifest list route",
			method:     http.MethodPost,
			path:       "/v1/supplier/manifests",
			role:       "DRIVER",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "payloader can reach manifest action route",
			method:     http.MethodGet,
			path:       "/v1/supplier/manifests/manifest-1/seal",
			role:       "PAYLOADER",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "driver blocked from manifest action route",
			method:     http.MethodGet,
			path:       "/v1/supplier/manifests/manifest-1/seal",
			role:       "DRIVER",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "payloader can reach manifest exception route",
			method:     http.MethodGet,
			path:       "/v1/payload/manifest-exception",
			role:       "PAYLOADER",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "driver blocked from manifest exception route",
			method:     http.MethodGet,
			path:       "/v1/payload/manifest-exception",
			role:       "DRIVER",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := auth.GenerateTestToken("payload-route-user", test.role)
			if err != nil {
				t.Fatalf("GenerateTestToken: %v", err)
			}

			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestRegisterRoutes_AutoDispatchUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Spanner:     &spanner.Client{},
		ManifestSvc: &supplier.ManifestService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "auto-dispatch"),
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/supplier/manifests/auto-dispatch", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if got := response.Header().Get("X-Idempotency-Guard"); got != "auto-dispatch" {
		t.Fatalf("idempotency guard header = %q, want auto-dispatch", got)
	}
}

func TestRegisterRoutes_ManualDispatchUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateSupplierToken("supplier-user", "SUPPLIER", "GLOBAL_ADMIN", "")
	if err != nil {
		t.Fatalf("GenerateSupplierToken() error = %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Spanner:     &spanner.Client{},
		ManifestSvc: &supplier.ManifestService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "manual-dispatch"),
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/supplier/manifests/manual-dispatch", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if got := response.Header().Get("X-Idempotency-Guard"); got != "manual-dispatch" {
		t.Fatalf("idempotency guard header = %q, want manual-dispatch", got)
	}
}

func TestRegisterRoutes_ManifestSealUsesIdempotency(t *testing.T) {
	auth.Init("test-jwt-secret", "test-internal-key")
	token, err := auth.GenerateTestToken("payload-route-user", "PAYLOADER")
	if err != nil {
		t.Fatalf("GenerateTestToken: %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Spanner:     &spanner.Client{},
		ManifestSvc: &supplier.ManifestService{},
		Log:         passthroughMiddleware,
		Idempotency: markerMiddleware("X-Idempotency-Guard", "manifest-seal"),
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/supplier/manifests/manifest-1/seal", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("X-Idempotency-Guard"); got != "manifest-seal" {
		t.Fatalf("idempotency guard header = %q, want manifest-seal", got)
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
