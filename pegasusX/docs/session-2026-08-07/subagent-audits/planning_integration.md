# pegasusX Evidence Pack (code SoT)

Audit of `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`. Claims below are from live code, not docs.

---

## A) Problem coverage vs O9 / Kinaxis / Blue Yonder / ERP+WMS / B2B marketplaces

### A1. Planning / demand / replenishment algorithms (present)

| Capability | What code does | Evidence |
|---|---|---|
| **Demand classification (SBC)** | ADI/CV² → smooth / erratic / intermittent (`ADI≥1.32`, `CV²≥0.49`) | `planning/forecast/classify.go:4-76` |
| **Forecast models** | Holt–Winters (m=7), Croston-SBA, SES; holdout param grid; residual bands; blocks short history (`<60` days / `<14` non-zero) | `planning/forecast/fit.go:32-88`, `holtwinters.go:7-90`, `croston.go:3-44` |
| **Demand sensing** | 14-day adjustments from signals × base velocity; promo multipliers; UZ calendar fallback | `demand/worker_sensing.go:40-120` |
| **Reorder suggestions (retailer)** | From `DemandAdjustments` + sell-through merge; SS v2 or `demand·0.15`; lead from policy/observed | `replenishment/reorder_suggestion_batch.go:14-81`, `suggestions_api.go:25-42` |
| **Warehouse replenishment engine** | 7-day burn, urgency CRITICAL/WARNING via lead multipliers (`1.3`/`2.0`), suggested qty = ceil(ROP − effective) | `replenishment/engine.go:25-186` |
| **Safety stock** | `SS = z·√(L·σd² + d̄²·σL²)`, `ROP = d̄·L + SS`; legacy `burn·lead·1.15`; gated by `SAFETY_STOCK_V2_ENABLED` | `replenishment/safety_stock.go:16-125` |
| **MEIO (network)** | Multi-WH scan; surplus→deficit transfers; capital cap; not a true multi-echelon optimizer | `replenishment/mei_engine.go:69-248` (transfer qty `min(donor/2, suggested)`) |
| **Echelon targets** | `target = burn·horizon·serviceFactor`; safety from SS v2 or buffer | `replenishment/echelon_targets.go:37-69` |
| **Touchless replenishment** | Policy-gated auto-approve → `FactoryInternalTransfers` + outbox | `replenishment/touchless.go:16-103`, `policies.go:15-37` |
| **S&OP snapshot** | Env-calibrated factory/WH throughput vs open supply-request demand; utilization alert | `planning/service.go:294-384` |
| **Scenarios** | Shock params (downtime, demand Δ%), compare deltas, publish CAS | `planning/scenarios.go:23-100` |
| **Governed agents** | Allowlist only: approve insight / open supply request / broadcast | `planning/agents.go:5-41`, `executor.go:38-76` |
| **Control tower playbooks** | Exception→playbook match; suggest or auto-execute safe actions | `controltower/engine.go:42-100` |

### A2. Routing / optimization

| Capability | What code does | Evidence |
|---|---|---|
| **OR-Tools VRP (Python)** | Real constraint solver path for contract payloads (capacity, time, cold-chain gates, etc.) | `services/optimizer-core/server/contract_solver.py:1-18`, `14-15` |
| **Rust VRP sidecar** | Nearest-neighbor + 2-opt; status **HEURISTIC** (not proven optimal) | `services/optimizer-core/server-rust/src/solver/vrp.rs:6-33`, `cpsat.rs:6-7,137-138` |
| **“CP_SAT” factory assign** | Name says CP-SAT; implementation is **greedy + swap**, returns Heuristic | `cpsat.rs:6-139`, contract enum `packages/optimizer-contract/jobs.go:33-35` |
| **Auto-dispatch** | Background worker commits optimizer routes for opted-in warehouses | `warehouse/auto_dispatch.go:27-99` |

### A3. WMS features (present, mid-depth)

Mounted under warehouse ops via `stocklots`:

- Bins, lots, putaway, pick waves (+ confirm/waive shorts), cycle counts (+ ABC enqueue), adjustments approve/reject, inventory accuracy/reconcile, temperature readings  
  → `warehouseroutes/routes.go:57-84`
- FEFO/FIFO lot reservation  
  → `stocklots/fefo.go:32-49`
- Transfer receive → putaway into RECV/STAGE  
  → `warehouse/receive_items.go:51-121`
- Dispatch preview/execute, rescue, fleet live map, supply requests, cold-chain adjacent surfaces  
  → `warehouseroutes/routes.go:109-160`

**Not found in code:** slotting optimization, yard/dock appointment system, labor standards engine, full wave planning like enterprise WMS, ASN EDI as first-class WMS ASN module (EDI DESADV exists at partner layer instead).

### A4. Factory features (execution, not MES/MRP)

- CRUD factories, transfers create/transition, supply-request accept/fulfill options, loading bay (start-loading/seal), manifest dispatch/complete/rebalance, fleet/staff  
  → `factoryroutes/routes.go:34-74`
- Optimizer “factory slot” assignment is heuristic capacity packing, not production scheduling  
  → `cpsat.rs:6-58`

**Not found:** BOM, MRP, work orders, shop-floor MES, finite capacity production scheduling, yield/scrap accounting as manufacturing core.

### A5. Marketplace / catalog / ordering (B2B)

- Public catalog categories/products; supplier admin create/update  
  → `catalogroutes/routes.go:26-41`
- Retailer cart sync, checkout preview/unified/B2B, cash/card  
  → `retailerroutes/routes.go:167-170`, `paymentroutes/routes.go:26-29`
- Multi-supplier checkout paths exist in order package (files present: `order/multi_supplier_checkout.go`, `unified_checkout.go`)
- Partner machine catalog/price/stock upsert + orders  
  → `partner/routes.go:25-32`
- Retailer POS + store stock + auto-order (see C)  
  → `retailerroutes/routes.go:76-149`

This is a **vertical B2B procurement + ops network**, not a general marketplace (no public seller onboarding marketplace, bidding, or broad discovery engine comparable to Amazon Business).

### A6. Gaps vs enterprise APS (O9 / Kinaxis / Blue Yonder) and ERP+WMS

Honest shortfalls from code:

1. **APS depth:** statistical forecasts + heuristics + scenario shocks — no concurrent planning graph, constraint-based S&OP solver, or Kinaxis-style “what-if on shared model” engine. S&OP capacities are **env defaults × counts**, not calibrated calendars (`planning/service.go:341-349`).
2. **MEIO:** greedy donor/receiver with capital cap, not multi-echelon inventory optimization / network LP (`mei_engine.go:215`).
3. **Solver truthfulness:** Rust “CP_SAT” is not CP-SAT; OR-Tools VRP exists but dual/heuristic paths remain (`cpsat.rs:137-138`, `contract_solver.py`).
4. **Manufacturing:** no MRP/BOM/MES — factory is logistics/fulfillment node.
5. **WMS:** solid lot/bin/pick/count for this domain; missing enterprise WMS breadth (slotting, yard, labor eng, 3PL billing complexity).
6. **ERP:** AR/journals/COA/export exist; not GL/AP/FA/procurement ERP suite.
7. **Demand sensing Phase-1 shortcuts:** REGION/CITY treated as all retailers (`demand/worker_sensing.go:97-99`).

### A7. Vertical depth advantages that are real in code

1. **Factory → warehouse → retailer → POS sell-through loop** with shared Spanner entities (transfers, supply requests, replenishment insights, demand adjustments, reorder suggestions, auto-order).
2. **Multi-role native surfaces:** `apps/` includes retailer/supplier/warehouse/factory/payload/driver portals + iOS/Android apps (verified directory listing).
3. **Physical execution roles first-class:** loading bay seal, pick waves, FEFO, GS1 labels, driver/payload terminals — not bolted analytics-only.
4. **Autonomy with gates:** touchless factory replenishment + retailer auto-order soak gate (fail-closed).

---

## B) Integration with retailer/supplier systems (1C, SAP, WMS, POS, accounting)

### B1. What exists (M2M-oriented)

| Layer | Capability | Evidence |
|---|---|---|
| **Partner API** | OAuth client_credentials + API keys; scoped routes for orders/catalog/inventory/demand/webhooks/exports/EDI/COA/AS2 | `partner/routes.go:10-50`, scopes `partner/types.go:17-25` |
| **Webhooks** | Subscribe/ping/rotate/dead-letter/replay; allowlisted event subset | `partner/routes.go:33-39`, `webhook_events.go:8-23` |
| **EDI-lite** | ORDERS/ORDRSP/DESADV/INVOIC + CONTRL/APERAK + PRICAT/INVRPT/SLSRPT/RECADV/ORDCHG/DELFOR/REMADV — **explicitly not certified EDIFACT** | `partner/types.go:59-72`, `partner/edi/breadth.go:1-2`, `edi/orders.go:27-28` |
| **AS2** | RFC 4130 receive endpoint + config | `partner/routes.go:20-21`, `as2_receive.go` (package), `as2/` |
| **SFTP** | Partner SFTP config + export upload status enums | `partner/types.go:39-41`, admin routes `partner/routes.go:83-87` |
| **Exports** | orders/invoices/inventory/ledger/**journals**; CSV/JSON/XML | `partner/types.go:43-51` |
| **1C-style accounting** | Default COA `62.01` / `90.01` / `51.01`; journals XML `dialect="1c"` | `partner/coa.go:12-17`, `export_worker.go:319-322` |
| **Journal mapping** | AR OPEN/PAYMENT/CN + payment refunds → debit/credit accounts | `partner/export_journals.go:34-61` |
| **POS demand feed** | Partner `POST /demand/pos-feed` → `DEMAND_SIGNAL` | `partner/pos_demand_feed.go:28-98`, route `:32` |
| **GS1** | GTIN/GLN/SSCC Mod-10, SSCC mint, GS1-128 ZPL, ECC200 DataMatrix | `gs1/checkdigit.go:1-54`, `zpl.go`, `ecc200.go`, `datamatrix.go` |
| **SDK** | Generated Go partner client under `sdk/partner/go/` | glob confirmed |
| **Native POS** | In-platform retailer POS sessions/sales/void/refund/holds | `retailerroutes/routes.go:89-104` |

### B2. Machine-to-machine readiness (assessment)

**Ready for:** partners willing to speak Pegasus partner API / EDI-lite / AS2 / SFTP / webhook JSON; 1C-ish journal import after mapping; POS sell-through push; GS1 shipping labels.

**Not ready as drop-in for:**

| System | Gap in code |
|---|---|
| **SAP** | No IDoc/OData/RFC/BAPI connector packages; no SAP-specific mapping layer (grep shows no SAP integration modules) |
| **Certified EDI** | Code comments: “Still not certified EDIFACT” (`edi/breadth.go:1-2`) |
| **External WMS** | Partner inventory upsert/availability, not Manhattan/SAP EWM adapters |
| **External POS chains** | Feed API exists; no vendor-specific connectors (Toast/Square/1C Retail, etc.) |
| **Full GL sync** | Journals export only (AR/payments/CN), not bidirectional ERP master data sync |

### B3. What must exist for big players without re-keying

Minimum adoption kit implied by gaps:

1. **Certified/document-mapped EDI or SAP IDoc adapters** for ORDERS↔ORDRSP↔DESADV↔INVOIC (or equivalent API mapping), not only EDI-lite dialect.
2. **Master-data sync** (customers, GLN, GTIN, price lists, plants) with idempotent upsert + conflict rules (partially present for products/prices/stock).
3. **Inventory & ASN bidirectional** with their WMS (Pegasus already has internal lot/ASN-like DESADV packing tests).
4. **Accounting**: either certified 1C/SAP FI journal import profiles or REST posting to their GL — today: XML journals + configurable COA.
5. **POS/ERP sell-through** continuous feed (API exists) + reconciliation/void semantics at scale.
6. **Auth/ops**: OAuth+keys, sandbox, webhook DLQ already present — good baseline for enterprise IT.

---

## C) Unified platform existence (code-based position)

### Ideal vs code

| Ideal | Code reality |
|---|---|
| Quality suppliers + retailers on one network | Catalog + trading + multi-role apps exist |
| Near-zero human interaction for routine replenishment | **Partial:** supplier touchless transfers (`touchless.go`); retailer auto-order `shadow/draft/place` with soak gate fail-closed (`auto_order_worker.go:263-351`, `auto_order_soak_gate.go:48-83`) |
| Still support physical execution | Strong: factory loading bay, WMS pick/putaway/counts, warehouse dispatch, payload/driver apps |
| Closed-loop demand | POS → demand signal → adjustments → reorder suggestions → auto-order/cart/place |

### Autonomy maturity

- **Supplier side:** auto-approve replenishment insights → factory internal transfers (policy + confidence).
- **Retailer side:** modes `off|shadow|draft|place`; place requires soak stats (min proposals, WAPE, unmodified rate) unless break-glass flag.
- **Warehouse:** auto-dispatch worker when enabled.
- **Control tower:** playbooks can auto-execute only if actions marked auto-safe + flags.

This is **governed autonomy**, not fully hands-off APS.

### Honest compare to public systems

| Class | Typical public systems | pegasusX (from code) |
|---|---|---|
| **APS (O9/Kinaxis/BY)** | Deep concurrent planning, ML demand, MEIO solvers, enterprise S&OP | Lightweight forecasting + heuristics + scenario shocks + dashboards |
| **WMS (Manhattan/BY WMS)** | Full warehouse engineering | Domain WMS core (lots/bins/waves/FEFO/counts) + last-mile dispatch |
| **ERP (SAP/1C)** | Full financials/procurement/manufacturing | Vertical ops + AR/journals export; 1C-flavored COA; not ERP replacement |
| **B2B marketplace** | Broad discovery/marketplace economics | Supplier–retailer procurement network with native apps |
| **Unified vertical OS** | Rare; usually stitch ERP+WMS+TMS+OMS | **Real differentiator in code:** one transactional fabric from factory transfer → WH → route → retailer stock/POS → reorder |

**Verdict:** Unified platform **exists as an operational vertical stack** (factory↔warehouse↔delivery↔retailer↔POS↔replenishment↔partner I/O). It does **not** yet match enterprise APS/WMS/ERP suites on planning math, manufacturing, or certified ERP connectors. Competitive claim that holds in code: **end-to-end transactional + multi-role execution with gated autonomy**, not “Kinaxis-class planning.”

---

## Quick capability map (search hits → packages)

| Search theme | Primary packages |
|---|---|
| `demand*` / `forecast*` | `apps/backend-go/demand`, `planning/forecast` |
| `replenish*` / `suggest*` / `auto_order*` | `replenishment`, `retailer/auto_order_*` |
| `routing*` / `optimize*` | `services/optimizer-core`, `warehouse/auto_dispatch`, `dispatch/optimizerclient` |
| `plan*` | `planning` (S&OP, scenarios, agents, accuracy) |
| `wms*` / `warehouse*` | `warehouse`, `stocklots`, `warehouseroutes` |
| `factory*` | `factory`, `factoryroutes` |
| `integration*` / `webhook*` / `edi*` / `connector*` / `partner*` / `export*` | `partner`, `sdk/partner` |
| `1c*` | `partner/coa.go`, `export_worker.go` (dialect), journals |
| `sap*` | **no dedicated connector** |
| `gs1*` | `apps/backend-go/gs1` |
| `autonomy*` | touchless + auto-order soak + auto-dispatch + controltower |

---

### Bottom line

Code supports a **real factory-to-shelf ops network** with statistical replenishment, heuristic/OR-Tools routing, mid-depth WMS, partner M2M (API/EDI-lite/AS2/SFTP/webhooks/1C journals/GS1), and gated auto-ordering. It is **not** an enterprise APS or SAP replacement; big-retailer adoption without re-keying still needs certified ERP/EDI adapters and deeper master-data/WMS sync beyond the partner layer’s current dialect.