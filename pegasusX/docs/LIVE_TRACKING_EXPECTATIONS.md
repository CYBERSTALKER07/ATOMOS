# Live tracking expectations

Contract for what operators, retailers, and supplier finance teams should expect to see during an active delivery. Pairs with `DRIVER_SUPPORT_PLAYBOOK.md` (driver-side triage) and `DELIVERY_ESCALATION_POLICY.md` (escalation policy).

## Visibility contract

| Surface | Expected refresh cadence | Frame shape |
| --- | --- | --- |
| Operator fleet map | Near-real-time (single-digit seconds under healthy load) | Driver identity, vehicle identity, current manifest, last stop, next stop, last update timestamp |
| Retailer tracking screen | Near-real-time during active delivery; eventual consistency for pre and post states | Order status, ETA hint, driver-arrival flag, payment-required hint when applicable |
| Supplier finance dashboard | Eventually consistent within reconciliation window | Order final amount, payout split snapshot, settlement state, fee amount |

## Required states

Every live tracking surface must explicitly handle:

1. Loading: data is being fetched for the first time.
2. Empty: there are no active deliveries to display.
3. Offline or reconnecting: the WebSocket has dropped and is being re-established.
4. Stale data: the connection is open but no frames have arrived within the cadence threshold.
5. Permission-restricted: the viewer does not have authority to see the live frame.

Silent failures, such as a frozen map without an offline indicator, are not acceptable.

## Frame integrity

- Every telemetry frame must carry a trace ID that stitches it back to its originating driver event.
- Operator map hover or focus state should expose at minimum: driver identity, truck identity, route identity, assigned order count, current or next stop, and last update timestamp.
- Clicking a route, marker, driver, or truck should open the corresponding detail surface when the product already has or clearly needs one.

## Deviation surfacing

- If the planned route and the actual execution diverge, the deviation should be visible to operators rather than hidden.
- Operators should be able to tell whether the active sequencing is the optimized default or a driver-selected override.

## Out of band events

- Force-complete, force-reassign, and manual override actions must appear in the live tracking audit trail with the actor identity.
- These actions never replace the underlying telemetry; they only annotate it.

## Cross-references

- `DRIVER_SUPPORT_PLAYBOOK.md` for driver-side triage.
- `DELIVERY_ESCALATION_POLICY.md` for when to escalate.
- `REASSIGNMENT_SUPPORT_PLAYBOOK.md` for reassignment context that affects live tracking.
