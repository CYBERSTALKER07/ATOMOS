package bqexport

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const checkpointsTable = "BigQuerySinkCheckpoints"

// SpannerCheckpointStore persists monotonically advancing Kafka export watermarks
// in BigQuerySinkCheckpoints using StrongRead transactions (never unguarded single-read queries).
type SpannerCheckpointStore struct {
	client *spanner.Client
}

// NewSpannerCheckpointStore constructs a Spanner-backed CheckpointStore.
func NewSpannerCheckpointStore(client *spanner.Client) (*SpannerCheckpointStore, error) {
	if client == nil {
		return nil, errors.New("bqexport: nil spanner client")
	}
	return &SpannerCheckpointStore{client: client}, nil
}

// GetCheckpoint reads a single partition checkpoint using an explicit StrongRead transaction.
func (s *SpannerCheckpointStore) GetCheckpoint(ctx context.Context, sinkID, topic string, partition int64) (Checkpoint, bool, error) {
	if s == nil || s.client == nil {
		return Checkpoint{}, false, errors.New("bqexport: nil spanner checkpoint store")
	}
	sinkID = strings.TrimSpace(sinkID)
	topic = strings.TrimSpace(topic)
	if sinkID == "" || topic == "" {
		return Checkpoint{}, false, errors.New("bqexport: sink_id and topic are required")
	}

	ro := s.client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	defer ro.Close()

	row, err := ro.ReadRow(
		ctx,
		checkpointsTable,
		spanner.Key{sinkID, topic, partition},
		[]string{"SinkId", "Topic", "PartitionNum", "OffsetVal", "LastEventId", "RowsExported", "UpdatedAt"},
	)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return Checkpoint{}, false, nil
		}
		return Checkpoint{}, false, fmt.Errorf("read checkpoint (%s,%s,%d): %w", sinkID, topic, partition, err)
	}

	var cp Checkpoint
	var lastEvent spanner.NullString
	if err := row.Columns(&cp.SinkID, &cp.Topic, &cp.PartitionNum, &cp.OffsetVal, &lastEvent, &cp.RowsExported, &cp.UpdatedAt); err != nil {
		return Checkpoint{}, false, fmt.Errorf("scan checkpoint (%s,%s,%d): %w", sinkID, topic, partition, err)
	}
	if lastEvent.Valid {
		cp.LastEventID = lastEvent.StringVal
	}
	return cp, true, nil
}

// SaveCheckpoint advances the checkpoint monotonically inside a ReadWriteTransaction.
// Older or duplicate offsets never regress OffsetVal or double-count RowsExported.
func (s *SpannerCheckpointStore) SaveCheckpoint(ctx context.Context, cp Checkpoint) error {
	if s == nil || s.client == nil {
		return errors.New("bqexport: nil spanner checkpoint store")
	}
	cp.SinkID = strings.TrimSpace(cp.SinkID)
	cp.Topic = strings.TrimSpace(cp.Topic)
	if cp.SinkID == "" || cp.Topic == "" {
		return errors.New("bqexport: sink_id and topic are required")
	}
	if cp.OffsetVal < 0 {
		return fmt.Errorf("bqexport: invalid negative offset %d", cp.OffsetVal)
	}

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(
			ctx,
			checkpointsTable,
			spanner.Key{cp.SinkID, cp.Topic, cp.PartitionNum},
			[]string{"OffsetVal", "RowsExported"},
		)
		var existingRows int64
		if err == nil {
			var existingOffset int64
			if colErr := row.Columns(&existingOffset, &existingRows); colErr != nil {
				return colErr
			}
			if existingOffset >= cp.OffsetVal {
				// Monotonic guard: stale or replayed message never regresses watermark.
				return nil
			}
		} else if status.Code(err) != codes.NotFound {
			return err
		}

		totalRows := cp.RowsExported
		if cp.RowsDelta > 0 {
			totalRows = existingRows + cp.RowsDelta
		} else if totalRows < existingRows {
			totalRows = existingRows
		}

		var lastEvent any
		if trimmed := strings.TrimSpace(cp.LastEventID); trimmed != "" {
			lastEvent = trimmed
		}

		mut := spanner.InsertOrUpdateMap(checkpointsTable, map[string]any{
			"SinkId":       cp.SinkID,
			"Topic":        cp.Topic,
			"PartitionNum": cp.PartitionNum,
			"OffsetVal":    cp.OffsetVal,
			"LastEventId":  lastEvent,
			"RowsExported": totalRows,
			"UpdatedAt":    spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	if err != nil {
		return fmt.Errorf("save checkpoint (%s,%s,%d): %w", cp.SinkID, cp.Topic, cp.PartitionNum, err)
	}
	return nil
}

// ListCheckpoints returns all checkpoints for sinkID using StrongRead().
func (s *SpannerCheckpointStore) ListCheckpoints(ctx context.Context, sinkID string) ([]Checkpoint, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("bqexport: nil spanner checkpoint store")
	}
	sinkID = strings.TrimSpace(sinkID)
	if sinkID == "" {
		return nil, errors.New("bqexport: sink_id is required")
	}

	ro := s.client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	defer ro.Close()

	stmt := spanner.Statement{
		SQL: `SELECT SinkId, Topic, PartitionNum, OffsetVal, LastEventId, RowsExported, UpdatedAt
		      FROM BigQuerySinkCheckpoints
		      WHERE SinkId = @sink_id
		      ORDER BY Topic ASC, PartitionNum ASC`,
		Params: map[string]any{"sink_id": sinkID},
	}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()

	out := make([]Checkpoint, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list checkpoints (%s): %w", sinkID, err)
		}
		var cp Checkpoint
		var lastEvent spanner.NullString
		if err := row.Columns(&cp.SinkID, &cp.Topic, &cp.PartitionNum, &cp.OffsetVal, &lastEvent, &cp.RowsExported, &cp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan listed checkpoint (%s): %w", sinkID, err)
		}
		if lastEvent.Valid {
			cp.LastEventID = lastEvent.StringVal
		}
		out = append(out, cp)
	}
	return out, nil
}

// MemoryCheckpointStore is an in-memory CheckpointStore for unit tests and local memory fallback.
type MemoryCheckpointStore struct {
	mu    sync.RWMutex
	items map[string]Checkpoint
}

// NewMemoryCheckpointStore initializes an in-memory CheckpointStore.
func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{items: make(map[string]Checkpoint)}
}

func memCheckpointKey(sinkID, topic string, partition int64) string {
	return fmt.Sprintf("%s\x00%s\x00%d", sinkID, topic, partition)
}

func (m *MemoryCheckpointStore) GetCheckpoint(_ context.Context, sinkID, topic string, partition int64) (Checkpoint, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp, ok := m.items[memCheckpointKey(strings.TrimSpace(sinkID), strings.TrimSpace(topic), partition)]
	return cp, ok, nil
}

func (m *MemoryCheckpointStore) SaveCheckpoint(_ context.Context, cp Checkpoint) error {
	cp.SinkID = strings.TrimSpace(cp.SinkID)
	cp.Topic = strings.TrimSpace(cp.Topic)
	if cp.SinkID == "" || cp.Topic == "" {
		return errors.New("bqexport: sink_id and topic are required")
	}
	if cp.OffsetVal < 0 {
		return fmt.Errorf("bqexport: invalid negative offset %d", cp.OffsetVal)
	}
	k := memCheckpointKey(cp.SinkID, cp.Topic, cp.PartitionNum)
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.items[k]
	if exists && existing.OffsetVal >= cp.OffsetVal {
		return nil
	}
	if cp.RowsDelta > 0 {
		cp.RowsExported = existing.RowsExported + cp.RowsDelta
	} else if exists && cp.RowsExported < existing.RowsExported {
		cp.RowsExported = existing.RowsExported
	}
	cp.UpdatedAt = time.Now().UTC()
	m.items[k] = cp
	return nil
}

func (m *MemoryCheckpointStore) ListCheckpoints(_ context.Context, sinkID string) ([]Checkpoint, error) {
	sinkID = strings.TrimSpace(sinkID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Checkpoint, 0)
	for _, cp := range m.items {
		if cp.SinkID == sinkID {
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Topic != out[j].Topic {
			return out[i].Topic < out[j].Topic
		}
		return out[i].PartitionNum < out[j].PartitionNum
	})
	return out, nil
}
