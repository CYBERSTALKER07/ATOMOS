package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	limit := flag.Int("limit", 500, "max orders to scan")
	dryRun := flag.Bool("dry-run", false, "list candidates without writing")
	timeout := flag.Duration("timeout", 10*time.Minute, "overall job timeout")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load bootstrap config", "err", err)
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("new spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	result, err := backfill(ctx, client, *limit, *dryRun)
	if err != nil {
		slog.Error("backfill failed", "err", err)
		os.Exit(1)
	}
	raw, _ := json.Marshal(result)
	slog.Info("order timeline backfill complete", "result", string(raw))
}

type backfillResult struct {
	Scanned   int `json:"scanned"`
	Backfilled int `json:"backfilled"`
	Skipped   int `json:"skipped"`
}

func backfill(ctx context.Context, client *spanner.Client, limit int, dryRun bool) (backfillResult, error) {
	out := backfillResult{}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, Status, CreatedAt, UpdatedAt
		      FROM Orders
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"lim": limit},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var mutations []*spanner.Mutation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out, err
		}
		var orderID, status string
		var createdAt, updatedAt time.Time
		if err := row.Columns(&orderID, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		out.Scanned++

		has, err := orderHasTimeline(ctx, client, orderID)
		if err != nil {
			return out, err
		}
		if has {
			out.Skipped++
			continue
		}

		eventKind := "STATUS_CHANGE"
		if order.Status(status) == order.StatusDelayed {
			eventKind = "DELAY"
		}

		mutations = append(mutations, spanner.InsertMap("OrderStatusTransitions", map[string]any{
			"OrderId":        orderID,
			"TransitionId":   uuid.NewString(),
			"PreviousStatus": "",
			"NewStatus":      status,
			"Reason":         "BACKFILL_FROM_ORDER_ROW",
			"ActorRole":      "SYSTEM",
			"ActorId":        "backfill-order-timeline",
			"EventKind":      eventKind,
			"MetadataJson":   []byte(`{"source":"ORDER_ROW_BACKFILL"}`),
			"CreatedAt":      coalesceTime(updatedAt, createdAt),
		}))
		out.Backfilled++
	}

	if dryRun || len(mutations) == 0 {
		return out, nil
	}
	const batch = 200
	for i := 0; i < len(mutations); i += batch {
		end := i + batch
		if end > len(mutations) {
			end = len(mutations)
		}
		if _, err := client.Apply(ctx, mutations[i:end]); err != nil {
			return out, fmt.Errorf("apply batch %d: %w", i/batch, err)
		}
	}
	return out, nil
}

func orderHasTimeline(ctx context.Context, client *spanner.Client, orderID string) (bool, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT TransitionId FROM OrderStatusTransitions WHERE OrderId = @oid LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	return err == nil, err
}

func coalesceTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}

func spannerClientOptions(cfg *bootstrap.Config) []option.ClientOption {
	if strings.TrimSpace(cfg.SpannerEmulatorHost) == "" {
		return nil
	}
	return []option.ClientOption{
		option.WithEndpoint(cfg.SpannerEmulatorHost),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)
}
