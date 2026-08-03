// schema-drift fails closed when required Spanner objects are absent.
//
// Usage:
//
//	SPANNER_PROJECT=… SPANNER_INSTANCE=… SPANNER_DATABASE=… go run ./cmd/schema-drift
//	# emulator:
//	SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/schema-drift
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/schemadrift"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	dbPath := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase,
	)
	client, err := spanner.NewClient(ctx, dbPath, spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "database", dbPath, "err", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := schemadrift.AssertShopClosedSchema(ctx, client); err != nil {
		slog.Error("schema-drift-FAIL", "database", dbPath, "err", err)
		os.Exit(1)
	}

	slog.Info("schema-drift-ok",
		"database", dbPath,
		"check", "shop_closed_proximity_partial",
	)
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
