# pegasusX Enterprise Execution Plan

Last updated: 2026-05-22

## Plan Authority
1. This file is the canonical phased execution roadmap for `pegasusX/`.
2. Every non-trivial implementation batch must map to one or more plan anchors in this file.
3. After each batch, update plan-anchor status using `implemented`, `in progress`, `blocked`, or `deferred`.
4. If code reality diverges from the roadmap, code remains the source of truth and this file must be updated in the same change set.
5. Scope is `pegasusX/` only. `../pegasus/` remains reference-only.

## Program Goal
1. Deliver a single-supplier logistics ecosystem with role-row parity across supplier, retailer, driver, warehouse, factory, and payload surfaces.
2. Preserve architectural compatibility with the Pegasus reference so contracts, event names, and operating concepts stay migratable.
3. Reach production quality for high-volume operation: comfortable at 1M-request-class daily traffic, durable under bursty concurrent demand, and structurally ready for multi-million request growth.
4. Keep local Docker SSMR validation as the mandatory proof loop for the same critical flows expected in production cloud environments.
5. Use Spanner, Kafka, Redis, Firebase, Kubernetes, and Terraform as first-class platform primitives, not post-launch add-ons.

## Platform Commitments
1. Spanner is the durable source of truth for transactional state, audit-critical records, and index-backed operational reads.
2. Kafka is the mandatory backbone for reliable business-event fanout, replay-safe async processing, and decoupled worker execution.
3. Redis is the fast coordination layer for cache invalidation, websocket relay, rate limiting, perimeter membership, and low-latency operational lookups.
4. Firebase bearer auth is the mobile-facing identity path when enabled; supplier setup remains cookie-session based.
5. Kubernetes is the target runtime for stateless scale-out, graceful shutdown, and controlled rollout behavior.
6. Terraform is the canonical source for managed cloud infrastructure wiring and environment separation.

## Delivery Principles
1. Product work and engineering work ship together. A user-facing phase is not complete unless backend, contracts, UI, realtime, and operations are coherent.
2. Non-technical readiness is part of delivery. Support procedures, onboarding, training, policy, and incident handling are planned alongside code.
3. Reliability is designed in early. Outbox, idempotency, bounded retry, backoff with jitter, traceability, and cache correctness are mandatory.
4. Role-row parity is non-negotiable. A capability for one role must land across that role's supported clients unless intentionally flagged and time-boxed.
5. Local proof is mandatory. New infrastructure-sensitive behavior must pass the SSMR sandbox before it is treated as production-candidate.

## Execution Status Snapshot
1. `PX0-A1` Plan authority, sync enforcement, and reconciliation template — `implemented`.
2. `PX0-A2` Role-row capability and ownership ledger - `implemented`.
3. `PX0-A3` Contract and event baseline freeze - `implemented`.
4. `PX0-A4` Reliability and scale acceptance profile - `implemented`.
5. `PX0-A5` Support, release, and observability baseline - `in progress`.
6. `PX1-A1` Supplier identity, session, and billing durability baseline - `in progress`.
7. `PX1-A2` Supplier topology builder and node provisioning - `in progress`.
8. `PX2-A1` Retailer commerce and order-capture baseline - `in progress`.
9. `PX3-A1` Payment session, attempt, and webhook durability baseline - `in progress`.
10. `PX7-A1` SSMR sandbox and infra proof gate baseline - `in progress`.

## Ordered Plan Anchors

### PX-0 anchors
1. `PX0-A1` Governance authority and sync enforcement
  Status: `implemented`
  Non-technical: establish one roadmap, one reconciliation model, and one documentation sync rule for pegasusX.
  Technical: wire `context/plan.md` into instruction files, context docs, and architecture graph as the canonical execution ledger.
  Dependencies: none.
  Exit evidence: plan authority is referenced by local doctrine and included in the sync set.
2. `PX0-A2` Role-row capability and ownership ledger
  Status: `implemented`
  Non-technical: define what each role owns, what support covers, and what outcomes each app must provide.
  Technical: produce a capability matrix mapping roles, apps, endpoints, events, offline expectations, and realtime consumers.
  Dependencies: `PX0-A1`.
  Exit evidence: role rows have a capability ledger that future feature work can diff against.
3. `PX0-A3` Contract and event baseline freeze
  Status: `implemented`
  Non-technical: make product and support terminology stable enough for training and rollout planning.
  Technical: lock route families, event names, aggregate IDs, DTO ownership, and shared-codegen expectations across TS, Kotlin, and Swift.
  Dependencies: `PX0-A1`.
  Exit evidence: canonical route and event inventory exists and can be validated before edits.
4. `PX0-A4` Reliability and scale acceptance profile
  Status: `implemented`
  Non-technical: define what “ready for 1M-request-class operation” means in business and support terms.
  Technical: set acceptance budgets for Spanner latency and indexing, Kafka lag and replay safety, Redis invalidation correctness, websocket fanout, webhook replay, and Kubernetes rollout behavior.
  Dependencies: `PX0-A1`.
  Exit evidence: every later phase has measurable reliability gates instead of vague quality goals.
5. `PX0-A5` Support, release, and observability baseline
  Status: `in progress`
  Non-technical: define hypercare ownership, incident paths, escalation ladders, and launch-support coverage.
  Technical: inventory dashboards, alerts, audit trails, rollback steps, and release evidence required before launch.
  Dependencies: `PX0-A2`, `PX0-A4`.
  Exit evidence: support and release readiness are visible before the product enters hardening.

### PX-1 anchors
1. `PX1-A1` Supplier identity, session, and billing durable baseline
  Status: `in progress`
  Non-technical: onboarding and billing recovery are documented for operators and support.
  Technical: durable supplier signup/login/profile/billing/session flows with claims-safe auth and idempotent retries.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: supplier setup survives retries, partial failure, and re-login without manual repair.
2. `PX1-A2` Supplier topology builder and node provisioning
  Status: `in progress`
  Non-technical: factories and warehouses are represented as real operating nodes, not setup placeholders.
  Technical: topology creation, validation, and downstream visibility for warehouse/factory role rows and contracts.
  Dependencies: `PX1-A1`.
  Exit evidence: created nodes are immediately consumable by downstream apps and services.
3. `PX1-A3` Supplier control tower v1
  Status: `not-started`
  Non-technical: supplier operators can monitor configuration, inventory, earnings, and order oversight from one surface.
  Technical: dashboard, inventory, earnings, orders, and audit surfaces wired to stable supplier contracts.
  Dependencies: `PX1-A1`, `PX1-A2`.
  Exit evidence: supplier portal can act as the daily operating cockpit.
4. `PX1-A4` Supplier org, staff, and fleet onboarding contracts
  Status: `deferred`
  Non-technical: define how staff, drivers, vehicles, and node admins are introduced into the system.
  Technical: contract and workflow baseline for org members, fleet entities, and node-scoped login readiness.
  Dependencies: `PX1-A2`.
  Exit evidence: downstream node and driver work is not blocked by missing supplier-side entity definitions.

### PX-2 anchors
1. `PX2-A1` Retailer registration and seeded supplier relationship
  Status: `in progress`
  Non-technical: retailers can join the ecosystem and understand their relationship to the single supplier.
  Technical: retailer auth/profile/supplier linkage baseline across desktop and native apps.
  Dependencies: `PX0-A3`, `PX1-A1`.
  Exit evidence: retailer identity and supplier association are stable across clients.
2. `PX2-A2` Cart sync, serviceability, and order capture
  Status: `in progress`
  Non-technical: retailers can build carts, place orders, and understand why an order is accepted or blocked.
  Technical: cart sync, zone validation, nearest-warehouse selection, order creation, and duplicate-request safety.
  Dependencies: `PX2-A1`, `PX0-A4`.
  Exit evidence: retailer commerce is resilient under reconnects, duplicate taps, and spatial edge cases.
3. `PX2-A3` Retailer pricing and catalog integrity
  Status: `not-started`
  Non-technical: catalog and pricing are explainable to supplier operators and retailer support staff.
  Technical: supplier pricing rules, retailer overrides, catalog contracts, and cross-client display consistency.
  Dependencies: `PX1-A3`, `PX2-A1`.
  Exit evidence: pricing authority is backend-controlled and surfaced consistently everywhere.
4. `PX2-A4` Retailer post-order decision flows
  Status: `not-started`
  Non-technical: cancellation, AI suggestions, preorder, pending-payment, and tracking flows are supportable and understandable.
  Technical: confirm/reject AI flows, cancel/request-cancel, preorder edit/confirm, active fulfillment, and tracking surfaces.
  Dependencies: `PX2-A2`, `PX3-A1`.
  Exit evidence: post-order retailer actions stay coherent across desktop and mobile.

### PX-3 anchors
1. `PX3-A1` Payment session, attempt, and webhook durability
  Status: `in progress`
  Non-technical: finance and support can explain payment state transitions and replay behavior.
  Technical: durable session/attempt/webhook storage, provider-exact verification, idempotent chargeback and reversal flows.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: one payment intent leads to one durable business outcome.
2. `PX3-A2` Settlement, ledger, and reconciliation authority
  Status: `not-started`
  Non-technical: treasury and support have a clear source of truth for cleared, pending, disputed, and reversed funds.
  Technical: double-entry-ready ledger model, settlement summaries, reconciliation reports, and mismatch handling.
  Dependencies: `PX3-A1`.
  Exit evidence: finance can reconcile without ad hoc database interpretation.
3. `PX3-A3` Supplier finance and dispute operations
  Status: `deferred`
  Non-technical: supplier operators have actionable payment and dispute visibility.
  Technical: supplier-facing payment status, exception, dispute, and treasury UI plus event-driven updates.
  Dependencies: `PX3-A2`, `PX1-A3`.
  Exit evidence: payment issues are operationally manageable from product surfaces.

### PX-4 anchors
1. `PX4-A1` Warehouse operational durability
  Status: `not-started`
  Non-technical: warehouse operators can trust inventory, order queues, dispatch preview, and supply requests.
  Technical: durable warehouse repository paths, cache discipline, node scoping, and realtime updates.
  Dependencies: `PX1-A2`, `PX2-A2`.
  Exit evidence: warehouse operations are not scaffold-only.
2. `PX4-A2` Factory manifest lifecycle durability
  Status: `not-started`
  Non-technical: factories can move from transfer planning to dispatch with visible state and recoverable exceptions.
  Technical: durable manifest lifecycle, dispatch, rebalance, cancellation, and exception handling.
  Dependencies: `PX4-A1`, `PX1-A4`.
  Exit evidence: factory movement is durable, replay-safe, and observable.
3. `PX4-A3` Payload execution and reassignment durability
  Status: `not-started`
  Non-technical: payload teams can fix physical loading reality without shadow tools.
  Technical: durable load/start/inject/seal/exception/reassign flows with source-target cache invalidation and typed fanout.
  Dependencies: `PX4-A2`.
  Exit evidence: payload repair flows are operationally trustworthy.

### PX-5 anchors
1. `PX5-A1` Driver execution baseline
  Status: `not-started`
  Non-technical: drivers can work quickly, safely, and with clear manifest expectations.
  Technical: profile, availability, manifest gate, pending collections, earnings, and delivery-state transitions.
  Dependencies: `PX4-A2`, `PX1-A4`.
  Exit evidence: driver execution is no longer blocked by upstream scaffolding gaps.
2. `PX5-A2` Telemetry and live tracking integrity
  Status: `not-started`
  Non-technical: supplier and retailer users can trust what “live” tracking means.
  Technical: authenticated telemetry ingress, websocket fanout, reconnect safety, and tracking contract parity.
  Dependencies: `PX5-A1`, `PX0-A4`.
  Exit evidence: live route progress survives network instability and scale bursts.
3. `PX5-A3` Retailer receipt, proof, and collection integrity
  Status: `deferred`
  Non-technical: delivery completion, cash collection, and receipt-side issues are supportable and auditable.
  Technical: geofence-sensitive completion, collection workflows, proof/receipt events, and retailer-visible completion state.
  Dependencies: `PX5-A1`, `PX3-A2`.
  Exit evidence: post-delivery disputes can be resolved from system truth.

### PX-6 anchors
1. `PX6-A1` Event-driven AI worker substrate
  Status: `not-started`
  Non-technical: AI operates as operator assistance, not opaque automation.
  Technical: Kafka-driven worker jobs, replay-safe advisory writes, status visibility, and bounded retry semantics.
  Dependencies: `PX0-A3`, `PX0-A4`.
  Exit evidence: AI work stays off the synchronous request path.
2. `PX6-A2` Forecasting and recommendation products
  Status: `deferred`
  Non-technical: supplier teams can see useful recommendations and retailer AI remains opt-in.
  Technical: forecast, replenishment, dispatch, and recommendation outputs plus UI surfaces to review them.
  Dependencies: `PX6-A1`, `PX2-A4`, `PX4-A1`.
  Exit evidence: recommendations are visible, bounded, and reviewable.
3. `PX6-A3` Human override and explainability
  Status: `deferred`
  Non-technical: operators understand why the system suggests something and can override it.
  Technical: explanation fields, audit trail, manual override capture, and safe rollback of automated suggestions.
  Dependencies: `PX6-A2`.
  Exit evidence: AI assistance does not create black-box operational risk.

### PX-7 anchors
1. `PX7-A1` SSMR and production-parity proof gate
  Status: `in progress`
  Non-technical: stakeholders can trust local proof as a precursor to cloud deployment.
  Technical: isolated Spanner/Redis/Kafka setup, bootstrap proofs, health checks, spatial checks, and Kafka round-trip validation.
  Dependencies: `PX0-A4`.
  Exit evidence: infra-sensitive changes are blocked from merging without sandbox proof.
2. `PX7-A2` Load, resilience, and security certification
  Status: `not-started`
  Non-technical: launch readiness is judged by evidence, not optimism.
  Technical: load tests, replay drills, Redis/Kafka/Spanner failure-mode tests, auth hardening, and webhook/security review.
  Dependencies: `PX3-A2`, `PX5-A2`, `PX7-A1`.
  Exit evidence: the platform is certified for burst traffic, failures, and attack-surface review.
3. `PX7-A3` Launch readiness, support, and hypercare
  Status: `deferred`
  Non-technical: release owners, support, and hypercare know exactly what to do at launch.
  Technical: final runbooks, dashboards, release gates, rollback rehearsal, and post-launch monitoring.
  Dependencies: `PX0-A5`, `PX7-A2`.
  Exit evidence: launch approval is operationally supportable.

## Cross-Phase Workstreams

### Product and operations
1. Role definitions, SOPs, escalation rules, approval policies, and service-level expectations.
2. Onboarding, training, support scripts, hypercare, and operator-facing recovery paths.
3. Exception management, dispute handling, financial controls, and audit-ready business processes.

### Backend and data
1. Domain services, Spanner schema, repositories, transactional outbox, idempotency, websocket fanout, and worker orchestration.
2. Event contracts, API DTOs, versioning, auth scope enforcement, and trace propagation.
3. Payment, ledger, reconciliation, and operational analytics integrity.

### Frontend, desktop, and mobile
1. Supplier portal, retailer desktop/mobile, driver mobile, warehouse portal/mobile, factory portal/mobile, and payload terminal/tablet parity.
2. Loading, empty, offline, stale, restricted, and failure states on every live surface.
3. Realtime consumption, reconnect behavior, local persistence, and role-appropriate task flows.

### Platform and reliability
1. Kafka partitioning, Redis invalidation discipline, Spanner index coverage, Kubernetes readiness, and rollout safety.
2. Load shedding, rate limiting, circuit protection, observability, and chaos/testing gates.
3. CI gates, SSMR smoke proof, rollback discipline, and incident response evidence.

## First Delivery Batches
1. `B01` PX-0 execution hardening
  Status: `implemented`
  Anchors: `PX0-A2`, `PX0-A3`, `PX0-A4`.
  Non-technical deliverables: role-row ownership matrix, support vocabulary baseline, launch KPI draft, release and escalation expectations.
  Technical deliverables: capability ledger, route and event inventory freeze, shared-contract ownership map, reliability acceptance matrix, gap list for missing guards or parity.
  Primary surfaces: `context/plan.md`, `context/architecture.md`, `context/technology-inventory.*`, `.github/*.md`.
  Validation: docs sync complete, JSON docs valid, anchor statuses updated, no ambiguity on role rows and route authority.
2. `B02` Supplier bootstrap durability and topology batch
  Status: `in progress`
  Anchors: `PX1-A1`, `PX1-A2`.
  Non-technical deliverables: supplier onboarding SOP, billing recovery script, topology-entry support guide.
  Technical deliverables: durable signup/login/profile/billing flows, topology persistence, downstream warehouse/factory visibility, contract-safe responses.
  Primary surfaces: `apps/backend-go/supplier/**`, `apps/supplier-portal/**`, `packages/types`, `packages/api-client`.
  Validation: targeted backend tests, portal type/build checks, SSMR proof if infra-sensitive changes are introduced.
  Current delta (2026-05-22): supplier Spanner repository now persists rich profile + topology (`SupplierProfiles`, warehouse/factory read/write), supplierroutes exposes `GET|PUT /v1/supplier/topology`, supplier portal register/billing payloads now align with backend contract and idempotency header, supplier portal now has a same-origin `/api/*` proxy transport to backend, and shared supplier contract/client bridges are now exported in `packages/types` + `packages/api-client`.
3. `B03` Retailer commerce and pricing batch
  Status: `queued`
  Anchors: `PX2-A1`, `PX2-A2`, `PX2-A3`.
  Non-technical deliverables: retailer onboarding/support flows, pricing authority rules, zone-miss communication policy.
  Technical deliverables: retailer registration/profile linkage, cart sync, pricing integrity, serviceability, order capture, warehouse assignment, duplicate-tap safety.
  Primary surfaces: `apps/backend-go/retailer/**`, `apps/backend-go/order/**`, `apps/backend-go/supplier/**pricing*`, retailer apps, shared contracts.
  Validation: retailer/backend tests, contract diff check, client build/type checks, spatial/SSMR proof when order-zone logic changes.
4. `B04` Payment integrity batch
  Status: `queued`
  Anchors: `PX3-A1`, `PX3-A2`.
  Non-technical deliverables: payment exception SOP, finance support workflow, dispute classification vocabulary.
  Technical deliverables: checkout consistency, webhook replay safety, attempt metadata continuity, settlement and ledger authority groundwork.
  Primary surfaces: `apps/backend-go/payment/**`, payment-facing retailer and supplier surfaces, contract docs.
  Validation: backend payment tests, webhook path validation, idempotency coverage, SSMR proof for infra-sensitive payment changes.
5. `B05` Node operations durability batch
  Status: `queued`
  Anchors: `PX4-A1`, `PX4-A2`, `PX4-A3`.
  Non-technical deliverables: warehouse/factory/payload SOPs for exceptions, reassignment, and transfer cancellation.
  Technical deliverables: durable warehouse/factory/payload repositories, manifest lifecycle integrity, reassignment correctness, realtime parity.
  Primary surfaces: `apps/backend-go/{warehouse,factory,payload}/**`, node apps, payload terminal, shared contracts.
  Validation: focused backend tests, event/contract validation, websocket parity checks, SSMR proof when infra or realtime wiring changes.
6. `B06` Driver and live-delivery batch
  Status: `queued`
  Anchors: `PX5-A1`, `PX5-A2`.
  Non-technical deliverables: driver support playbooks, live-tracking expectations, delivery escalation policy.
  Technical deliverables: driver execution parity, telemetry integrity, live tracking, reconnect safety, manifest-to-delivery completion chain.
  Primary surfaces: `apps/backend-go/driver/**`, telemetry paths, driver apps, retailer tracking surfaces.
  Validation: targeted backend tests, mobile build checks where available, websocket/telemetry validation, load-oriented proof for live updates.

## Phase Roadmap

### PX-0: Governance, Contracts, and Delivery Control

#### Objective
Create the execution control plane for pegasusX so roadmap, architecture, contracts, and support expectations stay synchronized while the product grows.

#### Business and operational outcomes
1. One clear roadmap exists for product, engineering, and operations.
2. Stakeholders can see what each phase is meant to achieve and what evidence closes it.
3. Support, rollout, and escalation ownership are defined before deeper workflow automation lands.

#### Engineering scope

Backend and data:
1. Lock canonical route families, event names, entity IDs, and role claims.
2. Establish the baseline mutation contract: auth, scope, validation, RW transaction, outbox emit, cache invalidation, trace logging, additive DTO.
3. Keep SSMR as the proof gate for Spanner, Redis, Kafka, backend health, and spatial readiness.

Frontend, desktop, and mobile:
1. Freeze the role-row surface matrix and the platform choices per role.
2. Define shared contract consumption rules for TS, Kotlin, and Swift clients.
3. Set cross-client parity expectations before adding deeper features.

Platform and operations:
1. Define target load envelope: 1M-request-class daily volume, thousands of concurrent devices/operators, burst-safe realtime and webhook handling.
2. Lock reliability requirements for Kafka delivery, Redis invalidation, Spanner indexing, and Kubernetes statelessness.
3. Establish observability, support, release, and rollback expectations.

#### Exit gate
1. Canonical roadmap approved and referenced by local instruction files.
2. Sync protocol defined for architecture, inventory, parity, and plan files.
3. SSMR remains the mandatory local proof step for infra-sensitive work.

### PX-1: Supplier Company Bootstrap and Control Tower Foundation

#### Objective
Turn the supplier row into a complete operating business surface: identity, network topology, billing, and core control-tower visibility.

#### Business and operational outcomes
1. The supplier can sign up, log in, configure the company, and finish billing without manual engineering help.
2. Factories, warehouses, staffing model, and commercial setup are captured as real operating topology.
3. Internal teams have a clear control tower for finance, inventory, and order oversight.

#### Engineering scope

Backend and data:
1. Supplier auth, profile, billing setup, payment configuration, and topology creation must persist durably in Spanner.
2. Supplier-owned entities keep `SupplierId` everywhere for future compatibility.
3. Supplier configuration changes must emit outbox events where downstream state depends on them and invalidate affected caches post-commit.

Frontend and desktop:
1. Supplier onboarding wizard covers account, topology, business, categories, and billing gate.
2. Supplier portal dashboard establishes inventory, earnings, order review, and configuration entry points.
3. Error and recovery UX must exist for incomplete setup, invalid billing, and topology mistakes.

Mobile and node apps:
1. Warehouse and factory login readiness must respect the supplier-created topology.
2. Mobile contract shapes for supplier-owned nodes must be aligned before deeper node workflows ship.

Platform and operations:
1. Supplier setup must be idempotent and replay-safe.
2. Admin-support runbooks must cover failed onboarding, billing misconfiguration, and incorrect topology data.

#### Exit gate
1. A supplier can bootstrap the company and network end-to-end.
2. Warehouse and factory nodes created during onboarding are visible to downstream role surfaces.
3. Support can recover a broken onboarding without direct database surgery.

### PX-2: Retailer Commerce, Catalog, and Order Capture

#### Objective
Create a reliable retailer commerce layer that supports discovery, cart continuity, order creation, and order-side decision flows.

#### Business and operational outcomes
1. Retailers can register, manage profile, choose the seeded supplier, build carts, and place orders across mobile and desktop surfaces.
2. Retailers get clear outcomes for valid orders, service-zone misses, pending payments, AI suggestions, cancellations, and preorders.
3. Retailer support can explain why an order was accepted, blocked, changed, or cancelled.

#### Engineering scope

Backend and data:
1. Catalog, retailer pricing, cart sync, profile, family-members, order create, cancel/request-cancel, preorder, AI confirm/reject, and tracking paths must be coherent.
2. Order creation must derive serviceability from backend-controlled spatial logic and assign the right warehouse candidate.
3. Retailer commerce data must remain additive and cross-client safe.

Frontend and desktop:
1. Retailer apps expose supplier selection, cart sync, checkout initiation, order history, active fulfillment, and payment state.
2. Desktop and mobile retain feature parity while using platform-appropriate UI controls.
3. Loading, empty cart, zone miss, stale cart, and offline resume states are mandatory.

Mobile and realtime:
1. Retailer surfaces must stay consistent across Android, iOS, and desktop.
2. Websocket updates for order status, payment state, and cart sync must drive visible UI refresh.
3. Reconnect behavior must not duplicate cart or payment actions.

Platform and operations:
1. Order creation and cart sync must be safe under burst traffic and duplicate taps.
2. Support scripts must cover zone-miss, cart drift, stuck pending-payment, and cancel-policy questions.

#### Exit gate
1. Retailers can move from registration to successful order placement on every supported retailer surface.
2. Zone and warehouse-selection logic are backend authoritative and observable.
3. Duplicate requests do not create duplicate business outcomes.

### PX-3: Payment, Ledger, Settlement, and Reconciliation Integrity

#### Objective
Make money movement safe, explainable, and operationally manageable from checkout through webhook settlement and disputes.

#### Business and operational outcomes
1. Retailers can pay with supported gateways and understand whether the order is awaiting payment, cleared, failed, or disputed.
2. Suppliers can monitor settlement, payment exceptions, and chargeback/reversal outcomes.
3. Finance and support teams can reconcile payment state without manual guesswork.

#### Engineering scope

Backend and data:
1. Payment sessions, attempts, chargebacks, reversals, and webhooks persist durably with idempotency and outbox discipline.
2. Gateway execution uses bounded retries, typed policy errors, and provider-exact webhook verification.
3. Ledger and settlement data must support future double-entry and treasury reporting with no float-based money math.

Frontend and desktop:
1. Retailer surfaces show pending payment, cleared payment, retry, and failure recovery states.
2. Supplier finance surfaces expose earnings, payment status, and dispute workflow visibility.
3. Payment UX must clearly separate redirect, hosted, direct, and retry outcomes.

Mobile and realtime:
1. Payment-required and payment-cleared updates must propagate to retailer and supplier surfaces in near real time.
2. Older clients must tolerate additive payment metadata without breaking.

Platform and operations:
1. Checkout and webhook paths must stay safe under replay and concurrent retries.
2. Support and finance runbooks must cover manual review, settlement mismatch, chargeback handling, and refund/reversal communication.

#### Exit gate
1. Checkout to settlement produces one durable business outcome per payment intent.
2. Webhook replay and duplicate client calls remain safe.
3. Finance can explain payment state transitions from stored records and emitted events.

### PX-4: Warehouse, Factory, and Payload Operational Execution

#### Objective
Connect upstream production and local fulfillment into a controlled manifest-driven execution model.

#### Business and operational outcomes
1. Warehouses can manage local inventory, supply requests, dispatch preview, and dispatch locks.
2. Factories can prepare transfers, manage manifests, load, seal, dispatch, complete, rebalance, or cancel movement.
3. Payload staff can validate truck readiness, load manifests, inject late orders before seal, raise exceptions, and reassign when physical reality changes.

#### Engineering scope

Backend and data:
1. Warehouse, factory, and payload repositories must move from scaffold-only behavior toward durable transactional state.
2. Manifest lifecycle, exception, reassignment, and dispatch operations must emit additive events and invalidate source/target caches correctly.
3. Supply-request and dispatch-preview logic must stay node-scoped and claims-derived.

Frontend and desktop:
1. Warehouse and factory portals expose operational dashboards, queues, previews, and exception views.
2. Payload terminal and tablet experiences focus on fast load-time execution and exception repair.
3. Every operational surface needs explicit empty, stale, locked, and failure states.

Mobile and realtime:
1. Warehouse/factory mobile apps and payload devices consume manifest lifecycle and exception events in real time.
2. Reassignment and manifest mutation envelopes must stay contract-safe across all clients.

Platform and operations:
1. Manifest operations must remain safe under concurrent operator activity.
2. SOPs must exist for overflow, damaged goods, late order injection, seal rejection, and transfer cancellation.

#### Exit gate
1. Physical transfer and loading workflows can be executed without spreadsheet or chat-based shadow systems.
2. Manifest state changes are durable, visible, and replay-safe.
3. Payload and factory/warehouse teams share one operational truth.

### PX-5: Driver Delivery, Telemetry, and Retailer Receipt

#### Objective
Close the loop from manifest dispatch to live delivery, proof, collection, and retailer-facing receipt.

#### Business and operational outcomes
1. Drivers can go available, receive manifests, execute stops, collect cash when required, and complete deliveries with clear rules.
2. Retailers can track live fulfillment, see arrival/progress, and handle receipt-side actions.
3. Operations teams can understand where drivers are, what route they are executing, and which orders are blocked or disputed.

#### Engineering scope

Backend and data:
1. Driver profile, history, earnings, availability, pending collections, manifest gate, manifest detail, and delivery-state transitions must be authoritative.
2. Telemetry ingestion must enforce authenticated driver identity and fan out safely to interested rooms.
3. Geofence-sensitive actions and completion logic must protect financial and operational integrity.

Frontend and mobile:
1. Driver apps prioritize low-latency execution, manifest clarity, and recovery from poor connectivity.
2. Retailer tracking and active-fulfillment views must reflect live delivery progress and payment/settlement state.
3. Warehouse/factory/supplier surfaces must expose driver state and delivery exceptions without requiring database inspection.

Realtime and platform:
1. Telemetry, order status, payment state, and manifest updates must reconnect cleanly after network drops.
2. Thousands of active devices must not overwhelm websocket or telemetry pipelines.
3. Traceability from request to event to websocket delivery remains mandatory.

#### Exit gate
1. A delivery can move end-to-end from manifest dispatch to completed retailer receipt.
2. Driver connectivity loss or reconnect does not create silent data corruption.
3. Support can explain live-delivery state from platform records.

### PX-6: Intelligence, Forecasting, and Planning Automation

#### Objective
Introduce AI and planning assistance that improves operations without taking control away from human operators.

#### Business and operational outcomes
1. Supplier teams receive useful forecasting, replenishment, dispatch, and exception-prioritization help.
2. Retailers can benefit from AI-suggested orders or reorder support without forced automation.
3. Human operators retain override authority where policy allows.

#### Engineering scope

Backend and data:
1. AI worker consumes Kafka events and produces replay-safe advisory outputs rather than synchronous request-path side effects.
2. Forecasting, recommendation, and optimization jobs run asynchronously with bounded retries, status visibility, and audit trail.
3. Planning outputs must be explainable enough for operators to trust or reject them.

Frontend and clients:
1. Supplier control surfaces expose forecast, dispatch guidance, exception ranking, and action history.
2. Retailer AI order suggestion flows remain explicit opt-in/confirm-reject interactions.
3. Node apps only surface actionable recommendations that fit the role.

Platform and operations:
1. AI/optimization flows must never block the checkout or order-mutation critical path.
2. Model and recommendation quality must be measurable, reversible, and supportable.

#### Exit gate
1. AI assistance reduces operator effort without creating opaque failure modes.
2. Async worker flows are observable, replay-safe, and operationally bounded.
3. Manual override remains available for high-consequence actions.

### PX-7: Scale, Security, and Launch Readiness

#### Objective
Certify pegasusX for production launch with scale, resilience, support, and governance readiness.

#### Business and operational outcomes
1. The system is ready for live supplier and retailer traffic with clear support ownership.
2. On-call, incident, rollback, and hypercare processes are ready before launch.
3. Leadership has clear go/no-go evidence based on quality and resilience, not optimism.

#### Engineering scope

Platform and reliability:
1. Run load tests for 1M-request-class daily traffic, burst concurrency, realtime fanout, webhook replay, and queue backlog behavior.
2. Validate Kafka partition strategy, Spanner hotspot posture, Redis failure modes, and Kubernetes drain/rollout behavior.
3. Confirm autoscaling, load shedding, circuit protection, and observability thresholds.

Security and compliance:
1. Finalize auth hardening, secret rotation, audit trail completeness, least-privilege verification, and webhook security posture.
2. Confirm mobile identity and session controls remain safe under rotation and replay.

Product and support:
1. Complete release checklists, training, documentation, support scripts, and stakeholder communication.
2. Prepare hypercare dashboards and escalation routing for launch week.

#### Exit gate
1. Performance, security, and rollback gates pass with evidence.
2. Support and hypercare are staffed and documented.
3. Launch approval is based on measured readiness, not feature count.

## Phase Synchronization Protocol
1. Before each meaningful execution batch, read this file and map the work to one or more plan anchors.
2. Prefer the active or earliest queued delivery batch unless the user explicitly reprioritizes.
3. After each batch, update statuses in this file and sync the context/docs set in the same change set when architecture or delivery truth changed.
4. When behavior diverges from the Pegasus reference, update `context/parity-ledger.md` in the same batch.
5. When the roadmap changes, update local instruction files so future agents follow the revised execution model.

## Documentation Sync Set
1. `.github/ACT.md`
2. `.github/copilot-instructions.md`
3. `.github/gemini-instructions.md`
4. `context/plan.md`
5. `context/architecture.md`
6. `context/architecture-graph.json`
7. `context/technology-inventory.md`
8. `context/technology-inventory.json`
9. `context/parity-ledger.md` when divergence from Pegasus changes

## Execution Checkpoint Template
- Checkpoint timestamp:
- Scope:
- Plan anchors touched:
- Files changed:
- Contracts or events impacted:
- Validation run:
- Status by anchor:
  - implemented:
  - in progress:
  - blocked:
  - deferred:
- Follow-up sync required: