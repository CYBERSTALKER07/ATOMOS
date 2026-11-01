# Role 6 Specification: Yard Marshal & Fleet Maintenance Operations

> **Role Number**: Role 6 of 9  
> **Persona ID**: `ROLE_YARD_MARSHAL` / `ROLE_FLEET_YARD`  
> **Domain Scope**: Depot Gate Security, Optical ALPR Clearance, 8-Point Pre-Trip DVIR Walkaround, Automated Red-Tag Grounding Gate, Dynamic Yard Bay Slotting, High-Security Bolt Seal Verification, Weighbridge Axle Load Monitoring & Mid-Route Fleet Breakdown Rescue  
> **Authoritative Backend Seams**:  
> - Foundry Fleet & Yard Core: [`apps/backend-go/foundry/fleet_yard.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundry/fleet_yard.go)  
> - Foundry Fleet Routes: [`apps/backend-go/foundryroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)  
> - Driver & Route Dispatch Seams: [`apps/backend-go/driverroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/driverroutes)  
> - Pre-Execution Edge-Case Guards: [`apps/backend-go/foundry/edge_guards.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundry/edge_guards.go)  
> **Frontend & Native Surfaces**:  
> - Warehouse Yard & Fleet Station: [`apps/warehouse-portal/app/fleet`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-portal/app/fleet)  
> - Yard Inspection Components: [`apps/warehouse-portal/components/fleet`](file:///Users/shakhzod/Desktop/pegasusx/apps/warehouse-portal/components/fleet)  
> - Fleet Yard Monitor: [`apps/admin-portal/components/FleetYardMonitor.tsx`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/components/FleetYardMonitor.tsx)  
> - Driver Fleet Inspection Views: [`apps/driver-app-ios/driverappios/driverappios/Views/FleetDVIRView.swift`](file:///Users/shakhzod/Desktop/pegasusx/apps/driver-app-ios/driverappios/driverappios/Views/FleetDVIRView.swift), [`apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/fleet`](file:///Users/shakhzod/Desktop/pegasusx/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/fleet)  
> **Primary Cloud Spanner Tables**: `FleetDVIRInspections`, `YardGateCheckEvents`, `Vehicles`, `DriverShifts`, `MaintenanceWorkOrders`  

---

## 1. Executive Summary & Operational Charter

The **Yard Marshal & Dock Master (Role 6)** governs physical fleet readiness, depot perimeter security, and traffic orchestration across the warehouse yard and loading docks. In an enterprise FMCG logistics network, rapid vehicle turnaround (< 10 minutes from gate to loading bay) must co-exist with uncompromising mechanical safety and cold-chain compliance.

The Yard Marshal enforces four mandatory pillars:
1. **Perimeter Security & Automated License Plate Recognition (ALPR)**: Continuous optical surveillance verifying vehicle license plates, driver credentials, and physical bolt seals at entry and exit barriers.
2. **Mandatory 8-Point Pre-Trip DVIR Walkaround**: Strict physical and photographic verification of vehicular roadworthiness (steering, service brakes, tire tread, lights, fluids, emergency gear, cargo box sanitation, and calibrated reefer temperatures).
3. **Automated Mechanical Red-Tag Grounding Gate**: Any failed critical inspection point automatically transitions the vehicle to `RED_TAG_QUARANTINE` (`OUT_OF_SERVICE`), severs active driver pairings, locks the vehicle out of the WMS loading queue, and spawns an authenticated workshop work order.
4. **Cold-Chain Predictive Telematics & Emergency Mid-Route Rescue**: Continuous monitoring of reefer compartment temperature curves and compressor performance, coupled with rapid mid-route breakdown rescue protocols that isolate thermal excursions and split driver financial liabilities.

---

## 2. Account Setup, Identity, Hardware & Provisioning

```
[Ruggedized Tablet Setup] -> [ALPR Camera Calibration] -> [Weighbridge Interface] -> [Yard Bay Sensor Mesh]
```

### 2.1. Yard Marshal Identity & Security Enrollment
1. **Account Provisioning**: The Logistics Operations Director invites the Yard Marshal via [`POST /v1/staff/invite`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/staffinvite) with role `ROLE_YARD_MARSHAL`, bound to the regional distribution center (`WarehouseID`).
2. **MFA & Shift Pairing**: The Marshal enrolls via mobile authenticator or hardware security key ([`apps/backend-go/mfa`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/mfa)). Every gate barrier override or Red-Tag release requires authenticated step-up authorization.

### 2.2. Hardware Peripheral & Yard Station Calibration
1. **Ruggedized Field Tablet Setup**: The Yard Marshal operates on a ruggedized MIL-STD-810H tablet (Samsung Galaxy Tab Active4 Pro or iPad Pro in OtterBox Defender) running the native PegasusX Yard App.
2. **ALPR Security Gate Camera Calibration**: Gate lane optical cameras (Axis or Hikvision IP cameras running OCR firmware) connect to the local depot edge gateway. The video stream is calibrated to read Uzbekistan vehicle license plates (e.g., `01 777 AAA`, `10 882 BAB`) with < 200ms latency.
3. **Electronic Weighbridge & Axle Scale Interface**: The depot entrance weighbridge connects via Modbus TCP / RS-485 to feed automated gross vehicle weight and per-axle weight readings directly into the intake gate event.
4. **BLE Yard Beacon Mesh**: Bluetooth Low Energy (BLE) anchors stationed across yard parking bays (`BAY-01` to `BAY-32`) locate vehicles in real time as they maneuver across the apron.

---

## 3. Comprehensive Role User Flow & Operational Cadence

```mermaid
sequenceDiagram
    autonumber
    actor Driver as Carrier / Fleet Driver
    actor Marshal as Yard Marshal / Dock Master
    participant Tablet as Ruggedized Yard Tablet
    participant Foundry as Foundry Fleet Engine (Go)
    participant Spanner as Cloud Spanner Multi-Cell
    actor Payloader as Warehouse Payloader
    actor Mechanic as Depot Workshop Mechanic

    %% Phase 1: Morning Gate-In & ALPR Clearance
    Note over Driver,Spanner: Phase 1: Gate Security Inbound Check-In & ALPR Scan
    Driver->>Marshal: Arrives at Depot Security Gate (Vehicle: TRUCK-104)
    Marshal->>Tablet: ALPR Camera captures license plate (01 777 AAA); Scan Driver CDL
    Tablet->>Foundry: POST /v1/foundry/fleet/gate-events (Direction: GATE_IN, Odometer: 142050)
    Foundry->>Spanner: Verify Vehicle Registration & Active Driver Assignment
    Spanner-->>Foundry: Clearance Approved (Vehicle Status: ACTIVE)
    Foundry->>Spanner: Assign Dynamic Parking Bay (BAY-08: INBOUND_STAGING)
    Foundry-->>Tablet: Gate Barrier Opens; Routing HUD Guides Driver to Bay 08

    %% Phase 2: Mandatory 8-Point Pre-Trip DVIR Walkaround
    Note over Driver,Foundry: Phase 2: Mandatory 8-Point Pre-Trip DVIR Walkaround
    Marshal->>Driver: Initiates 8-Point Pre-Trip Walkaround at Vehicle Side
    Marshal->>Tablet: Inspects Steering, Brakes, Tires, Lights, Fluids, Safety Kit, Sanitation, Reefer
    alt All 8 Points Pass
        Marshal->>Tablet: Submit Certified Inspection (All Checks = TRUE)
        Tablet->>Foundry: POST /v1/foundry/fleet/dvir (Passed = true)
        Foundry->>Spanner: Insert FleetDVIRInspections (Status: DISPATCH_AUTHORIZED)
        Foundry-->>Tablet: Green Badge: DISPATCH_AUTHORIZED (Valid for 24 Hours)
        Foundry-->>Payloader: Vehicle Unlocked for Loading Bay Dock Assignment
    else Critical Defect Detected (e.g. Brake Pressure Failure / Reefer Drift)
        Marshal->>Tablet: Flag Defect (Brake Fluid Leak + Photo Proof)
        Tablet->>Foundry: POST /v1/foundry/fleet/dvir (Passed = false, DefectNotes: "Brake Leak")
        Foundry->>Spanner: Transition Vehicle to RED_TAG_QUARANTINE (OUT_OF_SERVICE)
        Foundry->>Spanner: Atomically Sever Driver-Vehicle Pairing
        Foundry->>Spanner: Insert MaintenanceWorkOrders (Priority: CRITICAL)
        Foundry-->>Tablet: Audio Horn & Red Flashing UI: VEHICLE GROUNDED
        Foundry-->>Mechanic: Workshop Dispatch Order Generated
    end

    %% Phase 3: Loading Bay Alignment & Axle Weight Verification
    Note over Marshal,Payloader: Phase 3: Loading Bay Alignment & Weighbridge Check
    Payloader->>Marshal: Loading Complete on Bay 08 (Reverse-Stop LIFO Staged)
    Marshal->>Tablet: Verify Gross Vehicle Weight on Weighbridge Scale
    Foundry->>Foundry: Check Gross Weight vs GVWR & Axle Limits
    Marshal->>Driver: Inspect High-Security Steel Bolt Seal & Crimp
    Marshal->>Tablet: Scan BoltSealBarcode (SEAL-992140)
    Tablet->>Foundry: POST /v1/foundry/fleet/gate-events (Action: BIND_BOLT_SEAL)
    Foundry->>Spanner: Hash BoltSealBarcode with Payloader Manifest Hash

    %% Phase 4: Gate-Out Clearance & Route Dispatch
    Note over Driver,Spanner: Phase 4: Gate-Out Clearance & Security Barrier Open
    Driver->>Marshal: Pulls to Exit Barrier to begin delivery run
    Marshal->>Tablet: Scan Bolt Seal & Verify Manifest Hash
    Tablet->>Foundry: POST /v1/foundry/fleet/gate-events (Direction: GATE_OUT)
    Foundry->>Spanner: Verify DVIR is DISPATCH_AUTHORIZED & Seal is INTACT
    Spanner-->>Foundry: All Invariants Satisfied
    Foundry-->>Tablet: 200 OK: Barrier Lifted; Manifest Timestamped Dispatched
    Foundry->>Spanner: Update Route Status = EN_ROUTE

    %% Phase 5: Mid-Route Breakdown Rescue (Exception Path)
    Note over Driver,Foundry: Phase 5: Mid-Route Breakdown Rescue & Thermal Quarantine
    Driver->>Foundry: Emergency Call: Broken Axle at Route Stop 4
    Marshal->>Tablet: Activate Mid-Route Fleet Rescue Protocol
    Tablet->>Foundry: POST /v1/foundry/fleet/mid-route-rescue (BrokenTruckID, RescueTruckID, HandoverStop: 4)
    Foundry->>Foundry: Freeze Broken Driver 1 Cash Liability at Stops 1..4
    Foundry->>Foundry: Evaluate Reefer Excursion Duration on Chilled/Frozen Totes
    alt Thermal Excursion Exceeded Limit
        Foundry->>Spanner: Stamp Totes with MARKING:THERMAL_QUARANTINE_SPOILED
        Foundry-->>Tablet: Lock Spoiled Totes from Transfer (Must Return to Bio-Disposal)
    end
    Foundry->>Spanner: Transfer Surviving Totes to Rescue Driver 2 under New Bolt Seal
    Foundry->>Spanner: Transfer Cash/Custody Liability for Stops 5..N to Rescue Driver 2
    Foundry-->>Marshal: Rescue Vehicle En Route with Dynamic Manifest
```

---

## 4. Exhaustive Feature Catalog

### 4.1. ALPR Optical Security Gate Monitor
- **Feature Name**: `ALPRGateMonitor`
- **Purpose**: Automates depot gate throughput, captures vehicle ingress/egress timestamps, logs odometer mileage, and prevents unauthorized vehicle movements.
- **UI Mechanics**: Live RTSP camera feed overlay with optical bounding box highlighting vehicle license plates. Instant visual badge displaying vehicle ownership (`COMPANY_FLEET`, `3PL_CARRIER`, `VISITOR`), driver name, and active dispatch status.
- **Backend API Seam**: [`POST /v1/foundry/fleet/gate-events`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Pre-Execution Invariants**:
  - `GATE_OUT` requests require an intact `BoltSealBarcode` matching the sealed manifest hash.
  - Vehicles with active `RED_TAG_QUARANTINE` or pending maintenance holds are hard-blocked at the gate barrier (`409 Conflict`).

### 4.2. Mandatory 8-Point Pre-Trip DVIR Inspection Suite
- **Feature Name**: `DVIRWalkaroundInspection`
- **Purpose**: Enforces statutory transportation safety regulations and food safety standards before any commercial vehicle takes to public roads.
- **UI Mechanics**: Step-by-step checklist requiring affirmative toggle and photographic evidence for each inspection point:
  1. `steering_and_horn_ok`: Steering gear backlash and audible horn warning.
  2. `service_and_parking_brakes_ok`: Air brake pressure buildup, brake pedal resistance, and parking brake hold.
  3. `tires_tread_and_pressure_ok`: Minimum 3.0mm tread depth, no sidewall bulges, tire pressure verified via digital gauge.
  4. `lights_and_hazards_ok`: High/low beam headlights, brake lights, turn indicators, side markers, and hazard flashers.
  5. `wipers_and_fluids_ok`: Wiper blade condition, washer spray, engine oil level, coolant level, and brake fluid level.
  6. `emergency_kit_and_extinguisher_ok`: Charged fire extinguisher, high-visibility vest, reflective warning triangles, and first-aid kit.
  7. `cargo_box_sanitation_ok`: Cargo compartment washed, swept clean, free of chemical odors, pest-free (HACCP compliance).
  8. `reefer_temp_calibrated_ok`: Cold-chain multi-temp compartment calibrated, digital probe operational, pre-cooled to target setpoint ($-18^\circ\text{C}$ frozen, $+4^\circ\text{C}$ chilled).
- **Backend API Seam**: [`POST /v1/foundry/fleet/dvir`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Certification Window**: A passed DVIR inspection is valid for 24 hours. Expired inspections automatically reset the vehicle state to `AWAITING_INSPECTION`.

### 4.3. Kinetic Red-Tag Grounding Gate & Workshop Link
- **Feature Name**: `RedTagGroundingGate`
- **Purpose**: Prevents unsafe or non-compliant vehicles from being driven or loaded with commercial merchandise, eliminating driver discretion and management pressure.
- **UI Mechanics**: High-contrast red screen banner with warning klaxon. Displays defect details, timestamp, Marshal ID, and required repair specifications.
- **Backend Implementation**: [`apps/backend-go/foundry/fleet_yard.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundry/fleet_yard.go#L85-L120).
- **Automated Actions**:
  - Immediately transitions vehicle state to `RED_TAG_QUARANTINE` (`OUT_OF_SERVICE`).
  - Uncouples any assigned driver from the vehicle in Cloud Spanner.
  - Blocks the vehicle from selection in WMS wave loading, route planning, and gate exit dispatch.
  - Automatically generates an authenticated `MaintenanceWorkOrder` in the depot workshop queue.
  - Releasing a Red-Tag strictly requires secondary mechanic sign-off via [`POST /v1/foundry/fleet/red-tag/release`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes).

### 4.4. Dynamic Yard Bay Allocation & Turnaround Radar
- **Feature Name**: `YardBayTurnaroundRadar`
- **Purpose**: Directs incoming trucks to optimal yard bays to prevent apron gridlock and maintain a < 10 minute gate-to-dock turnaround SLA.
- **UI Mechanics**: Bird's-eye 2D SVG canvas of the depot yard displaying all 32 bays color-coded by occupancy and purpose:
  - `INBOUND_STAGING`: ASN receiving from manufacturing plants.
  - `CROSS_DOCK`: Fast-turn pallet transfers between line-haul and last-mile vehicles.
  - `OUTBOUND_LOADING`: Route delivery pick/pack loading bays.
  - `MECHANICAL_SERVICE`: Workshop repair bays and wash racks.
- **Turnaround Timer**: Displays running countdown timers for docked vehicles. Alerts when dwell time exceeds 25 minutes.

### 4.5. Outbound High-Security Bolt Seal Binder
- **Feature Name**: `BoltSealBinder`
- **Purpose**: Eliminates in-transit cargo theft and carton pilferage by enforcing cryptographic binding between the physical bolt seal barcode and the manifest hash.
- **UI Mechanics**: Camera-based barcode scanner capturing the serialized steel bolt seal barcode (`SEAL-...`). Displays verified manifest line count, carton totals, and customer drop sequence.
- **Backend API Seam**: [`POST /v1/foundry/fleet/gate-events`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Security Invariant**: The warehouse payloader binds the seal at the loading dock; the Yard Marshal independently scans and verifies the seal at the exit gate. Any barcode discrepancy aborts gate opening.

### 4.6. Weighbridge Gross Vehicle Weight & Axle Scale Monitor
- **Feature Name**: `WeighbridgeAxleMonitor`
- **Purpose**: Protects vehicles from structural damage and prevents road transport regulatory fines by ensuring vehicles never exceed Gross Vehicle Weight Ratings (GVWR) or individual axle load limits.
- **UI Mechanics**: Real-time scale readout showing front axle weight, drive axle weight, trailer axle weight, and gross vehicle weight in kilograms. Visual green/red bar gauges compare actual weights against vehicle tare and maximum limits.
- **Validation Rule**: If gross weight exceeds GVWR by > 1.5%, the gate barrier locks and the truck is routed back to the payloader dock for offloading.

### 4.7. Mid-Route Breakdown Rescue & Thermal Quarantine Coordinator
- **Feature Name**: `MidRouteBreakdownRescue`
- **Purpose**: Rescues active delivery routes when a truck suffers a roadside mechanical breakdown or refrigeration compressor failure, preserving customer order SLAs and preventing food safety hazards.
- **UI Mechanics**: Emergency dispatch console mapping broken vehicle location, nearby available certified rescue vehicles, and temperature telemetry trends.
- **Backend API Seam**: [`POST /v1/foundry/fleet/mid-route-rescue`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/foundryroutes)
- **Deterministic Rescue Protocol**:
  1. **Cash Liability Freezing**: Driver 1's cash liability is hard-frozen at completed stops (`Stops 1..K`).
  2. **Thermal Excursion Evaluation**: The engine computes the duration that chilled or frozen totes spent above safe temperature thresholds ($\Delta T > \text{Tolerance}$). Spoiled totes are tagged `MARKING:THERMAL_QUARANTINE_SPOILED` and locked from transfer.
  3. **Custody Transfer**: Surviving totes and cash liabilities for `Stops K+1..N` are transferred to Rescue Driver 2 under a fresh emergency bolt seal.

---

## 5. Security Markings, Error Matrix & Operational Invariants

### 5.1. Yard Marshal Error Codes
| Error Code | HTTP Status | Root Cause | Remediation Procedure |
| :--- | :---: | :--- | :--- |
| `ERR_GATE_OUT_SEAL_MISMATCH` | `409 Conflict` | Physical bolt seal does not match the sealed manifest hash. | Hold vehicle at exit; inspect trailer doors and re-audit loading manifest. |
| `ERR_VEHICLE_RED_TAGGED` | `422 Unprocessable` | Attempted to dispatch a vehicle under active mechanical Red-Tag. | Vehicle cannot move. Route must be re-assigned to a certified standby van. |
| `ERR_DVIR_EXPIRED_OR_MISSING` | `412 Precondition Failed` | Vehicle lacks a valid 24-hour DVIR inspection certificate. | Marshal must execute the mandatory 8-point physical walkaround immediately. |
| `ERR_AXLE_WEIGHT_OVERLOAD` | `400 Bad Request` | Weighbridge detected vehicle gross weight or axle load exceeding GVWR. | Route vehicle back to loading bay for load re-balancing or partial de-manifesting. |
| `ERR_THERMAL_EXCURSION_SPOILED` | `409 Conflict` | Chilled/frozen totes suffered irreversible temperature spike during breakdown. | Spoilage quarantine confirmed. Totes routed to disposal; replacement wave triggered. |

### 5.2. Role Invariants Checklist
- [x] **Zero Gate Exit Without Intact Seal**: Vehicles on active route manifests cannot exit the perimeter without a verified, matching bolt seal barcode.
- [x] **Non-Bypassable Red-Tags**: Only certified workshop mechanics can clear a mechanical Red-Tag; drivers and dispatchers cannot override.
- [x] **HACCP Cold-Chain Compliance**: Pre-trip reefer temperature setpoints must be verified before perishable loading.
- [x] **Split Liability on Rescue**: Breakdown rescue deterministically splits cash liability between initial and rescue drivers.
