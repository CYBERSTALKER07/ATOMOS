# Pegasus Patent Line-Art Prompt Library For Nano Banana

**Назначение**
Этот документ является исходным prompt-pack для генерации патентных иллюстраций Pegasus в Nano Banana. Все фигуры должны быть черно-белыми техническими схемами, а не цветными визуалами, UI-скриншотами или маркетинговыми диаграммами.

## 1. Global Prompt Prefix

Use this exact prefix at the start of every image prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration
```

Use this exact negative prompt for every image:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

## 2. Export Contract

1. Primary output: SVG.
1. Filing output: PDF generated from the SVG without rasterizing text or arrows.
1. Preview output: PNG 4K, 3840 x 2160 minimum, pure black on white.
1. Labels: uppercase only, short technical nouns, no prose paragraphs inside the figure.
1. Structure: module boxes, boundary boxes, swimlanes, callout numerals, arrows, and small formula panels only.
1. Prohibition: no source code excerpts, no endpoint-level private maps, no secrets, no deployment sizing, no real UI screenshot reconstruction.

## 3. Priority Batches

1. Patent-core first: FIG. 01, 04, 05, 06, 13, 14, 16, 18, 20, 22, 26, 28, 30, 38, 40.
1. Role and UI plates second: FIG. 07-12 and FIG. 31-36.
1. Supporting control and math plates third: FIG. 02, 03, 15, 17, 19, 21, 23-25, 27, 29, 37, 39.

## 4. Nano Banana Figure Prompts

### FIG. 01 - FULL PEGASUS INFRASTRUCTURE ENVELOPE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 01 FULL PEGASUS INFRASTRUCTURE ENVELOPE. Draw a left-to-right system envelope showing ROLE CLIENTS entering PROTECTED ROUTING, then BACKEND HANDLERS, LIVE HUBS, SPANNER DATA CORE, REDIS CACHE BUS, KAFKA OUTBOX, WORKERS, FINANCE, and OBSERVABILITY. Use nested boundary boxes for CLIENT SURFACES, LOAD BOUNDARY, SERVICE CORE, DATA PLANE, EVENT PLANE, and CONTROL PLANE. Show arrows for REQUEST FLOW, REALTIME FLOW, OUTBOX FLOW, CACHE INVALIDATION, FINANCIAL RECONCILIATION, and TELEMETRY. Labels must be uppercase and concise: SUPPLIER, RETAILER, DRIVER, PAYLOAD, FACTORY, WAREHOUSE, ROUTER, HANDLERS, HUBS, SPANNER, REDIS, OUTBOX, KAFKA, AI WORKER, RECONCILER, METRICS. Keep it abstract and patent-safe, no private endpoint map.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 02 - MAGLEV STATELESS INGRESS PLATE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 02 MAGLEV STATELESS INGRESS PLATE. Show WEB, DESKTOP, MOBILE, and TERMINAL CLIENTS entering a LOAD BOUNDARY labeled MAGLEV CONSISTENT HASHING. Inside the boundary, draw HASH KEY NORMALIZATION, LOOKUP TABLE, HEALTH FILTER, and STATELESS POD SELECTION. To the right, draw several identical STATELESS BACKEND SERVICE PODS feeding ROUTE FAMILIES. Add small callouts for NO STICKY SESSION, DRAIN SAFE, REBALANCE SAFE, and SAME REQUEST CONTRACT. Include route-family boxes at a grouped level only: AUTH, SUPPLIER, RETAILER, DRIVER, PAYLOAD, FACTORY, WAREHOUSE, ORDER, PAYMENT, TELEMETRY.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 03 - ROUTE-COMPOSITION CONTRACT GRAPH

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 03 ROUTE-COMPOSITION CONTRACT GRAPH. Draw a grouped backend route-composition graph, not endpoint-level maps. Center a CHI ROUTER box. Around it place domain route-family boxes: AUTH ROUTES, ORDER ROUTES, SUPPLIER CORE, SUPPLIER CATALOG, SUPPLIER LOGISTICS, SUPPLIER PLANNING, SUPPLIER OPERATIONS, SUPPLIER INSIGHTS, RETAILER ROUTES, DRIVER ROUTES, PAYLOAD ROUTES, FACTORY ROUTES, WAREHOUSE ROUTES, PAYMENT ROUTES, WEBHOOK ROUTES, TELEMETRY ROUTES, INFRA ROUTES. Show each route family pointing to DOMAIN HANDLERS, then to SERVICE LAYER, REPOSITORY LAYER, and EVENT LAYER. Include a small DEPS CONTRACT callout with NARROW FIELDS, MIDDLEWARE, NO COMPOSITION LEAKAGE.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 04 - ROLE-SURFACE MATRIX

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 04 ROLE-SURFACE MATRIX. Draw a matrix with rows SUPPLIER, RETAILER, DRIVER, PAYLOAD, FACTORY ADMIN, WAREHOUSE ADMIN and columns BACKEND CONTRACT, WEB DESKTOP, ANDROID, IOS, TERMINAL, REALTIME CHANNEL. Fill each intersection with simple outline cells and short labels showing available surface classes. Add a right-side SYNC RULE bracket: SHARED TYPE, API CLIENT, VIEW MODEL, UI SURFACE, REALTIME, OFFLINE, VERSION GATE. Use check marks as plain line symbols only, not decorative icons.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 05 - ROLE RESPONSIBILITY AND HANDOFF MAP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 05 ROLE RESPONSIBILITY AND HANDOFF MAP. Draw a circular or looped handoff map: SUPPLIER CONTROL, FACTORY PRODUCTION, WAREHOUSE STOCK, PAYLOAD LOADING, DRIVER EXECUTION, RETAILER RECEIPT, SUPPLIER FEEDBACK. Each role is a module box with handoff arrows between boxes. Add small guard boxes on arrows: SCOPE, MANIFEST, SCAN, GEOFENCE, RECEIPT, ANALYTICS. Include a central AUDIT AND EVENT SPINE box receiving dotted arrows from every handoff.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 06 - FULL ECOSYSTEM WORKFLOW LOOP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 06 FULL ECOSYSTEM WORKFLOW LOOP. Draw a closed operational loop with these ordered modules: PLANNING, ORDERING, STOCK PRESSURE, REPLENISHMENT, LOADING, ROUTE EXECUTION, RECEIPT, EXCEPTION, ANALYTICS, NEXT DEMAND CYCLE. Show the loop as a clean technical ring with arrows. Put SPANNER STATE, OUTBOX EVENTS, REALTIME CHANNELS, and LEDGER as four horizontal rails crossing underneath the loop. Add feedback arrows from EXCEPTION and ANALYTICS back to PLANNING.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 07 - SUPPLIER CONTROL-PLANE CHAIN

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 07 SUPPLIER CONTROL-PLANE CHAIN. Draw a sequential supplier operations chain: ONBOARDING, BILLING, CATALOG, PRICING, INVENTORY, FLEET, WAREHOUSES, FACTORIES, DISPATCH, MANIFESTS, ANALYTICS, TREASURY. Use a wide horizontal plate with grouped bands: SETUP, COMMERCIAL, OPERATIONS, CONTROL, FINANCE. Show each band connected to SUPPLIER SCOPE and AUDIT LOG rails. Make it an abstract flow, not a portal screenshot.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 08 - RETAILER COMMERCE CAPTURE FLOW

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 08 RETAILER COMMERCE CAPTURE FLOW. Draw a retailer flow from DISCOVERY to CATALOG, CART, CHECKOUT, PAYMENT, ORDER TRACKING, RECEIPT CONFIRMATION, SAVED CARDS, AUTO ORDER. Add guard modules for SUPPLIER SCOPE, PAYMENT STATE, IDEMPOTENCY KEY, TRACKING CHANNEL, and DEMAND FEEDBACK. Place RETAILER CLIENTS on the left and ORDER CORE on the right.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 09 - DRIVER EXECUTION SPINE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 09 DRIVER EXECUTION SPINE. Draw a vertical execution spine: LOGIN, HOME-NODE SCOPE, MISSION SUMMARY, ROUTE MAP, QR SCAN, ARRIVAL VALIDATION, OFFLOAD, PAYMENT OR CASH, CORRECTION, COMPLETION. Add side guard boxes: IDEMPOTENCY, GEOFENCE, MANIFEST, TELEMETRY, RECEIPT, AUDIT. Show optimized route as default and manual next-stop selection as a controlled side branch.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 10 - PAYLOAD LOADING WORKSPACE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 10 PAYLOAD LOADING WORKSPACE. Draw an abstract loading workspace flow: TRUCK SELECTION, ORDER LIST, CHECKLIST GATE, MARK LOADED, EXCEPTION PATH, SEAL ACTION, DISPATCH SUCCESS, SYNC EVENT. Include manifest state boxes for DRAFT, LOADING, SEALED, DISPATCHED. Add side rail for PAYLOAD REALTIME SYNC and SUPPLIER VISIBILITY.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 11 - FACTORY REPLENISHMENT LOOP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 11 FACTORY REPLENISHMENT LOOP. Draw a factory replenishment loop: THRESHOLD SIGNAL, SUPPLY REQUEST, ACCEPTANCE, PRODUCTION READINESS, TRANSFER MANIFEST, LOADING BAY, HANDOFF, WAREHOUSE RECEIPT. Show FACTORY ADMIN and WAREHOUSE ADMIN as scoped operator boxes. Add rails for STOCK STATE, TRANSFER STATE, LIVE UPDATE, and AUDIT EVENT.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 12 - WAREHOUSE DISPATCH AND REPLENISHMENT LOOP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 12 WAREHOUSE DISPATCH AND REPLENISHMENT LOOP. Draw warehouse control loop modules: STOCK POSTURE, FORECAST, SUPPLY REQUEST, DISPATCH PREVIEW, DISPATCH LOCK, VEHICLE AVAILABILITY, DRIVER AVAILABILITY, LIVE CHANNEL. Show two outgoing branches: REPLENISHMENT TO FACTORY and ROUTE DISPATCH TO DRIVER. Include stale/offline live-channel state as a small signal box.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 13 - GOVERNED STATE-TRANSITION PATH

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 13 GOVERNED STATE-TRANSITION PATH. Draw a single mutation traveling through INTENT, IDENTITY, SCOPE, OWNERSHIP, POLICY, IDEMPOTENCY, VERSION, TRANSACTION, OUTBOX, CACHE INVALIDATION, REALTIME, AUDIT. Use a strict linear chain with each gate as a boxed checkpoint. Add reject arrows from SCOPE, POLICY, IDEMPOTENCY, and VERSION to labeled terminal boxes: FORBIDDEN, CONFLICT, REPLAY, STALE.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 14 - TRANSACTIONAL OUTBOX RELAY

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 14 TRANSACTIONAL OUTBOX RELAY. Draw an atomic transaction boundary containing DOMAIN ROW WRITE and OUTBOX ROW WRITE. From COMMIT, show RELAY POLLING, KAFKA PUBLISH, WORKER CONSUMPTION, NOTIFICATION FORMATTING, REALTIME DELIVERY. Use one thick boundary around the transaction and dashed arrows for polling. Add small guarantee labels: ATOMIC STATE EVENT, AGGREGATE KEY ORDER, RETRY SAFE, PUBLISHED MARK.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 15 - ORDER REASSIGNMENT EVENT CASCADE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 15 ORDER REASSIGNMENT EVENT CASCADE. Draw a cascade beginning with REASSIGNMENT INTENT, then FREEZE LOCK CHECK, DISPATCH LOCK CHECK, TRANSACTION, OUTBOX EVENT, KAFKA TOPIC, NOTIFICATION CONSUMER, WEBSOCKET BROADCAST, CLIENT REFRESH. Include rejection branches for LOCKED ENTITY, OUT OF SCOPE, and VERSION CONFLICT. Keep event names generic and patent-level.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 16 - H3 GEOGRAPHIC PLANNING

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 16 H3 GEOGRAPHIC PLANNING. Draw coordinates entering COORDINATE NORMALIZATION, then H3 CELL DERIVATION, NEIGHBOR RINGS, WAREHOUSE COVERAGE CELLS, RETAILER DEMAND CELLS, REGIONAL PLANNING GRAPH. Use abstract hexagon outlines only, black strokes, no filled map. Show warehouse cells and retailer cells as different line patterns, not different colors. Include arrows for COVERAGE CHECK, DEMAND AGGREGATION, ROUTE ELIGIBILITY.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 17 - WAREHOUSE LOAD PENALTY CURVE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 17 WAREHOUSE LOAD PENALTY CURVE. Draw a clean Cartesian chart with X-axis UTILIZATION and Y-axis RECOMMENDATION PENALTY. Show a gentle low-slope region below threshold, a knee at SATURATION THRESHOLD, and a steep quadratic high-penalty region near capacity. Add small callout boxes: AVAILABLE HEADROOM, CAUTION BAND, SATURATION BAND, ROUTE AVOIDANCE. Use only black lines and labels, no fills.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 18 - DISPATCH CLUSTERING AND BIN-PACKING

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 18 DISPATCH CLUSTERING AND BIN-PACKING. Draw a pipeline: ELIGIBLE ORDERS, H3 CLUSTER, VEHICLE CAPACITY FIT, SPLIT OVERSIZED MANIFEST, ROUTE EVENT, MANIFEST EVENT. Show order dots grouped into outline hex cells, then packed into vehicle capacity boxes. Add a split branch for OVERSIZED CLUSTER and a commit branch for ROUTE MANIFEST CREATION. Use abstract math panels for CAPACITY, PROXIMITY, COHESION.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 19 - OPTIMIZER HANDOFF AND FALLBACK

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 19 OPTIMIZER HANDOFF AND FALLBACK. Draw PLANNING SERVICE sending candidate work to OPTIMIZATION WORKER. Worker returns ROUTE SEQUENCE and SCORED ASSIGNMENT. Add failure branch from WORKER UNAVAILABLE to DETERMINISTIC FALLBACK, then to SAFE ROUTE SEQUENCE. Include guard boxes: TIMEOUT, CIRCUIT BREAKER, STATIC HEURISTIC, AUDIT OF FALLBACK. Show fallback as first-class, not error-only.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 20 - FREEZE-LOCK HUMAN AUTOMATION CONSENSUS

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 20 FREEZE-LOCK HUMAN AUTOMATION CONSENSUS. Draw OPERATOR ACTION acquiring FREEZE LOCK. Show AI WORK QUEUE removing affected entities, OPERATOR MUTATION completing, AUDIT WRITTEN, LOCK RELEASED, AUTOMATION RESUMES. Place HUMAN CONTROL LANE above AUTOMATION LANE with a shared LOCK REGISTRY between them. Add blocked arrows from AI QUEUE to locked entities while lock is active.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 21 - HOME-NODE PRINCIPLE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 21 HOME-NODE PRINCIPLE. Draw DRIVERS and VEHICLES bound to either WAREHOUSE HOME NODE or FACTORY HOME NODE. Show normal allowed movement inside each home-node boundary. Between boundaries draw INTER-HUB MANIFEST as the only crossing authority. Add rejection branch: NO MANIFEST, NO CROSSING. Include identity labels HOME NODE TYPE, HOME NODE ID, LEGACY WAREHOUSE COMPATIBILITY at abstract level only.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 22 - PAYMENT SETTLEMENT AND DOUBLE-ENTRY LEDGER

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 22 PAYMENT SETTLEMENT AND DOUBLE-ENTRY LEDGER. Draw ORDER STATE flowing to GATEWAY AUTH, CAPTURE, SETTLEMENT, then PAIRED LEDGER ENTRIES. Show ledger pairs as DEBIT and CREDIT rows inside one transaction boundary. Connect to SUPPLIER WALLET, RETAILER LIABILITY, PLATFORM FEE, GATEWAY CLEARING, and RECONCILIATION. Add invariant callout: SUM PER CURRENCY EQUALS ZERO. Use ledger table outline only, no real account numbers.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 23 - PAYMENT WEBHOOK TRUST BOUNDARY

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 23 PAYMENT WEBHOOK TRUST BOUNDARY. Draw GATEWAY CALLBACK crossing an EXTERNAL TRUST BOUNDARY. First internal gate is SIGNATURE FIRST VERIFICATION, followed by IDEMPOTENCY, TYPED PARSING, TRANSACTION UPDATE, OUTBOX, LEDGER, RESPONSE. Add rejection path directly from SIGNATURE VERIFY to REJECTED CALLBACK. Use bold boundary line around untrusted input and internal settlement core.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 24 - CASH COLLECTION BRANCH

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 24 CASH COLLECTION BRANCH. Draw DRIVER CASH CONFIRMATION entering REPLAY SAFE MUTATION, then LEDGER POSTING, SUPPLIER CREDIT, PLATFORM FEE, ROUTE COMPLETION, RECEIPT CONFIRMATION. Add branch for CASH DISCREPANCY leading to CORRECTION QUEUE and AUDIT EVENT. Show cash path as controlled alternative to gateway settlement, with same idempotency and ledger rails.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 25 - OFFLINE PROOF AND CORRECTION PATH

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 25 OFFLINE PROOF AND CORRECTION PATH. Draw OFFLINE DELIVERY PROOF stored in LOCAL BUFFER, then SYNC, CONFLICT GATE, DEDUPE, CORRECTION, REFUND DELTA, CANONICAL AMENDMENT. Include retry arrows from SYNC FAILURE back to LOCAL BUFFER. Add guard boxes: DEVICE TIME, BODY HASH, VERSION CHECK, AMENDMENT AUDIT. Keep it mobile-abstract, no phone UI screenshot.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 26 - WEBSOCKET CROSS-SERVICE RELAY

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 26 WEBSOCKET CROSS-SERVICE RELAY. Draw AUTHENTICATED SOCKET entering ROOM ASSIGNMENT. From one service pod, show LOCAL FANOUT and REDIS PUBSUB. Then show PEER POD FANOUT to other subscribed clients. Add HEARTBEAT, RECONNECT, and FAIL OPEN LOCAL DELIVERY as side modules. Include a degraded path where REDIS PUBSUB FAILURE does not block LOCAL FANOUT. Use role-scoped room labels only.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 27 - TELEMETRY GEOFENCE SIGNAL LOOP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 27 TELEMETRY GEOFENCE SIGNAL LOOP. Draw DRIVER COORDINATES flowing to ROUTE PROGRESS, FLEET CHANNEL, SUPPLIER MAP, PROXIMITY EVENT, GUARDED COMPLETION VALIDATION. Show planned route and actual route as two black line styles. Add GEOFENCE RING around retailer destination with VALID COMPLETION and REJECT OUTSIDE RADIUS branches. Include stale telemetry branch to STALE STATE.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 28 - PRIORITY GUARD AND BACKPRESSURE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 28 PRIORITY GUARD AND BACKPRESSURE. Draw incoming traffic tiers: AUTH, PAYMENT, DISPATCH, READS, BULK, BACKGROUND. Feed all tiers into QUEUE PRESSURE SENSOR and PRIORITY GUARD. Show protected lanes for AUTH, PAYMENT, DISPATCH. Show low-priority shedding with RETRY AFTER SIGNAL. Add TOKEN BUCKET and CIRCUIT BREAKER side guards. Use arrows to show fail-fast protection of data core.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 29 - CIRCUIT BREAKER STATE MACHINE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 29 CIRCUIT BREAKER STATE MACHINE. Draw three large state boxes: CLOSED, OPEN, HALF OPEN. Add transition arrows: FAILURE THRESHOLD from CLOSED to OPEN, FAST FAIL while OPEN, PROBE from OPEN to HALF OPEN, RECOVERY to CLOSED, PROBE FAILURE back to OPEN. Add METRICS and ALERTS as observer boxes connected to every state. Use simple finite-state-machine layout.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 30 - IDEMPOTENCY REPLAY GATE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 30 IDEMPOTENCY REPLAY GATE. Draw REQUEST KEY and BODY HASH entering IDEMPOTENCY STORE. First path: NO RECORD leads to MUTATION and FIRST RESPONSE STORE. Second path: SAME BODY REPLAY leads to STORED RESPONSE. Third path: DIFFERENT BODY REPLAY leads to CONFLICT. Add separate retention labels for API WINDOW and WEBHOOK WINDOW without exact sensitive policy values. Use three clean branches from a central decision diamond.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 31 - BENTO DASHBOARD MOSAIC

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 31 BENTO DASHBOARD MOSAIC. Draw an abstract dashboard layout as patent wireframe, not a UI screenshot. Show grid cells labeled ANCHOR, STAT, LIST, CONTROL, WIDE, FULL. Include SKELETON LOADING variant as dashed cell outlines, STALE STATE marker, and DRILL DOWN ARROWS from cells to DETAIL DRAWER. Use dense grid geometry and reference numerals for cell classes.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 32 - MATERIAL NATIVE UI CONSISTENCY MAP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 32 MATERIAL NATIVE UI CONSISTENCY MAP. Draw platform consistency map with columns WEB, ANDROID, IOS, PAYLOAD TERMINAL. Under WEB and ANDROID show MATERIAL 3 TOKENS. Under IOS show SWIFTUI NATIVE PATTERNS. Under PAYLOAD TERMINAL show OPERATIONAL DENSITY. Center a SHARED SEMANTIC TOKENS box feeding all columns. Add rows for STATE, TYPOGRAPHY, ACTIONS, FEEDBACK, OFFLINE, RESTRICTED.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 33 - SUPPLIER PAYMENT GATEWAY CONFIGURATION SURFACE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 33 SUPPLIER PAYMENT GATEWAY CONFIGURATION SURFACE. Draw abstract supplier payment configuration modules: PROVIDER CARDS, ONBOARDING STATUS, MANUAL SETUP, UPDATE, DEACTIVATE, CREDENTIAL GOVERNANCE, CHECKOUT READINESS. Add a secure boundary around CREDENTIAL STORAGE and a downstream arrow to RETAIL CHECKOUT ENABLEMENT. Do not draw real provider logos or credential fields.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 34 - RETAIL CHECKOUT ORDER SPLIT

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 34 RETAIL CHECKOUT ORDER SPLIT. Draw COMMERCE INTENT flowing to PAYMENT SELECTION, SUPPLIER SCOPED ORDER CREATION, IDEMPOTENT PLACEMENT, SETTLEMENT PREPARATION, TRACKING. Show split branches for CARD PAYMENT and CASH PAYMENT joining at ORDER CONFIRMATION. Add supplier boundary around each order grouping and ledger preparation rail below.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 35 - DRIVER MAP EXECUTION COCKPIT

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 35 DRIVER MAP EXECUTION COCKPIT. Draw a patent wireframe of an execution cockpit, not a screenshot. Include LIVE ROUTE MAP, MISSION MARKERS, SELECTED MISSION PANEL, FOCUS CONTROLS, SCAN ACTION, CORRECTION ACTION, PAYMENT BRANCH, GEOFENCE CONTEXT. Use outline panels and arrows from map markers to mission detail. Include offline and stale telemetry states as small status boxes.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 36 - PAYLOAD DISPATCH SUCCESS EVIDENCE

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 36 PAYLOAD DISPATCH SUCCESS EVIDENCE. Draw evidence modules after sealing: SEALED MANIFEST, ACTIVE TRUCK, SECURED STATE, DISPATCH CODE, RESET TO NEW MANIFEST, MACHINE READABLE RELEASE. Show DISPATCH CODE as abstract barcode-like line block without real data. Connect evidence to AUDIT LOG, SUPPLIER VISIBILITY, and DRIVER RELEASE.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 37 - REPLENISHMENT THRESHOLD LOOK-AHEAD LOOP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 37 REPLENISHMENT THRESHOLD LOOK-AHEAD LOOP. Draw SAFETY STOCK and CURRENT STOCK feeding BREACH DETECTION. From breach, show REPLENISHMENT LOCK, PULL MATRIX, PREDICTIVE PUSH, SUPPLIER NOTIFICATION, WAREHOUSE RESPONSE, FACTORY RESPONSE. Add look-ahead window as a small timeline panel and feedback arrow back to threshold tuning. Keep formulas abstract and concise.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 38 - SECURITY SCOPE ENFORCEMENT MATRIX

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 38 SECURITY SCOPE ENFORCEMENT MATRIX. Draw matrix rows JWT ROLE, SUPPLIER SCOPE, WAREHOUSE SCOPE, FACTORY SCOPE, HOME NODE SCOPE, ALLOWED ROUTE FAMILIES, REJECTED BODY OVERRIDES, SECURITY LOG. Columns are SUPPLIER, RETAILER, DRIVER, PAYLOAD, FACTORY ADMIN, WAREHOUSE ADMIN. Add arrows from BODY OVERRIDE to REJECTED and from RESOLVED CLAIMS to ALLOWED ACTION. Keep route families grouped only.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 39 - VISUAL ASSET PRODUCTION MAP

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 39 VISUAL ASSET PRODUCTION MAP. Draw an asset generation workflow with source lanes: ARCHITECTURE VISUALS, ROLE DIAGRAMS, WORKFLOW DIAGRAMS, TECH STACK COMPOSITES, MAGLEV, RELIABILITY, AUTO DISPATCH, PEGASUS IDENTITY ASSETS. Feed all lanes into PROMPT NORMALIZATION, LINE ART GENERATION, SVG REVIEW, PDF EXPORT, PNG 4K PREVIEW, FILING PACKAGE. Add QA gates: BLACK ONLY, LABEL LEGIBILITY, NO UI SCREENSHOT, NO SECRET DISCLOSURE.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

### FIG. 40 - CONTROLLED DISCLOSURE BOUNDARY

Nano Banana prompt:

```text
Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, orthographic technical plate, publication-ready legal illustration. FIG. 40 CONTROLLED DISCLOSURE BOUNDARY. Draw a two-column boundary plate. Left column DISCLOSED contains ARCHITECTURE, FORMULAS, ROLE BEHAVIOR, STATE TRANSITIONS, EVENT CATEGORIES, SECURITY PRINCIPLES. Right column WITHHELD contains SOURCE CODE, PRIVATE SCHEMAS, ENDPOINT MAPS, SECRETS, EXACT CONSTANTS, DEPLOYMENT SIZING. Put a thick vertical LEGAL DISCLOSURE BOUNDARY between columns. Add arrows from PATENT FIGURES to DISCLOSED and blocked arrows to WITHHELD.
```

Negative prompt:

```text
no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion
```

Export: SVG primary, PDF filing copy, PNG 4K preview.

## 5. Editorial Checklist Before Generation

1. Confirm the full global prompt prefix appears at the start of the active prompt.
1. Confirm the negative prompt is attached unchanged.
1. Confirm the requested output is SVG first, then PDF, then PNG 4K.
1. Confirm no figure requests color, grayscale, gradients, shadows, textures, screenshot realism, or UI screenshots.
1. Confirm every label in the image is uppercase and short.
1. Confirm private implementation material is abstracted: no source code, no private endpoint maps, no secrets, no deployment sizing, no real schema dump.
1. Confirm arrows preserve causality and state direction.
1. Confirm formulas or curves are legible without explanatory prose.

## 6. Reference Sources Used For Regeneration

1. `pegasus/docs/assets/architecture-overview.svg` as an information reference only; remove all glass, gradients, shadows, color, and marketing style.
1. `pegasus/docs/assets/maglev-load-balancers.svg` as an information reference only; convert to strict stateless ingress patent plate.
1. `pegasus/docs/assets/autodispatch-pipeline.svg` as an information reference only; convert to dispatch clustering, fallback, lock, and route-event patent plates.
1. `pegasus/docs/assets/reliability-control-plane.svg` as an information reference only; convert to outbox, priority guard, circuit breaker, WebSocket relay, and idempotency patent plates.
1. `pegasus/assets/diagrams/*.svg` as role/workflow references only; regenerate all figures in one consistent black-and-white legal line-art style.