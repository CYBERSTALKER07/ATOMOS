package orgoidc

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

func openOrgOIDCSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner orgoidc test")
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
	_, pErr := ro.ReadRow(ctx, "SupplierOIDC", spanner.Key{"__probe__"}, []string{"SupplierId"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		client.Close()
		t.Fatalf("SupplierOIDC table unavailable: %v", pErr)
	}
	t.Cleanup(client.Close)
	return client
}

func TestSpannerStore_Get_NotFoundReturnsFalseNilError(t *testing.T) {
	client := openOrgOIDCSpannerClient(t)
	store := NewSpannerStore(client)

	cfg, ok, err := store.Get(context.Background(), "__missing_oidc_supplier__")
	if err != nil {
		t.Fatalf("expected nil error on missing SupplierOIDC row, got: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false on missing SupplierOIDC row, got config: %+v", cfg)
	}
}
