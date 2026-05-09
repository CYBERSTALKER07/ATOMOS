package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	
	"backend-go/outbox"
	"backend-go/telemetry"
)

// PrometheusResponse matches the JSON structure of a Prometheus API query.
type PrometheusResponse struct {
        Status string `json:"status"`
        Data   struct {
                ResultType string `json:"resultType"`
                Result     []struct {
                        Metric map[string]string `json:"metric"`
                        Value  []interface{}     `json:"value"`
                } `json:"result"`
        } `json:"data"`
}

// UtilityMeter polls Prometheus for API usage per supplier and writes overage fees.
type UtilityMeter struct {
        spannerClient *spanner.Client
        prometheusURL string
        httpClient    *http.Client
}

// NewUtilityMeter creates a new over-quota fee engine.
func NewUtilityMeter(client *spanner.Client) *UtilityMeter {
        pURL := os.Getenv("PROMETHEUS_API_URL")
        if pURL == "" {
                pURL = "http://prometheus:9090"
        }
        return &UtilityMeter{
                spannerClient: client,
                prometheusURL: pURL,
                httpClient:    &http.Client{Timeout: 10 * time.Second},
        }
}

// Run executes a single metering check. Expected to be scheduled nightly.
func (u *UtilityMeter) Run(ctx context.Context) error {
        slog.InfoContext(ctx, "utility_meter.run_started")
        
        query := `sum(increase(pegasus_http_requests_total{supplier_id!="",supplier_id!="unknown",supplier_id!="anonymous"}[30d])) by (supplier_id)`
        
        req, err := http.NewRequestWithContext(ctx, "GET", u.prometheusURL+"/api/v1/query", nil)
        if err != nil {
                return fmt.Errorf("build request: %w", err)
        }
        q := req.URL.Query()
        q.Add("query", query)
        req.URL.RawQuery = q.Encode()

        resp, err := u.httpClient.Do(req)
        if err != nil {
                slog.ErrorContext(ctx, "utility_meter.prometheus_fetch_failed", "err", err)
                return err // non-fatal for system, but fail the tick
        }
        defer resp.Body.Close()

        var pResp PrometheusResponse
        if err := json.NewDecoder(resp.Body).Decode(&pResp); err != nil {
                return fmt.Errorf("decode prometheus: %w", err)
        }

        if pResp.Status != "success" {
                return fmt.Errorf("prometheus returned non-success: %s", pResp.Status)
        }

        for _, res := range pResp.Data.Result {
                supplierID := res.Metric["supplier_id"]
                if supplierID == "" {
                        continue
                }

                if len(res.Value) < 2 {
                        continue
                }

                valStr, ok := res.Value[1].(string)
                if !ok {
                        continue
                }

                reqCountFloat, err := strconv.ParseFloat(valStr, 64)
                if err != nil {
                        continue
                }

                reqCount := int64(reqCountFloat)
                
                // If usage > 1M hits, write a micro-fee ledger debit ($0.01 per 1000 requests)
                if reqCount > 1000000 {
                        // Overage is everything above 1M.
                        overage := reqCount - 1000000
                        
                        // $0.01 per 1000 requests = 1 cent (1 minor unit in USD) per 1000.
                        // Assuming minor units: 100 cents = $1.
                        cents := overage / 1000
                        if cents > 0 {
                                u.chargeOverage(ctx, supplierID, cents)
                        }
                }
        }

        slog.InfoContext(ctx, "utility_meter.run_completed")
        return nil
}

// chargeOverage writes an ENTRY_TYPE_INFRA_OVERAGE debit to the supplier's wallet.
func (u *UtilityMeter) chargeOverage(ctx context.Context, supplierID string, minorUnits int64) {
        _, err := u.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
                entryID := spanner.CommitTimestamp.String()
                
                var muts []*spanner.Mutation
                
                muts = append(muts, spanner.Insert("LedgerEntries",
                        []string{"EntryId", "AccountId", "Amount", "Currency", "EntryType", "CreatedAt", "Status"},
                        []interface{}{entryID + "-debit", "supplier:" + supplierID + ":wallet", -minorUnits, "USD", "ENTRY_TYPE_INFRA_OVERAGE", spanner.CommitTimestamp, "SETTLED"},
                ))
                
                muts = append(muts, spanner.Insert("LedgerEntries",
                        []string{"EntryId", "AccountId", "Amount", "Currency", "EntryType", "CreatedAt", "Status"},
                        []interface{}{entryID + "-credit", "platform:fee", minorUnits, "USD", "ENTRY_TYPE_INFRA_OVERAGE", spanner.CommitTimestamp, "SETTLED"},
                ))
                
                if err := txn.BufferWrite(muts); err != nil {
                        return err
                }
                
                type OverageEvent struct {
                        SupplierID string `json:"supplier_id"`
                        Amount     int64  `json:"amount"`
                        Currency   string `json:"currency"`
                }
                return outbox.EmitJSON(txn, "Ledger", supplierID, "ENTRY_TYPE_INFRA_OVERAGE", "ledger", OverageEvent{
                        SupplierID: supplierID,
                        Amount:     minorUnits,
                        Currency:   "USD",
                }, telemetry.TraceIDFromContext(ctx))
        })

        if err != nil {
                slog.ErrorContext(ctx, "utility_meter.charge_failed", "supplier_id", supplierID, "err", err)
        } else {
                slog.InfoContext(ctx, "utility_meter.charge_success", "supplier_id", supplierID, "amount_cents", minorUnits)
        }
}


