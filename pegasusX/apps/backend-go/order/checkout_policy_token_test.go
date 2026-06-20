package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckoutPolicyToken_GraceOnRejectFlip(t *testing.T) {
	secret := "test-secret"
	now := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)
	snap := CheckoutPolicySnapshot{
		WarehouseID: "wh-1",
		StockPolicy: outOfStockPolicyAcceptBackorder,
	}
	token, exp, err := IssueCheckoutPolicyToken(secret, snap, now)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.True(t, exp.After(now))

	resolved, err := ResolveCheckoutPolicyToken(secret, token, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, outOfStockPolicyAcceptBackorder, resolved.StockPolicy)

	effective := EffectiveWarehouseStockPolicy(outOfStockPolicyReject, &resolved)
	require.Equal(t, outOfStockPolicyAcceptBackorder, effective)

	effective = EffectiveWarehouseStockPolicy(outOfStockPolicyAcceptBackorder, &resolved)
	require.Equal(t, outOfStockPolicyAcceptBackorder, effective)
}
