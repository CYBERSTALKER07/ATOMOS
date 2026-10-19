package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/iterator"
)

// workerHeartbeatKey mirrors bootstrap worker liveness (avoid import cycle).
const workerHeartbeatKey = "pegasusx:runtime:worker:heartbeat"

// OpsDeps powers G4.B2 platform ops visibility.
type OpsDeps struct {
	Spanner   *spanner.Client
	Outbox    *outbox.SpannerStore
	Redis     *redis.Client
	RunMode   string
	RunsAPI   bool
	RunsWorker bool
}

// HandleOutboxSummary GET /v1/platform-admin/ops/outbox/summary
func (h *Handlers) HandleOutboxSummary(w http.ResponseWriter, r *http.Request) {
	if h.Svc == nil {
		writeErr(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	ops := h.Ops
	if ops == nil || ops.Outbox == nil {
		deadN, deadOK := int64(0), false
		if ops != nil {
			deadN, deadOK = countOutboxDeadLetters(r.Context(), ops.Spanner)
		}
		writeJSON(w, http.StatusOK, outboxSummaryWire(0, false, deadN, deadOK, 0, ""))
		return
	}
	n, err := ops.Outbox.CountUnpublished(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	oldestAge, oldestAt := oldestUnpublishedAge(r.Context(), ops.Spanner)
	deadN, deadOK := countOutboxDeadLetters(r.Context(), ops.Spanner)
	writeJSON(w, http.StatusOK, outboxSummaryWire(n, true, deadN, deadOK, oldestAge, oldestAt))
}

func oldestUnpublishedAge(ctx context.Context, client *spanner.Client) (seconds int64, createdAt string) {
	if client == nil {
		return 0, ""
	}
	stmt := spanner.Statement{
		SQL: `SELECT CreatedAt FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished}
		      WHERE PublishedAt IS NULL ORDER BY CreatedAt ASC LIMIT 1`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, ""
	}
	var ts time.Time
	if err := row.Columns(&ts); err != nil {
		return 0, ""
	}
	age := time.Since(ts.UTC()).Seconds()
	if age < 0 {
		age = 0
	}
	return int64(age), ts.UTC().Format(time.RFC3339Nano)
}

// HandleOutboxEvents GET /v1/platform-admin/ops/outbox/events?limit=
func (h *Handlers) HandleOutboxEvents(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	if ops == nil || ops.Spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"events": []any{}, "available": false})
		return
	}
	lim := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			lim = n
		}
	}
	stmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, TopicName, CreatedAt
		      FROM OutboxEvents@{FORCE_INDEX=Idx_OutboxEvents_Unpublished}
		      WHERE PublishedAt IS NULL
		      ORDER BY CreatedAt ASC
		      LIMIT @lim`,
		Params: map[string]interface{}{"lim": int64(lim)},
	}
	iter := ops.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()
	type row struct {
		EventID       string `json:"event_id"`
		AggregateType string `json:"aggregate_type"`
		AggregateID   string `json:"aggregate_id"`
		TopicName     string `json:"topic_name"`
		CreatedAt     string `json:"created_at"`
	}
	out := make([]row, 0, lim)
	for {
		rrow, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var eid, at, aid, topic string
		var created time.Time
		if err := rrow.Columns(&eid, &at, &aid, &topic, &created); err != nil {
			continue
		}
		out = append(out, row{
			EventID: eid, AggregateType: at, AggregateID: aid, TopicName: topic,
			CreatedAt: created.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events":    out,
		"count":     len(out),
		"available": true,
		"note":      "payload redacted; use Spanner/audit for full body",
	})
}

// HandleOutboxDeadLetters GET /v1/platform-admin/ops/outbox/dead-letters?limit=
// Lists Spanner OutboxDeadLetters (G7.2) — distinct from Kafka topic DLQ CLI replay.
func (h *Handlers) HandleOutboxDeadLetters(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	if ops == nil || ops.Spanner == nil {
		writeJSON(w, http.StatusOK, deadLettersListWire(nil, 0, 0, false, "spanner not wired on this process"))
		return
	}
	lim := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			lim = n
		}
	}
	stmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, TopicName, CreatedAt, DeadLetteredAt,
		             Attempts, COALESCE(LastError, '')
		      FROM OutboxDeadLetters
		      ORDER BY DeadLetteredAt DESC
		      LIMIT @lim`,
		Params: map[string]interface{}{"lim": int64(lim)},
	}
	iter := ops.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()
	type row struct {
		EventID        string `json:"event_id"`
		AggregateType  string `json:"aggregate_type"`
		AggregateID    string `json:"aggregate_id"`
		TopicName      string `json:"topic_name"`
		CreatedAt      string `json:"created_at"`
		DeadLetteredAt string `json:"dead_lettered_at"`
		Attempts       int64  `json:"attempts"`
		LastError      string `json:"last_error,omitempty"`
	}
	out := make([]row, 0, lim)
	for {
		rrow, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table absent in older envs.
			if strings.Contains(err.Error(), "OutboxDeadLetters") {
				writeJSON(w, http.StatusOK, deadLettersListWire(nil, 0, 0, false, "OutboxDeadLetters table not applied"))
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var eid, at, aid, topic, lastErr string
		var created, deadAt time.Time
		var attempts int64
		if err := rrow.Columns(&eid, &at, &aid, &topic, &created, &deadAt, &attempts, &lastErr); err != nil {
			continue
		}
		out = append(out, row{
			EventID: eid, AggregateType: at, AggregateID: aid, TopicName: topic,
			CreatedAt: created.UTC().Format(time.RFC3339Nano),
			DeadLetteredAt: deadAt.UTC().Format(time.RFC3339Nano),
			Attempts: attempts, LastError: lastErr,
		})
	}
	total, totalOK := countOutboxDeadLetters(r.Context(), ops.Spanner)
	writeJSON(w, http.StatusOK, deadLettersListWire(out, len(out), total, totalOK, "Spanner outbox dead letters; Kafka topic DLQ still via cmd/replay-dlq"))
}

// HandleRuntime GET /v1/platform-admin/ops/runtime
func (h *Handlers) HandleRuntime(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	runMode := "all"
	runsAPI, runsWorker := true, true
	workersLive := false
	if ops != nil {
		if ops.RunMode != "" {
			runMode = strings.ToLower(strings.TrimSpace(ops.RunMode))
			if runMode != "api" && runMode != "worker" {
				runMode = "all"
			}
		}
		runsAPI = ops.RunsAPI
		runsWorker = ops.RunsWorker
		if ops.Redis != nil {
			workersLive = workerLive(r.Context(), ops.Redis)
		}
	}
	relayOnThisProcess := runsWorker
	writeJSON(w, http.StatusOK, map[string]any{
		"run_mode":                runMode,
		"this_process_api":        runsAPI,
		"this_process_workers":    runsWorker,
		"outbox_relay_on_process": relayOnThisProcess,
		"workers_live_cluster":    workersLive,
		"full_bus_claimed":        runMode == "all" || (runsAPI && workersLive),
		"dlq_note":                "Spanner OutboxDeadLetters via /ops/outbox/dead-letters; Kafka topic DLQ via cmd/replay-dlq CLI",
		"honesty": map[string]any{
			"api_only_no_relay": runMode == "api" && !workersLive,
		},
	})
}

func workerLive(ctx context.Context, client *redis.Client) bool {
	if client == nil {
		return false
	}
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	n, err := client.Exists(c, workerHeartbeatKey).Result()
	return err == nil && n > 0
}

// countOutboxDeadLetters is SELECT COUNT(*) FROM OutboxDeadLetters.
// available=false means the table/client is missing — not an empty (zero) queue.
func countOutboxDeadLetters(ctx context.Context, client *spanner.Client) (int64, bool) {
	if client == nil {
		return 0, false
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: `SELECT COUNT(*) FROM OutboxDeadLetters`})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, false
	}
	var n int64
	if err := row.Columns(&n); err != nil {
		return 0, false
	}
	return n, true
}

// outboxSummaryWire never uses page length as dead_letter_count.
// When deadOK is false the count key is omitted so UI cannot treat 0 as empty.
func outboxSummaryWire(unpublished int64, unpublishedOK bool, dead int64, deadOK bool, oldestAge int64, oldestAt string) map[string]any {
	out := map[string]any{
		"unpublished_available": unpublishedOK,
		"dead_letter_available": deadOK,
		"oldest_age_seconds":    oldestAge,
		"oldest_created_at":     oldestAt,
		"lag_alert_threshold":   120,
		"lagging":               unpublishedOK && oldestAge > 120,
		"available":             unpublishedOK,
	}
	if unpublishedOK {
		out["unpublished_count"] = unpublished
	} else {
		out["note"] = "outbox store not wired on this process"
	}
	if deadOK {
		out["dead_letter_count"] = dead
	}
	return out
}

// deadLettersListWire keeps page_count (this response) distinct from dead_letter_count (table).
func deadLettersListWire(items any, pageCount int, total int64, available bool, note string) map[string]any {
	if items == nil {
		items = []any{}
	}
	out := map[string]any{
		"items":      items,
		"page_count": pageCount,
		"available":  available,
	}
	if note != "" {
		out["note"] = note
	}
	if available {
		out["dead_letter_count"] = total
	}
	return out
}

// HandleReplayDeadLetter POST /v1/platform-admin/ops/outbox/dead-letters/{eventId}/replay
// Re-queues a failed outbox event from OutboxDeadLetters back to OutboxEvents.
func (h *Handlers) HandleReplayDeadLetter(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	if ops == nil || ops.Spanner == nil {
		writeErr(w, http.StatusServiceUnavailable, "spanner not wired on this process")
		return
	}
	eventID := strings.TrimSpace(chi.URLParam(r, "eventId"))
	if eventID == "" {
		writeErr(w, http.StatusBadRequest, "eventId required")
		return
	}

	ctx := r.Context()
	row, err := ops.Spanner.Single().ReadRow(ctx, "OutboxDeadLetters", spanner.Key{eventID}, []string{
		"EventId", "AggregateType", "AggregateId", "TopicName", "Payload", "SupplierId",
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "dead letter not found")
		return
	}
	var eid, at, aid, topic, supplierID string
	var payload []byte
	if err := row.Columns(&eid, &at, &aid, &topic, &payload, &supplierID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = ops.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		supplierCol := supplierID
		if supplierCol == "" {
			supplierCol = outbox.PlatformSupplierID
		}
		outboxMutation := spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":         eid,
			"AggregateType":   at,
			"AggregateId":     aid,
			"TopicName":       topic,
			"Payload":         payload,
			"CreatedAt":       spanner.CommitTimestamp,
			"PublishedAt":     nil,
			"ClaimedBy":       nil,
			"ClaimedUntil":    nil,
			"PublishAttempts": int64(0),
			"SupplierId":      supplierCol,
		})
		deleteMutation := spanner.Delete("OutboxDeadLetters", spanner.Key{eid})
		return txn.BufferWrite([]*spanner.Mutation{outboxMutation, deleteMutation})
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to re-queue dead letter: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"replayed": true,
		"event_id": eid,
		"topic":    topic,
		"message":  "event re-queued to outbox for immediate processing",
	})
}

// HandleListDisputes GET /v1/platform-admin/ops/disputes?limit=
// Lists orders requiring reconciliation / dispute hold.
func (h *Handlers) HandleListDisputes(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	if ops == nil || ops.Spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "count": 0, "available": false})
		return
	}
	lim := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			lim = n
		}
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId, DriverId, Status, TotalMinor, Currency, CreatedAt, UpdatedAt
		      FROM Orders
		      WHERE Status = 'RECONCILIATION_REQUIRED'
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]interface{}{"lim": int64(lim)},
	}
	iter := ops.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	type disputeRow struct {
		OrderID    string `json:"order_id"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
		DriverID   string `json:"driver_id"`
		Status     string `json:"status"`
		TotalMinor int64  `json:"total_minor"`
		Currency   string `json:"currency"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}
	items := make([]disputeRow, 0, lim)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "Orders") {
				writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "count": 0, "available": false, "note": "Orders table not queryable"})
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var oid, sid, rid, drvID, status, curr string
		var total int64
		var created, updated time.Time
		if err := row.Columns(&oid, &sid, &rid, &drvID, &status, &total, &curr, &created, &updated); err != nil {
			continue
		}
		items = append(items, disputeRow{
			OrderID:    oid,
			SupplierID: sid,
			RetailerID: rid,
			DriverID:   drvID,
			Status:     status,
			TotalMinor: total,
			Currency:   curr,
			CreatedAt:  created.UTC().Format(time.RFC3339Nano),
			UpdatedAt:  updated.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     items,
		"count":     len(items),
		"available": true,
	})
}

type resolveDisputeRequest struct {
	Resolution string `json:"resolution"` // "CONFIRM_DELIVERY" or "CONFIRM_CANCEL"
	Notes      string `json:"notes"`
}

// HandleResolveDispute POST /v1/platform-admin/ops/disputes/{orderId}/resolve
// Resolves an offline delivery dispute by confirming delivery or cancellation.
func (h *Handlers) HandleResolveDispute(w http.ResponseWriter, r *http.Request) {
	ops := h.Ops
	if ops == nil || ops.Spanner == nil {
		writeErr(w, http.StatusServiceUnavailable, "spanner not wired on this process")
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if orderID == "" {
		writeErr(w, http.StatusBadRequest, "orderId required")
		return
	}
	var req resolveDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resolution := strings.ToUpper(strings.TrimSpace(req.Resolution))
	if resolution != "CONFIRM_DELIVERY" && resolution != "CONFIRM_CANCEL" {
		writeErr(w, http.StatusBadRequest, "resolution must be CONFIRM_DELIVERY or CONFIRM_CANCEL")
		return
	}

	ctx := r.Context()
	targetStatus := "AWAITING_PAYMENT"
	if resolution == "CONFIRM_CANCEL" {
		targetStatus = "CANCELLED"
	}

	_, err := ops.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Status", "SupplierId", "RetailerId"})
		if err != nil {
			return err
		}
		var currStatus, supplierID, retailerID string
		if err := row.Columns(&currStatus, &supplierID, &retailerID); err != nil {
			return err
		}
		if currStatus != "RECONCILIATION_REQUIRED" {
			return fmt.Errorf("order %s is in status %s, not RECONCILIATION_REQUIRED", orderID, currStatus)
		}

		mutation := spanner.UpdateMap("Orders", map[string]any{
			"OrderId":   orderID,
			"Status":    targetStatus,
			"UpdatedAt": spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":   orderID,
		"resolution": resolution,
		"status":     targetStatus,
		"resolved":   true,
	})
}

