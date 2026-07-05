package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runLoadTokens prints shell-exportable auth material for k6 load profiles.
func runLoadTokens(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 20 * time.Second}

	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}

	supplierID, cookie, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("supplier session: %w", err)
	}
	if err := putSupplierTopology(ctx, client, base, cookie, cfg); err != nil {
		return fmt.Errorf("supplier topology: %w", err)
	}

	poolSize := loadRetailerPoolSize()
	retailerTokens := make([]string, 0, poolSize)
	var primaryRetailerID string
	var primaryH3Cell string

	for i := 0; i < poolSize; i++ {
		phone := fmt.Sprintf("+998901%06d", 900000+i)
		retailerID, h3Cell, err := registerRetailerWithPhone(ctx, client, base, cfg, phone)
		if err != nil {
			return fmt.Errorf("retailer register %d: %w", i, err)
		}
		if err := grantRetailerCredit(ctx, client, base, cookie, retailerID, 2_000_000_000); err != nil {
			return fmt.Errorf("retailer credit grant %d: %w", i, err)
		}
		if i == 0 {
			primaryRetailerID = retailerID
			primaryH3Cell = h3Cell
		}

		token, err := auth.Issue(auth.Claims{
			Subject:    retailerID,
			Role:       auth.RoleRetailer,
			SupplierID: supplierID,
		}, auth.IssueOptions{
			Secret: cfg.JWTSecret,
			Issuer: cfg.JWTIssuer,
			TTL:    2 * time.Hour,
		})
		if err != nil {
			return fmt.Errorf("issue retailer jwt %d: %w", i, err)
		}
		retailerTokens = append(retailerTokens, token)
	}

	// Shell-safe single-quoted exports for `eval "$(go run ... loadtokens)"`.
	fmt.Fprintf(os.Stdout, "export PUBLIC_BASE_URL=%s\n", shellQuote(base))
	fmt.Fprintf(os.Stdout, "export SUPPLIER_ID=%s\n", shellQuote(supplierID))
	fmt.Fprintf(os.Stdout, "export SUPPLIER_COOKIE=%s\n", shellQuote(cookie))
	fmt.Fprintf(os.Stdout, "export RETAILER_TOKEN=%s\n", shellQuote(retailerTokens[0]))
	fmt.Fprintf(os.Stdout, "export RETAILER_TOKENS=%s\n", shellQuote(strings.Join(retailerTokens, "|")))
	fmt.Fprintf(os.Stdout, "export H3_CELL=%s\n", shellQuote(primaryH3Cell))
	fmt.Fprintf(os.Stdout, "export RETAILER_ID=%s\n", shellQuote(primaryRetailerID))
	fmt.Fprintf(os.Stdout, "export LOAD_RETAILER_POOL_SIZE=%d\n", poolSize)
	return nil
}

// loadTokensTimeout bounds bootstrap for multi-retailer k6 pools (cert/stress).
func loadTokensTimeout() time.Duration {
	pool := loadRetailerPoolSize()
	secs := 45 + pool*2
	if secs > 180 {
		secs = 180
	}
	return time.Duration(secs) * time.Second
}

func loadRetailerPoolSize() int {
	raw := strings.TrimSpace(os.Getenv("LOAD_RETAILER_POOL_SIZE"))
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	if n > 256 {
		return 256
	}
	return n
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
