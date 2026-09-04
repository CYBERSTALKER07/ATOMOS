package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubFirebaseVerifier struct {
	claims Claims
	err    error
	called *bool
}

func (s stubFirebaseVerifier) VerifyIDToken(context.Context, string) (Claims, error) {
	if s.called != nil {
		*s.called = true
	}
	if s.err != nil {
		return Claims{}, s.err
	}
	return s.claims, nil
}

func TestFirebaseAuth_DoesNotAttachFromAuthorization(t *testing.T) {
	t.Parallel()
	called := false
	v := stubFirebaseVerifier{
		claims: Claims{Subject: "fb-user", Role: RoleRetailer, SupplierID: "sup-1"},
		called: &called,
	}
	h := FirebaseAuth(v)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); ok {
			t.Fatal("Authorization Firebase ID must not attach session claims")
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer firebase-id-token")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if called {
		t.Fatal("VerifyIDToken must not run for Authorization Bearer")
	}
}

func TestFirebaseAuth_DoesNotOverwriteJWTClaims(t *testing.T) {
	t.Parallel()
	v := stubFirebaseVerifier{
		claims: Claims{Subject: "fb-user", Role: RoleAdmin, SupplierID: "sup-x"},
	}
	h := FirebaseAuth(v)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		c, ok := FromContext(r.Context())
		if !ok || c.Subject != "jwt-user" || c.Role != RoleRetailer {
			t.Fatalf("jwt claims overwritten: ok=%v sub=%q role=%q", ok, c.Subject, c.Role)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{Subject: "jwt-user", Role: RoleRetailer, SupplierID: "sup-1"}))
	req.Header.Set("Authorization", "Bearer firebase-id-token")
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func TestFirebaseAuth_NilVerifierPassThrough(t *testing.T) {
	t.Parallel()
	h := FirebaseAuth(nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); ok {
			t.Fatal("nil verifier must not attach claims")
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header.Set("Authorization", "Bearer anything")
	h.ServeHTTP(httptest.NewRecorder(), req)
}
