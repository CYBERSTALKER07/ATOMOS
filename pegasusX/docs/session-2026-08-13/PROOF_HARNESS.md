# Proof harness — contract greps + standard tests
> **POINT-IN-TIME SNAPSHOT (2026-08-13) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


Run from `pegasusX/apps/backend-go` unless noted.  
Baselines recorded at Phase 0 kickoff (2026-08-13). Counts **drift** — interpret, do not treat raw count=0 as always required.

## 1. Build

```bash
cd apps/backend-go
go build -o /tmp/pegasusx-backend .
```

## 2. Money / law packages (G1)

```bash
go test ./order/ ./ar/ ./payment/ ./payout/ ./claims/ ./credit/ ./driver/ ./kafka/ -count=1
```

## 3. Logistics packages (G2)

```bash
go test ./stocklots/ ./warehouse/ ./payload/ ./factory/ ./returns/ ./manifest/ -count=1
```

## 4. Grep recipes (investigation, not always zero)

### Post-commit AR / money fail-open smells

```bash
rg -n 'AR pay-down|OpenFromCreditLeave\(|ClearBalance|log\.(Error|Warn).*AR' order/ ar/ credit/ --glob '*.go'
```

### Silent success stubs (should be 503/501, not 200)

```bash
rg -n 'StatusOK.*departed|status.:.cancelled|status.:.created' --glob '*compat*.go' --glob '*core_handlers*.go' || true
rg -n 'order_service_unwired|depart_unwired|not_implemented|StatusServiceUnavailable|StatusNotImplemented' driver/ retailer/ --glob '*.go' | head
```

### Emit nil / BufferWrite without outbox (sample)

```bash
rg -n 'emit.*nil|EmitJSON.*nil' --glob '*.go' | head -40
rg -n 'BufferWrite' stocklots/ --glob '*.go' | wc -l
rg -n 'outbox\.|EmitJSON|OutboxEvents' stocklots/ --glob '*.go' | wc -l
```

### Body supplier_id trust

```bash
rg -n 'body\.SupplierID|json:"supplier_id"' payout/ partner/ --glob '*handler*.go' | head
```

### Fiscal default

```bash
rg -n 'FISCAL_PROVIDER' ../../.env.example ../../infra/k8s --glob '*.{yaml,yml,example}' | head
```

## 5. Client theatre (from monorepo root)

```bash
rg -n 'onClick\s*=\s*\{\s*\}' apps/retailer-app-android --glob '*.kt' | head
rg -n 'orders/.*/state|update-order-during-delivery|feature_disabled' apps/driver-app-* apps/supplier-* --glob '*.{kt,swift,ts,tsx}' | head
```

## 6. Phase exit rule

A phase may exit only when:

1. Touched packages `go test` green  
2. `go build` green  
3. Greps for **that phase’s** residual class are clean or explained in `04_PROOF.md`  
4. Cross-role matrix alignment checkboxes done  
5. `SCORECARD.md` + `GAP_LEDGER.md` updated  

## Baseline snapshot (Phase 0)

| Check | Result (kickoff) |
|-------|------------------|
| `go build` backend | expected green (verify when running G1) |
| B1–B7 packages | previously green |
| Cash AR pay-down | still post-commit log (G1-A1 OPEN) |
| FISCAL_PROVIDER default | PEGASUS |
| WMS pick/cycle/cold | false in `.env.example` |
