# 00 — Inventory — Phase G1 Money & law

**Status:** SEEDED at Phase 0 — expand before coding G1.A  
**Open IDs:** G1-A1, G1-A2, G1-B1, G1-B2, G1-C1–C4, G1-D1, G1-D2  

## Current truth (code anchors)

| Area | Path | Finding |
|------|------|---------|
| Credit leave AR open | `order/driver_edges.go` | `OpenFromCreditLeaveInTxn` same RW txn (B6) ✅ |
| Cash AR pay-down | `order/service.go` ~2188–2197 | Post-commit `RecordPaymentForOrder`; log on error ❌ G1-A1 |
| Shop-closed AR | `order/worker_shop_closed.go` | Post-commit open but returns error (retry) |
| Fiscal default | `order/fiscal_provider.go`, `.env.example` | `FISCAL_PROVIDER=PEGASUS` |
| MY_SOLIQ | `order/fiscal_*.go`, `fiscal/signer_pkcs12.go` | Code exists; not default |
| State patch | `driver/mobile_compat.go` | Always 501 |
| Mid-delivery | `order/delivery_handshake.go` | `not_implemented` error |
| Negotiation | `order/negotiation_disabled.go` | 410 unless flag |
| Credit scores | `credit/repository.go` GetScoresForRetailers | Empty map stub |
| Payout | `payout/rail.go` | BankFileRail; ErrNoLiveRail if live |
| Collect cash key | `order/service.go` | `cash-`+orderID stable (B1) ✅ |

## Flags

| Flag | Default | Notes |
|------|---------|-------|
| FISCAL_PROVIDER | PEGASUS | Tax markets need MY_SOLIQ profile |
| QUANTITY_NEGOTIATION_ENABLED | false | 410 API |
| AR_DUNNING_ENABLED | false base CM | Staging/SSMR may true |

## Clients to audit in G1.C

- `driver-app-android` / `driver-app-ios` — state patch, mid-delivery, negotiation  
- `supplier-portal` / natives — negotiation list, credit scores, payout  
- `retailer-*` — AR views after pay-down co-atomic  

## Greps to re-run at G1 start

See `PROOF_HARNESS.md` § money/law.

## Design order (do not reorder)

1. **G1.A** AR pay-down / ClearBalance co-atomic  
2. **G1.B** Fiscal profiles (code + config docs)  
3. **G1.C** Theatre kill / mid-delivery decision  
4. **G1.D** Payout honesty + FCM loudness  
