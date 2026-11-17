package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewApp_WiresAuditIAMAndFoundryHook_LiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Audit/IAM wiring test")
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
		if _, pErr := c.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "AuditEvents", spanner.Key{"__bs_aud__", "2026-01-01", "__bs_ev__"}, []string{"EventId"}); pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
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

	if app.Audit == nil || app.Audit.Repo() == nil {
		t.Fatalf("expected app.Audit wired on live Spanner")
	}
	if app.IAM == nil || app.IAM.Repo() == nil {
		t.Fatalf("expected app.IAM wired on live Spanner")
	}
	if app.FoundryEngine == nil {
		t.Fatalf("expected app.FoundryEngine wired")
	}

	supID := "sup-bs-aud-" + uuid.NewString()[:8]
	today := time.Now().UTC().Format("2006-01-02")
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = app.Spanner.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("AuditEvents", spanner.KeyRange{
				Start: spanner.Key{supID, today},
				End:   spanner.Key{supID, today},
				Kind:  spanner.ClosedClosed,
			}),
		})
	})

	app.FoundryEngine.RegisterPrecondition("REORDER_APPROVE_TEST", func(_ context.Context, _ foundry.ActionExecutionRequest) foundry.PreconditionResult {
		return foundry.PreconditionResult{RuleName: "test_ok", Passed: true}
	})

	rec, err := app.FoundryEngine.ExecuteKineticAction(context.Background(), foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     supID,
		ActionType:     "REORDER_APPROVE_TEST",
		ActorID:        "agent-kinetic-1",
		ActorRole:      "AGENT",
		IdempotencyKey: "bs-kinetic-" + uuid.NewString()[:8],
		Parameters:     map[string]any{"sku": "sku-1", "qty": int64(5)},
	}, func(context.Context) (map[string]any, error) {
		return map[string]any{"approved_qty": int64(5)}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteKineticAction: %v", err)
	}
	if rec == nil || rec.ActionExecutionID == "" {
		t.Fatalf("unexpected action execution record: %+v", rec)
	}

	verify, err := app.Audit.VerifyDayChain(context.Background(), supID, today)
	if err != nil {
		t.Fatalf("VerifyDayChain: %v", err)
	}
	if !verify.Valid || verify.EventCount < 1 {
		t.Fatalf("expected valid Foundry audit chain with >=1 event, got %+v", verify)
	}
}
