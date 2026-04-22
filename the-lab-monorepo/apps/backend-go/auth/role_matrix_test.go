package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Role Matrix — Systematic verification of every role against every route group
// ═══════════════════════════════════════════════════════════════════════════════

// routeGroup represents a unique combination of allowed roles used in main.go.
type routeGroup struct {
	Name  string
	Roles []string
}

// All distinct role groups registered in main.go RequireRole() calls.
var routeGroups = []routeGroup{
	{"DRIVER-only", []string{"DRIVER"}},
	{"RETAILER-only", []string{"RETAILER"}},
	{"PAYLOADER-only", []string{"PAYLOADER"}},
	{"ADMIN-only", []string{"ADMIN"}},
	{"SUPPLIER+ADMIN", []string{"SUPPLIER", "ADMIN"}},
	{"RETAILER+DRIVER", []string{"RETAILER", "DRIVER"}},
	{"SUPPLIER+DRIVER+ADMIN", []string{"SUPPLIER", "DRIVER", "ADMIN"}},
	{"DRIVER+ADMIN+SUPPLIER", []string{"DRIVER", "ADMIN", "SUPPLIER"}},
	{"ALL-4-ROLES", []string{"RETAILER", "DRIVER", "SUPPLIER", "ADMIN"}},
}

var allRoles = []string{"ADMIN", "SUPPLIER", "RETAILER", "DRIVER", "PAYLOADER"}

func isAllowed(role string, allowedRoles []string) bool {
	for _, r := range allowedRoles {
		if r == role {
			return true
		}
	}
	return false
}

func TestRoleMatrix_FullCoverage(t *testing.T) {
	for _, group := range routeGroups {
		for _, role := range allRoles {
			t.Run(group.Name+"/"+role, func(t *testing.T) {
				tok, err := GenerateTestToken("matrix-user", role)
				if err != nil {
					t.Fatalf("GenerateTestToken: %v", err)
				}

				handler := RequireRole(group.Roles, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest(http.MethodGet, "/test-matrix", nil)
				req.Header.Set("Authorization", "Bearer "+tok)
				w := httptest.NewRecorder()
				handler(w, req)

				expected := http.StatusForbidden
				if isAllowed(role, group.Roles) {
					expected = http.StatusOK
				}

				if w.Code != expected {
					t.Errorf("role=%s group=%s: got %d, want %d", role, group.Name, w.Code, expected)
				}
			})
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cookie-Based Auth Matrix
// ═══════════════════════════════════════════════════════════════════════════════

var roleToCookie = map[string]string{
	"ADMIN":     "admin_jwt",
	"SUPPLIER":  "supplier_jwt",
	"RETAILER":  "retailer_jwt",
	"DRIVER":    "driver_jwt",
	"PAYLOADER": "payloader_jwt",
}

func TestRoleMatrix_CookieAuth(t *testing.T) {
	for _, role := range allRoles {
		t.Run(role, func(t *testing.T) {
			tok, _ := GenerateTestToken("cookie-user", role)
			cookieName := roleToCookie[role]

			handler := RequireRole([]string{role}, func(w http.ResponseWriter, r *http.Request) {
				claims := r.Context().Value(ClaimsContextKey).(*LabClaims)
				if claims.Role != role {
					t.Errorf("claims.Role = %q, want %q", claims.Role, role)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test-cookie", nil)
			req.AddCookie(&http.Cookie{Name: cookieName, Value: tok})
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("cookie auth for %s via %s: got %d, want 200", role, cookieName, w.Code)
			}
		})
	}
}

func TestRoleMatrix_WrongCookie_StillWorks(t *testing.T) {
	// A DRIVER token placed in the admin_jwt cookie should still authenticate
	// because extractTokenFromRequest doesn't verify role-cookie alignment.
	tok, _ := GenerateTestToken("sneaky-user", "DRIVER")
	handler := RequireRole([]string{"DRIVER"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "admin_jwt", Value: tok})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DRIVER token in admin_jwt cookie: got %d, want 200", w.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// WebSocket Query Param Auth
// ═══════════════════════════════════════════════════════════════════════════════

func TestRoleMatrix_WSQueryParamAuth(t *testing.T) {
	for _, role := range allRoles {
		t.Run(role, func(t *testing.T) {
			tok, _ := GenerateTestToken("ws-user", role)
			handler := RequireRole([]string{role}, func(w http.ResponseWriter, r *http.Request) {
				claims := r.Context().Value(ClaimsContextKey).(*LabClaims)
				if claims.Role != role {
					t.Errorf("WS claims.Role = %q, want %q", claims.Role, role)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/ws?token="+tok, nil)
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("WS query param for %s: got %d, want 200", role, w.Code)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// INTERNAL Role Bypass
// ═══════════════════════════════════════════════════════════════════════════════

func TestInternalRole_BypassesAllGroups(t *testing.T) {
	for _, group := range routeGroups {
		t.Run(group.Name, func(t *testing.T) {
			handler := RequireRole(group.Roles, func(w http.ResponseWriter, r *http.Request) {
				claims := r.Context().Value(ClaimsContextKey).(*LabClaims)
				if claims.Role != "INTERNAL" {
					t.Errorf("expected INTERNAL role, got %q", claims.Role)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test-internal", nil)
			req.Header.Set("X-Internal-Key", internalAPIKey)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("INTERNAL bypass on %s: got %d, want 200", group.Name, w.Code)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Cases: Token with INTERNAL role via JWT (not X-Internal-Key)
// ═══════════════════════════════════════════════════════════════════════════════

func TestINTERNALRoleViaJWT_BypassesRoleCheck(t *testing.T) {
	// A JWT with role "INTERNAL" should bypass any role group check
	// because RequireRole checks claims.Role == "INTERNAL" before role matching.
	tok, _ := GenerateTestToken("jwt-internal-user", "INTERNAL")
	handler := RequireRole([]string{"PAYLOADER"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("INTERNAL role JWT should bypass role check: got %d, want 200", w.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Claims Injection Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestClaimsInjected_UserIDPreserved(t *testing.T) {
	for _, role := range allRoles {
		t.Run(role, func(t *testing.T) {
			userID := "uid-" + role
			tok, _ := GenerateTestToken(userID, role)
			handler := RequireRole([]string{role}, func(w http.ResponseWriter, r *http.Request) {
				claims := r.Context().Value(ClaimsContextKey).(*LabClaims)
				if claims.UserID != userID {
					t.Errorf("UserID = %q, want %q", claims.UserID, userID)
				}
				if claims.Role != role {
					t.Errorf("Role = %q, want %q", claims.Role, role)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test-claims", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d for role %s", w.Code, role)
			}
		})
	}
}
