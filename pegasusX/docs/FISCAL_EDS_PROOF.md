# Fiscal EDS proof (MY_SOLIQ / Soliq OFD)

**Date:** 2026-08-12  
**Gap:** P1-7  
**Status:** Contract sign→submit→poll proven in CI/SSMR. Live operator sandbox remains opt-in; production EDS requires E-IMZO PKCS#12 procurement.

## What “proven” means for the prod bar

| Layer | Proof | How |
|-------|--------|-----|
| Sign | Canonical EHF body + attached signature | `fiscal.AttachSignature` + `DevHMACSigner` (non-prod) / `PKCS12Signer` (prod) |
| Submit | POST `/v1/ehf/submit` with Bearer + Idempotency-Key | `order.MySoliqProvider.CreateReceipt` → `soliq.Client.Submit` |
| Poll | GET `/v1/ehf/{id}/status` | `soliq.Client.CheckStatus` |
| Fail-closed | No unsigned receipts; no `dev-hmac` in prod/ssmr | `fiscal.SignerFromEnv` |

CI evidence:

- `order.TestMySoliqContract_SignSubmitPoll`
- `order.TestMySoliqContract_SubmitSuccess` (+ golden envelope)
- SSMR marker `PX_E2E_SOLIQ_CONTRACT_OK` from `cmd/ssmr-smokecheck` (`e2e_soliq.go`)

## Live operator sandbox (optional)

```bash
export FISCAL_PROVIDER=MY_SOLIQ
export FISCAL_MY_SOLIQ_BASE_URL=...
export FISCAL_MY_SOLIQ_API_KEY=...
export FISCAL_MY_SOLIQ_TIN=...
export FISCAL_MY_SOLIQ_SIGNER=pkcs12   # or dev-hmac only outside prod/ssmr
export FISCAL_MY_SOLIQ_LIVE_PROOF=1   # required — creds alone do not print OK
go run ./cmd/ssmr-smokecheck e2e
# expect: PX_E2E_SOLIQ_SANDBOX_LIVE_OK
```

Without `LIVE_PROOF=1`, present credentials print `PX_E2E_SOLIQ_SANDBOX_CREDS_PRESENT_LIVE_PROOF_OFF` so env-presence cannot be mistaken for a round-trip.

## Owner gate (production EDS)

Legal Soliq receipts need a real E-IMZO key:

- `FISCAL_MY_SOLIQ_SIGNER=pkcs12`
- `FISCAL_MY_SOLIQ_PKCS12_FILE` / `FISCAL_MY_SOLIQ_PKCS12_PASSWORD`

Until procurement lands, production must not enable `FISCAL_PROVIDER=MY_SOLIQ` with `dev-hmac` (construction fails in `production`/`prod`/`ssmr`).

## Non-goals

- Replacing PEGASUS commercial receipts for markets that do not require Soliq OFD
- Claiming live Soliq ACCEPTED without `LIVE_PROOF=1` and operator credentials
