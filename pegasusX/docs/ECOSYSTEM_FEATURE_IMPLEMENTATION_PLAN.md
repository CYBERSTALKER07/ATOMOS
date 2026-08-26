# V.O.I.D. Ecosystem Feature Implementation Plan

**Date:** 2026-08-19  
**Scope:** `pegasusX/` only  
**Purpose:** End-to-end implementation blueprint for the next enterprise logistics capabilities.  
**Status rule:** This document is a plan, not evidence that a feature is implemented. Live code remains the source of truth.

## 1. Product Constraints

Every slice in this plan must preserve these existing laws:

- `SupplierId` is the only tenant key. `market_code`, `home_cell`, country, city, and region are attributes.
- MarketPack owns currency, decimal places, PSP catalog, fiscal adapter, payout rail, timezone, and operational limits.
- Orders, credit, fiscalization, and settlement remain same-market only.
- Default fulfillment is closest eligible covering warehouse, then closest eligible factory.
- Supplier pins override proximity; missing geography fails closed.
- Money is integer minor units paired with an explicit currency.
- Durable state changes use Spanner `ReadWriteTransaction` plus transactional outbox.
- Mutations invalidate caches after commit; user-visible events use the existing scoped WebSocket and push paths.
- Factory transfer manifests and supplier last-mile manifests remain separate planes.
- Factory planning place and retailer auto-order place remain disabled by default.
- Planned or unkeyed adapters return honest `404`, `409`, `422`, `501`, or `410` responses. No fake success payloads.

## 2. Current Reuse Map

These are existing foundations to extend, not replace:

| Concern | Existing owner | Reuse rule |
|---|---|---|
| Order lifecycle | `apps/backend-go/order`, `orderroutes` | Extend the state machine additively; do not create a second order status model. |
| Warehouse resolution | `order/warehouse_resolver_spanner.go`, `proximity` | Make catalog, quote, preview, and create use the same resolver and coverage engine. |
| Inventory | `SupplierInventoryV2`, `inventory`, `stocklots`, retailer stock | Preserve reservations, version checks, lot traceability, and audit movements. |
| Dispatch | `dispatch`, supplier/warehouse dispatch services, optimizer sidecar | Improve scoring and inputs; keep deterministic fallback and freeze locks. |
| Factory planning | `factory`, `SupplyLanes`, `FactoryInternalTransfers` | Keep flags off until soak evidence; write factory-plane manifests only. |
| Payments and ledger | `payment`, `PaymentSessions`, `PaymentLedgerEntries` | Add capabilities around existing attempts, webhooks, reconciliation, and immutable entries. |
| Fiscal | `order` fiscal handlers/workers | Fiscal failure remains a blocking state, never a warning. |
| Claims and returns | `claims`, `returns`, `creditnote` | Use one claim/liability spine; do not add a parallel chargeback product. |
| Exceptions | `controltower`, scored exceptions, playbooks | Convert passive exceptions into owned, auditable actions. |
| Freshness | cache ETags, Redis invalidation, scoped WS events | Use event-driven refresh with a bounded polling safety net. |
| Shared contracts | `packages/types`, `packages/api-client`, event schema | Add fields additively and update all role-row clients together. |

## 3. Delivery Order

| Slice | Capability | Primary roles | Why this order |
|---|---|---|---|
| E0 | Contract and data-quality hardening | All | Prevents new planning logic from consuming conflicting truth. |
| E1 | Promise and availability engine | Supplier, warehouse, retailer | Makes the commercial promise match actual stock and capacity. |
| E2 | Warehouse execution intelligence | Warehouse, supplier, payload | Prevents short picks, expiry loss, and late dock discovery. |
| E3 | Exception command center and playbooks | Supplier, warehouse, factory, retailer | Converts visibility into controlled action and recovery. |
| E4 | Supplier S&OP and factory scenarios | Supplier, factory, warehouse | Connects forecast, capacity, inventory, and financial trade-offs. |
| E5 | Returns, claims, and liability closure | Retailer, warehouse, supplier | Links physical stock, evidence, reverse logistics, and money. |
| E6 | Supplier and carrier collaboration | Supplier, warehouse, factory, external partners | Adds enterprise network capability without changing last-mile authority. |
| E7 | Sustainability and cost-to-serve | Supplier, warehouse, platform | Makes optimization measurable by cost, service, and emissions. |
| E8 | Gated commercial adapters | Platform, supplier, retailer | Only after code paths are real: live PSP, fiscal, EDI, SAML/SCIM. |

Each slice is a separate implementation batch. Do not implement E1-E8 as one mega-change.

## 4. E0 - Contract and Data-Quality Hardening

### Business outcome

All downstream decisions consume one authoritative representation of geography, SKU identity, inventory, currency, order commitment, and execution state.

### Existing code to adapt

- `proximity` coverage and H3 resolution helpers.
- `catalog/stock.go`, checkout quote, unified checkout, and warehouse resolver.
- `SupplierInventoryV2`, `stocklots`, retailer stock movements, and inventory audit foundations.
- `packages/types`, `packages/api-client`, `contracts/events.schema.json`.

### Implementation logic

1. Define canonical read models for `AvailableToPromise`, `SKUIdentity`, `InventoryPosition`, and `ExecutionCommitment` in the owning domain packages.
2. Make every stock and serviceability read use supplier scope, active retailer location, same-market checks, coverage pins, on-shift status, and H3 res 7.
3. Add additive source metadata: `source`, `authority`, `as_of`, `stale`, and `available`.
4. Reconcile duplicate SKU aliases through the existing entity-resolution and catalog identity paths.
5. Reject missing country, invalid H3, unknown currency, missing UOM, and supplier-scope mismatch before persistence.
6. Add indexes before new list or aggregate queries; use stale reads for dashboards and strong reads for mutation preconditions.

### Edge cases

- Retailer switches active store while a cart is open.
- A warehouse is geographically closest but off shift, closed, frozen, or below dispatch capacity.
- Catalog and checkout disagree on a retailer coordinate or SKU alias.
- A supplier has two products with the same barcode but different UOMs.
- An old client omits additive fields; the server keeps compatibility without inventing values.

### Acceptance

- Catalog, quote, preview, and create choose the same warehouse for the same input.
- Cross-market and incomplete-geography tests fail closed.
- No new query performs an unindexed supplier-wide scan.
- Contract tests cover web, Android, and iOS models for every affected role.

## 5. E1 - Promise and Availability Engine

### Business outcome

Retailers see a truthful promise before ordering: what can be supplied, from where, when, and with what confidence.

### Contract

`POST /v1/retailer/availability/quote` and the existing checkout quote/preview should return additive fields:

```text
promise_id, supplier_id, warehouse_id, factory_id,
requested_qty, allocatable_qty, backordered_qty,
promised_at, cutoff_at, service_level, confidence,
reason_codes[], currency, source, stale
```

The client must not choose `warehouse_id`; the server resolves it.

### Backend logic

1. Resolve the active retailer location from claims and `active_location_id`.
2. Group requested lines by supplier and enforce same-market rules.
3. Resolve eligible warehouse per supplier using coverage, pins, country, on-shift state, inventory, and dispatch capacity.
4. Read on-hand, reserved, inbound, safety stock, open commitments, preorder units, and approved substitutions.
5. Allocate in priority order: committed orders, service-level commitments, scheduled orders, then new demand.
6. If stock is insufficient, evaluate same-country factory replenishment and return a dated promise rather than pretending stock exists.
7. Persist the accepted promise snapshot with the order. Never recalculate historical promises from current inventory.
8. Emit `ORDER_PROMISE_CREATED` through outbox only when a durable promise is created; use a separate notification event for UI refresh.
9. Invalidate supplier, warehouse, retailer cart, and promise caches after commit.

### Client wiring

- Retailer desktop, Android, and iOS: show promise per supplier child order, partial availability, cutoff, and reason.
- Supplier portal, Android, and iOS: show promise failures, allocation pressure, and affected retailers.
- Warehouse portal, Android, and iOS: show committed demand, promised outbound, and allocation holds.
- Factory surfaces: show only factory-plane replenishment implications, never last-mile manifests.

### Edge cases

- Partial allocation across suppliers in one parent cart.
- Promise expires while payment is pending.
- Inventory becomes unavailable between quote and create.
- Factory transfer is late after a promise was issued.
- A retailer requests a substitute with a different price, UOM, or fiscal classification.

### Rollout gates

Start read-only beside current quote. Compare decisions for 30 days. Enable order enforcement per supplier only after mismatch, stockout, and promise-breach metrics meet thresholds. Auto-place remains off.

## 6. E2 - Warehouse Execution Intelligence

### Business outcome

Warehouse staff discover shortages, expiry, temperature violations, and labor bottlenecks before sealing a truck.

### Backend logic

1. Extend existing WMS pick waves with task priority, route cutoff, temperature zone, lot/expiry, and worker capability requirements.
2. Add FEFO allocation for lot-controlled products; FIFO remains an explicit product policy, never an implicit fallback.
3. Add dynamic slotting recommendations using pick frequency, cube, weight, expiry risk, replenishment cost, and compatible storage conditions.
4. Add task interleaving between pick, putaway, replenishment, cycle count, and exception review.
5. Add pick short workflow: worker reports shortage, supervisor accepts or rejects waiver, allocation engine re-plans, and retailer promise is updated.
6. Add receiving appointment and dock-capacity constraints before factory or supplier dispatch is confirmed.
7. Connect cold-chain readings to lots and tasks. A breach opens a quarantine/exception action, not a dashboard-only alert.
8. Keep inventory mutations in transactions with audit movements, outbox events, cache invalidation, and version checks.

### Client wiring

- Warehouse portal: wave planner, slotting review, labor board, cold-chain exceptions, and supervisor approvals.
- Warehouse Android/iOS: scanner-first pick, putaway, count, quarantine, and offline reconciliation flows.
- Supplier surfaces: committed quantity, short-pick reason, and revised promise.
- Payload surfaces: sealed quantity must equal accepted pick quantity; no silent over-injection.

### Edge cases

- Lot expires during a long pick wave.
- Cold-chain sensor is offline or reports an impossible reading.
- Worker loses connectivity after confirming a task.
- The same SKU is split across lots, bins, and temperature zones.
- A short pick affects multiple retailer child orders with different priorities.
- A sealed manifest contains a line later found quarantined.

### Rollout gates

Phase 1 is read-only recommendations. Phase 2 gates seal on accepted pick quantities for selected warehouses. Phase 3 enables dynamic slotting and task interleaving after inventory accuracy and duplicate-task metrics are proven.

## 7. E3 - Exception Command Center and Playbooks

### Business outcome

Every disruption has an owner, deadline, evidence trail, and bounded recovery action.

### Backend logic

1. Normalize exceptions from order, inventory, route, fiscal, payment, cold-chain, labor, transfer, and partner events into the existing control-tower model.
2. Assign severity from customer impact, money exposure, food/medical risk, SLA breach, and operational age.
3. Resolve owner from supplier, warehouse, factory, driver, or platform scope; never from request body IDs.
4. Create an immutable exception timeline with deduplication key `(supplier_id, aggregate_type, aggregate_id, reason, active_window)`.
5. Attach playbooks with preconditions, allowed actions, required approvals, rollback action, and maximum retry count.
6. Human action writes the aggregate mutation and outbox event in one transaction, then emits scoped notifications.
7. AI may recommend or simulate actions, but cannot bypass freeze locks, financial controls, fiscal gates, or role permissions.
8. Expired playbooks escalate to the next role and produce an actionable SLA breach event.

### Example playbooks

- Stockout: substitute, split, replenish, or notify retailer.
- Late transfer: reroute from a same-country factory, change promise, or escalate.
- Driver delay: replan remaining stops while respecting freeze locks and partial delivery state.
- Fiscal failure: retry with bounded backoff, then route to authorized reconciliation.
- Payment mismatch: hold settlement, open reconciliation case, never mark cleared manually without audit.

### Client wiring

Supplier, warehouse, factory, and retailer role rows receive only scoped exceptions. Desktop/iPad gets command board plus inspector; phone gets one-action ritual with confirmation and recovery. WebSocket and push events must be additive and backward compatible.

### Edge cases

- Duplicate Kafka event opens no duplicate exception.
- Two operators attempt different resolutions concurrently.
- A recommended action becomes invalid before approval.
- Redis is unavailable; local delivery continues but stale state is labeled.
- The exception crosses a factory/last-mile boundary; it must not merge the two manifest planes.

## 8. E4 - Supplier S&OP and Factory Scenarios

### Business outcome

Supplier leadership can compare demand, inventory, factory capacity, transport cost, service level, and cash impact before publishing a plan.

### Backend logic

1. Keep forecast, preorder, confirmed demand, and actual demand as separate series.
2. Create immutable scenario versions containing assumptions, horizon, supplier scope, market pack, and source snapshot timestamps.
3. Simulate demand changes, factory output constraints, supply lanes, warehouse capacity, labor, and route capacity.
4. Produce recommendations and deltas only; do not mutate inventory, transfers, or manifests during simulation.
5. Require supplier approval and optional dual control to publish a scenario.
6. Publishing creates explicit transfer drafts or planning intents; factory planning flags remain required for execution.
7. Record forecast accuracy by SKU, location, horizon, and source model. Do not display a forecast line when the source is unavailable.

### Client wiring

- Supplier `/planning`: Planning and Digital Brain tabs, scenario compare, approval, and publish status.
- Factory: capacity, lane, and transfer consequences on the factory plane.
- Warehouse: projected inbound, stock risk, and labor impact.
- Retailer: only retailer-safe promise or availability changes, never internal supplier assumptions.

### Edge cases

- Scenario uses stale inventory or missing capacity.
- Factory capacity is unlimited or absent; label the constraint as unavailable.
- Two scenarios publish concurrently.
- Published plan conflicts with a freeze lock or active manifest.
- Currency or market pack changes between scenario creation and publication.

## 9. E5 - Returns, Claims, and Liability Closure

### Business outcome

Visible damage, concealed damage, claims, quarantine, reverse logistics, credit notes, refunds, and supplier liability form one traceable chain.

### Backend logic

1. Reuse `claims` as the liability source of truth; do not create a second chargeback request table.
2. At delivery, accepted quantities exclude driver-reported damage or missing units. Do not create a duplicate concealed-damage claim.
3. After receipt, a retailer claim validates organization scope, order completion, claim window snapshot, line quantities, reason, and evidence.
4. Move affected retailer stock to `QUARANTINE` with a claim-hold movement when appropriate.
5. Supplier or warehouse approval opens or confirms reverse logistics and creates paired financial entries only at the approved settlement stage.
6. Rejection restores stock according to the original movement and records the reason.
7. Refunds and chargebacks are new immutable ledger reversals, never edits to historical entries.
8. Notify retailer, supplier, and warehouse through scoped events with claim, order, line, quantity, currency, and evidence identifiers.

### Client wiring

All retailer clients: stock-first claim entry, evidence upload, status timeline, and eligibility countdown. Supplier and warehouse clients: adjudication, reverse receipt, quarantine, and settlement views. Mobile clients must queue evidence and mutation intent offline with idempotency keys.

### Edge cases

- Claim window expires while a draft is offline.
- Same line is reported by driver and later claimed by retailer.
- Partial delivery and partial claim quantities overlap.
- Currency differs from current market configuration; historical order currency wins.
- A refund succeeds at the gateway but the webhook is delayed or duplicated.

## 10. E6 - Supplier and Carrier Collaboration

### Business outcome

Suppliers can collaborate with factories, 3PLs, carriers, and trading partners without weakening internal scope or fiscal controls.

### Backend logic

1. Add carrier/partner profiles, capabilities, service areas, rates, insurance, and compliance documents under supplier scope.
2. Add tender, accept, reject, appointment, pickup, proof, invoice, and performance states.
3. Use existing partner dialect and EDI/AS2 boundaries; add ASN and receiving variance only where the adapter is real.
4. Keep external-carrier execution separate from supplier truck manifests but project both into common operational read models with a plane field.
5. Calculate freight cost in minor units and explicit currency; store accessorials and detention as separate ledger-relevant entries.
6. Add carrier scorecards: acceptance, on-time pickup, on-time delivery, damage, claims, cost, and empty miles.

### Client wiring

Supplier portal/native: tender board and carrier scorecard. Warehouse: appointment and receiving board. Factory: pickup readiness. External partner: only authenticated scoped APIs or EDI acknowledgements. Retailer sees only the customer-safe promise and tracking projection.

### Edge cases

- Carrier accepts but lacks required vehicle or temperature capability.
- Tender expires and another carrier is selected.
- External proof conflicts with driver or payload proof.
- Partner retries an ASN with a changed quantity.
- Carrier currency differs from order currency; never silently convert at checkout.

## 11. E7 - Sustainability and Cost-to-Serve

### Business outcome

Planning decisions expose service, financial, and environmental trade-offs.

### Backend logic

1. Capture route distance, vehicle class, load utilization, empty miles, fuel/energy factors, temperature energy, and packaging factors.
2. Calculate cost-to-serve by supplier, warehouse, retailer, route, order, and SKU using explicit currency and versioned factors.
3. Store factor versions with every calculation so historical reports remain reproducible.
4. Feed cost and carbon as optional objective terms into scenario and dispatch scoring; never override safety, fiscal, or service constraints.
5. Expose unavailable reasons when vehicle, route, or factor data is missing.

### Client wiring

Supplier and platform see aggregate trade-offs. Warehouse and factory see operational drivers. Retailer sees only permitted delivery-option impacts. Driver sees no financial or private supplier analytics.

### Edge cases

- Route geometry is missing or later corrected.
- Vehicle fuel factor is unknown.
- A multi-supplier parent order must retain child-level cost attribution.
- Historical factor changes must not rewrite old reports.

## 12. E8 - Commercial Adapter Gates

These are integration gates, not substitutes for missing product logic:

- `checkout_reads_this`: enable only after SSMR fiscal runtime and MarketPack agree.
- Live PSP execution: only after provider credentials, webhook verification, capture/refund soak, ledger reconciliation, and replay tests.
- PEPPOL/EDI/1C: certify dialect and acknowledgement behavior before advertising live exchange.
- SAML/SCIM: add only with tenant-scoped identity lifecycle and deprovisioning tests.
- Second cell: apply isolated Terraform/Kubernetes only after cell isolation, secrets, DNS, and rollback evidence.

No adapter may turn `planned`, `unkeyed`, or `no_live_keys` into a successful redirect or settlement.

## 13. Cross-Role Completion Contract

For every slice:

1. Route is mounted in the owning `*routes` package with role and scope middleware.
2. Request IDs come from JWT-bound scope, never body authority.
3. Spanner schema, indexes, repository, service, DTO, and migration agree.
4. Mutations use transaction plus outbox, idempotency, post-commit invalidation, and trace logging.
5. Kafka event has one canonical producer/consumer shape and aggregate-root key.
6. WebSocket/push fanout is scoped, authenticated, reconnect-safe, and version compatible.
7. Every client in the role row has API model, repository/view-model mapping, UI state handling, and offline/stale behavior.
8. Loading, empty, unavailable, stale, offline, and restricted states are explicit.
9. Unit, contract, role-row E2E, replay, concurrency, and Spanner-emulator tests cover the slice.
10. `go build ./...`, `go vet ./...`, relevant client builds, and gap-hunter checks pass before the slice is called complete.

## 14. Recommended First Coding Slice

Start with **E0-A: resolver convergence and availability read model**.

It has the highest leverage and lowest product risk because it improves existing quote, catalog, preview, and create behavior before introducing new autonomous actions. It also directly protects the most important business promise: do not sell inventory or delivery capability that the network cannot actually fulfill.

The first implementation batch should remain read-only, use existing warehouse coverage and inventory authorities, return honest source/staleness metadata, and ship tests before any enforcement flag is enabled.
