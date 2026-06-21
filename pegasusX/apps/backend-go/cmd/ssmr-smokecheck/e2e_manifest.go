package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func runManifestSealSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 45 * time.Second}
	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	supplierID, _, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("supplier session: %w", err)
	}
	if err := runPayloaderE2E(ctx, client, base, cfg, supplierID, nil); err != nil {
		return err
	}
	fmt.Println("PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK")
	return nil
}
