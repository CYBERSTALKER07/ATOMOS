# PegasusX Comprehensive Role Operations, Account Setup & Exhaustive Feature Catalog

> **Document Identifier**: `PEGASUSX-SPEC-ROLES-ALL-009`  
> **System Architecture**: Multi-Cell Federated Supply Chain Operating System  
> **Repository Root**: `/Users/shakhzod/Desktop/pegasusx`  
> **Data Plane**: Google Cloud Spanner (Multi-Region), Kafka Mesh, Redis Sentinel  
> **Client Surfaces**: Web (Next.js 15), Native macOS (Swift 6 / SwiftUI), Desktop (Tauri 2 / Rust), Native Mobile (Android Jetpack Compose & iOS SwiftUI)

---

## 1. Executive Framework: The 9 Interlocking Ecosystem Roles

The PegasusX ecosystem coordinates **9 Specialized Operational Roles** across commercial, manufacturing, logistics, retail, and governance domains. Each role operates within strict cryptographic boundaries, zero-trust state machines, and mathematical invariants:

| # | Role Persona | Primary Surface | Access Paradigm | Core Mandate |
| :--- | :--- | :--- | :--- | :--- |
| **1** | **Platform Owner / Super Admin** | Web Console & macOS Native | Cross-Tenant Federated | Global multi-cell orchestration, dual-control governance, KYB approval, take-rate economics. |
| **2** | **Supplier / Brand Executive** | Web Portal & Mobile Apps | Tenant-Scoped (`SupplierId`) | Catalog & tiered pricing, credit lines, demand forecasting, recall blast radius, factoring payouts. |
| **3** | **Factory / Production Manager** | Web Portal & Industrial Apps | Plant-Scoped (`FactoryId`) | Work order execution, BOM staging, GS1-128/SSCC-18 pallet serialization, QA lab certification. |
| **4** | **Warehouse & Fulfillment Manager** | Web Portal & Zebra RF Scanners | DC-Scoped (`WarehouseId`) | Dynamic wave planning, FEFO slotting, snake-walk pick dispatch, cold-chain sensor verification. |
| **5** | **Depot / Warehouse Cashier** | Cash Desk Web & Mobile Terminals | Cash Desk / Vault Bound | Zero-trust blind driver cash intake, denomination breakdown (200k..1k UZS), variance escalation. |
| **6** | **Yard Marshal & Dock Master** | Rugged Tablet / Gate Terminals | Gate & Yard Facility | Optical ALPR plate clearance, 8-point DVIR safety check, tamper seal verification, dwell time tracking. |
| **7** | **Carrier & Fleet Driver** | Native Android & iOS Apps | Device & Shift Attested | Pre-trip photo DVIR, H3 turn-by-turn navigation, 100m geofence doorstep unlock, cash/card collection. |
| **8** | **Retailer & Merchant Partner** | Desktop POS (Tauri 2) & Mobile | Store-Scoped (`RetailerId`) | Nearest DC discovery, multi-supplier cart, ParentOrders saga checkout, QR delivery handshake. |
| **9** | **Enterprise SRE & FinOps Lead** | Native macOS & Tauri 2 Desktop | Secure Enclave & Multi-Cloud IAM | MenuBarExtra pulse, Spanner latency tuning, Kafka lag matrix, DLQ replay, tenant cloud cost attribution. |

---

## 2. Role-by-Role Deep Dive: Account Setup, Daily Operations & Feature Catalog

---

### Role 1: Platform Owner / Super Admin

#### 1. Account Setup, Identity & Provisioning
- **Root Provisioning**: Created during initial multi-cell bootstrap via secure administrative CLI (`cmd/seed-multitenant`). Root administrative keys are generated in Google Cloud KMS and hardware-backed HSMs.
- **Multi-Factor Authentication (MFA)**: Mandatory enrollment in TOTP (RFC 6238) and FIDO2/WebAuthn. Single-factor access is mathematically blocked by [`mfa.RequireVerified`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/mfa).
- **Session & Scoping**: Issued a short-lived administrative JWT with `role: PLATFORM_ADMIN`. In [`auth.RequireTenant`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/tenant.go), Platform Admins are specifically exempted from single-tenant isolation, granting federated read/write authority across cells (`cell-uz`, `cell-eu`, `cell-us`, `cell-kz`).

#### 2. Daily Operational Cadence
1. **08:00 - Morning Health Audit**: Open macOS Native Control Tower or Web Console. Review global multi-cell health (P99 read/write latencies, Cloud Spanner abort rates).
2. **10:00 - Dual-Control Governance**: Triage the Maker-Checker inbox. Inspect proposed changes to money-affecting feature flags or tenant tier overrides. Provide secondary cryptographic sign-off.
3. **13:00 - Tenant KYB Reviews**: Review newly onboarded Suppliers and Retailers pending legal verification. Validate State Tax Committee (Soliq) registry filings, bank requisites, and corporate charters.
4. **16:00 - Outbox & Event Lag Triage**: Inspect Kafka consumer partition lag and the Cloud Spanner `OutboxDeadLetters` table. Trigger payload inspection and atomic in-order replayers on any quarantined messages.
5. **19:00 - Evening FinOps Settlement**: Reconcile daily platform take-rate revenue (dynamically configured per tenant market pack) against multi-cloud infrastructure burn rates. Seal the session audit trail.

#### 3. Exhaustive Feature Catalog
- **Global EKG Topology Cockpit**: Live interactive visualization of all federated regional cells, cross-cell replication health, and inter-service HTTP/gRPC latencies.
- **Dual-Control Maker-Checker Inbox**: Four-Eyes Principle approval interface for sensitive mutations (`AR_INVOICES_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`, `FISCAL_PROVIDER`, tenant KYB activation).
- **Tenant KYB Lifecycle Manager**: Search, inspect, verify, suspend, or terminate tenant organizations. Direct integration with Soliq API to verify Tax Identification Numbers (INN/TIN).
- **Master Product Catalog & Entity Resolution**: Automated deduplication engine merging raw supplier barcodes into unified Global Trade Item Numbers (GTIN) and 17-digit Soliq MXIK codes.
- **Cloud Spanner Outbox & DLQ Console**: Real-time table browser for unpublished outbox events and quarantined poison messages, featuring payload decoders and one-click atomic replayers.
- **FinOps Cloud Cost & Take-Rate Analyzer**: Real-time unit-economics dashboard breaking down cloud spend (Spanner processing units, egress, Redis, Kafka) per tenant against the configured platform take-rate.
- **Global Kill-Switch & Circuit Breaker Terminal**: Emergency controls to gracefully throttle or isolate individual regions, cells, suppliers, or payment rails during upstream outages.
- **Cryptographic Audit Trail Sealer**: Immutable queryable ledger tracking every administrative action, IP address, user agent, and cryptographic signature.

#### 4. Guardrails & Invariants
- *Zero Single-Admin Overrides*: High-risk flags strictly require two distinct administrator signatures; self-approval is rejected by database constraints.
- *Strict Read-Only Production DB*: Direct SQL mutation bypasses are prohibited; all state mutations must flow through audited Go service endpoints.

---

### Role 2: Supplier / Brand Executive

#### 1. Account Setup, Identity & Provisioning
- **Self-Service / Enterprise Registration**: Supplier submits legal organization details via `/v1/tenant/register` including Tax Identification Number (TIN/INN), VAT certificate, corporate headquarters, and contact officers.
- **Banking & Clearing Setup**: Binds Central Bank (CBU) settlement account details (20-digit IBAN and 5-digit Bank Code MFO) for automated clearing.
- **Platform Verification**: State transition `PENDING -> ACTIVE` executed upon Platform Admin dual-control review.
- **Staff Provisioning**: Supplier Admin invites subordinate accounts (`RoleAdmin`, `RoleFinance`, `RoleLogistics`) with granular role-based permissions.
- **ERP Integration**: Enterprise ERP systems (1C:Enterprise, SAP) connected via secure AS2, EDIFACT, or authenticated REST webhooks using API keys stored in HashiCorp Vault.

#### 2. Daily Operational Cadence
1. **08:30 - Commercial Dashboard Review**: Inspect total daily order volume, active backorders, pending approvals, and accounts receivable (AR) aging.
2. **10:30 - Pricing & Margin Ladder Tuning**: Adjust wholesale price lists, set quantity break discount ladders (e.g. 5% off >= 50 cases), and configure seasonal promotional campaigns.
3. **12:00 - Supply Request & Factory Dispatch**: Review AI demand forecasts; release replenishment work orders to contracted manufacturing plants.
4. **14:30 - Credit & AR Terms Oversight**: Review retailer accounts and repayment trends; configure open trade terms (Net-15/30) without artificial credit caps or checkout blocking.
5. **16:30 - Traceability & Quality Sandbox**: Run simulated blast-radius evaluations on stock lots flagged with quality complaints; execute targeted quarantines if required.
6. **18:30 - Financial Settlement & Payouts**: Review delivery receipts (EPODs); process early factoring cashouts (instant disbursable net bank payout per dynamic waterfall); reconcile Soliq fiscal e-facturas.

#### 3. Exhaustive Feature Catalog
- **Executive Commercial Command Center**: Live KPI cards covering gross merchandise value (GMV), order fulfillment velocity, return rates, DSO (Days Sales Outstanding), and gross margins.
- **Catalog & SKU Master Registry**: Full product lifecycle management with multi-language descriptions, packaging dimensions, pallet layer counts, and GS1 barcode bindings.
- **Tiered Volume Break & Pricing Engine**: Dynamic rules engine enabling complex wholesale pricing structures based on customer tier, order volume, and geographic zones.
- **Retailer Trade Credit & AR Manager**: Transparent account ledger, payment term management, balance tracking, and automated reconciliation across invoices and multi-tender settlements.
- **AI Demand Sensing & Replenishment Recommender**: Machine learning pipeline analyzing 28-day historical sales velocity, weather trends, and seasonality to recommend optimal reorder quantities.
- **Foundry Traceability & 5-Stage Blast Radius Sandbox**: Interactive graph visualization mapping contaminated stock lots to packaging batches, line-haul manifests, warehouse bins, and retailer shelves.
- **Kinetic Lot Quarantine Tool**: One-click quarantine mechanism that locks barcodes in WMS, cancels unpicked orders, and blocks retailer POS checkouts across the network.
- **Fleet Live Radar (MapLibre GL)**: Real-time map displaying moving delivery trucks, route deviation alerts, stops completed, and live refrigerated compartment temperatures.
- **Logistics Claims & OS&D Adjudication**: Portal to review and approve retailer-reported shipping discrepancies (overages, shortages, damages) and issue automated credit notes.
- **Bank Payout Waterfall & Factoring Terminal**: Treasury dashboard detailing daily sales proceeds, platform take-rate deductions, factoring fees, and automated Central Bank wire transfers.

#### 4. Guardrails & Invariants
- *Multi-Tenant Data Isolation*: Every database read and write is strictly filtered by `SupplierId` via [`auth.PreferTenantSupplierID`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/tenant.go). Cross-tenant data leaks trigger immediate 403 Forbidden.
- *Zero Unreserved Order Commits*: Orders cannot transition to `CONFIRMED` without an atomic Spanner reservation on existing inventory lots.

---

### Role 3: Factory / Production Manager

#### 1. Account Setup, Identity & Provisioning
- **Plant Setup**: Production facility registered under the parent Supplier organization with physical address, GPS geofence, and certified production lines.
- **Hardware Integration**: Factory floor terminals paired with fixed industrial barcode scanners, floor platform weigh scales, thermal label applicators, and machine PLC IoT gateways.
- **Staff Provisioning**: Production floor supervisors, machine operators, and QA laboratory technicians onboarded with device-bound PIN and badge authentication.

#### 2. Daily Operational Cadence
1. **07:00 - Morning Shift Handoff**: Inspect machine maintenance logs and calibrate packaging line load cells and label printers.
2. **08:00 - Supply Request Ingestion**: Ingest replenishment orders issued by the Supplier commercial team; explode Bill of Materials (BOM) into raw ingredient allocations.
3. **09:30 - Raw Material Verification**: Scan inbound raw material lots to establish full upstream lot-genealogy before feeding ingredients into processing hoppers.
4. **11:00 - High-Speed Production Run**: Monitor packaging lines; track real-time machine telemetry (throughput rate, cycle time, thermal seals); aggregate units into cases.
5. **14:00 - Palletization & GS1 Serialization**: Automatically stack cartons onto standard pallets; generate cryptographically unique SSCC-18 pallet license plates.
6. **15:30 - QA Lab Clearance**: Draw random specimen samples for laboratory analysis (microbiology, moisture, seal integrity). Record electronic Certificate of Analysis (CoA).
7. **17:00 - Loading Bay Transfer**: Consolidate passed pallets into Factory Internal Transfer Manifests; load line-haul trucks; apply high-security bolt seals.

#### 3. Exhaustive Feature Catalog
- **Supply Request & Work Order Dispatcher**: Queue and schedule manufacturing production orders against real-time warehouse inventory depletion rates.
- **Bill of Materials (BOM) & Lineage Tracker**: Multi-level recipe configurator linking raw ingredient batch numbers to finished product SKUs for full forward/backward traceability.
- **Machine IoT Telemetry Dashboard**: Real-time graphs showing packaging line speeds, motor amperages, heat seal temperatures, and automated stoppage incident logs.
- **Automated Case & Pallet Serializer**: GS1-128 and SSCC-18 barcode label generator producing compliant ZPL code for industrial thermal transfer printers.
- **QA Laboratory Inspection & CoA Vault**: Digital quality control module for recording lab test parameters, attaching lab photos, and digitally signing certificates of analysis.
- **Red-Tag Quarantine Lock**: Physical and digital quarantine trigger preventing defective or un-tested pallets from being loaded onto line-haul vehicles.
- **Factory Loading Bay & Yard Gate Manager**: Dock scheduling board assigning trailers to specific factory loading bays and validating driver arrival credentials.
- **Internal Transfer Manifest Generator**: Generates statutory e-waybills and custody transfer documentation for shipments transferring between the factory and central distribution centers.

#### 4. Guardrails & Invariants
- *Zero Unserialized Pallet Releases*: No pallet may leave the factory floor without a verified, un-duplicated SSCC-18 barcode committed to Cloud Spanner.
- *Strict Quarantine Isolation*: Pallets marked with failed QA lab results cannot be selected or scanned into transfer manifests.

---

### Role 4: Warehouse & Fulfillment Manager

#### 1. Account Setup, Identity & Provisioning
- **DC Configuration**: Distribution center boundaries, dock doors (inbound/outbound), staging bays, and climate-controlled chambers configured in the WMS engine.
- **Storage Topology Mapping**: Warehouse spatial layout defined down to Zone (Dry, Chilled, Frozen), Aisle, Bay, Level/Tier, and specific Bin coordinates.
- **Peripherals Provisioning**: Zebra Android RF handheld terminals configured with the PegasusX WMS mobile app utilizing the native Zebra DataWedge scanning API.
- **Staff Provisioning**: Warehouse managers, wave planners, pickers, packers, forklift operators, and dock workers assigned granular operational permissions.

#### 2. Daily Operational Cadence
1. **06:00 - Shift Initialization**: Power on RF scanners, verify thermal printer ribbon/labels, and check overnight cold-room temperature sensor logs.
2. **07:30 - Inbound ASN Receiving**: Check in line-haul trucks arriving from factories; scan inbound pallets; execute FEFO putaway algorithms to direct forklift bin placement.
3. **09:30 - Wave Planning & Resequencing**: Aggregate pending retailer orders into picking waves based on shipping zone, delivery priority, and carrier cutoff windows.
4. **11:00 - Snake-Walk Pick Execution**: Distribute optimized pick lists to pickers' RF terminals; verify mandatory barcode scans on every item pick to eliminate errors.
5. **14:00 - Packing & Cartonization**: Consolidate picked items into durable shipping cartons; apply SSCC labels; verify carton gross weights against calculated tolerances.
6. **16:00 - Cold-Chain Verification**: Stage temperature-sensitive cartons in pre-chilled outbound bays; verify continuous wireless BLE sensor logs (< -18°C frozen, 2-4°C chilled).
7. **17:30 - Outbound Truck Staging & Seal Handover**: Load delivery vehicles following LIFO sequence; apply tamper-evident bolt seals; hand over manifest to Yard Marshal.

#### 3. Exhaustive Feature Catalog
- **WMS Spatial Layout & Bin Manager**: Visual 2D/3D warehouse map editor defining storage locations, picking zones, forklift paths, and storage volume constraints.
- **Automated FEFO Slotting Engine**: Placement algorithm that prioritizes shortest-expiry inventory for front-bin storage to eliminate product spoilage and waste.
- **Inbound ASN Receiving Dock**: Scan-to-receive interface comparing physical pallet arrivals against factory manifests and automatically identifying OS&D variances.
- **Dynamic Wave Planning Console**: Intelligent order batching tool that groups hundreds of retailer orders into optimal picking waves and routes.
- **RF Terminal Snake-Walk Pick Interface**: Minimalist, high-contrast UI for Zebra handhelds guiding pickers along the shortest travel path with mandatory barcode confirmation.
- **Pack & Weigh Verification Station**: Integrated scale and scanner station checking actual parcel weight against theoretical SKU bill-of-materials weight to catch missing items.
- **BLE Cold-Chain Telematics Monitor**: Live temperature logging interface capturing readings from wireless sensors placed inside cold rooms and refrigerated shipping totes.
- **Cycle Counting & Inventory Audit Module**: Perpetual inventory audit tool generating randomized daily bin count tasks without disrupting active picking operations.
- **Staging Dock & Truck Manifest Board**: Outbound loading dashboard linking shipping cartons to specific delivery routes and tracking trailer departure status.

#### 4. Guardrails & Invariants
- *Zero Mis-Pick Commits*: Pickers cannot advance to the next pick item without successfully scanning the exact barcode and bin location.
- *Cold-Chain Lockout*: Staging is blocked if refrigerated bay sensor readings exceed safe thermal boundaries for longer than 15 consecutive minutes.

---

### Role 5: Depot / Warehouse Cashier

#### 1. Account Setup, Identity & Provisioning
- **Cash Desk Terminal Pairing**: Assigned to a specific physical cash desk workstation, cash drawer, and heavy-gauge drop-safe vault.
- **Authentication**: High-security authentication utilizing biometric fingerprint sensors or password + hardware TOTP MFA. Role `RoleCashier` / `RoleWarehouse` granted.
- **Shift Opening Balance**: Mandatory physical declaration and counting of the opening change float fund before beginning operations.

#### 2. Daily Operational Cadence
1. **14:00 - Cash Desk Open**: Log into Cashier Terminal; count and confirm opening float cash drawer; verify currency counter calibration.
2. **15:30 - Driver Return Queue Monitoring**: Monitor incoming delivery drivers returning from completed routes.
3. **16:00 - Zero-Trust Blind Bag Intake**: Receive tamper-evident deposit bag from driver; scan bag barcode; inspect tamper-evident security tape.
4. **16:30 - Banknote Denomination Counting**: Open cash bag under CCTV; feed banknotes into high-speed sorting machine; record counts across denominations (1,000 to 200,000 UZS).
5. **17:30 - Variance Reconciliation & Escalation**: Submit declared counts to Spanner; system compares against expected collected cash; issue digital clearance or dispute report.
6. **19:30 - Vault Deposit & CIT Armored Handover**: Bundle cash into bank-standard currency straps; drop into time-delay vault safe; generate Armored CIT handover manifest.
7. **20:30 - Shift Balancing & Drawer Closure**: Close daily cash session; print consolidated fiscal balance summary; seal cashier audit trail.

#### 3. Exhaustive Feature Catalog
- **Active Driver Return Queue**: Real-time list of delivery drivers returning to depot, displaying route status, return timestamps, and check-in sequence.
- **Zero-Trust Blind Turn-In Interface**: Secure declaration form where expected cash amounts are strictly masked (`0 UZS`) to prevent cashiers or drivers from gaming counts.
- **Denomination Breakdown Calculator**: Ergonomic rapid-entry interface for entering banknote quantities across all statutory Uzbek currency denominations (200k, 100k, 50k, 20k, 10k, 5k, 2k, 1k UZS).
- **Automated Discrepancy & Dispute Engine**: Real-time variance evaluator that flags overages and shortages, captures cashier and driver signed dispute notes, and posts adjustments to the driver's ledger.
- **Driver Shortfall Payroll Deduction Link**: Directly integrates unresolved cash shortages with the driver payroll system for automated payroll deductions.
- **Time-Delay Vault Safe Drop Depository**: Digital logging tool tracking physical bundle deposits into the depot's heavy-duty time-delay safe.
- **Armored Cash-in-Transit (CIT) Manifest Generator**: Generates compliant handover manifests and cryptographic barcode seals for licensed bank collection guards.
- **CCTV Timestamp Reconciliation Tool**: One-click tool generating exact CCTV camera timestamps for every cash count to expedite audit investigations.

#### 4. Guardrails & Invariants
- *Zero Wire Disclosure of Expected Cash*: In [`cashrecon/handlers.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/cashrecon/handlers.go), `ExpectedCashMinor` is strictly masked to `0` on driver queries until physical declaration is finalized.
- *Strict Dual-Signoff on Discrepancies*: Cash differences cannot be closed without digital signatures from both the cashier and the driver.

---

### Role 6: Yard Marshal & Dock Master

#### 1. Account Setup, Identity & Provisioning
- **Gate Terminal Provisioning**: Industrial ruggedized tablet or fixed booth PC running the Yard Management interface (`apps/payload-terminal` or `apps/warehouse-portal/app/fleet`).
- **Peripheral Integration**: Connected to dual high-resolution ALPR optical cameras, pneumatic boom barrier relays, yard traffic lights, and gross vehicle axle weighbridges.
- **Staff Provisioning**: Yard marshals, dock masters, and security gate staff authenticated with device-bound sessions.

#### 2. Daily Operational Cadence
1. **06:00 - Gate Initialization**: Test boom barrier manual override switches, calibrate ALPR optical cameras, and zero-balance the weighbridge scale.
2. **07:00 - Inbound Fleet Check-In**: Vehicles arrive; ALPR cameras capture plates; system checks active dispatch manifests and vehicle whitelists.
3. **08:30 - Pre-Trip DVIR Compliance Audit**: Verify driver's 8-point digital walkaround DVIR inspection record; turn away trucks with red-flag safety defects.
4. **10:00 - Physical Bolt Seal & Weighbridge Inspection**: Inspect physical bolt seals; scan seal barcode; record gross axle weights; clear truck through boom barrier.
5. **12:00 - Dock Scheduling & Dwell Optimization**: Direct trucks to assigned loading bays; monitor turnaround timers to keep dwell time under 10 minutes.
6. **16:30 - Outbound Departure Clearance**: Verify loaded cargo; apply new outbound high-security bolt seal; record seal serial; open outbound boom gate.

#### 3. Exhaustive Feature Catalog
- **Automated License Plate Recognition (ALPR) Gate Monitor**: High-speed OCR engine that recognizes vehicle license plates in real time and queries active Spanner manifests.
- **Vehicle Whitelist & Gate Access Controller**: Automated gate barrier controller that triggers pneumatic boom gates only when license plate and scheduled arrival match.
- **Digital DVIR Safety Compliance Validator**: Pre-entry safety gate interface that blocks vehicle entry if the driver has not passed the mandatory 8-point pre-trip DVIR check.
- **Cryptographic Bolt Seal Verification Console**: Scanner interface comparing physical bolt seal serial numbers against the cryptographic seal hash recorded in the loading manifest.
- **Gross Vehicle Axle Weighbridge Interface**: Digital scale readout capturing steer, drive, and trailer axle weights to prevent vehicle overloading and highway violations.
- **Dock Door Assignment & Scheduling Board**: Real-time dock allocation screen showing the status of each loading bay (Vacant, Docked, Loading, Cleared).
- **Turnaround & Dwell Time Analytics**: Real-time countdown timer tracking turnaround times for each truck, triggering alerts if dwell exceeds 10 minutes.
- **Hazardous & Cold-Chain Excursion Alarm**: Audible and visual alarm system alerting yard staff if arriving refrigerated trailers exhibit critical temperature excursions.

#### 4. Guardrails & Invariants
- *Zero Gate Clearance on Failed DVIR*: Vehicles with un-remediated safety defects are physically and digitally blocked from passing the inbound gate barrier.
- *Strict Seal Integrity Check*: A truck with a broken or mismatched bolt seal cannot depart the yard without formal investigation by security.

---

### Role 7: Carrier & Fleet Driver

#### 1. Account Setup, Identity & Provisioning
- **Driver Onboarding & CDL Verification**: Driver submits government ID, commercial driver's license (CDL), medical fitness certificate, and smartphone phone number.
- **Carrier Fleet Association**: Driver is bound to a registered carrier organization and assigned a specific commercial delivery vehicle.
- **App Installation & Peripheral Pairing**: Driver installs native Android (`apps/driver-app-android`) or iOS (`apps/driver-app-ios`) app. Pairs in-cab Bluetooth reefer temperature sensors and portable card POS terminals.
- **Device Attestation**: Device hardware fingerprint and cryptographic keys registered to prevent credential spoofing.

#### 2. Daily Operational Cadence
1. **06:30 - Pre-Trip Walkaround DVIR**: Conduct mandatory 8-point walkaround vehicle inspection with photo evidence before starting the engine.
2. **07:30 - Warehouse Loading & SSCC Scan**: Drive to loading dock; scan loaded SSCC-18 carton barcodes into vehicle custody; confirm total piece count.
3. **08:30 - Route Navigation**: Accept optimized multi-stop route sequence; follow offline-capable turn-by-turn MapLibre navigation.
4. **10:00 - Doorstep Arrival & Proximity Unlock**: Arrive at retailer store; 100m geofence and velocity anti-spoofing guard unlock doorstep delivery actions.
5. **10:30 - Offload, Inspection & Exception Handling**: Scan carton SSCC barcodes; customer inspects goods; record any damaged items (OS&D partial offload).
6. **11:00 - Payment Collection & QR Handshake**: Collect cash COD or tap card on mobile POS terminal; merchant scans dynamic single-use delivery QR on driver's phone; Soliq e-factura issued.
7. **16:30 - Multi-Stop Route Execution**: Repeat delivery flow across all assigned retail stops (1..N).
8. **18:00 - Depot Return & Blind Cash Turn-In**: Drive back to depot; seal collected cash in tamper-evident bag; present bag to Cashier for zero-trust blind turn-in.

#### 3. Exhaustive Feature Catalog
- **8-Point Photographic Pre-Trip DVIR**: Structured digital vehicle inspection module covering tires, brakes, steering, lights, mirrors, horn, windshield, and reefer, requiring photographic evidence.
- **Bluetooth Reefer Telematics Streamer**: Background telemetry service reading in-cab Bluetooth temperature sensors and streaming live temperature data to dispatch.
- **H3 Dynamic Route Navigation (MapLibre GL)**: High-performance vector map navigation system providing turn-by-turn guidance optimized for commercial truck traffic.
- **Proximity Settlement & Velocity Guard**: Anti-fraud mechanism requiring the driver to be within 100m of the retailer coordinate and rejecting GPS jumping at speeds > 120 km/h.
- **SSCC Barcode Offload Scanner**: Continuous camera/laser barcode scanning interface verifying each carton handed over to the retailer to eliminate mis-deliveries.
- **Doorstep OS&D Discrepancy Logger**: Exception handling interface allowing drivers to record damaged or refused cartons with photo evidence, recalculating order totals on the fly.
- **Payment Collection Module**: Multi-tender payment capture supporting cash cash-on-delivery (COD), Humo/Uzcard contactless card taps, and trade credit terms.
- **Two-Party Dynamic QR Handshake**: Dynamic, cryptographically signed single-use QR code generated on the driver's phone and scanned by the merchant to commit electronic proof of delivery.
- **Mid-Route Rescue Request**: Emergency module allowing drivers experiencing mechanical breakdowns to request a rescue truck, transferring remaining cargo and stops seamlessly.
- **Depot Blind Cash Turn-In Assistant**: End-of-shift guide helping drivers bundle collected cash into tamper-evident bags and record security void-tape numbers.

#### 4. Guardrails & Invariants
- *Proximity Lock on Settlement*: Payment collection and proof-of-delivery actions remain strictly locked until physical presence at the store coordinate is mathematically verified.
- *No Route Start without Cleared DVIR*: The turn-by-turn navigation engine cannot be engaged if the daily DVIR inspection has not been completed.

---

### Role 8: Retailer & Merchant Partner

#### 1. Account Setup, Identity & Provisioning
- **Store Registration & TIN Verification**: Retailer registers business name, Tax Identification Number (TIN/INN), VAT registration status, and commercial store address.
- **Automatic Soliq Verification**: System queries the State Tax Committee registry to confirm active legal status and corporate registration.
- **GPS Storefront Pin & Nearest DC Discovery**: Merchant pins exact store entrance coordinates; PegasusX geospatial routing assigns the closest fulfillment warehouse distribution center.
- **Uncapped Trade Credit & Payment Account Setup**: Merchant configures trade payment preferences; system enables open trade credit and multi-tender terms (cash, card, wire, credit) without artificial payment ceilings or credit caps.
- **Application Deployment**: Merchant installs desktop application (`apps/retailer-app-desktop` via Tauri 2) or mobile apps (`apps/retailer-app-android/ios`).

#### 2. Daily Operational Cadence
1. **08:00 - Morning Store Inventory Audit**: Review store stock levels; inspect AI-generated sell-through replenishment suggestions.
2. **09:00 - Catalog Browse & Cart Building**: Browse wholesale catalog; check real-time stock at the assigned DC; add products from multiple suppliers into smart cart.
3. **10:00 - Unified ParentOrders Checkout**: Review order summary; system applies tiered volume discounts; submit uncapped order via distributed ParentOrders saga.
4. **13:00 - Real-Time Delivery Radar Tracking**: Track approaching delivery vehicle on live interactive map; view driver name, truck license plate, and live reefer compartment temperature.
5. **15:00 - Doorstep Receiving & Inspection**: Driver arrives; physically inspect delivered cartons; report any shipping damages in real time.
6. **15:30 - QR Delivery Confirmation & Settlement**: Scan driver's delivery QR code to execute cryptographic handoff; pay via credit, cash, or card; receive instant Soliq fiscal e-factura.
7. **16:00 - Retail Shelf Restocking & POS Sales**: Restock store shelves; sell to walk-in consumers using the built-in offline Retailer Point-of-Sale (POS) system.

#### 3. Exhaustive Feature Catalog
- **Wholesale Multi-Supplier Catalog**: Rich product browser displaying high-resolution images, nutritional specs, carton pack sizes, and real-time stock levels at the nearest DC.
- **Tiered Volume Discount Engine**: Real-time pricing calculator displaying tiered volume break prices and notifying merchants of additional savings on larger orders.
- **Unified Multi-Supplier Cart & ParentOrders Saga**: Smart shopping cart that splits multi-supplier purchases into synchronized child orders while maintaining all-or-nothing rollback guarantees.
- **Trade Credit & Invoice Pay-Down Manager**: Financial dashboard showing total credit line, current balance, available headroom, and upcoming invoice due dates with pay-down options.
- **Real-Time Delivery Radar (MapLibre GL)**: Live tracking map showing vehicle location, estimated arrival time, traffic delays, and real-time temperature logs.
- **Doorstep Receiving & Barcode Verification**: Mobile camera scanner allowing merchants to scan delivered cartons to verify shipment accuracy before accepting custody.
- **In-Line OS&D Damage Reporting**: Instant dispute tool allowing merchants to photograph damaged goods at doorstep, triggering immediate invoice adjustments and credit notes.
- **Dynamic Two-Party QR Handshake**: Merchant-side QR scanner that scans the driver's dynamic token to record legally binding Electronic Proof of Delivery (EPOD).
- **Soliq Fiscal E-Factura Digital Vault**: Statutory document vault storing authenticated e-invoices directly linked to the State Tax Committee system for seamless accounting.
- **Built-in Offline Retailer POS**: Complete retail point-of-sale checkout system supporting barcode scanning, receipt printing, and local inventory tracking for consumer store sales.
- **AI Sell-Through Analytics & Reorder Prompts**: Inventory forecasting tool analyzing store sales trends and suggesting automated reorders before stockouts occur.

#### 4. Guardrails & Invariants
- *Uncapped Multi-Tender Policy (No Credit or Payment Limits)*: Orders and settlements are fully uncapped across trade credit, cash, and card tender. There are no artificial payment ceilings and no credit limit caps blocking checkout or fulfillment. Retailers can order at any scale with instant Spanner ACID reservation and multi-tender settlement.

---

### Role 9: Enterprise SRE & FinOps Lead

#### 1. Account Setup, Identity & Provisioning
- **Native Application Setup**: Installs native macOS Control Tower (`apps/control-tower-macos` built with Swift 6 and SwiftUI) or cross-platform Tauri 2 Desktop client.
- **Hardware Enclave & Keyring Binding**: Cryptographic authentication credentials stored in the Apple macOS Keychain / Secure Enclave; hardware YubiKey enrollment required.
- **Cloud IAM Bindings**: Binds read-only and emergency break-glass IAM roles to Google Cloud (Spanner, Cloud Run, GCS), Kafka mesh clusters, and Redis Sentinel instances.

#### 2. Daily Operational Cadence
1. **08:00 - MenuBar Glance & System Pulse**: Glance at the macOS MenuBarExtra indicator pill; confirm all cells (Toshkent, Samarqand, Fergana) are green.
2. **09:30 - Cloud Spanner Latency & Replica Tuning**: Monitor Spanner P99 read and write latencies; tune bounded staleness windows for read replicas during high load.
3. **11:30 - Kafka Partition Lag & Consumer Health**: Inspect consumer group partition lag across Kafka topics; scale AI worker and outbox processing pools via Kubernetes HPA.
4. **14:00 - Outbox Dead-Letter Queue Triage**: Check `OutboxDeadLetters` for poison payloads; export un-sandboxed diagnostic bundles; execute atomic message replays.
5. **16:00 - FinOps Multi-Cloud Spend Attribution**: Reconcile cloud infrastructure spend per tenant; evaluate unit economics (cost per order, cost per delivery kilometer).
6. **18:00 - Chaos Drills & Failover Testing**: Execute simulated cell failover drills, test network partitions, and verify disaster recovery recovery time objectives (RTO < 30s).

#### 3. Exhaustive Feature Catalog
- **macOS Native MenuBarExtra Status Pill**: Persistent macOS menu bar popover providing continuous real-time glanceability of system health, active alerts, and cell status.
- **Multi-Cell Spanner EKG & Latency Profiler**: Deep telemetry monitor displaying read/write P99 latency percentiles, lock wait durations, and transaction abort frequencies across regional cells.
- **Kafka Consumer Partition Lag Matrix**: Real-time grid displaying consumer lag, message throughput rates, and rebalance events across all core messaging topics.
- **Un-sandboxed Diagnostic Bundle Exporter**: One-click desktop tool that packages full system logs, Spanner schema snapshots, Kafka offsets, and environment metrics into encrypted `.tar.gz` diagnostic archives.
- **Outbox DLQ Diagnostic & Replay Terminal**: Visual debugger for poisoned outbox messages, featuring syntax-highlighted JSON decoders, stack traces, and atomic re-injection controls.
- **Redis Semaphore Permitter Gauge**: Real-time utilization meter tracking distributed semaphore permit pools across edge API gateways to prevent resource starvation.
- **Tenant-Attributed Cloud Spend Waterfall**: Granular FinOps cost-allocation engine attributing cloud infrastructure spend (compute, storage, network egress) to individual tenants and hubs.
- **Unit Economics Metric Engine**: Real-time calculator computing exact cloud cost per order, cost per kilometer traveled, and infrastructure margin percentages.
- **Cell Failover & Chaos Injection Sandbox**: Disaster simulation terminal allowing SREs to inject simulated network latency, cell failovers, and replica disconnects to test resilience.

#### 4. Guardrails & Invariants
- *Zero-Loss Outbox Guarantee*: Outbox events are written in the same ACID transaction as the domain entity; no event can be dropped without an alert in the DLQ.
- *Strict Memory Ring Buffer*: WebSocket event streaming clients maintain a fixed 50-event ring buffer with FIFO eviction, guaranteeing constant memory consumption and silky 60fps rendering.

---

## 3. Cross-Role Shared Ledger & Event Flow Architecture

The 9 roles interact continuously over a shared Cloud Spanner transactional ledger and an asynchronous Kafka outbox event bus:

```
[Retailer] ---> (Unified Checkout Saga) ---> [Cloud Spanner Multi-Region]
                                                       |
        +----------------------------------------------+---------------------------------------+
        |                                              |                                       |
        v                                              v                                       v
[Supplier Portal]                              [WMS Warehouse]                         [Factory Plant]
• Accepts Order                                • Ingests ASN                           • Ingests Supply Request
• Reviews Credit Headroom                      • Wave Planning                         • Explodes BOM
• Monitors Payouts                             • FEFO Picking                          • SSCC-18 Palletizing
        |                                      • Cold-Chain Check                      • QA Lab Clearance
        |                                              |                                       |
        v                                              v                                       v
[Yard Marshal] <---------------------------- [Delivery Manifest] <--------------------+
• ALPR Plate Scan
• 8-Point DVIR Gate Check
• High-Security Bolt Seal Match
        |
        v
[Carrier Driver]
• Turn-by-Turn H3 Navigation
• 100m Geofence Doorstep Unlock (Anti-Spoofing Guard)
• SSCC Carton Offload Scan
• Cash COD / Card POS Collection
• Two-Party Dynamic QR Handshake Commit
        |
        v
[Retailer Storefront]
• Electronic Proof of Delivery (EPOD) Confirmed
• Soliq Fiscal E-Factura Issued
• Shelves Restocked for Built-in Store POS
        |
        v
[Warehouse Cashier Desk]
• Zero-Trust Blind Bag Intake
• Denomination Breakdown Counter (200k..1k UZS)
• Spanner Cash Variance Resolution
• Heavy Drop-Safe Vault CIT Handover
        |
        v
[Platform Owner & Enterprise SRE / FinOps]
• Dual-Control Maker-Checker Governance
• Multi-Cell Latency Tuning (P99 < 10ms)
• Dynamic Platform Take-Rate Waterfall vs Cloud Burn Rate
• Continuous MenuBarExtra Status Sealing
```

---

## 4. Verification & Dual-Root Parity

This detailed role operations and feature specification is synchronized with 100% parity across both root workspaces:
- Primary Root: `/Users/shakhzod/Desktop/pegasusx/docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md`
- Mirror Root: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/roles/DETAILED_ROLE_OPERATIONS_AND_FEATURES.md`

All diagrams, state machines, and code paths reflect actual production implementations across `apps/backend-go`, `apps/admin-portal`, `apps/control-tower-macos`, `apps/supplier-portal`, `apps/factory-portal`, `apps/warehouse-portal`, and the native mobile applications.
