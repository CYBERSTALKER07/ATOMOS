package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/kafka/workerpool"
	segmentkafka "github.com/segmentio/kafka-go"
)

type fakeDLQWriter struct {
	writeErr error
	messages []segmentkafka.Message
	closed   bool
	onWrite  func()
}

func (w *fakeDLQWriter) WriteMessages(_ context.Context, msgs ...segmentkafka.Message) error {
	if w.onWrite != nil {
		w.onWrite()
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	w.messages = append(w.messages, msgs...)
	return nil
}

func (w *fakeDLQWriter) Close() error {
	w.closed = true
	return nil
}

type fakeConsumerReader struct {
	message      segmentkafka.Message
	returnMsg    bool
	commits      int
	stopAfterOne bool
	stopped      bool
}

func (r *fakeConsumerReader) FetchMessage(ctx context.Context) (segmentkafka.Message, error) {
	if r.returnMsg {
		r.returnMsg = false
		return r.message, nil
	}
	if r.stopAfterOne || r.stopped {
		return segmentkafka.Message{}, context.Canceled
	}
	<-ctx.Done()
	return segmentkafka.Message{}, ctx.Err()
}

func (r *fakeConsumerReader) CommitMessages(_ context.Context, _ ...segmentkafka.Message) error {
	r.commits++
	return nil
}

func (r *fakeConsumerReader) Close() error { return nil }

func TestRetryBackoffWithJitter_AddsDeterministicJitter(t *testing.T) {
	original := retryJitterInt63n
	retryJitterInt63n = func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		return n - 1
	}
	t.Cleanup(func() {
		retryJitterInt63n = original
	})

	got := retryBackoffWithJitter(2)
	want := 200000000 + 99999999
	if got.Nanoseconds() != int64(want) {
		t.Fatalf("backoff = %d, want %d", got.Nanoseconds(), want)
	}
}

func TestSendToDLQ_AddsFailureContextHeaders(t *testing.T) {
	writer := &fakeDLQWriter{}
	consumer := &Consumer{deps: ConsumerDeps{Topic: "orders", DLQWriter: writer}}
	msg := segmentkafka.Message{Key: []byte("order-1"), Value: []byte("payload"), Partition: 4, Offset: 12}

	if err := consumer.sendToDLQ(context.Background(), msg, errors.New("boom")); err != nil {
		t.Fatalf("sendToDLQ error: %v", err)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("dlq messages = %d, want 1", len(writer.messages))
	}
	headers := map[string]string{}
	for _, header := range writer.messages[0].Headers {
		headers[header.Key] = string(header.Value)
	}
	if headers["dlq_reason"] != "boom" {
		t.Fatalf("dlq_reason = %q, want boom", headers["dlq_reason"])
	}
	if headers["original_topic"] != "orders" {
		t.Fatalf("original_topic = %q, want orders", headers["original_topic"])
	}
	if headers["original_partition"] != "4" {
		t.Fatalf("original_partition = %q, want 4", headers["original_partition"])
	}
	if headers["original_offset"] != "12" {
		t.Fatalf("original_offset = %q, want 12", headers["original_offset"])
	}
}

func TestDispatch_DoesNotCommitWhenDLQWriteFails(t *testing.T) {
	reader := &fakeConsumerReader{
		message:   segmentkafka.Message{Partition: 1, Offset: 9, Time: nowForConsumerTest()},
		returnMsg: true,
	}
	writer := &fakeDLQWriter{writeErr: errors.New("dlq unavailable")}
	writer.onWrite = func() {
		reader.stopAfterOne = true
	}
	consumer := &Consumer{
		reader: reader,
		deps: ConsumerDeps{
			Topic:       "orders",
			Handler:     func(context.Context, segmentkafka.Message) error { return errors.New("handler failed") },
			DLQWriter:   writer,
			MaxAttempts: 1,
		},
	}

	if err := consumer.dispatch(context.Background(), reader.message); !errors.Is(err, workerpool.ErrSkipCommit) {
		t.Fatalf("dispatch err = %v, want ErrSkipCommit", err)
	}

	if reader.commits != 0 {
		t.Fatalf("commits = %d, want 0", reader.commits)
	}
}

func TestDispatch_CommitsAfterSuccessfulDLQWrite(t *testing.T) {
	reader := &fakeConsumerReader{
		message:   segmentkafka.Message{Partition: 2, Offset: 14, Time: nowForConsumerTest()},
		returnMsg: true,
	}
	writer := &fakeDLQWriter{}
	writer.onWrite = func() {
		reader.stopAfterOne = true
	}
	consumer := &Consumer{
		reader: reader,
		deps: ConsumerDeps{
			Topic:       "orders",
			Handler:     func(context.Context, segmentkafka.Message) error { return errors.New("handler failed") },
			DLQWriter:   writer,
			MaxAttempts: 1,
		},
	}

	if err := consumer.dispatch(context.Background(), reader.message); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := reader.CommitMessages(context.Background(), reader.message); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if reader.commits != 1 {
		t.Fatalf("commits = %d, want 1", reader.commits)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("dlq messages = %d, want 1", len(writer.messages))
	}
}

func nowForConsumerTest() time.Time {
	return time.Now().UTC()
}
