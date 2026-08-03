# Billing Recovery Script (B02)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Use this script when a supplier cannot complete `/setup/billing`.

## Operator script

### Step 1: Collect incident facts
1. Ask for supplier phone number.
2. Ask for exact time (UTC or local with timezone).
3. Ask which action failed: save billing, login, or redirect loop.
4. Ask for visible error text from UI.

### Step 2: Confirm session state
1. Verify supplier can reach `/setup/billing`.
2. Confirm `supplier_jwt` cookie exists.
3. If no cookie, ask supplier to re-run login at `/auth/register` flow entry or login surface.

### Step 3: Diagnose by status code
1. `400 invalid_json`:
   - Action: verify request body shape and retry once.
2. `409 idempotency_key_payload_mismatch`:
   - Action: retry with a new `Idempotency-Key` header.
   - Rule: never reuse an old key with changed payload.
3. `422` with validation message:
   - Action: fix the named field and retry.
   - Common field failures: `bankName`, `accountHolder`, `accountNumber`, `swiftBic`, `selectedGateways`.
4. `502 upstream_unreachable`:
   - Action: check supplier portal API proxy target and backend health.
   - Escalate to platform if repeated.
5. `500`:
   - Action: immediate engineering escalation with evidence package.

### Step 4: Validate recovery
1. Re-submit billing setup.
2. Confirm response contains:
   - `supplier_id`
   - `is_configured: true`
   - `selected_gateways`
3. Confirm redirect to `/` (or dashboard entry path).
4. Confirm middleware no longer forces `/setup/billing` redirect.

## Fast command checklist (support + engineering)
1. Validate profile:
   - `GET /api/supplier/profile`
2. Validate billing-configured state:
   - check `is_configured` in profile response
3. Validate topology still readable post-billing:
   - `GET /api/supplier/topology`

## Escalation template
1. Incident type: Billing setup failure
2. Supplier identifier: `<phone or supplier_id>`
3. Endpoint: `POST /api/supplier/billing/setup`
4. Status/body: `<status + response>`
5. Timestamp: `<utc>`
6. Repeatability: `<always/intermittent>`
7. Environment notes: include proxy target if known (`SUPPLIER_BACKEND_BASE_URL` fallback chain)
