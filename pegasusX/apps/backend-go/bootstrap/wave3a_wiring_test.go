package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewApp_WiresAllWave3AModulesAndBridges_LiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Wave 3A wiring test")
	}

	t.Cleanup(func() {
		gl.SetPeriodLockChecker(nil)
		stocklots.SetPickValuationHook(nil)
		stocklots.SetAdjustmentValuationHook(nil)
	})

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
		ro := c.ReadOnlyTransaction()
		_, pErr := ro.ReadRow(ctx, "Periods", spanner.Key{"__bs_w3a__", "2026-01"}, []string{"Status"})
		ro.Close()
		if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
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
		t.Fatalf("NewApp(api + live Spanner Wave 3A): %v", err)
	}
	defer app.Close()

	if app.Periods == nil || app.Periods.Repo() == nil {
		t.Fatalf("expected app.Periods wired with non-nil repository")
	}
	if app.Approvals == nil || app.Approvals.Repo() == nil {
		t.Fatalf("expected app.Approvals wired with non-nil repository")
	}
	if app.Valuation == nil || app.Valuation.Repo() == nil {
		t.Fatalf("expected app.Valuation wired with non-nil repository")
	}
	if app.ESF == nil || app.ESF.Repo() == nil {
		t.Fatalf("expected app.ESF wired with non-nil repository")
	}
	if app.FeatureFlagHandlers == nil || app.FeatureFlagHandlers.ApprovalBridge == nil {
		t.Fatalf("expected app.FeatureFlagHandlers.ApprovalBridge wired")
	}
}
