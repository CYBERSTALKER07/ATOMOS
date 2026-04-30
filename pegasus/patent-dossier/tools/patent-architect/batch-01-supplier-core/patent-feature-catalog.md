# Comprehensive Feature Catalog

Batch: Batch 01 - Supplier Web Core
Coverage Note: Catalog is exhaustive for discovered Supplier web batch routes and extracted core backend orchestration mechanisms in this batch.

## Supplier Portal Feature Inventory (Batch 01)

| Feature Name | Technical Mechanism (Internal) | Patent Description (Official) | Node Dependency |
|---|---|---|---|
| Supplier Intelligence Dashboard | Supplier dashboard route with forecast summary, KPI derivation, chart rendering, and table aggregation from analytics endpoints | A multi-panel operational intelligence cockpit that fuses near-term demand signals with historical throughput and product performance indicators | Supplier node, warehouse data feeds |
| Analytics Hub | Supplier analytics route with metric and trend summaries | A supervisory analytics workspace for continuous operational monitoring and strategic tuning | Supplier node |
| Advanced Demand Analytics | Demand-history route with predicted-versus-actual chart and upcoming order table | A predictive comparison instrument for quantifying forecast alignment and imminent demand obligations | Supplier node, retailer demand stream |
| Catalog Management | Catalog route with searchable product registry and status controls | A digital goods registry that governs commercial and operational readiness of offerable units | Supplier node |
| Country Override Rules | Country override route with jurisdiction-specific control values | A jurisdictional policy override mechanism that localizes operating constraints and commercial behavior by country | Supplier node, jurisdiction context |
| Retailer Relationship Console | CRM route with account list and interaction timeline | A relationship operations console for managing retailer engagement, follow-up actions, and service posture | Supplier node, retailer accounts |
| Delivery Zone Planner | Delivery-zone route with map-centric zone editing | A geospatial service-area controller that encodes distribution coverage and execution boundaries | Supplier node, geospatial region |
| Depot Reconciliation Ledger | Depot reconciliation route with variance review | A variance adjudication interface that aligns planned and observed logistical outcomes | Supplier node, warehouse node |
| Dispatch Alias Redirect | Dispatch route alias redirecting to canonical order operations | A route alias normalization mechanism that preserves backward compatibility while enforcing a single operational entry point | Supplier node |
| Shop-Closed Exception Queue | Shop-closed exception route with triage controls | An exception lifecycle board for closure anomalies and remedial dispatch decisions | Supplier node, retailer endpoint |
| Factory Registry | Factory management route with node visibility and status | A production-node registry for capacity-aware planning and source-node governance | Supplier node, factory nodes |
| Fleet Control Surface | Fleet route with driver and vehicle assignments | A mobility orchestration console for workforce and transport asset coordination | Supplier node, driver nodes, vehicle nodes |
| Geospatial Report | Geo-report route with cell-level operational mapping | A spatial diagnostics view that correlates delivery behavior with hex-cell activity clusters | Supplier node, geospatial cells |
| Inventory Operations | Inventory route with stock and deficit visibility | A stock-governance workspace that reveals current inventory posture against policy thresholds | Supplier node, warehouse nodes |
| Manifest Exception Queue | Manifest-exception route for anomaly remediation | A manifest anomaly management surface for preserving execution integrity during irregular events | Supplier node, manifest entities |
| Manifest Alias Redirect | Manifest route alias redirecting to canonical order operations | A compatibility-preserving route bridge that consolidates manifest and order control paths | Supplier node |
| Supplier Onboarding Progress | Onboarding route tracking post-registration completion | A staged readiness progression system that transitions a new supplier into full operational state | Supplier node |
| Order Operations Console | Orders route with state-driven queue and detail interactions | A transaction command center for lifecycle management of delivery obligations | Supplier node, retailer nodes, driver nodes |
| Organization Profile Console | Organization route with legal and governance profile controls | A governance profile manager for institutional identity and administrative policy control | Supplier node |
| Payment Gateway Configuration | Payment configuration route for gateway credentials and controls | A settlement-channel provisioning surface for secure payment rail activation and validation | Supplier node, payment gateway |
| Pricing Management | Pricing route with product-level pricing matrix | A configurable commercial rule matrix for base pricing and policy-driven pricing behavior | Supplier node |
| Retailer Pricing Overrides | Retailer override route with account-level adjustments | A differentiated pricing override capability for account-specific commercial terms | Supplier node, retailer accounts |
| Product Registry | Product list route with indexing and bulk actions | A product corpus management layer for maintaining operationally available merchandise units | Supplier node |
| Product Detail Inspector | Product detail route with sectioned controls | A granular product instrumentation view for attribute control, logistics metadata, and pricing detail | Supplier node |
| Supplier Profile | Profile route with identity and contact maintenance | An institutional identity maintenance surface for supplier-level profile continuity | Supplier node |
| Returns Processing | Returns route with queue and resolution controls | A reverse-logistics adjudication interface for return and dispute outcomes | Supplier node, retailer endpoint |
| Supplier Settings | Settings route with policy toggles and threshold controls | A policy command layer for configuring dispatch, notification, and governance defaults | Supplier node |
| Staff Administration | Staff route with role and status controls | A personnel governance subsystem for operational role assignment and access posture management | Supplier node, warehouse nodes, factory nodes |
| Supply Lane Network | Supply-lanes route with origin-destination control | A lane-definition matrix for governing movement paths between operational nodes | Supplier node, warehouse nodes, factory nodes |
| Warehouse Registry | Warehouse route with node and utilization management | A storage-node registry for capacity stewardship and logistics staging governance | Supplier node, warehouse nodes |
| Supplier Registration Wizard | Registration route with staged account, location, business, and category capture | A multi-step enrollment apparatus that transforms prospective operator data into an authenticated supplier identity | Supplier node |
| Supplier Billing Setup | Billing setup route with bank and gateway onboarding | A post-enrollment settlement configuration stage for treasury readiness and payment interoperability | Supplier node, payment gateway |

## Core Backend Mechanism Inventory (Batch 01)

| Feature Name | Technical Mechanism (Internal) | Patent Description (Official) | Node Dependency |
|---|---|---|---|
| Hex Cell Assignment | Standard-resolution hex index assignment for geospatial coordinates | A deterministic geospatial indexing substrate that converts location points into dispatch-ready spatial partitions | Supplier node, warehouse node, retailer location |
| Neighbor Ring Coverage Expansion | Ring-based expansion around origin cell for radius capture | A spatial neighborhood derivation method that computes service-relevant nearby cells for candidate assignment | Geospatial topology |
| Capacity Buffer Dispatch Rule | Effective volume check using configured capacity buffer | A safety-envelope dispatch rule that preserves execution reliability under volumetric uncertainty | Driver node, vehicle node |
| Smallest-Fit Vehicle Selection | Escalation-based vehicle matching over sorted fleet capacities | A constrained resource matching algorithm that minimizes unused capacity while preserving feasibility | Driver node, vehicle node |
| Oversized Load Segmentation | Order chunking for volumes exceeding maximal effective vehicle envelope | A partitioning mechanism that transforms infeasible large loads into executable sub-deliveries | Driver node, vehicle node |
| Draft Manifest Persistence | Transactional insert of manifest and stop-sequence records | A pre-execution manifest formalization step that establishes deterministic route and stop order before movement | Supplier node, manifest entity |
| Dispatch Lock Acquisition | Dispatch lock row persistence with role-bound scope derivation | A manual governance lock that temporarily supersedes autonomous assignment authority | Supplier node, warehouse node, factory node |
| Freeze Lock Event Emission | Freeze lock acquired and released signals in dedicated stream path | A synchronization signal channel that coordinates manual override state with autonomous worker behavior | Supplier node, autonomous worker |
| Idempotent Mutation Guard | Request-key cache lookup, in-flight lock, and successful response replay | A replay-protection envelope that prevents duplicate state mutation from transport retries | Any mutating endpoint |
| Transactional Outbox Append | In-transaction outbox row creation with aggregate-root identity | A durability bridge that unifies business-state commit and downstream signal publication in one atomic boundary | Aggregate root entities |
| Outbox Relay Publication | Background batch reader and aggregate-keyed stream publisher with publish-marker commit | A reliable deferred publication conveyor that preserves ordering and retries unpublished events | Event stream infrastructure |
| Predictive Demand Aggregation | Forecast set scan and product-level demand consolidation within safety horizon | A forward-looking demand interpretation engine that quantifies upcoming product pressure before threshold breach | Supplier node, retailer demand stream |
| Shadow Deficit Computation | Deficit evaluation against current stock, safety level, and buffered demand | A proactive shortage detection mechanism that triggers replenishment before operational depletion | Warehouse node, inventory state |
| Source Node Optimization | Lane-aware source selection for replenishment origination | A constrained source-selection method for choosing an optimal fulfillment origin under policy and topology constraints | Supplier node, factory node, warehouse node |
| Internal Transfer Drafting | Transfer and transfer-item creation for proactive replenishment | A preemptive replenishment artifact that formalizes future stock movement before shortage realization | Factory node, warehouse node |
| Replenishment Trace Linking | Linkage of replenishment identity back to contributing order set | A trace continuity mechanism that binds forecast signal, replenishment artifact, and downstream execution lineage | Supplier node, order entities |
| Forecast Refinement on Preorder Events | Demand-model refresh and correction intake from preorder lifecycle events | A closed-loop learning mechanism that adjusts forecast posture based on preorder confirmations, edits, and cancellations | Retailer signal stream, AI worker |
| Freeze-Aware AI Queue Suppression | Frozen-entity map enforcement in autonomous worker pipeline | A concurrency control gate that prevents machine reassignment while human override is active | Autonomous worker, locked entity scope |

## Batch 01 Completion Note

This catalog intentionally halts at Batch 01 boundaries. Subsequent batches should extend the same table contract for driver mobile, retailer mobile and desktop, payload surfaces, and remaining backend domains.
