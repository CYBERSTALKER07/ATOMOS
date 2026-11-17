package controltower

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openCTSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Control Tower recon test")
	}

	dbPath := "projects/" + envOr("SPANNER_PROJECT", "pegasusx-local") +
		"/instances/" + envOr("SPANNER_INSTANCE", "pegasusx-instance") +
		"/databases/" + envOr("SPANNER_DATABASE", "pegasusx-db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner connect skipped: %v", err)
	}
	_, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReconciliationBreaks", spanner.Key{"__ct_probe__", "__ct_break__"}, []string{"BreakId"})
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("ReconciliationBreaks unavailable: %v", probeErr)
		}
		t.Skipf("ReconciliationBreaks unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func TestControlTower_ReconciliationBreaks_TenantIsolation_LiveSpanner(t *testing.T) {
	client := openCTSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	supA := "sup-cta-" + uuid.NewString()[:8]
	supB := "sup-ctb-" + uuid.NewString()[:8]
	brA := "brk-a-" + uuid.NewString()[:8]
	brB := "brk-b-" + uuid.NewString()[:8]
	past := time.Now().UTC().Add(-1 * time.Minute) // visible under 15s ExactStaleness immediately

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("ReconciliationBreaks", spanner.Key{supA, brA}),
			spanner.Delete("ReconciliationBreaks", spanner.Key{supB, brB}),
		})
	})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ReconciliationBreaks", map[string]any{
			"EntityId": supA, "BreakId": brA, "RunId": "run-a", "Dimension": "ORDERS_AR",
			"ObjectType": "DIMENSION", "ObjectId": "ORDERS_AR", "ExpectedMinor": int64(100_000),
			"ActualMinor": int64(50_000), "DeltaMinor": int64(-50_000), "Currency": "UZS",
			"Severity": "HIGH", "Status": "OPEN", "OwnerId": "FINANCE_CONTROLLER",
			"ControlTowerExceptionId": "ct-" + brA, "CreatedAt": past, "UpdatedAt": past,
		}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{
			"EntityId": supB, "BreakId": brB, "RunId": "run-b", "Dimension": "TAX_FISCAL",
			"ObjectType": "DIMENSION", "ObjectId": "TAX_FISCAL", "ExpectedMinor": int64(20_000),
			"ActualMinor": int64(0), "DeltaMinor": int64(-20_000), "Currency": "UZS",
			"Severity": "CRITICAL", "Status": "OPEN", "OwnerId": "TAX_COMPLIANCE",
			"ControlTowerExceptionId": "ct-" + brB, "CreatedAt": past, "UpdatedAt": past,
		}),
	}); err != nil {
		t.Fatalf("seed ReconciliationBreaks: %v", err)
	}

	repo := NewSpannerRepository(client)
	exA, err := repo.ListOpenExceptions(ctx, supA)
	if err != nil {
		t.Fatalf("ListOpenExceptions(supA): %v", err)
	}
	if len(exA) != 1 || exA[0].ExceptionID != "ct-"+brA || exA[0].SupplierID != supA || exA[0].Type != "RECON_ORDERS_AR" {
		t.Fatalf("supA exceptions = %+v, want strictly 1 RECON_ORDERS_AR for supA", exA)
	}

	supIDs, err := repo.ListSupplierIDsWithOpenExceptions(ctx)
	if err != nil {
		t.Fatalf("ListSupplierIDsWithOpenExceptions: %v", err)
	}
	hasA, hasB := false, false
	for _, s := range supIDs {
		if s == supA {
			hasA = true
		}
		if s == supB {
			hasB = true
		}
	}
	if !hasA || !hasB {
		t.Fatalf("ListSupplierIDsWithOpenExceptions = %v, want both %s and %s", supIDs, supA, supB)
	}
}
