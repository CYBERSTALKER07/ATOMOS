// Package payment — Payment Session Engine (Phase 13)
//
// Manages the lifecycle of durable payment sessions. Every electronic payment
// flows through a session: created at offload confirmation, settled by webhook,
// and queryable by admin/retailer at any time.
//
// A session is the canonical truth for "has this order been paid?"
// MasterInvoices continue to serve as the provider/settlement artifact.
// Orders.PaymentStatus is updated as a projection for fast queries.
package payment

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// ─── Session Status Constants ────────────────────────────────────────────────

const (
	SessionCreated       = "CREATED"
	SessionPending       = "PENDING"
	SessionAuthorized    = "AUTHORIZED"
	SessionSettled       = "SETTLED"
	SessionFailed        = "FAILED"
	SessionExpired       = "EXPIRED"
	SessionCancelled     = "CANCELLED"
	SessionPartiallyPaid = "PARTIALLY_PAID"
)

// ─── Attempt Status Constants ────────────────────────────────────────────────

const (
	AttemptInitiated  = "INITIATED"
	AttemptRedirected = "REDIRECTED"
	AttemptProcessing = "PROCESSING"
	AttemptSuccess    = "SUCCESS"
	AttemptFailed     = "FAILED"
	AttemptCancelled  = "CANCELLED"
	AttemptTimedOut   = "TIMED_OUT"
)

// ─── Domain Types ────────────────────────────────────────────────────────────

// PaymentSession is the Go representation of the PaymentSessions Spanner table.
type PaymentSession struct {
	SessionID         string     `json:"session_id"`
	OrderID           string     `json:"order_id"`
	RetailerID        string     `json:"retailer_id"`
	SupplierID        string     `json:"supplier_id"`
	Gateway           string     `json:"gateway"`
	LockedAmount      int64      `json:"locked_amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	CurrentAttemptNo  int64      `json:"current_attempt_no"`
	InvoiceID         string     `json:"invoice_id,omitempty"`
	RedirectURL       string     `json:"redirect_url,omitempty"`
	ProviderReference string     `json:"provider_reference,omitempty"`
	AuthorizationID   string     `json:"authorization_id,omitempty"`
	AuthorizedAmount  int64      `json:"authorized_amount,omitempty"`
	CapturedAmount    int64      `json:"captured_amount,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	LastErrorCode     string     `json:"last_error_code,omitempty"`
	LastErrorMessage  string     `json:"last_error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	SettledAt         *time.Time `json:"settled_at,omitempty"`
}

// PaymentAttempt is the Go representation of the PaymentAttempts Spanner table.
type PaymentAttempt struct {
	AttemptID             string     `json:"attempt_id"`
	SessionID             string     `json:"session_id"`
	AttemptNo             int64      `json:"attempt_no"`
	Gateway               string     `json:"gateway"`
	ProviderTransactionID string     `json:"provider_transaction_id,omitempty"`
	Status                string     `json:"status"`
	FailureCode           string     `json:"failure_code,omitempty"`
	FailureMessage        string     `json:"failure_message,omitempty"`
	RequestDigest         string     `json:"request_digest,omitempty"`
	StartedAt             time.Time  `json:"started_at"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
}

// CreateSessionRequest is the input for creating a new payment session.
type CreateSessionRequest struct {
	OrderID     string
	RetailerID  string
	SupplierID  string
	Gateway     string
	Amount      int64
	Currency    string
	InvoiceID   string // MasterInvoice ID (if applicable)
	RedirectURL string // Deep-link URL for the gateway
	ExpiresAt   *time.Time
}

// SessionService manages payment session lifecycle in Spanner.
type SessionService struct {
	Spanner *spanner.Client
}

func NewSessionService(client *spanner.Client) *SessionService {
	return &SessionService{Spanner: client}
}

// ─── Create Session ──────────────────────────────────────────────────────────

// CreateSession creates a new payment session for an order.
// Returns an error if an active session already exists for the order.
func (s *SessionService) CreateSession(ctx context.Context, req CreateSessionRequest) (*PaymentSession, error) {
	sessionID := uuid.New().String()
	now := time.Now().UTC()

	session := &PaymentSession{
		SessionID:    sessionID,
		OrderID:      req.OrderID,
		RetailerID:   req.RetailerID,
		SupplierID:   req.SupplierID,
		Gateway:      req.Gateway,
		LockedAmount: req.Amount,
		Currency:     "UZS",
		Status:       SessionCreated,
		InvoiceID:    req.InvoiceID,
		RedirectURL:  req.RedirectURL,
		ExpiresAt:    req.ExpiresAt,
		CreatedAt:    now,
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check for existing active session
		stmt := spanner.Statement{
			SQL: `SELECT SessionId FROM PaymentSessions
			      WHERE OrderId = @orderId AND Status IN ('CREATED', 'PENDING')
			      LIMIT 1`,
			Params: map[string]interface{}{"orderId": req.OrderID},
		}
		iter := txn.Query(ctx, stmt)
		existingRow, existingErr := iter.Next()
		iter.Stop()

		if existingErr == nil && existingRow != nil {
			var existingSessionID string
			if colErr := existingRow.Columns(&existingSessionID); colErr == nil {
				return fmt.Errorf("active payment session %s already exists for order %s", existingSessionID, req.OrderID)
			}
		}

		// Insert the new session
		cols := []string{
			"SessionId", "OrderId", "RetailerId", "SupplierId", "Gateway",
			"LockedAmount", "Currency", "Status", "CurrentAttemptNo",
			"InvoiceId", "RedirectUrl", "ExpiresAt", "CreatedAt",
		}
		vals := []interface{}{
			sessionID, req.OrderID, req.RetailerID, req.SupplierID, req.Gateway,
			req.Amount, "UZS", SessionCreated, int64(0),
			nullStr(req.InvoiceID), nullStr(req.RedirectURL), nullTime(req.ExpiresAt),
			spanner.CommitTimestamp,
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("PaymentSessions", cols, vals),
		})
	})

	if err != nil {
		return nil, err
	}

	log.Printf("[PAYMENT_SESSION] Created session %s for order %s (%s, %d)",
		sessionID, req.OrderID, req.Gateway, req.Amount)
	return session, nil
}

// ─── Create Attempt ──────────────────────────────────────────────────────────

// CreateAttempt creates a new payment attempt within a session and advances the attempt counter.
func (s *SessionService) CreateAttempt(ctx context.Context, sessionID, gateway string) (*PaymentAttempt, error) {
	attemptID := uuid.New().String()
	var attempt PaymentAttempt

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current session
		row, readErr := txn.ReadRow(ctx, "PaymentSessions", spanner.Key{sessionID},
			[]string{"Status", "CurrentAttemptNo"})
		if readErr != nil {
			return fmt.Errorf("session %s not found: %w", sessionID, readErr)
		}

		var status string
		var currentAttemptNo int64
		if err := row.Columns(&status, &currentAttemptNo); err != nil {
			return err
		}

		if status != SessionCreated && status != SessionFailed {
			return fmt.Errorf("session %s in status %s cannot start new attempt", sessionID, status)
		}

		newAttemptNo := currentAttemptNo + 1
		attempt = PaymentAttempt{
			AttemptID: attemptID,
			SessionID: sessionID,
			AttemptNo: newAttemptNo,
			Gateway:   gateway,
			Status:    AttemptInitiated,
			StartedAt: time.Now().UTC(),
		}

		// Update session
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "CurrentAttemptNo", "Status", "UpdatedAt"},
				[]interface{}{sessionID, newAttemptNo, SessionPending, spanner.CommitTimestamp},
			),
		})

		// Insert attempt
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("PaymentAttempts",
				[]string{"AttemptId", "SessionId", "AttemptNo", "Gateway", "Status", "StartedAt"},
				[]interface{}{attemptID, sessionID, newAttemptNo, gateway, AttemptInitiated, spanner.CommitTimestamp},
			),
		})
	})

	if err != nil {
		return nil, err
	}

	log.Printf("[PAYMENT_SESSION] Created attempt %s (#%d) for session %s",
		attemptID, attempt.AttemptNo, sessionID)
	return &attempt, nil
}

// BindProviderCheckout stores provider-created redirect metadata on the session
// after a hosted checkout has been initialized.
func (s *SessionService) BindProviderCheckout(ctx context.Context, sessionID, gateway, invoiceID, redirectURL, providerReference string, expiresAt *time.Time) error {
	_, err := s.Spanner.Apply(ctx, []*spanner.Mutation{
		spanner.Update("PaymentSessions",
			[]string{"SessionId", "Gateway", "InvoiceId", "RedirectUrl", "ProviderReference", "ExpiresAt", "UpdatedAt"},
			[]interface{}{sessionID, gateway, nullStr(invoiceID), nullStr(redirectURL), nullStr(providerReference), nullTime(expiresAt), spanner.CommitTimestamp},
		),
	})
	if err != nil {
		return fmt.Errorf("bind provider checkout failed for session %s: %w", sessionID, err)
	}
	return nil
}

// ─── Settle Session ──────────────────────────────────────────────────────────

// SettleSession marks the session and its current attempt as settled.
// Also updates Orders.PaymentStatus = 'PAID'.
func (s *SessionService) SettleSession(ctx context.Context, sessionID string, providerTxnID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "PaymentSessions", spanner.Key{sessionID},
			[]string{"Status", "OrderId", "CurrentAttemptNo"})
		if readErr != nil {
			return fmt.Errorf("session %s not found: %w", sessionID, readErr)
		}

		var status, orderID string
		var currentAttemptNo int64
		if err := row.Columns(&status, &orderID, &currentAttemptNo); err != nil {
			return err
		}

		if status == SessionSettled {
			log.Printf("[PAYMENT_SESSION] Session %s already settled — idempotent skip", sessionID)
			return nil
		}
		if status != SessionPending {
			return fmt.Errorf("session %s in status %s cannot be settled", sessionID, status)
		}

		now := time.Now().UTC()

		// Update session → SETTLED
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "SettledAt", "UpdatedAt"},
				[]interface{}{sessionID, SessionSettled, now, spanner.CommitTimestamp},
			),
		})

		// Find and update current attempt → SUCCESS
		attemptStmt := spanner.Statement{
			SQL: `SELECT AttemptId FROM PaymentAttempts
			      WHERE SessionId = @sessionId AND AttemptNo = @attemptNo LIMIT 1`,
			Params: map[string]interface{}{"sessionId": sessionID, "attemptNo": currentAttemptNo},
		}
		attemptIter := txn.Query(ctx, attemptStmt)
		attemptRow, attemptErr := attemptIter.Next()
		attemptIter.Stop()

		if attemptErr == nil {
			var attemptID string
			if colErr := attemptRow.Columns(&attemptID); colErr == nil {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentAttempts",
						[]string{"AttemptId", "Status", "ProviderTransactionId", "FinishedAt"},
						[]interface{}{attemptID, AttemptSuccess, nullStr(providerTxnID), now},
					),
				})
			}
		}

		// Update Orders.PaymentStatus = PAID
		_, updateErr := txn.Update(ctx, spanner.Statement{
			SQL:    `UPDATE Orders SET PaymentStatus = 'PAID' WHERE OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		})
		if updateErr != nil {
			log.Printf("[PAYMENT_SESSION] Failed to update PaymentStatus for order %s: %v", orderID, updateErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[PAYMENT_SESSION] Session %s settled (providerTxn=%s)", sessionID, providerTxnID)
	return nil
}

// ─── Fail Session ────────────────────────────────────────────────────────────

// FailSession marks the session and its current attempt as failed.
// Updates Orders.PaymentStatus = 'FAILED'.
func (s *SessionService) FailSession(ctx context.Context, sessionID, errorCode, errorMessage string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "PaymentSessions", spanner.Key{sessionID},
			[]string{"Status", "OrderId", "CurrentAttemptNo"})
		if readErr != nil {
			return fmt.Errorf("session %s not found: %w", sessionID, readErr)
		}

		var status, orderID string
		var currentAttemptNo int64
		if err := row.Columns(&status, &orderID, &currentAttemptNo); err != nil {
			return err
		}

		if status == SessionFailed || status == SessionCancelled {
			return nil // Idempotent
		}
		if status != SessionPending && status != SessionCreated {
			return fmt.Errorf("session %s in status %s cannot be failed", sessionID, status)
		}

		now := time.Now().UTC()

		// Update session → FAILED
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "LastErrorCode", "LastErrorMessage", "UpdatedAt"},
				[]interface{}{sessionID, SessionFailed, nullStr(errorCode), nullStr(errorMessage), spanner.CommitTimestamp},
			),
		})

		// Find and update current attempt → FAILED
		attemptStmt := spanner.Statement{
			SQL: `SELECT AttemptId FROM PaymentAttempts
			      WHERE SessionId = @sessionId AND AttemptNo = @attemptNo LIMIT 1`,
			Params: map[string]interface{}{"sessionId": sessionID, "attemptNo": currentAttemptNo},
		}
		attemptIter := txn.Query(ctx, attemptStmt)
		attemptRow, attemptErr := attemptIter.Next()
		attemptIter.Stop()

		if attemptErr == nil {
			var attemptID string
			if colErr := attemptRow.Columns(&attemptID); colErr == nil {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentAttempts",
						[]string{"AttemptId", "Status", "FailureCode", "FailureMessage", "FinishedAt"},
						[]interface{}{attemptID, AttemptFailed, nullStr(errorCode), nullStr(errorMessage), now},
					),
				})
			}
		}

		// Update Orders.PaymentStatus = FAILED
		_, updateErr := txn.Update(ctx, spanner.Statement{
			SQL:    `UPDATE Orders SET PaymentStatus = 'FAILED' WHERE OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		})
		if updateErr != nil {
			log.Printf("[PAYMENT_SESSION] Failed to update PaymentStatus for order %s: %v", orderID, updateErr)
		}

		return nil
	})
	return err
}

// ─── Expire Session ──────────────────────────────────────────────────────────

// ExpireSession marks the session and its current attempt as expired.
// Updates Orders.PaymentStatus = 'FAILED' (expired payments are a failure mode).
func (s *SessionService) ExpireSession(ctx context.Context, sessionID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "PaymentSessions", spanner.Key{sessionID},
			[]string{"Status", "OrderId", "CurrentAttemptNo"})
		if readErr != nil {
			return fmt.Errorf("session %s not found: %w", sessionID, readErr)
		}

		var status, orderID string
		var currentAttemptNo int64
		if err := row.Columns(&status, &orderID, &currentAttemptNo); err != nil {
			return err
		}

		if status == SessionExpired || status == SessionSettled || status == SessionCancelled {
			return nil // Already terminal — idempotent
		}
		if status != SessionPending && status != SessionCreated {
			return fmt.Errorf("session %s in status %s cannot be expired", sessionID, status)
		}

		now := time.Now().UTC()

		// Update session → EXPIRED
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "LastErrorCode", "LastErrorMessage", "UpdatedAt"},
				[]interface{}{sessionID, SessionExpired, nullStr("SESSION_EXPIRED"), nullStr("checkout window expired"), spanner.CommitTimestamp},
			),
		})

		// Find and update current attempt → TIMED_OUT
		attemptStmt := spanner.Statement{
			SQL: `SELECT AttemptId FROM PaymentAttempts
			      WHERE SessionId = @sessionId AND AttemptNo = @attemptNo LIMIT 1`,
			Params: map[string]interface{}{"sessionId": sessionID, "attemptNo": currentAttemptNo},
		}
		attemptIter := txn.Query(ctx, attemptStmt)
		attemptRow, attemptErr := attemptIter.Next()
		attemptIter.Stop()

		if attemptErr == nil {
			var attemptID string
			if colErr := attemptRow.Columns(&attemptID); colErr == nil {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentAttempts",
						[]string{"AttemptId", "Status", "FailureCode", "FailureMessage", "FinishedAt"},
						[]interface{}{attemptID, AttemptTimedOut, nullStr("SESSION_EXPIRED"), nullStr("checkout window expired"), now},
					),
				})
			}
		}

		// Update Orders.PaymentStatus = FAILED
		_, updateErr := txn.Update(ctx, spanner.Statement{
			SQL:    `UPDATE Orders SET PaymentStatus = 'FAILED' WHERE OrderId = @oid`,
			Params: map[string]interface{}{"oid": orderID},
		})
		if updateErr != nil {
			log.Printf("[PAYMENT_SESSION] Failed to update PaymentStatus for order %s: %v", orderID, updateErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[PAYMENT_SESSION] Session %s expired", sessionID)
	return nil
}

// ─── Sweeper Queries ─────────────────────────────────────────────────────────

// ListExpiredGlobalPaySessions returns GLOBAL_PAY sessions that are still
// CREATED or PENDING but whose ExpiresAt has passed.
func (s *SessionService) ListExpiredGlobalPaySessions(ctx context.Context, limit int) ([]PaymentSession, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE Gateway = 'GLOBAL_PAY'
		        AND Status IN ('CREATED', 'PENDING')
		        AND ExpiresAt IS NOT NULL
		        AND ExpiresAt < @now
		      ORDER BY ExpiresAt ASC
		      LIMIT @limit`,
		Params: map[string]interface{}{
			"now":   time.Now().UTC(),
			"limit": int64(limit),
		},
	}

	return s.querySessionList(ctx, stmt)
}

// ListStaleGlobalPaySessions returns GLOBAL_PAY sessions that are still
// PENDING but have been waiting longer than the given threshold without
// settlement or expiry. These are candidates for provider re-verification.
func (s *SessionService) ListStaleGlobalPaySessions(ctx context.Context, staleThreshold time.Duration, limit int) ([]PaymentSession, error) {
	if limit <= 0 {
		limit = 50
	}
	cutoff := time.Now().UTC().Add(-staleThreshold)
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE Gateway = 'GLOBAL_PAY'
		        AND Status = 'PENDING'
		        AND ProviderReference IS NOT NULL
		        AND CreatedAt < @cutoff
		        AND (ExpiresAt IS NULL OR ExpiresAt >= @now)
		      ORDER BY CreatedAt ASC
		      LIMIT @limit`,
		Params: map[string]interface{}{
			"cutoff": cutoff,
			"now":    time.Now().UTC(),
			"limit":  int64(limit),
		},
	}

	return s.querySessionList(ctx, stmt)
}

func (s *SessionService) querySessionList(ctx context.Context, stmt spanner.Statement) ([]PaymentSession, error) {
	var sessions []PaymentSession
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		session, scanErr := scanSession(row)
		if scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, *session)
	}
	return sessions, nil
}

// ─── Query ───────────────────────────────────────────────────────────────────

// GetSessionByOrder returns the most recent payment session for an order.
func (s *SessionService) GetSessionByOrder(ctx context.Context, orderID string) (*PaymentSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE OrderId = @orderId
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]interface{}{"orderId": orderID},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("no payment session found for order %s", orderID)
	}

	return scanSession(row)
}

// GetActiveSessionByOrder returns the active (CREATED or PENDING) session for an order.
func (s *SessionService) GetActiveSessionByOrder(ctx context.Context, orderID string) (*PaymentSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE OrderId = @orderId AND Status IN ('CREATED', 'PENDING')
		      LIMIT 1`,
		Params: map[string]interface{}{"orderId": orderID},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("no active payment session for order %s", orderID)
	}

	return scanSession(row)
}

// GetSession returns a payment session by ID.
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*PaymentSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE SessionId = @sessionId`,
		Params: map[string]interface{}{"sessionId": sessionID},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("payment session %s not found", sessionID)
	}

	return scanSession(row)
}

// GetAttemptsBySession returns all attempts for a given session.
func (s *SessionService) GetAttemptsBySession(ctx context.Context, sessionID string) ([]PaymentAttempt, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AttemptId, SessionId, AttemptNo, Gateway,
		             ProviderTransactionId, Status, FailureCode, FailureMessage,
		             StartedAt, FinishedAt
		      FROM PaymentAttempts
		      WHERE SessionId = @sessionId
		      ORDER BY AttemptNo ASC`,
		Params: map[string]interface{}{"sessionId": sessionID},
	}

	var attempts []PaymentAttempt
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		a, scanErr := scanAttempt(row)
		if scanErr != nil {
			return nil, scanErr
		}
		attempts = append(attempts, *a)
	}
	return attempts, nil
}

// GetSessionsBySupplier returns payment sessions filtered by supplier and optional status.
func (s *SessionService) GetSessionsBySupplier(ctx context.Context, supplierID, status string, limit int) ([]PaymentSession, error) {
	sql := `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
	               LockedAmount, Currency, Status, CurrentAttemptNo,
	               InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
	               LastErrorCode, LastErrorMessage,
	               CreatedAt, UpdatedAt, SettledAt
	        FROM PaymentSessions
	        WHERE SupplierId = @supplierId`
	params := map[string]interface{}{"supplierId": supplierID}

	if status != "" {
		sql += ` AND Status = @status`
		params["status"] = status
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @limit`
	params["limit"] = int64(limit)

	stmt := spanner.Statement{SQL: sql, Params: params}

	var sessions []PaymentSession
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		session, scanErr := scanSession(row)
		if scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, *session)
	}
	return sessions, nil
}

// ResolveSessionByInvoice finds the session linked to a MasterInvoice ID.
func (s *SessionService) ResolveSessionByInvoice(ctx context.Context, invoiceID string) (*PaymentSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE InvoiceId = @invoiceId
		      LIMIT 1`,
		Params: map[string]interface{}{"invoiceId": invoiceID},
	}

	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("no payment session found for invoice %s", invoiceID)
	}

	return scanSession(row)
}

// GetPendingSessionsByRetailer returns all active (CREATED/PENDING) payment sessions
// for a given retailer — used by the retailer's pending-payments endpoint.
func (s *SessionService) GetPendingSessionsByRetailer(ctx context.Context, retailerID string) ([]PaymentSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, OrderId, RetailerId, SupplierId, Gateway,
		             LockedAmount, Currency, Status, CurrentAttemptNo,
		             InvoiceId, RedirectUrl, ProviderReference, ExpiresAt,
		             LastErrorCode, LastErrorMessage,
		             CreatedAt, UpdatedAt, SettledAt
		      FROM PaymentSessions
		      WHERE RetailerId = @retailerId AND Status IN ('CREATED', 'PENDING')
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"retailerId": retailerID},
	}
	return s.querySessionList(ctx, stmt)
}

// ─── Scanners ────────────────────────────────────────────────────────────────

func scanSession(row *spanner.Row) (*PaymentSession, error) {
	var s PaymentSession
	var invoiceID, redirectURL, providerReference, lastErrCode, lastErrMsg spanner.NullString
	var expiresAt, updatedAt, settledAt spanner.NullTime

	if err := row.Columns(
		&s.SessionID, &s.OrderID, &s.RetailerID, &s.SupplierID, &s.Gateway,
		&s.LockedAmount, &s.Currency, &s.Status, &s.CurrentAttemptNo,
		&invoiceID, &redirectURL, &providerReference, &expiresAt,
		&lastErrCode, &lastErrMsg,
		&s.CreatedAt, &updatedAt, &settledAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan payment session: %w", err)
	}

	s.InvoiceID = invoiceID.StringVal
	s.RedirectURL = redirectURL.StringVal
	s.ProviderReference = providerReference.StringVal
	s.LastErrorCode = lastErrCode.StringVal
	s.LastErrorMessage = lastErrMsg.StringVal
	if expiresAt.Valid {
		s.ExpiresAt = &expiresAt.Time
	}
	if updatedAt.Valid {
		s.UpdatedAt = &updatedAt.Time
	}
	if settledAt.Valid {
		s.SettledAt = &settledAt.Time
	}

	return &s, nil
}

func scanAttempt(row *spanner.Row) (*PaymentAttempt, error) {
	var a PaymentAttempt
	var providerTxnID, failureCode, failureMsg spanner.NullString
	var finishedAt spanner.NullTime

	if err := row.Columns(
		&a.AttemptID, &a.SessionID, &a.AttemptNo, &a.Gateway,
		&providerTxnID, &a.Status, &failureCode, &failureMsg,
		&a.StartedAt, &finishedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan payment attempt: %w", err)
	}

	a.ProviderTransactionID = providerTxnID.StringVal
	a.FailureCode = failureCode.StringVal
	a.FailureMessage = failureMsg.StringVal
	if finishedAt.Valid {
		a.FinishedAt = &finishedAt.Time
	}

	return &a, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// ─── Partial Settlement (F-3) ────────────────────────────────────────────────

// PartialSettleSession marks a session as PARTIALLY_PAID when the collected
// amount is less than the locked amount. Records PaidAmount for the delta.
func (s *SessionService) PartialSettleSession(ctx context.Context, sessionID string, paidAmount int64, providerTxnID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "PaymentSessions", spanner.Key{sessionID},
			[]string{"Status", "OrderId", "LockedAmount", "CurrentAttemptNo"})
		if readErr != nil {
			return fmt.Errorf("session %s not found: %w", sessionID, readErr)
		}

		var status, orderID string
		var lockedAmount, currentAttemptNo int64
		if err := row.Columns(&status, &orderID, &lockedAmount, &currentAttemptNo); err != nil {
			return err
		}

		if status == SessionSettled || status == SessionPartiallyPaid {
			log.Printf("[PAYMENT_SESSION] Session %s already %s — idempotent skip", sessionID, status)
			return nil
		}
		if status != SessionPending {
			return fmt.Errorf("session %s in status %s cannot be partially settled", sessionID, status)
		}

		// If paid amount covers the full locked amount, do a full settlement instead
		if paidAmount >= lockedAmount {
			return nil // Caller should use SettleSession for full payment
		}

		now := time.Now().UTC()

		// Update session → PARTIALLY_PAID with recorded paid amount
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PaymentSessions",
				[]string{"SessionId", "Status", "PaidAmount", "SettledAt", "UpdatedAt"},
				[]interface{}{sessionID, SessionPartiallyPaid, paidAmount, now, spanner.CommitTimestamp},
			),
		})

		// Update current attempt → SUCCESS (partial success is still successful at gateway level)
		attemptStmt := spanner.Statement{
			SQL: `SELECT AttemptId FROM PaymentAttempts
			      WHERE SessionId = @sessionId AND AttemptNo = @attemptNo LIMIT 1`,
			Params: map[string]interface{}{"sessionId": sessionID, "attemptNo": currentAttemptNo},
		}
		attemptIter := txn.Query(ctx, attemptStmt)
		attemptRow, attemptErr := attemptIter.Next()
		attemptIter.Stop()

		if attemptErr == nil {
			var attemptID string
			if colErr := attemptRow.Columns(&attemptID); colErr == nil {
				txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentAttempts",
						[]string{"AttemptId", "Status", "ProviderTransactionId", "FinishedAt"},
						[]interface{}{attemptID, AttemptSuccess, nullStr(providerTxnID), now},
					),
				})
			}
		}

		log.Printf("[PAYMENT_SESSION] Session %s partially settled: %d/%d (order %s)",
			sessionID, paidAmount, lockedAmount, orderID)

		return nil
	})
	return err
}

// ─── Retry Payment (F-4: Card Expiry Mid-Route) ─────────────────────────────

// RetryPaymentSession cancels the current FAILED/EXPIRED session and creates a
// new one with the specified gateway. Used when the original payment method fails
// mid-route and the retailer/driver needs to switch (e.g., card → cash).
func (s *SessionService) RetryPaymentSession(ctx context.Context, orderID string, newGateway string) (*PaymentSession, error) {
	var newSession *PaymentSession

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Find the current active/failed session for this order
		stmt := spanner.Statement{
			SQL: `SELECT SessionId, Status, LockedAmount, RetailerId, SupplierId, Gateway
			      FROM PaymentSessions
			      WHERE OrderId = @oid
			      ORDER BY CreatedAt DESC
			      LIMIT 1`,
			Params: map[string]interface{}{"oid": orderID},
		}
		iter := txn.Query(ctx, stmt)
		row, err := iter.Next()
		iter.Stop()

		if err != nil {
			return fmt.Errorf("no payment session found for order %s: %w", orderID, err)
		}

		var oldSessionID, oldStatus, oldGateway, retailerID, supplierID string
		var lockedAmount int64
		if err := row.Columns(&oldSessionID, &oldStatus, &lockedAmount, &retailerID, &supplierID, &oldGateway); err != nil {
			return err
		}

		// Only allow retry from FAILED, EXPIRED, or CANCELLED states
		if oldStatus != SessionFailed && oldStatus != SessionExpired && oldStatus != SessionCancelled {
			return fmt.Errorf("session %s in status %s cannot be retried (must be FAILED/EXPIRED/CANCELLED)", oldSessionID, oldStatus)
		}

		now := time.Now().UTC()

		// 2. Cancel the old session if not already cancelled
		if oldStatus != SessionCancelled {
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("PaymentSessions",
					[]string{"SessionId", "Status", "UpdatedAt"},
					[]interface{}{oldSessionID, SessionCancelled, spanner.CommitTimestamp},
				),
			})
		}

		// 3. Create new session with the alternate gateway
		newSessionID := uuid.New().String()
		expiry := now.Add(30 * time.Minute)

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("PaymentSessions",
				[]string{"SessionId", "OrderId", "RetailerId", "SupplierId", "Gateway",
					"LockedAmount", "Currency", "Status", "CurrentAttemptNo",
					"ExpiresAt", "CreatedAt"},
				[]interface{}{newSessionID, orderID, retailerID, supplierID, newGateway,
					lockedAmount, "UZS", SessionCreated, int64(0),
					expiry, spanner.CommitTimestamp},
			),
		})

		newSession = &PaymentSession{
			SessionID:    newSessionID,
			OrderID:      orderID,
			RetailerID:   retailerID,
			SupplierID:   supplierID,
			Gateway:      newGateway,
			LockedAmount: lockedAmount,
			Currency:     "UZS",
			Status:       SessionCreated,
			ExpiresAt:    &expiry,
		}

		log.Printf("[PAYMENT_SESSION] Payment retry: order %s switched from %s→%s (old=%s, new=%s)",
			orderID, oldGateway, newGateway, oldSessionID, newSessionID)

		return nil
	})

	if err != nil {
		return nil, err
	}
	return newSession, nil
}

func nullStr(v string) spanner.NullString {
	if v == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: v, Valid: true}
}

func nullTime(t *time.Time) spanner.NullTime {
	if t == nil {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: *t, Valid: true}
}
