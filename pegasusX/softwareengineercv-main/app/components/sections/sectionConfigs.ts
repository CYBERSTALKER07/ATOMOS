export type PillarItem = {
  title: string;
  description: string;
  iconName: 'database' | 'cpu' | 'shield' | 'network' | 'layers' | 'lock' | 'git-branch' | 'zap' | 'truck' | 'workflow';
  accentColor?: string;
};

export type MatrixNode = {
  id: string;
  label: string;
  sublabel: string;
  status: 'active' | 'synced' | 'routing' | 'verified';
};

export type TacticalPillarsConfig = {
  eyebrow: string;
  title: string;
  description: string;
  specSummary: string;
  items: [PillarItem, PillarItem, PillarItem, PillarItem];
  protocolBlock: {
    tag: string;
    title: string;
    description: string;
    whitepaperText: string;
    whitepaperHref: string;
    nodes: MatrixNode[];
  };
};

export type TimelineBranch = {
  name: string;
  status: string;
  statusColor: 'emerald' | 'cyan' | 'amber' | 'blue';
  commitHash: string;
  message: string;
  author: string;
  metric: string;
  timeIndex: number; // 0 to 4
};

export type BranchingTimelineConfig = {
  eyebrow: string;
  title: string;
  description: string;
  mainBranchLabel: string;
  timestamps: [string, string, string, string, string];
  branches: [TimelineBranch, TimelineBranch, TimelineBranch];
  summaryStat: {
    value: string;
    label: string;
  };
};

export type RadialSatellite = {
  id: string;
  name: string;
  subtitle: string;
  status: string;
  iconName: 'cpu' | 'shield' | 'zap' | 'truck' | 'barcode' | 'file-check' | 'satellite' | 'database' | 'layers' | 'network' | 'workflow';
  accent: 'emerald' | 'cyan' | 'amber' | 'blue' | 'purple';
};

export type RadialHubSpokeConfig = {
  badge: string;
  title: string;
  description: string;
  primaryBtn: {
    label: string;
    href: string;
  };
  secondaryBtn: {
    label: string;
    href: string;
  };
  techBadges: string[];
  centerNode: {
    title: string;
    subtitle: string;
    latency: string;
  };
  satellites: [
    RadialSatellite,
    RadialSatellite,
    RadialSatellite,
    RadialSatellite,
    RadialSatellite,
    RadialSatellite
  ];
};

export type SubpageSectionConfig = {
  pillars: TacticalPillarsConfig;
  timeline: BranchingTimelineConfig;
  radial: RadialHubSpokeConfig;
};

export const SECTION_PAGE_CONFIGS: Record<string, SubpageSectionConfig> = {
  // 1. Logistics Automation (/logistics-automation)
  'logistics-automation': {
    pillars: {
      eyebrow: 'WHY PEGASUS AUTOMATION',
      title: 'DETERMINISTIC LOGISTICS ORCHESTRATION WITH SUB-SECOND PRECISION',
      description:
        'Engineered to eliminate manual dispatch bottlenecks, prevent yard gate leakage, and mathematically optimize fleet routing under strict legal SLAs.',
      specSummary: 'Google OR-Tools CVRP • Barcode Seal Verification • Under 100ms PoD Settlement',
      items: [
        {
          title: 'Algorithmic CVRP Dispatch',
          description: 'Groups pending orders by volume, payload GVW, and recipient delivery windows within 3 seconds.',
          iconName: 'cpu',
          accentColor: '#3B82F6',
        },
        {
          title: 'Tamper-Evident Gate Barcodes',
          description: 'Digital seal scanning ensures only verified vehicles leave the warehouse yard with matching manifests.',
          iconName: 'shield',
          accentColor: '#10B981',
        },
        {
          title: 'Simultaneous Ledger Settlement',
          description: 'Proof of delivery instantly reconciles cash collection, inventory stock, and invoice status in one atomic write.',
          iconName: 'database',
          accentColor: '#F59E0B',
        },
        {
          title: 'Offline Telemetry Sync',
          description: 'Drivers execute full stop-order workflows in dead zones, syncing cryptographically verified packets on reconnect.',
          iconName: 'network',
          accentColor: '#06B6D4',
        },
      ],
      protocolBlock: {
        tag: 'DETERMINISTIC PROTOCOL SPLIT',
        title: 'Zero Split-Brain Outbox Architecture',
        description:
          'State transitions are persisted directly alongside business mutations in atomic Spanner/Postgres outbox tables before streaming to Apache Kafka and Redis cluster brokers.',
        whitepaperText: 'Read the Outbox Protocol Whitepaper',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'OR-OPT', sublabel: 'CVRP Engine', status: 'active' },
          { id: '02', label: 'GATE-CHK', sublabel: 'Digital Seal', status: 'verified' },
          { id: '03', label: 'TX-LEDGER', sublabel: 'Double-Entry', status: 'synced' },
          { id: '04', label: 'POD-SYNC', sublabel: 'Glass Signature', status: 'active' },
          { id: '05', label: 'KFK-OUT', sublabel: 'Zero-Loss Bus', status: 'routing' },
          { id: '06', label: 'EDGE-TAS', sublabel: 'TAS-IX Peering', status: 'active' },
        ],
      },
    },
    timeline: {
      eyebrow: 'REVERSIBLE ORCHESTRATION',
      title: 'Instant Branching & Safe Route Simulation',
      description:
        'Safely simulate alternative dispatch routes, recalculate CVRP capacity on sudden order spikes, and merge optimized routes into active driver shifts with zero operational downtime.',
      mainBranchLabel: 'production / active-fleet',
      timestamps: ['08:00 AM (Dispatched)', '09:15 AM (En Route)', '10:30 AM (Checkpoint)', '11:45 AM (Geofenced)', '12:30 PM (Settled)'],
      branches: [
        {
          name: 'sim-route/samarkand-bypass',
          status: 'ACTIVE RE-ROUTE',
          statusColor: 'amber',
          commitHash: 'c8f309a',
          message: 'Avoid M39 traffic corridor; reroute 14 stops',
          author: 'OR-TOOLS ENGINE',
          metric: '-18 min ETA delta',
          timeIndex: 1,
        },
        {
          name: 'cvrp/payload-redistribute',
          status: 'OPTIMIZED',
          statusColor: 'cyan',
          commitHash: 'd7a110e',
          message: 'Transfer 400kg payload to secondary van at hub 02',
          author: 'DISPATCH AGENT',
          metric: '94.2% load factor',
          timeIndex: 2,
        },
        {
          name: 'audit/spanner-reconcile',
          status: 'VERIFIED',
          statusColor: 'emerald',
          commitHash: 'f49b882',
          message: 'Zero-drift invariant check against physical cash drawer',
          author: 'LEDGER WORKER',
          metric: '0.00% balance diff',
          timeIndex: 3,
        },
      ],
      summaryStat: {
        value: '100%',
        label: 'Divergence-Free Execution',
      },
    },
    radial: {
      badge: 'AUTOMATIONS UNLEASHED',
      title: 'Automations Unleashed: End-to-End Autonomous Dispatch',
      description:
        'Every critical milestone across distribution yards, carrier telematics, and retailer billing is connected into a unified real-time event graph. Eliminate coordination lag and operate with mathematical precision.',
      primaryBtn: {
        label: 'Launch Dispatch Console',
        href: '/contact',
      },
      secondaryBtn: {
        label: 'View System Blueprint',
        href: '/technology',
      },
      techBadges: ['OR-TOOLS CVRP', 'SPANNER OUTBOX', 'POSTGIS 3.4', 'KAFKA STREAMS', 'REDIS 7 CLUSTER', 'CLICKHOUSE OLAP'],
      centerNode: {
        title: 'CORE DISPATCH ENGINE',
        subtitle: 'Autonomous Routing Node',
        latency: '3.4ms RTT',
      },
      satellites: [
        { id: 'sat-1', name: 'CVRP Optimizer', subtitle: 'Google OR-Tools Engine', status: 'LIVE', iconName: 'cpu', accent: 'cyan' },
        { id: 'sat-2', name: 'Digital Seal Gate', subtitle: 'Tamper Barcode Scans', status: 'ACTIVE', iconName: 'barcode', accent: 'emerald' },
        { id: 'sat-3', name: 'PoD Settlement', subtitle: 'Glass Signature & Cash', status: 'INSTANT', iconName: 'file-check', accent: 'blue' },
        { id: 'sat-4', name: 'Carrier Telemetry', subtitle: 'Sub-second GPS Geofence', status: 'STREAMING', iconName: 'truck', accent: 'amber' },
        { id: 'sat-5', name: 'Double-Entry Ledger', subtitle: 'Strict Minor-Unit Cash', status: 'VERIFIED', iconName: 'database', accent: 'purple' },
        { id: 'sat-6', name: 'Transactional Outbox', subtitle: 'Zero Split-Brain Kafka', status: 'REPLICATED', iconName: 'zap', accent: 'emerald' },
      ],
    },
  },

  // 2. Global Logistics (/global-logistics)
  'global-logistics': {
    pillars: {
      eyebrow: 'MULTI-COUNTRY TOPOLOGY',
      title: 'HYPER-SCALE SUPPLY NETWORK WITH LOCAL-FIRST SOVEREIGNTY',
      description:
        'Run cross-border distribution networks with global Spanner multi-tenant clusters alongside local cell isolation for sovereign in-country data residency.',
      specSummary: 'Global Spanner Mesh • Maglev Consistent Hashing • In-Country Cell Replicas',
      items: [
        {
          title: 'Cell-Based Sharding',
          description: 'Tenant workloads are partitioned into autonomous regional cells (`cell-uz`, `cell-eu`) with independent failover.',
          iconName: 'layers',
          accentColor: '#3B82F6',
        },
        {
          title: 'Maglev Consistent Hashing',
          description: 'Deterministic traffic distribution ensures zero packet starvation during dynamic container rebalancing.',
          iconName: 'network',
          accentColor: '#06B6D4',
        },
        {
          title: 'Sovereign Data Fencing',
          description: 'Regulatory compliance enforced at database level with geo-partitioned rows and hardware-isolated keys.',
          iconName: 'lock',
          accentColor: '#10B981',
        },
        {
          title: 'Cross-Border Exchange',
          description: 'Automated multi-currency rate locks with microsecond double-entry ledger postings across banking rails.',
          iconName: 'database',
          accentColor: '#F59E0B',
        },
      ],
      protocolBlock: {
        tag: 'GLOBAL CONDUIT ARCHITECTURE',
        title: 'Distributed State Across 3 Continents',
        description:
          'Synchronous read-write transactions execute in under 12 milliseconds across Google Cloud Spanner multi-region backbones with zero downtime upgrades.',
        whitepaperText: 'Read the Multi-Region Whitepaper',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'CELL-EU', sublabel: 'Frankfurt Core', status: 'active' },
          { id: '02', label: 'CELL-UZ', sublabel: 'Tashkent Sovereign', status: 'active' },
          { id: '03', label: 'CELL-US', sublabel: 'Virginia Ingress', status: 'synced' },
          { id: '04', label: 'MAGLEV', sublabel: 'Consistent Hash', status: 'verified' },
          { id: '05', label: 'SPANNER', sublabel: 'TrueTime Consensus', status: 'active' },
          { id: '06', label: 'KFK-GEO', sublabel: 'MirrorMaker 2', status: 'routing' },
        ],
      },
    },
    timeline: {
      eyebrow: 'CONTINUOUS RESILIENCE',
      title: 'Dynamic Cross-Zone Failover & Traffic Shifting',
      description:
        'Simulate instantaneous cloud zone outages, verify zero-loss data failover, and reroute fleet requests across backup transit nodes without dropping a single active delivery session.',
      mainBranchLabel: 'global-traffic / primary-backbone',
      timestamps: ['00:00 (Nominal)', '00:02 (Zone Degraded)', '00:04 (Failover Triggered)', '00:07 (Rerouted)', '00:10 (Restored)'],
      branches: [
        {
          name: 'failover/zone-eu-central',
          status: 'SIMULATED FAILOVER',
          statusColor: 'amber',
          commitHash: 'a109ec4',
          message: 'Simulate az-3 disconnect; shift 180k req/s to az-1 & az-2',
          author: 'CHAOS MESH',
          metric: '0 dropped packets',
          timeIndex: 1,
        },
        {
          name: 'traffic/maglev-rebalance',
          status: 'BALANCED',
          statusColor: 'cyan',
          commitHash: 'b441f92',
          message: 'Maglev lookup table rotated in 1.2ms without cache churn',
          author: 'TRAFFIC PROXY',
          metric: '99.999% SLA',
          timeIndex: 2,
        },
        {
          name: 'ledger/truetime-audit',
          status: 'CONSISTENT',
          statusColor: 'emerald',
          commitHash: 'e991c01',
          message: 'TrueTime bounded uncertainty epsilon < 2.5ms',
          author: 'SPANNER AUDITOR',
          metric: '100% linearizability',
          timeIndex: 3,
        },
      ],
      summaryStat: {
        value: '99.999%',
        label: 'Availability Guaranteed',
      },
    },
    radial: {
      badge: 'GLOBAL MESH TOPOLOGY',
      title: 'Automations Unleashed: Planetary Fleet Connectivity',
      description:
        'Unify multi-tenant carrier networks across sovereign jurisdictions. Pegasus coordinates customs pipelines, interstate linehauls, and urban last-mile vehicles into one coordinated topology.',
      primaryBtn: {
        label: 'Explore Global Cells',
        href: '/platform',
      },
      secondaryBtn: {
        label: 'Global Network Map',
        href: '/contact',
      },
      techBadges: ['GOOGLE CLOUD SPANNER', 'MAGLEV HASHING', 'KAFKA MIRRORMAKER', 'ENVOY PROXY', 'POSTGIS CLUSTER', 'TAURI V2 SOVEREIGN'],
      centerNode: {
        title: 'GLOBAL MESH CORE',
        subtitle: 'Planetary Orchestration Bus',
        latency: '8.2ms Global RTT',
      },
      satellites: [
        { id: 'sat-1', name: 'Regional Cell UZ', subtitle: 'Tashkent Tier-III Node', status: 'ONLINE', iconName: 'layers', accent: 'emerald' },
        { id: 'sat-2', name: 'Regional Cell EU', subtitle: 'Frankfurt Direct Transit', status: 'ONLINE', iconName: 'layers', accent: 'cyan' },
        { id: 'sat-3', name: 'Maglev Router', subtitle: 'Sub-microsecond Hashing', status: 'ACTIVE', iconName: 'network', accent: 'purple' },
        { id: 'sat-4', name: 'TrueTime Consensus', subtitle: 'Hardware GPS Clocks', status: 'LOCKED', iconName: 'cpu', accent: 'blue' },
        { id: 'sat-5', name: 'Cross-Border Tariff', subtitle: 'Multi-Currency Settlement', status: 'INSTANT', iconName: 'database', accent: 'amber' },
        { id: 'sat-6', name: 'Encrypted Telemetry', subtitle: 'TLS 1.3 WireGuard Mesh', status: 'SECURE', iconName: 'shield', accent: 'emerald' },
      ],
    },
  },

  // 3. Supply Chain Software (/supply-chain-software)
  'supply-chain-software': {
    pillars: {
      eyebrow: 'OPERATIONAL EXCELLENCE',
      title: 'ENTERPRISE ARCHITECTURE DESIGNED FOR ZERO DOWNTIME AND RIGOROUS AUDIT',
      description:
        'Built for modern enterprise distributors who demand microsecond transaction logs, seamless warehouse automation, and flawless partner integration.',
      specSummary: 'Immutable Event Ledger • Automated Inventory Allocations • Multi-Role Access Control',
      items: [
        {
          title: 'Immutable Ledger Audit',
          description: 'Every stock reservation, SKU scan, and driver handoff is written to an append-only cryptographic event trail.',
          iconName: 'database',
          accentColor: '#10B981',
        },
        {
          title: 'Dynamic Yard Staging',
          description: 'Auto-allocates loading bays and dock doors according to vehicle arrival telemetry and pallet pick density.',
          iconName: 'workflow',
          accentColor: '#3B82F6',
        },
        {
          title: 'Fine-Grained RBAC',
          description: 'Strict security boundaries separating supplier executives, warehouse dispatchers, and external third-party carriers.',
          iconName: 'shield',
          accentColor: '#8B5CF6',
        },
        {
          title: 'Real-Time ERP Connectors',
          description: 'High-throughput bi-directional synchronization with SAP S/4HANA, 1C Enterprise, and Oracle NetSuite.',
          iconName: 'zap',
          accentColor: '#F59E0B',
        },
      ],
      protocolBlock: {
        tag: 'ENTERPRISE INTEGRATION BUS',
        title: 'Synchronous Warehouse Execution with Legacy ERPs',
        description:
          'Connect modern cloud dispatch algorithms with existing on-premise ERP platforms via transactional change-data-capture (CDC) pipelines.',
        whitepaperText: 'Read the Enterprise Integration Guide',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'ERP-GATE', sublabel: 'SAP / 1C Connect', status: 'active' },
          { id: '02', label: 'WMS-DOCK', sublabel: 'Staging Engine', status: 'active' },
          { id: '03', label: 'CDC-STREAM', sublabel: 'Debezium Pipeline', status: 'synced' },
          { id: '04', label: 'INV-ALLOC', sublabel: 'Stock Locks', status: 'verified' },
          { id: '05', label: 'RBAC-SEC', sublabel: 'Zero Trust Auth', status: 'active' },
          { id: '06', label: 'AUDIT-LOG', sublabel: 'Append-Only Trail', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'PARALLEL TESTING',
      title: 'Isolated Simulation of Supply Chain Disruption',
      description:
        'Branch warehouse allocations and supplier purchase orders into sandboxed staging environments. Stress test sudden supplier stockouts before affecting customer orders.',
      mainBranchLabel: 'enterprise / live-inventory',
      timestamps: ['T-0 (Order Ingest)', 'T+1m (Allocation)', 'T+3m (Wave Staged)', 'T+5m (Bay Assigned)', 'T+8m (Dispatched)'],
      branches: [
        {
          name: 'sandbox/stockout-stress-test',
          status: 'SIMULATED SPIKE',
          statusColor: 'amber',
          commitHash: 'e39b710',
          message: 'Inject 30% supplier shortage on key beverage SKUs',
          author: 'INVENTORY OPTIMIZER',
          metric: 'Auto-split to backup DC',
          timeIndex: 1,
        },
        {
          name: 'realloc/cross-dock-batch',
          status: 'ALLOCATED',
          statusColor: 'cyan',
          commitHash: 'c70144f',
          message: 'Direct transfer from incoming rail freight to outbound vans',
          author: 'WMS CONTROLLER',
          metric: 'Zero storage dwell time',
          timeIndex: 2,
        },
        {
          name: 'settle/batch-reconcile',
          status: 'MATCHED',
          statusColor: 'emerald',
          commitHash: '98d221a',
          message: 'Automatic invoice generation against confirmed loading manifest',
          author: 'FINANCE WORKER',
          metric: '100% ledger balance',
          timeIndex: 3,
        },
      ],
      summaryStat: {
        value: '0ms',
        label: 'Reconciliation Latency',
      },
    },
    radial: {
      badge: 'UNIFIED WAREHOUSE CORE',
      title: 'Automations Unleashed: The Autonomous Distribution Hub',
      description:
        'Orchestrate docks, forklifts, picking zones, and transport fleets from an integrated digital control tower. Pegasus removes paper manifests and synchronizes high-velocity supply chains.',
      primaryBtn: {
        label: 'Schedule Software Demo',
        href: '/contact',
      },
      secondaryBtn: {
        label: 'Explore System Capabilities',
        href: '/platform',
      },
      techBadges: ['POSTGRESQL 16', 'TIMESCALEDB', 'APACHE KAFKA', 'DEBEZIUM CDC', 'REDIS STREAMS', 'DOCKER COMPOSE'],
      centerNode: {
        title: 'WMS COMMAND HUB',
        subtitle: 'Unified Distribution Node',
        latency: '1.8ms Internal Bus',
      },
      satellites: [
        { id: 'sat-1', name: 'Wave Picking', subtitle: 'Optimized Bin Traversal', status: 'ACTIVE', iconName: 'workflow', accent: 'cyan' },
        { id: 'sat-2', name: 'Bay Allocation', subtitle: 'Dynamic Dock Scheduling', status: 'OPTIMAL', iconName: 'truck', accent: 'emerald' },
        { id: 'sat-3', name: 'Barcode Tracking', subtitle: 'GS1-128 Compliance', status: 'VERIFIED', iconName: 'barcode', accent: 'blue' },
        { id: 'sat-4', name: 'Inventory Locks', subtitle: 'Zero Double-Allocation', status: 'LOCKED', iconName: 'shield', accent: 'purple' },
        { id: 'sat-5', name: 'Cash Reconciliation', subtitle: 'Daily Shift Drawer Balance', status: 'BALANCED', iconName: 'database', accent: 'amber' },
        { id: 'sat-6', name: 'ERP Outbox Sync', subtitle: 'Real-time SAP / 1C Stream', status: 'STREAMING', iconName: 'zap', accent: 'emerald' },
      ],
    },
  },

  // 4. Cloud Ecosystem (/cloud-ecosystem)
  'cloud-ecosystem': {
    pillars: {
      eyebrow: 'INFRASTRUCTURE BLUEPRINT',
      title: 'HYBRID CLUSTER ARCHITECTURE: SOVEREIGN EDGE TO HIGH-AVAILABILITY CLOUD',
      description:
        'Purpose-built infrastructure pairing low-latency local edge nodes with scalable cloud managed services for maximum resilience and cost efficiency.',
      specSummary: 'Google Cloud Spanner • Servercore Tashkent Tier III • Direct TAS-IX Peering',
      items: [
        {
          title: 'Direct Local IX Peering',
          description: 'Sovereign deployment peering directly with local national internet exchanges (TAS-IX) for under 15ms driver ping.',
          iconName: 'network',
          accentColor: '#3B82F6',
        },
        {
          title: 'Transactional Outbox Pipeline',
          description: 'Postgres and Spanner outbox daemons publish verified transaction events directly to Kafka and Redis Streams.',
          iconName: 'zap',
          accentColor: '#10B981',
        },
        {
          title: 'TimescaleDB Metrics Engine',
          description: 'High-density telemetry storage indexing millions of driver GPS breadcrumbs with automatic data chunk compression.',
          iconName: 'database',
          accentColor: '#F59E0B',
        },
        {
          title: 'Hardware-Enforced Secrets',
          description: 'Zero plaintext environment files. All cryptographic secrets managed via Cloud KMS and Vault HSM instances.',
          iconName: 'lock',
          accentColor: '#8B5CF6',
        },
      ],
      protocolBlock: {
        tag: 'ZERO-TRUST NETWORK TOPOLOGY',
        title: 'Cryptographic Microservice Isolation',
        description:
          'All communication between Go backend microservices, Redis brokers, and frontend gateways is encrypted via mutual TLS (mTLS) with SPIFFE identities.',
        whitepaperText: 'Read the Security Architecture Document',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'MTLS-MESH', sublabel: 'SPIFFE Identity', status: 'verified' },
          { id: '02', label: 'TAS-IX', sublabel: 'Direct Peering', status: 'active' },
          { id: '03', label: 'TIMESCALE', sublabel: 'GPS Compression', status: 'active' },
          { id: '04', label: 'REDIS-STR', sublabel: 'Presence Bus', status: 'active' },
          { id: '05', label: 'VAULT-HSM', sublabel: 'Hardware Keys', status: 'verified' },
          { id: '06', label: 'PG-OUTBOX', sublabel: 'Sub-5ms Poller', status: 'synced' },
        ],
      },
    },
    timeline: {
      eyebrow: 'DEVOPS AUTOMATION',
      title: 'Zero-Downtime Schema & Service Rollouts',
      description:
        'Execute database migrations, rolling service deployments, and failover checks in live production without dropping a single active WebSocket connection.',
      mainBranchLabel: 'cloud-infra / k8s-production',
      timestamps: ['Stage-0 (Canary)', 'Stage-1 (Traffic 10%)', 'Stage-2 (Traffic 50%)', 'Stage-3 (Verification)', 'Stage-4 (100% Live)'],
      branches: [
        {
          name: 'schema/spanner-ddl-v2',
          status: 'APPLIED',
          statusColor: 'emerald',
          commitHash: '72c019d',
          message: 'Zero-downtime interleaved column addition on Shipments',
          author: 'DDL MIGRATOR',
          metric: '0 table locks',
          timeIndex: 1,
        },
        {
          name: 'canary/cvrp-go-v1.4',
          status: 'EVALUATING',
          statusColor: 'cyan',
          commitHash: '88a31e2',
          message: 'Canary rollout of OR-Tools multi-trip calculation model',
          author: 'DEPLOY BOT',
          metric: '3.1ms avg latency',
          timeIndex: 2,
        },
        {
          name: 'mesh/mtls-cert-rotation',
          status: 'ROTATED',
          statusColor: 'blue',
          commitHash: 'b55018f',
          message: 'Automated 30-day mTLS certificate rotation across 24 pods',
          author: 'CERT MANAGER',
          metric: '100% verified',
          timeIndex: 3,
        },
      ],
      summaryStat: {
        value: '100%',
        label: 'Zero-Downtime Deployments',
      },
    },
    radial: {
      badge: 'HYBRID CLOUD CORE',
      title: 'Automations Unleashed: Distributed Cloud Architecture',
      description:
        'Connect local sovereign edge datacenters with global multi-region cloud services. Pegasus provides full system observability, sub-millisecond bus fanout, and automated failover.',
      primaryBtn: {
        label: 'Explore Cloud Stack',
        href: '/technology',
      },
      secondaryBtn: {
        label: 'Request Infrastructure Audit',
        href: '/contact',
      },
      techBadges: ['KUBERNETES', 'DOCKER', 'POSTGRES 16', 'TIMESCALEDB', 'REDIS 7', 'ENVOY PROXY'],
      centerNode: {
        title: 'CLOUD GATEWAY CORE',
        subtitle: 'Dynamic Mesh Router',
        latency: '0.9ms Internal RTT',
      },
      satellites: [
        { id: 'sat-1', name: 'Postgres & Spanner', subtitle: 'Dual-Engine Persistence', status: 'ONLINE', iconName: 'database', accent: 'cyan' },
        { id: 'sat-2', name: 'Redis Streams Hub', subtitle: 'Driver Presence & Realtime', status: 'STREAMING', iconName: 'zap', accent: 'emerald' },
        { id: 'sat-3', name: 'Kafka Event Bus', subtitle: 'Transactional Event Log', status: 'ACTIVE', iconName: 'network', accent: 'blue' },
        { id: 'sat-4', name: 'TimescaleDB Engine', subtitle: 'GPS & Telematics Matrix', status: 'HEALTHY', iconName: 'cpu', accent: 'purple' },
        { id: 'sat-5', name: 'Edge TAS-IX Peering', subtitle: 'Direct Local Fiber Line', status: 'DIRECT', iconName: 'satellite', accent: 'amber' },
        { id: 'sat-6', name: 'Vault KMS Security', subtitle: 'Zero Plaintext Secrets', status: 'ENCRYPTED', iconName: 'shield', accent: 'emerald' },
      ],
    },
  },

  // 5. Alternatives / Comparisons (/alternatives, /compare)
  'alternatives': {
    pillars: {
      eyebrow: 'ENTERPRISE BENCHMARK',
      title: 'WHY MODERN FLEETS MIGRATE FROM LEGACY TMS TO PEGASUS',
      description:
        'Compare Pegasus against monolithic legacy systems. Zero licensing lock-in, open API standards, native mobile client suites, and true real-time synchronization.',
      specSummary: 'Deterministic CVRP vs Heuristic • Atomic Ledgers vs Batch Sync • Open Microservices',
      items: [
        {
          title: 'Mathematical CVRP vs Heuristic',
          description: 'Legacy TMS uses simplistic distance rules. Pegasus solves NP-hard vehicle routing problems with exact capacity constraints.',
          iconName: 'cpu',
          accentColor: '#3B82F6',
        },
        {
          title: 'Sub-100ms vs 15-Minute Sync',
          description: 'Legacy tools batch updates every quarter-hour. Pegasus reflects PoD, signatures, and payments instantaneously.',
          iconName: 'zap',
          accentColor: '#10B981',
        },
        {
          title: 'Native Mobile vs Slow Webviews',
          description: 'Drivers and warehouse staff run high-performance native Kotlin/Swift and Tauri apps with full offline resilience.',
          iconName: 'truck',
          accentColor: '#F59E0B',
        },
        {
          title: 'Double-Entry Invariants',
          description: 'Eliminates lost cash and reconciliation disputes with mathematical minor-unit balance enforcement.',
          iconName: 'shield',
          accentColor: '#8B5CF6',
        },
      ],
      protocolBlock: {
        tag: 'FEATURE MATRIX COMPARISON',
        title: 'Architectural Parity & Competitive Advantage',
        description:
          'Evaluate how Pegasus outperforms traditional enterprise tools like SAP Transportation Management, Oracle OTM, and fragmented local dispatch software.',
        whitepaperText: 'Read the Complete Competitive Audit',
        whitepaperHref: '/compare',
        nodes: [
          { id: '01', label: 'PEGASUS', sublabel: 'Real-Time Core', status: 'active' },
          { id: '02', label: 'LEGACY-TMS', sublabel: 'Batch Polling', status: 'synced' },
          { id: '03', label: 'CVRP-OPT', sublabel: 'Google OR-Tools', status: 'verified' },
          { id: '04', label: 'NATIVE-APP', sublabel: 'Offline SQLite', status: 'active' },
          { id: '05', label: 'LATENCY', sublabel: '< 100ms vs 15min', status: 'active' },
          { id: '06', label: 'TCO-COST', sublabel: '68% Lower Cloud Spend', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'MIGRATION PLAYBOOK',
      title: 'Frictionless Dual-Run Migration Pipeline',
      description:
        'Run Pegasus in parallel with your legacy TMS. Replicate live orders, validate routing outputs, and switch production cutover without interrupting a single delivery run.',
      mainBranchLabel: 'migration-track / dual-run-stream',
      timestamps: ['Day 1 (CDC Ingest)', 'Day 7 (Shadow Dispatch)', 'Day 14 (Route Benchmark)', 'Day 21 (Driver Onboard)', 'Day 30 (Full Cutover)'],
      branches: [
        {
          name: 'shadow/cvrp-benchmark',
          status: 'VALIDATED',
          statusColor: 'emerald',
          commitHash: '20df14a',
          message: 'Shadow dispatch shows 19% fuel savings vs existing manual planner',
          author: 'BENCHMARK AGENT',
          metric: '19% fuel reduction',
          timeIndex: 2,
        },
        {
          name: 'driver/pilot-rollout',
          status: 'ACTIVE PILOT',
          statusColor: 'cyan',
          commitHash: '39a88bc',
          message: '25 drivers in Tashkent city center running native Android app',
          author: 'PILOT LEAD',
          metric: '4.9/5 driver rating',
          timeIndex: 3,
        },
        {
          name: 'cutover/erp-outbox-switch',
          status: 'READY',
          statusColor: 'blue',
          commitHash: '551e901',
          message: 'Switch primary invoice generation from legacy ERP to Pegasus ledger',
          author: 'FINANCE LEAD',
          metric: 'Zero ledger delta',
          timeIndex: 4,
        },
      ],
      summaryStat: {
        value: '30 Days',
        label: 'Average Production Cutover',
      },
    },
    radial: {
      badge: 'MODERNIZATION ENGINE',
      title: 'Automations Unleashed: Replace Fragile Legacy Software',
      description:
        'Upgrade your entire logistics technology stack without operational downtime. Pegasus replaces fragile spreadsheets and legacy batch systems with a unified event-driven platform.',
      primaryBtn: {
        label: 'Request Migration Blueprint',
        href: '/contact',
      },
      secondaryBtn: {
        label: 'Compare Feature Matrix',
        href: '/compare',
      },
      techBadges: ['OPEN API', 'OFFLINE NATIVE', 'CVRP ENGINE', 'AUDIT TRAILS', 'MICROSERVICES', 'ZERO-LOCKIN'],
      centerNode: {
        title: 'PEGASUS MIGRATION CORE',
        subtitle: 'Dual-Run Compatibility Bus',
        latency: 'Zero Data Loss Guarantee',
      },
      satellites: [
        { id: 'sat-1', name: 'Legacy Ingest', subtitle: 'Automated CSV & API Sync', status: 'ONLINE', iconName: 'database', accent: 'cyan' },
        { id: 'sat-2', name: 'Routing Simulator', subtitle: 'Side-by-Side CVRP Metric', status: 'VALIDATED', iconName: 'cpu', accent: 'emerald' },
        { id: 'sat-3', name: 'Mobile App Rollout', subtitle: 'Instant Driver Onboarding', status: 'PILOT', iconName: 'truck', accent: 'blue' },
        { id: 'sat-4', name: 'Reconciliation Check', subtitle: 'Double-Entry Invariant Test', status: 'MATCHED', iconName: 'shield', accent: 'purple' },
        { id: 'sat-5', name: 'ERP Outbox Connector', subtitle: 'Bi-Directional SAP / 1C', status: 'READY', iconName: 'zap', accent: 'amber' },
        { id: 'sat-6', name: 'Cutover Switch', subtitle: 'Zero Downtime Deployment', status: 'ARMED', iconName: 'satellite', accent: 'emerald' },
      ],
    },
  },

  // 6. Markets / Regional Coverage (/markets)
  'markets': {
    pillars: {
      eyebrow: 'REGIONAL INFRASTRUCTURE',
      title: 'LOCALIZED LOGISTICS RUNNING AT NATIONAL HIGHWAY SCALE',
      description:
        'Engineered for regional emerging markets with challenging connectivity, cash-on-delivery workflows, and complex inter-city transit corridors.',
      specSummary: 'Offline Resilience • Cash Reconciliation • Native Cyrillic & Latin Support',
      items: [
        {
          title: 'Cash-on-Delivery (COD) Security',
          description: 'Tamper-proof digital cash envelopes and real-time physical drawer balance checking on every driver check-in.',
          iconName: 'database',
          accentColor: '#10B981',
        },
        {
          title: 'Corridor Optimization',
          description: 'Specialized routing models for key trade corridors (Tashkent-Samarkand-Bukhara-Fergana) balancing toll roads and traffic.',
          iconName: 'truck',
          accentColor: '#3B82F6',
        },
        {
          title: 'Full Offline Operation',
          description: 'Drivers continue scanning orders and recording delivery signatures in rural highway dead zones without interruption.',
          iconName: 'network',
          accentColor: '#F59E0B',
        },
        {
          title: 'Multi-Lingual Interface',
          description: 'Full native localization in Uzbek, Russian, and English across mobile driver apps, warehouse tablets, and desktop portals.',
          iconName: 'layers',
          accentColor: '#8B5CF6',
        },
      ],
      protocolBlock: {
        tag: 'SOVEREIGN CLOUD PEERING',
        title: 'Low-Latency National Peering Infrastructure',
        description:
          'Direct network interconnection with local telecommunication backbones ensures instantaneous driver telemetry updates even over 3G cellular connections.',
        whitepaperText: 'Read the Regional Coverage Report',
        whitepaperHref: '/technology',
        nodes: [
          { id: '01', label: 'TAS-CORE', sublabel: 'Tashkent Primary Hub', status: 'active' },
          { id: '02', label: 'SAM-NODE', sublabel: 'Samarkand Regional DC', status: 'active' },
          { id: '03', label: 'FER-EDGE', sublabel: 'Fergana Valley Gateway', status: 'synced' },
          { id: '04', label: 'BUK-TRANS', sublabel: 'Bukhara Transit Depot', status: 'active' },
          { id: '05', label: 'COD-VAULT', sublabel: 'Cash Treasury Sync', status: 'verified' },
          { id: '06', label: 'TELCO-PEER', sublabel: 'Local Direct Fiber', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'CORRIDOR DISPATCH',
      title: 'Real-Time Cross-Region Freight Dispatch',
      description:
        'Track and optimize linehaul shipments traversing national highway networks with real-time temperature telemetry, waypoint geofencing, and security seal tracking.',
      mainBranchLabel: 'national-linehaul / tashkent-fergana',
      timestamps: ['04:00 AM (Depot Departure)', '06:30 AM (Mountain Pass)', '08:15 AM (Kokand Gateway)', '10:00 AM (Fergana Hub)', '11:30 AM (Last Mile Dispatched)'],
      branches: [
        {
          name: 'kamchik/weather-reroute',
          status: 'WEATHER ADAPT',
          statusColor: 'amber',
          commitHash: '8910ebc',
          message: 'Speed restriction detected on Kamchik Pass; adjust ETA by +22m',
          author: 'TELEMETRY WORKER',
          metric: 'Proactive SLA warning',
          timeIndex: 1,
        },
        {
          name: 'dock/kokand-cross-dock',
          status: 'TRANSFERRED',
          statusColor: 'cyan',
          commitHash: 'f40122d',
          message: 'Offload 6 pallets for regional southern distribution',
          author: 'GATE SCANNER',
          metric: '8m dwell time',
          timeIndex: 2,
        },
        {
          name: 'treasury/daily-reconcile',
          status: 'VERIFIED',
          statusColor: 'emerald',
          commitHash: 'aa7710c',
          message: 'Reconcile 142M UZS in cash collections with physical bank deposit',
          author: 'TREASURY BOT',
          metric: '0 sum discrepancy',
          timeIndex: 4,
        },
      ],
      summaryStat: {
        value: '100%',
        label: 'Corridor Visibility',
      },
    },
    radial: {
      badge: 'NATIONAL DISTRIBUTION NETWORK',
      title: 'Automations Unleashed: Regional Scale, Local Execution',
      description:
        'Deploy Pegasus across diverse regional networks. Coordinate urban van fleets, intercity heavy trucks, and decentralized transit hubs through one intuitive platform.',
      primaryBtn: {
        label: 'Explore Regional Solutions',
        href: '/contact',
      },
      secondaryBtn: {
        label: 'Download Coverage Map',
        href: '/resources',
      },
      techBadges: ['OFFLINE SQLITE', 'TAS-IX FIBER', 'UZBEK / RUSSIAN / ENGLISH', 'COD ENVELOPES', 'CVRP MULTI-STOP', 'GPS GEOFENCING'],
      centerNode: {
        title: 'REGIONAL DISPATCH CORE',
        subtitle: 'National Infrastructure Bus',
        latency: '12ms Average Regional Ping',
      },
      satellites: [
        { id: 'sat-1', name: 'Tashkent Metro', subtitle: 'Dense Van Delivery Grid', status: 'ONLINE', iconName: 'truck', accent: 'cyan' },
        { id: 'sat-2', name: 'Highway Corridors', subtitle: 'Linehaul Telemetry Tracking', status: 'STREAMING', iconName: 'satellite', accent: 'emerald' },
        { id: 'sat-3', name: 'Cash Treasury', subtitle: 'Tamper-Proof COD Ingestion', status: 'VERIFIED', iconName: 'database', accent: 'amber' },
        { id: 'sat-4', name: 'Offline Driver Apps', subtitle: 'Encrypted Local SQLite Store', status: 'ACTIVE', iconName: 'shield', accent: 'purple' },
        { id: 'sat-5', name: 'Regional Gate Check', subtitle: 'Barcode Manifest Verification', status: 'OPTIMAL', iconName: 'barcode', accent: 'blue' },
        { id: 'sat-6', name: 'Multi-Lingual Voice', subtitle: 'Driver Prompt Engine', status: 'ENABLED', iconName: 'zap', accent: 'emerald' },
      ],
    },
  },

  // 7. General Fallback / Category Hubs (platform, technology, roles, apps-deploy, resources, company)
  'platform': {
    pillars: {
      eyebrow: 'UNIFIED PLATFORM PILLARS',
      title: 'A COMPREHENSIVE SUITE FOR AUTONOMOUS LOGISTICS & DISPATCH',
      description:
        'Pegasus combines mathematical route optimization, real-time fleet telematics, automated warehouse dispatch, and instant financial settlement into a single platform.',
      specSummary: 'End-to-End Visibility • Multi-Tenant Architecture • High-Density Telemetry',
      items: [
        {
          title: 'Automated Dispatch Console',
          description: 'Real-time drag-and-drop fleet scheduling with continuous algorithmic route recalculation.',
          iconName: 'workflow',
          accentColor: '#3B82F6',
        },
        {
          title: 'Native Cross-Role Clients',
          description: 'Specialized apps for drivers, warehouse operators, suppliers, and retailers with shared data contracts.',
          iconName: 'truck',
          accentColor: '#10B981',
        },
        {
          title: 'Unified Order Lifecycle',
          description: 'Track orders from cart checkout to warehouse staging, vehicle transit, delivery glass signature, and cash settlement.',
          iconName: 'database',
          accentColor: '#F59E0B',
        },
        {
          title: 'High-Throughput Telemetry',
          description: 'Ingest and process millions of GPS coordinates, speed vectors, and geofence triggers per minute.',
          iconName: 'cpu',
          accentColor: '#8B5CF6',
        },
      ],
      protocolBlock: {
        tag: 'PLATFORM CONCURRENCY ENGINE',
        title: 'Lock-Free Real-Time State Synchronization',
        description:
          'Every update across driver mobile devices and warehouse dispatch boards flows through high-speed Redis Streams and WebSocket hubs with sub-20ms delivery.',
        whitepaperText: 'Explore Platform Architecture',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'HUB-CORE', sublabel: 'Platform Engine', status: 'active' },
          { id: '02', label: 'ROLE-CLIENTS', sublabel: 'Native Swift/Kotlin', status: 'active' },
          { id: '03', label: 'TELEMETRY', sublabel: 'High-Density GPS', status: 'routing' },
          { id: '04', label: 'ORDER-LIFECYCLE', sublabel: 'End-to-End State', status: 'synced' },
          { id: '05', label: 'EVENT-STREAM', sublabel: 'Redis & Kafka', status: 'active' },
          { id: '06', label: 'AUDIT-LOG', sublabel: 'Cryptographic Ledger', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'LIFECYCLE ENGINE',
      title: 'Continuous State-Machine Progression',
      description:
        'Monitor delivery progression through deterministic state machines with guaranteed idempotency and zero duplicate order execution.',
      mainBranchLabel: 'order-lifecycle / primary-pipeline',
      timestamps: ['09:00 (Placed)', '09:15 (Packaged)', '09:40 (Loaded & Sealed)', '10:20 (Geofenced)', '11:00 (Settled)'],
      branches: [
        {
          name: 'gate/security-seal-scan',
          status: 'VERIFIED',
          statusColor: 'emerald',
          commitHash: 'a88190c',
          message: 'Digital barcode seal matched with outbound driver manifest',
          author: 'GATE OPERATOR',
          metric: '100% manifest match',
          timeIndex: 2,
        },
        {
          name: 'cvrp/dynamic-reroute',
          status: 'REROUTED',
          statusColor: 'amber',
          commitHash: 'b71239e',
          message: 'Dynamic insertion of priority emergency order into active route',
          author: 'DISPATCH BOT',
          metric: '+6m detour only',
          timeIndex: 3,
        },
        {
          name: 'pod/glass-signature-sync',
          status: 'COMPLETED',
          statusColor: 'cyan',
          commitHash: 'cc4019a',
          message: 'Recipient digital signature captured with GPS geo-stamp',
          author: 'DRIVER APP',
          metric: '0.04s cloud sync',
          timeIndex: 4,
        },
      ],
      summaryStat: {
        value: '< 50ms',
        label: 'State Propagation Latency',
      },
    },
    radial: {
      badge: 'PLATFORM AUTOMATION SUITE',
      title: 'Automations Unleashed: The Full Logistics Cloud',
      description:
        'Unlock automated efficiency across your entire supply chain. Pegasus provides full ecosystem integration, native multi-platform clients, and real-time operational control.',
      primaryBtn: {
        label: 'Schedule a Platform Tour',
        href: '/contact',
      },
      secondaryBtn: {
        label: 'Read System Specs',
        href: '/technology',
      },
      techBadges: ['CVRP ROUTING', 'GATE BARCODES', 'GLASS SIGNATURE', 'GEOFENCING', 'REDIS 7', 'DOUBLE-ENTRY'],
      centerNode: {
        title: 'PLATFORM NERVE CENTER',
        subtitle: 'Event-Driven Core',
        latency: '4.2ms System RTT',
      },
      satellites: [
        { id: 'sat-1', name: 'Order Management', subtitle: 'Multi-Channel Ingest', status: 'ACTIVE', iconName: 'database', accent: 'cyan' },
        { id: 'sat-2', name: 'Route Optimization', subtitle: 'CVRP Algorithmic Solver', status: 'OPTIMAL', iconName: 'cpu', accent: 'emerald' },
        { id: 'sat-3', name: 'Fleet Telematics', subtitle: 'Live Driver GPS Tracking', status: 'STREAMING', iconName: 'truck', accent: 'blue' },
        { id: 'sat-4', name: 'Warehouse Execution', subtitle: 'Dock Staging & Dwell Time', status: 'ONLINE', iconName: 'workflow', accent: 'purple' },
        { id: 'sat-5', name: 'Proof of Delivery', subtitle: 'Instant Signature & Cash', status: 'VERIFIED', iconName: 'file-check', accent: 'amber' },
        { id: 'sat-6', name: 'Realtime WebSockets', subtitle: 'Live Desktop & Mobile Sync', status: 'CONNECTED', iconName: 'zap', accent: 'emerald' },
      ],
    },
  },

  // 8. Technology (/technology)
  'technology': {
    pillars: {
      eyebrow: 'ENGINEERING PRINCIPLES',
      title: 'COMPILED SPEED, DETERMINISTIC CONSISTENCY, ZERO DATA LOSS',
      description:
        'Built with Go microservices, Google Cloud Spanner, Redis cluster, and native mobile runtimes for unmatched performance and military-grade data integrity.',
      specSummary: 'Strict Minor-Unit Arithmetic • Single-Flight Mutex Locks • Atomic Outbox Event Bus',
      items: [
        {
          title: 'Strict Minor-Unit Arithmetic',
          description: 'All monetary values and balances stored as 64-bit integers (tiyin / cents) to prevent floating-point rounding errors.',
          iconName: 'database',
          accentColor: '#10B981',
        },
        {
          title: 'Atomic Outbox Event Bus',
          description: 'Business state mutations and downstream Kafka events are committed in the exact same database transaction.',
          iconName: 'zap',
          accentColor: '#3B82F6',
        },
        {
          title: 'Google OR-Tools Solver',
          description: 'Capacitated Vehicle Routing Problem (CVRP) solved using state-of-the-art constraint programming and metaheuristics.',
          iconName: 'cpu',
          accentColor: '#06B6D4',
        },
        {
          title: 'Cryptographic Security & mTLS',
          description: 'Zero trust internal networking with hardware-backed key rotation, AES-256-GCM encryption, and signed JWT claims.',
          iconName: 'shield',
          accentColor: '#F59E0B',
        },
      ],
      protocolBlock: {
        tag: 'HIGH-PERFORMANCE RUNTIME',
        title: 'Sub-Millisecond Go Internal Microservice Bus',
        description:
          'Backend services communicate via binary gRPC protocols with zero JSON serialization overhead, delivering over 120,000 requests per second per core.',
        whitepaperText: 'Read the Go Microservices Benchmark',
        whitepaperHref: '/technology/architecture',
        nodes: [
          { id: '01', label: 'GO-CORE', sublabel: 'Chi Router / gRPC', status: 'active' },
          { id: '02', label: 'OR-TOOLS', sublabel: 'C++ Native Bindings', status: 'active' },
          { id: '03', label: 'SPANNER-RW', sublabel: 'ReadWriteTx Closure', status: 'verified' },
          { id: '04', label: 'PGX-V5', sublabel: 'Direct Binary Protocol', status: 'synced' },
          { id: '05', label: 'KAFKA-BUS', sublabel: 'Exact-Once Outbox', status: 'active' },
          { id: '06', label: 'REDIS-RING', sublabel: 'Consistent Hash Ring', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'HIGH-FREQUENCY TRANSACTIONS',
      title: 'Atomic State Transitions in Under 10 Milliseconds',
      description:
        'Trace an incoming delivery mutation through API ingress, constraint validation, Spanner transactional commit, and real-time Kafka event broadcast.',
      mainBranchLabel: 'transaction-trace / rw-tx-00189',
      timestamps: ['0.0ms (Ingress)', '1.4ms (Validate)', '3.2ms (Spanner Commit)', '4.8ms (Outbox Emit)', '7.2ms (Client Inbox)'],
      branches: [
        {
          name: 'cvrp/matrix-eval',
          status: 'COMPUTED',
          statusColor: 'cyan',
          commitHash: '9901efa',
          message: 'Distance matrix lookup computed across 450 nodes in 0.8ms',
          author: 'OR-TOOLS WORKER',
          metric: '0.8ms computation',
          timeIndex: 1,
        },
        {
          name: 'ledger/double-entry-write',
          status: 'COMMITTED',
          statusColor: 'emerald',
          commitHash: 'bc4102e',
          message: 'Spanner ReadWriteTx balance update: Debit + Credit == 0',
          author: 'LEDGER REPO',
          metric: '100% ACID',
          timeIndex: 2,
        },
        {
          name: 'stream/fanout-broadcast',
          status: 'BROADCAST',
          statusColor: 'blue',
          commitHash: 'df8120a',
          message: 'WebSocket notification pushed to 14 connected dispatch consoles',
          author: 'WS HUB',
          metric: '1.6ms delivery',
          timeIndex: 3,
        },
      ],
      summaryStat: {
        value: '< 10ms',
        label: 'End-to-End Latency',
      },
    },
    radial: {
      badge: 'TECHNICAL EXCELLENCE',
      title: 'Automations Unleashed: Cutting-Edge Engineering Stack',
      description:
        'Inspect the modern architectural layers powering Pegasus. From distributed Google Cloud Spanner schemas to native SwiftUI and Kotlin mobile runtimes, every component is crafted for speed.',
      primaryBtn: {
        label: 'View Architecture Docs',
        href: '/technology/architecture',
      },
      secondaryBtn: {
        label: 'Explore GitHub Repos',
        href: '/contact',
      },
      techBadges: ['GO 1.22', 'C++ OR-TOOLS', 'SPANNER', 'POSTGRES 16', 'KAFKA', 'TAURI V2', 'KOTLIN', 'SWIFTUI'],
      centerNode: {
        title: 'GO DISTRIBUTED CORE',
        subtitle: 'High-Concurrency Runtime',
        latency: '0.4ms Internal Latency',
      },
      satellites: [
        { id: 'sat-1', name: 'Google Cloud Spanner', subtitle: 'Globally Distributed DDL', status: 'ACTIVE', iconName: 'database', accent: 'cyan' },
        { id: 'sat-2', name: 'Google OR-Tools', subtitle: 'C++ Routing Heuristics', status: 'SOLVED', iconName: 'cpu', accent: 'emerald' },
        { id: 'sat-3', name: 'Apache Kafka', subtitle: 'Transactional Event Log', status: 'STREAMING', iconName: 'network', accent: 'blue' },
        { id: 'sat-4', name: 'Redis 7 Cluster', subtitle: 'Live Session & Presence', status: 'ONLINE', iconName: 'zap', accent: 'purple' },
        { id: 'sat-5', name: 'PostGIS Spatial', subtitle: 'Sub-second Geofence Match', status: 'HEALTHY', iconName: 'satellite', accent: 'amber' },
        { id: 'sat-6', name: 'Tauri v2 Desktop', subtitle: 'Rust-Powered Ops Portal', status: 'SYNCED', iconName: 'shield', accent: 'emerald' },
      ],
    },
  },

  // 9. Roles (/roles)
  'roles': {
    pillars: {
      eyebrow: 'ROLE-BASED ORCHESTRATION',
      title: 'TAILORED EXPERIENCES FOR EVERY ACTOR IN THE SUPPLY ECOSYSTEM',
      description:
        'From C-suite supplier executives and warehouse gate staff to field drivers and retail shopkeepers, Pegasus delivers dedicated interfaces with shared data truth.',
      specSummary: 'Supplier Portal • Warehouse Desktop • Driver Mobile • Retailer Telegram Mini App',
      items: [
        {
          title: 'Suppliers: Executive Portal',
          description: 'Monitor catalog demand, regional warehouse inventory, linehaul dispatch schedules, and real-time cash collections.',
          iconName: 'layers',
          accentColor: '#3B82F6',
        },
        {
          title: 'Warehouses: Tactical Desktop',
          description: 'Tauri v2 desktop app built for high-throughput barcode scanning, dock staging, and automated gate seal verification.',
          iconName: 'workflow',
          accentColor: '#10B981',
        },
        {
          title: 'Drivers: Native Mobile Mission',
          description: 'Offline-first Kotlin & Swift apps with turn-by-turn routing, glass signatures, and digital cash receipts.',
          iconName: 'truck',
          accentColor: '#F59E0B',
        },
        {
          title: 'Retailers: Telegram Mini App & Portal',
          description: 'Order restocks in 2 clicks, track incoming delivery truck on live map, and verify invoice digital receipts.',
          iconName: 'shield',
          accentColor: '#8B5CF6',
        },
      ],
      protocolBlock: {
        tag: 'CROSS-ROLE DATA BUS',
        title: 'Single Source of Truth Across 5 User Roles',
        description:
          'When a driver records a proof-of-delivery signature, the supplier dashboard updates its cash balance and the warehouse inventory decrement occurs in the exact same millisecond.',
        whitepaperText: 'Explore Role-Based Workflows',
        whitepaperHref: '/roles',
        nodes: [
          { id: '01', label: 'SUPPLIER', sublabel: 'Web Portal & BI', status: 'active' },
          { id: '02', label: 'WAREHOUSE', sublabel: 'Tauri Desktop', status: 'active' },
          { id: '03', label: 'DRIVER', sublabel: 'Native Android/iOS', status: 'active' },
          { id: '04', label: 'RETAILER', sublabel: 'Telegram Mini App', status: 'synced' },
          { id: '05', label: 'DISPATCHER', sublabel: 'Control Tower', status: 'verified' },
          { id: '06', label: 'TREASURY', sublabel: 'Settlement Engine', status: 'verified' },
        ],
      },
    },
    timeline: {
      eyebrow: 'HANDOFF TIMELINE',
      title: 'Seamless Cross-Role State Handshake',
      description:
        'Follow an order as it flows through supplier confirmation, warehouse dock loading, driver transport, retailer glass signing, and treasury cash deposit.',
      mainBranchLabel: 'handoff-chain / order-90218',
      timestamps: ['07:30 (Retailer Order)', '08:00 (Warehouse Staged)', '08:45 (Gate Check-out)', '10:15 (Driver Delivery)', '11:00 (Cash Reconciled)'],
      branches: [
        {
          name: 'warehouse/seal-scan',
          status: 'GATE PASSED',
          statusColor: 'emerald',
          commitHash: '77a019e',
          message: 'Warehouse gate scanner confirms barcode seal before departure',
          author: 'GATE OPERATOR',
          metric: 'Zero manifest errors',
          timeIndex: 2,
        },
        {
          name: 'driver/glass-signature',
          status: 'DELIVERED',
          statusColor: 'cyan',
          commitHash: '88b220f',
          message: 'Retailer signs digital screen; 1,450,000 UZS collected in cash',
          author: 'DRIVER MISSION',
          metric: 'Instant invoice update',
          timeIndex: 3,
        },
        {
          name: 'treasury/daily-lock',
          status: 'SETTLED',
          statusColor: 'blue',
          commitHash: '99c331a',
          message: 'Driver returns to depot; cashier validates cash against digital record',
          author: 'DEPOT CASHIER',
          metric: '100% cash match',
          timeIndex: 4,
        },
      ],
      summaryStat: {
        value: '5 Roles',
        label: 'Unified on One Platform',
      },
    },
    radial: {
      badge: 'COLLABORATIVE ECOSYSTEM',
      title: 'Automations Unleashed: Connect Every Supply Chain Actor',
      description:
        'Break down organizational silos. Pegasus connects your entire operational team, drivers, external carriers, and retail partners into one collaborative operating system.',
      primaryBtn: {
        label: 'View Role Apps',
        href: '/roles',
      },
      secondaryBtn: {
        label: 'Request Access',
        href: '/contact',
      },
      techBadges: ['SUPPLIER PORTAL', 'WAREHOUSE TAURI', 'DRIVER ANDROID', 'DRIVER IOS', 'TELEGRAM MINI APP', 'DISPATCH CONSOLE'],
      centerNode: {
        title: 'ROLE FEDERATION HUB',
        subtitle: 'Multi-Actor Event Router',
        latency: 'Unified Data Model',
      },
      satellites: [
        { id: 'sat-1', name: 'Suppliers', subtitle: 'Financial & Inventory Analytics', status: 'ONLINE', iconName: 'layers', accent: 'cyan' },
        { id: 'sat-2', name: 'Warehouse Ops', subtitle: 'Tauri v2 High-Density Desktop', status: 'ACTIVE', iconName: 'workflow', accent: 'emerald' },
        { id: 'sat-3', name: 'Fleet Drivers', subtitle: 'Turn-by-Turn Offline Mobile', status: 'STREAMING', iconName: 'truck', accent: 'blue' },
        { id: 'sat-4', name: 'Retailers', subtitle: 'Telegram Bot & Mini App', status: 'CONNECTED', iconName: 'shield', accent: 'purple' },
        { id: 'sat-5', name: 'Dispatchers', subtitle: 'Realtime CVRP Routing Tower', status: 'OPTIMAL', iconName: 'cpu', accent: 'amber' },
        { id: 'sat-6', name: 'Treasury & Cashiers', subtitle: 'Double-Entry Reconciliation', status: 'BALANCED', iconName: 'database', accent: 'emerald' },
      ],
    },
  },
};

// Fallback helper to ensure every subpage gets a valid, complete section config
export function getSectionConfigForPage(pageId?: string): SubpageSectionConfig {
  if (pageId && SECTION_PAGE_CONFIGS[pageId]) {
    return SECTION_PAGE_CONFIGS[pageId];
  }
  // Try mapped matches
  if (pageId?.includes('automation')) return SECTION_PAGE_CONFIGS['logistics-automation'];
  if (pageId?.includes('global') || pageId?.includes('markets')) return SECTION_PAGE_CONFIGS['global-logistics'];
  if (pageId?.includes('tech') || pageId?.includes('architecture')) return SECTION_PAGE_CONFIGS['technology'];
  if (pageId?.includes('role')) return SECTION_PAGE_CONFIGS['roles'];
  if (pageId?.includes('cloud')) return SECTION_PAGE_CONFIGS['cloud-ecosystem'];
  if (pageId?.includes('compare') || pageId?.includes('alternative')) return SECTION_PAGE_CONFIGS['alternatives'];
  if (pageId?.includes('supply') || pageId?.includes('software')) return SECTION_PAGE_CONFIGS['supply-chain-software'];

  return SECTION_PAGE_CONFIGS['platform'];
}
