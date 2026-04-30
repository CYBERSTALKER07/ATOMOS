# Batch 02B - Retailer Multi-Surface Alternative Embodiments

## Embodiment Matrix

| Feature Family | Base Embodiment | Alternative A | Alternative B | Alternative C | Data Integrity Guard |
|---|---|---|---|---|---|
| Catalog Discovery | Category + search blended catalog with supplier drill-down | Search-first global SKU index | Supplier-first storefront mode | Forecast-prioritized feed | Product responses remain additive and versioned |
| Checkout and Settlement | Cart recap plus gateway or cash finalization | Single-click reorder checkout for known baskets | Deferred settlement with signed delivery token | Corporate credit line checkout with post-settlement ledger adjustment | Idempotency key + request hash per checkout mutation |
| Active Delivery Tracking | Dedicated active-deliveries surface with detail/QR handoff | Inline tracking card in orders list | Map-only tracking with simplified details | Event-only textual tracking for low bandwidth | Tracking state sourced from same order event lineage |
| AI Procurement Assist | Suggested line items accepted into cart | Full AI-drafted purchase order with review mode | Threshold-triggered auto draft (human approval required) | Supplier-cooperative forecast negotiation loop | AI outputs are advisory until explicit user confirmation |
| Profile and Governance | Profile hub with settings and auto-order controls | Split governance screens for policy and account | Supplier-level policy templates | Region-specific compliance profile variants | Policy writes require scoped auth and version checks |

## Continuity Notes

- Android, iOS, and desktop retain parity in contract shape while allowing platform-native interaction models.
- Alternatives may change interaction density, but event ordering and reconciliation safety remain invariant.
- Manual retailer decisions remain authoritative over AI recommendations in every embodiment.
