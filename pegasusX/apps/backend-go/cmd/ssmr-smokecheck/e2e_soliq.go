package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// runSoliqSandboxE2E proves the MY_SOLIQ OFD chain.
//
// Always: in-process sign → submit → poll against a Soliq-shaped mock
// (closes the "env-presence only" gap for CI/SSMR).
//
// Optional live: set FISCAL_MY_SOLIQ_LIVE_PROOF=1 with BaseURL + APIKey + TIN
// (+ real EDS signer via FISCAL_MY_SOLIQ_SIGNER) to hit the operator sandbox.
// Credentials alone without LIVE_PROOF print CREDS_PRESENT_LIVE_PROOF_OFF.
func runSoliqSandboxE2E(ctx context.Context, client *http.Client, base string) error {
	_ = client
	_ = base
	if err := proveSoliqSignSubmitPoll(ctx); err != nil {
		return fmt.Errorf("soliq contract sign→submit→poll: %w", err)
	}
	fmt.Println("PX_E2E_SOLIQ_CONTRACT_OK")

	provider := strings.TrimSpace(strings.ToUpper(os.Getenv("FISCAL_PROVIDER")))
	if provider != "MY_SOLIQ" {
		fmt.Println("PX_E2E_SOLIQ_SANDBOX_SKIPPED")
		return nil
	}
	liveBase := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_BASE_URL"))
	liveKey := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_API_KEY"))
	liveTIN := strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_TIN"))
	if liveBase == "" || liveKey == "" || liveTIN == "" {
		fmt.Println("PX_E2E_SOLIQ_SANDBOX_SKIPPED")
		return nil
	}
	if strings.TrimSpace(os.Getenv("FISCAL_MY_SOLIQ_LIVE_PROOF")) != "1" {
		fmt.Println("PX_E2E_SOLIQ_SANDBOX_CREDS_PRESENT_LIVE_PROOF_OFF")
		return nil
	}

	p, err := order.NewMySoliqProviderFromEnv()
	if err != nil {
		return fmt.Errorf("live mysoliq provider: %w", err)
	}
	res, err := p.CreateReceipt(ctx, order.FiscalCreateRequest{
		AttemptID:     fmt.Sprintf("live-proof-%d", os.Getpid()),
		OrderID:       "ord-live-proof",
		SupplierID:    "sup-live-proof",
		RetailerID:    "ret-live-proof",
		AmountMinor:   1000,
		Currency:      "UZS",
		PaymentMethod: "CARD",
		LineItems: []order.LineItem{
			{SKU: "PROOF", Name: "EDS live proof", Quantity: 1, UnitPrice: 1000},
		},
	})
	if err != nil {
		return fmt.Errorf("live CreateReceipt: %w", err)
	}
	if strings.TrimSpace(res.FiscalReceiptID) == "" {
		return fmt.Errorf("live CreateReceipt: empty receipt id")
	}
	st, err := p.GetSoliqClient().CheckStatus(ctx, res.FiscalReceiptID)
	if err != nil {
		return fmt.Errorf("live CheckStatus: %w", err)
	}
	if strings.TrimSpace(st.Status) == "" {
		return fmt.Errorf("live CheckStatus: empty status")
	}
	fmt.Printf("PX_E2E_SOLIQ_SANDBOX_LIVE_OK receipt=%s status=%s\n", res.FiscalReceiptID, st.Status)
	return nil
}

func proveSoliqSignSubmitPoll(ctx context.Context) error {
	const signKey = "ssmr-soliq-contract-key-16"
	signer, err := fiscal.NewDevHMACSigner([]byte(signKey))
	if err != nil {
		return err
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ehf/submit":
			body, _ := io.ReadAll(r.Body)
			var submitted map[string]any
			if err := json.Unmarshal(body, &submitted); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			sig, _ := submitted["signature"].(string)
			if !strings.HasPrefix(sig, "DEVHMAC.") {
				http.Error(w, "unsigned", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"EHF-SSMR-1"}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/ehf/") && strings.HasSuffix(r.URL.Path, "/status"):
			_, _ = w.Write([]byte(`{"success":true,"data":{"status":"ACCEPTED"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	p, err := order.NewMySoliqProvider(ts.URL, "ssmr-api-key", "300000000", signer)
	if err != nil {
		return err
	}
	res, err := p.CreateReceipt(ctx, order.FiscalCreateRequest{
		AttemptID:     "ssmr-soliq-1",
		OrderID:       "ord-ssmr-soliq",
		SupplierID:    "sup-ssmr",
		RetailerID:    "ret-ssmr",
		AmountMinor:   125000,
		Currency:      "UZS",
		PaymentMethod: "CARD",
		LineItems: []order.LineItem{
			{SKU: "SKU-1", Name: "Water 1.5L", Quantity: 10, UnitPrice: 12500},
		},
	})
	if err != nil {
		return err
	}
	if res.FiscalReceiptID != "EHF-SSMR-1" {
		return fmt.Errorf("receipt=%q want EHF-SSMR-1", res.FiscalReceiptID)
	}
	st, err := p.GetSoliqClient().CheckStatus(ctx, res.FiscalReceiptID)
	if err != nil {
		return err
	}
	if st.Status != "ACCEPTED" {
		return fmt.Errorf("status=%q want ACCEPTED", st.Status)
	}
	return nil
}
