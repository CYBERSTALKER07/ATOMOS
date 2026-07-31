# Multi-layer receipts (no Soliq required for now)

**Date:** 2026-08-01  
**Decision:** Ship **Pegasus branded platform receipts** as the ADR-009 hard-gate path.  
Soliq/OFD and Global Pay **payment receipts** plug in later without rewriting order lifecycle.

## Layers

| Layer | Provider | When | Blocks COMPLETED? | Legal class |
|-------|----------|------|-------------------|-------------|
| **1. Platform commercial** | `PEGASUS` | **Now (default)** | **Yes** | `platform_receipt` (`tax_ofd: false`) |
| **2. Payment provider** | `GLOBAL_PAY` | When GP receipt API creds arrive | **No** (best-effort attach) | payment receipt under `payment_receipt` in payload |
| **3. Tax OFD** | `MY_SOLIQ` | When Soliq sandbox/prod APIs arrive | Yes (if selected as primary) | `tax_ofd_receipt` |

## Env

```bash
# Product default (SSMR / staging / prod until Soliq)
FISCAL_PROVIDER=PEGASUS
PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app

# Optional branding overrides for Pegasus settlement receipts
# PEGASUS_COMPANY_NAME=PegasusX
# PEGASUS_ISSUER_TIN=

# Optional SSMR-only fail hooks for fiscal e2e (amount=13, order id contains fiscal-fail)
# Enabled automatically when PEGASUSX_ENV=ssmr
# FISCAL_PEGASUS_SSMR_HOOKS=true

# Later — Global Pay payment receipts (secondary; does not block COMPLETED)
# FISCAL_GLOBAL_PAY_RECEIPT_ENABLED=true
# FISCAL_GLOBAL_PAY_RECEIPT_BASE_URL=https://…
# FISCAL_GLOBAL_PAY_RECEIPT_API_KEY=…
# FISCAL_GLOBAL_PAY_RECEIPT_PATH=/v1/receipts

# Later — Soliq tax OFD (primary hard-gate replaces PEGASUS if you switch)
# FISCAL_PROVIDER=MY_SOLIQ
# FISCAL_MY_SOLIQ_BASE_URL=…
# FISCAL_MY_SOLIQ_API_KEY=…
# FISCAL_MY_SOLIQ_TIN=…
```

## What clients get today

1. Cash/card capture → order `FISCALIZING` → worker issues **`PX-RCPT-{attemptId}`**.
2. Row stored in `OrderFiscalReceipts` with `Provider=PEGASUS` (branding + country + line items in payload).
3. QR / deep link: `{PUBLIC_BASE_URL}/v1/platform/receipts/{receiptId}?format=html` (branded page).
4. Public GET:
   - default / `?format=json` → commercial JSON (`tax_ofd: false`, includes `html_url` / `pdf_url` / line items)
   - `?format=html` or `Accept: text/html` → Pegasus branded HTML receipt
   - `?format=pdf` → downloadable PDF
5. Authenticated party copies (same document, `party_copy` label):
   - `GET /v1/retailer/orders/{orderId}/receipt`
   - `GET /v1/supplier/orders/{orderId}/receipt`
   - `GET /v1/warehouse/orders/{orderId}/receipt`
6. Role UIs: retailer tracking (desktop + mobile), supplier order/compliance, warehouse order detail — view HTML / download PDF.

Country layout: `UZ` (default from UZS) and `KZ` (from KZT) via `order/receipt_layout.go`.

## What Boss still provides (later)

| Source | Need | Used for |
|--------|------|----------|
| **Global Pay** | Receipt API base URL + API key (+ path if non-default) | Layer 2 payment receipts |
| **Soliq / OFD** | Sandbox base URL + API key + TIN + EDS | Layer 3 tax fiscalization |
| **DNS** (Step 11) | `api-ssmr.pegasusx.app` → LB IP | Trusted HTTPS QR links |

Soliq adapter code remains in-tree (`MySoliqProvider` / `soliq` client) and env-commented — flip `FISCAL_PROVIDER=MY_SOLIQ` when credentials land.

## Code map

| File | Role |
|------|------|
| `order/fiscal_provider_pegasus.go` | Platform receipts + branding payload |
| `order/receipt_document.go` | HTML/PDF renderer |
| `order/receipt_layout.go` | Country labels / legal footer |
| `order/receipt_templates/receipt.html.tmpl` | Branded HTML |
| `order/receipt_assets/logo.svg` | Company mark |
| `order/fiscal_provider_globalpay.go` | GP HTTP adapter + multi-wrap |
| `order/fiscal_provider.go` | `ProviderFromEnv` selection (incl. deferred Soliq) |
| `order/receipt_handlers.go` | Public + role receipt handlers |
| `order/repository_spanner.go` | `GetFiscalByReceiptID` |

## Production target (your product plan)

1. **Pegasus** branded settlement receipt always (settlement / app QR / ops) — **Wired**.  
2. **Global Pay** payment receipt when their API is live.  
3. **Soliq** tax receipt when OFD sandbox/prod is approved — either switch primary to `MY_SOLIQ` or add as additional leg later.

Until (2) and (3), hard-gate remains **PEGASUS only** — orders complete with platform receipts (`tax_ofd: false`).
