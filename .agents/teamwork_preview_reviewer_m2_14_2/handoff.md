# Adversarial Review & Handoff Report: Milestone M2 (Requirement R2)

**Reviewer / Critic**: `teamwork_preview_reviewer_m2_14_2`  
**Timestamp**: 2026-09-25T12:21:00Z  
**Target Work Product**: `teamwork_preview_worker_m2_arch_gen2` handoff report & Milestone M2 deliverables  
**Workspace**: `/Users/shakhzod/Desktop/V.O.I.D`  
**Verdict**: **REQUEST_CHANGES**

---

## 1. Observation

Direct, empirical observations obtained from AST inspection, dependency graphs, static grep, and test executions across `pegasus.x` and `pegasusX`:

### 1.1 Forbidden Spanner & Kafka Dependency Scan in `pegasus.x/`
1. **Go Backend Dependencies & Imports**:
   - `pegasus.x/backend/go.mod` contains no references to `cloud.google.com/go/spanner`, `github.com/segmentio/kafka-go`, `github.com/IBM/sarama`, or `confluent-kafka-go`.
   - Command: `go list -m all` in `pegasus.x/backend` lists 48 direct and indirect modules; 0 match `spanner` or `kafka`.
   - Command: `go mod graph | grep -iE "spanner|kafka|sarama|confluent|segmentio"` returned exit code 1 (0 matches).
   - Command: `rg "^import \(" -A 20 pegasus.x/backend/ | grep -E "(spanner|kafka|sarama|confluent|segmentio)"` returned 0 matches.
   - Command: `rg '^import\s+"[^"]+"' pegasus.x/backend/ | grep -E "(spanner|kafka|sarama|confluent|segmentio)"` returned 0 matches.
2. **Dockerfiles & Container Definitions**:
   - 11 Dockerfiles found in `pegasus.x/` (`apps/supplier-desktop/Dockerfile`, `apps/telegram-bot/Dockerfile`, `apps/telegram-miniapp/Dockerfile`, `apps/warehouse-desktop/Dockerfile`, `backend/Dockerfile`, `docker/Dockerfile.backend`, `docker/Dockerfile.bot`, `docker/Dockerfile.miniapp`, `docker/Dockerfile.planning`, `docker/Dockerfile.portal`, `planning/Dockerfile`).
   - Command: `grep -iE "spanner|kafka" <11 Dockerfiles>` returned 0 matches.
3. **Frontend & Shared Packages**:
   - Command: `rg -i "@google-cloud/spanner|kafkajs|node-rdkafka" pegasus.x/` returned exit code 1 (0 matches).
4. **Terraform Infrastructure Discrepancy**:
   - File: `pegasus.x/infra/terraform/environments/production.tfvars` lines 20-28:
     ```hcl
     enable_spanner = false
     
     enable_managed_kafka = true
     kafka_cluster_id     = "pegasusx-prod-events"
     kafka_vcpu_count     = 3
     kafka_memory_bytes   = 17179869184
     ```
   - File: `pegasus.x/infra/terraform/cells/uz/cell.tfvars` lines 20-33:
     ```hcl
     enable_spanner                = false
     enable_managed_kafka = true
     kafka_cluster_id     = "pegasusx-uz-events"
     ```
   - File: `pegasus.x/infra/terraform/main.tf` line 54: `module "messaging"` provisions Google Managed Service for Apache Kafka when `enable_managed_kafka = true`.

---

### 1.2 Double-Entry Ledger Idempotency & Concurrency in `pegasusX`
1. **Spanner Schema & Indexes**:
   - `pegasusX/apps/backend-go/schema/spanner.ddl`:
     - `Idx_ArLedger_ByIdempotency`: `CREATE UNIQUE INDEX Idx_ArLedger_ByIdempotency ON ArLedgerEntries(IdempotencyKey);` (line 2616).
     - `Idx_OrderPaymentLegs_IdempotencyKey`: `CREATE UNIQUE INDEX Idx_OrderPaymentLegs_IdempotencyKey ON OrderPaymentLegs(IdempotencyKey);` (line 1821).
     - `Idx_PaymentLedgerEntries_GatewayTypeRef`: `CREATE UNIQUE NULL_FILTERED INDEX Idx_PaymentLedgerEntries_GatewayTypeRef ON PaymentLedgerEntries(Gateway, EntryType, ReferenceId);` (lines 682–683).
2. **Deterministic Repository Idempotency**:
   - `pegasusX/apps/backend-go/ar/service.go` lines 751–759:
     ```go
     iter := txn.Query(ctx, spanner.Statement{
         SQL:    `SELECT EntryId FROM ArLedgerEntries WHERE IdempotencyKey = @k LIMIT 1`,
         Params: map[string]any{"k": idempotencyKey},
     })
     _, qerr := iter.Next()
     iter.Stop()
     if qerr == nil {
         return nil // already applied
     }
     ```
   - `pegasusX/apps/backend-go/payment/repository_spanner.go` lines 558, 586, 608:
     - `LedgerEntryID: "pledger_session_" + s.SessionID, ReferenceID: s.SessionID`
     - `LedgerEntryID: "pledger_chargeback_" + c.ChargebackID, ReferenceID: c.ChargebackID`
     - `LedgerEntryID: "pledger_reversal_" + rev.ReversalID, ReferenceID: rev.ReversalID`
     Uses `spanner.InsertOrUpdateMap("PaymentLedgerEntries", ...)` with deterministic keys.
3. **Non-Deterministic Journal Entry Generation (Adversarial Flaw)**:
   - `pegasusX/apps/backend-go/payment/double_entry.go` lines 134–136:
     ```go
     je := &JournalEntry{
         EntryID:    fmt.Sprintf("jentry_order_%s_%d", orderID, time.Now().UTC().UnixNano()),
         OrderID:    orderID,
         ...
     ```
   - `pegasusX/apps/backend-go/payment/double_entry.go` line 213:
     ```go
     EntryID: fmt.Sprintf("jentry_settle_%s_%d", orderID, time.Now().UTC().UnixNano()),
     ```
   - `pegasusX/apps/backend-go/payment/double_entry.go` lines 268–278:
     ```go
     records = append(records, LedgerEntryRecord{
         LedgerEntryID: fmt.Sprintf("%s_p%d", je.EntryID, idx),
         ...
         ReferenceID:   je.EntryID,
     })
     ```
     Because `EntryID` and `ReferenceID` embed `time.Now().UTC().UnixNano()`, repeated executions/retries for the same order create distinct timestamps, producing distinct `ReferenceID`s that bypass Spanner's unique index `Idx_PaymentLedgerEntries_GatewayTypeRef(Gateway, EntryType, ReferenceId)`.

---

### 1.3 Currency Arithmetic & Floating-Point Usage in Financial Logic
The upstream worker handoff asserted:
> *"Strict 64-bit Integer Minor Unit Arithmetic (Tiyins): Zero float-based monetary calculations or storage in production domains. Therefore, financial accounting is mathematically exact with zero floating-point drift and strictly idempotent."*

Direct code inspection disproved this claim across multiple production domain files in `pegasus.x/backend`:

1. **Foreign Exchange Indexation & GL Postings**:
   - File: `pegasus.x/backend/internal/fleet/fx_index.go` lines 12, 20–22, 47, 58–69:
     ```go
     type CBURate struct {
         CurrencyCode  string    `json:"currency_code"`
         RateUZS       float64   `json:"rate_uzs"`
     }
     ...
     func CalculateFXAdjustment(orderID string, baseAmountTiyins int64, baseRate, lockRate float64) (*FXOrderAdjustment, error) {
         ...
         multiplier := lockRate / baseRate
         adjustedAmount := int64(math.Round(float64(baseAmountTiyins) * multiplier))
         variance := adjustedAmount - baseAmountTiyins
         ...
     ```
     Generates general ledger debit/credit postings to `4010:AR:RETAILER`, `9540:FX_GAIN`, and `9540:FX_LOSS` using amounts derived from `float64` division and `math.Round`.
2. **Three-Way Matching & Tax Calculations**:
   - File: `pegasus.x/backend/internal/matching/matching.go` lines 50, 93, 216–220:
     ```go
     type PurchaseOrderItem struct {
         ...
         VATRatePercent float64 `json:"vat_rate_percent"`
         ...
     }
     ...
     grQtyMilli := int64(grItem.ReceivedQty*1000.0 + 0.5)
     baseExpected := (grQtyMilli*poItem.UnitPriceMinor + 500) / 1000
     vatBps := int64(poItem.VATRatePercent*100.0 + 0.5)
     vatExpected := (baseExpected*vatBps + 5000) / 10000
     lineExpected := baseExpected + vatExpected
     ```
     VAT rates are declared as `float64` and converted using float arithmetic `poItem.VATRatePercent*100.0 + 0.5` instead of integer basis points (`int64`).
3. **Warehouse Shortage Claims**:
   - File: `pegasus.x/backend/internal/warehouse/service.go` line 604:
     ```go
     shortage := exp.ExpectedQty - actual
     claimAmount := int64(math.Round(shortage * float64(exp.UnitCostMinor)))
     ```
     Monetary claim amounts (`ClaimAmountMinor`) are computed by casting `exp.UnitCostMinor` (`int64`) to `float64`, multiplying by `shortage` (`float64`), and calling `math.Round`.
4. **Shuttle Inter-Warehouse Transport Costs**:
   - File: `pegasus.x/backend/internal/dispatch/shuttle.go` lines 73–88:
     ```go
     costPrimaryMinor := int64(math.Round(in.DistancePrimaryToRetailerKm*float64(in.FuelCostPerKmMinor))) +
         int64(math.Round(timePrimaryHours*float64(in.DriverHourlyCostMinor)))
     ...
     shuttleTotalCostMinor := int64(math.Round(in.DistanceInterWarehouseKm*float64(in.FuelCostPerKmMinor)*1.4)) +
         int64(math.Round(shuttleTripHours*float64(in.DriverHourlyCostMinor)*1.2))
     ```
     Direct `float64` arithmetic and `math.Round` on minor monetary costs (`FuelCostPerKmMinor`, `DriverHourlyCostMinor`).
5. **Fuel Theft Financial Loss Calculations**:
   - File: `pegasus.x/backend/internal/fleet/fuel_theft.go` lines 100, 129, 150, 205:
     ```go
     pricePerLiterMinor := int64(math.Round(float64(txn.TotalAmountMinor) / txn.LitersPurchased))
     loss := int64(math.Round(excessLiters * float64(pricePerLiterMinor)))
     lossMinor := int64(math.Round(dropLiters * float64(pricePerLiterMinor)))
     ```
6. **FSCM Credit Scoring Limit Allocation**:
   - File: `pegasus.x/backend/internal/fscm/scoring.go` line 103:
     ```go
     recommendedLimit := int64(math.Round(float64(metrics.Avg30dDeliveredVolumeMinor) * scoreFactor * tenureMultiplier))
     ```
7. **Smokecheck Financial Computations**:
   - File: `pegasus.x/backend/cmd/smokecheck/main.go` lines 343 & 406:
     ```go
     vatDelta := int64(math.Round(float64(subtotalDelta*int64(sku.VATPercent)) / 100.0))
     expectedBilledVAT := int64(math.Round(float64(expectedBilledSubtotal*12) / 100.0))
     ```

---

### 1.4 Test Suite & Type Check Execution
1. `go vet ./...` in `pegasus.x/backend`: Exited with code 0 (0 diagnostics).
2. `go test ./...` in `pegasus.x/backend`: Exited with code 0 (All unit tests and e2e integration suites passed in 19.3s).
3. `go test -v -count=1 ./outbox/... ./ar/... ./payment/...` in `pegasusX/apps/backend-go`: Exited with code 0 (All tests passed, including `TestDoubleEntry_*`, `TestRecordPaymentForOrderInTxn_Idempotent`, `TestRelayDrainOnceMarksPublishedOnSuccess`).

---

## 2. Logic Chain

1. **Integrity & Rigor of Worker Claims**:
   - Observation 1.3 demonstrates that multiple production files in `pegasus.x/backend` execute floating-point arithmetic (`float64`, `math.Round`) on currency values, prices, VAT rates, and ledger adjustments (`fleet/fx_index.go`, `matching/matching.go`, `warehouse/service.go`, `dispatch/shuttle.go`, `fleet/fuel_theft.go`, `fscm/scoring.go`).
   - The authoritative project directive in `ORIGINAL_REQUEST.md` (lines 225, 312, 338, 363) repeatedly mandates:
     - *"Zero floating-point arithmetic in financial calculations (strict 64-bit integer tiyins / minor units)."*
     - *"All financial amounts, prices, fees, margins, and taxes must be calculated and stored strictly in 64-bit integer tiyins (int64). Zero floats for currency."*
     - *"Audit all currency arithmetic to guarantee zero floating-point math, enforcing strict 64-bit integer tiyin minor units (int64)."*
   - The worker certified: *"Zero float-based monetary calculations or storage in production domains"* and *"financial accounting is mathematically exact with zero floating-point drift"*. This claim is factually false and reflects a self-certifying omission.

2. **Double-Entry Ledger Idempotency**:
   - Based on observation 1.2(1-2), `pegasusX` properly enforces unique index idempotency in Spanner for `ArLedgerEntries` (`Idx_ArLedger_ByIdempotency`) and `OrderPaymentLegs` (`Idx_OrderPaymentLegs_IdempotencyKey`), and uses deterministic keys for standard session/chargeback ledger mutations.
   - However, based on observation 1.2(3), `payment.BuildSplitTenderJournalEntry` and `payment.BuildSettlementJournalEntry` embed `time.Now().UTC().UnixNano()` into `EntryID` and `ReferenceID`. If these journal entries are created upon a retried API call or payment webhook retry, they will generate different `ReferenceID` values. The Spanner index `Idx_PaymentLedgerEntries_GatewayTypeRef(Gateway, EntryType, ReferenceId)` cannot detect or block the duplicate, violating retry idempotency.

3. **Infrastructure Boundary Drift**:
   - Based on observation 1.1(1-3), the application code in `pegasus.x/backend` is clean of Spanner and Kafka code.
   - However, based on observation 1.1(4), `pegasus.x/infra/terraform/environments/production.tfvars` and `cells/uz/cell.tfvars` have `enable_managed_kafka = true` and configure a 3-vCPU Managed Kafka cluster. This creates infrastructure drift and contradicts the Sovereign PostgreSQL + Redis architecture.

4. **Verdict Deduction**:
   - Under the Reviewer & Adversarial Critic protocol, any failure to uphold mandatory core invariants (specifically zero floating-point currency arithmetic and retry idempotency) requires a verdict of `REQUEST_CHANGES`.

---

## 3. Caveats

1. **External Infrastructure Availability**: Automated tests were run against local mock/database containers; actual Spanner and Kafka clusters were not connected in the test environment (standard CI behavior).
2. **Quantity vs Price Floating Point**: In `matching.go` and `warehouse/service.go`, quantities (`ReceivedQty`, `ShortageQty`) are floating-point to support fractional catch-weight kilograms/liters. While fractional units may be valid for weights, converting monetary unit prices (`int64`) to `float64` for multiplication and rounding is a violation of strict integer financial arithmetic (fixed-point milli-units or integer basis points should be used).
3. **No Code Edits Performed**: In accordance with the Reviewer role constraints, no source code files were altered by this reviewer.

---

## 4. Conclusion

**Verdict: REQUEST_CHANGES**

Milestone M2 (Requirement R2) cannot be approved in its current state due to the following findings:

### Critical Finding 1: Floating-Point Financial Calculations in `pegasus.x`
- **Location**:
  - `pegasus.x/backend/internal/fleet/fx_index.go` (lines 12, 20–22, 58–69)
  - `pegasus.x/backend/internal/matching/matching.go` (lines 50, 93, 216–220)
  - `pegasus.x/backend/internal/warehouse/service.go` (line 604)
  - `pegasus.x/backend/internal/dispatch/shuttle.go` (lines 73–88)
  - `pegasus.x/backend/internal/fleet/fuel_theft.go` (lines 100, 129, 150, 205)
  - `pegasus.x/backend/internal/fscm/scoring.go` (line 103)
  - `pegasus.x/backend/cmd/smokecheck/main.go` (lines 343, 406)
- **Problem**: Monetary values, FX adjustments, VAT rates, and shortage claims use `float64` arithmetic and `math.Round`, directly violating the requirement of 100% 64-bit integer minor unit arithmetic (tiyins).
- **Remediation**:
  - Refactor `fx_index.go` to use integer basis points or scaled integer math (such as `math/big.Int` or `Scale = 100_000_000`, following `pegasusX/apps/backend-go/fxrates/convert.go`).
  - Refactor `matching.go` to store `VATRateBps int64` instead of `float64`, and compute expected line totals using integer arithmetic.
  - Refactor `warehouse/service.go`, `shuttle.go`, and `fuel_theft.go` to use integer milli-units for fractional quantities/distances so monetary products remain integer-only.

### Major Finding 2: Non-Deterministic `EntryID` in `pegasusX` Double-Entry Ledger
- **Location**: `pegasusX/apps/backend-go/payment/double_entry.go` (lines 135, 213, 278)
- **Problem**: `BuildSplitTenderJournalEntry` and `BuildSettlementJournalEntry` embed `time.Now().UTC().UnixNano()` into `EntryID` and `ReferenceID`. Retries of the same transaction will generate differing reference IDs, circumventing the Spanner unique index `Idx_PaymentLedgerEntries_GatewayTypeRef(Gateway, EntryType, ReferenceId)`.
- **Remediation**: Pass a deterministic idempotency key or transaction ID into `BuildSplitTenderJournalEntry` and `BuildSettlementJournalEntry` (e.g. `fmt.Sprintf("jentry_order_%s", orderID)` or a caller-supplied `idempotencyKey`).

### Minor Finding 3: Managed Kafka Enabled in `pegasus.x` Terraform Configs
- **Location**: `pegasus.x/infra/terraform/environments/production.tfvars` (line 25), `cells/uz/cell.tfvars` (line 30)
- **Problem**: `enable_managed_kafka = true` is configured in production and uz cell tfvars, contrary to the sovereign Redis Streams architecture.
- **Remediation**: Set `enable_managed_kafka = false` across all `pegasus.x` terraform variable files.

---

## 5. Verification Method

To independently verify these findings, execute the following commands:

```bash
# 1. Verify floating point currency calculations in pegasus.x:
grep -n -C 5 "math.Round" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/fleet/fx_index.go
grep -n -C 5 "VATRatePercent float64" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/matching/matching.go
grep -n -C 5 "claimAmount :=" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/warehouse/service.go
grep -n -C 5 "costPrimaryMinor :=" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend/internal/dispatch/shuttle.go

# 2. Verify non-deterministic timestamp generation in double_entry.go:
grep -n -C 3 "UnixNano()" /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payment/double_entry.go

# 3. Verify Kafka enablement in pegasus.x terraform:
grep -n "enable_managed_kafka" /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/infra/terraform/environments/*.tfvars /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/infra/terraform/cells/*/*.tfvars

# 4. Verify test suite execution:
cd /Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend && go test ./...
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go && go test -v -count=1 ./outbox/... ./ar/... ./payment/...
```
