# Soliq / EHF Integration (Regulatory)

## Goal

On COMPLETED (or audited force-complete):

1. Build EHF JSON for the fiscal document.  
2. Sign with EDS (electronic digital signature).  
3. Submit to SoliqOnline or authorized OFD operator.  
4. Track clearance + buyer acceptance window (e.g. 10 days).  
5. Feed VAT reconciliation.

## Relationship to ADR-009

```
Payment capture → FISCALIZING
  → CreateReceipt (provider: MY_SOLIQ | PEGASUS | …)
  → SUCCESS → COMPLETED + store receipt ids
  → FAILED → FISCAL_FAILED → retry or admin force
```

EHF success should be required for **final ledger close** under production fiscal policy; PEGASUS remains interim commercial receipt path.

## How it works (workers)

```
Outbox FISCAL_RECEIPT_REQUESTED
  → order mutator ApplyFiscalWorkerResult
  → provider.CreateReceipt
  → OrderFiscalReceipts row SUCCESS|FAILED
  → events FISCAL_RECEIPT_SUCCEEDED|FAILED
```

## Edge cases

| Case | Regulatory rule |
|------|-----------------|
| Soliq timeout | Retry with backoff; do not COMPLETED until policy allows |
| Force-complete | Admin-only; still emit fiscal artifact attempt + audit |
| Buyer rejects EHF | Status path + notification; may reverse settlement policy |
| Multi-VAT order | Line-level tax codes from regime version |

## Env / secrets (future)

- Soliq API base, client certs/EDS keys in GSM  
- `FISCAL_PROVIDER=MY_SOLIQ` when ready  
- See `artifacts/receipts-multi-provider.md`  
