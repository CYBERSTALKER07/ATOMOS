package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	webhookInboxStatusPending = "PENDING"
	webhookInboxStatusDead    = "DEAD"
	webhookInboxMaxAttempts   = 5
)

// WebhookInboxStore persists failed webhook processing for async retry.
type WebhookInboxStore struct {
	client *spanner.Client
	now    func() time.Time
}

// NewWebhookInboxStore builds a Spanner-backed webhook inbox.
func NewWebhookInboxStore(client *spanner.Client) *WebhookInboxStore {
	return &WebhookInboxStore{
		client: client,
		now:    time.Now,
	}
}

// Enqueue stores a webhook for later processing.
func (s *WebhookInboxStore) Enqueue(ctx context.Context, row WebhookRecord, source string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("webhook inbox: nil client")
	}
	if strings.TrimSpace(row.WebhookID) == "" {
		row.WebhookID = "wh_inbox_" + uuid.NewString()[:12]
	}
	recordJSON, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal webhook record: %w", err)
	}
	now := s.now().UTC()
	_, err = s.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("WebhookInbox", map[string]any{
			"WebhookId":   row.WebhookID,
			"Gateway":     row.Gateway,
			"RecordJson":  string(recordJSON),
			"Source":      source,
			"Status":      webhookInboxStatusPending,
			"Attempts":    int64(0),
			"NextRetryAt": now,
			"CreatedAt":   now,
			"UpdatedAt":   now,
		}),
	})
	if err != nil {
		return fmt.Errorf("enqueue webhook inbox: %w", err)
	}
	return nil
}

// ProcessPending retries pending inbox rows; returns processed count.
func (s *WebhookInboxStore) ProcessPending(ctx context.Context, svc *Service, limit int) (int, error) {
	if s == nil || s.client == nil || svc == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 20
	}
	now := s.now().UTC()
	stmt := spanner.Statement{
		SQL: `SELECT WebhookId, Gateway, RecordJson, Source, Attempts
		      FROM WebhookInbox
		      WHERE Status = @status
		        AND (NextRetryAt IS NULL OR NextRetryAt <= @now)
		      ORDER BY NextRetryAt
		      LIMIT @lim`,
		Params: map[string]any{
			"status": webhookInboxStatusPending,
			"now":    now,
			"lim":    int64(limit),
		},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	processed := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return processed, fmt.Errorf("list webhook inbox: %w", err)
		}
		var webhookID, gateway, source string
		var recordJSON spanner.NullJSON
		var attempts int64
		if err := row.Columns(&webhookID, &gateway, &recordJSON, &source, &attempts); err != nil {
			continue
		}
		var record WebhookRecord
		
		recordBytes, marshalErr := json.Marshal(recordJSON.Value)
		if marshalErr != nil {
			_ = s.markDead(ctx, webhookID, attempts, marshalErr)
			continue
		}
		
		if err := json.Unmarshal(recordBytes, &record); err != nil {
			_ = s.markDead(ctx, webhookID, attempts, err)
			continue
		}
		if err := svc.persistWebhookWithOutbox(ctx, record, source, now); err != nil {
			_ = s.scheduleRetry(ctx, webhookID, attempts+1, err)
			continue
		}
		_, _ = s.client.Apply(ctx, []*spanner.Mutation{
			spanner.Delete("WebhookInbox", spanner.Key{webhookID}),
		})
		processed++
	}
	return processed, nil
}

func (s *WebhookInboxStore) scheduleRetry(ctx context.Context, webhookID string, attempts int64, cause error) error {
	status := webhookInboxStatusPending
	var nextRetry *time.Time
	if attempts >= webhookInboxMaxAttempts {
		status = webhookInboxStatusDead
	} else {
		t := s.now().UTC().Add(time.Duration(attempts*attempts) * time.Second)
		nextRetry = &t
	}
	_, err := s.client.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("WebhookInbox", map[string]any{
			"WebhookId":   webhookID,
			"Status":      status,
			"Attempts":    attempts,
			"NextRetryAt": nextRetry,
			"LastError":   cause.Error(),
			"UpdatedAt":   s.now().UTC(),
		}),
	})
	return err
}

func (s *WebhookInboxStore) markDead(ctx context.Context, webhookID string, attempts int64, cause error) error {
	return s.scheduleRetry(ctx, webhookID, webhookInboxMaxAttempts, cause)
}

// StartReconciler polls the inbox until ctx is cancelled.
func (s *WebhookInboxStore) StartReconciler(ctx context.Context, svc *Service, interval time.Duration) {
	if s == nil || svc == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.ProcessPending(ctx, svc, 20)
		}
	}
}
