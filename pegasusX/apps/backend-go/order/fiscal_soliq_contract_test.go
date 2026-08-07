package order

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/fiscal"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

// Record/replay contract tests for the MY_SOLIQ EHF adapter. The replay server
// mirrors Soliq's documented submit surface (POST /v1/ehf/submit, Bearer auth,
// Idempotency-Key, {"success":bool,"data":{"ehf_id":...},"error":{...}}).
// Re-record golden requests with UPDATE_GOLDEN=1.

const soliqContractSignKey = "contract-test-sign-key-0123456789"

func newSoliqContractProvider(t *testing.T, baseURL string) *MySoliqProvider {
	t.Helper()
	signer, err := fiscal.NewDevHMACSigner([]byte(soliqContractSignKey))
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	p := &MySoliqProvider{
		TIN:         "300000000",
		soliqClient: soliq.NewClient(soliq.SoliqConfig{BaseURL: baseURL, APIKey: "contract-api-key", TIN: "300000000"}),
	}
	p.SetSigner(signer)
	return p
}

func soliqContractRequest() FiscalCreateRequest {
	return FiscalCreateRequest{
		AttemptID:     "att-contract-1",
		OrderID:       "ord-contract-1",
		SupplierID:    "sup-contract",
		RetailerID:    "ret-contract",
		AmountMinor:   125000,
		Currency:      "UZS",
		PaymentMethod: "CARD",
		LineItems: []LineItem{
			{SKU: "SKU-1", Name: "Water 1.5L", Quantity: 10, UnitPrice: 12500},
		},
	}
}

func TestMySoliqContract_SubmitSuccess(t *testing.T) {
	var gotAuth, gotIdem string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ehf/submit" {
			http.NotFound(w, r)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotIdem = r.Header.Get("Idempotency-Key")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"ehf_id":"EHF-9001"}}`))
	}))
	defer ts.Close()

	p := newSoliqContractProvider(t, ts.URL)
	res, err := p.CreateReceipt(context.Background(), soliqContractRequest())
	if err != nil {
		t.Fatalf("CreateReceipt: %v", err)
	}
	if res.FiscalReceiptID != "EHF-9001" {
		t.Fatalf("receipt id = %q, want EHF-9001", res.FiscalReceiptID)
	}
	if gotAuth != "Bearer contract-api-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotIdem != "att-contract-1" {
		t.Fatalf("Idempotency-Key = %q, want att-contract-1", gotIdem)
	}

	// The submitted EHF must carry an attached signature over the canonical body.
	var submitted map[string]any
	if err := json.Unmarshal(gotBody, &submitted); err != nil {
		t.Fatalf("submitted body not JSON: %v", err)
	}
	sig, _ := submitted["signature"].(string)
	if !strings.HasPrefix(sig, "DEVHMAC.") {
		t.Fatalf("signature missing or wrong scheme: %q", sig)
	}
	if submitted["idempotency_key"] != "att-contract-1" {
		t.Fatalf("idempotency_key = %v", submitted["idempotency_key"])
	}

	// Golden replay: the canonical signed envelope must stay byte-stable.
	goldenPath := "testdata/soliq_submit_request.golden.json"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		var pretty strings.Builder
		enc := json.NewEncoder(&pretty)
		enc.SetIndent("", "  ")
		if err := enc.Encode(submitted); err != nil {
			t.Fatalf("encode golden: %v", err)
		}
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(pretty.String()), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Skip("golden recorded")
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (record with UPDATE_GOLDEN=1): %v", err)
	}
	var goldenMap map[string]any
	if err := json.Unmarshal(golden, &goldenMap); err != nil {
		t.Fatalf("golden not JSON: %v", err)
	}
	gotNorm, _ := json.Marshal(submitted)
	wantNorm, _ := json.Marshal(goldenMap)
	if string(gotNorm) != string(wantNorm) {
		t.Fatalf("submitted EHF envelope drifted from golden\n got: %s\nwant: %s", gotNorm, wantNorm)
	}
}

func TestMySoliqContract_BusinessRejectIsPermanent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"VALIDATION_TIN","message":"tin mismatch"}}`))
	}))
	defer ts.Close()

	p := newSoliqContractProvider(t, ts.URL)
	_, err := p.CreateReceipt(context.Background(), soliqContractRequest())
	if err == nil || !strings.Contains(err.Error(), "VALIDATION_TIN") {
		t.Fatalf("want permanent validation error, got %v", err)
	}
}

func TestMySoliqContract_ServerErrorIsRetryable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer ts.Close()

	p := newSoliqContractProvider(t, ts.URL)
	_, err := p.CreateReceipt(context.Background(), soliqContractRequest())
	if err == nil || !strings.Contains(err.Error(), "mysoliq error") {
		t.Fatalf("want retryable submit error, got %v", err)
	}
}

func TestMySoliqContract_MissingEhfIDFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer ts.Close()

	p := newSoliqContractProvider(t, ts.URL)
	_, err := p.CreateReceipt(context.Background(), soliqContractRequest())
	if err == nil || !strings.Contains(err.Error(), "missing receipt_id") {
		t.Fatalf("want missing receipt_id error, got %v", err)
	}
}

func TestSoliqSignerFromEnv_FailClosed(t *testing.T) {
	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "")
	if _, err := fiscal.SignerFromEnv("dev"); err == nil {
		t.Fatal("empty signer kind must be a construction error")
	}

	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "dev-hmac")
	t.Setenv("FISCAL_MY_SOLIQ_SIGN_KEY", "0123456789abcdef")
	if _, err := fiscal.SignerFromEnv("production"); err == nil {
		t.Fatal("dev-hmac must be rejected in production")
	}
	if _, err := fiscal.SignerFromEnv("dev"); err != nil {
		t.Fatalf("dev-hmac must work in dev: %v", err)
	}

	t.Setenv("FISCAL_MY_SOLIQ_SIGNER", "pkcs12")
	if _, err := fiscal.SignerFromEnv("production"); err == nil || !strings.Contains(err.Error(), "procurement") {
		t.Fatalf("pkcs12 must surface the procurement owner task, got %v", err)
	}
}
