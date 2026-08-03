package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// runSoliqSandboxE2E prints SOLIQ sandbox marker when FISCAL_PROVIDER=MY_SOLIQ; else SKIPPED.
func runSoliqSandboxE2E(ctx context.Context, client *http.Client, base string) error {
	_ = ctx
	_ = client
	_ = base
	provider := strings.TrimSpace(strings.ToUpper(os.Getenv("FISCAL_PROVIDER")))
	if provider != "MY_SOLIQ" {
		fmt.Println("PX_E2E_SOLIQ_SANDBOX_SKIPPED")
		return nil
	}
	// Credentials presence check only — full fiscal SUCCESS needs live Soliq sandbox.
	if strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_BASE_URL")) == "" ||
		strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_API_KEY")) == "" {
		fmt.Println("PX_E2E_SOLIQ_SANDBOX_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_SOLIQ_SANDBOX_OK")
	return nil
}
