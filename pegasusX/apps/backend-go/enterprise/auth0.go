// GS-I: do not mount SetupAuth0Middleware on the chi router.
// A process-global AUTH0_DOMAIN wrap 401s native HS256 staff/driver JWTs.
// Per-supplier OIDC lives in orgoidc (id_token exchange → auth.Issue).

package enterprise

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope        string `json:"scope"`
	Role         string `json:"https://pegasusx.com/role,omitempty"`
	SupplierID   string `json:"https://pegasusx.com/supplier_id,omitempty"`
	SupplierRole string `json:"https://pegasusx.com/supplier_role,omitempty"`
	IsConfigured bool   `json:"https://pegasusx.com/is_configured,omitempty"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// SetupAuth0Middleware configures the JWT validation middleware using Auth0's JWKS.
// It ensures that incoming requests have a valid Access Token signed by the Auth0 tenant.
func SetupAuth0Middleware() func(http.Handler) http.Handler {
	domain := os.Getenv("AUTH0_DOMAIN")
	aud := os.Getenv("AUTH0_AUDIENCE")
	if aud == "" {
		aud = "https://pegasusx-api" // fallback default
	}

	issuerURL, err := url.Parse("https://" + domain + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{os.Getenv("AUTH0_AUDIENCE")},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Encountered error while validating JWT: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	log.Println("Auth0 Enterprise Middleware configured successfully")

	// Return a wrapper that runs Auth0 check, then maps claims to auth.Claims
	return func(next http.Handler) http.Handler {
		return middleware.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Context().Value(jwtmiddleware.ContextKey{})
			if token != nil {
				validatedToken := token.(*validator.ValidatedClaims)
				custom := validatedToken.CustomClaims.(*CustomClaims)
				c := auth.Claims{
					Subject:      validatedToken.RegisteredClaims.Subject,
					Role:         auth.Role(custom.Role),
					SupplierID:   custom.SupplierID,
					SupplierRole: auth.Role(custom.SupplierRole),
					IsConfigured: custom.IsConfigured,
				}
				r = r.WithContext(auth.WithClaims(r.Context(), c))
			}
			next.ServeHTTP(w, r)
		}))
	}
}
