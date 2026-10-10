import type { FlowVariant } from '@/app/data/topicTypes';
import type { ComponentType, ReactNode } from 'react';
import {
  Boxes,
  Building2,
  Cpu,
  Database,
  Factory,
  Layers,
  Lock,
  Navigation,
  Network,
  Radio,
  Scale,
  ShieldCheck,
  Store,
  Terminal,
  Truck,
  Users,
  Workflow,
  type LucideIcon,
} from 'lucide-react';

export type HubVisualType = 'kpi' | 'flow' | 'fleet' | 'metrics' | 'devices' | 'none';

export interface DifferentiatorCardConfig {
  id?: string;
  title: string;
  description: string;
  icon?: LucideIcon | ComponentType<{ className?: string }>;
  badge?: string;
  kicker?: string;
  colSpan?: 1 | 2;
  featured?: boolean;
  previewType?: 'graph' | 'telemetry' | 'code' | 'metric';
  previewValue?: string;
  href?: string;
}

export interface BusinessValueStatConfig {
  value: string;
  label: string;
  context?: string;
  subtext?: string;
  trend?: string;
  delta?: string;
}

export interface BusinessValueTabConfig {
  id: string;
  label: string;
  stats: BusinessValueStatConfig[];
}

export interface CapabilityItemConfig {
  id?: string;
  title: string;
  description: string;
  href?: string;
  image?: string;
  tag?: string;
  workflowSteps?: string[];
  sla?: string;
}

export interface FaqItemConfig {
  id: string;
  question: string;
  answer: string;
  category?: string;
  tag?: string;
  highlights?: string[];
}

export type HubLayoutConfig = {
  visual: HubVisualType;
  flowVariant?: FlowVariant;
  visualTitle?: string;
  visualSubtitle?: string;
  topicGridLayout?: 'uniform' | 'masonry' | 'featured';
  showPromoInHero?: boolean;
  hidePromoBody?: boolean;
  showFleetBand?: boolean;
  heroVisual?: boolean;
  laneLabel?: string;
  laneIndex?: string;
  intro?: { eyebrow: string; title: string; body: string };
  businessValue?: {
    kicker?: string;
    title?: string;
    description?: string;
    tabs?: BusinessValueTabConfig[];
  };
  faq?: {
    kicker?: string;
    title?: string;
    description?: string;
    items?: FaqItemConfig[];
    allowMultiple?: boolean;
  } | boolean;
  differentiators?: {
    kicker?: string;
    title?: string;
    description?: string;
    cards?: DifferentiatorCardConfig[];
  };
  capabilities?: {
    kicker?: string;
    title?: string;
    description?: string;
    items?: CapabilityItemConfig[];
  };
  cta?: ReactNode;
};

// ============================================================================
// 1. PLATFORM HUB CONFIGURATION (ENGLISH)
// ============================================================================

const PLATFORM_DIFFERENTIATORS_EN: DifferentiatorCardConfig[] = [
  {
    icon: Network,
    badge: 'UNIFIED STATE',
    kicker: '[01 // CONTROL PLANE]',
    title: 'Universal 6-Role Execution Surface',
    description:
      'Eliminate siloed portals. Suppliers, warehouses, factories, line-haul carriers, last-mile drivers, and retail receivers operate on a single shared state machine. Every order transition is mutually verified with zero reconciliation delay.',
    previewType: 'telemetry',
    previewValue: '<18ms Mutex Lock',
  },
  {
    icon: Database,
    badge: 'ACID LEVEL-3',
    kicker: '[02 // STORAGE ENGINE]',
    title: 'Deterministic Storage & Transactional Outbox',
    description:
      'Strict serializability backed by Google Cloud Spanner distributed transactions. Mutations write state transitions and transactional outbox events atomically within the same commit, guaranteeing zero lost events and zero phantom inventory.',
    previewType: 'metric',
    previewValue: '99.999% Durability',
  },
  {
    icon: Cpu,
    badge: 'REAL-TIME BUS',
    kicker: '[03 // STREAMING PIPELINE]',
    title: 'Sub-Second Reactive Event Fanout',
    description:
      'High-throughput Go concurrency runtime with Kafka streaming pipelines. Millions of telemetry pings, dispatch changes, and gate transitions are broadcast to connected client dashboards with sub-85 millisecond p99 latency.',
    previewType: 'telemetry',
    previewValue: '<85ms p99 Fanout',
  },
  {
    icon: ShieldCheck,
    badge: 'ZERO DATA BLEED',
    kicker: '[04 // SECURITY PERIMETER]',
    title: 'Cryptographic Tenant & Vault Segregation',
    description:
      'Every query, mutation, and memory vector is bound to cryptographic JWT claims with Spanner row-level security. Competing brands sharing regional 3PL hubs never see, infer, or cross-reference counterpart routes, rates, or stock volumes.',
    previewType: 'code',
    previewValue: 'AES-256 + JWT Claims',
  },
  {
    icon: Scale,
    badge: 'AUTONOMOUS HARD-GATE',
    kicker: '[05 // EXCEPTION ENGINE]',
    title: 'Deterministic Hard-Gates & Freeze-Locks',
    description:
      'Automated guardrails enforce payment confirmations before dispatch authorization, trigger smart-fit overflow handling upon capacity limits, and enforce freeze-locks when order manifests enter staging gates.',
    previewType: 'graph',
    previewValue: '0% Ghost Dispatches',
  },
  {
    icon: Boxes,
    badge: 'LOCAL SQLITE',
    kicker: '[06 // EDGE RESILIENCE]',
    title: 'Offline-First Synchronous Edge Replay',
    description:
      'Driver mobile applications and remote warehouse handhelds continue scanning, signing, and routing during total cellular network blackouts. Transactions queue in local encrypted SQLite and drain through idempotent outbox upon reconnect.',
    previewType: 'metric',
    previewValue: '100% Zero-Loss Drain',
  },
];

const PLATFORM_BUSINESS_VALUE_EN: BusinessValueTabConfig[] = [
  {
    id: 'network',
    label: 'Network Sync',
    stats: [
      {
        value: '60%',
        label: 'Stock-Out Reduction',
        delta: '-60.4% OOS',
        subtext: 'Automated re-order triggers',
        context: 'Real-time synchronization across supplier inventory and regional distribution centers prevents stockouts.',
        trend: 'up',
      },
      {
        value: '+53%',
        label: 'On-Time In-Full (OTIF)',
        delta: '+53.2% Fulfillment',
        subtext: 'Measured across 1.4M shipments',
        context: 'Unified scheduling and live route adjustments ensure deliveries arrive within promised customer windows.',
        trend: 'up',
      },
      {
        value: '<85ms',
        label: 'P99 Event Latency',
        delta: '-74% vs REST polling',
        subtext: 'Kafka outbox to WebSocket',
        context: 'State transitions propagate across all six network role screens in sub-second real-time.',
        trend: 'up',
      },
      {
        value: '6',
        label: 'Unified Role Surfaces',
        delta: '100% Single Truth',
        subtext: 'Supplier, DC, Driver, Retailer',
        context: 'One authoritative database record shared across enterprise roles without manual data re-entry.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'capital',
    label: 'Working Capital',
    stats: [
      {
        value: '-$4.8M',
        label: 'Working Capital Released',
        delta: '-22.8% Tied Inventory',
        subtext: 'Average enterprise saving',
        context: 'Dynamic safety stock algorithms eliminate buffer stockpiles across regional distribution nodes.',
        trend: 'up',
      },
      {
        value: '0 Days',
        label: 'Reconciliation Lag',
        delta: '-100% Audit Disputes',
        subtext: 'Instant cryptographic settlement',
        context: 'Digital proofs of delivery trigger automated invoicing, eliminating manual end-of-month reconciliations.',
        trend: 'up',
      },
      {
        value: '14 Days',
        label: 'Cash Cycle Acceleration',
        delta: '-14 Days DSO',
        subtext: 'Days Sales Outstanding',
        context: 'Pay-at-delivery escrow and immediate dispatch confirmation accelerate trade credit cycles.',
        trend: 'up',
      },
      {
        value: '99.98%',
        label: 'Billing Accuracy',
        delta: '+8.2% Margin Capture',
        subtext: 'Zero undetected leakages',
        context: 'Automated freight rate rating and dimensional verification eliminate carrier accessorial invoice disputes.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'resilience',
    label: 'Uptime & Reliability',
    stats: [
      {
        value: '99.999%',
        label: 'Database Availability SLA',
        delta: 'Multi-Region Spanner',
        subtext: 'Zero maintenance windows',
        context: 'Google Cloud Spanner globally distributed consensus guarantees continuous operations during zone failures.',
        trend: 'up',
      },
      {
        value: '0%',
        label: 'Data Loss on Offline Drop',
        delta: 'Zero Dropped Scans',
        subtext: 'Idempotent SQLite sync',
        context: 'Offline handheld transactions buffer locally and drain deterministically upon reconnect without dropped events.',
        trend: 'up',
      },
      {
        value: '<2s',
        label: 'Dynamic Re-plan Latency',
        delta: '-88% Dispatch Overhead',
        subtext: 'CVRP constraint solver',
        context: 'Immediate route recalculation when emergency manifests, weather bottlenecks, or vehicle breakdowns occur.',
        trend: 'up',
      },
      {
        value: '0',
        label: 'Race Condition Stockouts',
        delta: 'ACID Level 3 Guarantees',
        subtext: 'Distributed mutex locking',
        context: 'High-concurrency order spikes never oversell inventory due to row-level distributed mutex reservations.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'scalability',
    label: 'Enterprise Scale',
    stats: [
      {
        value: '250k+',
        label: 'Concurrent Orders / Sec',
        delta: 'Linear Cluster Scaling',
        subtext: 'Horizontal Go pods',
        context: 'Go concurrency model with goroutine worker pools handles national peak holiday freight surges effortlessly.',
        trend: 'up',
      },
      {
        value: '-68%',
        label: 'Manual Dispatch Overhead',
        delta: 'Touchless Automation',
        subtext: 'Rules + CVRP optimization',
        context: '92% of regular line-haul and last-mile shipments are planned, packed, and assigned touchlessly.',
        trend: 'up',
      },
      {
        value: '30+',
        label: 'Turnkey ERP Connectors',
        delta: 'REST, gRPC, EDIFACT',
        subtext: 'Pre-built SAP/Oracle/1C bridges',
        context: 'Zero-rip deployment integrates with enterprise master records in days rather than quarters.',
        trend: 'up',
      },
      {
        value: '12.4x',
        label: 'Network Simulation Velocity',
        delta: 'Real-time Digital Twin',
        subtext: 'Multi-horizon scenario tests',
        context: 'Simulate fuel price spikes, carrier strikes, or DC closures across historical freight graphs in seconds.',
        trend: 'up',
      },
    ],
  },
];

const PLATFORM_CAPABILITIES_EN: CapabilityItemConfig[] = [
  {
    id: 'atomos-control-plane',
    title: 'Autonomous Multi-Tier Order Orchestration',
    description:
      'Real-time order state machine ingests orders from SAP/Oracle, verifies credit and physical stock with row-level locks, computes delivery fulfillment windows, and issues atomic dispatch instructions to warehouse staging lanes.',
    tag: 'ORDER LIFECYCLE',
    sla: '<120ms End-to-End',
    href: '/platform/atomos-control-plane',
    image: '/Unknown-8.jpg',
    workflowSteps: ['01 Ingest ERP Order', '02 Distributed Lock', '03 Smart Fit Routing', '04 Outbox Event Relay'],
  },
  {
    id: 'network-topology',
    title: 'Multi-Tenant Inventory Partitioning & Cross-Dock',
    description:
      'Enables competing manufacturers to share physical cold-chain and distribution terminals. Cryptographic tenant claims isolate pallet metadata, automated cross-dock sorting gates, and temperature logs with zero visibility leaks.',
    tag: 'INVENTORY CORE',
    sla: 'Zero-Bleed Isolation',
    href: '/platform/network-topology',
    image: '/Unknown-10.jpg',
    workflowSteps: ['01 Inbound Scan', '02 Tenant JWT Verification', '03 Cross-Dock Allocation', '04 Seal Authorization'],
  },
  {
    id: 'order-lifecycle',
    title: 'Autonomous Fleet Dispatch & CVRP Re-Routing',
    description:
      'Continuous vehicle routing problem (CVRP) solver with time windows, volumetric axle limits, and driver rest constraints. Re-optimizes active delivery manifests dynamically when traffic bottlenecks or retailer delays arise.',
    tag: 'DISPATCH ENGINE',
    sla: '<450ms Dynamic CVRP Solve',
    href: '/platform/order-lifecycle',
    image: '/Unknown-5.jpg',
    workflowSteps: ['01 Constraint Check', '02 Heuristic CVRP Solve', '03 Mobile Manifest Push', '04 Live GPS Telemetry'],
  },
  {
    id: 'trust-reliability',
    title: 'High-Velocity Payment Hard-Gates & Instant Invoicing',
    description:
      'Eliminates payment risk and weeks of invoice reconciliation. Delivery completion triggers cryptographic biometric signature and photo ePOD, releasing escrow funds and posting general ledger entries instantly to corporate ERPs.',
    tag: 'FINANCIAL ENGINE',
    sla: '100% Atomic Settlement',
    href: '/platform/trust-reliability',
    image: '/Unknown-6.jpg',
    workflowSteps: ['01 Geo-Fence Arrival', '02 Biometric / QR ePOD', '03 Escrow Release', '04 Real-Time ERP Posting'],
  },
];

const PLATFORM_FAQS_EN: FaqItemConfig[] = [
  {
    id: 'platform-erp-integration',
    category: 'Integration',
    tag: 'ERP / WMS CONNECTORS',
    question: 'How does Pegasus connect with legacy enterprise ERPs (SAP, Oracle SCM, 1C:Enterprise)?',
    answer:
      'Pegasus acts as a high-velocity execution and dispatch overlay via non-invasive bidirectional connectors. Master data (product catalogs, customer credit limits, vendor master files) syncs through REST or gRPC APIs, while operational transactional events stream via Apache Kafka. Legacy ERPs remain the authoritative general ledger, while Pegasus executes real-time route optimization, dock scheduling, and atomic state synchronization without risky migrations.',
    highlights: [
      'Bidirectional REST, gRPC & Kafka connectors',
      'Zero disruption to existing ERP general ledgers',
      'Real-time master data synchronization',
    ],
  },
  {
    id: 'platform-concurrency-locks',
    category: 'Architecture',
    tag: 'ACID & OUTBOX PATTERN',
    question: 'How does the platform maintain strict transactional consistency during high-concurrency order spikes?',
    answer:
      'Pegasus combines Google Cloud Spanner distributed ACID transactions with the Transactional Outbox pattern. State changes, inventory locks, and event outbox entries are committed atomically in a single multi-region consensus round. Even during massive promotional spikes or flash freight allocations, row-level distributed mutexes prevent overselling or race conditions, ensuring deterministic execution across all distributed nodes.',
    highlights: [
      'Cloud Spanner distributed ACID consensus',
      'Zero-overselling distributed mutex locks',
      'Transactional outbox prevents event drops',
    ],
  },
  {
    id: 'platform-tenant-isolation',
    category: 'Security',
    tag: 'CRYPTOGRAPHIC ISOLATION',
    question: 'How does Pegasus guarantee cryptographic tenant isolation in shared 3PL warehouse operations?',
    answer:
      'Multi-tenancy is enforced cryptographically at the database and application layers. Every incoming request must provide a signed JWT specifying tenant, role, and permission scopes. Cloud Spanner applies row-level security policies and tenant-isolated partitions. Cross-tenant queries are blocked at the ORM/SQL compiler level, guaranteeing that competing suppliers sharing a warehouse facility have zero visibility into each other’s pallets, orders, or freight tariffs.',
    highlights: [
      'Strict JWT cryptographic token verification',
      'Cloud Spanner row-level security partitions',
      'Zero visibility bleed across competing tenants',
    ],
  },
  {
    id: 'platform-offline-sync',
    category: 'Reliability',
    tag: 'OFFLINE-FIRST EDGE',
    question: 'What mechanisms protect mobile drivers and warehouse staff from cellular connectivity dropouts?',
    answer:
      'Pegasus mobile and handheld applications employ an offline-first architecture with local encrypted SQLite databases. When drivers enter cellular dead zones or underground loading docks, barcode scans, signature captures, and GPS status changes are stored locally in an append-only transaction log. Once a connection is re-established, an idempotent replay pipeline synchronizes the events with cryptographic sequence validation, preventing duplicate mutations.',
    highlights: [
      'Local encrypted SQLite transaction queue',
      'Deterministic idempotent event replay',
      'Zero lost deliveries or scan dropouts',
    ],
  },
  {
    id: 'platform-routing-engine',
    category: 'Algorithms',
    tag: 'CVRPTW ENGINE',
    question: 'How does the Pegasus routing engine handle real-time dynamic re-planning during active shifts?',
    answer:
      'Our engine formulates logistics dispatch as a Capacitated Vehicle Routing Problem with Time Windows (CVRPTW). It utilizes modern Clarke-Wright heuristics accelerated by GPU parallel tabu search algorithms. When traffic congestion, retailer closures, or urgent order insertions occur, re-plan jobs solve across thousands of candidate delivery paths within 450 milliseconds, pushing turn-by-turn re-routed stops to driver navigation instantly.',
    highlights: [
      'Sub-450ms heuristic constraint solve',
      'Dynamic real-time traffic and window re-plan',
      'Instant push to driver navigation displays',
    ],
  },
  {
    id: 'platform-compliance-audit',
    category: 'Compliance',
    tag: 'SOC-2 & IMMUTABLE AUDIT',
    question: 'Does the platform meet enterprise compliance standards for audit logging and financial governance?',
    answer:
      'Yes. Pegasus maintains an immutable append-only event ledger tracking every state change, user intervention, override reason, and biometric verification. All data in transit is encrypted using TLS 1.3, and data at rest is secured with customer-managed encryption keys (CMEK) via AES-256. The platform adheres to SOC-2 Type II standards, ISO/IEC 27001 certifications, and GDPR/CCPA data privacy regulations.',
    highlights: [
      'SOC-2 Type II & ISO 27001 certified',
      'Immutable cryptographic audit trail',
      'Customer-managed CMEK encryption',
    ],
  },
  {
    id: 'platform-rollout-timeline',
    category: 'Rollout',
    tag: 'PHASED 90-DAY DEPLOYMENT',
    question: 'What is the typical enterprise deployment timeline and sandbox onboarding process?',
    answer:
      'A full enterprise deployment typically spans 60 to 90 days across three non-disruptive phases. Phase 1 (Weeks 1–4) involves API integration with existing ERP/WMS systems and shadow inventory modeling. Phase 2 (Weeks 5–8) conducts a pilot launch at 2–3 regional distribution centers with driver and warehouse apps. Phase 3 (Weeks 9–12) rolls out network-wide with automated financial settlement hard-gates, backed by a dedicated 24/7 technical account team.',
    highlights: [
      'Non-disruptive 3-phase rollout strategy',
      'Shadow mode verification before cutover',
      'Dedicated 24/7 enterprise engineering SLA',
    ],
  },
];

// ============================================================================
// 2. CAPABILITIES HUB CONFIGURATION (ENGLISH)
// ============================================================================

const CAPABILITIES_DIFFERENTIATORS_EN: DifferentiatorCardConfig[] = [
  {
    icon: Navigation,
    badge: 'SUB-SECOND SOLVER',
    kicker: '[01 // ROUTE INTELLIGENCE]',
    title: 'Constraint-Aware Fleet Routing (CVRPTW)',
    description:
      'Solves multi-depot, multi-vehicle routing problems with complex real-world constraints: vehicle weight/volume capacities, strict retailer receiving hours, driver hours-of-service regulations, and live road network congestion.',
    previewType: 'telemetry',
    previewValue: '<450ms Solve Time',
  },
  {
    icon: Boxes,
    badge: '3D BIN PACKING',
    kicker: '[02 // CARGO OPTIMIZATION]',
    title: 'Volumetric 3D Load Optimization',
    description:
      'Maximizes trailer cube utilization while maintaining axle weight distribution, carton crush limits, and delivery stop unloading order. Eliminates cargo damage and cuts required fleet tractor runs by up to 24%.',
    previewType: 'metric',
    previewValue: '94.2% Cube Utilization',
  },
  {
    icon: Truck,
    badge: 'DYNAMIC FLEET',
    kicker: '[03 // CAPACITY ALLOCATION]',
    title: 'Smart-Fit Dynamic Overflow & 3PL Brokering',
    description:
      'When freight volumes exceed owned fleet capacity, Pegasus automatically calculates the marginal cost and routes excess volume to vetted 3PL carrier spot-market partners via automated rate tenders and electronic booking.',
    previewType: 'graph',
    previewValue: '100% Capacity Cover',
  },
  {
    icon: ShieldCheck,
    badge: 'FRAUD PROOF',
    kicker: '[04 // LAST-MILE TRUST]',
    title: 'Cryptographic ePOD & Payment Hard-Gates',
    description:
      'Enforces geotagged camera scans, barcode parity validation, and customer digital signatures before freight custody transfers. Releases escrow payments and triggers instant electronic proof of delivery.',
    previewType: 'code',
    previewValue: 'GPS + Biometric Lock',
  },
  {
    icon: Cpu,
    badge: 'CAN-BUS & IOT',
    kicker: '[05 // SENSOR TELEMETRY]',
    title: 'Live CAN-Bus & Cold-Chain Monitoring',
    description:
      'Ingests vehicle OBD-II/CAN-bus diagnostics, fuel burn curves, and IoT temperature/humidity sensors at 5-second intervals. Automatically alerts dispatchers and triggers route diversions upon thermal thresholds.',
    previewType: 'telemetry',
    previewValue: '±0.2°C Temp Precision',
  },
  {
    icon: Layers,
    badge: 'ZERO DWELL TIME',
    kicker: '[06 // TERMINAL OPS]',
    title: 'Dynamic Cross-Dock Staging & Gate Allocation',
    description:
      'Coordinates inbound line-haul trailer arrival times directly with outbound delivery van loading docks. Eliminates intermediate warehouse put-away steps and slashes terminal freight dwell times by over 50%.',
    previewType: 'metric',
    previewValue: '-52% Terminal Dwell',
  },
];

const CAPABILITIES_BUSINESS_VALUE_EN: BusinessValueTabConfig[] = [
  {
    id: 'dispatch',
    label: 'Dispatch & Routing',
    stats: [
      {
        value: '-24.6%',
        label: 'Fleet Mileage Reduction',
        delta: '-24.6% Empty Miles',
        subtext: 'Clarke-Wright CVRPTW solver',
        context: 'Algorithmic route consolidation eliminates overlapping routes and backhaul empty miles across fleet operations.',
        trend: 'down',
      },
      {
        value: '98.4%',
        label: 'Automated Dispatch Rate',
        delta: '+41.2% Touchless',
        subtext: 'Zero human dispatcher touch',
        context: 'Orders automatically group into vehicle manifests based on capacity, delivery windows, and traffic curves.',
        trend: 'up',
      },
      {
        value: '<450ms',
        label: 'Dynamic Re-Route Time',
        delta: 'Instant Push to Fleet',
        subtext: 'Live turn-by-turn update',
        context: 'Sub-second route recomputation preserves SLA promises during unexpected traffic bottlenecks or weather events.',
        trend: 'up',
      },
      {
        value: '+31%',
        label: 'Stops Delivered Per Shift',
        delta: '+3.8 Stops / Driver Day',
        subtext: 'Optimized delivery density',
        context: 'Clustered drop sequencing reduces vehicle parking search times and driver walking intervals in urban cores.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'packing',
    label: '3D Cube & Safety',
    stats: [
      {
        value: '94.2%',
        label: 'Trailer Cube Utilization',
        delta: '+18.5% vs Manual Packing',
        subtext: '3D heuristic bin-packing',
        context: 'Algorithmic pallet and box orientation fills unused trailer head-space while respecting carton weight limits.',
        trend: 'up',
      },
      {
        value: '-78%',
        label: 'Transit Cargo Damage',
        delta: 'Near-Zero Damage Claims',
        subtext: 'Axle weight & crush limits',
        context: 'Algorithmic loading enforces heavy goods bottom placement and center-of-gravity stabilization automatically.',
        trend: 'down',
      },
      {
        value: '-19.2%',
        label: 'Required Tractor Runs',
        delta: 'Fewer Fleet Trips Needed',
        subtext: 'Consolidated freight loads',
        context: 'Higher vehicle fill rates mean fewer trips are required to move identical monthly shipment tonnage.',
        trend: 'down',
      },
      {
        value: '100%',
        label: 'Unload Sequence Parity',
        delta: 'LIFO Optimization',
        subtext: 'Last-In, First-Out packing',
        context: 'Trailer cargo is stacked in exact reverse delivery stop sequence, eliminating dock repacking delays.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'lastmile',
    label: 'Last-Mile & SLA',
    stats: [
      {
        value: '99.2%',
        label: 'On-Time Delivery Rate',
        delta: '+14.8% SLA Compliance',
        subtext: 'Across 850k urban deliveries',
        context: 'Accurate traffic models and geofence-triggered customer notifications ensure deliveries hit narrow time slots.',
        trend: 'up',
      },
      {
        value: '12 Min',
        label: 'Average Urban Dwell Time',
        delta: '-45% At-Stop Time',
        subtext: 'Mobile barcode & digital sign',
        context: 'Driver handheld scans confirm entire pallets in seconds with instant digital signature and photo proof.',
        trend: 'down',
      },
      {
        value: '0%',
        label: 'Lost Manifest Variance',
        delta: 'Cryptographic Audit Trail',
        subtext: 'Instant chain of custody',
        context: 'Every carton handover requires barcode scan verification, eliminating missing parcel disputes entirely.',
        trend: 'up',
      },
      {
        value: '96.8%',
        label: 'First-Attempt Success Rate',
        delta: '+22.4% Delivery Success',
        subtext: 'Live recipient notifications',
        context: 'Real-time SMS/WhatsApp tracking links with 15-minute ETA accuracy eliminate failed delivery attempts.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'telemetry',
    label: 'Telemetry & IoT',
    stats: [
      {
        value: '5 Sec',
        label: 'GPS & Telemetry Interval',
        delta: 'Sub-Second Edge Buffering',
        subtext: 'Cellular & Satellite hybrid',
        context: 'High-frequency vehicle position, speed, and fuel rate telemetry feeds central operations dashboards in real-time.',
        trend: 'up',
      },
      {
        value: '0.02%',
        label: 'Cold-Chain Excursion Rate',
        delta: '-94% Spoilage Losses',
        subtext: 'BLE multi-sensor monitoring',
        context: 'Real-time cargo bay temperature alerts trigger automated driver intervention before spoilage occurs.',
        trend: 'down',
      },
      {
        value: '-16.4%',
        label: 'Fleet Fuel Consumption',
        delta: 'Eco-Driving & Anti-Idling',
        subtext: 'CAN-bus telemetry coaching',
        context: 'Driver coaching, idling reduction, and optimized route gradients substantially reduce fleet fuel expenditures.',
        trend: 'down',
      },
      {
        value: '99.98%',
        label: 'Sensor Health & Uptime',
        delta: 'Automated Diagnostics',
        subtext: 'Self-healing IoT mesh',
        context: 'Diagnostic heartbeats detect sensor battery drain or tampering before vehicles leave the dispatch yard.',
        trend: 'up',
      },
    ],
  },
];

const CAPABILITIES_SHOWCASE_EN: CapabilityItemConfig[] = [
  {
    id: 'smarter-dispatch',
    title: 'Automated Multi-Vehicle CVRP Dispatching',
    description:
      'Aggregates pending warehouse orders into optimized multi-stop delivery routes. Evaluates vehicle cubic meters, max payload tonnage, driver shift limits, and retailer receiving docks to build mathematically optimal itineraries.',
    tag: 'DYNAMIC ROUTING',
    sla: '<300ms Route Computation',
    href: '/capabilities/smarter-dispatch',
    image: '/Unknown-7.jpg',
    workflowSteps: ['01 Order Pooling', '02 Capacity & Window Check', '03 Heuristic Route Solve', '04 Mobile Driver Push'],
  },
  {
    id: 'reliable-updates',
    title: '3D Truck & Pallet Volumetric Bin-Packing',
    description:
      'Computes 3D loading blueprints for mixed pallet and trailer stowage. Balances trailer axles, enforces vertical stacking limits for fragile goods, and groups cargo in exact reverse-drop order for seamless offloading.',
    tag: '3D PACKING ENGINE',
    sla: '<150ms 3D Orientation Solve',
    href: '/capabilities/reliable-updates',
    image: '/Unknown-11.jpg',
    workflowSteps: ['01 Carton Dimension Intake', '02 Crush Factor Analysis', '03 3D Spatial Layout', '04 Staging Sequencing'],
  },
  {
    id: 'live-fleet-tracking',
    title: 'Smart-Fit Overflow & Dynamic Brokering',
    description:
      'When distribution center volumes exceed private fleet capacity, Pegasus instantly tenders overflow loads to certified logistics partners, selecting optimal freight rates and tracking execution through unified driver links.',
    tag: 'CAPACITY BROKERING',
    sla: '<60s Spot Tender Response',
    href: '/capabilities/live-fleet-tracking',
    image: '/Unknown-5.jpg',
    workflowSteps: ['01 Capacity Deficit Alert', '02 Carrier Rate Scoring', '03 Automated Broadcast', '04 Contract Booking'],
  },
  {
    id: 'payment-confidence',
    title: 'Biometric ePOD & Automated Financial Release',
    description:
      'Protects high-value cargo deliveries with geofence-enforced electronic proof of delivery. Drivers scan barcodes, take inspection photos, and record receiver signatures, triggering immediate escrow payout and invoice generation.',
    tag: 'DIGITAL CUSTODY',
    sla: '100% Real-Time Verification',
    href: '/capabilities/payment-confidence',
    image: '/Unknown-6.jpg',
    workflowSteps: ['01 Geofence Proximity Lock', '02 Itemized Barcode Scan', '03 Photo & Signature ePOD', '04 Settlement Release'],
  },
];

const CAPABILITIES_FAQS_EN: FaqItemConfig[] = [
  {
    id: 'capabilities-routing-algorithms',
    category: 'Algorithms',
    tag: 'CVRPTW SOLVER',
    question: 'What mathematical models power the Pegasus Capacitated Vehicle Routing Problem (CVRP) solver?',
    answer:
      'Our solver engine implements a hybrid metaheuristic approach combining Clarke-Wright savings algorithms with Large Neighborhood Search (LNS) and GPU-accelerated Tabu Search. It simultaneously factors in vehicle weight/volume capacities, dynamic time windows, driver break regulations, multi-depot configurations, and historical traffic speeds, producing near-optimal routing solutions in under 450 milliseconds.',
    highlights: [
      'Clarke-Wright & Large Neighborhood Search (LNS)',
      'GPU-accelerated Tabu Search constraint solver',
      'Sub-450 millisecond solve latency',
    ],
  },
  {
    id: 'capabilities-3d-bin-packing',
    category: 'Operations',
    tag: '3D SPATIAL HEURISTICS',
    question: 'How does 3D load packing account for carton fragility, tilt restrictions, and axle weight limits?',
    answer:
      'Pegasus utilizes a multi-constraint 3D bin-packing heuristic. Each SKU carries physical metadata including dimensions, tare weight, maximum vertical load crush limit, and permitted rotation axes (e.g. liquid containers must remain upright). The algorithm calculates cumulative weight distribution across front and rear trailer axles, ensuring road-legal balance while maximizing volumetric cubic efficiency.',
    highlights: [
      'Crush factor and orientation constraints',
      'Automated front/rear axle weight distribution',
      'Guaranteed road-legal trailer loading',
    ],
  },
  {
    id: 'capabilities-shop-closed-exception',
    category: 'Last-Mile',
    tag: 'EXCEPTION PLAYBOOKS',
    question: 'How does Pegasus handle shop-closed exceptions and delivery re-attempts?',
    answer:
      'If a driver arrives at a retailer location that is closed or unable to receive goods, the driver triggers a "Shop Closed" exception in the mobile application. The driver takes a timestamped, geotagged photo. Pegasus immediately marks the manifest item, alerts the dispatch coordinator, recalculates the driver\'s remaining itinerary without manual intervention, and automatically reschedules the delivery to the next available route window.',
    highlights: [
      'Geotagged photographic exception proof',
      'Dynamic mid-route itinerary recalculation',
      'Automated re-delivery slot scheduling',
    ],
  },
  {
    id: 'capabilities-telematics-integration',
    category: 'Integration',
    tag: 'TELEMATICS & ELD',
    question: 'Can Pegasus integrate with external telematics providers like Geotab, Samsara, and Omnitracs?',
    answer:
      'Yes. Pegasus includes pre-built ingestion connectors for major commercial telematics platforms including Geotab, Samsara, Omnitracs, and standard ELD devices. Vehicle GPS coordinates, speed, odometer, fuel levels, and diagnostic fault codes (DTCs) ingest through Webhook and REST streaming pipelines every 5 to 15 seconds, fusing with active dispatch boards seamlessly.',
    highlights: [
      'Turnkey connectors for Geotab, Samsara & ELD',
      'High-frequency 5–15 second telemetry streaming',
      'Integrated engine diagnostic fault monitoring',
    ],
  },
  {
    id: 'capabilities-cold-chain-monitoring',
    category: 'Cold-Chain',
    tag: 'IOT TEMPERATURE SENSORS',
    question: 'How does the platform monitor cold-chain temperature thresholds during transit?',
    answer:
      'Pegasus connects via Bluetooth Low Energy (BLE) or hardwired reefer telematics to wireless temperature and humidity probes inside refrigerated trailers. If temperatures breach pre-set thresholds (e.g., above 4°C for dairy or -18°C for frozen freight) for longer than 3 minutes, the system triggers an emergency alert to dispatch, notifies the driver via audio prompt, and flags the shipment for quality quarantine inspection upon delivery.',
    highlights: [
      'Continuous BLE wireless temperature probes',
      'Automated acoustic driver alerts upon breach',
      'Digital quality quarantine flags on ePOD',
    ],
  },
  {
    id: 'capabilities-hardware-compatibility',
    category: 'Hardware',
    tag: 'ZEBRA & HONEYWELL SDK',
    question: 'Does the driver app work on ruggedized warehouse Android scanners and consumer smartphones?',
    answer:
      'The Pegasus Driver and Warehouse apps are built as cross-platform native binaries supporting both iOS and Android. They offer native SDK integrations for Zebra, Honeywell, and Datalogic ruggedized handhelds, utilizing hardware laser scanner engines for rapid multi-barcode batch scanning, as well as high-performance camera scanning for standard consumer iPhones and Android devices.',
    highlights: [
      'Native laser scanner integration for Zebra & Honeywell',
      'Camera-based multi-barcode scanning on mobile',
      'Industrial IP67 ruggedized terminal support',
    ],
  },
  {
    id: 'capabilities-epod-security',
    category: 'Security',
    tag: 'CRYPTOGRAPHIC EPOD',
    question: 'How is proof of delivery legally protected against fraud or counterfeit signatures?',
    answer:
      'Every ePOD event recorded in Pegasus captures a SHA-256 cryptographic hash combining the driver\'s GPS location within a verified geofence, a microsecond UTC timestamp, the recipient\'s digital signature vector, and high-resolution delivery photos. This package is saved into an immutable append-only record, generating a tamper-proof PDF delivery certificate with a verifiable QR verification link.',
    highlights: [
      'SHA-256 tamper-proof cryptographic audit hash',
      'Geofence proximity lock & UTC microsecond stamp',
      'Instant verifiable PDF delivery certificate',
    ],
  },
];

// ============================================================================
// 3. PLATFORM HUB CONFIGURATION (RUSSIAN)
// ============================================================================

const PLATFORM_DIFFERENTIATORS_RU: DifferentiatorCardConfig[] = [
  {
    icon: Network,
    badge: 'ЕДИНОЕ СОСТОЯНИЕ',
    kicker: '[01 // ПАНЕЛЬ УПРАВЛЕНИЯ]',
    title: 'Единая операционная среда для 6 ролей',
    description:
      'Устранение разрозненных порталов. Поставщики, склады, фабрики, магистральные перевозчики, водители последней мили и ритейлеры работают в рамках единой детерминированной стейт-машины. Каждое изменение статуса проверяется взаимно и без задержек сверки.',
    previewType: 'telemetry',
    previewValue: '<18ms Захват мьютекса',
  },
  {
    icon: Database,
    badge: 'ACID УРОВЕНЬ-3',
    kicker: '[02 // ДВИЖОК ХРАНЕНИЯ]',
    title: 'Детерминированное ядро и транзакционный Outbox',
    description:
      'Строгая сериализуемость на базе распределенных транзакций Google Cloud Spanner. Мутации состояния заказа и события для шины Kafka записываются атомарно в рамках одного коммита, исключая потерю событий и фантомные остатки.',
    previewType: 'metric',
    previewValue: '99.999% Долговечность',
  },
  {
    icon: Cpu,
    badge: 'РЕАЛЬНОЕ ВРЕМЯ',
    kicker: '[03 // ПОТОКОВЫЙ ПАЙПЛАЙН]',
    title: 'Субсекундная трансляция событий',
    description:
      'Высокопроизводительная параллельная среда Go с потоковыми конвейерами Kafka. Миллионы телеметрических пакетов, изменений диспетчеризации и отметок на КПП транслируются на клиентские дашборды с задержкой p99 менее 85 миллисекунд.',
    previewType: 'telemetry',
    previewValue: '<85ms p99 Задержка',
  },
  {
    icon: ShieldCheck,
    badge: 'НУЛЕВАЯ УТЕЧКА ДАННЫХ',
    kicker: '[04 // ПЕРИМЕТР БЕЗОПАСНОСТИ]',
    title: 'Криптографическая изоляция арендаторов',
    description:
      'Каждый запрос, мутация и вектор в памяти привязаны к криптографическим утверждениям JWT и политикам Row-Level Security в Cloud Spanner. Конкурирующие бренды на общих 3PL-складах никогда не увидят остатки, тарифы или маршруты друг друга.',
    previewType: 'code',
    previewValue: 'AES-256 + JWT Claims',
  },
  {
    icon: Scale,
    badge: 'АВТОМАТИЧЕСКИЙ ШЛЮЗ',
    kicker: '[05 // ДВИЖОК ИСКЛЮЧЕНИЙ]',
    title: 'Детерминированные жесткие шлюзы и Freeze-Lock',
    description:
      'Автоматические предохранители требуют подтверждения оплаты до авторизации погрузки, перенаправляют избыточный объем при переполнении и фиксируют неизменяемость состава заказа при въезде в зону комплектации.',
    previewType: 'graph',
    previewValue: '0% Ошибочных рейсов',
  },
  {
    icon: Boxes,
    badge: 'ЛОКАЛЬНЫЙ SQLITE',
    kicker: '[06 // УСТОЙЧИВОСТЬ КЛИЕНТОВ]',
    title: 'Синхронное воспроизведение Offline-First',
    description:
      'Мобильные приложения водителей и складские ТСД продолжают сканировать, подписывать и вести учет даже при полном обрыве сотовой связи. Транзакции сохраняются в локальной зашифрованной SQLite и идемпотентно синхронизируются при возврате сети.',
    previewType: 'metric',
    previewValue: '100% Без потерь',
  },
];

const PLATFORM_BUSINESS_VALUE_RU: BusinessValueTabConfig[] = [
  {
    id: 'network',
    label: 'Сеть и синхронизация',
    stats: [
      {
        value: '60%',
        label: 'Снижение дефицита на полке',
        delta: '-60.4% OOS',
        subtext: 'Автоматические триггеры дозаказа',
        context: 'Синхронизация остатков поставщика и складов в реальном времени исключает нехватку товаров.',
        trend: 'up',
      },
      {
        value: '+53%',
        label: 'Своевременность доставки (OTIF)',
        delta: '+53.2% Выполнение плана',
        subtext: 'По результатам 1.4 млн доставок',
        context: 'Сквозное расписание и динамическая корректировка гарантируют доставку в согласованные временные окна.',
        trend: 'up',
      },
      {
        value: '<85ms',
        label: 'P99 задержка событий сети',
        delta: '-74% против опроса REST',
        subtext: 'Шина Kafka Outbox к WebSocket',
        context: 'Изменения статусов заказов распространяются по экранам всех 6 ролей быстрее, чем за секунду.',
        trend: 'up',
      },
      {
        value: '6',
        label: 'Единых ролевых интерфейсов',
        delta: '100% Единый источник правды',
        subtext: 'Поставщик, Склад, Водитель, Ритейлер',
        context: 'Поставщик, склад, фабрика, водитель, ритейлер и КПП в единой базе без ручного дублирования.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'capital',
    label: 'Оборотный капитал',
    stats: [
      {
        value: '-$4.8M',
        label: 'Высвобожденный капитал',
        delta: '-22.8% Замороженных запасов',
        subtext: 'Средняя экономия предприятия',
        context: 'Динамические алгоритмы страхового запаса исключают избыточное складирование на распределительных центрах.',
        trend: 'up',
      },
      {
        value: '0 Дней',
        label: 'Задержка финансовой сверки',
        delta: '-100% Спорных расхождений',
        subtext: 'Мгновенные закрывающие документы',
        context: 'Электронные подтверждения доставки формируют закрывающие документы моментально в момент вручения груза.',
        trend: 'up',
      },
      {
        value: '14 Дней',
        label: 'Ускорение оборачиваемости',
        delta: '-14 Дней цикла расчетов',
        subtext: 'Days Sales Outstanding (DSO)',
        context: 'Эскроу-платежи при вручении и мгновенное подтверждение приемки сокращают финансовый цикл факторинга.',
        trend: 'up',
      },
      {
        value: '99.98%',
        label: 'Точность начислений',
        delta: '+8.2% Сохранение маржи',
        subtext: 'Нулевая потеря платежей',
        context: 'Автоматическая верификация габаритов и маршрутных тарифов исключает штрафы и претензии перевозчиков.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'resilience',
    label: 'Надежность и доступность',
    stats: [
      {
        value: '99.999%',
        label: 'Доступность базы данных SLA',
        delta: 'Мультирегиональный Spanner',
        subtext: 'Без технологических окон',
        context: 'Глобально распределенный консенсус Google Cloud Spanner гарантирует непрерывную работу без сервисных окон.',
        trend: 'up',
      },
      {
        value: '0%',
        label: 'Потеря данных в офлайн-зонах',
        delta: 'Ноль потерянных сканов',
        subtext: 'Идемпотентная SQLite очередь',
        context: 'Транзакции на мобильных ТСД буферизуются локально и детерминированно отправляются при появлении сигнала.',
        trend: 'up',
      },
      {
        value: '<2s',
        label: 'Перерасчет маршрутов сети',
        delta: '-88% Ручных корректировок',
        subtext: 'CVRP оптимизатор маршрутов',
        context: 'Мгновенный перерасчет логистических цепочек при возникновении дорожных заторов, задержек или аварий.',
        trend: 'up',
      },
      {
        value: '0',
        label: 'Ошибок оверселлинга',
        delta: 'Гарантии ACID уровня 3',
        subtext: 'Распределенные мьютексы',
        context: 'Пиковые всплески заказов никогда не приводят к перепродаже благодаря распределенным мьютексам на уровне строк.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'scalability',
    label: 'Масштабируемость',
    stats: [
      {
        value: '250k+',
        label: 'Заказов в секунду',
        delta: 'Линейное масштабирование',
        subtext: 'Горизонтальные Go поды',
        context: 'Параллельная модель Go с пулами горутин легко выдерживает пиковые национальные сезонные нагрузки.',
        trend: 'up',
      },
      {
        value: '-68%',
        label: 'Затраты на диспетчеризацию',
        delta: 'Автоматизация без операторов',
        subtext: 'Правила + CVRP маршрутизация',
        context: '92% регулярных магистральных и городских рейсов формируются и распределяются полностью автоматически.',
        trend: 'up',
      },
      {
        value: '30+',
        label: 'Готовых ERP-коннекторов',
        delta: 'REST, gRPC, 1C, SAP, Oracle',
        subtext: 'Быстрые корпоративные шлюзы',
        context: 'Интеграция с существующими корпоративными мастер-системами занимает дни вместо долгих месяцев.',
        trend: 'up',
      },
      {
        value: '12.4x',
        label: 'Скорость симуляции сети',
        delta: 'Цифровой двойник сети',
        subtext: 'Сценарное моделирование',
        context: 'Моделирование скачков цен на топливо, закрытия складов или забастовок на исторических графах за секунды.',
        trend: 'up',
      },
    ],
  },
];

const PLATFORM_CAPABILITIES_RU: CapabilityItemConfig[] = [
  {
    id: 'atomos-control-plane',
    title: 'Автономная оркестрация мульти-эшелонных заказов',
    description:
      'Стейт-машина заказов в реальном времени принимает заявки из SAP/1C, валидирует лимиты и складские остатки блокировками строк Spanner, рассчитывает окна поставки и выдает атомарные задания на комплектацию.',
    tag: 'ЖИЗНЕННЫЙ ЦИКЛ ЗАКАЗА',
    sla: '<120ms Сквозное исполнение',
    href: '/platform/atomos-control-plane',
    image: '/Unknown-8.jpg',
    workflowSteps: ['01 Импорт из ERP', '02 Распределенный Lock', '03 Smart Fit Маршрутизация', '04 Трансляция Outbox'],
  },
  {
    id: 'network-topology',
    title: 'Мульти-арендное партиционирование и Cross-Dock',
    description:
      'Позволяет конкурирующим производителям использовать единые распределительные хабы. Криптографические claims изолируют данные паллет, автоматическую сортировку ворот и логи температурных датчиков без утечек.',
    tag: 'УПРАВЛЕНИЕ ЗАПАСАМИ',
    sla: 'Криптографическая изоляция',
    href: '/platform/network-topology',
    image: '/Unknown-10.jpg',
    workflowSteps: ['01 Приемка и сканирование', '02 Проверка JWT арендатора', '03 Размещение Cross-Dock', '04 Пломбирование ворот'],
  },
  {
    id: 'order-lifecycle',
    title: 'Автономная диспетчеризация и перестроение маршрутов',
    description:
      'Алгоритмический оптимизатор маршрутизации с временными окнами, весогабаритными нормами и режимом труда водителей. Пересчитывает активные маршрутные листы при возникновении заторов или смещении окон приемки.',
    tag: 'ДИСПЕТЧЕРСКИЙ ДВИЖОК',
    sla: '<450ms Динамический расчет CVRP',
    href: '/platform/order-lifecycle',
    image: '/Unknown-5.jpg',
    workflowSteps: ['01 Проверка ограничений', '02 Решение задачи CVRPTW', '03 Отправка в приложение', '04 Живая телеметрия GPS'],
  },
  {
    id: 'trust-reliability',
    title: 'Высокоскоростные платежные шлюзы и авто-инвойсинг',
    description:
      'Устраняет кредитные риски и задержки сверки счетов. Факт вручения груза подтверждается криптографической подписью и фото-ePOD, освобождая средства из эскроу и мгновенно регистрируя проводки в главной книге ERP.',
    tag: 'ФИНАНСОВЫЙ КОНТУР',
    sla: '100% Атомарные взаиморасчеты',
    href: '/platform/trust-reliability',
    image: '/Unknown-6.jpg',
    workflowSteps: ['01 Геозона точки выгрузки', '02 Биометрия / QR-код ePOD', '03 Разблокировка эскроу', '04 Проводка в ERP'],
  },
];

const PLATFORM_FAQS_RU: FaqItemConfig[] = [
  {
    id: 'platform-erp-integration-ru',
    category: 'Интеграция',
    tag: 'ERP / WMS КОННЕКТОРЫ',
    question: 'Как Pegasus интегрируется с существующими ERP-системами (1C:Предприятие, SAP, Oracle SCM)?',
    answer:
      'Pegasus работает как высокоскоростной контур исполнения и диспетчеризации поверх существующих систем через неинвазивные двунаправленные коннекторы. Мастер-данные (номенклатура, контрагенты, лимиты) синхронизируются через REST и gRPC, а транзакционные операционные события передаются через Apache Kafka. Корпоративная ERP сохраняет статус главной бухгалтерской книги, в то время как Pegasus берет на себя субсекундную диспетчеризацию, геоконтроль и атомарную фиксацию остатков.',
    highlights: [
      'Двунаправленные коннекторы REST, gRPC и Kafka',
      'Полная сохранность существующих проводок в ERP',
      'Синхронизация мастер-данных в реальном времени',
    ],
  },
  {
    id: 'platform-concurrency-locks-ru',
    category: 'Архитектура',
    tag: 'ACID И OUTBOX ПАТТЕРН',
    question: 'Как платформа обеспечивает строгую транзакционную согласованность при пиковых нагрузках?',
    answer:
      'Архитектура Pegasus объединяет распределенные ACID-транзакции Google Cloud Spanner с паттерном Transactional Outbox. Изменение статуса заказа, резервирование остатка на складе и регистрация исходящего события в Outbox коммитятся атомарно в одном раунде консенсуса Spanner. Даже при взрывном наплыве заказов распределенные мьютексы на уровне строк предотвращают состояние гонки и оверселлинг.',
    highlights: [
      'Распределенный ACID-консенсус Cloud Spanner',
      'Защита от оверселлинга мьютексами строк',
      'Transactional Outbox исключает потерю сообщений',
    ],
  },
  {
    id: 'platform-tenant-isolation-ru',
    category: 'Безопасность',
    tag: 'КРИПТО-ИЗОЛЯЦИЯ',
    question: 'Как гарантируется изоляция данных конкурирующих арендаторов на общих 3PL-складах?',
    answer:
      'Многоарендность защищена на уровне базы данных и прикладного ядра. Каждый запрос обязан содержать подписанный JWT-токен с идентификатором арендатора и ролевыми привилегиями. В Cloud Spanner действуют политики Row-Level Security и изолированные партиции таблиц. Запросы к данным других арендаторов блокируются на этапе компиляции запроса, полностью исключая просмотр чужих остатков, тарифов или маршрутов.',
    highlights: [
      'Криптографическая верификация токенов JWT',
      'Изоляция строк Row-Level Security в Spanner',
      'Гарантия нулевой утечки данных между клиентами',
    ],
  },
  {
    id: 'platform-offline-sync-ru',
    category: 'Надежность',
    tag: 'OFFLINE-FIRST ЯДРО',
    question: 'Что происходит при потере мобильной связи во время рейса водителя или на удаленном складе?',
    answer:
      'Мобильные приложения Pegasus для водителей и сотрудников складов построены по принципу Offline-First с локальной зашифрованной базой данных SQLite. При отсутствии сотовой связи все сканирования штрихкодов, электронные подписи получателей и отметки времени записываются в локальный журнал транзакций. При возобновлении связи события идемпотентно отправляются на сервер с проверкой порядковых номеров без дублей и потерь.',
    highlights: [
      'Локальная зашифрованная очередь в SQLite',
      'Идемпотентная репликация событий при связи',
      'Гарантия 100% сохранности сканирований и подписей',
    ],
  },
  {
    id: 'platform-routing-engine-ru',
    category: 'Алгоритмы',
    tag: 'CVRPTW ДВИЖОК',
    question: 'Как диспетчерский движок Pegasus пересчитывает маршруты в режиме реального времени?',
    answer:
      'Маршрутизация строится на математической модели задачи развоза с временными окнами (CVRPTW). Движок использует оптимизированные эвристики Кларка-Райта и параллельный поиск с запретами (Tabu Search) на GPU. При возникновении пробок, перекрытий улиц или добавлении срочных заказов перерасчет оптимальных траекторий для сотен машин занимает менее 450 миллисекунд, моментально обновляя навигацию в смартфонах водителей.',
    highlights: [
      'Расчет ограничений за время менее 450 мс',
      'Динамический перерасчет с учетом дорожного трафика',
      'Мгновенная передача маршрута в навигатор водителя',
    ],
  },
  {
    id: 'platform-compliance-audit-ru',
    category: 'Соответствие',
    tag: 'SOC-2 И АУДИТ',
    question: 'Соответствует ли платформа стандартам аудита безопасности и финансового контроля?',
    answer:
      'Да. Pegasus ведет неизменяемый криптографический журнал аудита (Append-Only Event Ledger), регистрирующий каждую мутацию статуса, ручное вмешательство диспетчера и биометрическую подпись. Все данные при передаче шифруются по протоколу TLS 1.3, а данные в покое — алгоритмом AES-256 с возможностью использования ключей клиента (CMEK). Система сертифицирована по стандартам SOC-2 Type II и ISO/IEC 27001.',
    highlights: [
      'Сертификация по стандартам SOC-2 Type II и ISO 27001',
      'Неизменяемый криптографический журнал событий',
      'Шифрование данных AES-256 с ключами клиента CMEK',
    ],
  },
  {
    id: 'platform-rollout-timeline-ru',
    category: 'Внедрение',
    tag: '90-ДНЕВНЫЙ ПЛАН',
    question: 'Каковы этапы и типичные сроки корпоративного внедрения платформы Pegasus?',
    answer:
      'Комплексное внедрение в распределительную сеть обычно занимает от 60 до 90 дней и проходит в 3 безопасных этапа. Фаза 1 (недели 1–4): интеграция коннекторов ERP/WMS и теневое моделирование остатков. Фаза 2 (недели 5–8): пилотный запуск на 2–3 ключевых распределительных центрах с обучением диспетчеров и водителей. Фаза 3 (недели 9–12): масштабирование на всю сеть с включением автоматических финансовых шлюзов под контролем круглосуточной команды инженеров.',
    highlights: [
      'Безопасный поэтапный переход без остановки операций',
      'Теневой режим верификации до боевого переключения',
      'Выделенная инженерная поддержка SLA 24/7',
    ],
  },
];

// ============================================================================
// 4. CAPABILITIES HUB CONFIGURATION (RUSSIAN)
// ============================================================================

const CAPABILITIES_DIFFERENTIATORS_RU: DifferentiatorCardConfig[] = [
  {
    icon: Navigation,
    badge: 'СУБСЕКУНДНЫЙ СОЛВЕР',
    kicker: '[01 // МАРШРУТНЫЙ ИНТЕЛЛЕКТ]',
    title: 'Маршрутизация парка с учетом ограничений (CVRPTW)',
    description:
      'Решение задачи маршрутизации для нескольких депо и сотен машин с учетом жестких ограничений: весогабариты, окна приемки магазинов, режим труда и отдыха водителей и актуальный трафик.',
    previewType: 'telemetry',
    previewValue: '<450ms Расчет маршрута',
  },
  {
    icon: Boxes,
    badge: '3D BIN PACKING',
    kicker: '[02 // ОПТИМИЗАЦИЯ КУЗОВА]',
    title: '3D-укладка и балансировка нагрузки по осям',
    description:
      'Максимизация коэффициента полезного объема полуприцепа с контролем давления на оси, пределов прочности упаковки и очередности выгрузки на точках (LIFO). Сокращает потребность в рейсах до 24%.',
    previewType: 'metric',
    previewValue: '94.2% Заполнение кузова',
  },
  {
    icon: Truck,
    badge: 'ГИБКИЙ АВТОПАРК',
    kicker: '[03 // РАСПРЕДЕЛЕНИЕ ЕМКОСТИ]',
    title: 'Smart-Fit: динамический оверфлоу и брокеридж',
    description:
      'Когда объем отгрузок превышает возможности собственного парка, Pegasus автоматически вычисляет предельную стоимость и выставляет излишек на спотовый рынок проверенным 3PL-перевозчикам через авто-тендеры.',
    previewType: 'graph',
    previewValue: '100% Покрытие пиков',
  },
  {
    icon: ShieldCheck,
    badge: 'ЗАЩИТА ОТ МОШЕННИЧЕСТВА',
    kicker: '[04 // ДОВЕРИЕ ПОСЛЕДНЕЙ МИЛИ]',
    title: 'Криптографический ePOD и жесткие шлюзы оплаты',
    description:
      'Обязательная фиксация геопозиции выгрузки, сканирование штрихкодов коробок и цифровая подпись клиента до перехода права собственности на товар. Мгновенное снятие эскроу-блокировки платежа.',
    previewType: 'code',
    previewValue: 'GPS + Биометрия',
  },
  {
    icon: Cpu,
    badge: 'CAN-BUS И IOT',
    kicker: '[05 // ДАТЧИКИ И ТЕЛЕМЕТРИЯ]',
    title: 'Телеметрия CAN-шины и холодная цепь IoT',
    description:
      'Сбор диагностических данных CAN-шины тягача, расхода топлива и показателей беспроводных датчиков температуры/влажности каждые 5 секунд. Автоматические оповещения диспетчера при отклонениях от нормы.',
    previewType: 'telemetry',
    previewValue: '±0.2°C Точность температуры',
  },
  {
    icon: Layers,
    badge: 'МИНИМАЛЬНЫЙ ПРОСТОЙ',
    kicker: '[06 // ОПЕРАЦИИ ТЕРМИНАЛА]',
    title: 'Динамическая зона Cross-Dock и привязка ворот',
    description:
      'Синхронизация прибытия магистральных автопоездов напрямую с разгрузочно-погрузочными воротами развозных фургонов. Исключает промежуточное размещение на складе и вдвое ускоряет грузооборот.',
    previewType: 'metric',
    previewValue: '-52% Простой на рампе',
  },
];

const CAPABILITIES_BUSINESS_VALUE_RU: BusinessValueTabConfig[] = [
  {
    id: 'dispatch',
    label: 'Диспетчеризация и маршруты',
    stats: [
      {
        value: '-24.6%',
        label: 'Сокращение холостого пробега',
        delta: '-24.6% Холостых км',
        subtext: 'Солвер Кларка-Райта CVRPTW',
        context: 'Алгоритмическая консолидация исключает дублирующие рейсы и порожние обратные пробеги парка.',
        trend: 'down',
      },
      {
        value: '98.4%',
        label: 'Автоматическая диспетчеризация',
        delta: '+41.2% Без оператора',
        subtext: 'Нулевое ручное вмешательство',
        context: 'Заказы автоматически группируются в рейсы по весу, объему, временным окнам и прогнозу дорожной ситуации.',
        trend: 'up',
      },
      {
        value: '<450ms',
        label: 'Время перерасчета маршрута',
        delta: 'Мгновенный пуш в автопарк',
        subtext: 'Живое обновление навигатора',
        context: 'Субсекундный перерасчет маршрута при заторах или авариях гарантирует сохранение обещаний по доставке.',
        trend: 'up',
      },
      {
        value: '+31%',
        label: 'Точек выгрузки за смену',
        delta: '+3.8 Точек на водителя в день',
        subtext: 'Оптимальная плотность рейса',
        context: 'Кластеризация точек сокращает поиск парковки и пеший переход водителя в плотных городских кварталах.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'packing',
    label: '3D-укладка и безопасность',
    stats: [
      {
        value: '94.2%',
        label: 'Коэффициент заполнения кузова',
        delta: '+18.5% к ручной укладке',
        subtext: '3D эвристическая укладка',
        context: 'Эвристический расчет пространственного размещения паллет и коробок заполняет пустоты под потолком фуры.',
        trend: 'up',
      },
      {
        value: '-78%',
        label: 'Повреждения груза в пути',
        delta: 'Минимум рекламаций по бою',
        subtext: 'Учет нагрузки на оси и ярусы',
        context: 'Контроль давления на нижние ряды коробок и автоматическая стабилизация центра тяжести полуприцепа.',
        trend: 'down',
      },
      {
        value: '-19.2%',
        label: 'Количество необходимых рейсов',
        delta: 'Меньше рейсов на тоннаж',
        subtext: 'Консолидация грузопотока',
        context: 'Повышение плотности загрузки сокращает требуемое количество машин для перевозки того же тоннажа.',
        trend: 'down',
      },
      {
        value: '100%',
        label: 'Соответствие выгрузке (LIFO)',
        delta: 'Оптимизация LIFO',
        subtext: 'Точный обратный порядок',
        context: 'Груз укладывается в строгом обратном порядке точек маршрута, исключая перекладывание товара у борта.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'lastmile',
    label: 'Последняя миля и SLA',
    stats: [
      {
        value: '99.2%',
        label: 'Доставка точно в срок (On-Time)',
        delta: '+14.8% Выполнение SLA',
        subtext: 'По 850k городским рейсам',
        context: 'Точные дорожные модели и автоматические уведомления клиентов обеспечивают попадание в узкие тайм-слоты.',
        trend: 'up',
      },
      {
        value: '12 Мин',
        label: 'Средний простой на точке',
        delta: '-45% Времени выгрузки',
        subtext: 'Штрихкодирование и цифровая подпись',
        context: 'Сканирование штрихкодов паллеты целиком и электронная подпись на ТСД экономят десятки минут на точке.',
        trend: 'down',
      },
      {
        value: '0%',
        label: 'Потери мест в накладной',
        delta: 'Криптографический аудит-след',
        subtext: 'Сквозная цепочка владения',
        context: 'Каждая передача коробки фиксируется сканером, что полностью исключает споры о недостачах.',
        trend: 'up',
      },
      {
        value: '96.8%',
        label: 'Успешность с первой попытки',
        delta: '+22.4% Доставок с 1 раза',
        subtext: 'Информирование получателя',
        context: 'Уведомления в SMS/мессенджерах с точностью ETA до 15 минут исключают ситуации закрытых дверей.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'telemetry',
    label: 'Телеметрия и датчики',
    stats: [
      {
        value: '5 Сек',
        label: 'Интервал телеметрии парка',
        delta: 'Субсекундный бортовой буфер',
        subtext: 'Сотовая связь и спутниковый гибрид',
        context: 'Высокочастотный сбор координат, скорости и расхода топлива питает центральный диспетчерский пункт в реальном времени.',
        trend: 'up',
      },
      {
        value: '0.02%',
        label: 'Отклонения холодной цепи',
        delta: '-94% Потерь от порчи',
        subtext: 'Беспроводные BLE-датчики',
        context: 'Предупреждения о перегреве или переохлаждении в режиме реального времени позволяют спасти скоропортящийся груз.',
        trend: 'down',
      },
      {
        value: '-16.4%',
        label: 'Снижение расхода топлива',
        delta: 'Эко-вождение и контроль ХХ',
        subtext: 'Аналитика CAN-шины тягача',
        context: 'Контроль холостого хода, стиля вождения и профилей уклона дорог заметно снижает счета за дизель.',
        trend: 'down',
      },
      {
        value: '99.98%',
        label: 'Надежность датчиков IoT',
        delta: 'Автоматическая самодиагностика',
        subtext: 'Самовосстанавливающаяся сеть',
        context: 'Диагностический мониторинг выявляет разряд батарей или повреждение BLE-датчиков до выезда с базы.',
        trend: 'up',
      },
    ],
  },
];

const CAPABILITIES_SHOWCASE_RU: CapabilityItemConfig[] = [
  {
    id: 'smarter-dispatch',
    title: 'Автоматическое распределение рейсов CVRPTW',
    description:
      'Агрегирует ожидающие заказы склада в оптимизированные маршруты. Учитывает кубатуру кузова, грузоподъемность, нормы смен водителей и рампы ритейлеров для формирования идеального маршрута.',
    tag: 'ДИНАМИЧЕСКИЙ РОУТИНГ',
    sla: '<300ms Расчет маршрута',
    href: '/capabilities/smarter-dispatch',
    image: '/Unknown-7.jpg',
    workflowSteps: ['01 Пул заказов', '02 Проверка веса и окон', '03 Расчет маршрута CVRPTW', '04 Пуш в приложение водителя'],
  },
  {
    id: 'reliable-updates',
    title: 'Пространственная 3D-укладка паллет и кузовов',
    description:
      'Создает 3D-чертежи укладки для смешанных паллет и полуприцепов. Балансирует нагрузку по осям, защищает хрупкие товары и группирует груз в точном обратном порядке выгрузки.',
    tag: '3D-ДВИЖОК УКЛАДКИ',
    sla: '<150ms 3D-ориентация',
    href: '/capabilities/reliable-updates',
    image: '/Unknown-11.jpg',
    workflowSteps: ['01 Импорт габаритов коробок', '02 Анализ предела нагрузки', '03 Построение 3D-схемы', '04 Очередь на комплектацию'],
  },
  {
    id: 'live-fleet-tracking',
    title: 'Smart-Fit оверфлоу и внешние перевозчики',
    description:
      'При превышении пропускной способности собственного парка Pegasus мгновенно распределяет излишки сертифицированным логистическим партнерам по лучшим тарифам через единый интерфейс.',
    tag: 'БРОКЕРИДЖ ЕМКОСТИ',
    sla: '<60s Спотовый тендер',
    href: '/capabilities/live-fleet-tracking',
    image: '/Unknown-5.jpg',
    workflowSteps: ['01 Сигнал дефицита машин', '02 Оценка тарифов 3PL', '03 Автоматический тендер', '04 Подтверждение контракта'],
  },
  {
    id: 'payment-confidence',
    title: 'Биометрический ePOD и мгновенные выплаты',
    description:
      'Защищает дорогостоящие поставки подтверждением доставки в строгой геозоне. Водитель сканирует штрихкоды, делает фото и фиксирует подпись, запуская мгновенный перевод средств и закрытие накладной.',
    tag: 'ЦИФРОВОЕ ВРУЧЕНИЕ',
    sla: '100% Верификация в реальном времени',
    href: '/capabilities/payment-confidence',
    image: '/Unknown-6.jpg',
    workflowSteps: ['01 Захват геозоны точки', '02 Поштучное сканирование', '03 Фото и цифровая подпись', '04 Проводка платежа'],
  },
];

const CAPABILITIES_FAQS_RU: FaqItemConfig[] = [
  {
    id: 'capabilities-routing-algorithms-ru',
    category: 'Алгоритмы',
    tag: 'CVRPTW СОЛВЕР',
    question: 'Какие математические модели используются для решения задачи CVRPTW?',
    answer:
      'Наш алгоритмический движок объединяет эвристику сбережений Кларка-Райта, метод поиска в больших окрестностях (LNS) и параллельный алгоритм Tabu Search на GPU. Солвер одновременно оптимизирует весовые и объемные ограничения фур, динамические окна доставки, нормы труда водителей, работу нескольких распределительных центров и дорожные пробки, находя решение за время менее 450 миллисекунд.',
    highlights: [
      'Эвристика Кларка-Райта и Large Neighborhood Search',
      'Параллельный алгоритм Tabu Search на GPU',
      'Задержка оптимизации менее 450 миллисекунд',
    ],
  },
  {
    id: 'capabilities-3d-bin-packing-ru',
    category: 'Операции',
    tag: '3D-ЭВРИСТИКА УКЛАДКИ',
    question: 'Как 3D-укладка учитывает хрупкость упаковок, запреты на кантование и осевые нагрузки?',
    answer:
      'Pegasus применяет многофакторную эвристику 3D-упаковки емкостей. Каждая номенклатурная единица содержит физические атрибуты: габариты, вес брутто, максимальное вертикальное давление и допустимые плоскости вращения (например, вертикально для жидкостей). Алгоритм вычисляет развесовку по передней и задней осям полуприцепа, гарантируя допустимые нагрузки на автодорогах при 94% плотности заполнения.',
    highlights: [
      'Ограничения по допустимому давлению на коробки',
      'Автоматический баланс нагрузок по осям автопоезда',
      'Гарантированное соблюдение правил перевозки грузов',
    ],
  },
  {
    id: 'capabilities-shop-closed-exception-ru',
    category: 'Последняя миля',
    tag: 'ОБРАБОТКА ИСКЛЮЧЕНИЙ',
    question: 'Как в Pegasus обрабатываются ситуации «Магазин закрыт» и повторные доставки?',
    answer:
      'Если водитель прибывает на точку, а ритейлер закрыт или отказывается принимать товар, водитель регистрирует инцидент в приложении с геопривязанным фото закрытой двери. Pegasus мгновенно фиксирует статус, информирует координатора, пересчитывает оставшийся маршрут без задержек и автоматически переносит вручение в следующий доступный тайм-слот без ручной рутины.',
    highlights: [
      'Фотофиксация закрытия с геолокацией и таймстампом',
      'Динамический перерасчет оставшегося пути на лету',
      'Автоматический подбор нового слота повторного рейса',
    ],
  },
  {
    id: 'capabilities-telematics-integration-ru',
    category: 'Интеграция',
    tag: 'ТЕЛЕМАТИКА И ТРЕКЕРЫ',
    question: 'Возможно ли подключение внешних телематических систем (Geotab, Samsara, Omnitracs, ЭРА-ГЛОНАСС)?',
    answer:
      'Да. В Pegasus встроены готовые шлюзы приема данных для коммерческих телематических платформ, включая Geotab, Samsara, Omnitracs и сертифицированные бортовые трекеры. Данные координат GPS/ГЛОНАСС, скорости, одометра, уровня топлива в баке и диагностических кодов ошибок (DTC) поступают через Webhook и потоковые шины с интервалом от 5 до 15 секунд, объединяясь с диспетчерской доской.',
    highlights: [
      'Готовые коннекторы для Geotab, Samsara и трекеров',
      'Высокочастотный сбор телеметрии каждые 5–15 секунд',
      'Интегрированный мониторинг кодов неисправностей двигателя',
    ],
  },
  {
    id: 'capabilities-cold-chain-monitoring-ru',
    category: 'Холодная цепь',
    tag: 'ТЕМПЕРАТУРНЫЕ ДАТЧИКИ',
    question: 'Как осуществляется контроль температурного режима при перевозке скоропортящихся товаров?',
    answer:
      'Pegasus считывает показатели беспроводных датчиков Bluetooth Low Energy (BLE) и штатных терморегистраторов рефрижератора. При выходе температуры за допустимые рамки (например, выше +4°C для молочной продукции или теплее -18°C для заморозки) дольше 3 минут система поднимает тревогу диспетчеру, отправляет звуковое push-уведомление водителю и маркирует партию для карантинного контроля при приемке.',
    highlights: [
      'Непрерывный опрос беспроводных BLE-термодатчиков',
      'Автоматический звуковой сигнал водителю при нарушении',
      'Электронный флаг карантина в закрывающем документе ePOD',
    ],
  },
  {
    id: 'capabilities-hardware-compatibility-ru',
    category: 'Оборудование',
    tag: 'СКАНИРОВАНИЕ И ТСД',
    question: 'Работает ли приложение на промышленных складских ТСД и потребительских смартфонах?',
    answer:
      'Мобильные клиенты Pegasus для водителей и складов разработаны в виде кроссплатформенных нативных приложений под Android и iOS. Они содержат прямую интеграцию с SDK промышленных терминалов Zebra, Honeywell и Datalogic для пакетного лазерного считывания штрихкодов со скоростью до 10 сканов в секунду, а также оптический сканер через камеру для обычных смартфонов.',
    highlights: [
      'Нативная поддержка аппаратных лазеров Zebra и Honeywell',
      'Оптическое высокоскоростное сканирование камерой телефона',
      'Полная совместимость с ударопрочными терминалами IP67',
    ],
  },
  {
    id: 'capabilities-epod-security-ru',
    category: 'Безопасность',
    tag: 'КРИПТО-EPOD',
    question: 'Как электронное подтверждение доставки защищено от фальсификации и оспаривания?',
    answer:
      'Каждое подтверждение вручения в Pegasus подписывается криптографическим хэшем SHA-256, связывающим GPS-координаты внутри радиуса геозоны магазина, точное время UTC до микросекунд, векторную цифровую подпись получателя и фотографии распакованных паллет. Этот пакет неизменяем, сохраняется в журнале аудита и генерирует официальный сертификат доставки в формате PDF с верифицируемым QR-кодом.',
    highlights: [
      'Неизменяемый криптографический хэш SHA-256 в аудите',
      'Блокировка подписи только внутри геозоны точки выгрузки',
      'Мгновенная генерация проверяемого PDF-сертификата с QR',
    ],
  },
];

// ============================================================================
// 5. OPERATIONS HUB CONFIGURATION (ENGLISH)
// ============================================================================

const OPERATIONS_DIFFERENTIATORS_EN: DifferentiatorCardConfig[] = [
  {
    icon: Users,
    badge: 'ROLE: HQ / SUPPLIER',
    kicker: '[01 // CONTROL PLANE]',
    title: 'Autonomous Master Control Plane & Network Governance',
    description:
      'Provides enterprise headquarters and suppliers with absolute visibility over network topology, customer credit limits, multi-tier pricing contracts, and macro dispatch authorization. Real-time distributed mutex locks prevent catalog overselling across regional franchise territories.',
    previewType: 'telemetry',
    previewValue: '<15ms Multi-Tenant Lock',
  },
  {
    icon: Building2,
    badge: 'ROLE: WAREHOUSE',
    kicker: '[02 // FULFILLMENT CORE]',
    title: 'Smart Staging Lanes & Volumetric Load Execution',
    description:
      'Empowers warehouse coordinators with real-time wave picking, 3D volumetric bin packing, and automated cross-dock staging. Automated freeze-locks lock manifest states the moment loading bay doors open, preventing phantom inventory discrepancies.',
    previewType: 'metric',
    previewValue: '99.4% Bay Utilization',
  },
  {
    icon: Factory,
    badge: 'ROLE: FACTORY',
    kicker: '[03 // PRODUCTION GATEWAY]',
    title: 'Manufacturing Batch Synchronization & Bay Scheduling',
    description:
      'Synchronizes production line output directly with line-haul freight schedules. Dynamic bay calloff triggers loading assignments only when finished goods pass automated QA, eliminating factory yard congestion and idle trailer demurrage.',
    previewType: 'telemetry',
    previewValue: '<60s Batch Calloff',
  },
  {
    icon: Truck,
    badge: 'ROLE: DRIVER',
    kicker: '[04 // MOBILE COCKPIT]',
    title: 'Offline-First Mobile Cockpit & Geofenced Execution',
    description:
      'Arm drivers with turnkey turn-by-turn navigation, dynamic mid-shift re-routing, and geofenced proof-of-delivery (ePOD). Operates seamlessly in cellular blackouts with local encrypted SQLite, replaying barcode scans and biometric signatures upon reconnect.',
    previewType: 'metric',
    previewValue: '100% Geofenced ePOD',
  },
  {
    icon: Store,
    badge: 'ROLE: RETAILER',
    kicker: '[05 // B2B COMMERCE]',
    title: 'B2B Self-Service Ordering & Live Inbound Radar',
    description:
      'Eliminates blind delivery arrivals. Retail store managers access transparent credit terms, submit self-service replenishment orders with instant stock confirmation, and track inbound driver trucks on real-time radar with sub-minute ETA precision.',
    previewType: 'code',
    previewValue: '<30s Self-Order Checkout',
  },
  {
    icon: ShieldCheck,
    badge: 'ROLE: GATE PASS',
    kicker: '[06 // SECURITY PERIMETER]',
    title: 'Cryptographic Gate Clearance & Tamper-Evident Seals',
    description:
      'Secures terminal entry and departure points through digital gate passes. Automated optical scanners match physical trailer tamper seals and driver biometric tokens against active Spanner manifests before triggering barrier arm releases.',
    previewType: 'graph',
    previewValue: '0% Unauthorized Gate Exits',
  },
];

const OPERATIONS_BUSINESS_VALUE_EN: BusinessValueTabConfig[] = [
  {
    id: 'orchestration',
    label: 'Orchestration',
    stats: [
      {
        value: '91.4%',
        label: 'Touchless Order Routing',
        delta: '-78% Manual Touch',
        subtext: 'Automated rule execution',
        context: 'End-to-end orders flow from ERP ingestion to warehouse dispatch without human scheduler intervention.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'End-to-End Visibility',
        delta: 'Zero Blind Spots',
        subtext: 'Across all 6 network tiers',
        context: 'Real-time state synchronization keeps HQ, warehouse directors, and store managers in continuous lockstep.',
        trend: 'up',
      },
      {
        value: '-44%',
        label: 'Inter-Facility Dwell Time',
        delta: '-28 Min Turnaround',
        subtext: 'Cross-dock staging velocity',
        context: 'Dynamic staging lane allocation and pre-arrival driver notifications eliminate trailer yard gridlock.',
        trend: 'up',
      },
      {
        value: '98.6%',
        label: 'Master Schedule Adherence',
        delta: '+14.2% On-Schedule',
        subtext: 'Multi-echelon synchronization',
        context: 'Continuous constraint-based scheduling aligns manufacturing line output with regional distribution waves.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'fulfillment',
    label: 'Fulfillment',
    stats: [
      {
        value: '99.2%',
        label: 'On-Time In-Full (OTIF)',
        delta: '+8.4% vs Baseline',
        subtext: 'Guaranteed delivery windows',
        context: 'Dynamic traffic-aware sequencing and real-time routing preserve delivery promises across city networks.',
        trend: 'up',
      },
      {
        value: '99.8%',
        label: 'Line Item Fill Rate (LIFR)',
        delta: 'Zero Pick Shorts',
        subtext: 'Atomic reserve verification',
        context: 'Strict row-level inventory locks eliminate pick shortages and partial delivery friction.',
        trend: 'up',
      },
      {
        value: '22 min',
        label: 'Dock-to-Stock Intake',
        delta: '-65% Intake Latency',
        subtext: 'Inbound pallet processing',
        context: 'High-speed barcode gate scanning populates warehouse management racks without manual re-tagging.',
        trend: 'up',
      },
      {
        value: '4.2x',
        label: 'Staging Bay Turnover',
        delta: '+52% Daily Capacity',
        subtext: 'Per-dock throughput volume',
        context: '3D volumetric packing heuristics and automated driver calloff maximize physical dock door throughput.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'exceptions',
    label: 'Exceptions',
    stats: [
      {
        value: '<45s',
        label: 'Dynamic Reroute Recovery',
        delta: 'Instant Turnaround',
        subtext: 'Mid-shift CVRP recalculation',
        context: 'When unexpected road closures or vehicle faults occur, heuristics re-sequence stops in under 45 seconds.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Shop-Closed Resolution',
        delta: 'Zero Stranded Pallets',
        subtext: 'Guarded return & retry flows',
        context: 'Geofenced photo proof records store closures and triggers automated next-window rescheduling without inventory leakage.',
        trend: 'up',
      },
      {
        value: '0.00%',
        label: 'Phantom Stockout Rate',
        delta: 'Zero Overselling',
        subtext: 'Hardware mutex locks',
        context: 'Concurrent multi-store ordering never allocates identical inventory units to conflicting retail buyers.',
        trend: 'up',
      },
      {
        value: '94.7%',
        label: 'Split-Load Automation',
        delta: 'Touchless Overflow Allocation',
        subtext: 'Smart Fit volumetric engine',
        context: 'Overweight and over-cube orders are automatically partitioned into balanced secondary dispatches.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'cashflow',
    label: 'Cash Flow',
    stats: [
      {
        value: '-12.5 Days',
        label: 'DSO Acceleration',
        delta: 'Faster Capital Velocity',
        subtext: 'Days Sales Outstanding',
        context: 'Digital cryptographic ePOD and instant reconciliation eliminate weeks of billing holdbacks and disputes.',
        trend: 'up',
      },
      {
        value: '99.96%',
        label: 'Invoice Matching Accuracy',
        delta: 'Zero Billing Drift',
        subtext: 'Automated 3-way match',
        context: 'Purchase orders, warehouse gate scans, and customer delivery receipts reconcile automatically without manual audits.',
        trend: 'up',
      },
      {
        value: '0.00%',
        label: 'Cash-at-Door Drift',
        delta: 'Integer Minor Units',
        subtext: 'Cent-accurate collections',
        context: 'Driver handheld cash recording uses strict integer arithmetic, eliminating driver shortage disputes.',
        trend: 'up',
      },
      {
        value: '$1.4M',
        label: 'Freight Audit Recovery',
        delta: 'Annual Operational Saving',
        subtext: 'Automated tariff validation',
        context: 'Every carrier accessorial charge, detention fee, and fuel surcharge is validated against contractual rates.',
        trend: 'up',
      },
    ],
  },
];

const OPERATIONS_CAPABILITIES_EN: CapabilityItemConfig[] = [
  {
    id: 'shop-closed-exception',
    title: 'Shop-Closed Exception Re-route',
    description:
      'When a retail receiver is closed, the driver logs a geofenced photo proof. Pegasus recalculates the remaining multi-stop itinerary within seconds, preserves freight security, and schedules a next-day delivery slot.',
    tag: 'EXCEPTION PLAYBOOK',
    sla: '<45s Dynamic Recovery',
    href: '/operations/shop-closed-at-delivery',
    image: '/images/topics/mobile_cockpit.jpg',
    workflowSteps: ['Geofence Exception Proof', 'Manifest Item Freeze', 'Mid-Route CVRP Re-Solve', 'Automated Reschedule Slot'],
  },
  {
    id: 'concurrent-stock-lock',
    title: 'Concurrent Atomic Stock Lock',
    description:
      'Resolves flash demand collisions across multi-channel B2B stores. Row-level distributed mutexes guarantee zero overselling with deterministic allocation timestamps and sub-second inventory updates.',
    tag: 'INVENTORY INTEGRITY',
    sla: '<18ms Mutex Guarantee',
    href: '/operations/concurrent-stock-reject',
    image: '/images/topics/warehouse_robotics.jpg',
    workflowSteps: ['Multi-Channel Order Ingress', 'Distributed Mutex Lock', 'Deterministic Stock Deduction', 'Transactional Outbox Sync'],
  },
  {
    id: 'driver-incapacity-replay',
    title: 'Dynamic Driver Incapacity Replay',
    description:
      'When a line-haul or last-mile driver experiences mid-route vehicle breakdown or incapacity, active manifests are safely unassigned and re-sequenced to backup fleet units without warehouse repack.',
    tag: 'FLEET RESILIENCE',
    sla: '<90s Manifest Transfer',
    href: '/operations/driver-reassignment',
    image: '/images/topics/route_mesh.jpg',
    workflowSteps: ['Driver Incident Alert', 'Active Manifest Freeze', 'Backup Unit Dispatch Solve', 'Digital Custody Transfer'],
  },
  {
    id: 'gate-pass-tamper-seal',
    title: 'Digital Gate Pass Tamper Seal',
    description:
      'Verifies outbound trailer integrity and driver authorization. Barcode gate scanners match cryptographic seal serial numbers with driver biometric IDs before triggering automated gate barrier release.',
    tag: 'SECURITY GATE',
    sla: '<12s Terminal Clearance',
    href: '/operations/wrong-truck-sealed',
    image: '/images/topics/port_intermodal.jpg',
    workflowSteps: ['Terminal Optical Scan', 'Driver Identity Match', 'Cryptographic Seal Verification', 'Barrier Release Authorization'],
  },
];

const OPERATIONS_FAQS_EN: FaqItemConfig[] = [
  {
    id: 'ops-faq-wave-dispatch',
    category: 'Warehouse',
    tag: 'WAVE PICKING',
    question: 'How does Pegasus coordinate warehouse pick-and-pack waves with driver dispatch schedules?',
    answer:
      'Pegasus coordinates warehouse fulfillment waves using continuous backward scheduling from target delivery time windows. When an order is committed, the engine calculates the required loading dock departure time, subtracts staging, packing, and picking durations, and allocates staging bays automatically. As warehouse pickers scan barcodes on handheld terminals, real-time pick progress updates the dispatch board. If a pick wave falls behind schedule, Pegasus dynamically adjusts dock door assignments and alerts staging coordinators before drivers arrive.',
    highlights: [
      'Continuous backward constraint scheduling',
      'Real-time picker scan telemetry integration',
      'Dynamic staging bay and dock door rebalancing',
    ],
  },
  {
    id: 'ops-faq-shop-closed',
    category: 'Exceptions',
    tag: 'STORE CLOSURES',
    question: 'How does the "Shop Closed at Delivery" exception protocol prevent lost inventory and driver detention?',
    answer:
      'When a driver arrives at a retailer location that is closed or inaccessible, the mobile cockpit enforces a geofence check within 50 meters and requires a timestamped, geotagged photograph. Submitting this exception marks the order items as "Delivery Attempt Failed — Closed", immediately recalculates the driver\'s remaining turn-by-turn route to eliminate wasted dwell time, and generates a warehouse return manifest or schedules an automated re-attempt for the retailer\'s next operating window.',
    highlights: [
      '50-meter geofenced photographic exception validation',
      'Instant automated route re-sequencing for remaining stops',
      'Automated next-window reschedule without manual dispatcher intervention',
    ],
  },
  {
    id: 'ops-faq-gate-security',
    category: 'Gate Security',
    tag: 'DIGITAL GATE PASS',
    question: 'How does Pegasus enforce physical and digital security at warehouse gate terminals?',
    answer:
      'Gate pass security operates as a cryptographic gatekeeper. Outbound trailers cannot clear security checkpoints unless three independent factors match: the driver\'s cryptographic token, the verified trailer registration number, and the unique barcode of the tamper-evident cable seal applied at the loading dock. Optical scanners and mobile handhelds scan these elements in under 12 seconds. Any discrepancy triggers an immediate security lockdown and notifies the logistics operations center.',
    highlights: [
      'Three-factor gate verification (Driver, Trailer, Tamper Seal)',
      'Sub-12 second optical barcode and token validation',
      'Instant facility lockdown trigger upon seal mismatch',
    ],
  },
  {
    id: 'ops-faq-cod-reconciliation',
    category: 'Finance',
    tag: 'CASH RECONCILIATION',
    question: 'How are COD (Cash on Delivery) and cash-at-door collections secured and reconciled?',
    answer:
      'Cash-at-door collections are managed through strict integer minor-unit accounting (recording every transaction in cents or tiyins) to eliminate floating-point calculation errors. Drivers record cash received in the mobile cockpit, which issues an SMS/QR receipt to the retailer. Upon end-of-shift depot return, the gate terminal tallies the driver\'s expected cash pouch against the manifest receipts. Reconciliation happens instantly in Cloud Spanner, posting confirmed collections directly to the corporate general ledger.',
    highlights: [
      'Integer minor-unit accounting eliminates rounding drift',
      'Instant digital SMS and QR receipts for store receivers',
      'End-of-shift cash pouch reconciliation with automated ERP ledger entry',
    ],
  },
  {
    id: 'ops-faq-truck-too-small',
    category: 'Dispatch',
    tag: 'CAPACITY OVERFLOW',
    question: 'What happens when an order exceeds the volumetric or weight capacity of an assigned truck ("Truck Too Small")?',
    answer:
      'If warehouse staging reveals an order exceeds truck volume or axle weight limits, coordinators trigger the "Smart Fit Overflow" protocol. Pegasus\'s 3D bin-packing algorithm partitions the order into a primary dispatch that maximizes the truck\'s cube utilization (typically 94–96%) and a secondary overflow batch. The secondary batch is automatically injected into the next scheduled delivery wave or tendered to backup regional fleet capacity without cancelling the original customer order.',
    highlights: [
      '3D bin-packing heuristic partitions oversized orders',
      'Primary truck dispatched at 94–96% cube utilization',
      'Secondary overflow batch auto-assigned to next delivery wave',
    ],
  },
  {
    id: 'ops-faq-driver-reassignment',
    category: 'Fleet',
    tag: 'INCIDENT RECOVERY',
    question: 'How does Pegasus handle driver reassignment during active delivery shifts?',
    answer:
      'If a driver experiences an acute health emergency or vehicle mechanical breakdown mid-route, the operations dispatcher triggers the "Driver Incapacity Replay" workflow. The active route manifest freezes instantly, isolating already-completed deliveries from pending stops. Pegasus calculates the geographic centroid of remaining stops, evaluates nearby active drivers with available capacity and remaining hours of service (HOS), and transfers the remaining manifest stops to the replacement driver\'s mobile cockpit without requiring inventory unsealing or warehouse return.',
    highlights: [
      'Instant manifest state freeze isolating completed deliveries',
      'Dynamic capacity and Hours of Service (HOS) evaluation',
      'Seamless digital manifest transfer to replacement driver cockpit',
    ],
  },
  {
    id: 'ops-faq-factory-batch-sync',
    category: 'Manufacturing',
    tag: 'PRODUCTION BATCHES',
    question: 'How do factory production batches transition into warehouse cross-dock manifests?',
    answer:
      'Manufacturing facilities running Pegasus push batch completion events via automated REST/gRPC webhooks. The platform validates pallet counts, temperature compliance, and packaging standards against active downstream customer demand. Pallets destined for immediate cross-docking bypass long-term warehouse storage racks and are directed straight to staging bays assigned to outbound regional line-haul trailers, reducing total handling touches and cutting dock dwell time by up to 44%.',
    highlights: [
      'Automated manufacturing batch completion webhook triggers',
      'Direct cross-dock routing bypassing long-term storage racks',
      'Up to 44% reduction in warehouse yard and dock dwell time',
    ],
  },
];

// ============================================================================
// 6. OPERATIONS HUB CONFIGURATION (RUSSIAN)
// ============================================================================

const OPERATIONS_DIFFERENTIATORS_RU: DifferentiatorCardConfig[] = [
  {
    icon: Users,
    badge: 'РОЛЬ: ШТАБ-КВАРТИРА / ПОСТАВЩИК',
    kicker: '[01 // КОНТУР УПРАВЛЕНИЯ]',
    title: 'Автономный контур управления и сетевая координация',
    description:
      'Предоставляет руководству сети и поставщикам полный контроль над топологией распределения, кредитными лимитами контрагентов, сложными тарифными сетками и макро-диспетчеризацией. Распределенные мьютексы исключают оверселлинг товарных позиций в региональных франшизах.',
    previewType: 'telemetry',
    previewValue: '<15ms Multi-Tenant Lock',
  },
  {
    icon: Building2,
    badge: 'РОЛЬ: СКЛАД И РЦ',
    kicker: '[02 // ЯДРО ИСПОЛНЕНИЯ]',
    title: 'Интеллектуальные зоны комплектации и объемная укладка',
    description:
      'Обеспечивает начальников смен инструментами волновой сборки заказов, 3D-тетрисом укладки полуприцепов и сквозным кросс-докингом. Автоматический freeze-lock фиксирует состав рейса в момент открытия ворот рампы, предотвращая расхождения в остатках.',
    previewType: 'metric',
    previewValue: '99.4% Bay Utilization',
  },
  {
    icon: Factory,
    badge: 'РОЛЬ: ФАБРИКА / ПРОИЗВОДСТВО',
    kicker: '[03 // ПРОИЗВОДСТВЕННЫЙ ШЛЮЗ]',
    title: 'Синхронизация производственных партий и диспетчеризация рамп',
    description:
      'Связывает сход готовой продукции с конвейера с графиком подачи магистрального транспорта. Динамический вызов тягачей на погрузку инициируется только после прохождения электронного контроля качества ОТК, устраняя простой транспорта на территории завода.',
    previewType: 'telemetry',
    previewValue: '<60s Batch Calloff',
  },
  {
    icon: Truck,
    badge: 'РОЛЬ: ВОДИТЕЛЬ',
    kicker: '[04 // МОБИЛЬНЫЙ КОКПИТ]',
    title: 'Офлайн-кокпит водителя и геозонированная доставка',
    description:
      'Предоставляет водителям пошаговую грузовую навигацию, динамический перерасчет остановок в рейсе и электронное подтверждение доставки (ePOD). Надежно работает при полном отсутствии сотовой связи на базе локальной SQLite, передавая подписи и сканы при выходе в сеть.',
    previewType: 'metric',
    previewValue: '100% Geofenced ePOD',
  },
  {
    icon: Store,
    badge: 'РОЛЬ: ТОЧКА ПРОДАЖ / РИТЕЙЛЕР',
    kicker: '[05 // B2B-КОММЕРЦИЯ]',
    title: 'B2B-самообслуживание и радар приближения доставки',
    description:
      'Устраняет неопределенность поставок в розницу. Управляющие торговых точек видят актуальный баланс взаиморасчетов, формируют заказы с мгновенным подтверждением резерва и отслеживают приближение рейса на интерактивном радаре с минутной точностью прибытия.',
    previewType: 'code',
    previewValue: '<30s Self-Order Checkout',
  },
  {
    icon: ShieldCheck,
    badge: 'РОЛЬ: ТЕРМИНАЛ КПП',
    kicker: '[06 // КОНТУР БЕЗОПАСНОСТИ]',
    title: 'Криптографический пропускной контроль и электронные пломбы',
    description:
      'Защищает въезды и выезды с логистических комплексов цифровыми пропусками. Оптические сканеры терминала КПП сверяют уникальные номера пломб и биометрические токены водителя с активными манифестами Spanner до автоматического открытия шлагбаума.',
    previewType: 'graph',
    previewValue: '0% Unauthorized Gate Exits',
  },
];

const OPERATIONS_BUSINESS_VALUE_RU: BusinessValueTabConfig[] = [
  {
    id: 'orchestration',
    label: 'Координация',
    stats: [
      {
        value: '91.4%',
        label: 'Автоматическая маршрутизация',
        delta: '-78% Ручных правок',
        subtext: 'Исполнение бизнес-правил',
        context: 'Сквозное движение заказов от импорта из ERP до назначения рейса без ручного вмешательства диспетчеров.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Сквозная прослеживаемость',
        delta: 'Ноль слепых зон',
        subtext: 'На всех 6 уровнях цепочки',
        context: 'Синхронизация статусов в реальном времени связывает руководство, склад и розничные точки в единый контур.',
        trend: 'up',
      },
      {
        value: '-44%',
        label: 'Время простоя на РЦ',
        delta: '-28 минут на рейс',
        subtext: 'Скорость кросс-докинга',
        context: 'Динамическое назначение ворот и предварительное оповещение водителей устраняют заторы на погрузочных дворах.',
        trend: 'up',
      },
      {
        value: '98.6%',
        label: 'Соблюдение графика отгрузок',
        delta: '+14.2% Точности графика',
        subtext: 'Многоуровневая синхронизация',
        context: 'Непрерывное планирование с учетом ограничений согласует производственные графики с волнами распределения.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'fulfillment',
    label: 'Исполнение',
    stats: [
      {
        value: '99.2%',
        label: 'Показатель OTIF (В срок и целиком)',
        delta: '+8.4% к среднему рынку',
        subtext: 'Гарантированные окна доставки',
        context: 'Динамическое планирование с учетом дорожного трафика обеспечивает доставку строго в согласованное окно.',
        trend: 'up',
      },
      {
        value: '99.8%',
        label: 'Полнота строк заказа (LIFR)',
        delta: 'Ноль недовозов',
        subtext: 'Атомарная проверка резервов',
        context: 'Строгие блокировки остатков на уровне строк БД исключают недостачи при сборке и конфликты на выгрузке.',
        trend: 'up',
      },
      {
        value: '22 мин',
        label: 'Приемка от рампы до полки',
        delta: '-65% Задержки приемки',
        subtext: 'Обработка входящих паллет',
        context: 'Скоростное сканирование штрихкодов на КПП автоматически размещает паллеты в топологии склада.',
        trend: 'up',
      },
      {
        value: '4.2x',
        label: 'Оборачиваемость погрузочных рамп',
        delta: '+52% Суточной мощности',
        subtext: 'Объем на одни ворота',
        context: '3D-эвристики объемной укладки и автоматический вызов водителей максимизируют пропускную способность рамп.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'exceptions',
    label: 'Исключения',
    stats: [
      {
        value: '<45с',
        label: 'Перерасчет маршрута при сбое',
        delta: 'Мгновенная реакция',
        subtext: 'Пересчет CVRP в реальном времени',
        context: 'При перекрытии дорог или поломке машины система перестраивает оставшиеся точки маршрута менее чем за 45 секунд.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Разрешение "Магазин закрыт"',
        delta: 'Ноль потерянных грузов',
        subtext: 'Защищенные регламенты возврата',
        context: 'Фотофиксация в геозоне регистрирует отказ в приемке и автоматически переносит доставку на следующее окно.',
        trend: 'up',
      },
      {
        value: '0.00%',
        label: 'Уровень ложных дефицитов',
        delta: 'Ноль оверселлинга',
        subtext: 'Аппаратные мьютексы',
        context: 'Конкурентные заказы от сотен магазинов никогда не резервируют одну и ту же физическую паллету.',
        trend: 'up',
      },
      {
        value: '94.7%',
        label: 'Авторазделение негабаритов',
        delta: 'Автоматическое распределение',
        subtext: 'Объемный движок Smart Fit',
        context: 'Превышающие грузоподъемность или объем фуры партии автоматически разбиваются на сбалансированные рейсы.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'cashflow',
    label: 'Денежный поток',
    stats: [
      {
        value: '-12.5 Дней',
        label: 'Ускорение оборачиваемости (DSO)',
        delta: 'Высвобождение капитала',
        subtext: 'Период погашения дебиторки',
        context: 'Цифровой криптографический ePOD и мгновенная сверка устраняют многонедельные споры по актам приемки.',
        trend: 'up',
      },
      {
        value: '99.96%',
        label: 'Точность трехсторонней сверки',
        delta: 'Ноль расхождений в счетах',
        subtext: 'Автоматический 3-way match',
        context: 'Заказы на поставку, сканы с терминалов КПП и электронные накладные покупателя сверяются автоматически.',
        trend: 'up',
      },
      {
        value: '0.00%',
        label: 'Расхождения по наличным (COD)',
        delta: 'Целочисленный учет',
        subtext: 'Точность до копейки / тийина',
        context: 'Фиксация приема наличных в приложении водителя ведется в минимальных денежных единицах без погрешностей.',
        trend: 'up',
      },
      {
        value: '$1.4M',
        label: 'Возврат потерь на тарифах',
        delta: 'Годовая экономия на логистике',
        subtext: 'Автоматический аудит ставок',
        context: 'Каждый счет за простой, дополнительный заезд или топливную надбавку верифицируется по условиям контракта.',
        trend: 'up',
      },
    ],
  },
];

const OPERATIONS_CAPABILITIES_RU: CapabilityItemConfig[] = [
  {
    id: 'shop-closed-exception',
    title: 'Регламент при закрытой торговой точке',
    description:
      'Если магазин закрыт или не принимает груз, водитель фиксирует факт геопривязанным фото. Pegasus за секунды пересчитывает оставшийся маршрут, сохраняет груз и планирует повторную доставку.',
    tag: 'РЕГЛАМЕНТ ИСКЛЮЧЕНИЙ',
    sla: '<45с Динамическое восстановление',
    href: '/operations/shop-closed-at-delivery',
    image: '/images/topics/mobile_cockpit.jpg',
    workflowSteps: ['Фиксация в геозоне', 'Заморозка строки манифеста', 'Пересчет маршрута CVRP', 'Автоперенос в график'],
  },
  {
    id: 'concurrent-stock-lock',
    title: 'Конкурентный атомарный резерв остатков',
    description:
      'Разрешает пиковые коллизии спроса в B2B-каналах. Распределенные мьютексы на уровне строк БД гарантируют нулевой оверселлинг с детерминированными временными метками фиксации.',
    tag: 'ЦЕЛОСТНОСТЬ ОСТАТКОВ',
    sla: '<18мс Гарантия мьютекса',
    href: '/operations/concurrent-stock-reject',
    image: '/images/topics/warehouse_robotics.jpg',
    workflowSteps: ['Поступление заказов', 'Распределенный мьютекс', 'Атомарное списание', 'Трансляция через Outbox'],
  },
  {
    id: 'driver-incapacity-replay',
    title: 'Динамическая замена водителя в рейсе',
    description:
      'При внезапной поломке машины или болезни водителя на маршруте активный манифест безопасно замораживается и передается резервному экипажу без возврата груза на склад.',
    tag: 'НАДЕЖНОСТЬ АВТОПАРКА',
    sla: '<90с Передача манифеста',
    href: '/operations/driver-reassignment',
    image: '/images/topics/route_mesh.jpg',
    workflowSteps: ['Сигнал об инциденте', 'Заморозка манифеста', 'Подбор резервного борта', 'Электронный прием рейса'],
  },
  {
    id: 'gate-pass-tamper-seal',
    title: 'Электронный пропуск КПП и контроль пломб',
    description:
      'Контролирует выезд транспорта и полномочия водителя. Оптические сканеры на выезде сверяют уникальный номер пломбы и биометрический токен водителя перед открытием шлагбаума.',
    tag: 'КОНТРОЛЬ КПП',
    sla: '<12с Проезд терминала',
    href: '/operations/wrong-truck-sealed',
    image: '/images/topics/port_intermodal.jpg',
    workflowSteps: ['Сканирование на КПП', 'Проверка личности водителя', 'Верификация пломбы', 'Команда на открытие'],
  },
];

const OPERATIONS_FAQS_RU: FaqItemConfig[] = [
  {
    id: 'ops-faq-wave-dispatch',
    category: 'Склад',
    tag: 'ВОЛНОВАЯ СБОРКА',
    question: 'Как Pegasus согласует волны складской сборки с графиками подачи транспорта?',
    answer:
      'Pegasus рассчитывает волны сборки методом обратного планирования от целевого окна доставки клиенту. При подтверждении заказа система вычисляет необходимое время выезда с рампы, вычитает время упаковки и отбора и автоматически резервирует погрузочные ворота. По мере того как комплектовщики сканируют штрихкоды терминалами сбора данных, статус сборки отображается на пульте диспетчера. Если волна задерживается, система заранее предупреждает старшего смены и переназначает ворота до прибытия водителя.',
    highlights: [
      'Обратное планирование от временных окон доставки',
      'Телеметрия сканирования ТСД в реальном времени',
      'Динамическая балансировка погрузочных ворот и рамп',
    ],
  },
  {
    id: 'ops-faq-shop-closed',
    category: 'Исключения',
    tag: 'ЗАКРЫТЫЕ МАГАЗИНЫ',
    question: 'Как регламент "Магазин закрыт" предотвращает потери товара и простой водителей?',
    answer:
      'Когда водитель прибывает к закрытой торговой точке, мобильное приложение проверяет нахождение в радиусе 50 метров по GPS и запрашивает снимок закрытого фасада с меткой времени. Отправка отчета переводит позицию в статус "Попытка не удалась", мгновенно перестраивает оставшийся маршрут без потери времени на ожидание и формирует акт возврата на склад либо автоматически назначает повторный заезд на следующий рабочий день магазина.',
    highlights: [
      'Фотофиксация причины отказа в радиусе 50 метров по GPS',
      'Мгновенный пересчет маршрута по оставшимся точкам доставки',
      'Автоматический перенос рейса без ручного вмешательства логиста',
    ],
  },
  {
    id: 'ops-faq-gate-security',
    category: 'Безопасность КПП',
    tag: 'ЦИФРОВОЙ ПРОПУСК',
    question: 'Как обеспечивается физическая и цифровая безопасность на контрольно-пропускных пунктах складов?',
    answer:
      'Безопасность на КПП основана на трехфакторной проверке. Тягач с полуприцепом не может покинуть территорию, пока оптические сканеры терминала не подтвердят совпадение трех элементов: криптографического токена водителя, госномера тягача и уникального штрихкода тросовой пломбы, навешенной на рампе. Проверка занимает менее 12 секунд. Любое несовпадение немедленно блокирует шлагбаум и отправляет тревожное оповещение в службу безопасности.',
    highlights: [
      'Трехфакторная проверка: водитель, тягач, номер пломбы',
      'Оптическое распознавание и верификация менее чем за 12 секунд',
      'Автоматическая блокировка шлагбаума при расхождении данных',
    ],
  },
  {
    id: 'ops-faq-cod-reconciliation',
    category: 'Финансы',
    tag: 'СВЕРКА НАЛИЧНЫХ',
    question: 'Как защищены и сверяются платежи при оплате наличными при доставке (COD)?',
    answer:
      'Прием наличных при доставке ведется в целочисленных минимальных единицах валюты (копейки или тийины), что полностью исключает ошибки округления с плавающей точкой. Водитель регистрирует сумму в приложении, отправляя покупателю электронный чек по SMS или QR-коду. По возвращении на базу терминал КПП сверяет сумму сданной инкассаторской сумки с манифестом. Сверка в Cloud Spanner происходит мгновенно, автоматически формируя проводки в главной бухгалтерской книге предприятия.',
    highlights: [
      'Целочисленный учет исключает накопление погрешностей округления',
      'Мгновенная выдача электронных чеков клиентам по QR и SMS',
      'Сверка инкассаторских сумок на КПП с автоматической проводкой в ERP',
    ],
  },
  {
    id: 'ops-faq-truck-too-small',
    category: 'Диспетчеризация',
    tag: 'ИЗБЫТОК ОБЪЕМА',
    question: 'Что происходит, если объем или масса заказа превышают вместимость назначенного транспорта ("Кузов переполнен")?',
    answer:
      'Если при формировании рейса обнаруживается превышение предельной грузоподъемности по осям или кубатуры кузова, запускается протокол "Smart Fit Overflow". Алгоритм 3D-укладки разбивает партию на основной рейс с максимальным коэффициентом заполнения (94–96%) и излишек. Излишек автоматически ставится в следующую волну отгрузки или передается привлеченному транспорту без отмены и повторного заведения исходного заказа в системе.',
    highlights: [
      'Алгоритм 3D-укладки разделяет негабаритную партию',
      'Основной борт уходит с плотностью загрузки 94–96%',
      'Излишек автоматически переносится в следующую волну доставки',
    ],
  },
  {
    id: 'ops-faq-driver-reassignment',
    category: 'Автопарк',
    tag: 'ЗАМЕНА В РЕЙСЕ',
    question: 'Как Pegasus управляет экстренной заменой водителя в процессе выполнения рейса?',
    answer:
      'В случае внезапной болезни водителя или технической неисправности тягача на линии диспетчер активирует сценарий "Driver Incapacity Replay". Активный маршрут мгновенно замораживается, отделяя уже врученные заказы от оставшихся. Pegasus определяет координаты оставшихся точек, находит ближайший свободный экипаж с достаточным запасом рабочего времени по тахографу и передает манифест на мобильное устройство сменного водителя без вскрытия пломб и заезда на склад.',
    highlights: [
      'Мгновенная фиксация статуса выполненных доставок в рейсе',
      'Подбор подменного экипажа с учетом доступных часов по тахографу',
      'Бесшовная передача электронного манифеста на планшет нового водителя',
    ],
  },
  {
    id: 'ops-faq-factory-batch-sync',
    category: 'Производство',
    tag: 'ПРОИЗВОДСТВЕННЫЙ ПОТОК',
    question: 'Как производственные партии с завода переводятся в кросс-докинг манифесты распределительных центров?',
    answer:
      'Производственные площадки передают события выпуска партий через защищенные вебхуки REST/gRPC. Pegasus сопоставляет количество паллет, температурные параметры и партионные номера с открытыми заказами клиентов. Паллеты, подлежащие прямой кросс-докинг отгрузке, минуют стеллажи длительного хранения и направляются прямо в зоны накопления к воротам магистральных фур, сокращая количество перемещений и уменьшая время простоя на 44%.',
    highlights: [
      'Автоматическая передача событий выпуска продукции по вебхукам',
      'Прямой кросс-докинг без размещения на стеллажах хранения',
      'Сокращение времени нахождения груза на терминале до 44%',
    ],
  },
];

// ============================================================================
// 7. TECHNOLOGY HUB CONFIGURATION (ENGLISH)
// ============================================================================

const TECHNOLOGY_DIFFERENTIATORS_EN: DifferentiatorCardConfig[] = [
  {
    icon: Terminal,
    badge: 'GO RUNTIME',
    kicker: '[01 // APPLICATION CORE]',
    title: 'Go High-Concurrency Modular Monolith',
    description:
      'Unified high-throughput backend compiled to native binary with zero runtime reflection overhead. Goroutine worker pools handle high-volume order ingestion, tariff matrices, and state transitions with sub-2ms p50 internal latency and minimal memory footprint, avoiding microservice network tax.',
    previewType: 'code',
    previewValue: '400+ Handlers / <2ms p50',
  },
  {
    icon: Database,
    badge: 'SPANNER ACID',
    kicker: '[02 // STORAGE ENGINE]',
    title: 'Cloud Spanner Globally Distributed ACID Ledger',
    description:
      'TrueTime-synchronized Google Cloud Spanner multi-region database serving as the immutable single system of record. Enforces strict serializability across continents with automated Paxos consensus, eliminating split-brain risks and distributed transaction deadlocks.',
    previewType: 'metric',
    previewValue: '99.999% Zero-Split Brain',
  },
  {
    icon: Workflow,
    badge: 'OUTBOX RELAY',
    kicker: '[03 // RELIABLE MESSAGING]',
    title: 'Guaranteed Transactional Outbox Kafka Relay',
    description:
      'Eliminates dual-write inconsistencies. Database mutations write domain entities and outbox event envelopes in a single atomic Spanner commit. Dedicated background pollers stream outbox events to Kafka with guaranteed at-least-once delivery and monotonic sequence order.',
    previewType: 'code',
    previewValue: 'Zero Dropped Kafka Events',
  },
  {
    icon: Radio,
    badge: 'WEBSOCKET MESH',
    kicker: '[04 // STREAMING FANOUT]',
    title: 'Role-Partitioned WebSocket Streaming Mesh',
    description:
      'Distributed WebSocket cluster multiplexed by organization, role, and geographic territory. Dispatch updates, driver location telemetry, and gate transitions fan out to thousands of connected browser and mobile clients in under 80 milliseconds without polling.',
    previewType: 'telemetry',
    previewValue: '<80ms Sub-Second Fanout',
  },
  {
    icon: Cpu,
    badge: 'CVRP SOLVER',
    kicker: '[05 // ALGORITHMIC ENGINE]',
    title: 'GPU-Accelerated Dynamic CVRP Optimization',
    description:
      'Solves Capacitated Vehicle Routing Problems with hard time windows (CVRPTW), 3D volumetric stacking limits, and driver shift constraints using hybrid Clarke-Wright and parallel Large Neighborhood Search (LNS) in sub-second cycles.',
    previewType: 'telemetry',
    previewValue: '<350ms Heuristic Solve',
  },
  {
    icon: Lock,
    badge: 'ZERO-TRUST TENANCY',
    kicker: '[06 // SECURITY PERIMETER]',
    title: 'Zero-Trust Cryptographic Multi-Tenancy',
    description:
      'Row-level security policies bound to cryptographically signed JWT claims enforce absolute tenant boundaries. Competing brands sharing cross-dock terminals or delivery fleets have mathematical guarantees against cross-tenant data leakage or route inference.',
    previewType: 'graph',
    previewValue: 'Strict Tenant Isolation',
  },
];

const TECHNOLOGY_BUSINESS_VALUE_EN: BusinessValueTabConfig[] = [
  {
    id: 'throughput',
    label: 'Throughput',
    stats: [
      {
        value: '24,000+',
        label: 'Mutation Write Rate',
        delta: '+340% vs Legacy RDBMS',
        subtext: 'Cloud Spanner distributed commits',
        context: 'Handles peak order ingestion across dozens of enterprise brands without write throttling or deadlock rollbacks.',
        trend: 'up',
      },
      {
        value: '1.8M',
        label: 'Telemetry Ingest Velocity',
        delta: 'Zero Backpressure',
        subtext: 'Kafka streaming pipeline',
        context: 'Continuous GPS pings, BLE temperature readings, and engine CAN-bus telemetry processed concurrently.',
        trend: 'up',
      },
      {
        value: '150,000+',
        label: 'Concurrent Live Sockets',
        delta: 'Sub-80MB RAM per Node',
        subtext: 'Go WebSocket hub multiplexing',
        context: 'Simultaneous active connections across warehouse terminals, driver devices, and retailer portals.',
        trend: 'up',
      },
      {
        value: '85,000',
        label: 'Order Lines / Minute',
        delta: '<2.4s Ingest-to-Floor',
        subtext: 'Parallel ERP batch streams',
        context: 'High-velocity EDI and REST import pipelines parse, validate, and schedule master order batches.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'latency',
    label: 'Latency',
    stats: [
      {
        value: '4.2ms',
        label: 'Internal API p95 Latency',
        delta: '-62% vs Node/Python',
        subtext: 'Go compiled binary runtime',
        context: 'Role-scoped API handlers execute business logic and validations without garbage collection pauses.',
        trend: 'up',
      },
      {
        value: '<65ms',
        label: 'End-to-End Event Fanout',
        delta: 'Near-Instant Sync',
        subtext: 'Database commit to client screen',
        context: 'State transitions in Spanner stream via outbox and WebSockets to mobile cockpits instantaneously.',
        trend: 'up',
      },
      {
        value: '<14ms',
        label: 'Distributed Lock Acquisition',
        delta: 'Strict Serializability',
        subtext: 'Spanner row-level lock lease',
        context: 'Guarantees atomic stock allocation and prevents double-booking across multi-channel retailers.',
        trend: 'up',
      },
      {
        value: '<42ms',
        label: 'Multi-Region Consensus',
        delta: 'Global Read Consistency',
        subtext: 'Google Cloud Spanner TrueTime',
        context: 'Cross-datacenter state replication achieves consensus without stale read anomalies.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'consistency',
    label: 'Consistency',
    stats: [
      {
        value: '100%',
        label: 'Transactional Serializability',
        delta: 'Zero Dirty Reads',
        subtext: 'Google Cloud Spanner TrueTime',
        context: 'Guaranteed external consistency: all nodes observe transactions in identical chronological order.',
        trend: 'up',
      },
      {
        value: '0ms',
        label: 'Eventual Consistency Gap',
        delta: 'Zero Replication Drift',
        subtext: 'Strong consistency reads',
        context: 'Eliminates reconciliation delays: reading an order immediately after modification always returns current state.',
        trend: 'up',
      },
      {
        value: '0.000%',
        label: 'Lost Outbox Mutations',
        delta: 'Guaranteed At-Least-Once',
        subtext: 'Idempotent Kafka publishers',
        context: 'Transactional outbox guarantees that every database state mutation results in an event publish.',
        trend: 'up',
      },
      {
        value: '0.000%',
        label: 'Duplicate Broadcasts',
        delta: 'Exact-Once Processing',
        subtext: 'Monotonic sequence keys',
        context: 'Client devices and downstream consumers deduplicate message streams using monotonic sequence keys.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'availability',
    label: 'Availability',
    stats: [
      {
        value: '99.999%',
        label: 'Core Engine Availability',
        delta: '<5.2 Min Downtime/Yr',
        subtext: 'Multi-region active-active clusters',
        context: 'Redundant multi-zone compute topology guarantees continuous mission-critical supply chain operations.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Zero-Downtime Migrations',
        delta: 'Continuous CI/CD CD',
        subtext: 'Dual-write schema evolution',
        context: 'Database schema expansions and application releases deploy with zero downtime for warehouse floors.',
        trend: 'up',
      },
      {
        value: '<1.8s',
        label: 'Regional Failover Time',
        delta: 'Zero Operator Intervention',
        subtext: 'Automated DNS & GKE Failover',
        context: 'Unplanned cloud availability zone outages fail over to standby replicas without lost transactions.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Partition Tolerance',
        delta: 'Mathematical Safety',
        subtext: 'Spanner Paxos quorum',
        context: 'Network partitions safely reject conflicting mutations rather than split-braining inventory ledgers.',
        trend: 'up',
      },
    ],
  },
];

const TECHNOLOGY_CAPABILITIES_EN: CapabilityItemConfig[] = [
  {
    id: 'outbox-kafka-relay',
    title: 'Transactional Outbox Kafka Relay',
    description:
      'Solves the dual-write problem by atomically writing domain entity mutations and event outbox records in the same Spanner transaction, streamed to Kafka by idempotent workers with monotonic sequencing.',
    tag: 'EVENT PIPELINE',
    sla: '<50ms Relay Latency',
    href: '/technology/redis-kafka',
    image: '/images/topics/event_pipeline.jpg',
    workflowSteps: ['Domain Mutation', 'Atomic Outbox Write', 'Poller Relay Worker', 'Kafka Topic Publish'],
  },
  {
    id: 'spanner-mutation-interceptor',
    title: 'Spanner Mutation Interceptor',
    description:
      'Intercepts database writes to enforce row-level tenant authorization, audit log generation, and monotonic version validation before passing commits to Cloud Spanner distributed consensus.',
    tag: 'STORAGE ENGINE',
    sla: '<12ms Intercept & Commit',
    href: '/technology/cloud-spanner',
    image: '/images/topics/cloud_spanner.jpg',
    workflowSteps: ['Ingress Request', 'JWT Claims Parsing', 'Mutation Validation', 'Spanner Commit'],
  },
  {
    id: 'sub200ms-telemetry-fanout',
    title: 'Sub-200ms Telemetry Fanout',
    description:
      'High-performance Go WebSocket cluster that ingests GPS and IoT telemetry from hundreds of active delivery vehicles and broadcasts updates to role-scoped client dashboards in real-time.',
    tag: 'STREAMING MESH',
    sla: '<80ms Broadcast Latency',
    href: '/technology/websocket-hubs',
    image: '/images/topics/fleet_radar.jpg',
    workflowSteps: ['Vehicle Telemetry Ping', 'Ingestion Broker', 'Role-Room Filtering', 'WebSocket Fanout'],
  },
  {
    id: 'lock-lease-recovery',
    title: 'Distributed Lock Lease Recovery',
    description:
      'Fault-tolerant distributed locking mechanism that ensures warehouse staging bays and high-demand inventory allocations never deadlock during unexpected worker process crashes.',
    tag: 'CONCURRENCY CONTROL',
    sla: '<2.5s Failover Eviction',
    href: '/technology/go-backend-platform',
    image: '/images/topics/control_plane.jpg',
    workflowSteps: ['Mutex Acquisition', 'Heartbeat Lease Renewal', 'Crash Detection Timeout', 'Safe Lease Eviction'],
  },
];

const TECHNOLOGY_FAQS_EN: FaqItemConfig[] = [
  {
    id: 'tech-faq-spanner-selection',
    category: 'Database',
    tag: 'GLOBAL CONSENSUS',
    question: 'Why did Pegasus choose Google Cloud Spanner over traditional PostgreSQL or Cassandra?',
    answer:
      'Enterprise supply chain networks require both global multi-region scalability and strict ACID serializability. Traditional relational databases like PostgreSQL struggle with multi-region active-active replication without latency penalties or split-brain risks, while NoSQL datastores like Cassandra sacrifice consistency for availability (CAP theorem). Google Cloud Spanner leverages hardware atomic clocks (TrueTime API) to deliver externally consistent distributed transactions, automatic sharding, and 99.999% availability, ensuring that inventory reservations and dispatch commitments are globally deterministic.',
    highlights: [
      'Hardware TrueTime atomic clocks guarantee external consistency',
      'Multi-region active-active scalability without split-brain risk',
      'Strict ACID serializability prevents phantom inventory allocations',
    ],
  },
  {
    id: 'tech-faq-go-monolith',
    category: 'Architecture',
    tag: 'GO RUNTIME',
    question: 'How does the Go modular monolith compare with microservices for logistics operations?',
    answer:
      'Microservice architectures introduce heavy network serialization latency, distributed tracing overhead, and complex partial failure modes across hundreds of remote procedure calls. In contrast, Pegasus is built as a modular monolith in Go. Domain boundaries (dispatch, inventory, routing, payments) are strictly enforced at the compiler level via internal packages and interfaces, while executing within a single high-efficiency binary. This design delivers sub-2ms inter-module call latencies, trivial local testing, and minimal operational overhead while retaining pristine domain separation.',
    highlights: [
      'Sub-2ms internal call latency with zero network RPC tax',
      'Compile-time modular domain boundaries via Go packages',
      'Single high-efficiency binary reduces operational and infra footprint',
    ],
  },
  {
    id: 'tech-faq-outbox-pattern',
    category: 'Streaming',
    tag: 'TRANSACTIONAL OUTBOX',
    question: 'How does the Transactional Outbox pattern eliminate dual-write race conditions?',
    answer:
      'In distributed systems, updating a database and publishing an event to a message broker in two separate operations introduces dual-write failures: if the message broker fails after the database commit, downstream subscribers miss the event; if the database rollback occurs after event publishing, ghost events propagate. Pegasus writes domain entity changes and an event envelope record into an "outbox" table within the same Cloud Spanner transaction. A decoupled worker pool polls or listens to outbox mutations, relays them to Apache Kafka with guaranteed at-least-once delivery, and flags them as published upon broker acknowledgment.',
    highlights: [
      'Atomic entity and event outbox write in single Spanner commit',
      'Zero ghost events or dropped state notifications',
      'Guaranteed at-least-once Kafka streaming with idempotent consumer keys',
    ],
  },
  {
    id: 'tech-faq-websocket-mesh',
    category: 'Real-Time',
    tag: 'WEBSOCKET MESH',
    question: 'How does the WebSocket streaming mesh maintain high availability under mobile network jitter?',
    answer:
      'Our WebSocket infrastructure runs on a distributed Go cluster using an event-driven epoll/kqueue architecture. Clients authenticate with short-lived JWT tokens and subscribe to role- and tenant-partitioned communication rooms. To survive cellular dead zones and Wi-Fi handoffs, the client SDK maintains a client-side monotonic sequence number and an encrypted local ring buffer. Upon reconnection, the client issues a catch-up request with its last received sequence ID, prompting the server to stream missing state deltas before resuming real-time broadcast.',
    highlights: [
      'Epoll/kqueue event loop supports 150,000+ active connections per node',
      'Monotonic sequence IDs guarantee seamless reconnection catch-up',
      'Role-partitioned message channels eliminate broadcast noise',
    ],
  },
  {
    id: 'tech-faq-cvrp-optimization',
    category: 'Algorithms',
    tag: 'CVRP OPTIMIZATION',
    question: 'How does the CVRP route optimization solver handle dynamic midday order insertions?',
    answer:
      'Pegasus treats vehicle routing as a dynamic Capacitated Vehicle Routing Problem with Time Windows (CVRPTW). The solver runs a hybrid optimization algorithm: an initial Clarke-Wright savings heuristic generates feasible cluster routes, followed by GPU-accelerated parallel Large Neighborhood Search (LNS) with ruin-and-recreate operators. When an urgent order is inserted or a driver reports an obstacle, the solver fixes past visited stops, isolates the unvisited sub-graph, and produces an updated route plan within 350 milliseconds.',
    highlights: [
      'Hybrid Clarke-Wright and parallel Large Neighborhood Search (LNS)',
      'Sub-350ms optimization cycles for dynamic order insertions',
      'Hard constraint satisfaction for vehicle cube, weight, and driver hours',
    ],
  },
  {
    id: 'tech-faq-tenant-security',
    category: 'Security',
    tag: 'ZERO-TRUST TENANCY',
    question: 'How is zero-trust multi-tenancy enforced at the database and application layers?',
    answer:
      'Tenant boundaries are cryptographically enforced across every layer of the Pegasus stack. At the API ingress, reverse proxies validate cryptographically signed JWT tokens, extracting the tenant organization ID and injecting it into the Go execution context. Database queries pass through a Spanner mutation interceptor that injects tenant filters into every SQL statement and mutation block. Cloud Spanner row-level security policies ensure that even in the event of an application bug, cross-tenant reads or writes are rejected at the database engine level.',
    highlights: [
      'Cryptographic JWT tenant context injected at API ingress',
      'Automated SQL mutation interceptor enforces tenant WHERE clauses',
      'Cloud Spanner row-level security provides defense-in-depth isolation',
    ],
  },
  {
    id: 'tech-faq-schema-migrations',
    category: 'Infrastructure',
    tag: 'ZERO-DOWNTIME DDL',
    question: 'How does Pegasus manage zero-downtime database schema migrations in a 24/7 operating environment?',
    answer:
      'Because supply chain operations run 24 hours a day across global distribution networks, scheduled maintenance windows are unacceptable. Pegasus executes schema migrations using the expand-and-contract pattern. In the expand phase, new columns or tables are added as nullable or default-valued without altering existing schema. Application code is deployed with dual-write logic. Once historical records are backfilled asynchronously, code is shifted to read from the new structure, and the contract phase drops legacy fields during the subsequent release cycle.',
    highlights: [
      'Expand-and-contract migration strategy eliminates downtime',
      'Dual-write application adapters during transition phases',
      'Asynchronous background backfill with zero performance degradation',
    ],
  },
];

// ============================================================================
// 8. TECHNOLOGY HUB CONFIGURATION (RUSSIAN)
// ============================================================================

const TECHNOLOGY_DIFFERENTIATORS_RU: DifferentiatorCardConfig[] = [
  {
    icon: Terminal,
    badge: 'GO RUNTIME',
    kicker: '[01 // АРХИТЕКТУРА ПРИЛОЖЕНИЯ]',
    title: 'Высокопроизводительный модульный монолит на Go',
    description:
      'Единый высокопроизводительный бэкенд, компилируемый в нативный бинарный код без накладных расходов на рефлексию. Пулы горутин обрабатывают пиковый поток заказов, тарифные сетки и переходы состояний с задержкой p50 менее 2 мс, исключая сетевой оверхед микросервисов.',
    previewType: 'code',
    previewValue: '400+ Handlers / <2ms p50',
  },
  {
    icon: Database,
    badge: 'SPANNER ACID',
    kicker: '[02 // БАЗОВОЕ ХРАНИЛИЩЕ]',
    title: 'Глобально распределенный ACID-реестр Cloud Spanner',
    description:
      'Мультирегиональная база данных Google Cloud Spanner с синхронизацией по TrueTime, выступающая неизменяемым единым источником правды. Гарантирует строгую сериализуемость транзакций на разных континентах с автоматическим консенсусом Paxos и защитой от split-brain.',
    previewType: 'metric',
    previewValue: '99.999% Zero-Split Brain',
  },
  {
    icon: Workflow,
    badge: 'OUTBOX RELAY',
    kicker: '[03 // НАДЕЖНЫЙ ОБМЕН СООБЩЕНИЯМИ]',
    title: 'Гарантированная доставка Transactional Outbox в Kafka',
    description:
      'Устраняет проблему двойной записи. Изменения сущностей и события outbox сохраняются в Spanner в рамках единой атомарной транзакции. Фоновые воркеры считывают очередь и транслируют события в Apache Kafka с гарантией доставки at-least-once.',
    previewType: 'code',
    previewValue: 'Zero Dropped Kafka Events',
  },
  {
    icon: Radio,
    badge: 'WEBSOCKET MESH',
    kicker: '[04 // ПОТОКОВАЯ ТРАНСЛЯЦИЯ]',
    title: 'Ролевая сеть потоковых сокетов WebSocket',
    description:
      'Распределенный кластер WebSocket с мультиплексированием по организациям, ролям и географическим зонам. Обновления рейсов, телеметрия водителей и отметки КПП транслируются на тысячи подключенных клиентов менее чем за 80 миллисекунд.',
    previewType: 'telemetry',
    previewValue: '<80ms Sub-Second Fanout',
  },
  {
    icon: Cpu,
    badge: 'CVRP SOLVER',
    kicker: '[05 // АЛГОРИТМИЧЕСКИЙ ДВИЖОК]',
    title: 'GPU-ускоренная динамическая оптимизация маршрутов CVRP',
    description:
      'Решает задачу маршрутизации транспорта с временными окнами (CVRPTW), 3D-укладкой коробок и ограничениями смен водителей с помощью гибридного алгоритма Кларка-Райта и параллельного поиска в расширенной окрестности (LNS) за доли секунды.',
    previewType: 'telemetry',
    previewValue: '<350ms Heuristic Solve',
  },
  {
    icon: Lock,
    badge: 'ZERO-TRUST TENANCY',
    kicker: '[06 // КОНТУР БЕЗОПАСНОСТИ]',
    title: 'Криптографическая изоляция арендаторов Zero-Trust',
    description:
      'Политики безопасности на уровне строк БД, привязанные к криптографически подписанным токенам JWT, обеспечивают абсолютные границы арендаторов. Конкурирующие бренды на общих складах математически защищены от утечек данных.',
    previewType: 'graph',
    previewValue: 'Strict Tenant Isolation',
  },
];

const TECHNOLOGY_BUSINESS_VALUE_RU: BusinessValueTabConfig[] = [
  {
    id: 'throughput',
    label: 'Пропускная способность',
    stats: [
      {
        value: '24,000+',
        label: 'Скорость записи транзакций',
        delta: '+340% к реляционным СУБД',
        subtext: 'Распределенные коммиты Spanner',
        context: 'Выдерживает пиковый наплыв заказов десятков крупных брендов без троттлинга записи и блокировок.',
        trend: 'up',
      },
      {
        value: '1.8M',
        label: 'Скорость приема телеметрии',
        delta: 'Ноль противодавления',
        subtext: 'Потоковый конвейер Kafka',
        context: 'Непрерывный поток координат GPS, показаний датчиков температуры и CAN-шины обрабатывается параллельно.',
        trend: 'up',
      },
      {
        value: '150,000+',
        label: 'Одновременных активных сокетов',
        delta: 'Менее 80МБ RAM на узел',
        subtext: 'Мультиплексирование на Go',
        context: 'Параллельные подключения терминалов складов, планшетов водителей и веб-порталов ритейлеров.',
        trend: 'up',
      },
      {
        value: '85,000',
        label: 'Строк заказов в минуту',
        delta: '<2.4с От приема до склада',
        subtext: 'Параллельный импорт из ERP',
        context: 'Высокоскоростные конвейеры EDI и REST парсят, валидируют и планируют крупные пакеты поставок.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'latency',
    label: 'Задержка',
    stats: [
      {
        value: '4.2мс',
        label: 'Задержка API p95',
        delta: '-62% к Node/Python',
        subtext: 'Скомпилированный бинарный код Go',
        context: 'Ролевые обработчики API исполняют бизнес-логику и проверки без пауз сборщика мусора.',
        trend: 'up',
      },
      {
        value: '<65мс',
        label: 'Сквозная доставка событий',
        delta: 'Мгновенная синхронизация',
        subtext: 'От коммита БД до экрана',
        context: 'Смена статуса в Spanner транслируется через Outbox и WebSocket на планшет водителя практически мгновенно.',
        trend: 'up',
      },
      {
        value: '<14мс',
        label: 'Захват распределенного мьютекса',
        delta: 'Строгая сериализуемость',
        subtext: 'Блокировка строк в Spanner',
        context: 'Гарантирует атомарное резервирование остатков и исключает двойные продажи в мультиканальной сети.',
        trend: 'up',
      },
      {
        value: '<42мс',
        label: 'Мультирегиональный консенсус',
        delta: 'Глобальная согласованность',
        subtext: 'Синхронизация TrueTime Spanner',
        context: 'Репликация состояния между дата-центрами достигает кворума без аномалий устаревшего чтения.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'consistency',
    label: 'Согласованность',
    stats: [
      {
        value: '100%',
        label: 'Транзакционная сериализуемость',
        delta: 'Ноль "грязных" чтений',
        subtext: 'Аппаратный TrueTime Spanner',
        context: 'Внешняя согласованность: все узлы сети наблюдают транзакции в строго одинаковом хронологическом порядке.',
        trend: 'up',
      },
      {
        value: '0мс',
        label: 'Зазор согласованности в конечном счете',
        delta: 'Ноль дрейфа репликации',
        subtext: 'Чтение строгой согласованности',
        context: 'Устраняет задержки сверки: чтение заказа сразу после изменения всегда возвращает актуальное состояние.',
        trend: 'up',
      },
      {
        value: '0.000%',
        label: 'Потерянных записей Outbox',
        delta: 'Гарантия At-Least-Once',
        subtext: 'Идемпотентные паблишеры Kafka',
        context: 'Паттерн Transactional Outbox гарантирует, что каждая мутация в БД порождает отправку события в брокер.',
        trend: 'up',
      },
      {
        value: '0.000%',
        label: 'Дублирующих широковещаний',
        delta: 'Обработка Exactly-Once',
        subtext: 'Монотонные ключи последовательностей',
        context: 'Устройства клиентов и консьюмеры фильтруют дубликаты сообщений по уникальным монотонным идентификаторам.',
        trend: 'up',
      },
    ],
  },
  {
    id: 'availability',
    label: 'Надежность',
    stats: [
      {
        value: '99.999%',
        label: 'Доступность платформы',
        delta: '<5.2 минут простоя в год',
        subtext: 'Мультирегиональные кластеры',
        context: 'Отказоустойчивая топология вычислительных узлов гарантирует бесперебойную работу логистических цепочек.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Миграции без остановки системы',
        delta: 'Непрерывный деплой CI/CD',
        subtext: 'Схема "расширение-сжатие"',
        context: 'Обновление структуры БД и релизы бэкенда развертываются без остановки работы складов и терминалов.',
        trend: 'up',
      },
      {
        value: '<1.8с',
        label: 'Время аварийного переключения',
        delta: 'Без участия человека',
        subtext: 'Автофейловер DNS и GKE',
        context: 'При сбое зоны доступности облака трафик автоматически переключается на резервные реплики без потерь.',
        trend: 'up',
      },
      {
        value: '100%',
        label: 'Устойчивость к разделению сети',
        delta: 'Математическая надежность',
        subtext: 'Кворум Paxos в Spanner',
        context: 'Сетевые сбои безопасно блокируют конфликтующие записи, предотвращая рассинхронизацию остатков.',
        trend: 'up',
      },
    ],
  },
];

const TECHNOLOGY_CAPABILITIES_RU: CapabilityItemConfig[] = [
  {
    id: 'outbox-kafka-relay',
    title: 'Реле Transactional Outbox в Kafka',
    description:
      'Решает проблему двойной записи путем атомарного сохранения изменений сущностей и очереди сообщений в одной транзакции Spanner с потоковой передачей в Kafka.',
    tag: 'КОНВЕЙЕР СОБЫТИЙ',
    sla: '<50мс Задержка реле',
    href: '/technology/redis-kafka',
    image: '/images/topics/event_pipeline.jpg',
    workflowSteps: ['Мутация сущности', 'Атомарная запись в Outbox', 'Опрос воркером-реле', 'Публикация в Kafka'],
  },
  {
    id: 'spanner-mutation-interceptor',
    title: 'Перехватчик мутаций Spanner',
    description:
      'Перехватывает запросы к БД для проверки прав арендатора, генерации аудиторского следа и проверки монотонных версий до фиксации в консенсусе Cloud Spanner.',
    tag: 'ДВИЖОК ХРАНЕНИЯ',
    sla: '<12мс Перехват и фиксация',
    href: '/technology/cloud-spanner',
    image: '/images/topics/cloud_spanner.jpg',
    workflowSteps: ['Входящий запрос', 'Парсинг токена JWT', 'Валидация мутации', 'Фиксация в Spanner'],
  },
  {
    id: 'sub200ms-telemetry-fanout',
    title: 'Потоковая рассылка телеметрии Sub-200ms',
    description:
      'Высокопроизводительный кластер WebSocket на Go, который принимает координаты GPS и IoT-данные от сотен машин и транслирует обновления на ролевые пульты.',
    tag: 'ПОТОКОВАЯ СЕТЬ',
    sla: '<80мс Задержка рассылки',
    href: '/technology/websocket-hubs',
    image: '/images/topics/fleet_radar.jpg',
    workflowSteps: ['Пинг телеметрии машины', 'Брокер приема данных', 'Фильтрация по ролям', 'Рассылка через WebSocket'],
  },
  {
    id: 'lock-lease-recovery',
    title: 'Восстановление аренды распределенных блокировок',
    description:
      'Отказоустойчивый механизм распределенных блокировок, гарантирующий, что складские зоны и дефицитные остатки никогда не зависнут при аварийном падении воркеров.',
    tag: 'КОНТРОЛЬ КОНКУРЕНТНОСТИ',
    sla: '<2.5с Сброс зависшей аренды',
    href: '/technology/go-backend-platform',
    image: '/images/topics/control_plane.jpg',
    workflowSteps: ['Захват мьютекса', 'Продление аренды heartbeat', 'Таймаут сбоя процесса', 'Безопасное снятие блокировки'],
  },
];

const TECHNOLOGY_FAQS_RU: FaqItemConfig[] = [
  {
    id: 'tech-faq-spanner-selection',
    category: 'База данных',
    tag: 'ГЛОБАЛЬНЫЙ КОНСЕНСУС',
    question: 'Почему Pegasus выбрал Google Cloud Spanner вместо традиционных PostgreSQL или Cassandra?',
    answer:
      'Корпоративные цепочки поставок требуют глобальной масштабируемости в нескольких регионах и строгой сериализуемости ACID. Традиционные реляционные СУБД вроде PostgreSQL не поддерживают мультирегиональную репликацию active-active без задержек и риска split-brain, а NoSQL-хранилища вроде Cassandra жертвуют согласованностью ради доступности. Google Cloud Spanner использует аппаратные атомные часы (TrueTime API) для обеспечения внешней согласованности распределенных транзакций, автоматического шардирования и доступности 99.999%.',
    highlights: [
      'Атомные часы TrueTime гарантируют внешнюю согласованность',
      'Масштабируемость active-active без риска split-brain',
      'Строгая сериализуемость ACID исключает ошибки в остатках',
    ],
  },
  {
    id: 'tech-faq-go-monolith',
    category: 'Архитектура',
    tag: 'GO RUNTIME',
    question: 'В чем преимущества модульного монолита на Go перед микросервисной архитектурой в логистике?',
    answer:
      'Микросервисы создают задержки сериализации по сети, усложняют распределенную трассировку и порождают каскадные сбои при сотнях удаленных вызовов RPC. Pegasus спроектирован как модульный монолит на языке Go. Границы доменов (диспетчеризация, склад, маршруты, платежи) строго изолированы на уровне пакетов и интерфейсов компилятора, но исполняются внутри единого бинарного файла. Это дает задержку внутренних вызовов менее 2 мс, простоту тестирования и минимальные затраты на инфраструктуру.',
    highlights: [
      'Внутренние вызовы функций быстрее 2 мс без накладных расходов RPC',
      'Строгие границы доменов проверяются компилятором Go',
      'Единый бинарный файл снижает требования к инфраструктуре',
    ],
  },
  {
    id: 'tech-faq-outbox-pattern',
    category: 'Потоковые данные',
    tag: 'TRANSACTIONAL OUTBOX',
    question: 'Как паттерн Transactional Outbox предотвращает рассинхронизацию данных (dual-write)?',
    answer:
      'В распределенных системах раздельное сохранение данных в базу и отправка события в очередь сообщений часто приводит к сбоям: если брокер недоступен после коммита, подписчики теряют событие; если произошел откат транзакции после отправки в брокер, возникают события-призраки. Pegasus записывает бизнес-сущность и конверт события в специальную таблицу outbox в рамках одной транзакции Spanner. Независимый пул воркеров считывает записи из outbox и публикует их в Apache Kafka с гарантией at-least-once.',
    highlights: [
      'Атомарная запись бизнес-сущности и события в одной транзакции Spanner',
      'Полное исключение событий-призраков и потери сообщений',
      'Гарантированная отправка в Kafka с дедупликацией на стороне подписчиков',
    ],
  },
  {
    id: 'tech-faq-websocket-mesh',
    category: 'Реальное время',
    tag: 'WEBSOCKET MESH',
    question: 'Как сеть сокетов WebSocket сохраняет устойчивость при нестабильной мобильной связи?',
    answer:
      'Кластер WebSocket работает на событийно-ориентированном цикле epoll/kqueue на языке Go. Клиенты авторизуются через короткоживущие токены JWT и подписываются на каналы своих ролей и подразделений. При обрыве связи мобильный SDK сохраняет монотонный номер последнего принятого сообщения в локальном зашифрованном буфере. При восстановлении соединения клиент запрашивает досылку пропущенных дельт по своему ID последовательности, после чего возобновляется обычная трансляция.',
    highlights: [
      'Событийный цикл epoll/kqueue держит более 150 000 соединений на узел',
      'Монотонные номера сообщений гарантируют досылку пропущенных данных',
      'Ролевые каналы исключают лишний сетевой шум на устройствах',
    ],
  },
  {
    id: 'tech-faq-cvrp-optimization',
    category: 'Алгоритмы',
    tag: 'ОПТИМИЗАЦИЯ CVRP',
    question: 'Как алгоритм CVRP справляется с добавлением срочных заказов в середине рабочего дня?',
    answer:
      'Pegasus решает задачу маршрутизации транспорта с временными окнами (CVRPTW). Оптимизатор использует гибридный метод: алгоритм Кларка-Райта быстро строит начальные маршруты, после чего параллельный алгоритм Large Neighborhood Search (LNS) на GPU оптимизирует траектории. При поступлении срочного заказа решатель фиксирует уже пройденные водителем точки, изолирует оставшуюся подсеть и находит обновленный маршрут менее чем за 350 миллисекунд.',
    highlights: [
      'Гибрид алгоритма Кларка-Райта и параллельного поиска LNS на GPU',
      'Пересчет маршрутов менее чем за 350 мс при внезапных изменениях',
      'Строгий учет кубатуры кузова, осевых нагрузок и режима труда водителей',
    ],
  },
  {
    id: 'tech-faq-tenant-security',
    category: 'Безопасность',
    tag: 'ИЗОЛЯЦИЯ АРЕНДАТОРОВ',
    question: 'Как реализуется изоляция арендаторов Zero-Trust на уровне базы данных и приложения?',
    answer:
      'Границы арендаторов контролируются криптографически на каждом уровне архитектуры. На входе в API прокси проверяет подпись токена JWT, извлекает ID организации и передает его в контекст выполнения Go. Любые запросы к базе данных проходят через перехватчик мутаций, который внедряет условие проверки арендатора в каждый запрос SQL. Политики безопасности на уровне строк Cloud Spanner гарантируют отклонение любых межклиентских обращений на уровне ядра СУБД.',
    highlights: [
      'Контекст арендатора извлекается из криптографического токена JWT',
      'Автоматический перехватчик внедряет фильтр арендатора во все запросы',
      'Политики безопасности на уровне строк Spanner гарантируют защиту данных',
    ],
  },
  {
    id: 'tech-faq-schema-migrations',
    category: 'Инфраструктура',
    tag: 'МИГРАЦИИ БЕЗ ПРОСТОЯ',
    question: 'Как осуществляются миграции схемы базы данных без простоя в круглосуточном режиме работы?',
    answer:
      'Логистические цепочки работают 24/7 по всему миру, поэтому технологические окна для обслуживания недопустимы. Pegasus выполняет миграции схемы по паттерну "расширение-сжатие" (expand-and-contract). На этапе расширения добавляются новые столбцы или таблицы со значениями по умолчанию без изменения старых. Приложение развертывается с логикой двойной записи. После фонового заполнения исторических данных чтение переключается на новую структуру, а устаревшие поля удаляются в следующем релизе.',
    highlights: [
      'Паттерн "расширение-сжатие" исключает простой системы при миграциях',
      'Адаптеры двойной записи в коде приложения на переходный период',
      'Асинхронная фоновая миграция исторических данных без падения скорости',
    ],
  },
];

// ============================================================================
// 9. ASSEMBLED CONFIGURATIONS (BILINGUAL)
// ============================================================================

export const hubLayoutConfigsEn: Record<string, HubLayoutConfig> = {
  platform: {
    visual: 'kpi',
    topicGridLayout: 'featured',
    showPromoInHero: true,
    laneLabel: 'Control plane',
    laneIndex: '01',
    intro: {
      eyebrow: 'Platform',
      title: 'One control plane across six roles',
      body: 'Order lifecycle, payments, topology, and treasury on one shared record — so every role sees the same confirmed status.',
    },
    differentiators: {
      kicker: 'KEY DIFFERENTIATORS',
      title: 'Why Industry Leaders Choose Pegasus Platform',
      description:
        'Deterministic state execution, cryptographic tenant isolation, and zero handoff gaps across all six network roles.',
      cards: PLATFORM_DIFFERENTIATORS_EN,
    },
    businessValue: {
      kicker: 'QUANTIFIED BUSINESS VALUE',
      title: 'Measurable Operational & Capital Outcomes',
      description:
        'Verified benchmarks across network velocity, stockout prevention, touchless dispatch, and working capital release.',
      tabs: PLATFORM_BUSINESS_VALUE_EN,
    },
    capabilities: {
      kicker: 'OPERATIONAL USE CASES',
      title: 'End-to-End Execution Modules',
      description:
        'Mission-critical orchestration workflows connecting suppliers, warehouses, drivers, and financial settlement.',
      items: PLATFORM_CAPABILITIES_EN,
    },
    faq: {
      kicker: 'ENTERPRISE ARCHITECTURE FAQ',
      title: 'Frequently Answered Questions on Integration & Reliability',
      description:
        'Technical details regarding legacy ERP/WMS overlays, cryptographic tenant isolation, fault-tolerant offline sync, and rollout timelines.',
      items: PLATFORM_FAQS_EN,
    },
  },
  capabilities: {
    visual: 'kpi',
    topicGridLayout: 'masonry',
    laneLabel: 'Features',
    laneIndex: '04',
    intro: {
      eyebrow: 'Capabilities',
      title: 'What the network runs on',
      body: 'Smarter dispatch, reliable updates, pay-at-delivery, live fleet tracking, and topology — modules that work the same on portal and native apps.',
    },
    differentiators: {
      kicker: 'EXECUTION CAPABILITIES',
      title: 'Why Fleet & Logistics Leaders Choose Pegasus',
      description:
        'Algorithmic vehicle routing, volumetric trailer packing, and IoT cold-chain telemetry engineered for zero-margin-leak operations.',
      cards: CAPABILITIES_DIFFERENTIATORS_EN,
    },
    businessValue: {
      kicker: 'QUANTIFIED FLEET IMPACT',
      title: 'Measurable Transport & Execution Outcomes',
      description:
        'Verified metrics across empty-mile elimination, trailer cube utilization, on-time delivery rates, and cold-chain compliance.',
      tabs: CAPABILITIES_BUSINESS_VALUE_EN,
    },
    capabilities: {
      kicker: 'OPERATIONAL CAPABILITIES',
      title: 'Fleet & Dispatch Execution Modules',
      description:
        'Modular capabilities designed for high-density delivery zones, mixed-freight loading, and tamper-proof electronic custody transfer.',
      items: CAPABILITIES_SHOWCASE_EN,
    },
    faq: {
      kicker: 'CAPABILITIES & DISPATCH FAQ',
      title: 'Technical Questions on Routing, Telematics & Packing',
      description:
        'In-depth answers on CVRP algorithms, 3D volumetric bin-packing heuristics, cold-chain IoT hardware, and ruggedized scanner support.',
      items: CAPABILITIES_FAQS_EN,
    },
  },
  technology: {
    visual: 'flow',
    flowVariant: 'mutatingHandler',
    topicGridLayout: 'uniform',
    showPromoInHero: true,
    laneLabel: 'Engineering',
    laneIndex: '02',
    intro: {
      eyebrow: 'Architecture',
      title: 'High-Throughput Go Core, Spanner ACID & Live Streaming',
      body: 'Engineered for sub-second synchronization, guaranteed message delivery, and cryptographic tenant isolation across high-volume global supply chains.',
    },
    differentiators: {
      kicker: 'ARCHITECTURAL PILLARS',
      title: 'Why Engineers Choose Pegasus Distributed Core',
      description:
        'Native Go concurrency, Google Cloud Spanner global ACID transactions, transactional outbox relays, and sub-second WebSocket telemetry fanout.',
      cards: TECHNOLOGY_DIFFERENTIATORS_EN,
    },
    businessValue: {
      kicker: 'SYSTEM PERFORMANCE BENCHMARKS',
      title: 'Quantified Throughput, Latency, Consistency & Availability',
      description:
        'Empirically tested benchmarks demonstrating massive mutation throughput, sub-65ms event fanout, zero replication drift, and five-nines uptime.',
      tabs: TECHNOLOGY_BUSINESS_VALUE_EN,
    },
    capabilities: {
      kicker: 'CORE ARCHITECTURAL MECHANISMS',
      title: 'Distributed Systems Patterns in Production',
      description:
        'Deep-dive into transactional outbox pipelines, Spanner mutation interceptors, role-partitioned WebSocket clusters, and distributed lock recovery.',
      items: TECHNOLOGY_CAPABILITIES_EN,
    },
    faq: {
      kicker: 'ENGINEERING FAQ',
      title: 'Distributed Systems, Consensus & Data Integrity Architecture',
      description:
        'Technical breakdowns of TrueTime consensus, Go modular monolith design, WebSocket connection multiplexing, and zero-downtime schema evolution.',
      items: TECHNOLOGY_FAQS_EN,
    },
  },
  operations: {
    visual: 'flow',
    flowVariant: 'dispatchBoard',
    topicGridLayout: 'masonry',
    showPromoInHero: true,
    laneLabel: 'Live ops',
    laneIndex: '03',
    intro: {
      eyebrow: 'Operations',
      title: 'Real-Time Floor Execution Across 6 Roles',
      body: 'Visual dispatch boards, volumetric Smart Fit overflow, shop-closed playbooks, and gate security — deterministic operational execution grounded in the order lifecycle.',
    },
    differentiators: {
      kicker: 'OPERATIONAL DIFFERENTIATORS',
      title: '6 Unified Roles, 1 Operational Truth',
      description:
        'Eliminate communication silos across headquarters, warehouses, production bays, delivery drivers, retail receivers, and gate terminals.',
      cards: OPERATIONS_DIFFERENTIATORS_EN,
    },
    businessValue: {
      kicker: 'QUANTIFIED OPERATIONAL IMPACT',
      title: 'Measurable Velocity, Fulfillment & Cash Flow Outcomes',
      description:
        'Verified floor benchmarks across touchless dispatch, OTIF delivery adherence, shop-closed recovery, and cash-at-door settlement.',
      tabs: OPERATIONS_BUSINESS_VALUE_EN,
    },
    capabilities: {
      kicker: 'OPERATIONAL PLAYBOOKS',
      title: 'Guarded Workflows for Real-World Edge Cases',
      description:
        'Battle-tested operational playbooks designed to recover from unexpected shop closures, stock contention, driver illness, and gate seal discrepancies.',
      items: OPERATIONS_CAPABILITIES_EN,
    },
    faq: {
      kicker: 'OPERATIONS FAQ',
      title: 'Technical Execution, Floor Safety & Exception Handling',
      description:
        'Detailed answers on warehouse wave coordination, COD reconciliation, driver reassignment, and gate pass tamper security.',
      items: OPERATIONS_FAQS_EN,
    },
  },
  'ai-vision': {
    visual: 'metrics',
    topicGridLayout: 'featured',
    laneLabel: 'Governed AI',
    laneIndex: '05',
    intro: {
      eyebrow: 'AI',
      title: 'Assist with proven fallback',
      body: 'Route assist and recommendations never block the floor — if AI is slow, proven planning rules take over.',
    },
  },
  'apps-deploy': {
    visual: 'devices',
    topicGridLayout: 'uniform',
    showFleetBand: true,
    showPromoInHero: true,
    laneLabel: 'Surfaces',
    laneIndex: '06',
    intro: {
      eyebrow: 'Deploy',
      title: 'Portal, mobile, and desktop for every role',
      body: 'Supplier and warehouse portals, driver mobile apps, retailer desktop, gate terminals — same workflows, automatic live refresh.',
    },
  },
  roles: {
    visual: 'flow',
    flowVariant: 'roleJourney',
    topicGridLayout: 'featured',
    intro: {
      eyebrow: 'Roles',
      title: 'Six roles, one order truth',
      body: 'Supplier, warehouse, factory, driver, retailer, and payload/gate — each surface mapped with clear features and role-based access.',
    },
  },
};

export const hubLayoutConfigsRu: Record<string, HubLayoutConfig> = {
  platform: {
    visual: 'kpi',
    topicGridLayout: 'featured',
    showPromoInHero: true,
    laneLabel: 'Контур управления',
    laneIndex: '01',
    intro: {
      eyebrow: 'Платформа',
      title: 'Единая панель управления для шести ролей',
      body: 'Жизненный цикл заказа, платежи, топология сети и казначейство в едином источнике правды — каждая роль видит согласованное и подтвержденное состояние.',
    },
    differentiators: {
      kicker: 'КЛЮЧЕВЫЕ ДИФФЕРЕНЦИАТОРЫ',
      title: 'Почему отраслевые лидеры выбирают платформу Pegasus',
      description:
        'Детерминированное ядро исполнения, криптографическая изоляция арендаторов и отсутствие разрывов между шестью ролями сети.',
      cards: PLATFORM_DIFFERENTIATORS_RU,
    },
    businessValue: {
      kicker: 'ИЗМЕРИМЫЙ БИЗНЕС-ЭФФЕКТ',
      title: 'Измеримые результаты внедрения на платформе Pegasus',
      description:
        'Проверенные показатели скорости сети, устранения дефицита, автоматической диспетчеризации и высвобождения оборотного капитала.',
      tabs: PLATFORM_BUSINESS_VALUE_RU,
    },
    capabilities: {
      kicker: 'ОПЕРАЦИОННЫЕ МОДУЛИ',
      title: 'Сквозные модули исполнения заказов',
      description:
        'Критически важные рабочие процессы, связывающие поставщиков, склады, водителей и финансовые взаиморасчеты.',
      items: PLATFORM_CAPABILITIES_RU,
    },
    faq: {
      kicker: 'ВОПРОСЫ И ОТВЕТЫ ПО АРХИТЕКТУРЕ',
      title: 'Часто задаваемые вопросы об интеграции и надежности',
      description:
        'Технические детали интеграции с существующими ERP/WMS, изоляции арендаторов, отказоустойчивости в офлайне и сроков внедрения.',
      items: PLATFORM_FAQS_RU,
    },
  },
  capabilities: {
    visual: 'kpi',
    topicGridLayout: 'masonry',
    laneLabel: 'Возможности',
    laneIndex: '04',
    intro: {
      eyebrow: 'Возможности',
      title: 'Функциональный стек системы',
      body: 'Интеллектуальная диспетчеризация, надежные обновления, оплата при доставке, мониторинг автопарка и топология — модули одинаково стабильны в веб-интерфейсах и мобильных приложениях.',
    },
    differentiators: {
      kicker: 'ВОЗМОЖНОСТИ ИСПОЛНЕНИЯ',
      title: 'Почему лидеры логистики выбирают технологии Pegasus',
      description:
        'Алгоритмическая маршрутизация парка, объемная 3D-укладка полуприцепов и IoT-телеметрия холодной цепи для минимизации потерь.',
      cards: CAPABILITIES_DIFFERENTIATORS_RU,
    },
    businessValue: {
      kicker: 'ИЗМЕРИМЫЙ ЭФФЕКТ ДЛЯ АВТОПАРКА',
      title: 'Измеримые показатели транспортировки и исполнения',
      description:
        'Подтвержденные метрики сокращения холостого пробега, коэффициента заполнения кузова, своевременности доставок и сохранности грузов.',
      tabs: CAPABILITIES_BUSINESS_VALUE_RU,
    },
    capabilities: {
      kicker: 'ОПЕРАЦИОННЫЕ ВОЗМОЖНОСТИ',
      title: 'Модули диспетчеризации и управления автопарком',
      description:
        'Модульные компоненты для плотных зон городской доставки, смешанной загрузки фур и защищенной электронной передачи материальной ответственности.',
      items: CAPABILITIES_SHOWCASE_RU,
    },
    faq: {
      kicker: 'ВОПРОСЫ И ОТВЕТЫ ПО ВОЗМОЖНОСТЯМ',
      title: 'Технические вопросы по маршрутизации, телематике и укладке',
      description:
        'Подробные ответы по математическим алгоритмам CVRPTW, 3D-эвристике раскладки коробок, оборудованию температурного контроля и ТСД.',
      items: CAPABILITIES_FAQS_RU,
    },
  },
  technology: {
    visual: 'flow',
    flowVariant: 'mutatingHandler',
    topicGridLayout: 'uniform',
    showPromoInHero: true,
    laneLabel: 'Инженерия',
    laneIndex: '02',
    intro: {
      eyebrow: 'Стек',
      title: 'Высокопроизводительное ядро Go, ACID Spanner и потоковая синхронизация',
      body: 'Спроектировано для субсекундной синхронизации, гарантированной доставки сообщений и криптографической изоляции арендаторов в глобальных цепочках поставок.',
    },
    differentiators: {
      kicker: 'АРХИТЕКТУРНЫЕ СТОЛПЫ',
      title: 'Почему инженеры выбирают распределенное ядро Pegasus',
      description:
        'Параллелизм на Go, глобальные транзакции Google Cloud Spanner, надежные реле Transactional Outbox и мгновенная рассылка телеметрии по WebSocket.',
      cards: TECHNOLOGY_DIFFERENTIATORS_RU,
    },
    businessValue: {
      kicker: 'ПОКАЗАТЕЛИ ПРОИЗВОДИТЕЛЬНОСТИ',
      title: 'Измеримая пропускная способность, задержка, согласованность и надежность',
      description:
        'Эмпирические тесты подтверждают высокую скорость записи, доставку событий за 65 мс, нулевой дрейф репликации и доступность пять девяток.',
      tabs: TECHNOLOGY_BUSINESS_VALUE_RU,
    },
    capabilities: {
      kicker: 'АРХИТЕКТУРНЫЕ МЕХАНИЗМЫ',
      title: 'Паттерны распределенных систем в производственной среде',
      description:
        'Детальный разбор конвейера Transactional Outbox, перехватчиков мутаций Spanner, ролевых сокетов WebSocket и восстановления распределенных блокировок.',
      items: TECHNOLOGY_CAPABILITIES_RU,
    },
    faq: {
      kicker: 'ИНЖЕНЕРНЫЕ ВОПРОСЫ И ОТВЕТЫ',
      title: 'Архитектура распределенных систем, консенсус и целостность данных',
      description:
        'Технический разбор консенсуса TrueTime, монолита на Go, мультиплексирования WebSocket и эволюции схемы БД без остановки работы.',
      items: TECHNOLOGY_FAQS_RU,
    },
  },
  operations: {
    visual: 'flow',
    flowVariant: 'dispatchBoard',
    topicGridLayout: 'masonry',
    showPromoInHero: true,
    laneLabel: 'Операции',
    laneIndex: '03',
    intro: {
      eyebrow: 'Склад и рейс',
      title: 'Сквозное исполнение операций для шести ролей',
      body: 'Визуальные доски диспетчеризации, распределение излишков Smart Fit, регламенты закрытых точек и защита КПП — детерминированные процессы на основе жизненного цикла заказа.',
    },
    differentiators: {
      kicker: 'ОПЕРАЦИОННЫЕ ДИФФЕРЕНЦИАТОРЫ',
      title: 'Шесть ролей, один источник операционной правды',
      description:
        'Устранение разрывов связи между штаб-квартирой, складом, фабрикой, водителями, розничными точками и терминалами КПП.',
      cards: OPERATIONS_DIFFERENTIATORS_RU,
    },
    businessValue: {
      kicker: 'ИЗМЕРИМЫЙ ОПЕРАЦИОННЫЙ ЭФФЕКТ',
      title: 'Измеримые результаты скорости, точности и денежного потока',
      description:
        'Подтвержденные показатели автоматической диспетчеризации, соблюдения OTIF, разрешения исключений и моментальной инкассации.',
      tabs: OPERATIONS_BUSINESS_VALUE_RU,
    },
    capabilities: {
      kicker: 'ОПЕРАЦИОННЫЕ РЕГЛАМЕНТЫ',
      title: 'Защищенные сценарии для реальных нештатных ситуаций',
      description:
        'Проверенные регламенты восстановления при закрытии магазинов, пиковом спросе на остатки, болезни водителей и несовпадении пломб.',
      items: OPERATIONS_CAPABILITIES_RU,
    },
    faq: {
      kicker: 'ВОПРОСЫ И ОТВЕТЫ ПО ОПЕРАЦИЯМ',
      title: 'Исполнение процессов, безопасность склада и обработка исключений',
      description:
        'Подробные ответы по согласованию волн сборки, сверке наличных платежей, замене экипажа в рейсе и контролю пломб на КПП.',
      items: OPERATIONS_FAQS_RU,
    },
  },
  'ai-vision': {
    visual: 'metrics',
    topicGridLayout: 'featured',
    laneLabel: 'Контролируемый ИИ',
    laneIndex: '05',
    intro: {
      eyebrow: 'ИИ',
      title: 'Ассистент с надежным резервным сценарием',
      body: 'Подсказки маршрутов и рекомендации никогда не блокируют склад — при задержке нейросетей управление перехватывают проверенные правила планирования.',
    },
  },
  'apps-deploy': {
    visual: 'devices',
    topicGridLayout: 'uniform',
    showFleetBand: true,
    showPromoInHero: true,
    laneLabel: 'Интерфейсы',
    laneIndex: '06',
    intro: {
      eyebrow: 'Развертывание',
      title: 'Портал, мобильные и десктоп приложения для каждой роли',
      body: 'Порталы поставщика и склада, мобильные приложения водителей, десктоп ритейлера, терминалы КПП — одинаковые процессы с автоматическим обновлением.',
    },
  },
  roles: {
    visual: 'flow',
    flowVariant: 'roleJourney',
    topicGridLayout: 'featured',
    intro: {
      eyebrow: 'Роли',
      title: 'Шесть ролей, один источник правды по заказу',
      body: 'Поставщик, склад, фабрика, водитель, ритейлер и терминал КПП — для каждого рабочего места определены свои функции и ролевой доступ.',
    },
  },
};

/**
 * Resolves the configuration for a category hub in the active language.
 */
export function getHubLayoutConfig(hubId: string, lang = 'en'): HubLayoutConfig {
  const isRu = lang === 'ru';
  const configs = isRu ? hubLayoutConfigsRu : hubLayoutConfigsEn;
  return (
    configs[hubId] ??
    hubLayoutConfigsEn[hubId] ?? {
      visual: 'kpi',
      topicGridLayout: 'uniform',
    }
  );
}

// Backward compatibility for static imports
export const hubLayoutConfigs = hubLayoutConfigsEn;
