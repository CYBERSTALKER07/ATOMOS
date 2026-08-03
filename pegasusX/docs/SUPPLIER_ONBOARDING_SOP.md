# Supplier Onboarding SOP (B02)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose
Provide a repeatable support procedure for supplier onboarding in pegasusX.

## Scope
- Portal registration: `/auth/register` and API `POST /api/auth/supplier/register`
- Billing setup: `/setup/billing` and API `POST /api/supplier/billing/setup`
- Profile and topology readiness checks: `GET /api/supplier/profile`, `GET /api/supplier/topology`

## Ownership
- First response: supplier support
- Escalation: backend platform team for auth, persistence, idempotency, or proxy transport faults

## Preconditions
1. Supplier is using the supplier portal app.
2. Backend is reachable through the portal proxy route `/api/*`.
3. Supplier can provide contact phone, legal name, and timestamp of failure.

## Happy-path checklist
1. Confirm supplier starts at `/auth/register` and completes all 4 steps.
2. Registration request returns `201` with payload fields:
   - `supplier_id`
   - `is_configured` (expected `false` before billing)
   - `next_step` (expected `/setup/billing`)
3. Confirm browser has `supplier_jwt` cookie after registration.
4. Confirm supplier is redirected to `/setup/billing`.
5. Billing request returns `200` with payload fields:
   - `supplier_id`
   - `is_configured` (expected `true`)
   - `selected_gateways`
6. Confirm middleware no longer redirects to `/setup/billing` after billing succeeds.
7. Validate profile by calling `GET /api/supplier/profile`.
8. Validate topology by calling `GET /api/supplier/topology`.

## Failure handling matrix

| Symptom | Expected error/body | Meaning | Support action |
|---|---|---|---|
| Register fails with `400` | `{"error":"invalid_json"}` or `read_body:*` | Malformed request payload | Re-run from step with corrected input shape |
| Register or billing fails with `409` | `{"error":"idempotency_key_payload_mismatch"}` | Same idempotency key used with different payload | Retry with a new idempotency key and stable payload |
| Login fails with `401` | `{"error":"invalid_credentials"}` | Wrong phone/password or missing credential row | Verify phone format and password reset path |
| Register or billing fails with `422` | `{"error":"..."}` | Business validation failed | Resolve field-level validation message and retry |
| Any proxied call fails with `502` | `{"error":"upstream_unreachable"}` | Portal cannot reach backend target | Verify backend base URL env and backend health |
| Topology update fails with `400` | `warehouses_required`, `name_required`, `lat_out_of_range`, `lng_out_of_range` | Topology payload invalid | Correct the offending index/field and retry |

## Escalate immediately when
1. Reproducible `500` responses from supplier handlers.
2. `supplier_jwt` cookie is not set after successful register/login/billing response.
3. `GET /api/supplier/profile` or `GET /api/supplier/topology` repeatedly fails after successful write calls.
4. Outage pattern includes widespread `upstream_unreachable` from portal proxy.

## Evidence to attach in escalation ticket
1. Supplier phone and supplier_id (if available).
2. Request endpoint and method.
3. HTTP status and response body.
4. Approximate UTC timestamp.
5. Browser/network trace snippet showing request and response headers.
