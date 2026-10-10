package mfa

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Handlers expose PLATFORM_ADMIN MFA endpoints.
type Handlers struct {
	Svc       *Service
	JWTSecret string
	JWTIssuer string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	st, err := h.Svc.Status(r.Context(), auth.ActorLabel(c), c.MFAVerified)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *Handlers) HandleEnroll(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	subject := auth.ActorLabel(c)
	secret, uri, err := h.Svc.BeginEnroll(r.Context(), subject)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"secret":      secret,
		"otpauth_url": uri,
		"subject":     subject,
	})
}

func (h *Handlers) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	subject := auth.ActorLabel(c)
	if err := h.Svc.ConfirmEnroll(r.Context(), subject, req.Code); err != nil {
		status := http.StatusUnprocessableEntity
		if err.Error() == "invalid_totp" {
			status = http.StatusUnauthorized
		}
		writeErr(w, status, err.Error())
		return
	}
	token, err := h.issueVerified(c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "issue_token_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enrolled": true, "token": token})
}

func (h *Handlers) HandleVerify(w http.ResponseWriter, r *http.Request) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	subject := auth.ActorLabel(c)
	if err := h.Svc.Verify(r.Context(), subject, req.Code); err != nil {
		status := http.StatusUnprocessableEntity
		switch err.Error() {
		case "invalid_totp":
			status = http.StatusUnauthorized
		case "mfa_not_enrolled":
			status = http.StatusConflict
		}
		writeErr(w, status, err.Error())
		return
	}
	token, err := h.issueVerified(c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "issue_token_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "verified": true, "token": token})
}

func (h *Handlers) issueVerified(c auth.Claims) (string, error) {
	c.MFAVerified = true
	c.JTI = "" // new jti for stepped-up session
	return auth.Issue(c, auth.IssueOptions{
		Secret: h.JWTSecret,
		Issuer: h.JWTIssuer,
		TTL:    8 * time.Hour,
	})
}

// RegisterRoutes mounts MFA + step-up gate under /v1/platform-admin.
// Call before or after other platform-admin route registrations; step-up is
// applied via RequireStepUp on sibling route groups.
func RegisterRoutes(r chi.Router, h *Handlers) {
	if h == nil || h.Svc == nil {
		return
	}
	r.Route("/v1/platform-admin/mfa", func(mr chi.Router) {
		mr.Use(auth.RequireRole(auth.RolePlatformAdmin))
		mr.Get("/status", h.HandleStatus)
		mr.Post("/enroll", h.HandleEnroll)
		mr.Post("/confirm", h.HandleConfirm)
		mr.Post("/verify", h.HandleVerify)
	})
}

// RequireStepUp rejects PLATFORM_ADMIN governance calls without mfa_verified
// when the subject is enrolled or PLATFORM_ADMIN_MFA_REQUIRED is set.
func RequireStepUp(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if svc == nil {
				next.ServeHTTP(w, r)
				return
			}
			if strings.Contains(r.URL.Path, "/platform-admin/mfa/") {
				next.ServeHTTP(w, r)
				return
			}
			c, ok := auth.FromContext(r.Context())
			if !ok || c.Role != auth.RolePlatformAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if c.MFAVerified {
				next.ServeHTTP(w, r)
				return
			}
			subject := auth.ActorLabel(c)
			need, err := svc.NeedsStepUp(r.Context(), subject)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "mfa_status_failed")
				return
			}
			if !need {
				next.ServeHTTP(w, r)
				return
			}
			enrolled, _ := svc.IsEnrolled(r.Context(), subject)
			msg := "mfa_verification_required"
			if !enrolled && svc.Required() {
				msg = "mfa_enrollment_required"
			}
			writeErr(w, http.StatusForbidden, msg)
		})
	}
}
