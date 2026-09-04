# Payout bank-file runbook (G1.D)

**Product truth:** the default payout rail is **bank-file**, not a live bank API.  
`live=true` dispatch returns **`no_live_rail`** (HTTP 409). Money leaves the platform only after a human confirms bank settlement via **MarkPaid**.

## Workflow

1. **Generate** — `POST /v1/supplier/payouts/batches`  
   Body: `{ "period_start": "YYYY-MM-DD", "period_end": "YYYY-MM-DD", "idempotency_key": "..." }`  
   Response includes `batch` + `rail` honesty metadata.

2. **Export CSV** — `POST /v1/supplier/payouts/batches/{batchID}/export`  
   Downloads bank instruction CSV; batch status → **EXPORTED**.

3. **Bank processes file** — ops submits CSV to the bank (manual / SFTP / treasury desk).  
   Not performed by PegasusX.

4. **Mark paid** — `POST /v1/supplier/payouts/batches/{batchID}/mark-paid`  
   After bank confirms settlement; batch status → **PAID**.

## Honesty API

- `GET /v1/supplier/payouts/rail` — `{ name, is_live, workflow, steps, message }`
- Live dispatch without a live rail: `{ "code": "no_live_rail", "rail": { "is_live": false, ... } }`

## Live rails (future)

A real rail implements `payout.Rail` with `IsLive() == true`, and settlement lands via  
`POST /v1/webhooks/payouts/settlement` + `PAYOUT_RAIL_WEBHOOK_SECRET`.  
Until then, **never** advertise automatic supplier payouts.
