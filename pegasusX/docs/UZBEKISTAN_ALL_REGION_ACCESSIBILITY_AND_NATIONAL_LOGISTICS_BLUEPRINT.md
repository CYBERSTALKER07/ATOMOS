# PEGASUS.X: ALL-UZBEKISTAN ACCESSIBILITY & NATIONAL LOGISTICS BLUEPRINT
## Nationwide Multi-Region, Multi-Dialect, Offline-Resilient Supply Chain Architecture

**Document Version:** 1.0.0  
**Target Platform:** Pegasus.X (`/Users/shakhzod/Desktop/pegasus.x`)  
**Scope:** Complete accessibility across all 14 administrative divisions of the Republic of Uzbekistan.  
**Operating Standard:** Big-Tech Compiler-Grade, Zero-Mock, Fail-Closed Enterprise Reality.

---

## 1. Executive Mission & Territorial Realities

For Pegasus.X to be genuinely accessible **all over Uzbekistan**, it cannot be architected as a "Tashkent-only" software solution. Outside the capital, supply chains confront severe structural challenges:
1. **Extreme Physical Topography**: The Kamchik Pass (2,268m) isolates the Fergana Valley; the Taxtaqoracha Pass separates Samarkand and Kashkadarya; 450+ km of barren Kyzylkum desert highway (A-380) disconnects Bukhara from Khorezm and Karakalpakstan.
2. **Telecommunications Instability**: Rural cotton/wheat districts, mountain tunnels, and subterranean bazaar stalls frequently degrade to 2G/EDGE or complete radio silence (Ucell, Beeline UZ, Mobiuz, Uzmobile).
3. **Linguistic & Script Diversity**: Official Latin Uzbek coexists with widespread Cyrillic Uzbek in traditional commerce, Karakalpak (*Qaraqalpaq tili*) in the sovereign Republic of Karakalpakstan, and Russian in corporate wholesale.
4. **Informal Credit & Cash Dominance**: Rural and bazaar commerce (*bozorlar*) relies heavily on physical cash, handwritten debt ledgers (*daftar / nasiya*), and bilateral trust credit, bound by strict Central Bank cash transaction caps (25,000,000 UZS).
5. **Traditional Trade Supremacy**: Over 85% of retail FMCG volume flows through micro-stores (*mahalla dukonlari*) visited by foot or car by Field Sales Representatives (*savdo agentlari*).

---

## 2. National Geographic & Facility Topology (14 Regions)

Pegasus.X models Uzbekistan through the official **SOATO** (*Maʼmuriy-hududiy birliklarni belgilash tizimi*) administrative hierarchy, grouping 208 districts into 5 Macro-Logistics Zones:

```
                                  ┌─────────────────────────────────┐
                                  │   TASHKENT MASTER CENTRAL DC    │
                                  │  (CDC-01, Sergeli Logistics Hub)│
                                  └───────────────┬─────────────────┘
                                                  │
                ┌─────────────────────────────────┼─────────────────────────────────┐
                │                                 │                                 │
  Line-Haul over Kamchik Pass       Line-Haul via M-39 Corridor       Line-Haul via A-380 Desert Corridor
                │                                 │                                 │
                ▼                                 ▼                                 ▼
┌───────────────────────────────┐ ┌───────────────────────────────┐ ┌───────────────────────────────┐
│       EAST RDC (KOKAND)       │ │     CENTRAL RDC (SAMARKAND)   │ │      WEST RDC (URGENCH)       │
│ Serves: Fergana, Andijan,     │ │ Serves: Samarkand, Jizzakh,   │ │ Serves: Khorezm &             │
│         Namangan              │ │         Syrdarya, Navoi       │ │         Rep. of Karakalpakstan│
└───────────────┬───────────────┘ └───────────────┬───────────────┘ └───────────────┬───────────────┘
                │                                 │                                 │
        10 Local Cross-Docks              12 Local Cross-Docks              8 Local Cross-Docks
        (Andijan, Namangan,               (Jizzakh, Guliston,               (Nukus, Kungrad,
         Fergana, Margilan)                Navoi, Kattakurgan)               Beruniy, Khiva)
                                                  │
                                                  ▼
                                  ┌───────────────────────────────┐
                                  │       SOUTH RDC (KARSHI)      │
                                  │ Serves: Kashkadarya &         │
                                  │         Surkhandarya (Termez) │
                                  └───────────────┬───────────────┘
                                                  │
                                          8 Local Cross-Docks
                                          (Shahrisabz, Denau, Termez)
```

### 2.1 Regional Registry & SOATO Partitioning

| Macro-Zone | Administrative Territory | SOATO Code | Regional Capital / Primary Hub | Topographical & Logistical Constraint |
|---|---|---|---|---|
| **Metropolitan** | Toshkent shahri | `1726` | Tashkent | Dense urban traffic, low-emission delivery windows, high digital card penetration. |
| **Metropolitan** | Toshkent viloyati | `1727` | Nurafshon / Chirchiq | Heavy industrial zones, agricultural sprawl, suburban feeder links. |
| **East (Fergana Valley)** | Andijon viloyati | `1703` | Andijan | Highest population density in Central Asia; micro-retailer saturation; short inter-store hops. |
| **East (Fergana Valley)** | Fargʻona viloyati | `1730` | Kokand / Fergana | Traditional craft wholesale hub (Qoʻqon ulgurji bozori); heavy Cyrillic and cash reliance. |
| **East (Fergana Valley)** | Namangan viloyati | `1714` | Namangan | Mountainous terrain (Chust, Pop); tight bazaar corridors. |
| **Central Oasis** | Samarqand viloyati | `1718` | Samarkand / Urgut | Urgut wholesale bazaar; crossroad between North, South, and West; bilingual (Uzbek/Tajik). |
| **Central Oasis** | Jizzax viloyati | `1708` | Jizzakh | Transit corridor; seasonal agricultural volume spikes; wide rural distances. |
| **Central Oasis** | Sirdaryo viloyati | `1724` | Guliston | River valley transit; high temperature in summer; agricultural distribution. |
| **Central Oasis** | Navoiy viloyati | `1712` | Navoi / Zarafshan | Mining logistics; extreme desert distances (Zarafshan, Uchkuduk: 300+ km isolated cells). |
| **South Oasis** | Qashqadaryo viloyati | `1710` | Karshi / Shahrisabz | Separated from Samarkand by Taxtaqoracha Pass; high cash & debt ledger ratio. |
| **South Oasis** | Surxondaryo viloyati | `1722` | Termez / Denau | Subtropical extreme heat (+48°C cold-chain risk); Denau trading corridor; Afghan border proximity. |
| **West Oasis** | Buxoro viloyati | `1706` | Bukhara / Gijduvan | Ancient bazaar streets requiring compact 1.5-ton vans; high tourist retail coexistence. |
| **West Oasis** | Xorazm viloyati | `1733` | Urgench / Khiva | Distinct dialect; linear river-oasis settlement pattern along Amu Darya. |
| **Northwest Sovereign** | Qoraqalpogʻiston Resp. | `1735` | Nukus / Kungrad | Sovereign territory; official Karakalpak language; vast arid distances (Moynaq: 200 km from Nukus). |

### 2.2 Topographical Bottlenecks & Automated Line-Haul Routing

1. **Kamchik Pass (Qamchiq dovoni — A-373, 2,268m Altitude)**:
   - *Physical Bottleneck:* The sole arterial link between Tashkent and the 10.5 million inhabitants of the Fergana Valley. Frequent winter blizzard closures, heavy vehicle weight-check scales, zero-overtaking avalanche galleries.
   - *Pegasus.X Invariant:* Last-mile 5-ton delivery trucks are strictly forbidden from traversing Kamchik directly to serve retail stores. All Fergana Valley demand is consolidated at CDC Tashkent into **20-ton Line-Haul Shuttles** (`CONSOLIDATED_FEEDER_SHUTTLE`), offloaded at Kokand RDC (`RDC-EAST`), and cross-docked into regional delivery fleets.
2. **Taxtaqoracha Pass (Kitob dovoni — M-39, 1,600m Altitude)**:
   - *Physical Bottleneck:* Steep switchbacks connecting Samarkand and Kashkadarya. Heavy multi-axle trucks are frequently diverted via the longer Guzar bypass.
   - *Pegasus.X Invariant:* The CVRP algorithm automatically detects vehicle axle weight. Heavy vehicles (>7.5 tons) are barred from M-39 routing and routed via the A-380/Karshi bypass.
3. **Kyzylkum Desert Corridor (A-380 Highway)**:
   - *Physical Bottleneck:* 450 km of desolate highway between Bukhara and Urgench/Nukus with ambient summer temperatures reaching +50°C.
   - *Pegasus.X Invariant:* Beverage and dairy consignments require automated telemetry-tracked reefer logging (`WarehouseTemperatureReadings`). If ambient temperature exceeds +40°C, the optimizer enforces early-morning (03:00–09:00) night shuttle runs.

---

## 3. Telecommunications Resilience & Zero-Connectivity Data Plane

Outside major metropolitan centers, 4G/LTE drops rapidly to 2G/EDGE or complete signal blackouts inside metal bazaar stalls (e.g. Abu Saxiy, Urgut, Kokand Bazaar) and mountain passes. Pegasus.X implements a 4-tier network resilience engine:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            NETWORK RESILIENCE ENGINE                        │
├─────────────────────────────────────────────────────────────────────────────┤
│  Tier 1: High-Speed 4G/LTE / Wi-Fi                                          │
│  ↳ Full JSON REST API + Real-time Bi-directional WebSocket Telemetry        │
├─────────────────────────────────────────────────────────────────────────────┤
│  Tier 2: Degraded 2G/EDGE (Latency > 1500ms, Packet Loss > 20%)             │
│  ↳ Binary Protocol (CBOR / MessagePack) + Brotli / Gzip Delta Changelogs     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Tier 3: Total Radio Blackout (Basement Stores, Remote Qishloqs)            │
│  ↳ 100% Offline SQLite + Offline 200MB OSM Vector Maps + CRDT Mutation Queue│
├─────────────────────────────────────────────────────────────────────────────┤
│  Tier 4: Critical Emergency Air-Gap Fallback                                │
│  ↳ Signed 160-Character SMS / USSD Proof-of-Delivery Gateway               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.1 100% Offline Mobile Operation (Driver, Retailer & Field Sales)

1. **Local Embedded SQLite Data Store**:
   - Every client app (Native Kotlin Android, Native Swift iOS, Expo Field Sales) embeds a local SQLite database (Room / SwiftData / Expo SQLite).
   - All critical domain entities (`orders`, `stock_lots`, `customers`, `price_tiers`, `route_manifests`) are cached locally.
2. **Offline Vector Map Rendering (Zero Cellular Map Streaming)**:
   - **Problem:** Streaming Google Maps or Mapbox vector tiles over 2G in rural Fergana or Karakalpakstan consumes excessive bandwidth, drains battery, and results in blank gray screens.
   - **Pegasus.X Solution:** Pre-bundles or downloads once over warehouse Wi-Fi an optimized **200MB OpenStreetMap (OSM) Uzbekistan Vector Tile Package** (`.mbtiles` / `.pmtiles`). Rendered locally on device via **MapLibre GL Native** at 60 FPS with zero data consumption.
3. **Pre-Computed Polyline Manifests**:
   - Morning dispatch at the distribution center downloads the full turn-by-turn route geometry and stop sequence onto the driver's device.
   - GPS dead-reckoning and navigation continue without dropping a single turn during complete network disconnection.
4. **Store-and-Forward Mutation Queue (CRDT Monotonic Sequencing)**:
   - When the driver delivers goods, collects cash, or files a UMP damage claim offline, the transaction is committed locally to `offline_mutations` with:
     - `mutation_id`: UUIDv4
     - `client_seq`: Monotonically increasing 64-bit integer
     - `occurred_at`: Hardware GPS UTC timestamp
     - `signature`: Cryptographic HMAC-SHA256 of payload signed by driver session key.
   - When 2G/3G/Wi-Fi connectivity is re-established, the background sync worker drains the queue in strict sequence order using `POST /v1/sync/mutations/drain`.
   - The backend processes mutations idempotently: `ON CONFLICT (mutation_id) DO NOTHING`.

### 3.2 Tier 4: Encrypted SMS / USSD Delivery Handshake Gateway

In remote regions where mobile packet data is completely disabled or SIM credit is exhausted, drivers and retailers can finalize deliveries via carrier SMS/USSD:

```
Driver Phone ────────> Signed 160-char SMS ────────> Pegasus.X SMPP Gateway
                      (Shortcode: 7800)             (Direct Telco Interconnect)
```

**Payload Specification (Base64-Hex Encoded, 128 chars):**
```
PXZ1:POD:[ORDER_UUID_HEX]:[STATUS_CODE]:[COLLECTED_TIYINS]:[TIMESTAMP]:[HMAC_TAG]
```
- Example: `PXZ1:POD:4a9f81bc:DELIVERED:360000000:1757112000:a7b8c9d0`
- The backend SMPP worker receives the short message from Ucell/Beeline/Mobiuz/Uztelecom, verifies the driver's phone number and HMAC signature, confirms the order, and queues the Soliq E-Factura.

---

## 4. Linguistic, Cultural & Dialect Inclusivity

Accessibility across Uzbekistan requires eliminating language, script, and literacy barriers for shopkeepers and enterprise staff.

### 4.1 Quadrilingual & Dual-Script Localization Matrix

The entire Pegasus.X platform supports 4 official language profiles:

| Language Key | Language Name | Script | Target Audience & Geographic Domain |
|---|---|---|---|
| `uz-latn` | Oʻzbek tili (Lotin) | Latin | Official state standard; youth; modern retail chains (Korzinka, Makro, Havas). |
| `uz-cyrl` | Ўзбек тили (Кирилл) | Cyrillic | Traditional trade shopkeepers, older accountants, wholesale bazaars across Fergana, Samarkand, and Surkhandarya. |
| `kaa` | Qaraqalpaq tili | Latin / Cyrillic | Republic of Karakalpakstan (Nukus, Kungrad, Chimbay); constitutional state language. |
| `ru` | Русский язык | Cyrillic | Enterprise corporate back-office, 1C accounting operators, multinational FMCG distributors. |

### 4.2 Dialect-Adaptive Voice AI Ordering (Telegram Mini App)

Small dukon owners in mahallas rarely type orders on keyboards; they communicate via Telegram voice messages. Pegasus.X incorporates an Uzbek dialect normalization engine trained on regional trade speech patterns:

```
[Storekeeper Spoken Audio] ──> [Whisper Large-v3 Turbo] ──> [Uzbek Dialect Normalizer] ──> [Deterministic Order JSON]
```

#### Regional Dialect Normalization Mappings:

```
1. Fergana Valley Dialect (Andijan, Kokand):
   Spoken:  "Akaxon, manavi 1.5 lik koʻk koladan 20 pachka bervoring, pulini dushanba kuni yopamiz"
   Parsed:  SKU: sku_pepsi_15l | Qty: 20 PACK | Terms: NET_7 (Monday settlement)

2. Tashkent Colloquial:
   Spoken:  "Brat, ertaga ertalabga oʻnta blok pepsi, ikkita qop un tashlab keting"
   Parsed:  SKU: sku_pepsi_15l (Qty: 10 PACK) + SKU: sku_flour_premium_50kg (Qty: 2 BAG)

3. Khorezmian Dialect (Urgench, Khiva):
   Spoken:  "Ogʻa, ertangʻa bizgʻa oʻn besh quti kola va besh quti sok yetkazib bering"
   Parsed:  SKU: sku_pepsi_15l (Qty: 15 BOX) + SKU: sku_juice_1l (Qty: 5 BOX)

4. Karakalpak Dialect (Nukus):
   Spoken:  "Agʻa, erteng azanda on yashik shiyrin suw jiberinʻ"
   Parsed:  SKU: sku_pepsi_15l (Qty: 10 BOX)
```

---

## 5. Regional Cash, Credit ("Nasiya") & Payment Architecture

Payment reality in Uzbekistan is starkly divided: Tashkent is heavily card/QR-centric, while the regions are dominated by cash and bilateral trade credit (*nasiya*).

### 5.1 Multi-Tender Doorstep Settlement Protocol

At the retailer doorstep, drivers can split the order total across any combination of payment tenders:

$$\text{Order Total} = \text{Cash Collected} + \text{Humo/Uzcard Card Tap} + \text{Revolving Credit (Nasiya)} + \text{Customer Deposit Balance}$$

```go
type DoorstepSplitTenderRequest struct {
    OrderID            string `json:"order_id"`
    CashTiyins         int64  `json:"cash_tiyins"`
    CardTiyins         int64  `json:"card_tiyins"`
    CardRRNumber       string `json:"card_rrn,omitempty"`
    CreditLedgerTiyins int64  `json:"credit_ledger_tiyins"` // Nasiya allocation
    DepositDeduction   int64  `json:"deposit_deduction_tiyins"`
}
```

### 5.2 SoftPOS (Smartphone NFC Card Tapping)

Instead of requiring distributors to procure, charge, and maintain $250 Pax/Verifone hardware POS terminals for thousands of drivers across the country, Pegasus.X integrates **SoftPOS NFC technology** into the driver's commodity Android smartphone:
1. Customer taps their physical **Humo** (contactless NFC) or **Uzcard** card directly against the driver's phone.
2. The encrypted EMV payload is routed directly through the **GlobalPay / Uzcard / Humo Kernel Gateway**.
3. Instant authorization is received within 800ms; zero hardware maintenance cost.

### 5.3 Central Bank 25,000,000 UZS Legal Cash Cap Enforcement

Under **Central Bank of Uzbekistan Regulation No. 3220**, cash settlements between legal entities (*YTT, MCHJ*) cannot exceed **25,000,000 UZS (2,500,000,000 tiyins)** per transaction.
- **Fail-Closed Validation Invariant**:
  ```go
  const MaxLegalCashLimitTiyins int64 = 2500000000 // 25,000,000.00 UZS

  if req.CashTiyins > MaxLegalCashLimitTiyins {
      excessTiyins := req.CashTiyins - MaxLegalCashLimitTiyins
      return ErrCashCeilingExceeded(fmt.Sprintf(
          "Central Bank Reg 3220 cash limit exceeded: %d tiyins. Excess (%d tiyins) must be settled via Card, Bank Wire, or Nasiya credit",
          req.CashTiyins, excessTiyins,
      ))
  }
  ```

### 5.4 Nationwide Regional Bank Payouts via GlobalPay OCT

Field sales agents and delivery drivers operate across different regional banking partners. The Pegasus.X payroll engine dispatches instant salary advances and commission payouts directly to any card scheme across all Uzbek banks:
- **Agrobank & Xalq Banki**: Ubiquitous in rural districts, cotton sectors, and mahallas.
- **Hamkorbank**: Heavy presence and market dominance in the Fergana Valley.
- **Ipak Yuli Bank & NBU**: Commercial and trade finance centers in Samarkand and Bukhara.
- **Kapitalbank**: High-velocity card processing in urban centers.

---

## 6. Regulatory & Fiscal Turnkey Integration

Nationwide commercial operations require strict adherence to Uzbekistan's digital regulatory bodies.

```
                          ┌───────────────────────────────┐
                          │     PEGASUS.X FISCAL CORE     │
                          └───────────────┬───────────────┘
                                          │
                  ┌───────────────────────┴───────────────────────┐
                  ▼                                               ▼
  ┌───────────────────────────────┐               ┌───────────────────────────────┐
  │     SOLIQ.UZ E-FACTURA        │               │   ASL BELGISI (CRPT TURON)    │
  │  - 17-digit MXIK (IKPU)       │               │  - Decree No. 833 / No. 631   │
  │  - E-IMZO PKCS#7 Digital Seal │               │  - Water & Soft Drink Track   │
  │  - 12% VAT Auto-Computation   │               │  - SSCC-18 Pallet Aggregation │
  │  - Corrective Credit Invoices │               │  - GS1 DataMatrix Verification│
  └───────────────────────────────┘               └───────────────────────────────┘
```

### 6.1 State Tax Committee (Soliq.uz) Integration
1. **MXIK (IKPU) Product Classification**:
   - Every product record in Pegasus.X is bound to an official 17-digit MXIK code (e.g. `02202001001000000` for Non-alcoholic beverages). Missing MXIK codes fail closed and block dispatch.
2. **E-IMZO Electronic Digital Signature (EDS)**:
   - Invoices and shipping documents are cryptographically signed using official **E-IMZO** keys (National Certification Center).
   - Signatures are stamped with signer PINFL (14 digits) and legal STIR (9 digits).
3. **Automated Corrective Invoices (*Tuzatilgan hisob-faktura*)**:
   - When a doorstep UMP discrepancy occurs (e.g., 2 broken bottles rejected by retailer), Pegasus.X automatically generates a linked corrective tax invoice reducing VAT and gross total without manual accountant intervention.

### 6.2 Asl Belgisi (CRPT Turon) Mandatory Serialization
Under Cabinet of Ministers Resolution No. 631 and Decree No. 833:
1. **Water and Soft Drinks** are legally subject to digital marking across Uzbekistan.
2. **Pallet Hierarchy**:
   - Master Pallet: **SSCC-18** barcode (`004780001234567892`).
   - Secondary Carton: GS1-128 carton code.
   - Consumer Unit: GS1 DataMatrix on bottle cap.
3. Pegasus.X records inbound pallet disaggregation (*dock break-bulk*) so that partial pallet shipments to rural stores remain fully compliant with CRPT Turon inventory trace rules.

---

## 7. Field Sales & Traditional Trade ("Bozor & Mahalla Dukon") Client

Because rural and suburban distribution is driven by Field Sales Agents (*savdo agentlari*), Pegasus.X provides the dedicated `apps/field-sales-mobile` client:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    FIELD SALES MOBILE WORKFLOW (OFFLINE)                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ 1. Morning Sync (at Depot Wi-Fi):                                          │
│    ↳ Downloads daily route: 35 retail stores in Urgut District              │
│    ↳ Caches full SKU catalog + store-specific credit limits + offline tiles │
├─────────────────────────────────────────────────────────────────────────────┤
│ 2. Geofenced Bazaar Visit:                                                  │
│    ↳ Agent arrives at "Bahor Mahalla Dukon"                                 │
│    ↳ GPS geofence (<50m) validates presence (prevents fraudulent check-ins)│
├─────────────────────────────────────────────────────────────────────────────┤
│ 3. Offline Order Capture & Shelf Audit:                                     │
│    ↳ Scans store shelves; logs competitor facing counts & retail prices     │
│    ↳ Inputs storekeeper replenishment order                                 │
│    ↳ Checks bilateral credit headroom in local SQLite                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ 4. Digital Signature / Audio Handshake:                                     │
│    ↳ Storekeeper signs on screen or leaves voice confirmation               │
│    ↳ Order saved to local outbox; syncs automatically when network appears  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Nationwide Architectural Implementation Plan for Pegasus.X

To embed full national accessibility into the `/Users/shakhzod/Desktop/pegasus.x` repository, the following architectural modules are established:

```
pegasus.x/
├── backend/internal/
│   ├── regional/               <-- NEW: Regional topology & SOATO code manager
│   │   ├── soato.go            // 14 regions, 208 district codes, population weights
│   │   ├── passes.go           // Mountain pass operational gates (Kamchik, Taxtaqoracha)
│   │   └── linehaul.go         // CDC-to-RDC feeder shuttle economics & scheduling
│   ├── offline/                <-- NEW: Monotonic CRDT sync & SMS fallback engine
│   │   ├── sync.go             // Monotonic sequence validator & mutation drainer
│   │   └── smpp.go             // Cellular SMS/USSD emergency POD parser
│   ├── speech/                 <-- NEW: Uzbek regional dialect normalization
│   │   ├── whisper.go          // Whisper ASR transcription gateway
│   │   └── dialects.go         // Tashkent, Fergana, Khorezm, Karakalpak phonetic lexicons
│   ├── softpos/                <-- NEW: NFC smartphone card tap driver module
│   │   └── emv.go              // Humo/Uzcard contactless EMV kernel driver
│   └── soliq/                  // Existing: MySoliq E-Factura & MXIK classifier
│       └── eimzo.go            // E-IMZO cryptographic legal signing
├── apps/
│   ├── field-sales-mobile/     // Existing: React Native / Expo offline sales agent app
│   ├── retailer-telegram-miniapp/ // Existing: TMA with voice note order ingestion
│   └── driver-app-android/     // Native Kotlin: SoftPOS NFC tap + offline OSM vector tiles
└── contracts/
    └── regional_types.ts       // Shared regional definitions, SOATO enums & SMS specs
```

### 8.1 Database Schema Extensions (`database/migrations/`)

```sql
-- 1. National Administrative Regions (SOATO)
CREATE TABLE IF NOT EXISTS regional_zones (
    soato_code VARCHAR(10) PRIMARY KEY,
    zone_name VARCHAR(100) NOT NULL,
    macro_zone VARCHAR(50) NOT NULL, -- 'METROPOLITAN', 'EAST_FERGANA', 'CENTRAL_OASIS', 'SOUTH_OASIS', 'WEST_OASIS', 'KARAKALPAKSTAN'
    primary_rdc_id VARCHAR(36) NOT NULL,
    mountain_pass_restricted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Offline Store-and-Forward Mutation Queue Ledger
CREATE TABLE IF NOT EXISTS offline_mutation_ledger (
    mutation_id UUID PRIMARY KEY,
    client_id VARCHAR(100) NOT NULL,
    client_seq BIGINT NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- 'ORDER', 'DELIVERY_POD', 'UMP_CLAIM', 'CASH_RECEIPT'
    entity_id VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    hardware_gps_lat DOUBLE PRECISION,
    hardware_gps_lng DOUBLE PRECISION,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    signature_hmac VARCHAR(128) NOT NULL,
    CONSTRAINT uq_client_sequence UNIQUE (client_id, client_seq)
);

-- 3. SMS Emergency Proof-of-Delivery Audit Log
CREATE TABLE IF NOT EXISTS sms_emergency_pod_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_phone VARCHAR(20) NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    status_code VARCHAR(30) NOT NULL,
    collected_tiyins BIGINT NOT NULL,
    hmac_verified BOOLEAN NOT NULL,
    raw_payload VARCHAR(160) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

---

## 9. Verification & Quality Gates for Nationwide Accessibility

Any feature claiming national accessibility in Pegasus.X must pass these 5 non-negotiable gates:

1. **Mountain Pass Weight Gate**: CVRP planner fails closed if a truck >7.5 tons is routed directly through Taxtaqoracha or a 5-ton van is routed across Kamchik without line-haul shuttle aggregation.
2. **2G Low-Bandwidth Gate**: Catalog sync and checkout must complete within <15KB payload size using Brotli delta compression.
3. **100% Offline Delivery Gate**: A driver must be able to launch the app in airplane mode, navigate using local OSM vector tiles, record a doorstep delivery with cash/card/credit split, and store the mutation in SQLite without crashing or hanging.
4. **CBU Cash Limit Gate**: Any transaction with `cash_tiyins > 2,500,000,000` must return HTTP 422 with a Central Bank Reg 3220 statutory violation message.
5. **Quadrilingual UI & Audio Gate**: All portal and mobile screens must render cleanly in `uz-latn`, `uz-cyrl`, `kaa`, and `ru`. Voice ordering must correctly parse Fergana, Tashkent, Khorezm, and Karakalpak audio clips into deterministic SKU line items.

---
*Blueprint formulated for Pegasus.X national deployment across the Republic of Uzbekistan.*
