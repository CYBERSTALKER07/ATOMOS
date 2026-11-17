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

func TestNewApp_WiresReconServiceOnLiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Recon wiring test")
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
		if _, pErr := c.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReconciliationRuns", spanner.Key{"__bs_recon__", "__bs_run__"}, []string{"RunId"}); pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
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

	if app.Recon == nil || app.Recon.Repo() == nil {
		t.Fatalf("expected app.Recon wired with non-nil repository on live Spanner")
	}
}
