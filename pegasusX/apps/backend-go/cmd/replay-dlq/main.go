package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/segmentio/kafka-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	maxMsgs := flag.Int("max", 100, "Maximum number of DLQ messages to replay")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	if len(brokers) == 0 || brokers[0] == "" {
		slog.Error("KafkaBrokers not configured")
		os.Exit(1)
	}
	dlqTopic := cfg.KafkaTopicMainDLQ
	mainTopic := cfg.KafkaTopicMain
	if dlqTopic == "" || mainTopic == "" {
		slog.Error("Kafka DLQ or Main topic not configured")
		os.Exit(1)
	}

	slog.Info("Starting DLQ replay", "brokers", brokers, "dlq_topic", dlqTopic, "main_topic", mainTopic, "max", *maxMsgs)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "dlq-replayer",
		Topic:    dlqTopic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  mainTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: false,
	}
	defer writer.Close()

	var replayed int
	for i := 0; i < *maxMsgs; i++ {
		// Try to read a message, timeout after 5 seconds to allow graceful exit if empty
		readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
		msg, err := reader.FetchMessage(readCtx)
		readCancel()
		if err != nil {
			if err == context.DeadlineExceeded {
				slog.Info("No more messages in DLQ or reached timeout waiting")
				break
			}
			slog.Error("Error fetching message from DLQ", "err", err)
			break
		}

		slog.Info("Replaying message", "offset", msg.Offset, "key", string(msg.Key), "time", msg.Time)

		err = writer.WriteMessages(ctx, kafka.Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: msg.Headers,
		})
		if err != nil {
			slog.Error("Error writing message to main topic", "err", err)
			break
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Error committing DLQ message", "err", err)
			break
		}
		replayed++
	}

	slog.Info("DLQ replay finished", "replayed_count", replayed)
}
