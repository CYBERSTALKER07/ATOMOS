package mfa

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Auditor records MFA enroll/verify events into PlatformAdminAudit.
type Auditor interface {
	RecordFlagAudit(ctx context.Context, actor, action, tenantType, tenantID, detailJSON string) error
}

// Service manages PLATFORM_ADMIN TOTP enrollment and verification.
type Service struct {
	repo     Repository
	issuer   string
	required bool
	audit    Auditor
}

func NewService(repo Repository, issuer string, required bool, audit Auditor) *Service {
	if issuer == "" {
		issuer = "PegasusX"
	}
	return &Service{repo: repo, issuer: issuer, required: required, audit: audit}
}

// Required reports whether PLATFORM_ADMIN_MFA_REQUIRED is on.
func (s *Service) Required() bool {
	return s != nil && s.required
}

// Status for a subject.
type Status struct {
	Enrolled bool `json:"enrolled"`
	Required bool `json:"required"`
	Verified bool `json:"verified"` // from caller JWT claim
}

func (s *Service) Status(ctx context.Context, subject string, verifiedClaim bool) (Status, error) {
	st := Status{Required: s.Required(), Verified: verifiedClaim}
	if s == nil || s.repo == nil || strings.TrimSpace(subject) == "" {
		return st, nil
	}
	rec, ok, err := s.repo.Get(ctx, subject)
	if err != nil {
		return st, err
	}
	st.Enrolled = ok && rec.Enabled
	return st, nil
}

// IsEnrolled reports whether the subject has confirmed TOTP.
func (s *Service) IsEnrolled(ctx context.Context, subject string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	rec, ok, err := s.repo.Get(ctx, strings.TrimSpace(subject))
	if err != nil || !ok {
		return false, err
	}
	return rec.Enabled, nil
}

// NeedsStepUp is true when governance routes require mfa_verified.
func (s *Service) NeedsStepUp(ctx context.Context, subject string) (bool, error) {
	enrolled, err := s.IsEnrolled(ctx, subject)
	if err != nil {
		return false, err
	}
	return enrolled || s.Required(), nil
}

// BeginEnroll creates/rotates a pending secret (Enabled=false until Confirm).
func (s *Service) BeginEnroll(ctx context.Context, subject string) (secret, otpauthURL string, err error) {
	if s == nil || s.repo == nil {
		return "", "", fmt.Errorf("mfa_unavailable")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", "", fmt.Errorf("subject_required")
	}
	secret, err = GenerateSecret()
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	if err := s.repo.Upsert(ctx, Record{
		Subject:   subject,
		Secret:    secret,
		Enabled:   false,
		CreatedAt: now,
	}); err != nil {
		return "", "", err
	}
	return secret, OTPAuthURL(s.issuer, subject, secret), nil
}

// ConfirmEnroll enables MFA after a valid first TOTP code.
func (s *Service) ConfirmEnroll(ctx context.Context, subject, code string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("mfa_unavailable")
	}
	subject = strings.TrimSpace(subject)
	rec, ok, err := s.repo.Get(ctx, subject)
	if err != nil {
		return err
	}
	if !ok || strings.TrimSpace(rec.Secret) == "" {
		return fmt.Errorf("enrollment_not_started")
	}
	if !ValidateCode(rec.Secret, code, time.Now().UTC()) {
		return fmt.Errorf("invalid_totp")
	}
	rec.Enabled = true
	rec.EnabledAt = time.Now().UTC()
	if err := s.repo.Upsert(ctx, rec); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.RecordFlagAudit(ctx, subject, "MFA_ENROLL_CONFIRM", "PLATFORM", subject, `{"mfa":"totp"}`)
	}
	return nil
}

// Verify checks TOTP for an enrolled subject.
func (s *Service) Verify(ctx context.Context, subject, code string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("mfa_unavailable")
	}
	subject = strings.TrimSpace(subject)
	rec, ok, err := s.repo.Get(ctx, subject)
	if err != nil {
		return err
	}
	if !ok || !rec.Enabled {
		return fmt.Errorf("mfa_not_enrolled")
	}
	if !ValidateCode(rec.Secret, code, time.Now().UTC()) {
		return fmt.Errorf("invalid_totp")
	}
	if s.audit != nil {
		_ = s.audit.RecordFlagAudit(ctx, subject, "MFA_VERIFY", "PLATFORM", subject, `{"mfa":"totp"}`)
	}
	return nil
}
