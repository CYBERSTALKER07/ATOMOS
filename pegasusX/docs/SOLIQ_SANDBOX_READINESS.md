# L5 Soliq OFD — Sandbox Readiness (Delivery-only)

## Status

- Adapter: `FISCAL_PROVIDER=MY_SOLIQ` → `order.MySoliqProvider` + `soliq.Client`
- Contract proof (always in smokecheck / unit tests): `PX_E2E_SOLIQ_CONTRACT_OK` (in-process mock sign→submit→poll)
- Live sandbox: opt-in only — full detail in [`FISCAL_EDS_PROOF.md`](./FISCAL_EDS_PROOF.md)
- SSMR keeps `FISCAL_PROVIDER=PEGASUS` until secrets + EDS land
- **POS OFD is out of scope** (local receipt number only)

## GSM / env (owner)

| Secret / env | Purpose |
|--------------|---------|
| `FISCAL_MY_SOLIQ_BASE_URL` | Sandbox/prod API base |
| `FISCAL_MY_SOLIQ_API_KEY` | Bearer |
| `FISCAL_MY_SOLIQ_TIN` | STIR |
| `FISCAL_MY_SOLIQ_SIGNER` | `pkcs12` (prod/ssmr) or `dev-hmac` (non-prod only) |
| `FISCAL_MY_SOLIQ_PKCS12_FILE` / `…_PASSWORD` | E-IMZO container (prod) |
| `FISCAL_MY_SOLIQ_SIGN_KEY` | dev-hmac key (≥16 bytes, non-prod) |
| `FISCAL_MY_SOLIQ_LIVE_PROOF=1` | Required for live round-trip marker |
| `FISCAL_MY_SOLIQ_TIMEOUT_MS` | Optional (default 8000) |

Note: `FISCAL_MY_SOLIQ_PATH` is **currently unused** if set; the client posts to `{BASE}/v1/ehf/submit` and GETs `{BASE}/v1/ehf/{id}/status`.

Misconfigured MY_SOLIQ **hard-fails** CreateReceipt (by design). `dev-hmac` is **forbidden** in `production` / `prod` / `ssmr`.

## Enable path (canary)

1. Load secrets into GSM + ESO; procure E-IMZO PKCS#12 for prod/ssmr.
2. Set `FISCAL_PROVIDER=MY_SOLIQ` + `FISCAL_MY_SOLIQ_SIGNER=pkcs12` on a canary worker only.
3. For live proof: `FISCAL_MY_SOLIQ_LIVE_PROOF=1` and run smokecheck e2e.
4. Rollback: `FISCAL_PROVIDER=PEGASUS`.

## Markers

| Condition | Marker |
|-----------|--------|
| No MY_SOLIQ / incomplete creds | `PX_E2E_SOLIQ_SANDBOX_SKIPPED` |
| Creds present, `LIVE_PROOF` off | `PX_E2E_SOLIQ_SANDBOX_CREDS_PRESENT_LIVE_PROOF_OFF` |
| Live success | `PX_E2E_SOLIQ_SANDBOX_LIVE_OK` |
| Contract (CI/SSMR always) | `PX_E2E_SOLIQ_CONTRACT_OK` |
