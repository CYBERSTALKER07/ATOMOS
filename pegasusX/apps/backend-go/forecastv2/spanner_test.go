package forecastv2

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func newForecastEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if os.Getenv("PARITY_REQUIRE_SPANNER") == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner test")
	}
	envOr := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))
	cli, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return cli
}

func TestLiveSpanner_ForecastV2RunApproveAndOutbox(t *testing.T) {
	ctx := context.Background()
	client := newForecastEmulatorClient(t, ctx)
	defer client.Close()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	supID := "fs-" + tag
	whID := "fw-" + tag
	skuID := "sku-" + tag
	asOf := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	asOfDay := civil.DateOf(asOf)

	defer func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("ForecastValues", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID + "\uffff"}}),
			spanner.Delete("ForecastExplanations", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID + "\uffff"}}),
			spanner.Delete("ForecastRuns", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID + "\uffff"}}),
			spanner.Delete("DemandForecastBaseline", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID + "\uffff"}}),
		})
	}()

	// Seed 28 days of DemandForecastBaseline history so SpannerRepository.LoadHistoricalSeries finds series cleanly.
	var seedMuts []*spanner.Mutation
	start := asOfDay.AddDays(-27)
	for i := 0; i < 28; i++ {
		d := start.AddDays(i)
		qty := int64(12 + (i % 4))
		seedMuts = append(seedMuts, spanner.InsertOrUpdateMap("DemandForecastBaseline", map[string]any{
			"SupplierId":     supID,
			"ForecastDate":   d,
			"WarehouseId":    whID,
			"ProductId":      skuID,
			"BaselineQty":    qty,
			"Confidence":     0.8,
			"Source":         "seed_history",
			"BaselineSource": "seed_history",
			"CreatedAt":      spanner.CommitTimestamp,
		}))
	}
	if _, err := client.Apply(ctx, seedMuts); err != nil {
		t.Fatalf("seed DemandForecastBaseline: %v", err)
	}

	svc := NewService(client)
	svc.SetNow(func() time.Time { return asOf })

	runOut, err := svc.RunForecast(ctx, RunForecastInput{
		EntityID:    supID,
		HorizonDays: 5,
		AsOf:        asOf,
		TriggeredBy: "planner-live-1",
	})
	if err != nil {
		t.Fatalf("live RunForecast failed: %v", err)
	}
	if runOut.Run.SkuCount != 1 || len(runOut.Forecasts) != 10 || len(runOut.Explanations) != 1 {
		t.Fatalf("unexpected live RunForecast output: run=%+v forecasts=%d expls=%d", runOut.Run, len(runOut.Forecasts), len(runOut.Explanations))
	}

	// Propose a demand adjustment and approve it as human manager, verifying ForecastValueAddBps persisted in Spanner.
	adjBps := int64(2000)
	prop, err := svc.ProposeDemandAdjustment(ctx, ProposeAdjustmentInput{
		EntityID:         supID,
		RunID:            runOut.Run.RunID,
		LocationID:       whID,
		SKU:              skuID,
		AdjustmentBps:    &adjBps,
		AdjustmentReason: "Navruz tea display",
		ProposedBy:       "agent:planner_bot",
	})
	if err != nil {
		t.Fatalf("live ProposeDemandAdjustment failed: %v", err)
	}

	// Verify agent approval rejected cleanly.
	if _, err := svc.ApproveDemandAdjustment(ctx, ApproveAdjustmentInput{
		EntityID:      supID,
		ExplanationID: prop.ExplanationID,
		ActorID:       "agent:planner_bot",
		ActorType:     "agent",
		ActorRole:     "ADMIN",
	}); err != ErrAgentForbidden {
		t.Fatalf("expected ErrAgentForbidden, got %v", err)
	}

	actualMilli := prop.AdjustedP50Milli
	approved, err := svc.ApproveDemandAdjustment(ctx, ApproveAdjustmentInput{
		EntityID:       supID,
		ExplanationID:  prop.ExplanationID,
		ActorID:        "usr-planner-2",
		ActorType:      "human",
		ActorRole:      string(auth.RoleAdmin),
		Comment:        "verified promotion calendar",
		ActualQtyMilli: &actualMilli,
	})
	if err != nil {
		t.Fatalf("live ApproveDemandAdjustment failed: %v", err)
	}
	if approved.Status != ExplStatusApproved || approved.ForecastValueAddBps <= 0 {
		t.Fatalf("expected APPROVED with positive FVA bps, got %+v", approved)
	}

	// Read back via GetForecast and check OutboxEvents for EventForecastRunCompleted + EventPlanningForecastUpdated.
	view, err := svc.GetForecast(ctx, ForecastQuery{EntityID: supID, HorizonDays: 5})
	if err != nil {
		t.Fatalf("live GetForecast failed: %v", err)
	}
	if view.Run == nil || len(view.Forecasts) == 0 || len(view.Explanations) < 2 {
		t.Fatalf("expected persisted forecast run/values/explanations, got %+v", view)
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	obIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT CAST(Payload AS STRING) FROM OutboxEvents
		      WHERE AggregateId IN (@eid, @runid)`,
		Params: map[string]any{"eid": supID, "runid": runOut.Run.RunID},
	})
	seen := map[string]bool{}
	for {
		row, err := obIter.Next()
		if err == iterator.Done || err != nil {
			break
		}
		var payload string
		if row.Columns(&payload) == nil {
			if strings.Contains(payload, events.EventForecastRunCompleted) {
				seen[events.EventForecastRunCompleted] = true
			}
			if strings.Contains(payload, events.EventPlanningForecastUpdated) {
				seen[events.EventPlanningForecastUpdated] = true
			}
		}
	}
	obIter.Stop()
	if !seen[events.EventForecastRunCompleted] || !seen[events.EventPlanningForecastUpdated] {
		t.Fatalf("missing forecast outbox events in live Spanner: %v", seen)
	}
}
