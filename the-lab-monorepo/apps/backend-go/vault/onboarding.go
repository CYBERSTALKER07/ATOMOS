package vault

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"backend-go/hotspot"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// OnboardingStatus tracks a gateway connect session lifecycle.
type OnboardingStatus string

const (
	OnboardingCreated   OnboardingStatus = "CREATED"
	OnboardingPending   OnboardingStatus = "PENDING"
	OnboardingCompleted OnboardingStatus = "COMPLETED"
	OnboardingFailed    OnboardingStatus = "FAILED"
	OnboardingCancelled OnboardingStatus = "CANCELLED"
	OnboardingExpired   OnboardingStatus = "EXPIRED"
)

// OnboardingSession represents a supplier's gateway connect attempt.
type OnboardingSession struct {
	SessionID     string           `json:"session_id"`
	SupplierID    string           `json:"supplier_id"`
	Gateway       string           `json:"gateway"`
	Status        OnboardingStatus `json:"status"`
	StateNonce    string           `json:"state_nonce,omitempty"`
	ReturnSurface string           `json:"return_surface"` // "web" or "desktop"
	RedirectURL   string           `json:"redirect_url,omitempty"`
	ErrorMessage  string           `json:"error_message,omitempty"`
	ExpiresAt     time.Time        `json:"expires_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

// OnboardingSessionSummary is the safe version returned to frontend.
type OnboardingSessionSummary struct {
	SessionID     string           `json:"session_id"`
	Gateway       string           `json:"gateway"`
	Status        OnboardingStatus `json:"status"`
	ReturnSurface string           `json:"return_surface"`
	RedirectURL   string           `json:"redirect_url,omitempty"`
	ErrorMessage  string           `json:"error_message,omitempty"`
	ExpiresAt     time.Time        `json:"expires_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

const onboardingSessionTTL = 15 * time.Minute

// generateStateNonce produces a cryptographically random 32-byte hex string
// used as CSRF protection for OAuth/redirect flows.
func generateStateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("nonce generation: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateOnboardingSession starts a new gateway connect attempt.
// Returns an error if redirect is not supported for the gateway.
func (s *Service) CreateOnboardingSession(ctx context.Context, supplierID, gateway, returnSurface string) (*OnboardingSessionSummary, error) {
	if !SupportsRedirect(gateway) {
		return nil, fmt.Errorf("gateway %s does not support redirect onboarding — use manual configuration", gateway)
	}

	nonce, err := generateStateNonce()
	if err != nil {
		return nil, err
	}

	sessionID := hotspot.NewOrderID()
	now := time.Now()
	expiresAt := now.Add(onboardingSessionTTL)

	_, err = s.Spanner.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("GatewayOnboardingSessions",
			[]string{"SessionId", "SupplierId", "Gateway", "Status", "StateNonce", "ReturnSurface", "ExpiresAt", "CreatedAt", "UpdatedAt"},
			[]interface{}{sessionID, supplierID, gateway, string(OnboardingCreated), nonce, returnSurface, expiresAt, spanner.CommitTimestamp, spanner.CommitTimestamp},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("create onboarding session: %w", err)
	}

	return &OnboardingSessionSummary{
		SessionID:     sessionID,
		Gateway:       gateway,
		Status:        OnboardingCreated,
		ReturnSurface: returnSurface,
		ExpiresAt:     expiresAt,
	}, nil
}

// GetOnboardingSession retrieves a session owned by the supplier.
func (s *Service) GetOnboardingSession(ctx context.Context, supplierID, sessionID string) (*OnboardingSessionSummary, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, Gateway, Status, ReturnSurface, RedirectUrl, ErrorMessage, ExpiresAt, CreatedAt
		      FROM GatewayOnboardingSessions
		      WHERE SessionId = @sid AND SupplierId = @supplierID`,
		Params: map[string]interface{}{"sid": sessionID, "supplierID": supplierID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("onboarding session not found")
	}
	if err != nil {
		return nil, err
	}

	var sess OnboardingSessionSummary
	var redirectURL, errorMsg spanner.NullString
	if err := row.Columns(&sess.SessionID, &sess.Gateway, &sess.Status, &sess.ReturnSurface, &redirectURL, &errorMsg, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
		return nil, err
	}
	if redirectURL.Valid {
		sess.RedirectURL = redirectURL.StringVal
	}
	if errorMsg.Valid {
		sess.ErrorMessage = errorMsg.StringVal
	}
	return &sess, nil
}

// CancelOnboardingSession marks a session as cancelled if it's still active.
func (s *Service) CancelOnboardingSession(ctx context.Context, supplierID, sessionID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "GatewayOnboardingSessions", spanner.Key{sessionID},
			[]string{"SupplierId", "Status"})
		if readErr != nil {
			return fmt.Errorf("session not found: %w", readErr)
		}
		var owner, status string
		if err := row.Columns(&owner, &status); err != nil {
			return err
		}
		if owner != supplierID {
			return fmt.Errorf("session does not belong to supplier")
		}
		if status != string(OnboardingCreated) && status != string(OnboardingPending) {
			return fmt.Errorf("session cannot be cancelled in status %s", status)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("GatewayOnboardingSessions",
				[]string{"SessionId", "Status", "UpdatedAt"},
				[]interface{}{sessionID, string(OnboardingCancelled), spanner.CommitTimestamp}),
		})
	})
	return err
}

// ListActiveOnboardingSessions returns non-terminal sessions for the supplier.
func (s *Service) ListActiveOnboardingSessions(ctx context.Context, supplierID string) ([]OnboardingSessionSummary, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, Gateway, Status, ReturnSurface, RedirectUrl, ErrorMessage, ExpiresAt, CreatedAt
		      FROM GatewayOnboardingSessions
		      WHERE SupplierId = @sid AND Status IN ('CREATED', 'PENDING')
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var sessions []OnboardingSessionSummary
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sess OnboardingSessionSummary
		var redirectURL, errorMsg spanner.NullString
		if err := row.Columns(&sess.SessionID, &sess.Gateway, &sess.Status, &sess.ReturnSurface, &redirectURL, &errorMsg, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, err
		}
		if redirectURL.Valid {
			sess.RedirectURL = redirectURL.StringVal
		}
		if errorMsg.Valid {
			sess.ErrorMessage = errorMsg.StringVal
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}
