# ADR-001: Pay-at-delivery only

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** PegasusX logistics OS — retailers pay when the driver arrives, not at checkout.

## Decision

All order payment collection happens at delivery (`ARRIVED` → `AWAITING_PAYMENT` / cash collection). Pre-pay card flows are removed from PegasusX; legacy initiate endpoints return HTTP 410.

Supported settlement paths at delivery:

- Cash collection by driver (`CollectCash`)
- Card at door via payment session created on arrival
- Credit delivery (`DELIVERED_ON_CREDIT`) where retailer credit policy allows

## Consequences

- Checkout creates orders without charging cards.
- Webhook handlers reconcile gateway events against delivery-time sessions only.
- Client apps must not show pre-pay UI; warehouse/dispatch are unaffected by payment timing.
- AI preorder confirmation does not create payment sessions.

## References

- `order/service.go` — `MarkArrived`, `CollectCash`, payment-required emits
- `payment/` — session lifecycle
- `proximity/geofence.go` — 500m delivery geofence gates arrival
