package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	limit := flag.Int("limit", 100, "max manifests to process")
	dryRun := flag.Bool("dry-run", false, "list candidates without writing")
	statesCSV := flag.String("states", "SEALED,DISPATCHED,LOADING,DRAFT", "comma-separated manifest states")
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

	database := spannerDatabasePath(cfg)
	if database == "" {
		slog.Error("spanner database path is empty")
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, database, spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("new spanner client", "database", database, "err", err)
		os.Exit(1)
	}
	defer client.Close()

	store := manifest.NewStore(client)
	googleRoutes := routing.NewGoogleRoutesClient(cfg.GoogleMapsAPIKey, "", nil)
	osrmClient := routing.NewOSRMClient(cfg.RoutingOSRMURL, nil)
	store.SetGeometryBuilder(routing.NewGeometryBuilder(
		googleRoutes,
		osrmClient,
		routing.ParseRoutingProviderMode(cfg.RoutingProvider),
	))
	if googleRoutes != nil {
		slog.Info("Google Routes geometry enabled", "provider_mode", cfg.RoutingProvider)
	}
	if osrmClient != nil {
		slog.Info("OSRM routing enabled", "base_url", cfg.RoutingOSRMURL)
	}

	states := parseStates(*statesCSV)
	result, err := store.BackfillRouteGeometry(ctx, manifest.BackfillOptions{
		Limit:  *limit,
		DryRun: *dryRun,
		States: states,
	})
	if err != nil {
		slog.Error("route geometry backfill failed", "err", err)
		os.Exit(1)
	}

	payload, _ := json.Marshal(result)
	slog.Info("route geometry backfill complete", "result", string(payload))
}

func parseStates(csv string) []string {
	parts := strings.Split(csv, ",")
	states := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		states = append(states, part)
	}
	return states
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
	project := strings.TrimSpace(cfg.SpannerProject)
	instance := strings.TrimSpace(cfg.SpannerInstance)
	database := strings.TrimSpace(cfg.SpannerDatabase)
	if project == "" || instance == "" || database == "" {
		return ""
	}
	return "projects/" + project + "/instances/" + instance + "/databases/" + database
}
