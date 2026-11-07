package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
)

// SpannerDLQUpdater updates Spanner job records and dead letter tables when messages are routed to DLQ.
type SpannerDLQUpdater struct {
	client *spanner.Client
	log    *slog.Logger
}

// NewSpannerDLQUpdater constructs a new SpannerDLQUpdater.
func NewSpannerDLQUpdater(client *spanner.Client, log *slog.Logger) *SpannerDLQUpdater {
	if log == nil {
		log = slog.Default()
	}
	return &SpannerDLQUpdater{
		client: client,
		log:    log,
	}
}

// Hook returns a DLQHook suitable for ConsumerDeps.OnDLQ.
func (u *SpannerDLQUpdater) Hook() DLQHook {
	return u.HandleDLQ
}

// JobDLQMetadata holds extracted job metadata from Kafka message.
type JobDLQMetadata struct {
	JobID         string
	AggregateID   string
	SupplierID    string
	AggregateType string
	SessionID     string
	TenantID      string
	TenantType    string
}

func extractJobMetadata(msg kafka.Message) JobDLQMetadata {
	var meta JobDLQMetadata

	// 1. Inspect headers
	for _, h := range msg.Headers {
		k := strings.ToLower(strings.TrimSpace(h.Key))
		v := strings.TrimSpace(string(h.Value))
		if v == "" {
			continue
		}
		switch k {
		case "job_id", "jobid", "job-id":
			if meta.JobID == "" {
				meta.JobID = v
			}
		case "aggregate_id", "aggregateid", "aggregate-id":
			if meta.AggregateID == "" {
				meta.AggregateID = v
			}
		case "supplier_id", "supplierid", "supplier-id":
			if meta.SupplierID == "" {
				meta.SupplierID = v
			}
		case "aggregate_type", "aggregatetype", "aggregate-type", "event_type", "type":
			if meta.AggregateType == "" {
				meta.AggregateType = v
			}
		case "session_id", "sessionid", "session-id":
			if meta.SessionID == "" {
				meta.SessionID = v
			}
		case "tenant_id", "tenantid", "tenant-id":
			if meta.TenantID == "" {
				meta.TenantID = v
			}
		case "tenant_type", "tenanttype", "tenant-type":
			if meta.TenantType == "" {
				meta.TenantType = v
			}
		}
	}

	// 2. Inspect payload JSON if present
	if len(msg.Value) > 0 {
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(msg.Value, &payloadMap); err == nil && payloadMap != nil {
			for k, v := range payloadMap {
				strVal := ""
				if s, ok := v.(string); ok {
					strVal = strings.TrimSpace(s)
				}
				if strVal == "" {
					continue
				}
				lk := strings.ToLower(strings.TrimSpace(k))
				switch lk {
				case "job_id", "jobid", "job-id", "id":
					if meta.JobID == "" {
						meta.JobID = strVal
					}
				case "aggregate_id", "aggregateid", "aggregate-id":
					if meta.AggregateID == "" {
						meta.AggregateID = strVal
					}
				case "supplier_id", "supplierid", "supplier-id":
					if meta.SupplierID == "" {
						meta.SupplierID = strVal
					}
				case "aggregate_type", "aggregatetype", "aggregate-type", "event_type", "type", "resource":
					if meta.AggregateType == "" {
						meta.AggregateType = strVal
					}
				case "session_id", "sessionid", "session-id":
					if meta.SessionID == "" {
						meta.SessionID = strVal
					}
				case "tenant_id", "tenantid", "tenant-id":
					if meta.TenantID == "" {
						meta.TenantID = strVal
					}
				case "tenant_type", "tenanttype", "tenant-type":
					if meta.TenantType == "" {
						meta.TenantType = strVal
					}
				}
			}
		}
	}

	// 3. Fallbacks from key
	keyStr := strings.TrimSpace(string(msg.Key))
	if meta.JobID == "" && keyStr != "" {
		meta.JobID = keyStr
	}
	if meta.AggregateID == "" && keyStr != "" {
		meta.AggregateID = keyStr
	}
	if meta.JobID == "" && meta.AggregateID != "" {
		meta.JobID = meta.AggregateID
	}
	if meta.AggregateID == "" && meta.JobID != "" {
		meta.AggregateID = meta.JobID
	}
	if meta.AggregateType == "" {
		meta.AggregateType = msg.Topic
		if meta.AggregateType == "" {
			meta.AggregateType = "UNKNOWN"
		}
	}

	return meta
}

// HandleDLQ executes Spanner mutations transitioning job records to FAILED and archiving to OutboxDeadLetters.
func (u *SpannerDLQUpdater) HandleDLQ(ctx context.Context, msg kafka.Message, reason error) error {
	if u == nil || u.client == nil {
		return nil
	}

	reasonStr := "dlq failure"
	if reason != nil && strings.TrimSpace(reason.Error()) != "" {
		reasonStr = reason.Error()
	}
	if len(reasonStr) > 1024 {
		reasonStr = reasonStr[:1024]
	}

	meta := extractJobMetadata(msg)

	u.log.InfoContext(ctx, "spanner dlq updater handling message",
		"job_id", meta.JobID,
		"aggregate_id", meta.AggregateID,
		"supplier_id", meta.SupplierID,
		"aggregate_type", meta.AggregateType,
		"topic", msg.Topic,
		"reason", reasonStr,
	)

	_, err := u.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var mutations []*spanner.Mutation
		now := time.Now().UTC()

		// 1. OptimizationJobs
		if meta.JobID != "" {
			row, err := txn.ReadRow(ctx, "OptimizationJobs", spanner.Key{meta.JobID}, []string{"JobId"})
			if err == nil && row != nil {
				mutations = append(mutations, spanner.UpdateMap("OptimizationJobs", map[string]interface{}{
					"JobId":     meta.JobID,
					"Status":    "FAILED",
					"UpdatedAt": spanner.CommitTimestamp,
				}))
			} else if err != nil && spanner.ErrCode(err) != codes.NotFound {
				u.log.WarnContext(ctx, "reading OptimizationJobs failed in dlq hook", "job_id", meta.JobID, "err", err)
			}
		}

		// 2. PartnerExportJobs
		if meta.JobID != "" {
			row, err := txn.ReadRow(ctx, "PartnerExportJobs", spanner.Key{meta.JobID}, []string{"JobId"})
			if err == nil && row != nil {
				mutations = append(mutations, spanner.UpdateMap("PartnerExportJobs", map[string]interface{}{
					"JobId":      meta.JobID,
					"Status":     "FAILED",
					"Error":      reasonStr,
					"FinishedAt": spanner.CommitTimestamp,
				}))
			} else if err != nil && spanner.ErrCode(err) != codes.NotFound {
				u.log.WarnContext(ctx, "reading PartnerExportJobs failed in dlq hook", "job_id", meta.JobID, "err", err)
			}
		}

		// 3. SupplierImportSessions
		sessionID := meta.SessionID
		if sessionID == "" {
			sessionID = meta.JobID
		}
		if meta.SupplierID != "" && sessionID != "" {
			row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{meta.SupplierID, sessionID}, []string{"session_id"})
			if err == nil && row != nil {
				errSummary := fmt.Sprintf(`{"error": %q, "failed_at": %q}`, reasonStr, now.Format(time.RFC3339))
				mutations = append(mutations, spanner.UpdateMap("SupplierImportSessions", map[string]interface{}{
					"supplier_id":   meta.SupplierID,
					"session_id":    sessionID,
					"status":        "FAILED",
					"error_summary": spanner.NullJSON{Value: json.RawMessage(errSummary), Valid: true},
					"updated_at":    spanner.CommitTimestamp,
				}))
			} else if err != nil && spanner.ErrCode(err) != codes.NotFound {
				u.log.WarnContext(ctx, "reading SupplierImportSessions failed in dlq hook", "session_id", sessionID, "err", err)
			}
		}

		// 4. OutboxDeadLetters
		eventID := string(msg.Key)
		if strings.TrimSpace(eventID) == "" {
			eventID = fmt.Sprintf("dlq-%s-%d-%d", msg.Topic, msg.Partition, msg.Offset)
		}
		createdAt := msg.Time.UTC()
		if createdAt.IsZero() {
			createdAt = now
		}
		deadLetterCols := map[string]interface{}{
			"EventId":        eventID,
			"AggregateType":  meta.AggregateType,
			"AggregateId":    meta.AggregateID,
			"TopicName":      msg.Topic,
			"Payload":        msg.Value,
			"CreatedAt":      createdAt,
			"DeadLetteredAt": spanner.CommitTimestamp,
			"Attempts":       int64(4),
			"LastError":      reasonStr,
		}
		if meta.SupplierID != "" {
			deadLetterCols["SupplierId"] = meta.SupplierID
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxDeadLetters", deadLetterCols))

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		u.log.ErrorContext(ctx, "spanner dlq updater transaction failed", "err", err)
		return fmt.Errorf("spanner dlq updater: %w", err)
	}
	return nil
}
