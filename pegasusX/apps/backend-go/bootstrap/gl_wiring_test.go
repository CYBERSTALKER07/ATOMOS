package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewApp_WiresGLServiceAndPostingRulesOnLiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner GL wiring test")
	}

	stubRuntimeConstructors(t)
	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return &fakeRedisAdapter{}, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string, _ kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(ctx context.Context, dsn string) (*spanner.Client, error) {
		c, err := spanner.NewClient(ctx, dsn,
			option.WithEndpoint(host),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
		if err != nil {
			return nil, err
		}
		if _, pErr := c.Single().ReadRow(ctx, "GlAccounts", spanner.Key{"__bs_probe__", "__bs_probe__"}, []string{"AccountCode"}); pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
			c.Close()
			return nil, pErr
		}
		return c, nil
	}
	cfg := testConfig()
	cfg.RunMode = RunModeAPI
	cfg.RequireInfraAdapters = false

	app, err := NewApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewApp(api + live Spanner): %v", err)
	}
	defer app.Close()

	if app.GL == nil {
		t.Fatal("expected app.GL to be initialized when Spanner client is present")
	}
	rules, err := app.GL.ListPostingRules(context.Background(), "UZ", "")
	if err != nil {
		t.Fatalf("app.GL.ListPostingRules(UZ): %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected seeded UZ posting rules on startup")
	}
}
