# 3. Problem Coverage vs Existing Logistics / Planning Software

## 3.1 Capability-by-capability comparison

Capabilities are marked from PegasusX's **code-verified** state today (W = wired-live, F = flag-gated real code, P = partial/heuristic, D = decorative, – = absent). Enterprise columns reflect the public 2026 state of each category.

| Capability | O9 / Kinaxis | Blue Yonder | ERP+WMS stack (SAP/1C+best-of-breed) | Pure B2B marketplace | **PegasusX today** |
|---|---|---|---|---|---|
| Demand forecasting (ML-grade, multi-signal) | ● Leader | ● Leader | ◐ module | ◐ basic | **F** — classical stats only (HW/Croston/SES + classification + WAPE accuracy); no ML, no POS feed |
| S&OP / IBP (scenario planning, financial reconciliation) | ● | ◐ | ◐ | – | **D** — `GetSAndOP` returns `factories × 700 × 7` (`planning/service.go:252`); `projectStockouts` returns literal strings `sku-projection-1/2` |
| Multi-echelon inventory optimization | ● | ● | ◐ | – | **P** — `RunMEIONetwork` is a two-node greedy donor/receiver swap per SKU (`replenishment/mei_engine.go:168`); echelon targets heuristic |
| Safety stock / service-level inventory | ● | ● | ● | – | **F** — correct `SS = z·√(Lσ_d² + d̄²σ_L²)` with residual-σ loop, flag-gated; legacy heuristic default |
| Warehouse management (lots/FEFO, waves, counts, cold chain) | – (integrates) | ● native WMS | ● | – | **F** — real backend (`stocklots/`: FEFO, pick waves + seal gate, cycle counts ABC, cold-chain quarantine) but **portal-only execution; mobile floor apps can't run it** |
| Transportation / routing optimization | – (integrates) | ● native TMS | ◐ | ◐ 3PL | **P** — OR-Tools VRP with time windows/cold-chain/hazmat constraints built but **0 replicas in prod**; live = H3+bin-pack+2-opt; Haversine ETAs |
| Order capture / B2B storefront | – | ◐ OMS | ◐ | ● | **W** — native apps per role, quoted checkout, idempotency, offline queues |
| Last-mile execution (driver app, POD, geofence, cash) | – | ◐ | – | ◐ | **W** — the strongest layer: telemetry, POD photo/signature, COD reconciliation, rescue |
| Credit / collections / embedded finance | – | – | ◐ AR module | ● (survivors monetize it) | **F** — terms/AR/aging/dunning/auto-hold implemented, flags off; scoring removed; no off-app reach |
| Fiscal compliance (UZ OFD/Soliq) | – | – | ● (1C/local) | – | **P** — framework + hard gate real; legal provider non-functional as shipped; default = non-tax commercial receipt |
| Control tower / real-time visibility | ◐ via partners | ● | ◐ | – | **D/W split** — event-driven twin projection is real (`twin/consumer.go`), but the "live network" dashboard broadcasts **random mock data** (`simulator/control_tower.go:53-79`) |
| Integration (EDI/AS2/API/1C) | ● certified | ● certified | ● | ◐ | **F/P** — EDI-lite + AS2 (uncertified, shipped off) + partner API + 1C journal files; no certified EDIFACT/CommerceML |

## 3.2 What PegasusX actually solves today that the others do not

- **One transactional model from factory floor to shop shelf.** Factory manifests → inter-hub transfers → warehouse loading bay with a dedicated payloader role → volumetric truck packing (VU) → geofenced driver handshake → COD/split/credit delivery with ledger and cash reconciliation → claims/quarantine/reverse → fiscal receipt. O9 and Kinaxis explicitly have no native execution (they integrate to WMS/TMS); Blue Yonder has execution but nothing like the COD field layer, the payloader role, or CIS fiscal plumbing; marketplaces bolt onto 3PLs and never run a warehouse floor.
- **A native app per physical role**, all wired to the same backend: retailer (3 variants), supplier (3), driver (2), payloader (2 + terminal), factory (3), warehouse (3). No competitor in any adjacent category ships this role-complete a client set.
- **COD + credit + split-payment delivery as first-class transaction states** (`PENDING_CASH_COLLECTION`, `DELIVERED_ON_CREDIT`, `FISCALIZING`), with cash reconciliation gating driver shift close (`cashrecon/service.go:152-161`). This is the operational reality of the target market that O9/Kinaxis/BY do not model.

## 3.3 Where it falls short — honestly

- **Planning depth is not in the same league as O9/Kinaxis/BY.** No scenario planning, no concurrent replanning, no financial reconciliation of plans, no ML demand sensing. The S&OP surface is a decorative stub (`planning/service.go:212,252`). A CPG manufacturer would not buy this for planning.
- **Optimization is built-but-not-run.** The OR-Tools solver with constraint fidelity, multi-depot, and OSRM matrices exists in code and sits at `replicas: 0` in every shipped overlay (`infra/k8s/overlays/prod/kustomization.yaml:44-50`). Production routing is a competent heuristic, not optimization.
- **Warehouse execution caps the addressable market despite real backend progress.** Lots/FEFO/pick-waves/cycle-counts/cold-chain are flag-gated and **portal-only**; the floor worker's Android scanner is a dead stub that reports success without any API call (`ScannerViewModel.kt:22,47`, orphaned from navigation). Food/pharma-grade WMS is close in the backend and absent in the aisle.
- **It cannot yet be the system of record for anyone else's money.** Broken card capture, non-tax default receipts, and DB-unenforced payment idempotency (§7) disqualify it as a financial SoT until fixed.

## 3.4 The vertical-depth advantage, stated precisely

The defensible asset is not "a platform connecting suppliers and retailers" (that category exists and is crowded). It is the **single-stack vertical depth with one event bus**: 155 event types flowing from one transactional core to every role app and to the partner surface. For a large distributor who wants their factory, warehouse, fleet, field, and retail customers in one model — that is genuinely rare and is worth more as **deep software for one large distributor** than as a thin marketplace for many. Every adjacent category leader either plans without executing (O9/Kinaxis), executes without owning the B2B field layer (Blue Yonder), or sells without operating (marketplaces).
