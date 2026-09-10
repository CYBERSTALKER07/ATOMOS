import {
  Database,
  Server,
  Cpu,
  Layers,
  Globe2,
  Truck,
  ShieldCheck,
  Terminal,
  Smartphone,
  Laptop,
  Radio,
  Zap,
  BarChart3,
  Boxes,
  Eye,
  Workflow,
  Compass,
  Lock,
  Scale,
  Building2,
  Package,
  Activity,
  UserCheck,
  Store,
  Container,
} from 'lucide-react';
import type { DossierHeroProps } from './DossierHero';

export type DossierPageConfig = Omit<DossierHeroProps, 'className'>;

export const DOSSIER_PAGE_CONFIGS: Record<string, DossierPageConfig> = {
  // 1. PLATFORM
  platform: {
    tabLabel: 'PEGASUS / PLATFORM',
    tabSublabel: 'DISTRIBUTED ARCHITECTURE',
    highlightText: 'distributed - multi-tenant',
    headlineRest: 'cloud operating core.|',
    missionTitle: 'Sovereign Scaling, Zero Interruption',
    missionBody:
      'Engineered for global multi-tenant operations. Orchestrating Google Cloud Spanner distributed transactions, dynamic cell read-routing via Maglev consistent hashing, and transactional outbox event streams with zero cross-tenant contamination.',
    midTierLabel: 'TRANSACTIONAL CONSENSUS MESH',
    baseTierLabel: 'GOOGLE CLOUD SPANNER DISTRIBUTED CORE',
    accentColor: '#38BDF8',
    pills: [
      { label: 'Architecture', active: true },
      { label: 'Multi-Tenant' },
      { label: 'Spanner DDL' },
      { label: 'Cell Router', href: '/cloud-ecosystem' },
      { label: 'Deploy ↗', href: '/apps-deploy' },
    ],
    topNodes: [
      { id: 'p1', label: 'TENANT PACK', icon: Boxes },
      { id: 'p2', label: 'CELL ROUTER', icon: Server },
      { id: 'p3', label: 'OUTBOX WORKER', icon: Zap },
      { id: 'p4', label: 'SPANNER RW', icon: Database },
    ],
  },

  // 2. TECHNOLOGY
  technology: {
    tabLabel: 'PEGASUS / TECH STACK',
    tabSublabel: 'ENGINEERING SPECIFICATIONS',
    highlightText: 'high-frequency - consensus',
    headlineRest: 'engine specifications.|',
    missionTitle: 'Deterministic State Machines',
    missionBody:
      'Built with Go 1.22+ concurrent microservices, sub-millisecond Redis 7 Streams, gRPC multiplexing, and mathematical double-entry ledger invariants ensuring sub-second throughput under maximum morning dispatch peak loads.',
    midTierLabel: 'HIGH-THROUGHPUT EVENT BUS',
    baseTierLabel: 'LINUX KERNEL & BARE-METAL INFRASTRUCTURE',
    accentColor: '#CEFF00',
    pills: [
      { label: 'Go Microservices', active: true },
      { label: 'Redis 7 Streams' },
      { label: 'gRPC Multiplex' },
      { label: 'OR-Tools Engine', href: '/logistics-automation' },
      { label: 'Benchmarks ↗', href: '/compare' },
    ],
    topNodes: [
      { id: 't1', label: 'GO RUNTIME', icon: Terminal },
      { id: 't2', label: 'gRPC PIPELINE', icon: Cpu },
      { id: 't3', label: 'REDIS STREAMS', icon: Zap },
      { id: 't4', label: 'OR-TOOLS ENGINE', icon: Workflow },
    ],
  },

  // 3. ROLES
  roles: {
    tabLabel: 'PEGASUS / ROLES',
    tabSublabel: 'ECOSYSTEM STAKEHOLDERS',
    highlightText: 'multi-sided - network',
    headlineRest: 'ecosystem orchestration.|',
    missionTitle: 'Synchronized Stakeholder Workflows',
    missionBody:
      'Connecting suppliers, multi-depot warehouse teams, line-haul carriers, last-mile drivers, and retail storefronts into a unified transactional graph with role-scoped permission claims and instantaneous state replication.',
    midTierLabel: 'UNIFIED COMMERCE GRAPH',
    baseTierLabel: 'MULTI-STAKEHOLDER CLEARING LEDGER',
    accentColor: '#A78BFA',
    pills: [
      { label: 'Supplier Hub', active: true },
      { label: 'Warehouse Board' },
      { label: 'Driver Suite', href: '/mobile-apps' },
      { label: 'Retailer Store', href: '/web-apps' },
      { label: 'Join Network ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'r1', label: 'SUPPLIER OPS', icon: Building2 },
      { id: 'r2', label: 'WAREHOUSE', icon: Package },
      { id: 'r3', label: 'DRIVER SUITE', icon: Truck },
      { id: 'r4', label: 'RETAILER BOT', icon: Store },
    ],
  },

  // 4. DESKTOP APPS
  'desktop-apps': {
    tabLabel: 'PEGASUS / DESKTOP',
    tabSublabel: 'COMMAND CENTERS',
    highlightText: 'mission-critical - control',
    headlineRest: 'desktop command centers.|',
    missionTitle: 'Large-Format Multi-Display Dispatch',
    missionBody:
      'Native desktop power built on Tauri v2 and high-density Next.js control panels. Delivers millisecond drag-and-drop route overrides, yard inventory radar, and financial treasury oversight across multi-monitor control rooms.',
    midTierLabel: 'NATIVE IPC & WEBSOCKET BRIDGE',
    baseTierLabel: 'HARDWARE ACCELERATED DISPLAY ENGINE',
    accentColor: '#38BDF8',
    pills: [
      { label: 'Tauri v2 Core', active: true },
      { label: 'Multi-Monitor' },
      { label: 'Dispatch Board' },
      { label: 'Local SQLite' },
      { label: 'Install Client ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'd1', label: 'DISPATCH HUD', icon: Laptop },
      { id: 'd2', label: 'YARD RADAR', icon: Radio },
      { id: 'd3', label: 'IPC BRIDGE', icon: Cpu },
      { id: 'd4', label: 'LOCAL DB', icon: Database },
    ],
  },

  // 5. MOBILE APPS
  'mobile-apps': {
    tabLabel: 'PEGASUS / MOBILE',
    tabSublabel: 'EDGE FIELD SUITE',
    highlightText: 'offline-first - telemetry',
    headlineRest: 'edge mobile suite.|',
    missionTitle: 'Continuous Sub-Second Field Sync',
    missionBody:
      'Native Android (Kotlin Compose) and iOS (SwiftUI) applications engineered for extreme field resilience. Supports offline-first delta replication, pre-trip DVIR vehicle inspections, and cryptographic proof of delivery.',
    midTierLabel: 'OFFLINE DELTA REPLICATION ENGINE',
    baseTierLabel: 'BACKGROUND GEOFENCING & TELEMETRY CORE',
    accentColor: '#4ADE80',
    pills: [
      { label: 'Driver App', active: true },
      { label: 'DVIR Inspection' },
      { label: 'POD Scanner' },
      { label: 'Telegram Mini-App' },
      { label: 'Download APK ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'm1', label: 'DRIVER MOBILE', icon: Smartphone },
      { id: 'm2', label: 'DVIR PRE-TRIP', icon: ShieldCheck },
      { id: 'm3', label: 'CRYPTO POD', icon: Lock },
      { id: 'm4', label: 'DELTA SYNC', icon: Zap },
    ],
  },

  // 6. WEB APPS
  'web-apps': {
    tabLabel: 'PEGASUS / WEB PORTALS',
    tabSublabel: 'GLOBAL ENTERPRISE PORTALS',
    highlightText: 'instant-edge - access',
    headlineRest: 'global web portals.|',
    missionTitle: 'Multi-Tenant Enterprise Portals',
    missionBody:
      'Server-rendered Next.js 15 portals for supplier executive teams, distributor catalog managers, and enterprise billing departments with instant cold-boot speeds, deep permission scoping, and real-time live updates.',
    midTierLabel: 'EDGE SSR & SERVER ACTIONS PIPELINE',
    baseTierLabel: 'GLOBAL CDN & SECURITY FIREWALL EDGE',
    accentColor: '#F472B6',
    pills: [
      { label: 'Supplier Hub', active: true },
      { label: 'Catalog Manager' },
      { label: 'Billing Engine' },
      { label: 'SLA Dashboard' },
      { label: 'Launch Web ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'w1', label: 'SUPPLIER HUB', icon: Building2 },
      { id: 'w2', label: 'CATALOG SKU', icon: Boxes },
      { id: 'w3', label: 'CLEARING LEDGER', icon: Scale },
      { id: 'w4', label: 'ANALYTICS BI', icon: BarChart3 },
    ],
  },

  // 7. CLOUD ECOSYSTEM
  'cloud-ecosystem': {
    tabLabel: 'PEGASUS / CLOUD',
    tabSublabel: 'MULTI-REGION CELL ARCHITECTURE',
    highlightText: 'multi-region - sovereign',
    headlineRest: 'cell cloud ecosystem.|',
    missionTitle: 'Autonomous Geographic Isolation',
    missionBody:
      'Cloned sovereign cells (cell-uz, cell-eu, cell-us) running Google Cloud Spanner global distribution, Maglev consistent hash routing, and localized data sovereignty with zero single point of failure.',
    midTierLabel: 'DISTRIBUTED CONSENSUS PIPELINE',
    baseTierLabel: 'MULTI-MASTER SPANNER CLUSTER',
    accentColor: '#60A5FA',
    pills: [
      { label: 'Cell-UZ', active: true },
      { label: 'Cell-EU' },
      { label: 'Cell-US' },
      { label: 'Maglev Hashing' },
      { label: 'Cloud Topology ↗', href: '/platform' },
    ],
    topNodes: [
      { id: 'c1', label: 'CELL-UZ', icon: Server },
      { id: 'c2', label: 'CELL-EU', icon: Server },
      { id: 'c3', label: 'MAGLEV ROUTER', icon: Cpu },
      { id: 'c4', label: 'GLOBAL SPANNER', icon: Database },
    ],
  },

  // 8. GLOBAL LOGISTICS
  'global-logistics': {
    tabLabel: 'PEGASUS / GLOBAL FREIGHT',
    tabSublabel: 'CROSS-BORDER CORRIDORS',
    highlightText: 'cross-border - corridor',
    headlineRest: 'freight operating network.|',
    missionTitle: 'Intermodal Visibility, Automated',
    missionBody:
      'Unifying dry ports, maritime container shipping, bonded customs clearance, and long-haul intercity transit into a continuous real-time ledger that eliminates manual customs delays and cargo dwell time.',
    midTierLabel: 'DYNAMIC BORDER & TARIFF CLEARANCE ENGINE',
    baseTierLabel: 'INTERNATIONAL TRANSIT CORRIDOR MESH',
    accentColor: '#38BDF8',
    pills: [
      { label: 'Intermodal Fleet', active: true },
      { label: 'Bonded Customs' },
      { label: 'Maritime Rail' },
      { label: 'Waybills' },
      { label: 'Track Shipment ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'gl1', label: 'INTERMODAL', icon: Container },
      { id: 'gl2', label: 'CUSTOMS AUTO', icon: ShieldCheck },
      { id: 'gl3', label: 'RAIL CORRIDOR', icon: Compass },
      { id: 'gl4', label: 'TRANSIT LEDGER', icon: Database },
    ],
  },

  // 9. LOGISTICS AUTOMATION
  'logistics-automation': {
    tabLabel: 'PEGASUS / AUTOMATION',
    tabSublabel: 'ALGORITHMIC ROBOTICS',
    highlightText: 'algorithmic - robotics',
    headlineRest: 'automated dispatch robotics.|',
    missionTitle: 'Google OR-Tools CVRP Solvers',
    missionBody:
      'Autonomous fleet capacity allocation, time-window constrained routing, 3D cubic load packing, and robotic sorting integration executing mathematically optimal dispatch schedules without human overhead.',
    midTierLabel: 'ALGORITHMIC OPTIMIZATION MATRIX',
    baseTierLabel: 'CONSTRAINT SATISFACTION ENGINE',
    accentColor: '#F59E0B',
    pills: [
      { label: 'CVRP Solvers', active: true },
      { label: '3D Load Packing' },
      { label: 'Auto-Dispatch' },
      { label: 'Robotics API' },
      { label: 'Simulate Route ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'la1', label: 'OR-TOOLS SOLVER', icon: Cpu },
      { id: 'la2', label: '3D CUBIC PACK', icon: Boxes },
      { id: 'la3', label: 'DYNAMIC HOT-SWAP', icon: Zap },
      { id: 'la4', label: 'ROBOTICS INTERFACE', icon: Workflow },
    ],
  },

  // 10. SUPPLY CHAIN SOFTWARE
  'supply-chain-software': {
    tabLabel: 'PEGASUS / SUPPLY CHAIN',
    tabSublabel: 'ENTERPRISE ERP & TMS',
    highlightText: 'end-to-end - visibility',
    headlineRest: 'enterprise supply chain core.|',
    missionTitle: 'Real-Time Ledger Invariants',
    missionBody:
      'Prevent stockouts and invoice mismatch with immutable double-entry inventory reservations, supplier credit clearing pools, and automated cross-dock replenishment synchronized across every depot.',
    midTierLabel: 'DOUBLE-ENTRY TRANSACTION LEDGER',
    baseTierLabel: 'REAL-TIME SKU INVENTORY ENGINE',
    accentColor: '#10B981',
    pills: [
      { label: 'Inventory Ledger', active: true },
      { label: 'Replenishment' },
      { label: 'Cross-Dock' },
      { label: 'Credit Clearing' },
      { label: 'Request Demo ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'sc1', label: 'REPLENISHMENT', icon: Package },
      { id: 'sc2', label: 'MULTI-DEPOT', icon: Building2 },
      { id: 'sc3', label: 'SUPPLIER POOL', icon: Scale },
      { id: 'sc4', label: 'SKU LEDGER', icon: Database },
    ],
  },

  // 11. OPERATIONS
  operations: {
    tabLabel: 'PEGASUS / OPERATIONS',
    tabSublabel: 'CONTROL TOWER',
    highlightText: 'real-time - control',
    headlineRest: 'command control tower.|',
    missionTitle: 'Live Exception Interception',
    missionBody:
      'Operational incident interception with sub-second vehicle hot-swapping, automated SLA breach rerouting, driver SOS dispatch, and dynamic load reassignment with zero dispatcher friction.',
    midTierLabel: 'AUTOMATED EXCEPTION PLAYBOOK BUS',
    baseTierLabel: 'REAL-TIME OPERATIONAL STATE MACHINE',
    accentColor: '#EF4444',
    pills: [
      { label: 'Hot-Swapping', active: true },
      { label: 'SLA Interceptor' },
      { label: 'Driver SOS' },
      { label: 'Yard Control' },
      { label: 'Control Room ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'op1', label: 'SLA MONITOR', icon: Activity },
      { id: 'op2', label: 'HOT-SWAPPER', icon: Zap },
      { id: 'op3', label: 'DRIVER SOS', icon: ShieldCheck },
      { id: 'op4', label: 'EXCEPTION BUS', icon: Radio },
    ],
  },

  // 12. CAPABILITIES
  capabilities: {
    tabLabel: 'PEGASUS / CAPABILITIES',
    tabSublabel: 'FULFILLMENT SPECTRUM',
    highlightText: 'specialized - execution',
    headlineRest: 'modular logistics capabilities.|',
    missionTitle: 'Enterprise Fulfillment Spectrum',
    missionBody:
      'Micro-fulfillment staging, cold-chain temperature telemetry, hazardous material containment, and white-glove delivery verified by cryptographic proof and automated SLA guarantees.',
    midTierLabel: 'CRYPTOGRAPHIC VERIFICATION MESH',
    baseTierLabel: 'SLA QUALITY CONTRACT ENGINE',
    accentColor: '#38BDF8',
    pills: [
      { label: 'Cold Chain', active: true },
      { label: 'Micro-Hubs' },
      { label: 'Hazmat Safe' },
      { label: 'Crypto Proof' },
      { label: 'Audit SLAs ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'cap1', label: 'COLD SENSOR', icon: Activity },
      { id: 'cap2', label: 'MICRO-HUB', icon: Building2 },
      { id: 'cap3', label: 'HAZMAT SHIELD', icon: ShieldCheck },
      { id: 'cap4', label: 'SLA CONTRACT', icon: Lock },
    ],
  },

  // 13. AI VISION
  'ai-vision': {
    tabLabel: 'PEGASUS / AI VISION',
    tabSublabel: 'EDGE NEURAL MODELS',
    highlightText: 'neural - inspection',
    headlineRest: 'computer vision inspection.|',
    missionTitle: 'Autonomous Cargo Scanning',
    missionBody:
      'Edge-deployed neural vision models for pre-trip vehicle damage classification, automated bill-of-lading optical character recognition, and real-time volumetric pallet dimensioning.',
    midTierLabel: 'EDGE NEURAL INFERENCE PIPELINE',
    baseTierLabel: 'GPU ACCELERATED INSPECTION CORE',
    accentColor: '#818CF8',
    pills: [
      { label: 'Damage Classifier', active: true },
      { label: 'Volumetric 3D' },
      { label: 'Waybill OCR' },
      { label: 'Edge Inference' },
      { label: 'Test Models ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'ai1', label: 'DAMAGE AI', icon: Eye },
      { id: 'ai2', label: 'VOLUMETRIC 3D', icon: Boxes },
      { id: 'ai3', label: 'WAYBILL OCR', icon: Terminal },
      { id: 'ai4', label: 'INFERENCE GPU', icon: Cpu },
    ],
  },

  // 14. ALTERNATIVES
  alternatives: {
    tabLabel: 'PEGASUS / ALTERNATIVES',
    tabSublabel: 'SOVEREIGN ADVANTAGE',
    highlightText: 'sovereign - architecture',
    headlineRest: 'modern logistics platform.|',
    missionTitle: 'Replacing Monolithic Legacy ERPs',
    missionBody:
      'Why tier-1 global distributors and national enterprises are migrating from brittle legacy ERPs and expensive SaaS stacks to Pegasus sovereign, high-throughput cloud architecture.',
    midTierLabel: 'SOVEREIGN MIGRATION BRIDGE',
    baseTierLabel: 'MODERN DISTRIBUTED INFRASTRUCTURE',
    accentColor: '#CEFF00',
    pills: [
      { label: 'vs Legacy ERP', active: true },
      { label: '10x Throughput' },
      { label: 'Zero Lock-in' },
      { label: 'TCO Benchmark' },
      { label: 'Migration Plan ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'alt1', label: '10x THROUGHPUT', icon: Zap },
      { id: 'alt2', label: '90% LESS TCO', icon: Scale },
      { id: 'alt3', label: 'ZERO LOCK-IN', icon: Lock },
      { id: 'alt4', label: 'OPEN APIS', icon: Terminal },
    ],
  },

  // 15. COMPARE
  compare: {
    tabLabel: 'PEGASUS / COMPARISON',
    tabSublabel: 'ARCHITECTURAL BENCHMARK',
    highlightText: 'matrix - evaluation',
    headlineRest: 'competitive architectural analysis.|',
    missionTitle: 'Side-by-Side Capability Matrix',
    missionBody:
      'Detailed quantitative benchmark comparing Pegasus against Samsara, SAP TM, Oracle OTM, and Manhattan Associates across throughput, latency, integer accounting, and deployment flexibility.',
    midTierLabel: 'ARCHITECTURAL BENCHMARK MATRIX',
    baseTierLabel: 'QUANTITATIVE EVALUATION FRAMEWORK',
    accentColor: '#38BDF8',
    pills: [
      { label: 'Throughput', active: true },
      { label: 'Latency Tests' },
      { label: 'Feature Matrix' },
      { label: 'Security Specs' },
      { label: 'Full Report ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'cmp1', label: 'LOW LATENCY', icon: Zap },
      { id: 'cmp2', label: 'INTEGER LEDGER', icon: Scale },
      { id: 'cmp3', label: 'NATIVE TAURI', icon: Laptop },
      { id: 'cmp4', label: 'BENCHMARKS', icon: BarChart3 },
    ],
  },

  // 16. APPS DEPLOY
  'apps-deploy': {
    tabLabel: 'PEGASUS / DEPLOYMENT',
    tabSublabel: 'INFRASTRUCTURE RUNTIME',
    highlightText: 'hybrid - cloud',
    headlineRest: 'sovereign deployment matrix.|',
    missionTitle: 'From Cloud Run to Bare-Metal Nodes',
    missionBody:
      'Deploy anywhere: Google Cloud multi-region serverless, Kubernetes enterprise operators, or single-tenant Tier III sovereign datacenters with direct national IX peering.',
    midTierLabel: 'GITOPS AUTOMATED CI/CD ENGINE',
    baseTierLabel: 'HARDENED LINUX INFRASTRUCTURE',
    accentColor: '#34D399',
    pills: [
      { label: 'Bare Metal', active: true },
      { label: 'Google Cloud' },
      { label: 'Kubernetes' },
      { label: 'Sovereign Node' },
      { label: 'Deploy Docs ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'dep1', label: 'CLOUD RUN', icon: Server },
      { id: 'dep2', label: 'K8s CLUSTER', icon: Layers },
      { id: 'dep3', label: 'TIER-III DC', icon: Building2 },
      { id: 'dep4', label: 'DOCKER EDGE', icon: Cpu },
    ],
  },

  // 17. MARKETS
  markets: {
    tabLabel: 'PEGASUS / MARKETS',
    tabSublabel: 'TERRITORIAL COMPLIANCE',
    highlightText: 'territorial - expansion',
    headlineRest: 'sovereign market localization.|',
    missionTitle: 'Central Asian & Emerging Market Gateways',
    missionBody:
      'Full regulatory, tax, and currency compliance for Uzbekistan (UZS tiyins), Kazakhstan, UAE, and European corridors with national payment rails and fiscal data storage.',
    midTierLabel: 'MULTI-CURRENCY FRACTIONAL SETTLEMENT',
    baseTierLabel: 'SOVEREIGN BANKING RAILS INTEGRATION',
    accentColor: '#F59E0B',
    pills: [
      { label: 'Uzbekistan Core', active: true },
      { label: 'Kazakhstan Transit' },
      { label: 'UAE Gateway' },
      { label: 'Fiscal Compliance' },
      { label: 'Enter Market ↗', href: '/join' },
    ],
    topNodes: [
      { id: 'mkt1', label: 'UZ MARKET', icon: Globe2 },
      { id: 'mkt2', label: 'KZ CORRIDOR', icon: Compass },
      { id: 'mkt3', label: 'UAE GATEWAY', icon: Building2 },
      { id: 'mkt4', label: 'FISCAL RAILS', icon: Scale },
    ],
  },
};

export function getDossierConfig(id: string): DossierPageConfig | undefined {
  // Normalize slug or route
  const cleanId = id.replace(/^\/+/, '').split('/')[0];
  return DOSSIER_PAGE_CONFIGS[cleanId];
}
