package supplier

import (
	"context"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// WireMarketProfileLookup registers Suppliers.MarketCode / HomeCell as the
// GS-A2 profile reader for auth.Issue and GET /v1/auth/session.
// Empty columns return ok=false so session source is env, not a silent UZ choice.
func WireMarketProfileLookup(repo Repository) {
	if repo == nil {
		auth.SetMarketProfileLookup(nil)
		return
	}
	auth.SetMarketProfileLookup(func(supplierID string) (auth.MarketProfile, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		p, ok, err := repo.GetProfile(ctx, supplierID)
		if err != nil || !ok {
			return auth.MarketProfile{}, false
		}
		code := auth.NormalizeMarketCode(p.MarketCode)
		if code == "" {
			return auth.MarketProfile{}, false
		}
		return auth.MarketProfile{
			MarketCode: code,
			HomeCell:   strings.ToLower(strings.TrimSpace(p.HomeCell)),
		}, true
	})
}
