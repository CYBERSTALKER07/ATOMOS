// Command backfill-outbox-supplier-id stamps OutboxEvents.SupplierId from payload
// for rows left NULL before write-path stamping (Gate 5 / Week 9 residual).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	limit := flag.Int("limit", 500, "max OutboxEvents rows to scan/update")
	timeout := flag.Duration("timeout", 10*time.Minute, "overall job timeout")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	n, err := outbox.BackfillSupplierID(ctx, client, *limit)
	if err != nil {
		slog.Error("backfill", "err", err)
		os.Exit(1)
	}
	slog.Info("outbox supplier backfill complete", "updated", n)
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
