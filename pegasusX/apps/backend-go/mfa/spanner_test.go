package mfa

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openMFASpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner mfa test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	ro := client.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "PlatformAdminMFA", spanner.Key{"__probe__"}, []string{"Subject"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		client.Close()
		t.Fatalf("PlatformAdminMFA table unavailable: %v", pErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestSpannerRepo_Get_NotFoundReturnsFalseNilError(t *testing.T) {
	client := openMFASpannerClient(t)
	repo := NewSpannerRepository(client)

	rec, ok, err := repo.Get(context.Background(), "__missing_mfa_subject__")
	if err != nil {
		t.Fatalf("expected nil error on missing MFA row, got: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false on missing MFA row, got record: %+v", rec)
	}
}
