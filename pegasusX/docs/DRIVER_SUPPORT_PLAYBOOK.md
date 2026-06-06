# Driver support playbook

Operational playbook for triaging driver execution issues across Android and iOS. Pairs with `LIVE_TRACKING_EXPECTATIONS.md` (visibility contract) and `DELIVERY_ESCALATION_POLICY.md` (when to escalate beyond first-line support).

## Scope

- Driver login and home-node binding issues.
- Manifest receive, sealed-load, depart, in-transit, arrival, and complete transitions.
- Geofence-gated `complete` failures and driver-side override permissions.
- Telemetry stalls, reconnect loops, and "driver pin frozen on map" reports.
- Cash collection, payment-cleared handshake, and offload confirmation.

Out of scope: factory or warehouse exception triage (see `WAREHOUSE_EXCEPTION_SOP.md` and `REASSIGNMENT_SUPPORT_PLAYBOOK.md`).

## Triage entry points

| Symptom | First-line action | Owning surface |
| --- | --- | --- |
| Driver cannot log in | Verify supplier and home node binding; re-issue credentials if onboarding incomplete | `apps/backend-go/auth`, driver apps |
| Manifest not visible after sealed-load | Confirm manifest dispatch event was emitted; check ws driver hub subscription | `apps/backend-go/{supplier,driver}` |
| `complete` rejected at retailer location | Confirm geofence policy and retailer coordinates; instruct driver to re-arrive | `apps/backend-go/order` |
| Driver pin frozen on map | Ask driver to check connectivity; verify telemetry hub frame cadence | telemetry pipeline |
| Cash collected but order not cleared | Confirm `collect-cash` succeeded and settlement event fanned out | `apps/backend-go/{order,payment}` |
| Reassignment confusion mid-route | Defer to `REASSIGNMENT_SUPPORT_PLAYBOOK.md` and freeze-lock policy | payload/factory services |

## Resolution paths

1. Login or home-node binding
   1. Confirm the supplier still has the driver active and home-node assignment is current.
   2. If JWT-side binding drifted, the driver should sign out and back in to refresh claims.
   3. If the issue persists, route to engineering with the driver ID and supplier ID.

2. Manifest visibility
   1. Confirm the supplier has dispatched the manifest and it is in a `SEALED` or `DISPATCHED` state.
   2. Ask the driver to refresh the app once to force a clean WebSocket reconnect.
   3. If the manifest is still missing, check the driver hub fanout journal for the corresponding dispatch event.

3. Geofence-gated complete
   1. Confirm retailer location accuracy with the retailer support team.
   2. If location drift is the cause, drivers may use the supplier-approved manual arrival flow; do not bypass the backend geofence check at the client layer.
   3. Reattempt `complete` once the driver is inside the geofence radius.

4. Telemetry stalls
   1. Check the device's network connectivity and background app permissions.
   2. If the device is healthy, ask the driver to manually trigger a heartbeat by re-opening the active manifest screen.
   3. Confirm the telemetry hub frame is reaching the operator surface; if not, route to engineering with the driver ID and last seen timestamp.

5. Cash and payment handshake
   1. Verify the `collect-cash` mutation completed and the order is in `AWAITING_PAYMENT_SETTLEMENT` or `COMPLETED`.
   2. If the order is stuck before settlement, defer to `PAYMENT_EXCEPTION_SOP.md`.

## Communication tone

- Acknowledge the driver is on the road; keep instructions short and actionable.
- Never ask the driver to operate an admin surface mid-route.
- For irreversible actions (force-complete, manifest reassignment), escalate before suggesting.

## Logging

Every driver support touch should be logged with: driver ID, supplier ID, manifest ID (if applicable), order ID (if applicable), reported symptom, action taken, resolution.

## Cross-references

- `LIVE_TRACKING_EXPECTATIONS.md` for what the operator and retailer should expect to see during a live delivery.
- `DELIVERY_ESCALATION_POLICY.md` for when to involve engineering or treasury.
- `PAYMENT_EXCEPTION_SOP.md` for payment-affected delivery issues.
- `REASSIGNMENT_SUPPORT_PLAYBOOK.md` for payload or factory reassignment context.
