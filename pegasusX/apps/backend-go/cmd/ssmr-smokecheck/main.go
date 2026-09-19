package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/schemadrift"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/ssmr-smokecheck [spanner|kafka|spatial|e2e|tenant|parent-order|global-products|gap-closure|lifecycle-vertical|fiscal|payment|shop-closed|manifest-seal|claims|negotiation-isolation|loadtokens|planning-baseline-seed]")
		os.Exit(1)
	}

	check := strings.TrimSpace(os.Args[1])
	logOut := os.Stdout
	if check == "loadtokens" {
		// stdout is reserved for shell `export` lines consumed by load_cert.sh
		logOut = os.Stderr
	}
	logger := slog.New(slog.NewJSONHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	timeout := 30 * time.Second
	switch check {
	case "loadtokens":
		timeout = loadTokensTimeout()
	case "e2e":
		timeout = e2eTimeout()
	case "tenant":
		timeout = 2 * time.Minute
	case "parent-order":
		timeout = 3 * time.Minute
	case "global-products":
		timeout = 2 * time.Minute
	case "gap-closure":
		timeout = 3 * time.Minute
	case "lifecycle-vertical":
		// Focused spine is shorter than full ecosystem e2e.
		timeout = e2eTimeout()
		if timeout > 2*time.Minute {
			timeout = 2 * time.Minute
		}
	case "fiscal":
		timeout = e2eTimeout()
		if timeout < 4*time.Minute {
			timeout = 4 * time.Minute
		}
	case "claims":
		timeout = e2eTimeout()
		if timeout < 4*time.Minute {
			timeout = 4 * time.Minute
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	var checkErr error
	switch check {
	case "spanner":
		checkErr = runSpannerCheck(ctx, cfg)
	case "kafka":
		checkErr = runKafkaCheck(ctx, cfg)
	case "spatial":
		checkErr = runSpatialCheck(ctx, cfg)
	case "e2e":
		checkErr = runE2ECheck(ctx, cfg)
	case "tenant":
		checkErr = runTenantSmokeCheck(ctx, cfg)
	case "parent-order":
		checkErr = runParentOrderSmokeCheck(ctx, cfg)
	case "global-products":
		checkErr = runGlobalProductsSmokeCheck(ctx, cfg)
	case "gap-closure":
		checkErr = runGapClosureSmokeCheck(ctx, cfg)
	case "lifecycle-vertical":
		checkErr = runLifecycleVerticalE2E(ctx, cfg)
	case "fiscal":
		checkErr = runFiscalE2E(ctx, cfg)
	case "payment":
		checkErr = runPaymentSmokeCheck(ctx, cfg)
	case "shop-closed":
		checkErr = runShopClosedSmokeCheck(ctx, cfg)
	case "manifest-seal":
		checkErr = runManifestSealSmokeCheck(ctx, cfg)
	case "claims":
		checkErr = runClaimsE2E(ctx, cfg)
	case "negotiation-isolation":
		checkErr = runNegotiationIsolationCheck(ctx, cfg)
	case "loadtokens":
		checkErr = runLoadTokens(ctx, cfg)
	case "planning-baseline-seed":
		checkErr = runPlanningBaselineSeed(ctx, cfg)
	default:
		checkErr = fmt.Errorf("unknown check %q", check)
	}

	if checkErr != nil {
		slog.Error("ssmr smokecheck failed", "check", check, "err", checkErr)
		os.Exit(1)
	}

	if check != "loadtokens" {
		slog.Info("ssmr smokecheck passed", "check", check)
	}
}

func runSpannerCheck(ctx context.Context, cfg *bootstrap.Config) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	if err := assertSeedSupplier(ctx, client, cfg); err != nil {
		return err
	}
	if err := assertRetailerSchema(ctx, client); err != nil {
		return err
	}
	if err := schemadrift.AssertShopClosedSchema(ctx, client); err != nil {
		return err
	}

	return nil
}

func assertSeedSupplier(ctx context.Context, client *spanner.Client, cfg *bootstrap.Config) error {
	// Identity is DefaultSupplierID. EnsureDemoScopeLinks rewrites Name to
	// "SSMR Smoke Supplier" after cmd/setup upserts SEED_SUPPLIER_NAME.
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, CountryCode, Currency
FROM Suppliers
WHERE SupplierId = @id
LIMIT 1`,
		Params: map[string]any{"id": seed.DefaultSupplierID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return fmt.Errorf("seed supplier %q missing", seed.DefaultSupplierID)
		}
		return fmt.Errorf("query seeded supplier: %w", err)
	}

	var supplierID string
	var countryCode string
	var currency string
	if err := row.Columns(&supplierID, &countryCode, &currency); err != nil {
		return fmt.Errorf("decode seeded supplier: %w", err)
	}
	if strings.TrimSpace(supplierID) == "" {
		return fmt.Errorf("seed supplier returned empty SupplierId")
	}
	if countryCode != cfg.SeedSupplierCountry {
		return fmt.Errorf("seed supplier country mismatch: got %q want %q", countryCode, cfg.SeedSupplierCountry)
	}
	if currency != cfg.SeedSupplierCurrency {
		return fmt.Errorf("seed supplier currency mismatch: got %q want %q", currency, cfg.SeedSupplierCurrency)
	}

	return nil
}

func assertRetailerSchema(ctx context.Context, client *spanner.Client) error {
	stmt := spanner.Statement{
		SQL: `SELECT COLUMN_NAME
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_NAME = @table_name
  AND COLUMN_NAME IN UNNEST(@required_columns)`,
		Params: map[string]any{
			"table_name":       "Retailers",
			"required_columns": []string{"RetailerId", "H3Cell"},
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	found := map[string]bool{}
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return fmt.Errorf("query retailer schema: %w", err)
		}

		var columnName string
		if err := row.Columns(&columnName); err != nil {
			return fmt.Errorf("decode retailer schema row: %w", err)
		}
		found[columnName] = true
	}

	if !found["RetailerId"] || !found["H3Cell"] {
		return fmt.Errorf("retailers schema missing required columns: found=%v", found)
	}

	return nil
}

func runKafkaCheck(ctx context.Context, cfg *bootstrap.Config) error {
	broker := firstBroker(cfg.KafkaBrokers)
	if broker == "" {
		return fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}

	if err := assertKafkaTopicIsolation(ctx, broker, expectedTopics(cfg)); err != nil {
		return err
	}
	if err := assertKafkaRoundTrip(ctx, broker, cfg.KafkaTopicMain); err != nil {
		return err
	}

	return nil
}

func runSpatialCheck(ctx context.Context, cfg *bootstrap.Config) error {
	redisBackend, err := cache.NewRedisBackend(cache.RedisConfig{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPassword,
		PoolSize:        cfg.RedisPoolSize,
		MaxRetries:      cfg.RedisMaxRetries,
		TLSEnabled:      cfg.RedisTLSEnabled,
		MinIdleConns:    2,
		MaxIdleTime:     time.Minute,
		DialTimeout:     time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		MinRetryBackoff: time.Millisecond * 8,
		MaxRetryBackoff: time.Millisecond * 512,
	})
	if err != nil {
		return fmt.Errorf("new redis backend: %w", err)
	}
	defer redisBackend.Close()

	if err := redisBackend.Ping(ctx); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}

	proximity := retailer.NewRetailerProximityService(redisBackend, retailer.RetailerProximityConfig{
		Resolution: cfg.DeliveryZoneResolution,
		Log:        slog.Default(),
	})

	ready, err := proximity.PerimeterReady(ctx)
	if err != nil {
		return fmt.Errorf("perimeter ready check: %w", err)
	}
	if !ready {
		return fmt.Errorf("delivery perimeter key %q missing", retailer.DeliveryPerimeterKey)
	}

	sampleCell, err := proximity.CellForCoordinate(cfg.DeliveryZoneCenterLat, cfg.DeliveryZoneCenterLng)
	if err != nil {
		return fmt.Errorf("derive sample h3 cell: %w", err)
	}
	if err := proximity.IsRetailerInZone(ctx, sampleCell); err != nil {
		return fmt.Errorf("sample cell membership check failed: %w", err)
	}

	return nil
}

func assertKafkaTopicIsolation(ctx context.Context, broker string, expected []string) error {
	conn, err := kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		return fmt.Errorf("dial kafka broker %s: %w", broker, err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("read kafka partitions: %w", err)
	}

	expectedSet := map[string]struct{}{}
	for _, topic := range expected {
		expectedSet[topic] = struct{}{}
	}

	seen := map[string]struct{}{}
	for _, partition := range partitions {
		if strings.HasPrefix(partition.Topic, "__") {
			continue
		}
		seen[partition.Topic] = struct{}{}
	}

	missing := make([]string, 0)
	for _, topic := range expected {
		if _, ok := seen[topic]; !ok {
			missing = append(missing, topic)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("isolated kafka topics missing: %s", strings.Join(missing, ", "))
	}

	unexpected := make([]string, 0)
	for topic := range seen {
		if _, ok := expectedSet[topic]; !ok {
			unexpected = append(unexpected, topic)
		}
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		return fmt.Errorf("unexpected non-internal topics on isolated broker: %s", strings.Join(unexpected, ", "))
	}

	return nil
}

func assertKafkaRoundTrip(ctx context.Context, broker string, topic string) error {
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("kafka topic main is empty")
	}

	conn, err := kafka.DialLeader(ctx, "tcp", broker, topic, 0)
	if err != nil {
		return fmt.Errorf("dial leader for topic %s: %w", topic, err)
	}
	defer conn.Close()

	_, lastOffset, err := conn.ReadOffsets()
	if err != nil {
		return fmt.Errorf("read offsets for topic %s: %w", topic, err)
	}

	payload := fmt.Sprintf("ssmr-smoke-%d", time.Now().UTC().UnixNano())
	if _, err := conn.WriteMessages(kafka.Message{
		Key:   []byte("ssmr-smoke"),
		Value: []byte(payload),
		Time:  time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("write kafka smoke message: %w", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{broker},
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  1_048_576,
	})
	defer reader.Close()

	if err := reader.SetOffset(lastOffset); err != nil {
		return fmt.Errorf("set kafka reader offset: %w", err)
	}

	readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	message, err := reader.ReadMessage(readCtx)
	if err != nil {
		return fmt.Errorf("read kafka smoke message: %w", err)
	}
	if string(message.Value) != payload {
		return fmt.Errorf("kafka smoke payload mismatch: got %q want %q", string(message.Value), payload)
	}

	return nil
}

func expectedTopics(cfg *bootstrap.Config) []string {
	topics := []string{
		cfg.KafkaTopicMain,
		cfg.KafkaTopicMainDLQ,
		envOr("KAFKA_TOPIC_ORDERS", "pegasusx-orders"),
		envOr("KAFKA_TOPIC_DISPATCH", "pegasusx-dispatch"),
		envOr("KAFKA_TOPIC_SPATIAL", "ssmr.events.spatial"),
		envOr("KAFKA_TOPIC_REALTIME", "ssmr.events.realtime"),
		envOr("KAFKA_TOPIC_WEBHOOKS", "ssmr.events.webhooks"),
		envOr("KAFKA_TOPIC_FREEZE_LOCKS", "pegasusx-freeze-locks"),
		envOr("KAFKA_TOPIC_INVENTORY_IMPORT", events.TopicInventoryImportEvents),
		envOr("KAFKA_TOPIC_DEMAND", events.TopicDemand),
		envOr("KAFKA_TOPIC_EXCEPTIONS", events.TopicExceptions),
		events.TopicPlanningSignalIngest,
		events.TopicPlanningForecastRequest,
		events.TopicPlanningForecastResult,
	}
	out := make([]string, 0, len(topics))
	for _, topic := range topics {
		if strings.TrimSpace(topic) != "" {
			out = append(out, topic)
		}
	}
	return out
}

func firstBroker(brokersCSV string) string {
	for _, broker := range strings.Split(brokersCSV, ",") {
		trimmed := strings.TrimSpace(broker)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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
	return fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject,
		cfg.SpannerInstance,
		cfg.SpannerDatabase,
	)
}

func envOr(key string, fallback string) string {
	if strings.HasPrefix(key, "SSMR_") {
		sandboxKey := "SANDBOX_" + strings.TrimPrefix(key, "SSMR_")
		if v := strings.TrimSpace(os.Getenv(sandboxKey)); v != "" {
			return v
		}
	}
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	if auth.IsSandbox() && sandboxIdentityKey(key) {
		return ""
	}
	return fallback
}

func sandboxIdentityKey(key string) bool {
	switch key {
	case "SSMR_SMOKE_DRIVER_ID", "SSMR_SMOKE_WAREHOUSE_ID",
		"SSMR_SMOKE_SUPPLIER_PASSWORD", "SSMR_SMOKE_SUPPLIER_PHONE",
		"SSMR_SMOKE_RETAILER_PHONE", "SSMR_SMOKE_SUPPLIER_ID":
		return true
	default:
		return false
	}
}
