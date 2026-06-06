# Retailer Onboarding Support Flows (B03)

## Purpose
Define repeatable support handling for retailer onboarding and early commerce flows in pegasusX.

## Scope
- Retailer registration: `POST /v1/auth/retailer/register`
- Retailer profile: `GET|PUT /v1/retailer/profile`
- Supplier relationship: `GET /v1/retailer/suppliers`, `POST /v1/retailer/suppliers/{supplierID}/{action}`
- Cart sync: `GET|POST /v1/retailer/cart/sync`
- Order capture entrypoint: `POST /v1/order/create`

## Ownership
1. First response: retailer support operations.
2. Escalation: backend platform team when auth, persistence, idempotency, or routing authority appears inconsistent.

## Onboarding flow checklist
1. Submit registration with phone + coordinates + H3 cell to `POST /v1/auth/retailer/register`.
2. Expect `201` with `retailer_id`, `phone`, `h3_cell`, `created_at`.
3. Verify profile retrieval through `GET /v1/retailer/profile`.
4. Verify supplier association using `GET /v1/retailer/suppliers`.
5. Verify cart round-trip using `POST /v1/retailer/cart/sync` followed by `GET /v1/retailer/cart/sync`.
6. Verify first order path with `POST /v1/order/create` when line items and location fields are present.

## Failure matrix

| Endpoint | Typical status | Error body | Support action |
|---|---|---|---|
| `POST /v1/auth/retailer/register` | 400 | `invalid_json` or `read_body:*` | Correct payload format and retry |
| `POST /v1/auth/retailer/register` | 409 | `idempotency_key_payload_mismatch` | Retry with a new idempotency key |
| `POST /v1/auth/retailer/register` | 422 | validation message | Correct field-level issue (`phone`, `lat/lng`, `h3_cell`) |
| `GET|PUT /v1/retailer/profile` | 401 | `unauthorized` | Re-authenticate and retry with valid retailer identity context |
| `POST /v1/retailer/suppliers/{supplierID}/{action}` | 400 | `supplierID and action(add|remove) required` | Correct path params and retry |
| `GET|POST /v1/retailer/cart/sync` | 401 or 400 | `unauthorized` or `invalid_json` | Restore auth context or fix cart payload |
| `POST /v1/order/create` | 422 | validation message | Fix line items/H3/location values and retry |

## Escalation triggers
1. Registration response is successful but `GET /v1/retailer/profile` consistently returns not found.
2. Duplicate registration calls with same payload produce divergent `retailer_id` values.
3. Supplier list endpoint returns empty or invalid seeded supplier association for newly created retailers.
4. Cart data cannot be read immediately after successful cart sync write.

## Evidence package for escalations
1. Retailer phone and `retailer_id`.
2. Endpoint + method.
3. Request timestamp (UTC).
4. HTTP status and response body.
5. Correlated client/network trace where available.
