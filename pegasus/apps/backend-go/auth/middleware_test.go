package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ─── GenerateTestToken ──────────────────────────────────────────────────────

func TestGenerateTestToken_AllRoles(t *testing.T) {
	roles := []string{"ADMIN", "SUPPLIER", "RETAILER", "DRIVER", "PAYLOADER"}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			tok, err := GenerateTestToken("user-"+role, role)
			if err != nil {
				t.Fatalf("GenerateTestToken(%s) error: %v", role, err)
			}
			if tok == "" {
				t.Fatal("empty token")
			}

			// Parse back and verify claims
			claims := &PegasusClaims{}
			parsed, err := jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
				return JWTSecret, nil
			})
			if err != nil || !parsed.Valid {
				t.Fatalf("token invalid: %v", err)
			}
			if claims.Role != role {
				t.Errorf("role = %q, want %q", claims.Role, role)
			}
			if claims.UserID != "user-"+role {
				t.Errorf("userID = %q, want %q", claims.UserID, "user-"+role)
			}
		})
	}
}

func TestGenerateTestToken_Expiry24h(t *testing.T) {
	tok, err := GenerateTestToken("u1", "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	claims := &PegasusClaims{}
	jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	expiry := claims.ExpiresAt.Time
	diff := time.Until(expiry)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expiry diff = %v, want ~24h", diff)
	}
}

func TestGenerateTestToken_Uniqueness(t *testing.T) {
	// Different userIDs must produce different tokens
	tok1, _ := GenerateTestToken("user-aaa", "ADMIN")
	tok2, _ := GenerateTestToken("user-bbb", "ADMIN")
	if tok1 == tok2 {
		t.Error("different userIDs should produce different tokens")
	}
}

func TestMintIdentityToken_PreservesScopedClaims(t *testing.T) {
	tok, err := MintIdentityToken(&PegasusClaims{
		UserID:        "worker-1",
		SupplierID:    "supplier-1",
		Role:          "PAYLOADER",
		WarehouseID:   "warehouse-1",
		WarehouseRole: "PAYLOADER",
	})
	if err != nil {
		t.Fatalf("MintIdentityToken error: %v", err)
	}

	claims := &PegasusClaims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("token invalid: %v", err)
	}
	if claims.SupplierID != "supplier-1" {
		t.Fatalf("supplier_id = %q, want %q", claims.SupplierID, "supplier-1")
	}
	if claims.WarehouseID != "warehouse-1" {
		t.Fatalf("warehouse_id = %q, want %q", claims.WarehouseID, "warehouse-1")
	}
	if claims.WarehouseRole != "PAYLOADER" {
		t.Fatalf("warehouse_role = %q, want %q", claims.WarehouseRole, "PAYLOADER")
	}
}

// ─── extractTokenFromRequest ────────────────────────────────────────────────

func TestExtractToken_Bearer(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer my-jwt-token")
	got := extractTokenFromRequest(r)
	if got != "my-jwt-token" {
		t.Errorf("got %q, want %q", got, "my-jwt-token")
	}
}

func TestExtractToken_Cookies(t *testing.T) {
	cookies := []string{"pegasus_admin_jwt", "pegasus_supplier_jwt", "pegasus_retailer_jwt", "pegasus_driver_jwt", "pegasus_payloader_jwt"}
	for _, name := range cookies {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.AddCookie(&http.Cookie{Name: name, Value: "cookie-token"})
			got := extractTokenFromRequest(r)
			if got != "cookie-token" {
				t.Errorf("got %q, want %q", got, "cookie-token")
			}
		})
	}
}

func TestExtractToken_WSQueryParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws/telemetry?token=ws-token", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	got := extractTokenFromRequest(r)
	if got != "ws-token" {
		t.Errorf("got %q, want %q", got, "ws-token")
	}
}

func TestExtractToken_WSQueryParamFactoryEndpoint(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/ws/factory?token=factory-token", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	got := extractTokenFromRequest(r)
	if got != "factory-token" {
		t.Errorf("got %q, want %q", got, "factory-token")
	}
}

func TestExtractToken_WSQueryParamRejectedOnUnknownEndpoint(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws?token=ws-token", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	got := extractTokenFromRequest(r)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractToken_Empty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := extractTokenFromRequest(r)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtractToken_BearerPriority(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer bearer-wins")
	r.AddCookie(&http.Cookie{Name: "pegasus_admin_jwt", Value: "cookie-loses"})
	got := extractTokenFromRequest(r)
	if got != "bearer-wins" {
		t.Errorf("Bearer should take priority, got %q", got)
	}
}

// ─── isWebSocketUpgrade ─────────────────────────────────────────────────────

func TestIsWebSocketUpgrade_True(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	if !isWebSocketUpgrade(r) {
		t.Error("expected true for valid WS upgrade headers")
	}
}

func TestIsWebSocketUpgrade_CaseInsensitive(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Connection", "upgrade")
	r.Header.Set("Upgrade", "WebSocket")
	if !isWebSocketUpgrade(r) {
		t.Error("expected true for case-insensitive WS headers")
	}
}

func TestIsWebSocketUpgrade_False(t *testing.T) {
	r := httptest.NewRequest("GET", "/api", nil)
	if isWebSocketUpgrade(r) {
		t.Error("expected false for normal HTTP request")
	}
}

// ─── transportKind ──────────────────────────────────────────────────────────

func TestTransportKind_HTTP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if transportKind(r) != "http" {
		t.Error("expected http")
	}
}

func TestTransportKind_WebSocket(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	if transportKind(r) != "websocket" {
		t.Error("expected websocket")
	}
}

// ─── RequireRole ────────────────────────────────────────────────────────────

func TestRequireRole_ValidToken(t *testing.T) {
	tok, _ := GenerateTestToken("user-1", "ADMIN")
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"user_id": claims.UserID, "role": claims.Role})
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["user_id"] != "user-1" || body["role"] != "ADMIN" {
		t.Errorf("claims mismatch: %v", body)
	}
}

func TestRequireRole_WrongRole(t *testing.T) {
	tok, _ := GenerateTestToken("user-1", "DRIVER")
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 403 {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestRequireRole_MissingToken(t *testing.T) {
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireRole_InvalidToken(t *testing.T) {
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer not-a-real-jwt")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireRole_ExpiredToken(t *testing.T) {
	claims := &PegasusClaims{
		UserID: "user-1",
		Role:   "ADMIN",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, _ := token.SignedString(JWTSecret)

	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireRole_OptionsPassthrough(t *testing.T) {
	called := false
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	r := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	if !called {
		t.Error("OPTIONS request should pass through without auth")
	}
	if w.Code != 204 {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestRequireRole_InternalAPIKey(t *testing.T) {
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if claims.Role != "INTERNAL" {
			t.Errorf("role = %q, want INTERNAL", claims.Role)
		}
		if claims.UserID != "system:ai-worker" {
			t.Errorf("userID = %q, want system:ai-worker", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Internal-Key", internalAPIKey)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireRole_WrongInternalKey(t *testing.T) {
	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Internal-Key", "wrong-key")
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireRole_MultipleRoles(t *testing.T) {
	tok, _ := GenerateTestToken("user-1", "RETAILER")
	handler := RequireRole([]string{"ADMIN", "RETAILER", "DRIVER"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestRequireRole_TamperedPayload(t *testing.T) {
	tok, _ := GenerateTestToken("user-1", "ADMIN")
	// Tamper with the payload portion (second segment)
	parts := strings.SplitN(tok, ".", 3)
	if len(parts) == 3 {
		parts[1] = "eyJyb2xlIjoiU1VQRVJBRE1JTiIsInVzZXJfaWQiOiJoYWNrZXIifQ"
		tok = strings.Join(parts, ".")
	}

	handler := RequireRole([]string{"ADMIN"}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with tampered token")
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ─── Context Round-Trip ─────────────────────────────────────────────────────

func TestClaimsContextRoundTrip(t *testing.T) {
	original := &PegasusClaims{UserID: "usr-42", Role: "SUPPLIER"}
	ctx := context.WithValue(context.Background(), ClaimsContextKey, original)
	extracted := ctx.Value(ClaimsContextKey).(*PegasusClaims)
	if extracted.UserID != "usr-42" || extracted.Role != "SUPPLIER" {
		t.Errorf("context round-trip failed: %+v", extracted)
	}
}

func TestRequireRoleWithGrace_AllowsExpiredDriverWithinWindow(t *testing.T) {
	expiry := time.Now().Add(-15 * time.Minute)
	claims := &PegasusClaims{
		UserID: "driver-1",
		Role:   "DRIVER",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, _ := token.SignedString(JWTSecret)

	handler := RequireRoleWithGrace([]string{"DRIVER"}, 2*time.Hour, func(w http.ResponseWriter, r *http.Request) {
		got := r.Context().Value(ClaimsContextKey).(*PegasusClaims)
		if !got.GracePeriod {
			t.Fatal("expected GracePeriod=true")
		}
		if got.GraceDeadline.IsZero() {
			t.Fatal("expected GraceDeadline to be populated")
		}
		if got.GraceDeadline.Before(expiry.Add(2*time.Hour-time.Minute)) || got.GraceDeadline.After(expiry.Add(2*time.Hour+time.Minute)) {
			t.Fatalf("grace deadline = %s, want around %s", got.GraceDeadline, expiry.Add(2*time.Hour))
		}
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/ws/telemetry", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireRoleWithGrace_RejectsExpiredDriverBeyondWindow(t *testing.T) {
	claims := &PegasusClaims{
		UserID: "driver-2",
		Role:   "DRIVER",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-3 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, _ := token.SignedString(JWTSecret)

	handler := RequireRoleWithGrace([]string{"DRIVER"}, 2*time.Hour, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for tokens beyond grace window")
	})

	r := httptest.NewRequest("GET", "/ws/telemetry", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
