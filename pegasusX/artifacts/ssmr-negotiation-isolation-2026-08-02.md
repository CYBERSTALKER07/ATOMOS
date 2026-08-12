# SSMR negotiation isolation check — 2026-08-02


> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`../docs/DOCS_SOURCE_OF_TRUTH.md`](../docs/DOCS_SOURCE_OF_TRUTH.md) · [`../docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md) · [`../context/current_status.md`](../context/current_status.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.

**Target:** `https://api-ssmr.pegasusx.app`  
**Image:** `backend-go:ssmr-ao-place-a1fafaa0-sup`  
**Command:** `go run ./cmd/ssmr-smokecheck negotiation-isolation`  
**Result:** **PASS**

## Intent

Prove quantity negotiations are **product-disabled** and do **not** break shop-closed or claims surfaces.

| Path | Moment | Status |
|------|--------|--------|
| Quantity negotiations | During delivery, qty propose/resolve | **OFF** (410 / empty) |
| Shop-closed | Retailer unavailable at stop | Live |
| Claims | Post-delivery OS&D | Live |

## Markers

```
PX_E2E_HEALTH_OK
PX_E2E_NEGOTIATION_GATE_COMPILE_OFF
PX_E2E_NEGOTIATE_PROPOSE_410_OK
PX_E2E_NEGOTIATE_RESOLVE_410_OK
PX_E2E_NEGOTIATE_PENDING_EMPTY_OK
PX_E2E_NEGOTIATION_DISABLED_OK
PX_E2E_SHOP_CLOSED_SURFACE_OK
PX_E2E_CLAIMS_SURFACE_OK status=200
PX_E2E_NEGOTIATION_ISOLATION_OK
```

## Probes

| Endpoint | Auth | Expected | Observed |
|----------|------|----------|----------|
| `POST /v1/delivery/negotiate` | Driver JWT | **410** `feature_disabled` | 410 |
| `POST /v1/supplier/negotiate/resolve` | Admin JWT | **410** | 410 |
| `GET /v1/supplier/negotiations/pending` | Admin JWT | **200** empty `data` | 200 empty |
| `GET /v1/supplier/shop-closed/active` | Admin JWT | **200** | 200 |
| `GET /v1/retailer/claims?limit=5` | Retailer JWT | **200** (not 410) | 200 |

## Notes

- Live image already enforces the disabled gate (no 2xx negotiate path).
- Local source gate: `quantityNegotiationDisabled = true`.
- Re-run anytime:

```bash
export PUBLIC_BASE_URL=https://api-ssmr.pegasusx.app
export JWT_SECRET="$(kubectl -n pegasusx-ssmr get secret backend-go-secrets -o jsonpath='{.data.jwt-secret}' | base64 -d)"
cd apps/backend-go && go run ./cmd/ssmr-smokecheck negotiation-isolation
```
