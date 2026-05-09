import re

with open('pegasus/apps/backend-go/outbox/relay.go', 'r') as f:
    content = f.read()

import_block = """import (
        "context"
        "encoding/json"
        "hash/fnv"
        "log/slog"
        "runtime"
        "sync"
        "time"

        "cloud.google.com/go/spanner"
        goKafka "github.com/segmentio/kafka-go"
        "google.golang.org/api/iterator"

        "backend-go/telemetry"
)"""

content = re.sub(r'import \([\s\S]*?"google\.golang\.org/api/iterator"\n\)', import_block, content)

run_shard_block = """func (r *Relay) runShard(ctx context.Context, shardIdx int) {
        for {
                select {
                case <-ctx.Done():
                        return
                case job := <-r.shardChs[shardIdx]:
                        err := r.publish(ctx, job.topic, job.aggregateID, job.eventType, job.traceID, job.payload)
                        if err != nil {
                                slog.Error("outbox.relay.publish",
                                        "shard", shardIdx,
                                        "topic", job.topic,
                                        "event_id", job.eventID,
                                        "aggregate_id", job.aggregateID,
                                        "err", err)
                                if r.onFailure != nil {
                                        r.onFailure(job.eventID, job.aggregateID, job.topic, err)
                                }
                        } else {
                                // Extract supplier_id for telemetry metric
                                var partial struct {
                                        SupplierID string `json:"supplier_id"`
                                }
                                supplierID := "unknown"
                                if unmarshalErr := json.Unmarshal(job.payload, &partial); unmarshalErr == nil && partial.SupplierID != "" {
                                        supplierID = partial.SupplierID
                                }
                                telemetry.KafkaEventsTotal.WithLabelValues(supplierID, job.topic).Inc()
                        }
                        job.resultCh <- shardResult{eventID: job.eventID, err: err}
                }
        }
}"""

content = re.sub(r'func \(r \*Relay\) runShard[\s\S]*?^\}', run_shard_block, content, flags=re.MULTILINE)

with open('pegasus/apps/backend-go/outbox/relay.go', 'w') as f:
    f.write(content)
