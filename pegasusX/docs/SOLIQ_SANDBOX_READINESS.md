# L5 Soliq OFD — Sandbox Readiness (Delivery-only)

## Status

- Adapter path exists: `FISCAL_PROVIDER=MY_SOLIQ` in `apps/backend-go/order/fiscal_provider.go`
- SSMR keeps `FISCAL_PROVIDER=PEGASUS` until secrets land
- **POS OFD is out of scope** for this program (local receipt number only)

## GSM slots (owner)

| Secret / env | Purpose |
|--------------|---------|
| `FISCAL_MY_SOLIQ_BASE_URL` | Sandbox API base |
| `FISCAL_MY_SOLIQ_API_KEY` | Auth |
| `FISCAL_MY_SOLIQ_TIN` | Taxpayer STIR |
| `FISCAL_MY_SOLIQ_PATH` | Optional path (default `/v1/receipts`) |
| `FISCAL_MY_SOLIQ_TIMEOUT_MS` | Optional (default 8000) |

Misconfigured MY_SOLIQ **hard-fails** CreateReceipt (by design).

## Enable path (canary)

1. Load secrets into GSM + ESO.
2. Set `FISCAL_PROVIDER=MY_SOLIQ` on a canary worker only.
3. Run delivery fiscal path; expect marker `PX_E2E_SOLIQ_SANDBOX_OK` when wired.
4. Rollback: `FISCAL_PROVIDER=PEGASUS`.

## Marker

Until credentials exist, smokecheck may print `PX_E2E_SOLIQ_SANDBOX_SKIPPED`.
