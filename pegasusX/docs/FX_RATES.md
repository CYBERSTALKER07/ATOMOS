# FX rates (theatre #13)

**Status:** Wave 1 Wired (rates + payment mismatch). Wave 2 Wired (billing GMV ConvertMinor + settlement operating rollup + portal FX UI). Wave 2+ Wired (flag-gated order currency allowlist + desktop/Android/iOS picker).  
**Not claimed:** multi-currency AR/credit aging ledger, Airwallex live FX, ISO-4217 minor-unit tables beyond the 2-decimal assumption.

Operating currency remains **UZS** (`SEED_SUPPLIER_CURRENCY`). FX exists so explicit conversion can run when rates are present — never as silent 1:1 across currencies.

Apply migration `apps/backend-go/schema/migrations/20260806_fx_rates.ddl` (`FxRates`).

## Model

| Field | Meaning |
|-------|---------|
| `BaseCurrency` / `QuoteCurrency` | ISO-4217 |
| `RateScaled` / `Scale` | Quote-per-base as integer (`Scale` default `1e8`) |
| `EffectiveAt` | As-of lookup picks latest `EffectiveAt <= at` |
| `Source` | `SEED` / `ADMIN` / `MANUAL` / `SSMR` |

## Convert API

Package `apps/backend-go/fxrates`:

- `ConvertMinor(ctx, from, to, amountMinor, at)` — same currency → identity; missing rate → `fx_rate_missing` (fail closed); optional inverse when only quote→base exists (`AllowInverse=true` by default).
- `AssertSameCurrency` — used by payment checkout / chargeback / webhook stamping.

## Seed + admin / supplier

Bootstrap seeds identity `{operating}/{operating}` (default `UZS/UZS`). Optional `FX_SEED_USD_UZS_SCALED` seeds `USD/UZS` for tests.

| Method | Path | Auth |
|--------|------|------|
| GET | `/v1/admin/fx-rates` | Admin JWT — list latest per pair |
| PUT | `/v1/admin/fx-rates` | Admin JWT — upsert manual/admin rate |
| GET | `/v1/supplier/fx-rates` | Supplier portal ADMIN session — read-only list |

Portal UI: `/settings/fx-rates` (list + upsert human rate → `rate_scaled`).

## Payment gate (Wave 1)

On card checkout init (and chargeback / webhook paths that stamp currency from the request):

1. Empty request currency → order currency
2. Non-empty request currency ≠ order/session currency → **422** `currency_mismatch` (no silent overwrite)

## Wave 2 — settlement + metering

**Billing GMV:** `ORDER_FINALIZED` consumer converts `total.currency` → operating via `ConvertMinor` before writing `BillingSupplierMeters`. Missing rate → **skip** meter write (do not mix currencies into `CurrentValue`).

**Settlement authority:** `GET /v1/payment/settlement/authority` keeps native `totals_by_currency` and adds display-only:

- `operating_currency`
- `operating_currency_total_minor`
- `operating_conversion_partial` (true if any group could not convert)

Ledger **writes** stay in native currency.

## Wave 2+ — order currency picker (flag-gated)

Default **off**. When enabled, retailers may stamp an allowlisted ISO-4217 on create / unified checkout.

| Env | Meaning |
|-----|---------|
| `ORDER_CURRENCY_PICKER_ENABLED` | `true`/`1`/`on` to honour request `currency` |
| `ORDER_CURRENCY_ALLOWLIST` | Comma-separated ISO-4217; always includes operating (`SEED_SUPPLIER_CURRENCY`) |

| Method | Path | Auth |
|--------|------|------|
| GET | `/v1/order/currencies` | Retailer — `{ enabled, operating_currency, allowlist }` |

Behaviour:

1. Flag **off** → ignore client currency; stamp operating (today’s behaviour)
2. Flag **on**, empty currency → operating
3. Flag **on**, non-allowlisted → **422** `currency_not_allowed`

Clients (desktop / Android / iOS) show a picker only when `enabled` is true. Catalog prices and cart line sync remain in operating currency; this does **not** convert line unit prices.

## SSMR markers

| Marker | Meaning |
|--------|---------|
| `PX_E2E_FX_RATE_SEEDED_OK` / `_SKIPPED` | Identity rate visible via admin list |
| `PX_E2E_CURRENCY_MISMATCH_DENIED` / `_SKIPPED` | Card checkout with wrong currency rejected |
| `PX_E2E_FX_SETTLEMENT_CONVERT_OK` / `_SKIPPED` | Settlement authority includes `operating_currency_total_minor` |
| `PX_E2E_ORDER_CURRENCY_PICKER_OK` / `_SKIPPED` | Currencies endpoint + allowlist shape (deny soft when create blocked earlier) |

## Residual (honest)

- Multi-currency AR / credit aging ledger
- Airwallex FX (redirect stub only today)
- Full ISO-4217 minor-unit table (still assumes 2-decimal minors)
