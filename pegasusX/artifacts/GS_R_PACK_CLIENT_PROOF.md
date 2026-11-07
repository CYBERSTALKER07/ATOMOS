# GS-R pack client bind proof

**Date:** 2026-08-16
**Method:** structural greps + supplier-portal vitest. No checkout_reads_this flip. No terraform apply.

| Claim | Evidence | Result |
|-------|----------|--------|
| Session pack bind | `GET /v1/auth/session` via `fetchAuthSession` / `MarketPackBinder` | PASS |
| Splash fields | currency + receipts (Soliq / commercial) on portal shells + native homes | PASS |
| Native cell pin | `CellPinInterceptor` + iOS `pinApiBaseUrl` (localhost stays bootstrap) | PASS |
| No web-only currency | retailer/supplier formatters + checkout read pack currency | PASS (bind). Deep POS UZS labels remain continuous leftover |
| Flag | `checkout_reads_this` still false | PASS |

Leftover: linguistic i18n; remaining hardcoded UZS on deep screens; maps SDK swap. Next code = GS-P only when asked.
