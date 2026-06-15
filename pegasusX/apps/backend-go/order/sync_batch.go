package order

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

// SyncBatchRequest is the wire shape for POST /v1/sync/batch.
type SyncBatchRequest struct {
	DriverID   string          `json:"driver_id"`
	Deliveries []BatchDelivery `json:"deliveries"`
}

// BatchDelivery represents a single offline delivery entry.
type BatchDelivery struct {
	OrderID   string  `json:"order_id"`
	Signature string  `json:"signature"`
	Timestamp float64 `json:"timestamp"`
	Status    string  `json:"status"`
}

// SyncBatchResult represents the outcome of an offline sync batch.
type SyncBatchResult struct {
	Status    string   `json:"status"`
	Processed []string `json:"processed"`
	Skipped   int      `json:"skipped"`
}

// HandleSyncBatch serves POST /v1/sync/batch for offline deliveries.
func (s *Service) HandleSyncBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 128*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	defer r.Body.Close()

	var req SyncBatchRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	driverID := strings.TrimSpace(req.DriverID)
	if driverID == "" || (claims.Subject != driverID && claims.Role != auth.RoleAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "driver_id_mismatch"})
		return
	}

	processed := make([]string, 0, len(req.Deliveries))
	skipped := 0

	for _, delivery := range req.Deliveries {
		orderID, err := s.processBatchDelivery(r.Context(), claims, delivery)
		if err != nil {
			if errors.Is(err, ErrInvalidStatusTransition) || strings.Contains(err.Error(), "no_change") {
				processed = append(processed, delivery.OrderID)
				continue
			}
			if s.log != nil {
				s.log.Warn("sync batch delivery failed", "order_id", delivery.OrderID, "err", err)
			}
			skipped++
			continue
		}
		processed = append(processed, orderID)
	}

	writeJSON(w, http.StatusOK, SyncBatchResult{
		Status:    "OK",
		Processed: processed,
		Skipped:   skipped,
	})
}

func (s *Service) processBatchDelivery(ctx context.Context, claims auth.Claims, delivery BatchDelivery) (string, error) {
	orderID := strings.TrimSpace(delivery.OrderID)
	signature := strings.TrimSpace(delivery.Signature)
	if orderID == "" || signature == "" {
		return "", errors.New("order_id and signature required")
	}

	if s.idem != nil {
		key := "sync-batch:" + orderID + ":" + signature
		hash := sha256HexBytes([]byte(key))
		rec, hit, guardErr := idempotency.Guard(ctx, s.idem, key, hash)
		if guardErr != nil && !errors.Is(guardErr, idempotency.ErrConflict) {
			return "", guardErr
		}
		if hit && rec.StatusCode == http.StatusOK {
			return orderID, nil
		}
	}

	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrOrderNotFound
	}

	publicToken := s.publicDeliveryToken(orderRecord)
	if s.handoff == nil {
		s.handoff = handoff.FromEnv()
	}
	if s.handoff.HashToken(publicToken) != signature {
		return "", errors.New("invalid offline signature")
	}

	subReq := DeliverySubmitRequest{
		OrderID:        orderID,
		ScannedToken:   publicToken,
		BypassGeofence: true,
	}
	if delivery.Timestamp > 0 {
		ts := time.Unix(int64(delivery.Timestamp), 0).UTC()
		subReq.ClientTimestamp = &ts
	}

	_, err = s.SubmitDelivery(ctx, claims, subReq)
	if err == nil && s.idem != nil {
		key := "sync-batch:" + orderID + ":" + signature
		hash := sha256HexBytes([]byte(key))
		_ = s.idem.Save(ctx, key, idempotency.Record{
			BodyHash:   hash,
			StatusCode: http.StatusOK,
			Response:   []byte(`{"status":"OK","order_id":"` + orderID + `"}`),
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour)
	}
	return orderID, err
}
