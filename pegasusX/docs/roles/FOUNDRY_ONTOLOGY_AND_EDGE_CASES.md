# PegasusX Foundry Architecture: Semantic Ontology, Kinetic Actions, Scenario Twin & 12 Edge-Case Guards

> **Specification Standard**: Palantir Foundry-Grade Semantic-Kinetic-Dynamic Operating Layer for PegasusX  
> **Repository Target**: `/Users/shakhzod/Desktop/pegasusx`  
> **Authoritative Backend Seams**: `apps/backend-go/foundry/`, `apps/backend-go/foundryroutes/`, `apps/backend-go/schema/spanner.ddl`  

---

## 1. The Three-Layer Foundry Triad on Cloud Spanner + Kafka + Redis

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                         SURFACE LAYER (9 Role Apps + Control Tower UI)                          │
│         Object Views • Graph "Search Around" • Scenario Sandbox • Action Forms • Monitors       │
└────────────────────────────────────────────────┬────────────────────────────────────────────────┘
                                                 │
┌────────────────────────────────────────────────▼────────────────────────────────────────────────┐
│                      DYNAMIC LAYER: SCENARIO & DIGITAL TWIN ENGINE                              │
│   Copy-on-Write State Overlay:  ResolvedView(Object) = LiveSpannerState ⊕ ScenarioDeltaLog      │
│   Optimistic Concurrency Commit: Validates BaseVersion Watermarks Before Production Promotion   │
└────────────────────────────────────────────────┬────────────────────────────────────────────────┘
                                                 │
┌────────────────────────────────────────────────▼────────────────────────────────────────────────┐
│                     KINETIC LAYER: ACTION ENGINE & PRE-EXECUTION RUBRICS                        │
│   Action Types • Pre-Execution Edge Guards • Multi-Signer Approvals • Idempotent Diffs          │
│   Side-Effect Outbox (Soliq EHF, SMS, CIT Manifests) • Immutable KineticActionExecutions        │
└────────────────────────────────────────────────┬────────────────────────────────────────────────┘
                                                 │
┌────────────────────────────────────────────────▼────────────────────────────────────────────────┐
│                     SEMANTIC LAYER: THE ONTOLOGY & OBJECT SET SERVICE                           │
│   OntologyObjects • OntologyGraphEdges (Bidirectional Graph) • Propagating SecurityMarkings     │
│   Millisecond Multi-Hop "SearchAround" Traversal (RawMaterialLot -> Pallet -> Order -> Factura) │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.1. Semantic Layer (`apps/backend-go/foundry/ontology.go`)
- **First-Class Ontology Objects (`OntologyObjects`)**: Every domain noun (`RawMaterialLot`, `WorkOrder`, `FinishedLot`, `SSCCPallet`, `PickWave`, `Tote`, `Vehicle`, `RouteManifest`, `Order`, `CashBag`, `InvoiceSettlementSlice`, `SoliqFactura`, `MerchantLegalEntity`) is addressable via a canonical `(CellId, ObjectType, ObjectId)` tuple with monotonic `Version` and `SecurityMarkings`.
- **Bidirectional Graph Traversal (`OntologyGraphEdges` & `SearchAround`)**: Enables $N$-hop breadth-first traversal across object sets in milliseconds (e.g., 1-click Factory-to-Retailer contaminated lot recall blast radius).
- **Propagating Security Markings**: When an object is stamped with a mandatory marking (`MARKING:AML_INVESTIGATION`, `MARKING:DISCREPANT_QUARANTINE`, `MARKING:THERMAL_QUARANTINE_SPOILED`, `MARKING:MICRO_SKIMMING_SUSPECT`), linking it to downstream derived objects automatically propagates the marking across the lineage graph.

### 1.2. Kinetic Layer (`apps/backend-go/foundry/kinetic.go`)
- **Pre-Execution Guard Evaluation**: Every state mutation runs through `ExecuteKineticAction`, which evaluates registered `PreconditionEvaluator` functions before committing mutations.
- **Immutable Provenance**: Every action execution records actor identity, role, scenario context, pre-condition rubric report, before/after state diff JSON, and idempotency key in `KineticActionExecutions`.

### 1.3. Dynamic Layer (`apps/backend-go/foundry/scenario_overlay.go`)
- **Copy-on-Write Overlay**: Mutations inside a `ScenarioId` write exclusively to `ScenarioObjectPatches` (`UPSERT` or `DELETE` patches anchored to `BaseVersion`).
- **Overlay Resolution**: Solvers read `ResolveObject(ctx, scenarioId, objectType, objectId)` which computes `BaseState ⊕ ScenarioPatches - ScenarioDeletions`.
- **Watermark-Protected Commit**: Committing a Scenario verifies that every touched object's current production `Version` equals the `BaseVersion` captured when the patch was created. Any drift aborts the commit (`409 Conflict`) and returns structured `ScenarioConflict` diffs for 1-click re-simulation.

---

## 2. The 12 Mandatory Cross-Role Pre-Execution Edge-Case Guards

| Guard # | Edge Case Name | Roles Involved | Pre-Execution Invariant & Deterministic Resolution |
| :---: | :--- | :--- | :--- |
| **1** | **Split-Tender Doorstep Partial Rejection** | Role 1, Role 5, Role 6 | If rejection reason is `DAMAGED_IN_TRANSIT` or `SHORT_SHIPPED`, preserve original volume-break discount tier (`QualificationQty = OrderedQty`, `BillableQty = AcceptedQty`). If `BUYER_REMORSE_REFUSAL`, recalculate tier at `AcceptedQty`. Apply reduction to `CREDIT` draw first before `CASH`. |
| **2** | **Multi-Tender Tracking & Uncapped Policy** | Role 1, Role 5, Role 6, Role 7 | Aggregate daily cash payments across parent `LegalEntityTIN` + `SupplierId` in `MerchantDailyCashExposure` for audit reporting. Under the uncapped policy (`UNLIMITED_PAYMENTS=1` / `AML_LIMITS_DISABLED=1`), checkout and collections proceed without payment ceilings or credit caps. |
| **3** | **Two-Phase Soliq Fiscal Pre-Flight & Payout Gate** | Role 1, Role 3, Role 8 | Block Pick-Wave release (Step 5) if any order SKU lacks a valid IKPU/MXIK code or if Supplier PKCS#12 `.p12` cert expires in `< 72h`. Block `InvoiceSettlementSlices` from `BATCHED` (Step 10) unless `FiscalState == REGISTERED_SOLIQ`. |
| **4** | **Mid-Route Breakdown Rescue & Thermal Quarantine** | Role 3, Role 5, Role 7, Role 9 | Freeze Driver 1 cash liability at `Stops 1..K`. Quarantine chilled/frozen totes (`MARKING:THERMAL_QUARANTINE_SPOILED`) if thermal excursion exceeds tolerance. Bind new bolt seal and assign `Stops K+1..N` liability to Rescue Driver 2. |
| **5** | **Zero-Destructive RRN Tender Reclassification** | Role 5, Role 7, Role 8 | Never overwrite `OrderPaymentLegs`. Require a verified gateway RRN within `±15m` of the ePOD geofence timestamp and matching Tiyin amount to append compensating `CASH_REVERSAL` + `CARD_VERIFIED_RRN` entries. |
| **6** | **3D Volumetric Crushability & Returnable Containers (`RTI`)** | Role 3, Role 4, Role 5, Role 6 | Block loading heavy items (`GrossWeightG`) on top of lower `MaxStackCrushKg` fragile items; account for reverse-logistics volume of picked-up empty crates/kegs in `ReturnableContainerLedger`. |
| **7** | **Scenario Commit Stale-State Conflict** | Role 1, Role 3, Role 8, Role 9 | Compare `ScenarioObjectPatches.BaseVersion` against live `OntologyObjects.Version`. Reject stale commits (`409`) with structured per-object conflict diffs. |
| **8** | **Post-Settlement Clawback Slice & Rolling Reserve** | Role 1, Role 6, Role 8 | Approved concealed damage claims on `BATCHED` slices append a negative `InvoiceSettlementSlice` (`CONCEALED_DAMAGE_CLAIM_APPROVED`) backed by the Supplier's rolling reserve escrow. |
| **9** | **FX Volatility Corridor & Wave Price Freeze** | Role 1, Role 3, Role 6 | Enforce `LOCK_AT_CHECKOUT` vs `FLOAT_WITH_CORRIDOR_CAP` before wave release. Once a Pick Wave is released (Step 5), UZS Tiyin prices are hard-frozen. |
| **10** | **Abandoned Pick-Wave Tote Heartbeat Takeover** | Role 3 | Monitor `PICKER_IN_HAND` totes; if scan heartbeat stalls `> 20m`, allow atomic supervisory tote takeover without losing already-picked FEFO lots. |
| **11** | **Dual-Blind Cashier Recount Ratchet & Anti-Skimming** | Role 5, Role 7 | Mask `ExpectedCashMinor` on initial input AND first variance recount (`BLIND_RECOUNT_REQUIRED`). Flag `MARKING:MICRO_SKIMMING_SUSPECT` if a driver hits negative minor tolerance on `>= 80%` of recent shifts. |
| **12** | **Pre-Trip 8-Point DVIR Red-Tag Grounding Gate** | Role 4, Role 5, Role 9 | Any critical defect in the 8-point walkaround transitions vehicle to `RED_TAG_QUARANTINE` (`OUT_OF_SERVICE`), uncouples driver pairing, blocks Payloader/Dispatch selection, and creates a maintenance work order. |
