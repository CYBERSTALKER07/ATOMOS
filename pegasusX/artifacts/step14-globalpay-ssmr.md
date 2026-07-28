# Step 14 — Global Pay staging webhooks (SSMR)

**Date:** 2026-07-27  
**Project:** `pegasus-503013`  
**Namespace:** `pegasusx-ssmr`  
**Ingress IP:** `136.69.43.141`  
**Host:** `api-ssmr.pegasusx.app`

## Verdict

| Item | Status |
|------|--------|
| Endpoint live | **PASS** — `POST /v1/webhooks/global-pay` |
| `GLOBAL_PAY_ENV=staging` | **PASS** |
| `PUBLIC_BASE_URL` | **PASS** — `https://api-ssmr.pegasusx.app` |
| Webhook secret wired | **PASS** — GSM `pegasusx-ssmr-global-pay-webhook-secret` |
| Reject no/bad auth | **PASS** — HTTP **401** `invalid_signature` |
| Accept signed CAPTURED | **PASS** — HTTP **200** `status=accepted` |
| Idempotent replay | **PASS** — second POST still **200** |
| SUCCESS + status verify | **Blocked on real merchant creds** — hits `checkout-api-staging.globalpay.uz` and fails auth (expected) |
| Outbound merchant login | **Needs Global Pay portal** — password still placeholder |

## Register this URL in Global Pay staging backoffice

```
https://api-ssmr.pegasusx.app/v1/webhooks/global-pay
```

Until ManagedCertificate + DNS are Active (Step 11 gap), external HTTPS may fail validation for Global Pay’s servers. Interim:

```
http://api-ssmr.pegasusx.app/v1/webhooks/global-pay
```

with DNS A → `136.69.43.141` (or temporary Host-header testing).

**Auth expected by backend**

```
Authorization: Basic base64("Paycom:" + GLOBAL_PAY_WEBHOOK_SECRET)
```

Secret value is in GSM `pegasusx-ssmr-global-pay-webhook-secret` (do not commit).

## Smoke results (via Ingress)

```
no auth          → 401 invalid_signature
bad secret       → 401 invalid_signature
CAPTURED + auth  → 200 {"gateway":"global-pay","status":"accepted","transaction_id":"..."}
replay CAPTURED  → 200 accepted (idempotent)
SUCCESS + auth   → 502 verification_failed
                 (staging API: user not authorized — dummy/placeholder merchant password)
```

### Smoke script

```bash
HOST=api-ssmr.pegasusx.app
IP=136.69.43.141
SECRET="$(gcloud secrets versions access latest \
  --secret=pegasusx-ssmr-global-pay-webhook-secret --project=pegasus-503013)"
AUTH="Basic $(printf 'Paycom:%s' "$SECRET" | base64)"

curl -fsS --resolve "${HOST}:80:${IP}" \
  -H "Authorization: $AUTH" -H 'Content-Type: application/json' \
  -d '{
    "session_id":"sess-smoke",
    "transaction_id":"tx-smoke-1",
    "status":"CAPTURED",
    "order_id":"ord-smoke",
    "amount_minor":100000,
    "currency":"UZS"
  }' \
  "http://${HOST}/v1/webhooks/global-pay"
```

## Config / secrets

| Env / secret | Value / note |
|--------------|--------------|
| `GLOBAL_PAY_ENV` | `staging` → checkout/backoffice `*.globalpay.uz` staging hosts |
| `GLOBAL_PAY_WEBHOOK_SECRET` | wired (GSM) |
| `GLOBAL_PAY_SERVICE_ID` | `ssmr-staging-service` (placeholder until portal) |
| `GLOBAL_PAY_USERNAME` | `ssmr-staging-merchant` (placeholder) |
| `GLOBAL_PAY_PASSWORD` | `REPLACE_WITH_GP_STAGING_PASSWORD` — **must be replaced** |
| Checkout base (staging) | `https://checkout-api-staging.globalpay.uz/checkout` |

## Behaviour notes (product)

1. **Auth:** Basic `Paycom:<webhook_secret>` only (not `X-GlobalPay-Signature` on this path).
2. **Required body fields:** `session_id`, `transaction_id` (or payment_id), `status`.
3. **SUCCESS / SETTLED:** backend calls Global Pay merchant auth + status API before accept. Real staging username/password required.
4. **CAPTURED / other:** accepted after signature check without remote status verify (used by SSMR payment smoke).

Full order→checkout→webhook e2e:  
`PUBLIC_BASE_URL=… JWT_SECRET=… GLOBAL_PAY_WEBHOOK_SECRET=… go run ./cmd/ssmr-smokecheck payment`

## Your action for full staging SUCCESS webhooks

1. Obtain from Global Pay merchant staging portal:
   - service_id, username, password  
   - confirm webhook secret (or set ours and share with them)
2. Update GSM:
   ```bash
   printf '%s' "$REAL_SERVICE_ID" | gcloud secrets versions add pegasusx-ssmr-global-pay-service-id --project=pegasus-503013 --data-file=-
   printf '%s' "$REAL_USER" | gcloud secrets versions add pegasusx-ssmr-global-pay-username --project=pegasus-503013 --data-file=-
   printf '%s' "$REAL_PASS" | gcloud secrets versions add pegasusx-ssmr-global-pay-password --project=pegasus-503013 --data-file=-
   # optional: webhook secret if they dictate
   ```
3. ESO refresh or patch `backend-go-secrets` + restart API.
4. Finish Step 11 DNS + ManagedCertificate so HTTPS webhook URL is trusted.
5. Register callback URL in Global Pay staging dashboard.

## Next

**Step 15** — OFD sandbox (after 10+14).  
Or inject real Global Pay staging merchant credentials and re-test SUCCESS path.
