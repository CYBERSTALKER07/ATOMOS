package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HandleTokenRefresh accepts a POST with an existing JWT (valid or expired within 24h grace)
// and re-issues a fresh 1-hour token with the same user_id and role.
func HandleTokenRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse the JWT — allow tokens expired within the last 24h grace window.
		claims := &LabClaims{}
		parser := jwt.NewParser(
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithLeeway(24*time.Hour),
		)
		token, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Token invalid or expired beyond grace period", http.StatusUnauthorized)
			return
		}

		if claims.UserID == "" || claims.Role == "" {
			http.Error(w, "Malformed token claims", http.StatusUnauthorized)
			return
		}

		// Mint a fresh token
		newToken, err := GenerateTestToken(claims.UserID, claims.Role)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": newToken,
		})
	}
}
