package supplierlogisticsroutes

import (
	"net/http"
	"net/http/httptest"
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
