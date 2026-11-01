# PegasusX Master Cross-Role Inventory & Ecosystem Fleet Matrix

> **Document Scope**: Master System Inventory, Cross-Role Parity Matrix across 9 Personas & 24 Client Applications, End-to-End Multi-Role Saga Topology, Zero Hardcoded Fee Policy & Uncapped Multi-Tender Terms  
> **Repository Root**: `/Users/shakhzod/Desktop/pegasusx`  
> **Target Subsystems**: Cloud Spanner Multi-Cell Clusters, Kafka Event Mesh, Transactional Outbox, Native macOS / iOS / Android & Desktop Tauri Runtimes  
> **Detailed Feature Specifications**: [`docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md)  

---

## 1. Executive System Topology: The 9 Operational Roles

PegasusX is a hyper-scale, single-supplier multi-retailer (SSMR) and multi-cell distribution operating system. The ecosystem coordinates **9 distinct operational roles** spanning manufacturing, distribution center warehousing, loading bays, gate security, fleet logistics, retail POS storefronts, cash vault settlement, and enterprise cloud reliability:

```
+----------------------------------------------------------------------------------------------------------------------------------------+
|                                                   PEGASUSX 9-ROLE ECOSYSTEM TOPOLOGY                                                   |
+----+--------------------------------+----------------------------+----------------------------+----------------------------+-----------+
| #  | Operational Role & Persona     | Web / Desktop Application  | Native Android Application | Native iOS Application     | Backend   |
+----+--------------------------------+----------------------------+----------------------------+----------------------------+-----------+
| 1  | Platform Owner & Admin         | apps/admin-portal          | (CLI / Enclave Terminal)   | apps/control-tower-macos   | 81 Rts    |
| 2  | Supplier & Brand Executive     | apps/supplier-portal       | apps/supplier-app-android  | apps/supplier-app-ios      | 211 Rts   |
| 3  | Factory & Production Manager   | apps/factory-portal        | apps/factory-app-android   | apps/factory-app-ios       | 45 Rts    |
| 4  | Warehouse Manager (WMS)        | apps/warehouse-portal      | apps/warehouse-app-android | apps/warehouse-app-ios     | 119 Rts   |
| 5  | Warehouse & Depot Cashier      | apps/warehouse-portal      | apps/warehouse-app-android | apps/warehouse-app-ios     | 29 Rts    |
| 6  | Yard Marshal & Dock Master     | apps/warehouse-portal      | apps/driver-app-android    | apps/driver-app-ios        | 36 Rts    |
| 7  | Carrier & Fleet Driver         | (Mobile Dedicated)         | apps/driver-app-android    | apps/driver-app-ios        | 60 Rts    |
| 8  | Retailer & Storefront (POS)    | apps/retailer-app-desktop  | apps/retailer-app-android  | apps/retailer-app-ios      | 158 Rts   |
| 9  | Enterprise SRE & FinOps Lead   | apps/control-tower-macos   | (SRE Diagnostic Terminal)  | apps/control-tower-macos   | 34 Rts    |
+----+--------------------------------+----------------------------+----------------------------+----------------------------+-----------+
|    | TOTAL ECOSYSTEM FLEET          | 8 Desktop/Web Consoles     | 6 Native Android Apps      | 7 Native Apple Apps        | 773 Rts   |
+----+--------------------------------+----------------------------+----------------------------+----------------------------+-----------+
```

---

## 2. Master Directory of Role Specifications

Each role is governed by an exhaustive operational and architectural specification detailing account provisioning, hardware calibration, daily cadence, state transitions, and feature catalogs:

| Role # | Persona Identifier | Role Name & Domain Scope | Specification Link |
| :---: | :--- | :--- | :--- |
| **Role 1** | `ROLE_PLATFORM_ADMIN` | **Platform Owner & Global Administrator**: Multi-tenant cell federation, dual-control maker-checker governance, outbox DLQ triage, global catalog entity resolution. | [`ROLE_PLATFORM_ADMIN.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_PLATFORM_ADMIN.md) |
| **Role 2** | `ROLE_SUPPLIER` | **Supplier & Brand Executive**: Commercial command center, wholesale pricing ladders, 1C/SAP integration, Foundry 5-stage blast-radius recall sandbox, dynamic settlement waterfall. | [`ROLE_SUPPLIER.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_SUPPLIER.md) |
| **Role 3** | `ROLE_FACTORY` | **Factory & Production Manager**: Production scheduling, BOM explosion, raw material reservation, automated SSCC-18 pallet serialization, inter-facility transfer manifests. | [`ROLE_FACTORY.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_FACTORY.md) |
| **Role 4** | `ROLE_WAREHOUSE` | **Warehouse Operations Manager (WMS)**: ASN receiving, ABC velocity slotting, dynamic wave planning, snake-walk pick dispatch, BLE cold-chain monitoring. | [`ROLE_WAREHOUSE.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_WAREHOUSE.md) |
| **Role 5** | `ROLE_DEPOT_CASHIER` | **Warehouse Cashier & Depot Vault Ops**: Serialized tamper-evident bag intake, zero-trust dual-blind denomination counting ritual, RRN tender reclassification, CIT armored bank transfer. | [`ROLE_DEPOT_CASHIER.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_DEPOT_CASHIER.md) |
| **Role 6** | `ROLE_FLEET_YARD` | **Yard Marshal & Dock Master**: Optical ALPR gate security, 8-point pre-trip DVIR walkaround, automated mechanical Red-Tag grounding gate, high-security bolt seal verification. | [`ROLE_FLEET_YARD.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_FLEET_YARD.md) |
| **Role 7** | `ROLE_DRIVER` | **Carrier & Fleet Driver**: Turn-by-turn H3 vector navigation, 150m/50m doorstep geofence unlock with velocity anti-spoofing, dynamic two-party QR ePOD handshake, split-tender cash intake. | [`ROLE_DRIVER.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_DRIVER.md) |
| **Role 8** | `ROLE_RETAILER` | **Retailer & Merchant Partner (Retail OS)**: Wholesale catalog browse, ParentOrders checkout saga, offline-first desktop POS, delivery radar, 48h concealed damage claims. | [`ROLE_RETAILER.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_RETAILER.md) |
| **Role 9** | `ROLE_SRE_FINOPS` | **Enterprise SRE & FinOps Lead**: Native macOS MenuBarExtra pulse, Spanner multi-cell EKG latency tuning, Kafka consumer partition lag scaling, atomic DLQ replay, FinOps cost attribution. | [`ROLE_SRE_FINOPS.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_SRE_FINOPS.md) |
| **Shared** | `ROLE_PAYLOAD` | **Payloader & Loading Dock Master**: 3D volumetric bin-packing, crushability Ti/Hi rules, Gross Vehicle Weight axle scaling, returnable transport item (RTI) reverse-space accounting. | [`ROLE_PAYLOAD.md`](file:///Users/shakhzod/Desktop/pegasusx/docs/roles/ROLE_PAYLOAD.md) |

---

## 3. End-to-End 9-Role Interaction Sequence

```mermaid
sequenceDiagram
    autonumber
    actor Retailer as Role 8: Retailer Storefront
    participant Edge as Edge API Gateway
    participant Spanner as Cloud Spanner Multi-Cell
    actor Supplier as Role 2: Supplier Executive
    actor Factory as Role 3: Factory Manager
    actor WMS as Role 4: Warehouse Manager
    actor Yard as Role 6: Yard Marshal
    actor Driver as Role 7: Carrier Driver
    actor Cashier as Role 5: Depot Cashier
    actor SRE as Role 9: Enterprise SRE & FinOps

    %% Step 1: Procurement & Reservation
    Retailer->>Edge: POST /v1/order/create (Uncapped Multi-Tender Terms)
    Edge->>Spanner: ACID ReadWriteTransaction (Reserve Inventory - DRAFT -> RESERVED)
    Spanner-->>Retailer: 201 Created (Order Confirmed)
    Spanner-->>Supplier: WebSocket Push: New Confirmed Wholesale Demand

    %% Step 2: Manufacturing & ASN Transfer
    opt Replenishment Triggered
        Supplier->>Factory: Release Production Work Order
        Factory->>Factory: BOM Explosion & Pallet SSCC-18 Barcoding
        Factory->>WMS: Inter-Facility Transfer Manifest
        WMS->>WMS: ASN Ingestion & FEFO Aisle Slotting
    end

    %% Step 3: DC Wave Planning & Picking
    WMS->>WMS: Slot Orders into Pick Wave 81 (Snake-Walk Algorithm)
    WMS->>Spanner: Update Status = PICKED -> PACKED_SSCC -> STAGED

    %% Step 4: Fleet Yard Clearance & Manifest Sealing
    Yard->>Yard: Execute Mandatory 8-Point Pre-Trip DVIR Walkaround
    Yard->>Spanner: Authorize Vehicle (Status: DISPATCH_AUTHORIZED)
    Driver->>WMS: Load Pallets LIFO & Crimp High-Security Bolt Seal
    Yard->>Yard: Scan BoltSealBarcode & Weighbridge Gross Axle Scale
    Yard->>Spanner: Release Exit Barrier (Status: IN_TRANSIT)

    %% Step 5: Doorstep Delivery & Dynamic QR Handshake
    Driver->>Driver: H3 Turn-by-Turn GPS Navigation to Storefront
    Driver->>Driver: Proximity Handshake (< 100m Geofence + Velocity Guard)
    Retailer->>Driver: Scan Dynamic QR Token (EPOD Cryptographic Sign-Off)
    Retailer->>Driver: Settle via Uncapped Cash / Card / Credit Terms
    Driver->>Spanner: Update Status = ARRIVED -> DELIVERED -> FISCALIZING

    %% Step 6: End-of-Shift Cash Turn-In & Vault Balancing
    Driver->>Cashier: Turn In Serialized Tamper-Evident Cash Bag
    Cashier->>Cashier: Execute Dual-Blind Denomination Counting Ritual (200k..1k UZS)
    Cashier->>Spanner: Reconcile Session to BALANCED; Append DepotVaultLedgerEntries
    Cashier->>Spanner: Handover Consolidated Vault Cash to Armored CIT Courier

    %% Step 7: FinOps Waterfall & System Health Sealing
    Spanner-->>SRE: Ingest Order State Commit to Outbox
    SRE->>SRE: Compute Attributed Multi-Cloud Spend & Reconcile Dynamic Platform Take-Rate
    SRE->>SRE: Verify Zero Outbox Lag & Seal System Integrity in MenuBarExtra
```

---

## 4. Ecosystem Policies & Immutable Invariants

### 4.1. Uncapped Multi-Tender Settlement Policy ("No Limits")
- **Uncapped Multi-Tender Policy**: All payment and credit ceilings are fully deactivated across the ecosystem (`AML_LIMITS_DISABLED=1` / `UNLIMITED_PAYMENTS=1`). Commercial B2B orders of any volume are permitted across cash, Humo/Uzcard corporate cards, and open trade credit without artificial ceilings.
- **Zero Artificial Credit Holds**: Credit checks during order checkout are unblocked (`CREDIT_RESERVE_AT_CREATE` disabled by default), enabling flexible wholesale relationships with continuous Spanner ACID tracking.
- **Enterprise Ledger Auditability**: While limits are deactivated, all financial movements continue to be recorded transparently in `MerchantDailyCashExposure` and `DepotVaultLedgerEntries` for tax, accounting, and anti-fraud auditing.

### 4.2. Dynamic Market Pack Fee Policy (Zero Hardcoded Values)
- **Zero Hardcoded Take-Rates or Fees**: All platform fees, logistics handling tariffs, Soliq VAT withholdings, and payment gateway cuts are dynamically configured per tenant contract via declarative Market Packs:
  $$\text{NetSupplierPayout} = \text{GrossGMV} - (\text{PlatformFee} + \text{LogisticsTariff} + \text{SoliqVAT} + \text{GatewayFee})$$
- **Per-SKU Soliq VAT Determination**: VAT is dynamically computed per item based on official 17-digit State Tax Committee MXIK/IKPU classifications and merchant tax regimes.

### 4.3. Currency Precision & Architectural Invariants
- **Tiyin Integer Math Invariant**: Floating-point numbers are strictly prohibited in financial paths. All currency values are stored, calculated, and transmitted as 64-bit integer minor units (`1 UZS = 100 Tiyin`).
- **Multi-Cell Request-Scoped Tenancy**: Every Spanner transaction strictly enforces `TenantContext` isolation, preventing cross-tenant data leakage or IDOR vulnerabilities.
