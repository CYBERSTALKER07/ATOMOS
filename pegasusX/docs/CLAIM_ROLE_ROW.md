# Claims role-row (logistics OS&D + chargebacks)

Status: **wired** on supplier portal + Android + iOS (2026-07-29 ecosystem close).

## Contract owners

| Surface | Location |
|---------|----------|
| DTOs | `packages/types` — `Claim`, `ApproveClaimRequest`, `ClaimSettlementMode`, `CLAIM_SETTLEMENT_MODES`, `ClaimChargebacksResponse`, … |
| HTTP client | `packages/api-client` — `listSupplierClaims`, `approveClaim`, `rejectClaim`, `listSupplierClaimChargebacks` |
| Backend | `orderroutes` (`GET /v1/supplier/claims`, `POST /v1/claims/{id}/approve|reject`), `paymentroutes` (`GET /v1/supplier/claim-chargebacks`) |

## Supplier role row

| Client | Claims queue (list/approve/reject + settlement_mode) | Claim chargebacks ledger |
|--------|------------------------------------------------------|---------------------------|
| Portal | `/exceptions/claims` via `@pegasusx/api-client` | `/chargebacks/claims` |
| Android | `ClaimsScreen` · routes `claims` | `ClaimChargebacksScreen` · `claim_chargebacks` |
| iOS | `ClaimsView` · section `.claims` | `ClaimChargebacksView` · `.claimChargebacks` |

Settlement modes (all three clients):

- `LEDGER_ONLY` (default, skip gateway)
- `STORE_CREDIT`
- `GATEWAY_REFUND` (only path that does not force `skip_gateway_refund`)

## Realtime

- Approve/reject still use prior backend outbox (`CLAIM_*` events).
- Claim chargebacks UIs are **HTTP refresh** (load/filter/pull-to-refresh), not WS ledger streams.
- Documented: acceptable for finance desk; optional future silent WS refresh.

## SSMR

Backend markers in `cmd/ssmr-smokecheck/e2e_claims.go` (file → approve → claim-chargebacks list).

## Explicit non-goals (this close)

- Soliq / OFD
- Supplier mobile **manual PSP** chargebacks UI already existed; this pass adds **logistics claim** surfaces only
- Native Quicktype `Generated/` stubs not required (apps use hand-aligned models)

## Parity notes

Previously portal-only settlement_mode + claim-chargebacks page (`46a0f321`) was a role-row gap under the ecosystem alignment rule. Closed by shared contracts + Android/iOS screens.
