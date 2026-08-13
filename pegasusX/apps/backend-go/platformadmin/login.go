package platformadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// LoginDeps issues PLATFORM_ADMIN JWTs after password verification.
type LoginDeps struct {
	Spanner   *spanner.Client
	JWTSecret string
	JWTIssuer string
	TTL       time.Duration
}

// HandleLogin POST /v1/auth/platform-admin/login — G4.B durable admin identity.
// Body: { "subject"|"email", "password" }.
func (d *LoginDeps) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if strings.TrimSpace(d.JWTSecret) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "jwt_not_configured"})
		return
	}
	var req struct {
		Subject  string `json:"subject"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	password := req.Password
	if password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password_required"})
		return
	}
	subject, hash, ok, err := d.lookupAdmin(r.Context(), strings.TrimSpace(req.Subject), strings.TrimSpace(req.Email))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup_failed"})
		return
	}
	if !ok {
		// Env bootstrap admin (SSMR / first boot) when table empty or no row.
		if sub, h, envOK := envBootstrapAdmin(req.Subject, req.Email); envOK {
			subject, hash = sub, h
			ok = true
		}
	}
	if !ok || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		// Constant-time-ish failure message.
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		return
	}
	ttl := d.TTL
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	tok, err := auth.Issue(auth.Claims{
		Subject:      subject,
		Role:         auth.RolePlatformAdmin,
		IsRegistered: true,
		IsConfigured: true,
	}, auth.IssueOptions{Secret: d.JWTSecret, Issuer: d.JWTIssuer, TTL: ttl})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue_token_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      tok,
		"role":       string(auth.RolePlatformAdmin),
		"subject":    subject,
		"token_type": "Bearer",
		"expires_in": int(ttl.Seconds()),
		"mfa_next":   true,
	})
}

func (d *LoginDeps) lookupAdmin(ctx context.Context, subject, email string) (sub, hash string, ok bool, err error) {
	if d == nil || d.Spanner == nil {
		return "", "", false, nil
	}
	if subject != "" {
		row, err := d.Spanner.Single().ReadRow(ctx, "PlatformAdminUsers", spanner.Key{subject},
			[]string{"Subject", "PasswordHash", "Enabled"})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				return "", "", false, nil
			}
			return "", "", false, err
		}
		var enabled bool
		if err := row.Columns(&sub, &hash, &enabled); err != nil {
			return "", "", false, err
		}
		if !enabled {
			return "", "", false, nil
		}
		return sub, hash, true, nil
	}
	if email == "" {
		return "", "", false, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT Subject, PasswordHash, Enabled FROM PlatformAdminUsers WHERE Email = @email LIMIT 1`,
		Params: map[string]interface{}{"email": email},
	}
	iter := d.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return "", "", false, nil
	}
	if err != nil {
		// Table missing.
		if strings.Contains(err.Error(), "PlatformAdminUsers") {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	var enabled bool
	if err := row.Columns(&sub, &hash, &enabled); err != nil {
		return "", "", false, err
	}
	if !enabled {
		return "", "", false, nil
	}
	return sub, hash, true, nil
}

// envBootstrapAdmin allows PLATFORM_ADMIN_SUBJECT + PLATFORM_ADMIN_PASSWORD (plaintext)
// for first-boot SSMR. Hash is generated in-process; not for multi-admin production.
func envBootstrapAdmin(subject, email string) (sub, hash string, ok bool) {
	envSub := strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_SUBJECT"))
	envPass := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	if envSub == "" || envPass == "" {
		return "", "", false
	}
	want := envSub
	if subject != "" && subject != envSub {
		return "", "", false
	}
	if email != "" && !strings.EqualFold(email, envSub) && !strings.EqualFold(email, os.Getenv("PLATFORM_ADMIN_EMAIL")) {
		return "", "", false
	}
	if subject == "" && email != "" && !strings.EqualFold(email, envSub) && !strings.EqualFold(email, os.Getenv("PLATFORM_ADMIN_EMAIL")) {
		return "", "", false
	}
	h, err := bcrypt.GenerateFromPassword([]byte(envPass), bcrypt.DefaultCost)
	if err != nil {
		return "", "", false
	}
	return want, string(h), true
}

// EnsureAdminFromEnv upserts PlatformAdminUsers from PLATFORM_ADMIN_* env (optional).
func EnsureAdminFromEnv(ctx context.Context, client *spanner.Client) error {
	if client == nil {
		return nil
	}
	sub := strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_SUBJECT"))
	pass := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	if sub == "" || pass == "" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	email := strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_EMAIL"))
	now := time.Now().UTC()
	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("PlatformAdminUsers", map[string]interface{}{
			"Subject":      sub,
			"Email":        spanner.NullString{StringVal: email, Valid: email != ""},
			"PasswordHash": string(hash),
			"Enabled":      true,
			"CreatedAt":    now,
			"UpdatedAt":    now,
		}),
	})
	if err != nil && strings.Contains(err.Error(), "PlatformAdminUsers") {
		return nil // migration not applied yet — env bootstrap login still works
	}
	return err
}
