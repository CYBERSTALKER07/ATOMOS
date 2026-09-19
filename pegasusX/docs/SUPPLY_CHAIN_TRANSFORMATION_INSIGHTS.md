# PegasusX — Exhaustive Supply Chain Transformation Insights

This document captures **all** problems, solutions, technical features, and business/legalization constraints extracted from FMCG leaders (Coty, Nestlé, Philip Morris, Acuity Brands) and explicitly maps how every single one must work within the PegasusX ecosystem.

---

## 1. Data, Architecture, and Systems Integration (Tech & Legal)

### 1.1 Fragmentation & System Disconnect (Coty)
* **Problem:** Supply chains rely on 10-20 fragmented systems (ERP, APO, Excel) leading to severe data loss at interfaces. The commercial plan is completely disconnected from the supply chain execution.
* **Solution:** Implement a single, end-to-end platform. Skip the "crawl-walk-run" trap that fails on massive initial data loads; instead, build a robust Data Lake architecture from day one.
* **How it works in PegasusX:** 
  * Pegasus eliminates interfaces by running all roles (Supplier, Warehouse, Retailer, Factory, Driver) on a single Spanner backend. 
  * Data loss is structurally prevented via transactional Outbox -> Kafka streams. 
  * The commercial side (Retailer checkout) and execution side (Warehouse WMS/Manifests) share the exact same `Orders` and `ParentOrders` tables, ensuring zero drift.

### 1.2 Manual Data Manipulation (Acuity Brands)
* **Problem:** Planners spend up to a month manually gathering and manipulating data, leaving no time to evaluate the data or make decisions.
* **Solution:** Automate data preparation so the baseline plan is generated instantly, shifting the planner’s role from data manipulation to evaluation.
* **How it works in PegasusX:** 
  * Real-time rollups (`GS-UF` freshness, ETag caching) automatically aggregate `orders_by_status`, `transfers_by_state`, and `manifests_by_state`.
  * Users view auto-generated `StatusStack` UI components. Planners evaluate these live metrics instead of manually aggregating daily CSV exports.

### 1.3 Customization vs. Standardization / "Clutter" (Nestlé/PMI)
* **Problem:** Companies try to customize new software to fit bad legacy processes, transferring operational "clutter" to the new system.
* **Solution:** Clean the house first. If a legacy process or channel doesn’t add value, drop it. Standardize processes globally before digitizing them.
* **How it works in PegasusX:** 
  * Pegasus enforces immutable product laws (`L1-L10`). 
  * Legacy workarounds are actively rejected. For example, a mixed-market cart is rejected (`cross_market_deferred`) rather than building complex cross-border schema logic. The business must adapt to the clean ecosystem model.

---

## 2. Planning, Forecasting, and Analytics (Tech & Biz)

### 2.1 Forecast Obsession vs. Gap Management (Nestlé/PMI)
* **Problem:** Organizations obsess over getting the forecast 100% accurate (which is impossible) rather than focusing on actionable discrepancy management.
* **Solution:** Shift to "Gap Management." Focus on identifying the gap between the current plan and the business target, and presenting options to close it.
* **How it works in PegasusX:** 
  * The `Plan & Brain` tabs (`GS-U3`) do not just plot historical accuracy; they highlight exceptions, `blocked_reason`s, and out-of-stock gaps.
  * Dashboards shift focus to KPI telemetry like *Mix Accuracy*, *Plan Bias*, and *Error Percentage* (tracked via `planning/accuracy_handlers.go`).

### 2.2 Scenario Planning & Digital Twin (Coty / PMI)
* **Problem:** Systems lack predictive and prescriptive tools, leaving teams unable to see the financial impact of supply chain tradeoffs.
* **Solution:** Create a "Digital Twin" that provides instant financial valuation (Net Revenue) instead of just volume (units). Implement tradeoff modeling.
* **How it works in PegasusX:** 
  * The `Supplier Dashboard` (`GS-U2`) and `Retailer Control Tower` (`GS-U6`) act as the digital twin, mapping physical truck/manifest states directly to `packCurrency` instantly. 
  * Planners can see the exact revenue tied to `FISCAL_FAILED` or `IN_TRANSIT` states in real time.

### 2.3 Exception-Based Automation (Nestlé/PMI)
* **Problem:** Decision paralysis caused by information overload.
* **Solution:** Automate baseline decisions and use exception-based alerting to highlight only what needs human attention.
* **How it works in PegasusX:** 
  * UI surfaces focus on exceptions via `StatusStack` chips (e.g., filtering for `ARRIVED_SHOP_CLOSED` or `FISCAL_FAILED`).
  * The Dead-letter KPI in the Platform Admin dashboard (`GS-U8`) strictly tracks exceptions (`COUNT(*) FROM OutboxDeadLetters`), hiding the metric entirely if there are no errors.

---

## 3. Organizational Alignment & Target Operating Model (Legal & Biz)

### 3.1 Misaligned KPIs & Silos (Nestlé/PMI)
* **Problem:** Functional silos have conflicting KPIs (e.g., Supply Chain optimizes OTIF and Days Cover; Finance optimizes Revenue). This creates a scenario of "three drivers in a car hitting gas, brake, and steering simultaneously."
* **Solution:** Implement Integrated Business Planning (IBP). Establish consensus planning and speak the language of the business (Net Revenue) across all teams.
* **How it works in PegasusX:** 
  * Every dashboard (Supplier, Warehouse, Retailer) surfaces metrics unified by `packCurrency`. 
  * The strict `MarketPack` law ensures everyone in the cell speaks the same financial language (integer minor money, single decimal format), bridging the gap between operations (trucks) and finance (cash).

### 3.2 Target Operating Model (TOM) Enforced (Coty)
* **Problem:** Without a defined operating model, transformation fails to take root.
* **Solution:** Define the TOM end-to-end (down to level 2.2 processes). Use the digital transformation as a forcing function to align all business units to this model.
* **How it works in PegasusX:** 
  * The Pegasus architecture *is* the TOM. The local-first, multi-supplier rules (e.g., closest warehouse `ResolveServingWarehouse`, pack-owned PSPs) are hardcoded into the platform. 
  * Legal boundaries (data residency per Cell, fiscal hard-gates) force the organization into compliance by design.

### 3.3 Time Fences & GIMO (Nestlé/PMI)
* **Problem:** Fear of making mistakes leads to taking too long to act.
* **Solution:** Establish clear decision cadences and SLAs (time fences). Adopt the GIMO (Good Enough, Move On) mentality for agility.
* **How it works in PegasusX:** 
  * SLAs are embedded in the platform, such as the Factory SLA board (request due) and driver `ARRIVED` grace minutes (dictated by the `MarketPack`). 
  * Timers and states force resolution rather than allowing endless pending states.

---

## 4. Value Realization & Governance (Legal & Biz)

### 4.1 Value Creation vs. Realization / Corporate Amnesia (Acuity Brands)
* **Problem:** Value created by a new tool (e.g., better forecasts) is not automatically realized in the P&L. Organizations suffer from "corporate amnesia" and forget the required ROI.
* **Solution:** Trace the investment thesis directly to end-of-period financial statements. Set up governance frameworks to hold the business accountable.
* **How it works in PegasusX:** 
  * Value tracking is built into the Platform Admin layer (`GS-U8`). 
  * System usage (throughput, volume, dead letters) is directly trackable against supplier billing and platform fees. Financial realization (Credit/AR) is processed in the same transactional boundary as the physical order execution.

### 4.2 Associate Productivity (Acuity Brands)
* **Problem:** Scaling a business traditionally requires adding proportional headcount (linear scaling).
* **Solution:** Focus on "Associate Productivity"—enabling the same team to handle a larger, more complex business portfolio without adding headcount.
* **How it works in PegasusX:** 
  * Feature parity across desktop, Android, and iOS (GS-R) ensures field workers (Drivers, Payloaders) and desk workers (Admins) process operations instantly.
  * Eliminating manual reconciliation (e.g., dual manifests, auto-generated receipts) allows the existing workforce to handle 10x the order volume.

### 4.3 Rigid Data Governance (Coty)
* **Problem:** Bad data quality ruins the trust in the system.
* **Solution:** Treat data governance with the same rigidity as software builds. Do not compromise on value drivers.
* **How it works in PegasusX:** 
  * Strict schema typing and validation. Missing a `CountryCode` results in a fail-closed error (`geography_incomplete`), rather than silently failing or dumping bad data into the lake.
  * Market parameters (Currency, PSPs, Fiscal rules) are strictly governed by the `MarketPack` registry, entirely outside the control of individual users or warehouses.
