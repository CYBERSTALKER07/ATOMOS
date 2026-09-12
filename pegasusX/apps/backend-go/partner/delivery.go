package partner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// DeliveryWorker posts signed webhook payloads with exponential backoff.
type DeliveryWorker struct {
	webhooks WebhookRepository
	client   *http.Client
	log      *slog.Logger
	now      func() time.Time
}

func NewDeliveryWorker(webhooks WebhookRepository, log *slog.Logger) *DeliveryWorker {
	if log == nil {
		log = slog.Default()
	}
	return &DeliveryWorker{
		webhooks: webhooks,
		client:   &http.Client{Timeout: 15 * time.Second},
		log:      log,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// DeliverHTTP posts one signed body to a subscription URL.
func (w *DeliveryWorker) DeliverHTTP(ctx context.Context, sub WebhookSubscription, eventID, eventType string, body []byte) error {
	ts := w.now().Unix()
	sig := SignPayload(sub.SigningSecret, ts, body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pegasus-Signature", "sha256="+sig)
	req.Header.Set("X-Pegasus-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Pegasus-Event-Id", eventID)
	req.Header.Set("X-Pegasus-Event-Type", eventType)
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http_%d", resp.StatusCode)
	}
	return nil
}

// RunOnce processes due delivery attempts.
func (w *DeliveryWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.webhooks == nil {
		return 0, nil
	}
	due, err := w.webhooks.ListDueAttempts(ctx, w.now(), 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, att := range due {
		sub, ok, err := w.webhooks.GetSubscription(ctx, att.SubscriptionID)
		if err != nil || !ok || !sub.IsActive {
			att.Status = DeliveryDead
			att.LastError = "subscription_inactive"
			_ = w.webhooks.UpdateAttempt(ctx, att)
			continue
		}
		err = w.DeliverHTTP(ctx, sub, att.EventID, att.EventType, att.PayloadJSON)
		att.AttemptCount++
		if err == nil {
			att.Status = DeliverySuccess
			att.HTTPCode = 200
			att.LastError = ""
			att.NextRetryAt = nil
			n++
		} else {
			att.LastError = err.Error()
			if att.AttemptCount >= MaxDeliveryAttempts {
				att.Status = DeliveryDead
			} else {
				att.Status = DeliveryFailed
				backoff := time.Duration(1<<min(att.AttemptCount, 6)) * time.Second
				next := w.now().Add(backoff)
				att.NextRetryAt = &next
			}
		}
		_ = w.webhooks.UpdateAttempt(ctx, att)
	}
	return n, nil
}

// Start loops until context cancel.
func (w *DeliveryWorker) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := w.RunOnce(ctx); err != nil {
				w.log.Warn("webhook delivery pass failed", "err", err)
			} else if n > 0 {
				w.log.Info("webhook delivery pass", "delivered", n)
			}
		}
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
