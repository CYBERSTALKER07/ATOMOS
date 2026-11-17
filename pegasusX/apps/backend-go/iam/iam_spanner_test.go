package iam

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openIAMSpannerTestClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner IAM integration test")
	}

	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := spanner.NewClient(
		ctx,
		dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "SodPolicies", spanner.Key{"__iam_probe__", "__iam_probe__"}, []string{"PolicyId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("SodPolicies unavailable: %v", probeErr)
		}
		t.Skipf("SodPolicies unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func envOrDefault(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func cleanupIAMTenant(t *testing.T, client *spanner.Client, tid string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("RolePermissions", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierUserRoles", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SodPolicies", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SodViolations", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		}
		iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, err := iter.Next()
			if err == iterator.Done || err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		_, _ = client.Apply(ctx, muts)
	})
}

func TestSpannerIAM_Lifecycle_Seed_Assign_SoD_And_Outbox(t *testing.T) {
	client := openIAMSpannerTestClient(t)
	ctx := context.Background()
	tid := "sup_iam_" + uuid.NewString()[:8]
	cleanupIAMTenant(t, client, tid)

	svc := NewService(client)
	pols, err := svc.ListSodPolicies(ctx, tid)
	if err != nil || len(pols) != 5 {
		t.Fatalf("ListSodPolicies = (%d, %v), want 5", len(pols), err)
	}

	roles, _, err := svc.ListRoles(ctx, tid)
	if err != nil || len(roles) != 9 {
		t.Fatalf("ListRoles = (%d, %v), want 9", len(roles), err)
	}

	// Assign ACCOUNTANT -> succeeds and emits EventIAMRoleChanged in OutboxEvents.
	assigned, err := svc.AssignRole(ctx, AssignRoleInput{
		TenantID:   tid,
		UserID:     "usr-1",
		RoleName:   auth.SupplierRoleAccountant,
		AssignedBy: "owner-1",
		TraceID:    "tr-iam-1",
	})
	if err != nil || len(assigned) != 1 {
		t.Fatalf("AssignRole ACCOUNTANT = (%+v, %v)", assigned, err)
	}

	// Conflicting FINANCE_MANAGER assignment blocked with ErrSodRoleConflict and logs SodViolations + outbox event.
	_, err = svc.AssignRole(ctx, AssignRoleInput{
		TenantID:   tid,
		UserID:     "usr-1",
		RoleName:   auth.SupplierRoleFinanceManager,
		AssignedBy: "owner-1",
		TraceID:    "tr-iam-2",
	})
	if !errors.Is(err, ErrSodRoleConflict) {
		t.Fatalf("expected ErrSodRoleConflict, got %v", err)
	}

	// CheckFourEyes blocks self-approval.
	err = svc.CheckFourEyes(ctx, CheckFourEyesInput{
		TenantID:    tid,
		ActorID:     "usr-1",
		Role:        auth.SupplierRoleAccountant,
		Action:      auth.PermJournalApprove,
		MakerAction: auth.PermJournalPost,
		ObjectType:  "JournalEntry",
		ObjectID:    "je-99",
		MakerID:     "usr-1",
		RoutePath:   "/v1/finance/journal/je-99/approve",
		IsApproval:  true,
	})
	if !errors.Is(err, ErrSodViolation) {
		t.Fatalf("CheckFourEyes self-approval got %v, want ErrSodViolation", err)
	}

	violations, err := svc.ListSodViolations(ctx, tid, "", 10)
	if err != nil || len(violations) != 2 {
		t.Fatalf("ListSodViolations = (%d, %v), want 2", len(violations), err)
	}

	// Verify OutboxEvents recorded both iam.role.changed and sod.violation.detected.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	var payloads []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			t.Fatalf("scan Payload: %v", err)
		}
		payloads = append(payloads, string(raw))
	}
	joined := strings.Join(payloads, "\n")
	if !strings.Contains(joined, `"iam.role.changed"`) || !strings.Contains(joined, `"sod.violation.detected"`) {
		t.Fatalf("expected iam.role.changed and sod.violation.detected in outbox, got:\n%s", joined)
	}
}
