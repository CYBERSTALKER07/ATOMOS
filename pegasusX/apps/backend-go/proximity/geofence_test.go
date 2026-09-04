package proximity

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestWithinDeliveryApproach_UZPack150(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	if !WithinDeliveryApproach(0.149) {
		t.Fatal("149m should be within UZ pack 150m")
	}
	if WithinDeliveryApproach(0.15) {
		t.Fatal("150m boundary is outside (strict less-than)")
	}
	if WithinDeliveryApproach(0.2) {
		t.Fatal("200m must fail at pack 150 (old 500m dual)")
	}
	if WithinDeliveryApproach(0.499) {
		t.Fatal("499m must not use deleted 500m approach radius")
	}
}

func TestWithinDeliveryApproach_PlannedPackFailClosed(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU"})
	if WithinDeliveryApproachForSupplier(ctx, "sup-1", 0.01) {
		t.Fatal("planned EU pack must not treat any distance as approaching")
	}
}
