package tax_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func envOr(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func newEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		t.Skip("SPANNER_EMULATOR_HOST is unset; skipping spanner test")
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))

	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return client
}

func TestTaxRepository_EndToEnd(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	repo := tax.NewSpannerRepository(client)

	country := "TZTEST"
	now := time.Now().UTC()
	regime1 := tax.TaxRegimeVersion{
		Id:              uuid.NewString(),
		CountryCode:     country,
		EffectiveFrom:   now.Add(-24 * time.Hour),
		Currency:        "USD",
		VatRatesBps:     []int64{1000, 500},
		SimplifiedRules: []byte(`{"threshold": 100}`),
		CreatedAt:       now,
		CreatedBy:       "test-admin",
		UpdatedAt:       now,
	}

	// 1. Create Regime
	err := repo.CreateRegime(ctx, regime1)
	require.NoError(t, err)

	// 2. Get Regime
	fetched, found, err := repo.GetRegime(ctx, regime1.Id)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, regime1.CountryCode, fetched.CountryCode)
	assert.ElementsMatch(t, regime1.VatRatesBps, fetched.VatRatesBps)

	// 3. Get Active Regime
	active, found, err := repo.GetActiveRegime(ctx, nil, country, now)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, regime1.Id, active.Id)

	// 4. List Regimes
	list, err := repo.ListRegimes(ctx, country, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// 5. Insert Line Snapshot
	snapshot := tax.OrderLineFiscalSnapshot{
		OrderId:     "order-123",
		OrderLineId: "line-1",
		RegimeId:    regime1.Id,
		VatRateBps:  1000,
		NetMinor:    10000,
		VatMinor:    1000,
		GrossMinor:  11000,
		SnapshotAt:  now,
		CreatedAt:   now,
	}
	err = repo.InsertLineSnapshot(ctx, nil, snapshot)
	require.NoError(t, err)
}
