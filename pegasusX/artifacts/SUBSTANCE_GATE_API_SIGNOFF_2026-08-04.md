# Substance Gate — API sign-off (2026-08-04)

```text
Environment: SSMR
API build / image: asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-substance-gate-a66868b8-084112
E2E log: artifacts/ssmr-e2e-substance-gate-2026-08-04.log
Claims probe: artifacts/ssmr-claims-substance-gate-2026-08-04.log
Marker gate: PASS (ssmr-ecosystem-marker-gate-ok)
Date: 2026-08-04  Operator: agent (backend-first plan)

Role          API   WEB   AND   IOS
Retailer      PASS  DEFERRED  DEFERRED  DEFERRED
Supplier      PASS  DEFERRED  DEFERRED  DEFERRED
Warehouse     PASS  DEFERRED  DEFERRED  DEFERRED
Factory       PASS  DEFERRED  DEFERRED  DEFERRED
Driver        PASS  N/A   DEFERRED  DEFERRED
Payload       PASS  N/A   DEFERRED  DEFERRED

Theatre exceptions still open:
- Touchless confidence / seasonality / billing meter product depth (meter_worker schema fixed; still thin)
- SHOP_CLOSED inbox soft-fail (notification type not always visible in capped inbox)
- Supplier/WH claim-window portal UX (API Done; portal settings open)

Blockers (ops):
- GP: PX_E2E_PAYMENT_CARD_SUCCESS_SKIPPED / GLOBAL_PAY_WEBHOOK_SKIPPED (merchant password / card SUCCESS path)
- Firebase SMS / SHA: device OTP not proven this pass (driver Firebase OTP not re-run as UI)
- TokenCreator: GCS claim media OK this pass (PX_E2E_CLAIM_MEDIA_GCS_OK)

Deferred (SKIPPED ≠ Done):
- NEGOTIATION, SOLIQ, MULTI_ORG picker/switch, OFFLINE_COUNT, ASSIST_SLA (smokecheck env / flag-gated)
- SUPPLIER_IMPORT_ASYNC (cloud worker cannot read local upload root)
- CREDIT_POLICY_SKIPPED
```

## Preflight (Phase 1)

| Check | Result |
|-------|--------|
| `/healthz` | OK |
| Spanner Claims + Supplier/WarehouseReturnPolicies + ClaimWindow* | present |
| ConfigMap `REQUIRE_INFRA_ADAPTERS=true`, `FISCAL_PROVIDER=PEGASUS`, `CLAIM_WINDOW_HOURS=48` | OK |
| Worker replicas | 1 |
| Cloud smoke | `PX11_CLOUD_SMOKE_OK` |

## Fixes landed this pass (for green API)

1. Smokecheck: unique idempotency keys (import/colocate/preorder/fleet/…) — fixed Redis cross-run replay Theatre  
2. Supplier + warehouse idempotency namespaced by supplier_id  
3. Claims e2e wired into full `e2e` check; WH reverse list uses PAYLOAD/WAREHOUSE_ADMIN scope  
4. Deployed image with G3 claim-eligibility routes  
5. Spanner: `ReservedMinor` on `RetailerCreditProfiles`; `ConfirmedAt`/`CommittedAt` commit-timestamp options  
6. Billing `MeterWorker` aligned to live `BillingSupplierMeters` / `BillingMeterEvents` schema  
7. Return-gate: fresh fleet mint + fiscal force-unstick parity  

## Client smoke (Phase 2 light)

`GET /v1/platform/client-policy` for RETAILER / SUPPLIER / WAREHOUSE / FACTORY / DRIVER / PAYLOAD × web/android/ios → all HTTP 200.  
Interactive UI walks (desktop/Android/iOS) remain **DEFERRED** — API column is authoritative for this sign-off.
