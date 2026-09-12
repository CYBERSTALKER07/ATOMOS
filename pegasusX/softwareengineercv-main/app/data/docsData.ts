export type DocCalloutType = 'important' | 'warning' | 'tip' | 'note';

export type DocCallout = {
  type: DocCalloutType;
  title: string;
  content: string;
};

export type DocHighlight = {
  title: string;
  desc: string;
  metric?: string;
};

export type DocStep = {
  stepNumber: string;
  title: string;
  desc: string;
  code?: {
    lang: string;
    code: string;
    filename?: string;
  };
  checklist?: string[];
};

export type DocCodeExample = {
  title: string;
  lang: string;
  code: string;
  filename?: string;
  description?: string;
};

export type DocChecklistItem = {
  label: string;
  detail: string;
};

export type DocRelatedTopic = {
  title: string;
  category: string;
  categoryId: string;
  slug: string;
  reason: string;
  badge: 'PREREQUISITE' | 'NEXT STEP' | 'RELATED ENGINE' | 'EDGE CASE' | 'PLAYBOOK';
};

export type DocArticle = {
  slug: string;
  categoryId: string;
  title: string;
  shortTitle?: string;
  badge?: string;
  version: string;
  lastUpdated: string;
  readTime: string;
  heroWordmark: [string, string]; // e.g. ["Pegasus", "Docs"] or ["Warehouse", "OS"]
  leadSentence: string;
  summary: string;
  highlights: DocHighlight[];
  architectureFlow?: {
    title: string;
    steps: string[];
    caption?: string;
  };
  steps?: DocStep[];
  codeExamples?: DocCodeExample[];
  callouts?: DocCallout[];
  operatorChecklist?: DocChecklistItem[];
  relatedTopics: DocRelatedTopic[];
};

export type DocCategory = {
  id: string;
  title: string;
  description: string;
  badge?: string;
  articles: Array<{
    slug: string;
    title: string;
    badge?: string;
  }>;
};

export const DOC_CATEGORIES: DocCategory[] = [
  {
    id: 'guides',
    title: 'Guides & Onboarding',
    description: 'System architecture, foundational invariants, and getting started.',
    badge: 'CORE',
    articles: [
      { slug: 'introduction', title: 'System Introduction', badge: 'OVERVIEW' },
      { slug: 'quickstart', title: '15-Minute Quickstart', badge: 'TUTORIAL' },
      { slug: 'network-topology', title: 'Network Topology & Zones', badge: 'CONFIG' },
    ],
  },
  {
    id: 'roles',
    title: 'Core Operator Roles',
    description: 'Manuals and operational interfaces across the six operational actors.',
    badge: '6 ROLES',
    articles: [
      { slug: 'supplier-control-plane', title: 'Supplier Control Plane', badge: 'WEB' },
      { slug: 'warehouse-operations', title: 'Warehouse Hub Operations', badge: 'DESKTOP' },
      { slug: 'driver-mobile-execution', title: 'Driver Route Execution', badge: 'MOBILE' },
      { slug: 'retailer-portal', title: 'Retailer Commerce Portal', badge: 'TELEGRAM' },
      { slug: 'factory-loading-bay', title: 'Factory Loading Bay', badge: 'BAY OS' },
      { slug: 'payload-gate-security', title: 'Payload Gate Security', badge: 'GATEWAY' },
    ],
  },
  {
    id: 'protocols',
    title: 'Protocols & Engines',
    description: 'Algorithmic solvers, synchronization engines, and ledger accounting.',
    badge: 'ENGINES',
    articles: [
      { slug: 'cvrp-route-optimizer', title: 'CVRP Route Optimizer', badge: 'OR-TOOLS' },
      { slug: 'transactional-outbox', title: 'Transactional Outbox & Bus', badge: 'KAFKA' },
      { slug: 'double-entry-ledger', title: 'Double-Entry Ledger Audit', badge: 'FINANCE' },
      { slug: 'offline-sqlite-sync', title: 'Offline-First SQLite Sync', badge: 'CRDT' },
    ],
  },
  {
    id: 'playbooks',
    title: 'Operational Playbooks',
    description: 'Handling real-world warehouse, road, and treasury edge cases.',
    badge: 'WAR ROOM',
    articles: [
      { slug: 'stockout-rejection', title: 'Concurrent Stockout Rejection', badge: 'INVENTORY' },
      { slug: 'mid-shift-driver-swap', title: 'Mid-Shift Breakdown & Swap', badge: 'FLEET' },
      { slug: 'tamper-seal-discrepancy', title: 'Tamper Seal Discrepancy', badge: 'SECURITY' },
      { slug: 'cash-at-door-cod', title: 'Cash at Door (COD) Protocols', badge: 'TREASURY' },
    ],
  },
  {
    id: 'api',
    title: 'API & Realtime Events',
    description: 'Contracts, WebSocket channels, and streaming schema specifications.',
    badge: 'DEV',
    articles: [
      { slug: 'rest-api-reference', title: 'REST API & Scopes', badge: 'v4.2' },
      { slug: 'websocket-hub', title: 'WebSocket Event Hub', badge: 'REALTIME' },
      { slug: 'kafka-event-contracts', title: 'Kafka Schema Contracts', badge: 'EVENTS' },
    ],
  },
];

export const DOC_ARTICLES: Record<string, DocArticle> = {
  // ---------------------------------------------------------------------------
  // GUIDES: INTRODUCTION
  // ---------------------------------------------------------------------------
  'guides/introduction': {
    slug: 'introduction',
    categoryId: 'guides',
    title: 'Pegasus OS Architecture & Foundation',
    shortTitle: 'System Introduction',
    badge: 'CORE ARCHITECTURE',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '6 min read',
    heroWordmark: ['Pegasus', 'Docs'],
    leadSentence:
      'Official operational blueprints, mathematical dispatch algorithms, and interface specifications for the Pegasus multi-tenant global logistics operating system.',
    summary:
      'Pegasus OS is a mission-critical supply-chain execution engine connecting 6 distinct operational roles into a single synchronized data plane. From supplier vetting and warehouse pick waves to Google OR-Tools CVRP routing, driver turn-by-turn mobile execution, and double-entry ledger settlement, Pegasus eliminates data siloing and tribal operational handoffs.',
    highlights: [
      { title: 'Deterministic Consistency', desc: 'Google Cloud Spanner and transactional PostgreSQL outboxes guarantee zero phantom inventory across all hubs.', metric: '99.999% SLA' },
      { title: 'Sub-second Fanout', desc: 'Kafka event streams fan out state changes to all 6 operational role applications in under 40 milliseconds.', metric: '<40ms' },
      { title: 'Offline Resilience', desc: 'Driver and gate terminals maintain full operational capability during cellular blackouts via local SQLite CRDT sync.', metric: '100% Offline' },
    ],
    architectureFlow: {
      title: 'Synchronized Cross-Role Order Execution Flow',
      steps: [
        '1. Retailer places order via Telegram Mini-App or Web Portal with credit-limit precheck',
        '2. Supplier Control Plane verifies inventory reservations and confirms multi-warehouse allocation',
        '3. Warehouse Desktop batches orders into optimal pick waves and triggers CVRP cluster optimization',
        '4. Driver Mobile App accepts route manifest, inspects cargo, and verifies digital tamper seal',
        '5. Gate Security scans biometric QR token and releases vehicle onto metropolitan transit corridor',
        '6. Delivery verified with GPS geofence + OTP; Double-entry ledger commits cash or card debit',
      ],
      caption: 'Every state transition is guarded by a mutating handler contract and transactional outbox emission.',
    },
    steps: [
      {
        stepNumber: '01',
        title: 'Understand the Sovereign Role Architecture',
        desc: 'Every actor in the Pegasus ecosystem interacts through an interface engineered specifically for their physical environment: suppliers use a high-density web command center, warehouse leads use a keyboard-first Tauri desktop terminal, drivers use a glove-compatible native mobile app, and retailers use a zero-friction Telegram Mini-App.',
        checklist: [
          'Verify tenant identifier and cell assignment (e.g., cell-uz-tashkent-01)',
          'Ensure double-entry ledger accounts are initialized in minor units (tiyins/cents)',
          'Confirm WebSocket coordination channels are listening for your role',
        ],
      },
      {
        stepNumber: '02',
        title: 'Verify Invariant Guardrails Before Execution',
        desc: 'Pegasus enforces strict transactional invariants. No order can transition to In Transit without a verified tamper seal ID. No payment can be marked settled without an exact ledger offset. And no driver can be dispatched without passing pre-trip inspection.',
        code: {
          lang: 'bash',
          filename: 'terminal-verify.sh',
          code: `# Check cluster health and active tenant status
curl -X GET "https://api.pegasus-logistics.io/v1/health" \\
  -H "Authorization: Bearer \${PEGASUS_JWT_TOKEN}" \\
  -H "X-Cell-Id: cell-uz-tashkent-01"`,
        },
      },
      {
        stepNumber: '03',
        title: 'Deploy and Configure Operational Nodes',
        desc: 'Connect your local edge terminals to the sovereign cluster gateway. Drivers download the native APK/IPA, while warehouse dispatchers launch the local Tauri desktop bundle with local thermal printer bindings.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Strict Architectural Separation',
        content: 'pegasusX uses Google Cloud Spanner and Apache Kafka for global multi-tenant enterprise scale, while pegasus.x uses PostgreSQL 16 and Redis Streams for sovereign national deployment. Never mix database drivers or messaging adapters across repositories.',
      },
      {
        type: 'tip',
        title: 'Real-time Telemetry Caching',
        content: 'Driver GPS breadcrumbs are cached in Redis Geospatial indexes before being batched into persistent time-series tables, guaranteeing real-time map tracking with negligible database write pressure.',
      },
    ],
    operatorChecklist: [
      { label: 'Role Scopes Configured', detail: 'JWT tokens possess required claims: supplier:admin, warehouse:operator, or driver:transit.' },
      { label: 'Geofence Boundaries Calibrated', detail: 'Warehouse dock zones and retail drop-off polygons configured with a minimum 50m tolerance.' },
      { label: 'Digital Seal Hardware Paired', detail: 'Gate terminals paired with Bluetooth BLE or QR tamper-seal scanners.' },
    ],
    relatedTopics: [
      {
        title: '15-Minute Quickstart',
        category: 'Guides & Onboarding',
        categoryId: 'guides',
        slug: 'quickstart',
        reason: 'Walk through creating your first order, generating a warehouse pick wave, and simulating driver delivery.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Supplier Control Plane',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'supplier-control-plane',
        reason: 'Master the high-density supplier web portal for inventory allocation, vetting, and financial overview.',
        badge: 'RELATED ENGINE',
      },
      {
        title: 'CVRP Route Optimizer',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'cvrp-route-optimizer',
        reason: 'Understand the mathematical CVRP constraint engine that clusters retail stops and minimizes fuel burn.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // GUIDES: QUICKSTART
  // ---------------------------------------------------------------------------
  'guides/quickstart': {
    slug: 'quickstart',
    categoryId: 'guides',
    title: '15-Minute Operational Quickstart',
    shortTitle: 'Quickstart Guide',
    badge: 'TUTORIAL',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['Quick', 'Start'],
    leadSentence:
      'Step-by-step tutorial to provision a test supplier cell, stage test inventory, generate an optimized dispatch route, and complete a simulated delivery.',
    summary:
      'This tutorial takes you from zero to a fully settled order-to-cash lifecycle. You will initialize an isolated test cell, register SKU inventory across multiple warehouse bins, place a sample retail order, run the automated CVRP solver, and trigger the outbox event chain.',
    highlights: [
      { title: 'Zero Configuration', desc: 'Pre-seeded local mock servers allow instant end-to-end testing without external cloud credentials.', metric: '0 Env Setup' },
      { title: 'Full Lifecycle Trace', desc: 'Observe orders transition across all 6 roles with live WebSocket terminal output.', metric: '15 Min' },
      { title: 'Ledger Verified', desc: 'Every step outputs verifiable double-entry transaction debit/credit balances in the console.', metric: '100% Balanced' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Authenticate and Retrieve Tenant Credentials',
        desc: 'Generate an operational session token with super-operator privileges for the designated test cell.',
        code: {
          lang: 'bash',
          filename: '01-auth.sh',
          code: `curl -X POST "https://api.pegasus-logistics.io/v1/auth/session" \\
  -H "Content-Type: application/json" \\
  -d '{"apiKey": "pegasus_test_k78x92", "cell": "cell-sandbox-01"}'`,
        },
      },
      {
        stepNumber: '02',
        title: 'Seed Catalog & Allocate Warehouse Stock',
        desc: 'Post a sample batch of Fast-Moving Consumer Goods (FMCG) items with dimensional weight and pack sizing to Warehouse Hub Alpha.',
        code: {
          lang: 'bash',
          filename: '02-seed-inventory.sh',
          code: `curl -X POST "https://api.pegasus-logistics.io/v1/inventory/batch-adjust" \\
  -H "Authorization: Bearer \${SESSION_TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "warehouseId": "wh-alpha-tashkent",
    "adjustments": [
      {"sku": "SKU-BEV-001", "quantity": 500, "unitCents": 1200000},
      {"sku": "SKU-SNK-042", "quantity": 1200, "unitCents": 450000}
    ]
  }'`,
        },
      },
      {
        stepNumber: '03',
        title: 'Dispatch the Automated CVRP Optimizer',
        desc: 'Cluster all pending orders for the morning shift. The solver calculates optimal vehicle-capacity loading and output turn-by-turn route manifests.',
        code: {
          lang: 'bash',
          filename: '03-solve-cvrp.sh',
          code: `curl -X POST "https://api.pegasus-logistics.io/v1/dispatch/optimize-routes" \\
  -H "Authorization: Bearer \${SESSION_TOKEN}" \\
  -H "Content-Type: application/json" \\
  -d '{"warehouseId": "wh-alpha-tashkent", "algorithm": "OR_TOOLS_PARALLEL_CHEAPEST_INSERTION"}'`,
        },
      },
    ],
    callouts: [
      {
        type: 'note',
        title: 'Sandbox Ledger Safe Mode',
        content: 'All quickstart operations execute against synthetic currency ledgers. No real financial debiting or live bank PSP callbacks are triggered.',
      },
    ],
    relatedTopics: [
      {
        title: 'Warehouse Hub Operations',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'warehouse-operations',
        reason: 'Learn how warehouse managers view and edit the generated CVRP routes before locking the morning manifest.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Driver Route Execution',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'driver-mobile-execution',
        reason: 'See how the driver receives this manifest on their mobile device and performs offline proof-of-delivery.',
        badge: 'RELATED ENGINE',
      },
      {
        title: 'Double-Entry Ledger Audit',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'double-entry-ledger',
        reason: 'Inspect the exact accounting entries generated when this test order transitions from Pending to Paid.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // GUIDES: NETWORK TOPOLOGY
  // ---------------------------------------------------------------------------
  'guides/network-topology': {
    slug: 'network-topology',
    categoryId: 'guides',
    title: 'Network Topology & Delivery Zone Configuration',
    shortTitle: 'Network Topology',
    badge: 'CONFIG',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '7 min read',
    heroWordmark: ['Network', 'Zones'],
    leadSentence:
      'Configure sovereign distribution centers, cross-dock transit nodes, security gate checkpoints, and polygon geofences.',
    summary:
      'Pegasus routes physical freight based on an explicit hierarchical network topology. Learn how to define facilities, dock loading bays, gate seals, and metropolitan delivery sectors to ensure orders are fulfilled from the closest warehouse.',
    highlights: [
      { title: 'Sub-Polygon Routing', desc: 'Define GeoJSON polygon boundaries with custom speed limits, bridge clearances, and vehicle weight restrictions.', metric: 'GeoJSON Spec' },
      { title: 'Dock Bay Scheduling', desc: 'Prevent yard gridlock by assigning automated 30-minute docking slots to inbound 18-wheelers.', metric: '30-Min Slots' },
      { title: 'Cross-Dock Transit', desc: 'Transfer pallets directly between long-haul trucks and last-mile delivery vans without warehouse putaway.', metric: '0 Storage Lag' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Define Central Warehouses and Loading Docks',
        desc: 'Register physical facility coordinates, capacity limits, and active loading bays. Loading bays support distinct constraints like cold storage or bulk pallet lifts.',
      },
      {
        stepNumber: '02',
        title: 'Draw Delivery Zone Geofences',
        desc: 'Upload GeoJSON polygons dividing the urban area into delivery zones (e.g., Chilanzar-North, Yunusabad-East). Each zone maps to designated dispatch fleets.',
      },
      {
        stepNumber: '03',
        title: 'Pair Gate Checkpoints with Tamper Verification Terminals',
        desc: 'Every perimeter exit requires an active Gate Node entry. Vehicles cannot be recorded as Departed until passing an authorized gate scan.',
      },
    ],
    callouts: [
      {
        type: 'warning',
        title: 'Zone Overlap Prevention',
        content: 'Ensure delivery zone polygons do not have intersecting boundaries unless priority tiering is explicitly configured in zone rules.',
      },
    ],
    relatedTopics: [
      {
        title: 'CVRP Route Optimizer',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'cvrp-route-optimizer',
        reason: 'The CVRP solver uses zone definitions to restrict vehicle routes and calculate distance matrices.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Payload Gate Security',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'payload-gate-security',
        reason: 'Gate security terminals execute physical checkpoint clearance based on the network topology nodes.',
        badge: 'RELATED ENGINE',
      },
      {
        title: 'Concurrent Stockout Rejection',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'stockout-rejection',
        reason: 'What happens when inventory at the nearest zone warehouse is depleted concurrently during order placement.',
        badge: 'PLAYBOOK',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: SUPPLIER CONTROL PLANE
  // ---------------------------------------------------------------------------
  'roles/supplier-control-plane': {
    slug: 'supplier-control-plane',
    categoryId: 'roles',
    title: 'Supplier Control Plane Operations',
    shortTitle: 'Supplier Portal',
    badge: 'WEB COMMAND',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '9 min read',
    heroWordmark: ['Supplier', 'Portal'],
    leadSentence:
      'The multi-tenant command center for catalog management, retailer credit vetting, multi-warehouse stock allocation, and financial treasury oversight.',
    summary:
      'The Supplier Control Plane is a high-density tactical web application engineered for executive operators and commercial dispatchers. It provides real-time visibility across all physical warehouses, active delivery routes, retailer credit health, and gross merchandise volume (GMV).',
    highlights: [
      { title: '3-Column Control Tower', desc: 'Nav Rail → Live Density Feed → Deep Inspector Drawer. Zero generic SaaS fluff.', metric: 'Tactical UI' },
      { title: 'Credit Limit Enforcement', desc: 'Prevent high-risk order fulfillment with automatic credit checks against the retailer ledger balance.', metric: 'Realtime Check' },
      { title: 'Batch Vetting Console', desc: 'Approve, adjust, or reject up to 500 incoming retail orders in a single keyboard-driven workflow.', metric: '500/min' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Monitor Live Inbound Orders',
        desc: 'Review incoming retail orders grouped by geographical zone and fulfillment urgency. Urgent cold-chain orders are highlighted with tactical amber banners.',
      },
      {
        stepNumber: '02',
        title: 'Perform Credit & Pricing Vetting',
        desc: 'Verify retailer credit terms. If a retailer exceeds their credit threshold, the system prompts for partial upfront payment via digital invoice.',
      },
      {
        stepNumber: '03',
        title: 'Lock Morning Warehouse Allocations',
        desc: 'Commit verified orders to their respective regional distribution centers. Committing triggers the warehouse pick wave generation.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Double-Entry Invariant',
        content: 'When an order is vetted and locked, stock is atomically decremented from available_inventory and credited to reserved_inventory within a single Spanner read-write transaction.',
      },
    ],
    relatedTopics: [
      {
        title: 'Warehouse Hub Operations',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'warehouse-operations',
        reason: 'Warehouse operators pick, pack, and palletize the orders committed from the Supplier Control Plane.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Double-Entry Ledger Audit',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'double-entry-ledger',
        reason: 'Review the mathematical balance sheets and payment settlement mechanisms configured in the treasury panel.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'Retailer Commerce Portal',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'retailer-portal',
        reason: 'Understand how retailers view pricing tiers, credit allowances, and place orders into the vetting queue.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: WAREHOUSE OPERATIONS
  // ---------------------------------------------------------------------------
  'roles/warehouse-operations': {
    slug: 'warehouse-operations',
    categoryId: 'roles',
    title: 'Warehouse Operations & Dispatch Hub',
    shortTitle: 'Warehouse OS',
    badge: 'TAURI DESKTOP',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '10 min read',
    heroWordmark: ['Warehouse', 'OS'],
    leadSentence:
      'Native keyboard-first Tauri desktop suite for morning pick waves, bulk pallet staging, truck loading validation, and automated CVRP route dispatch.',
    summary:
      'Built for the rugged reality of warehouse floors, the Warehouse Hub desktop application operates flawlessly under high-velocity barcode scanning. It manages morning order waves, optimizes picking paths inside aisles, validates pallet weights, and generates digital manifests for truck drivers.',
    highlights: [
      { title: 'Keyboard-First Speed', desc: 'Every warehouse action—from pick confirmation to route override—can be executed without touching a mouse.', metric: '100% Hotkeys' },
      { title: 'Sub-Millisecond Barcode Scan', desc: 'Direct USB/Bluetooth HID laser scanner integration with immediate audio-tactile feedback tones.', metric: '<1ms Latency' },
      { title: 'CVRP Visual Board', desc: 'Interactive Gantt and map board allowing warehouse managers to drag and drop orders between delivery trucks.', metric: 'Live Override' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Trigger Morning Wave Generation',
        desc: 'At 05:00, the system aggregates all vetted orders into optimized pick waves grouped by warehouse aisle coordinate.',
      },
      {
        stepNumber: '02',
        title: 'Scan and Stage Orders to Staging Bays',
        desc: 'Floor pickers scan items using mobile hand terminals. The desktop hub displays real-time progress bars for each staging lane.',
      },
      {
        stepNumber: '03',
        title: 'Load Trucks and Issue Manifests',
        desc: 'Pallets are rolled into trucks in reverse delivery order (Last In, First Out). The warehouse lead prints the route manifest and attaches the digital tamper seal.',
      },
    ],
    callouts: [
      {
        type: 'warning',
        title: 'Weight and Cube Invariants',
        content: 'The desktop app will block manifest generation if truck gross vehicle weight (GVW) or volumetric capacity exceeds 98% without supervisor sign-off.',
      },
    ],
    relatedTopics: [
      {
        title: 'Driver Route Execution',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'driver-mobile-execution',
        reason: 'Once the truck is loaded, the driver accepts the manifest on their mobile device and performs pre-trip inspection.',
        badge: 'NEXT STEP',
      },
      {
        title: 'CVRP Route Optimizer',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'cvrp-route-optimizer',
        reason: 'Learn how the CVRP solver determines the loading sequence and estimated time of arrival for every stop.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'Mid-Shift Breakdown & Swap',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'mid-shift-driver-swap',
        reason: 'Protocol for re-allocating a loaded truck if a mechanical failure occurs on the warehouse apron.',
        badge: 'PLAYBOOK',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: DRIVER MOBILE EXECUTION
  // ---------------------------------------------------------------------------
  'roles/driver-mobile-execution': {
    slug: 'driver-mobile-execution',
    categoryId: 'roles',
    title: 'Driver Mobile Route Execution',
    shortTitle: 'Driver Mobile',
    badge: 'NATIVE ANDROID / IOS',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['Driver', 'Mobile'],
    leadSentence:
      'Mission-critical mobile application with turn-by-turn routing, offline SQLite CRDT synchronization, electronic proof-of-delivery (ePOD), and cash collection.',
    summary:
      'Engineered for the demanding urban delivery driver, the Pegasus Driver app features high-contrast outdoor UI, single-thumb navigation ergonomics, automatic geofenced stop arrivals, and complete offline capability during cellular signal drops.',
    highlights: [
      { title: 'Offline-First SQLite', desc: 'Accept orders, collect digital signatures, and register cash payments even in basement supermarkets without signal.', metric: '100% Offline' },
      { title: 'Geofenced Arrivals', desc: 'Detects truck entry into retail drop-off polygons and prompts the driver with unloading instructions automatically.', metric: 'Auto-Geofence' },
      { title: 'Cash Vault Safeguards', desc: 'Real-time calculation of physical cash collected with strict courier liability caps and mid-route bank drop prompts.', metric: 'COD Security' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Accept Daily Manifest & Run Pre-Trip Inspection',
        desc: 'Driver inspects tire pressure, fluid levels, and cooling unit temperature, photographing any existing vehicle damage before leaving the yard.',
      },
      {
        stepNumber: '02',
        title: 'Follow Turn-by-Turn Navigation',
        desc: 'In-app OSRM navigation guides the driver through the sequence of retail stores, optimizing for commercial truck clearance and traffic patterns.',
      },
      {
        stepNumber: '03',
        title: 'Complete Proof-of-Delivery & Collect Payment',
        desc: 'Scan drop-off crates, record retailer barcode confirmation, collect digital signature or cash-on-delivery, and issue digital tax receipt.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Proof-of-Delivery Geofence Requirement',
        content: 'Delivery confirmation buttons remain disabled unless the driver GPS coordinates are verified within 75 meters of the registered retailer storefront.',
      },
    ],
    relatedTopics: [
      {
        title: 'Cash at Door (COD) Protocols',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'cash-at-door-cod',
        reason: 'Detailed procedures for handling cash shortages, counterfeit banknotes, or partial retail cash payments.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Offline-First SQLite Sync',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'offline-sqlite-sync',
        reason: 'How local SQLite mutations are queued, encrypted, and merged back into the cloud Spanner cluster.',
        badge: 'RELATED ENGINE',
      },
      {
        title: 'Payload Gate Security',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'payload-gate-security',
        reason: 'Gate protocols for driver barcode scanning upon warehouse exit and return at shift conclusion.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: RETAILER PORTAL
  // ---------------------------------------------------------------------------
  'roles/retailer-portal': {
    slug: 'retailer-portal',
    categoryId: 'roles',
    title: 'Retailer Commerce Portal & Bot',
    shortTitle: 'Retailer Portal',
    badge: 'TELEGRAM & WEB',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '6 min read',
    heroWordmark: ['Retailer', 'Portal'],
    leadSentence:
      'Frictionless ordering interface via Telegram Mini-App and web portal with live order tracking, transparent credit balances, and one-tap reordering.',
    summary:
      'Store owners and procurement managers order directly from FMCG manufacturers without intermediary sales broker markups. Features real-time stock availability, live truck GPS radar, digital return requests, and instant invoice downloads.',
    highlights: [
      { title: 'Zero-Install Telegram App', desc: 'Over 90% of regional retailers place orders directly through our lightweight Telegram Mini-App.', metric: '0 App Store' },
      { title: 'Live Truck Telemetry', desc: 'Retailers view their inbound delivery truck moving on a map with a 5-minute arrival warning push notification.', metric: 'Live Radar' },
      { title: 'Credit & Payment Ledger', desc: 'Clear visibility into outstanding invoices, credit limits, applied discounts, and payment history.', metric: 'Transparent' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Browse Supplier Catalog & Promotional Deals',
        desc: 'Store owners review manufacturer-direct pricing, volume discount thresholds, and guaranteed delivery cut-off times.',
      },
      {
        stepNumber: '02',
        title: 'Submit Order with Flexible Payment Method',
        desc: 'Select payment terms: Upfront Card/Bank transfer, 14-day Credit Term, or Cash on Delivery (COD) upon physical crate inspection.',
      },
      {
        stepNumber: '03',
        title: 'Track Delivery and Confirm Arrival OTP',
        desc: 'When the driver arrives, provide the 4-digit security OTP to confirm delivery and automatically update the digital ledger.',
      },
    ],
    callouts: [
      {
        type: 'tip',
        title: 'One-Tap Morning Reorder',
        content: 'Retailers can configure automated reorder templates based on historical weekly sales velocity to avoid stockouts on high-demand items.',
      },
    ],
    relatedTopics: [
      {
        title: 'Supplier Control Plane',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'supplier-control-plane',
        reason: 'Suppliers receive retail orders instantly and manage credit approvals through their control plane.',
        badge: 'RELATED ENGINE',
      },
      {
        title: 'Concurrent Stockout Rejection',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'stockout-rejection',
        reason: 'How the retailer portal handles situations where high demand depletes inventory right as order is submitted.',
        badge: 'EDGE CASE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: FACTORY LOADING BAY
  // ---------------------------------------------------------------------------
  'roles/factory-loading-bay': {
    slug: 'factory-loading-bay',
    categoryId: 'roles',
    title: 'Factory Loading Bay Operations',
    shortTitle: 'Factory Bay',
    badge: 'BAY OS',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '7 min read',
    heroWordmark: ['Factory', 'Bay OS'],
    leadSentence:
      'Industrial touch-screen terminal for production pallet release, inter-facility line-haul loading, and automated manifest stamping.',
    summary:
      'Connects the factory production line directly to the distribution logistics network. Loading bay operators scan SSCC pallet barcodes, record batch production numbers, assign pallets to line-haul trailers, and seal cross-country freight loads.',
    highlights: [
      { title: 'SSCC Pallet Tracking', desc: 'Standard GS1-128 SSCC pallet barcodes tracked from end-of-line palletizer to regional cross-dock.', metric: 'GS1-128' },
      { title: 'Bay Turnaround Timer', desc: 'Visual countdown displays for each loading bay to minimize trailer detention fees.', metric: '<35 Min Bay' },
      { title: 'Automated Manifest Sign-off', desc: 'Driver electronic signature and tamper seal registration automatically stamps the freight waybill.', metric: 'Paperless' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Check-in Line-Haul Transport Vehicle',
        desc: 'Validate arriving carrier trailer license plate, driver identity, and tare weight on the weighbridge scale.',
      },
      {
        stepNumber: '02',
        title: 'Scan and Forklift Pallets into Trailer',
        desc: 'Scan each pallet barcode with vehicle-mounted forklift terminals. The terminal sounds an alarm if the pallet belongs to a different destination facility.',
      },
      {
        stepNumber: '03',
        title: 'Apply Physical Tamper Seal and Seal Gate Record',
        desc: 'Close trailer doors, apply numbered cable seal, and photograph the locked latch for the gate verification record.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Lot Traceability Invariant',
        content: 'Every pallet scanned into a line-haul trailer must be linked to a valid quality-control batch release certificate in the database.',
      },
    ],
    relatedTopics: [
      {
        title: 'Payload Gate Security',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'payload-gate-security',
        reason: 'Trailers leaving the factory must pass the gate security checkpoint before entering public highways.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Tamper Seal Discrepancy',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'tamper-seal-discrepancy',
        reason: 'Security protocols if a trailer arrives at a regional warehouse with a damaged or mismatched seal.',
        badge: 'PLAYBOOK',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // ROLES: PAYLOAD GATE SECURITY
  // ---------------------------------------------------------------------------
  'roles/payload-gate-security': {
    slug: 'payload-gate-security',
    categoryId: 'roles',
    title: 'Payload Gate Security & Clearance',
    shortTitle: 'Gate Security',
    badge: 'PERIMETER SECURITY',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '7 min read',
    heroWordmark: ['Gate', 'Security'],
    leadSentence:
      'Perimeter checkpoint gate control system enforcing digital tamper seal verification, driver biometric authorization, and optical license plate recognition.',
    summary:
      'The Gate Security application is the final guardian of inventory integrity. No vehicle enters or exits a Pegasus-managed distribution campus without verified digital authorization, seal inspection, and load weight validation.',
    highlights: [
      { title: 'ALPR Camera Integration', desc: 'Automatic License Plate Recognition matches incoming trucks against active dispatch manifests in 400ms.', metric: '400ms ALPR' },
      { title: 'Digital Seal Check', desc: 'Compares physical cable seal number against the manifest stamped by warehouse dispatch.', metric: '0 Misload Gate' },
      { title: 'Weighbridge Integration', desc: 'Gross vehicle weight recorded automatically to verify physical load matches manifest invoice.', metric: 'Scale Synced' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Truck Approaches Gate Camera',
        desc: 'ALPR cameras capture front and rear plates, pulling up the active route manifest on the security officer terminal.',
      },
      {
        stepNumber: '02',
        title: 'Scan Driver QR Token & Inspect Cable Seal',
        desc: 'Security officer inspects the mechanical bolt/cable seal, confirming the engraved serial number matches the digital manifest record.',
      },
      {
        stepNumber: '03',
        title: 'Trigger Automated Barrier Arm Release',
        desc: 'System logs Departure timestamp, shifts vehicle status to In Transit, and opens the hydraulic security barrier.',
      },
    ],
    callouts: [
      {
        type: 'warning',
        title: 'Immediate Lockdown Trigger',
        content: 'If a seal number does not match the manifest, the gate terminal raises a Red Alert, holding the vehicle in the secondary inspection bay.',
      },
    ],
    relatedTopics: [
      {
        title: 'Tamper Seal Discrepancy',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'tamper-seal-discrepancy',
        reason: 'Standard operating procedure when gate security detects a broken, tampered, or mismatched seal.',
        badge: 'PLAYBOOK',
      },
      {
        title: 'Warehouse Hub Operations',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'warehouse-operations',
        reason: 'Warehouse supervisors stamp the digital seal record checked by gate security.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PROTOCOLS: CVRP ROUTE OPTIMIZER
  // ---------------------------------------------------------------------------
  'protocols/cvrp-route-optimizer': {
    slug: 'cvrp-route-optimizer',
    categoryId: 'protocols',
    title: 'CVRP Route Optimizer & Heuristics',
    shortTitle: 'CVRP Engine',
    badge: 'OR-TOOLS ENGINE',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '11 min read',
    heroWordmark: ['CVRP', 'Solver'],
    leadSentence:
      'Mathematical Capacitated Vehicle Routing Problem (CVRP) solver powered by Google OR-Tools with time-windows, volumetric constraints, and road-network distance matrices.',
    summary:
      'Pegasus solves complex combinatorial vehicle routing problems every morning across thousands of retail stops. The engine models multi-compartment vehicle volumes, maximum shift durations, store operating time-windows, vehicle turning radius restrictions, and live traffic forecasts to minimize total fleet kilometers and fuel burn.',
    highlights: [
      { title: 'Google OR-Tools Core', desc: 'Solves 1,500 delivery stops across 60 trucks in under 8 seconds using parallel Guided Local Search.', metric: '<8s Solve' },
      { title: 'Multi-Capacity Constraints', desc: 'Simultaneously balances gross payload weight (kg) and volumetric cube (m³) against vehicle capacity.', metric: 'Dual Capacity' },
      { title: 'Time-Window Windows', desc: 'Guarantees deliveries occur within store delivery hours, penalizing late arrivals with soft/hard bounds.', metric: 'Strict Windows' },
    ],
    architectureFlow: {
      title: 'CVRP Optimization Pipeline',
      steps: [
        '1. Ingestion: Filter vetted orders for target warehouse and delivery shift',
        '2. Distance Matrix: Compute OSRM travel durations and distances across all stop pairs',
        '3. Model Formulation: Initialize Google OR-Tools RoutingModel with vehicle dimension constraints',
        '4. Initial Solution: Parallel Cheapest Insertion heuristic generates seed solution',
        '5. Metaheuristic Search: Guided Local Search (GLS) iteratively explores neighbor solutions',
        '6. Assignment: Write optimal vehicle manifests to Spanner with route geo-polylines',
      ],
    },
    codeExamples: [
      {
        title: 'Go CVRP Optimizer Service Binding',
        lang: 'go',
        filename: 'cvrp_solver.go',
        code: `package routing

import (
  "context"
  "time"
  "github.com/pegasus/engine/ortools"
)

type RouteRequest struct {
  WarehouseID string            \`json:"warehouse_id"\`
  Stops       []DeliveryStop    \`json:"stops"\`
  Vehicles    []VehicleCapacity \`json:"vehicles"\`
}

func (s *Solver) SolveCVRP(ctx context.Context, req RouteRequest) (*RouteSolution, error) {
  matrix, err := s.osrmClient.GetDistanceMatrix(ctx, req.Stops)
  if err != nil {
    return nil, err
  }
  
  params := ortools.DefaultParameters()
  params.TimeLimit = 10 * time.Second
  params.Metaheuristic = ortools.GuidedLocalSearch
  
  return ortools.Solve(matrix, req.Vehicles, params)
}`,
      },
    ],
    callouts: [
      {
        type: 'tip',
        title: 'Pre-computed Distance Matrix Caching',
        content: 'Distance and duration pairs between frequently visited retail clusters are cached in Redis with a 24-hour TTL, reducing solver latency by up to 70%.',
      },
    ],
    relatedTopics: [
      {
        title: 'Warehouse Hub Operations',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'warehouse-operations',
        reason: 'Warehouse dispatchers view, adjust, and approve the CVRP solution on the interactive dispatch board.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Mid-Shift Breakdown & Swap',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'mid-shift-driver-swap',
        reason: 'How the CVRP solver re-computes sub-routes when a vehicle breaks down mid-route.',
        badge: 'EDGE CASE',
      },
      {
        title: 'Transactional Outbox & Bus',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'transactional-outbox',
        reason: 'Route assignment results are published to Kafka to update driver and warehouse apps in real time.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PROTOCOLS: TRANSACTIONAL OUTBOX
  // ---------------------------------------------------------------------------
  'protocols/transactional-outbox': {
    slug: 'transactional-outbox',
    categoryId: 'protocols',
    title: 'Transactional Outbox Pattern & Event Bus',
    shortTitle: 'Transactional Outbox',
    badge: 'KAFKA EVENT BUS',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '9 min read',
    heroWordmark: ['Outbox', 'Bus'],
    leadSentence:
      'Guaranteed at-least-once event delivery pairing atomic database mutations with Apache Kafka topic distribution.',
    summary:
      'Distributed systems cannot safely write to a database and publish to a message broker in two uncoordinated steps. Pegasus implements an atomic Transactional Outbox pattern: state mutations and outbox event rows are committed in the same database transaction. Dedicated Go worker pools tail the outbox and stream events to Kafka with zero data loss.',
    highlights: [
      { title: 'Zero Phantom Events', desc: 'Events are never emitted if a database transaction aborts or rolls back.', metric: '100% Atomic' },
      { title: 'Partition Ordering', desc: 'Events keyed by supplier_id and order_id guarantee strict per-entity causal ordering in Kafka partitions.', metric: 'FIFO Keyed' },
      { title: 'Sub-40ms Propagation', desc: 'High-performance poller and changefeed streaming pushes events to WebSocket hubs in under 40 milliseconds.', metric: '<40ms Lag' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Atomic Read-Write Transaction Write',
        desc: 'Inside a Spanner read-write transaction (or Postgres Serializable block), write the domain entity mutation and insert an OutboxEvent row.',
      },
      {
        stepNumber: '02',
        title: 'Go Outbox Publisher Polling',
        desc: 'Stateless Go background workers query un-published outbox records with row-level locks, publishing to corresponding Kafka topics.',
      },
      {
        stepNumber: '03',
        title: 'Consumer Acknowledgment & De-duplication',
        desc: 'Downstream services consume events and use idempotent deduplication tables to handle at-least-once message delivery safely.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'No Direct Broker Emits in Handlers',
        content: 'HTTP/gRPC request handlers are strictly forbidden from publishing directly to Kafka. All state emissions must pass through the atomic outbox table.',
      },
    ],
    relatedTopics: [
      {
        title: 'WebSocket Event Hub',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'websocket-hub',
        reason: 'Kafka events published by the outbox worker are fanned out to active user interfaces via the WebSocket hub.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Kafka Schema Contracts',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'kafka-event-contracts',
        reason: 'View the strict Protobuf and JSON Schema event envelopes published by the outbox engine.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PROTOCOLS: DOUBLE-ENTRY LEDGER
  // ---------------------------------------------------------------------------
  'protocols/double-entry-ledger': {
    slug: 'double-entry-ledger',
    categoryId: 'protocols',
    title: 'Double-Entry Ledger & Financial Invariants',
    shortTitle: 'Ledger Engine',
    badge: 'FINANCIAL AUDIT',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '10 min read',
    heroWordmark: ['Ledger', 'Finance'],
    leadSentence:
      'Class A financial accounting core tracking every cent and tiyin across supplier receivables, driver cash liability, and retailer credit balances.',
    summary:
      'In high-volume physical distribution, financial integrity is as critical as inventory accuracy. Pegasus implements an immutable double-entry journal ledger. Account balances are never directly updated; they are computed as the sum of immutable debit and credit lines. Every transaction sums strictly to zero.',
    highlights: [
      { title: 'Strict Minor Units', desc: 'All monetary values are stored as 64-bit signed integers (tiyins/cents). Floating-point math is strictly forbidden.', metric: 'Int64 Units' },
      { title: 'Sum-to-Zero Invariant', desc: 'Every journal entry must balance mathematically: Σ(Debits) - Σ(Credits) = 0. Unbalanced entries are rejected at DB level.', metric: 'Zero Sum' },
      { title: 'Immutable Audit Trail', desc: 'Ledger rows cannot be updated or deleted. Corrections are made strictly via timestamped reversal journals.', metric: 'Immutable' },
    ],
    architectureFlow: {
      title: 'Order-to-Cash Ledger Journal Example',
      steps: [
        '1. Order Placed (Pre-paid): DEBIT Retailer_Accounts_Receivable, CREDIT Inventory_Committed_Liability',
        '2. Warehouse Loaded: DEBIT In_Transit_Inventory, CREDIT Warehouse_Inventory_Asset',
        '3. Cash Delivered: DEBIT Driver_Cash_Vault, CREDIT Retailer_Accounts_Receivable',
        '4. Driver Reconciled: DEBIT Bank_Treasury_Settlement, CREDIT Driver_Cash_Vault',
      ],
    },
    callouts: [
      {
        type: 'important',
        title: 'Zero Float Rounding Errors',
        content: 'Always divide discounts and VAT across line items using the largest remainder method to ensure line-item penny totals match the master invoice perfectly.',
      },
    ],
    relatedTopics: [
      {
        title: 'Cash at Door (COD) Protocols',
        category: 'Operational Playbooks',
        categoryId: 'playbooks',
        slug: 'cash-at-door-cod',
        reason: 'Operational handbook for courier cash collection, till balancing, and physical bank deposits.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Supplier Control Plane',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'supplier-control-plane',
        reason: 'How the supplier finance team views automated ledger reconciliation and accounts aging reports.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PROTOCOLS: OFFLINE SQLITE SYNC
  // ---------------------------------------------------------------------------
  'protocols/offline-sqlite-sync': {
    slug: 'offline-sqlite-sync',
    categoryId: 'protocols',
    title: 'Offline-First SQLite Synchronization',
    shortTitle: 'Offline SQLite',
    badge: 'CRDT SYNC',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['Offline', 'SQLite'],
    leadSentence:
      'Conflict-free offline database synchronization for driver mobile devices and warehouse handheld scanners operating in dead zones.',
    summary:
      'Logistics happens in basements, steel warehouse complexes, and remote mountain highways where 4G/5G connectivity drops. Pegasus mobile applications read and write directly to an embedded SQLite database, queuing mutation intents and synchronizing with cloud servers via delta CRDT logs upon reconnection.',
    highlights: [
      { title: 'Zero UI Freezes', desc: 'Mobile screens render instantly from local SQLite without blocking on remote HTTP network round-trips.', metric: '0 Network Lag' },
      { title: 'Deterministic Conflict Resolution', desc: 'State machine rules ensure proof-of-delivery timestamps and customer signatures take precedence over server clock drift.', metric: 'CRDT Resolved' },
      { title: 'Encrypted at Rest', desc: 'SQLite files on driver phones are protected using SQLCipher 256-bit AES encryption.', metric: 'AES-256' },
    ],
    callouts: [
      {
        type: 'tip',
        title: 'Optimistic UI Updates',
        content: 'Drivers see completed checkmarks immediately upon scanning a package. Sync progress is displayed as a subtle status icon in the top header.',
      },
    ],
    relatedTopics: [
      {
        title: 'Driver Route Execution',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'driver-mobile-execution',
        reason: 'Explore the mobile application implementation that relies on this offline SQLite sync architecture.',
        badge: 'NEXT STEP',
      },
      {
        title: 'REST API & Scopes',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'rest-api-reference',
        reason: 'Review the /v1/sync/delta endpoint used by mobile clients to push and pull offline mutations.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PLAYBOOKS: STOCKOUT REJECTION
  // ---------------------------------------------------------------------------
  'playbooks/stockout-rejection': {
    slug: 'stockout-rejection',
    categoryId: 'playbooks',
    title: 'Playbook: Concurrent Stockout Rejection',
    shortTitle: 'Stockout Rejection',
    badge: 'INVENTORY PLAYBOOK',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '7 min read',
    heroWordmark: ['Stockout', 'Playbook'],
    leadSentence:
      'Standard operating procedure for managing simultaneous order placement when physical inventory is exhausted in real time.',
    summary:
      'During promotional campaigns or seasonal spikes, dozens of retailers may attempt to order the remaining pallet of goods within the same second. This playbook outlines how Pegasus handles serializable reservation conflicts, instant credit releases, and transparent retailer messaging.',
    highlights: [
      { title: 'Atomic Row Locking', desc: 'Database transactions use SELECT FOR UPDATE NOWAIT to prevent concurrent overselling.', metric: 'Zero Oversell' },
      { title: 'Instant Credit Refund', desc: 'Pre-authorized digital funds are released immediately back to the retailer balance with zero delay.', metric: '<50ms Refund' },
      { title: 'Automated Backorder Offer', desc: 'The retailer app offers to automatically fulfill the item from tomorrow morning factory line-haul.', metric: 'Auto Backorder' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Detect Reservation Conflict',
        desc: 'If requested quantity exceeds unreserved inventory, the database transaction returns an ErrInsufficientStock error code.',
      },
      {
        stepNumber: '02',
        title: 'Execute Immediate Compensating Action',
        desc: 'The mutating handler rolls back the order line, marks status as REJECTED_STOCKOUT, and reverses the credit reservation in the ledger.',
      },
      {
        stepNumber: '03',
        title: 'Notify Retailer and Propose Substitution',
        desc: 'Send an instant push notification to the Telegram Mini-App offering similar alternative SKUs or tomorrow delivery.',
      },
    ],
    callouts: [
      {
        type: 'warning',
        title: 'Avoid Partial Quantity Confusion',
        content: 'Always allow retailers to configure their preference: Partial Fulfillment Allowed vs All-or-Nothing to prevent surprise shipping fees.',
      },
    ],
    relatedTopics: [
      {
        title: 'Supplier Control Plane',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'supplier-control-plane',
        reason: 'Suppliers monitor real-time stock burn rates and configure emergency restock alerts in the control plane.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Double-Entry Ledger Audit',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'double-entry-ledger',
        reason: 'Review the reversing journal entries triggered when an order is cancelled due to a stockout.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PLAYBOOKS: MID-SHIFT DRIVER SWAP
  // ---------------------------------------------------------------------------
  'playbooks/mid-shift-driver-swap': {
    slug: 'mid-shift-driver-swap',
    categoryId: 'playbooks',
    title: 'Playbook: Mid-Shift Breakdown & Custody Swap',
    shortTitle: 'Mid-Shift Swap',
    badge: 'FLEET PLAYBOOK',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '9 min read',
    heroWordmark: ['Driver', 'Swap'],
    leadSentence:
      'Procedure for transferring active cargo, cash vault liabilities, and remaining route stops when a vehicle breaks down or a driver is incapacitated.',
    summary:
      'When an en-route truck suffers engine failure or an accident, every minute counts. This playbook establishes the protocol for re-assigning remaining stops to a relief vehicle, logging physical cargo count transfers, and maintaining a strict chain of custody for cash collected.',
    highlights: [
      { title: 'One-Click Relief Dispatch', desc: 'Warehouse dispatch selects the broken vehicle and triggers an automatic route split to nearest available relief driver.', metric: 'Instant Re-route' },
      { title: 'Barcode Handover Audit', desc: 'Relief driver scans packages transferred from the stranded truck to confirm zero missing items.', metric: '100% Chain-of-Custody' },
      { title: 'Cash Vault Handoff', desc: 'Digital receipt generated transferring cash collected up to the incident point to the relief driver liability account.', metric: 'Vault Balanced' },
    ],
    steps: [
      {
        stepNumber: '01',
        title: 'Driver Reports Breakdown via Mobile SOS',
        desc: 'Driver presses SOS in the mobile app, reporting vehicle location, breakdown category, and physical safety status.',
      },
      {
        stepNumber: '02',
        title: 'Dispatcher Assigns Relief Fleet Unit',
        desc: 'The warehouse desktop board recalculates remaining stops, sending the coordinates of the disabled truck to the nearest standby driver.',
      },
      {
        stepNumber: '03',
        title: 'Perform Roadside Scanning & Handover',
        desc: 'Both drivers authenticate on the relief driver phone, scanning the physical crates and signing the digital transfer manifest.',
      },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Security Seal Protocol',
        content: 'If the breakdown occurs on a sealed high-value freight trailer, the warehouse supervisor must authorize the seal break remotely via SMS OTP.',
      },
    ],
    relatedTopics: [
      {
        title: 'Warehouse Hub Operations',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'warehouse-operations',
        reason: 'The warehouse supervisor oversees the mid-shift re-assignment and updates customer ETAs.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Driver Route Execution',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'driver-mobile-execution',
        reason: 'Drivers utilize the in-app SOS mode and mutual QR transfer screen to complete the roadside handover.',
        badge: 'PREREQUISITE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PLAYBOOKS: TAMPER SEAL DISCREPANCY
  // ---------------------------------------------------------------------------
  'playbooks/tamper-seal-discrepancy': {
    slug: 'tamper-seal-discrepancy',
    categoryId: 'playbooks',
    title: 'Playbook: Tamper Seal Discrepancy & Quarantine',
    shortTitle: 'Seal Discrepancy',
    badge: 'SECURITY PLAYBOOK',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['Seal', 'Quarantine'],
    leadSentence:
      'Protocol for handling broken, altered, or mismatched cargo seals discovered at security gates or warehouse receiving docks.',
    summary:
      'A broken or unverified seal indicates potential cargo theft, tampering, or contraband introduction. This playbook defines the quarantine procedure, digital incident logging, supervisor escalation, and insurance claim packaging.',
    highlights: [
      { title: 'Immediate Bay Quarantine', desc: 'The transport vehicle is escorted to an isolated security quarantine bay monitored by CCTV.', metric: 'Quarantine Bay' },
      { title: 'Photo Audit Log', desc: 'Gate terminal takes multi-angle photographs of the damaged seal and latches, hashing them into the audit log.', metric: 'Cryptographic Proof' },
      { title: 'Full Blind Recount', desc: 'Warehouse receiving team performs an item-by-item blind inventory count against original line-haul manifest.', metric: '100% Recount' },
    ],
    callouts: [
      {
        type: 'warning',
        title: 'Never Clear Gate on Oral Confirmation',
        content: 'Security officers are strictly forbidden from clearing a vehicle whose seal does not match the system record without two supervisor digital keys.',
      },
    ],
    relatedTopics: [
      {
        title: 'Payload Gate Security',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'payload-gate-security',
        reason: 'Gate officers initiate the quarantine protocol upon scanning an invalid seal serial number.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'Factory Loading Bay',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'factory-loading-bay',
        reason: 'Trace back the initial seal affixation and CCTV recording at the manufacturing departure bay.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // PLAYBOOKS: CASH AT DOOR (COD)
  // ---------------------------------------------------------------------------
  'playbooks/cash-at-door-cod': {
    slug: 'cash-at-door-cod',
    categoryId: 'playbooks',
    title: 'Playbook: Cash at Door (COD) Collection Protocols',
    shortTitle: 'Cash at Door (COD)',
    badge: 'TREASURY PLAYBOOK',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['Cash', 'Vault'],
    leadSentence:
      'Rigorous field procedures for physical banknote verification, till count matching, and end-of-day bank vault deposit reconciliation.',
    summary:
      'In emerging and cash-dominant distribution markets, physical Cash-on-Delivery accounts for over 40% of retail transactions. This playbook enforces courier physical security, cash collection limits, partial payment dispute handling, and automated warehouse till settlement.',
    highlights: [
      { title: 'Real-time Vault Tracking', desc: 'The driver app maintains a running count of banknotes in the vehicle lockbox, enforcing liability limits.', metric: 'Dynamic Till' },
      { title: 'Mid-Route Safe Drops', desc: 'When cash exceeds the maximum insured threshold, the driver is routed to a partner bank branch for deposit.', metric: 'Insured Drops' },
      { title: 'Automated Cashier Count', desc: 'Warehouse cashier uses multi-currency banknote counters integrated directly with the Pegasus desktop hub.', metric: '<2 Min Settlement' },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Banknote Counterfeit Protocol',
        content: 'If a counterfeit banknote is detected during retail delivery, the driver marks the disputed note in the app, which generates a partial credit debit note instantly.',
      },
    ],
    relatedTopics: [
      {
        title: 'Double-Entry Ledger Audit',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'double-entry-ledger',
        reason: 'Explore how driver cash collection and warehouse bank deposits update the double-entry accounting books.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'Driver Route Execution',
        category: 'Core Operator Roles',
        categoryId: 'roles',
        slug: 'driver-mobile-execution',
        reason: 'Review the in-app cash collection screen and digital customer receipt generation on the driver mobile device.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // API: REST API REFERENCE
  // ---------------------------------------------------------------------------
  'api/rest-api-reference': {
    slug: 'rest-api-reference',
    categoryId: 'api',
    title: 'REST API & Authorization Scopes',
    shortTitle: 'REST API Reference',
    badge: 'API v4.2',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '10 min read',
    heroWordmark: ['REST', 'API'],
    leadSentence:
      'Complete specification of HTTP/2 and gRPC REST gateway endpoints, JWT bearer tokens, role scopes, and rate limit quotas.',
    summary:
      'The Pegasus API provides programmatic access to all supply chain entities. All requests require TLS 1.3, an authorized JSON Web Token (JWT) with relevant RBAC claims, and a valid tenant header.',
    highlights: [
      { title: 'Standard JSON:API Response', desc: 'Every API endpoint conforms to standard response wrappers with deterministic error codes and trace IDs.', metric: 'Clean Schema' },
      { title: 'Idempotency Keys', desc: 'State-changing POST/PUT requests accept Idempotency-Key headers to guarantee safe retries across spotty networks.', metric: 'UUIDv4 Keys' },
      { title: 'Adaptive Rate Limiting', desc: 'Token-bucket rate limiter with per-tenant burst quotas and Redis-backed state.', metric: '10,000 req/s' },
    ],
    codeExamples: [
      {
        title: 'Fetch Warehouse Order Manifest',
        lang: 'bash',
        filename: 'curl-get-manifest.sh',
        code: `curl -X GET "https://api.pegasus-logistics.io/v1/warehouses/wh-alpha/manifests/today" \\
  -H "Authorization: Bearer eyJhbGciOi..." \\
  -H "X-Tenant-Id: sup_tashkent_fmcg_01" \\
  -H "Accept: application/json"`,
      },
      {
        title: 'Submit Batch Order Creation',
        lang: 'bash',
        filename: 'curl-post-order.sh',
        code: `curl -X POST "https://api.pegasus-logistics.io/v1/orders" \\
  -H "Authorization: Bearer eyJhbGciOi..." \\
  -H "Idempotency-Key: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d" \\
  -H "Content-Type: application/json" \\
  -d '{
    "retailerId": "ret_chilanzar_77",
    "deliveryDate": "2026-09-12",
    "lines": [
      {"sku": "SKU-BEV-001", "quantity": 50},
      {"sku": "SKU-SNK-042", "quantity": 100}
    ]
  }'`,
      },
    ],
    callouts: [
      {
        type: 'note',
        title: 'Bearer Token Expiration',
        content: 'JWT bearer tokens have a 60-minute lifetime and should be refreshed using the /v1/auth/refresh endpoint before expiration.',
      },
    ],
    relatedTopics: [
      {
        title: 'WebSocket Event Hub',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'websocket-hub',
        reason: 'Subscribe to real-time events emitted when API mutations occur.',
        badge: 'NEXT STEP',
      },
      {
        title: 'Kafka Schema Contracts',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'kafka-event-contracts',
        reason: 'Learn about internal event contracts used for asynchronous event processing.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // API: WEBSOCKET HUB
  // ---------------------------------------------------------------------------
  'api/websocket-hub': {
    slug: 'websocket-hub',
    categoryId: 'api',
    title: 'WebSocket Event Hub & Realtime Fanout',
    shortTitle: 'WebSocket Hub',
    badge: 'REALTIME BUS',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '8 min read',
    heroWordmark: ['WebSocket', 'Hub'],
    leadSentence:
      'Low-latency bidirectional WebSocket connection pool delivering live telemetry, order status updates, and dispatch broadcasts.',
    summary:
      'The Pegasus WebSocket Hub maintains persistent, authenticated connections with tens of thousands of active client terminals. Role-based channel subscriptions allow clients to receive only the events relevant to their operational zone, eliminating polling overhead.',
    highlights: [
      { title: 'Role-Scoped Channels', desc: 'Clients subscribe to topics like cell:uz:orders, driver:uuid:route, or warehouse:uuid:dock.', metric: 'Scoped Topics' },
      { title: 'Sub-40ms Delivery', desc: 'Optimized Go Gorilla/Nhooyr WebSocket servers forward events with minimal memory overhead.', metric: '<40ms' },
      { title: 'Heartbeat & Resync', desc: '30-second ping/pong frames with automatic reconnection and missed-event sequence backfill.', metric: 'Auto Resync' },
    ],
    callouts: [
      {
        type: 'tip',
        title: 'Sequence Number Backfill',
        content: 'Every WebSocket event includes an incrementing seq_id. If a client disconnects briefly, it sends its last received seq_id to replay missed messages.',
      },
    ],
    relatedTopics: [
      {
        title: 'Transactional Outbox & Bus',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'transactional-outbox',
        reason: 'Understand how database transactions feed the outbox worker that publishes to the WebSocket hub.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'REST API & Scopes',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'rest-api-reference',
        reason: 'WebSocket authentication tokens are obtained via the standard REST login endpoint.',
        badge: 'RELATED ENGINE',
      },
    ],
  },

  // ---------------------------------------------------------------------------
  // API: KAFKA EVENT CONTRACTS
  // ---------------------------------------------------------------------------
  'api/kafka-event-contracts': {
    slug: 'kafka-event-contracts',
    categoryId: 'api',
    title: 'Kafka Event Contracts & Schema Registry',
    shortTitle: 'Kafka Contracts',
    badge: 'EVENT STREAMING',
    version: 'v4.2.0',
    lastUpdated: 'September 2026',
    readTime: '9 min read',
    heroWordmark: ['Kafka', 'Events'],
    leadSentence:
      'Protobuf and JSON Schema envelope definitions for all asynchronous domain events published across the Pegasus cluster.',
    summary:
      'All internal microservices and data pipelines communicate asynchronously via Kafka topics. To prevent contract drift, schemas are validated against a central Schema Registry with backward compatibility checks enforced in CI/CD.',
    highlights: [
      { title: 'Strict Envelope Structure', desc: 'Standard CloudEvents-compliant metadata header with trace_id, producer, and timestamp.', metric: 'CloudEvents' },
      { title: 'Zero Schema Drift', desc: 'Backward-compatible Protobuf and JSON schemas prevent producer/consumer contract breakages.', metric: 'CI Enforced' },
      { title: 'Dead-Letter Queues (DLQ)', desc: 'Poison pill messages automatically route to designated DLQs with automated alerting and replay tooling.', metric: 'DLQ Safe' },
    ],
    callouts: [
      {
        type: 'important',
        title: 'Partition Keying Law',
        content: 'Always specify the partition key as supplier_id or warehouse_id. Never publish order events without a partition key, as unordered processing will break state machines.',
      },
    ],
    relatedTopics: [
      {
        title: 'Transactional Outbox & Bus',
        category: 'Protocols & Engines',
        categoryId: 'protocols',
        slug: 'transactional-outbox',
        reason: 'Learn how the Go outbox daemon pulls events from the database and produces to these Kafka topics.',
        badge: 'PREREQUISITE',
      },
      {
        title: 'WebSocket Event Hub',
        category: 'API & Realtime Events',
        categoryId: 'api',
        slug: 'websocket-hub',
        reason: 'The WebSocket hub acts as a high-speed Kafka consumer bridge to front-end web and mobile applications.',
        badge: 'NEXT STEP',
      },
    ],
  },
};

export const DEFAULT_DOC_ARTICLE = DOC_ARTICLES['guides/introduction'];

export function getDocArticle(categoryId: string, slug: string): DocArticle | undefined {
  return DOC_ARTICLES[`${categoryId}/${slug}`];
}

export function getAllDocSlugs(): Array<{ category: string; slug: string }> {
  const result: Array<{ category: string; slug: string }> = [];
  for (const cat of DOC_CATEGORIES) {
    for (const art of cat.articles) {
      result.push({ category: cat.id, slug: art.slug });
    }
  }
  return result;
}

export function searchDocs(query: string): Array<{
  category: DocCategory;
  article: DocArticle;
  matchedField: string;
}> {
  if (!query || query.trim().length === 0) return [];
  const q = query.toLowerCase().trim();
  const matches: Array<{
    category: DocCategory;
    article: DocArticle;
    matchedField: string;
  }> = [];

  for (const cat of DOC_CATEGORIES) {
    for (const artMeta of cat.articles) {
      const art = DOC_ARTICLES[`${cat.id}/${artMeta.slug}`];
      if (!art) continue;

      if (art.title.toLowerCase().includes(q)) {
        matches.push({ category: cat, article: art, matchedField: 'Title' });
      } else if (art.leadSentence.toLowerCase().includes(q)) {
        matches.push({ category: cat, article: art, matchedField: 'Lead' });
      } else if (art.summary.toLowerCase().includes(q)) {
        matches.push({ category: cat, article: art, matchedField: 'Summary' });
      } else if (art.badge && art.badge.toLowerCase().includes(q)) {
        matches.push({ category: cat, article: art, matchedField: 'Badge' });
      } else if (art.highlights.some(h => h.title.toLowerCase().includes(q) || h.desc.toLowerCase().includes(q))) {
        matches.push({ category: cat, article: art, matchedField: 'Highlights' });
      }
    }
  }

  return matches;
}
