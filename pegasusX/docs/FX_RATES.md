# FX rates (theatre #13 Wave 1)

**Status:** Partial → Wired for rates + payment currency mismatch gate.  
**Not claimed:** multi-currency marketplace settlement ledger, Airwallex live FX, client currency pickers, ISO-4217 minor-unit tables beyond the 2-decimal assumption.

Operating currency remains **UZS** (`SEED_SUPPLIER_CURRENCY`). FX exists so explicit conversion can run when rates are present — never as silent 1:1 across currencies.

Apply migration `apps/backend-go/schema/migrations/20260806_fx_rates.ddl` (`FxRates`).

## Model

| Field | Meaning |
|-------|---------|
| `BaseCurrency` / `QuoteCurrency` | ISO-4217 |
| `RateScaled` / `Scale` | Quote-per-base as integer (`Scale` default `1e8`) |
| `EffectiveAt` | As-of lookup picks latest `EffectiveAt <= at` |
| `Source` | `SEED` / `ADMIN` / `MANUAL` |

## Convert API

Package `apps/backend-go/fxrates`:

- `ConvertMinor(ctx, from, to, amountMinor, at)` — same currency → identity; missing rate → `fx_rate_missing` (fail closed); optional inverse when only quote→base exists (`AllowInverse=true` by default).
- `AssertSameCurrency` — used by payment checkout / chargeback / webhook stamping.

## Seed + admin

Bootstrap seeds identity `{operating}/{operating}` (default `UZS/UZS`). Optional `FX_SEED_USD_UZS_SCALED` seeds `USD/UZS` for tests.

| Method | Path | Auth |
|--------|------|------|
| GET | `/v1/admin/fx-rates` | Admin JWT — list latest per pair |
| PUT | `/v1/admin/fx-rates` | Admin JWT — upsert manual/admin rate |

No portal UI in Wave 1 (ops/docs only).

## Payment gate

On card checkout init (and chargeback / webhook paths that stamp currency from the request):

1. Empty request currency → order currency
2. Non-empty request currency ≠ order/session currency → **422** `currency_mismatch` (no silent overwrite)

## SSMR markers

| Marker | Meaning |
|--------|---------|
| `PX_E2E_FX_RATE_SEEDED_OK` / `_SKIPPED` | Identity rate visible via admin list |
| `PX_E2E_CURRENCY_MISMATCH_DENIED` / `_SKIPPED` | Card checkout with wrong currency rejected |

## Residual (honest)

- Multi-currency settlement ledger / AR aging across currencies
- Airwallex FX (redirect stub only today)
- Client-facing currency pickers (orders still hardcode supplier currency)
- Full ISO-4217 minor-unit table (Wave 1 assumes 2-decimal minors)
