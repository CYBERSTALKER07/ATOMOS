# Additional V.O.I.D. Ecosystem Solutions Roadmap

**Date:** 2026-08-19  
**Scope:** `pegasusX/` only  
**Purpose:** Additional business solutions that extend the existing supplier, warehouse, retailer, factory, driver, payload, finance, planning, and partner foundations.  
**Status rule:** This is a product roadmap, not evidence that the proposed capabilities are implemented.

## 1. Strategic Direction

V.O.I.D. should become a trusted operating loop rather than a collection of disconnected enterprise screens:

```text
supplier promise
→ warehouse allocation
→ factory replenishment
→ pick and quality verification 
→ payload seal
→ driver execution
→ retailer receiving
→ payment and fiscal
→ claims and settlement
```

The strongest differentiator is a complete, auditable loop across physical goods, money, evidence, and decisions.

## 2. Product Laws

Every new solution must preserve:

- `SupplierId` as the only tenant key;
- same-market orders, payments, credit, payouts, and fiscalization;
- MarketPack authority for currency, decimals, PSP, fiscal, payout, timezone, and operational limits;
- local-first warehouse and factory matching;
- supplier pins as overrides to proximity;
- fail-closed missing geography;
- integer minor-unit money paired with currency;
- transactional outbox for durable state changes;
- post-commit cache invalidation;
- authenticated, scoped WebSocket and push fanout;
- separate factory-transfer and supplier-last-mile manifest planes;
- factory planning placement and retailer auto-order placement off by default;
- honest unavailable, planned, unkeyed, and deferred states.

## 3. Supplier Service Promise Network

### Business solution

Suppliers publish reliable service commitments to retailers:

- same-day, next-day, or scheduled delivery;
- minimum order quantity;
- cutoff time;
- delivery zones;
- fill-rate guarantee;
- warehouse coverage;
- accepted payment methods;
- return and claim policy;
- assigned account manager;
- service health score.

### Existing foundations to extend

MarketPack, warehouse coverage, checkout, dispatch, retailer supplier attachment, CRM, pricing, credit, and claims.

### Business logic

1. Store supplier service policies under supplier scope.
2. Resolve the policy against the retailer's active location and supplier market.
3. Calculate the promise from actual inventory, warehouse capacity, driver capacity, factory availability, cutoff, and SLA.
4. Snapshot the accepted promise on the order.
5. Expose promise breaches as scored exceptions with owners and deadlines.

### Critical rules

- Never promise from an off-shift, frozen, or unavailable warehouse.
- Never recalculate a historical promise from current data.
- Do not expose internal supplier assumptions to retailers.

## 4. Retailer-Supplier Trading Relationship Center

### Business solution

Turn supplier attachment into a structured commercial relationship containing:

- agreed price lists;
- payment terms;
- credit limit;
- delivery SLA;
- claim window;
- return rules;
- preferred warehouse;
- promotion eligibility;
- assigned account manager;
- relationship health score.

### Existing foundations to extend

Retailer supplier attachment, invitations, CRM, credit relationships, pricing overrides, loyalty, promotions, claims, and payout policy.

### Business logic

1. Create a supplier-retailer relationship record under the supplier and retailer organization scopes.
2. Version commercial terms with effective dates.
3. Snapshot the applicable terms onto orders, claims, and credit documents.
4. Require explicit authorization for changes to credit, payment, or return terms.
5. Notify both parties after an approved change.

## 5. Supplier Sales and Replenishment Copilot

### Business solution

Combine demand, inventory, AI recommendations, segmentation, replenishment, and S&OP into a governed assistant that recommends:

- retailers likely to stock out;
- SKUs requiring replenishment;
- warehouses requiring inventory;
- retailers eligible for promotions;
- orders at risk;
- actions with the highest service improvement.

### Business logic

1. Read forecast, preorder, confirmed demand, actual sales, inventory, service level, and route capacity.
2. Keep forecast, recommendation, decision, and execution as separate records.
3. Explain the inputs and confidence behind every recommendation.
4. Allow acknowledge, override, dismiss, reopen, and approve actions.
5. Require freeze locks and dual control before high-consequence mutations.
6. Keep factory planning and auto-order placement behind their existing flags.

### Prohibited behavior

The assistant must not silently change price, move inventory, place orders, or override a human freeze lock.

## 6. Warehouse Quality and Recall Network

### Business solution

Extend lots, cold-chain readings, claims, quarantine, and reverse logistics into:

- lot traceability;
- supplier recall campaigns;
- expiry warnings;
- temperature excursion cases;
- affected-order lookup;
- quarantine movement;
- retailer notification;
- supplier liability calculation;
- destruction or return disposition.

### Business logic

1. Register a recall against SKU, lot, supplier, and market scope.
2. Find affected inventory, open orders, delivered orders, and retailer stock.
3. Freeze affected stock from allocation and picking.
4. Create scoped exceptions and retailer-facing notifications.
5. Link return, claim, credit note, refund, and supplier chargeback records.
6. Preserve evidence and immutable event history.

### Critical edge case

A recalled lot must remain blocked even when its quantity appears available.

## 7. Retailer Receiving Intelligence

### Business solution

Give retailers a receiving workflow that compares:

```text
ordered quantity
→ loaded quantity
→ delivered quantity
→ accepted quantity
→ damaged/missing quantity
→ putaway quantity
```

### Capabilities

- barcode scanning;
- photo evidence;
- signed receiving;
- automatic variance;
- quarantine;
- claim creation;
- supplier notification;
- payment adjustment.

### Business logic

1. Load expected lines from the authoritative order and manifest.
2. Import driver-reported exception quantities.
3. Prevent damaged or missing quantities from entering sellable stock.
4. Require evidence for high-risk damage and temperature claims.
5. Create one claim/liability record, not parallel chargeback requests.
6. Reconcile accepted quantity, stock movement, payment, and fiscal amount.

## 8. Supplier Procurement and Factory Capacity Network

### Business solution

Provide a private supplier-owned network where factories and warehouses exchange:

- replenishment requests;
- production capacity;
- available stock;
- lane cost;
- transfer SLA;
- minimum batch quantity;
- pickup windows;
- quality constraints.

### Existing foundations to extend

Factory supply requests, supply lanes, warehouse transfers, factory planning, S&OP, and factory-plane manifests.

### Business logic

1. Rank eligible factories by same-market rules, active lanes, SLA, cost, capacity, and carbon.
2. Create a planning recommendation or transfer draft.
3. Require factory or supplier approval before execution.
4. Write only to `FactoryTruckManifests` for factory transfers.
5. Project inbound commitments to warehouse ATP without merging manifest planes.

## 9. Carrier and 3PL Collaboration

### Business solution

Support external logistics providers without merging them into supplier-owned truck manifests:

- carrier onboarding;
- insurance and compliance;
- vehicle capabilities;
- tendering;
- acceptance/rejection;
- pickup appointments;
- proof of pickup;
- proof of delivery;
- detention and accessorial charges;
- carrier scorecards.

### Transport planes

Represent transport explicitly as:

```text
SUPPLIER_FLEET
FACTORY_FLEET
EXTERNAL_CARRIER
```

External execution can project into common read models, but factory and supplier manifest authorities remain separate.

### Critical edge cases

- Carrier accepts but lacks required temperature or vehicle capability.
- Tender expires and another carrier is selected.
- External proof conflicts with driver or payload proof.
- Partner retries an ASN with changed quantities.
- Carrier currency differs from order currency.

## 10. Cash and Credit Operations Center

### Business solution

Combine cash collection, cash reconciliation, credit, AR, claims, chargebacks, and payouts into a finance operations surface:

- driver cash-bag balance;
- expected versus declared cash;
- shortfall and overage;
- open fiscal orders;
- retailer credit exposure;
- overdue AR;
- supplier payout holds;
- reconciliation exceptions;
- collection actions;
- write-off approval.

### Business rules

- Manual payment correction requires actor, reason, authorization, and immutable ledger entries.
- No manual mark-paid action may bypass fiscal or provider confirmation.
- Supplier payout must pause when unresolved cash, payment, or claim exposure exceeds policy.
- All totals are grouped by currency.

## 11. Product Master and UOM Governance

### Business solution

Create one canonical product identity model covering:

- supplier SKU;
- retailer local SKU;
- barcode aliases;
- case/carton/pack/piece UOM;
- conversion factors;
- dimensions and weight;
- temperature class;
- lot requirement;
- expiry policy;
- substitution group;
- tax/fiscal classification;
- product lifecycle status.

### Business value

This reduces wrong picks, wrong prices, incorrect stock quantities, bad route-volume estimates, and catalog/entity-resolution drift.

### Business logic

1. Resolve product identity before catalog, inventory, checkout, or import mutation.
2. Require explicit UOM conversion for quantity changes.
3. Version dimensions and prices by effective date.
4. Snapshot the product and UOM used by historical orders.

## 12. Retailer Planogram and Shelf Availability

### Business solution

Extend retailer sections and planogram foundations into:

- expected shelf layout;
- SKU facings;
- shelf audit;
- out-of-stock detection;
- misplaced product detection;
- promotion compliance;
- supplier execution score;
- photo-based verification.

### Role boundaries

- Retailer owns shelf tasks and store evidence.
- Supplier sees availability and promotion execution for its products.
- Warehouse and factory receive aggregated demand signals only.
- Retailer-private shelf details never become a supplier-wide data dump.

## 13. Supplier Promotions with Execution Guarantees

### Business solution

Promotions should include the complete lifecycle:

- supplier-funded discount;
- retailer-specific offer;
- start/end dates;
- eligible locations;
- inventory reservation;
- minimum quantity;
- display compliance;
- performance;
- settlement responsibility;
- refund and claim treatment.

### Business logic

Promotion terms must be snapshotted on the order. Editing a promotion later must not change historical price, fee, refund, or settlement calculations.

## 14. Digital Evidence Vault

### Business solution

Create one evidence dossier for each high-consequence business object:

- order;
- delivery;
- proof of delivery;
- claim;
- return;
- cold-chain excursion;
- payment;
- fiscal receipt;
- chargeback;
- payout;
- transfer;
- staff approval.

### Business value

This improves disputes, support resolution, compliance audits, and supplier-retailer trust.

### Technical rules

- Evidence objects use signed storage URLs and scoped access.
- Every artifact records actor, timestamp, source device, hash, and aggregate ID.
- Evidence links are immutable after settlement.
- Offline uploads remain pending until server acknowledgement.

## 15. Partner API and Integration Hub

### Business solution

Turn the existing partner and EDI/AS2 foundation into an integration product:

- supplier-scoped API keys;
- order import;
- catalog export;
- ASN;
- invoice export;
- delivery status;
- inventory snapshot;
- webhook subscriptions;
- retry and replay console;
- schema version negotiation;
- partner health score.

Every integration should expose:

```text
received
→ validated
→ accepted
→ processed
→ failed
→ replayed
→ dead-lettered
```

## 16. Cost-to-Serve and Sustainability Analytics

### Business solution

Show suppliers the true cost of serving each retailer:

- delivery distance;
- truck utilization;
- empty miles;
- failed delivery attempts;
- shop-closed incidents;
- payment collection cost;
- claims and returns;
- warehouse handling;
- cold-chain cost;
- credit exposure;
- supplier margin.

Then allow controlled decisions:

- change delivery days;
- move warehouse coverage;
- adjust minimum order size;
- offer pickup;
- change service tier;
- prioritize high-value routes.

Carbon and cost factors must be versioned so historical reports remain reproducible.

## 17. Resilience and Trust Center

Provide suppliers and platform operators with operational proof:

- worker health;
- Kafka lag;
- outbox backlog;
- dead letters;
- fiscal failures;
- payment webhook failures;
- stale WebSocket consumers;
- Redis degradation;
- last successful backup;
- last restore drill;
- current client-version distribution;
- feature-flag status.

This extends platform admin, client policy, observability, outbox, and worker heartbeat foundations.

## 18. Product Packaging

### Supplier OS

Orders, inventory, dispatch, finance, credit, claims, planning, AI recommendations, CRM, service promises, cost-to-serve, and partner integrations.

### Warehouse OS

Receiving, bins and lots, pick waves, cycle counts, cold chain, labor, dispatch, returns, quality, treasury, and appointments.

### Retailer OS

Procurement, supplier relationships, store stock, POS, receiving, claims, credit, tracking, planograms, demand feedback, and promotions.

### Network OS

Supplier and carrier collaboration, private trading relationships, service promises, partner APIs, exception orchestration, and network analytics.

## 19. Recommended Sequence

1. Service Promise and ATP.
2. Retailer Receiving Intelligence.
3. Warehouse Quality, Lots, and Recall.
4. Cash and Credit Operations Center.
5. Product Master and UOM Governance.
6. Supplier S&OP and Factory Capacity.
7. Carrier and 3PL Collaboration.
8. Cost-to-Serve and Sustainability.
9. Partner Integration Hub.
10. Planogram and Shelf Intelligence.

The first three improve the physical and commercial truth of the existing order loop. Finance and product-master work should precede autonomous planning or auto-order placement.
