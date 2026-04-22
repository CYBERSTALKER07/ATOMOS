package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret holds the HS256 signing key. Populated exclusively by Init().
// The server panics before serving traffic if Init() is not called with a
// non-empty value.
var JWTSecret []byte

// internalAPIKey authenticates service-to-service calls (e.g. AI Worker →
// Backend) via the X-Internal-Key header. Populated exclusively by Init().
var internalAPIKey string

// Init wires the auth package's signing key and internal service credential
// from the loaded configuration. It panics if either value is empty,
// regardless of environment — there are no development fallbacks. Callers
// (main + test setup) are responsible for supplying values.
func Init(jwtSecret, internalKey string) {
	if jwtSecret == "" {
		panic("auth.Init: JWT_SECRET is empty — refusing to boot")
	}
	if internalKey == "" {
		panic("auth.Init: INTERNAL_API_KEY is empty — refusing to boot")
	}
	JWTSecret = []byte(jwtSecret)
	internalAPIKey = internalKey
}

// LabClaims defines the payload inside our cryptographically sealed tokens
type LabClaims struct {
	UserID        string `json:"user_id"`
	SupplierID    string `json:"supplier_id"`    // Organisation-level supplier UUID — use ResolveSupplierID() in handlers
	Role          string `json:"role"`           // "ADMIN", "RETAILER", "SUPPLIER", "DRIVER", "FACTORY", "WAREHOUSE"
	WarehouseID   string `json:"warehouse_id"`   // Warehouse scope — empty = all warehouses (GLOBAL_ADMIN)
	SupplierRole  string `json:"supplier_role"`  // "GLOBAL_ADMIN" or "NODE_ADMIN" — empty for non-supplier roles
	FactoryID     string `json:"factory_id"`     // Factory scope — set for FACTORY role
	FactoryRole   string `json:"factory_role"`   // "FACTORY_ADMIN" or "FACTORY_PAYLOADER"
	WarehouseRole string `json:"warehouse_role"` // "WAREHOUSE_ADMIN" or "WAREHOUSE_STAFF" or "PAYLOADER"
	CountryCode   string `json:"country_code"`   // ISO 3166-1 alpha-2 — supplier's operating country (Phase I)
	IsConfigured  bool   `json:"is_configured"`  // True when billing/categories are set up — used for onboarding gate
	GracePeriod   bool   `json:"-"`              // True when token is expired but within telemetry grace window (A-4)
	jwt.RegisteredClaims
}

// ResolveSupplierID returns the organisation-level SupplierId.
// New tokens carry a dedicated SupplierID field; legacy tokens
// (minted before the field existed) fall back to UserID.
func (c *LabClaims) ResolveSupplierID() string {
	if c.SupplierID != "" {
		return c.SupplierID
	}
	return c.UserID
}

type contextKey string

const ClaimsContextKey = contextKey("claims")

// GenerateTestToken mints a cryptographically signed JWT for local testing
func GenerateTestToken(userID string, role string) (string, error) {
	claims := &LabClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Valid for 24 hours
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// GenerateSupplierToken mints a JWT with full supplier scope fields.
// supplierRole: "GLOBAL_ADMIN" or "NODE_ADMIN" (empty defaults to GLOBAL_ADMIN).
// warehouseID: assigned warehouse for NODE_ADMIN (empty = all warehouses).
func GenerateSupplierToken(userID, role, supplierRole, warehouseID string) (string, error) {
	if supplierRole == "" {
		supplierRole = "GLOBAL_ADMIN"
	}
	claims := &LabClaims{
		UserID:       userID,
		Role:         role,
		SupplierRole: supplierRole,
		WarehouseID:  warehouseID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// MintIdentityToken creates a signed JWT from pre-populated LabClaims.
// Sovereignty Protocol: 1-hour TTL. Eventual consistency accepted for revocation.
func MintIdentityToken(claims *LabClaims) (string, error) {
	if claims.SupplierRole == "" {
		claims.SupplierRole = "GLOBAL_ADMIN"
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// RequireGlobalAdmin checks if the caller is a GLOBAL_ADMIN supplier.
// Returns nil if authorized, writes 403 and returns error otherwise.
// Empty SupplierRole is treated as GLOBAL_ADMIN (root Suppliers table users).
func RequireGlobalAdmin(w http.ResponseWriter, claims *LabClaims) error {
	if claims.SupplierRole == "" || claims.SupplierRole == "GLOBAL_ADMIN" {
		return nil
	}
	log.Printf("[AUTH] 403 NODE_ADMIN %s attempted sovereign action (supplier_role=%s)", claims.UserID, claims.SupplierRole)
	http.Error(w, `{"error":"forbidden","message":"This action requires Global Admin privileges"}`, http.StatusForbidden)
	return errors.New("insufficient supplier role")
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func extractTokenFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	for _, cookieName := range []string{"admin_jwt", "supplier_jwt", "retailer_jwt", "driver_jwt", "payloader_jwt", "factory_jwt", "warehouse_jwt"} {
		if cookie, err := r.Cookie(cookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			return strings.TrimSpace(cookie.Value)
		}
	}

	if isWebSocketUpgrade(r) {
		return strings.TrimSpace(r.URL.Query().Get("token"))
	}

	return ""
}

// RequireRole is the Gatekeeper. It wraps your existing API endpoints.
func RequireRole(allowedRoles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Browsers send a preflight OPTIONS request before secure requests.
		// These do not carry the Authorization header. We must allow them to pass to the CORS layer.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := extractTokenFromRequest(r)
		if tokenStr == "" {
			// ── Internal service-account bypass ──────────────────────────
			// AI Worker and other internal services authenticate via a shared
			// API key in the X-Internal-Key header. This avoids minting JWTs
			// for machine-to-machine calls. The request gets synthetic claims
			// with role "INTERNAL" which satisfies any role check.
			if ik := r.Header.Get("X-Internal-Key"); ik != "" && ik == internalAPIKey {
				synthetic := &LabClaims{
					UserID: "system:ai-worker",
					Role:   "INTERNAL",
				}
				ctx := context.WithValue(r.Context(), ClaimsContextKey, synthetic)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			log.Printf("[AUTH] 401 %s %s transport=%s reason=missing_token allowed=%v", r.Method, r.URL.Path, transportKind(r), allowedRoles)
			http.Error(w, "Vault Locked: Missing Authentication Token", http.StatusUnauthorized)
			return
		}

		// ── Dual-mode token verification ────────────────────────────────
		// Try Firebase ID token first (asymmetric RS256, auto-rotated keys).
		// If Firebase is not initialized or token is not a Firebase token,
		// fall back to legacy HS256 JWT verification.
		claims := &LabClaims{}
		var verified bool

		if FirebaseAuthClient != nil {
			fbClaims, fbErr := VerifyFirebaseToken(r.Context(), tokenStr)
			if fbErr == nil && fbClaims != nil {
				claims = fbClaims
				verified = true
			}
		}

		if !verified {
			// Legacy path: parse HS256 JWT with shared secret
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return JWTSecret, nil
			})
			if err != nil || !token.Valid {
				log.Printf("[AUTH] 401 %s %s transport=%s reason=invalid_token err=%v", r.Method, r.URL.Path, transportKind(r), err)
				http.Error(w, "Vault Locked: Cryptographic Seal Busted", http.StatusUnauthorized)
				return
			}
		}

		// Verify Identity Role — INTERNAL bypasses all role checks
		roleAuthorized := claims.Role == "INTERNAL"
		if !roleAuthorized {
			for _, allowedRole := range allowedRoles {
				if claims.Role == allowedRole {
					roleAuthorized = true
					break
				}
			}
		}

		if !roleAuthorized {
			log.Printf("[AUTH] 403 %s %s transport=%s role=%s allowed=%v", r.Method, r.URL.Path, transportKind(r), claims.Role, allowedRoles)
			http.Error(w, "Clearance Level Insufficient", http.StatusForbidden)
			return
		}

		// Inject the verified claims into the request context for downstream handlers
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireRoleWithGrace is like RequireRole but allows expired DRIVER tokens
// within a 2-hour grace period for telemetry ingestion (A-4). Sets GracePeriod=true
// on claims when operating in grace mode so handlers can restrict to read-only.
func RequireRoleWithGrace(allowedRoles []string, graceWindow time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := extractTokenFromRequest(r)
		if tokenStr == "" {
			http.Error(w, "Vault Locked: Missing Authentication Token", http.StatusUnauthorized)
			return
		}

		claims := &LabClaims{}
		var verified bool

		if FirebaseAuthClient != nil {
			fbClaims, fbErr := VerifyFirebaseToken(r.Context(), tokenStr)
			if fbErr == nil && fbClaims != nil {
				claims = fbClaims
				verified = true
			}
		}

		if !verified {
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return JWTSecret, nil
			})
			if err != nil || !token.Valid {
				// Check if the error is specifically token-expired and within grace window
				if errors.Is(err, jwt.ErrTokenExpired) && claims.UserID != "" && claims.Role == "DRIVER" {
					if claims.ExpiresAt != nil {
						expiry := claims.ExpiresAt.Time
						if time.Since(expiry) <= graceWindow {
							claims.GracePeriod = true
							verified = true
							log.Printf("[AUTH_GRACE] DRIVER %s operating in telemetry grace period (expired %s ago)",
								claims.UserID, time.Since(expiry).Round(time.Second))
						}
					}
				}
				if !verified {
					http.Error(w, "Vault Locked: Cryptographic Seal Busted", http.StatusUnauthorized)
					return
				}
			}
		}

		roleAuthorized := false
		for _, allowedRole := range allowedRoles {
			if claims.Role == allowedRole {
				roleAuthorized = true
				break
			}
		}
		if !roleAuthorized {
			http.Error(w, "Clearance Level Insufficient", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func transportKind(r *http.Request) string {
	if isWebSocketUpgrade(r) {
		return "websocket"
	}
	return "http"
}
