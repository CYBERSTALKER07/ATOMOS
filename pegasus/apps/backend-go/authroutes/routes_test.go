package authroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"

	"github.com/go-chi/chi/v5"
)

func TestRegister_RefreshAliasesResolve(t *testing.T) {
	token, err := auth.MintIdentityToken(&auth.PegasusClaims{
		UserID: "factory-user",
		Role:   "FACTORY",
	})
	if err != nil {
		t.Fatalf("MintIdentityToken error: %v", err)
	}

	passthrough := func(next http.HandlerFunc) http.HandlerFunc {
		return next
	}

	router := chi.NewRouter()
	Register(router, Deps{
		Log:       passthrough,
		RateLimit: passthrough,
	})

	paths := []string{
		"/v1/auth/refresh",
		"/v1/auth/factory/refresh",
		"/v1/auth/warehouse/refresh",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
			}
		})
	}
}
