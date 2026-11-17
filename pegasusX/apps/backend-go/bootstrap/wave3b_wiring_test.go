package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func TestNewApp_WiresAllWave3BModulesAndBridges_LiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Wave 3B wiring test")
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
		_, pErr := ro.ReadRow(ctx, "Proposals", spanner.Key{"__bs_w3b__", "__probe__"}, []string{"Status"})
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
		t.Fatalf("NewApp(api + live Spanner Wave 3B): %v", err)
	}
	defer app.Close()

	if app.ForecastV2 == nil || app.ForecastV2.Repo() == nil {
		t.Fatalf("expected app.ForecastV2 wired with non-nil repository")
	}
	if app.ReplenishmentSuggestions == nil {
		t.Fatalf("expected app.ReplenishmentSuggestions wired")
	}
	if app.CreditService == nil {
		t.Fatalf("expected app.CreditService wired")
	}
	if app.ROI == nil || app.ROI.Repo() == nil {
		t.Fatalf("expected app.ROI wired with non-nil repository")
	}
	if app.Proposals == nil || app.Proposals.Repo() == nil {
		t.Fatalf("expected app.Proposals wired with non-nil repository")
	}
	if b := NewCreditLimitApprovalBridge(app.Approvals); b == nil {
		t.Fatalf("expected non-nil NewCreditLimitApprovalBridge when app.Approvals is wired")
	}
}
