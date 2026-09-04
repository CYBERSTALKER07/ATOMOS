package platformadmin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
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
