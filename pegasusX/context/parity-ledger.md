# Parity ledger (intentional divergences)

## Claims / chargebacks (2026-07-29)

| Item | Status | Notes |
|------|--------|--------|
| Supplier portal claims queue + settlement modes | Wired | `@pegasusx/types` + `@pegasusx/api-client` |
| Supplier Android / iOS claims queue + settlement modes | Wired | Same backend endpoints as portal |
| Supplier claim-chargebacks ledger (all 3 clients) | Wired | `GET /v1/supplier/claim-chargebacks` |
| Claim chargebacks live WS push | **Deferred** | Poll / pull-to-refresh only; CLAIM_* events still fan out for other inbox surfaces |
| Retailer file-claim media | Prior work | Camera both platforms; not part of this close |
| Manual PSP chargebacks (payment/chargeback) | Unchanged | Separate from logistics claim chargebacks |

See also: `docs/CLAIM_ROLE_ROW.md`.
