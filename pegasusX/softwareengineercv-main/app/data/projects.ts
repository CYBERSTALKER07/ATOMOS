export interface Project {
  id: string;
  slug: string;
  title: string;
  description: string;
  longDescription: string;
  category: string;
  tags: string[];
  technologies: string[];
  github: string;
  liveUrl?: string;
  image?: string;
  images?: string[];
  features: string[];
  challenges: string[];
  learnings: string[];
  status: 'completed' | 'in-progress' | 'archived';
  date: string;
  color: string;
}

export const projects: Project[] = [
  {
    id: '1',
    slug: 'dispatch-engine',
    title: 'Dispatch Engine',
    description: 'Visual warehouse dispatch with smart truck-and-order matching',
    longDescription:
      'The core dispatch module helps warehouse teams assign orders to trucks on a live board — with optional smart suggestions based on area, capacity, and eligibility.',
    category: 'Dispatch',
    tags: ['Dispatch', 'Warehouse', 'Fleet', 'Operations'],
    technologies: ['Go', 'Spanner', 'Next.js', 'WebSocket'],
    github: '#',
    features: [
      'Visual truck and order selection',
      'Capacity-aware load planning',
      'Gate seal before departure',
      'Live board updates for ops teams',
      'Overflow handling across trucks',
    ],
    challenges: [
      'Peak morning dispatch windows',
      'Matching capacity without overselling',
      'Coordinating warehouse and gate teams',
    ],
    learnings: [
      'Manual-first dispatch with smart assist',
      'Real-time ops board design',
      'Multi-role handoff accountability',
    ],
    status: 'completed',
    date: '2025',
    color: '#A9EBF9',
  },
  {
    id: '2',
    slug: 'supplier-control-plane',
    title: 'Supplier Control Plane',
    description: 'Network oversight for suppliers running multi-site logistics',
    longDescription:
      'Suppliers manage catalog, order vetting, dispatch preview, topology, and treasury from one dashboard — across warehouses, factories, and retailers.',
    category: 'Platform',
    tags: ['Supplier', 'Network', 'Treasury', 'Portal'],
    technologies: ['Next.js', 'TypeScript', 'Go API', 'Spanner'],
    github: '#',
    features: [
      'Order vetting before fulfillment',
      'Dispatch preview with override',
      'Topology and service area management',
      'Treasury and reconciliation views',
      'Multi-site network visibility',
    ],
    challenges: [
      'Single view across many downstream teams',
      'Approving orders without slowing dispatch',
      'Treasury truth across payment types',
    ],
    learnings: [
      'Supplier-led network orchestration',
      'Claims-scoped multi-tenant operations',
      'Executive ops dashboard patterns',
    ],
    status: 'completed',
    date: '2025',
    color: '#8DDC96',
  },
  {
    id: '3',
    slug: 'driver-execution-app',
    title: 'Driver Execution App',
    description: 'Route execution, stop progression, and cash collection in the field',
    longDescription:
      'Native driver apps guide routes stop by stop, confirm deliveries and payments, and report live progress back to ops and retailers.',
    category: 'Mobile',
    tags: ['Driver', 'Routes', 'Mobile', 'Delivery'],
    technologies: ['Kotlin', 'SwiftUI', 'Go API', 'Telemetry'],
    github: '#',
    features: [
      'Sealed manifest before departure',
      'Stop-by-stop delivery workflow',
      'Cash collection per stop',
      'Live route progress reporting',
      'Offline-tolerant field execution',
    ],
    challenges: [
      'Clear instructions for high-volume drivers',
      'Accurate payment capture in the field',
      'Reliable telemetry on poor networks',
    ],
    learnings: [
      'Field-first mobile UX',
      'Driver trust through sealed manifests',
      'Stop state machines that ops can audit',
    ],
    status: 'completed',
    date: '2025',
    color: '#DABDFF',
  },
  {
    id: '4',
    slug: 'retailer-commerce',
    title: 'Retailer Commerce',
    description: 'Catalog, checkout, scheduling, and live order tracking',
    longDescription:
      'Retailers browse supplier catalogs, place orders, choose delivery windows, and track shipments live — without calling support.',
    category: 'Web Application',
    tags: ['Retailer', 'Checkout', 'Tracking', 'Catalog'],
    technologies: ['Next.js', 'Tauri', 'React Native', 'Payments'],
    github: '#',
    features: [
      'Catalog with delivery zone checks',
      'Scheduled and on-demand ordering',
      'Live delivery tracking',
      'Receipt and proof of delivery',
      'Desktop and mobile parity',
    ],
    challenges: [
      'Self-serve tracking without support load',
      'Checkout flows for mixed payment types',
      'Accurate delivery window communication',
    ],
    learnings: [
      'Retailer-facing order transparency',
      'Zone-aware catalog experiences',
      'Tracking as a retention feature',
    ],
    status: 'completed',
    date: '2025',
    color: '#FFDA6F',
  },
  {
    id: '5',
    slug: 'fleet-telemetry',
    title: 'Fleet Telemetry',
    description: 'Live fleet map with planned-vs-actual route truth',
    longDescription:
      'Ops teams and retailers see where every truck is, whether it is on plan, and when deliveries will land — with deviation alerts before complaints.',
    category: 'Fleet',
    tags: ['Telemetry', 'Maps', 'Realtime', 'Ops'],
    technologies: ['MapLibre', 'Go', 'WebSocket', 'Redis'],
    github: '#',
    features: [
      'Live vs delayed vehicle status',
      'Planned route at dispatch time',
      'Deviation alerts for ops',
      'Retailer self-serve tracking',
      'Fleet-wide map for warehouses',
    ],
    challenges: [
      'Honest on-time vs delayed semantics',
      'High-volume GPS ingestion',
      'Alerting without noise',
    ],
    learnings: [
      'Telemetry as an ops trust layer',
      'Map UX for dispatch and support teams',
      'Route truth across roles',
    ],
    status: 'in-progress',
    date: '2025',
    color: '#BDE7FF',
  },
  {
    id: '6',
    slug: 'payment-integrity',
    title: 'Payment Integrity',
    description: 'Checkout, cash collection, and supplier reconciliation in one flow',
    longDescription:
      'From retailer checkout to driver cash collection to supplier treasury — payments stay aligned with duplicate protection and a clear audit trail.',
    category: 'Payments',
    tags: ['Payments', 'Treasury', 'Reconciliation', 'Ledger'],
    technologies: ['Go', 'Spanner', 'Webhooks', 'Idempotency'],
    github: '#',
    features: [
      'Card and cash-on-delivery paths',
      'Duplicate charge prevention',
      'Driver collection at delivery',
      'Supplier treasury dashboards',
      'Dispute records with delivery proof',
    ],
    challenges: [
      'Multi-payment-type reconciliation',
      'Field cash collection accuracy',
      'Replay-safe webhook processing',
    ],
    learnings: [
      'Financial integrity in physical logistics',
      'Idempotent payment progression',
      'Treasury views operators actually use',
    ],
    status: 'completed',
    date: '2025',
    color: '#FE5934',
  },
  {
    id: '7',
    slug: 'warehouse-operations',
    title: 'Warehouse Operations',
    description: 'Pre-orders, stock commits, and manual dispatch boards',
    longDescription:
      'Warehouse portals combine pre-order hubs, stock reservations, visual dispatch, and fleet maps for depot teams running daily load planning.',
    category: 'Warehouse',
    tags: ['Warehouse', 'Inventory', 'Dispatch', 'Portal'],
    technologies: ['Next.js', 'Go', 'WebSocket', 'Android'],
    github: '#',
    features: [
      'Morning dispatch board',
      'Pre-order calendar and commitments',
      'Stock reservation at order creation',
      'Live fleet map after departure',
      'Android parity for floor teams',
    ],
    challenges: [
      'Fast dispatch under time pressure',
      'Stock accuracy during peak demand',
      'Coordinating with gate and driver teams',
    ],
    learnings: [
      'Warehouse-first dispatch UX',
      'Stock commit policies that prevent oversell',
      'Ops boards that scale at peak',
    ],
    status: 'completed',
    date: '2025',
    color: '#A9EBF9',
  },
  {
    id: '8',
    slug: 'factory-loading',
    title: 'Factory Loading',
    description: 'Manifest lifecycle from loading lane to sealed departure',
    longDescription:
      'Factory teams track manifests through loading, sealing, and handoff — with supply requests flowing cleanly to warehouse fulfillment partners.',
    category: 'Factory',
    tags: ['Factory', 'Manifests', 'Loading', 'Supply'],
    technologies: ['Next.js', 'Go', 'Mobile', 'Realtime'],
    github: '#',
    features: [
      'Manifest load and seal workflow',
      'Supply request fulfillment status',
      'Loading bay coordination',
      'Cross-manifest exception handling',
      'Driver activation after seal',
    ],
    challenges: [
      'Loading lane synchronization',
      'Manifest exceptions without chaos',
      'Handoff accountability to drivers',
    ],
    learnings: [
      'Factory-floor operational clarity',
      'Manifest state machines ops can trust',
      'Cross-role supply request flows',
    ],
    status: 'in-progress',
    date: '2025',
    color: '#8DDC96',
  },
  {
    id: '9',
    slug: 'pegasus-marketing-site',
    title: 'Pegasus Marketing Site',
    description: 'Product storytelling for supplier-led logistics networks',
    longDescription:
      'The public-facing site explains Pegasus capabilities, roles, solutions, and customer outcomes — with the same bold visual language as the product brand.',
    category: 'Platform',
    tags: ['Marketing', 'Next.js', 'Brand', 'Product'],
    technologies: ['Next.js', 'TypeScript', 'GSAP', 'Tailwind CSS'],
    github: '#',
    features: [
      'Solutions and capability deep dives',
      'Role-based product narratives',
      'Customer proof and contact flows',
      'SEO and structured metadata',
      'Responsive product storytelling',
    ],
    challenges: [
      'Explaining complex ops simply',
      'Buyer vs operator messaging balance',
      'Performance with rich motion',
    ],
    learnings: [
      'B2B logistics product marketing',
      'Outcome-first copy for operators',
      'Multi-page product education',
    ],
    status: 'completed',
    date: '2025',
    color: '#DABDFF',
  },
  {
    id: '10',
    slug: 'payload-gate-control',
    title: 'Payload Gate Control',
    description: 'Gate seal workflow that keeps every load accountable',
    longDescription:
      'Gate teams scan, load, and seal manifests before trucks depart — blocking drivers until the load is verified and recorded.',
    category: 'Operations',
    tags: ['Gate', 'Seal', 'Terminal', 'Accountability'],
    technologies: ['Expo', 'Android', 'Go API', 'Realtime'],
    github: '#',
    features: [
      'Per-truck manifest scanning',
      'Batch seal and driver release',
      'Exception and reassignment flows',
      'Terminal and mobile gate apps',
      'Audit trail for every departure',
    ],
    challenges: [
      'Preventing incomplete loads leaving yard',
      'Fast gate throughput at peak hours',
      'Exception handling without paper trails',
    ],
    learnings: [
      'Gate as a hard operational checkpoint',
      'Terminal UX for loading bays',
      'Seal-gated driver activation',
    ],
    status: 'completed',
    date: '2025',
    color: '#FFDA6F',
  },
  {
    id: '11',
    slug: 'network-topology',
    title: 'Network Topology',
    description: 'One connected structure for suppliers, sites, and delivery zones',
    longDescription:
      'Suppliers define warehouses, factories, and service areas once — every downstream team inherits the same network map and order rules.',
    category: 'Platform',
    tags: ['Topology', 'Network', 'Zones', 'Multi-site'],
    technologies: ['Go', 'Spanner', 'H3', 'Portal'],
    github: '#',
    features: [
      'Warehouse and factory seeding',
      'Retailer delivery zone enforcement',
      'Topology changes propagate live',
      'Multi-supplier network isolation',
      'Service area visualization',
    ],
    challenges: [
      'Keeping network structure consistent',
      'Zone rules retailers understand',
      'Scaling to many sites per supplier',
    ],
    learnings: [
      'Topology as shared network truth',
      'Geo-aware ordering constraints',
      'Supplier network modeling',
    ],
    status: 'completed',
    date: '2025',
    color: '#BDE7FF',
  },
  {
    id: '12',
    slug: 'realtime-coordination',
    title: 'Realtime Coordination',
    description: 'Instant updates across dispatch boards, apps, and tracking',
    longDescription:
      'When status changes anywhere in the network, connected apps refresh within seconds — warehouses, drivers, retailers, and suppliers stay aligned.',
    category: 'Realtime',
    tags: ['WebSocket', 'Events', 'Ops', 'Sync'],
    technologies: ['Kafka', 'Redis', 'WebSocket', 'Go'],
    github: '#',
    features: [
      'Role-scoped live updates',
      'Silent refresh on connected clients',
      'Post-commit cache invalidation',
      'Cross-pod websocket relay',
      'Ops boards without manual refresh',
    ],
    challenges: [
      'High fanout during dispatch peaks',
      'Consistent state across six role apps',
      'Avoiding stale screens in the field',
    ],
    learnings: [
      'Realtime as an ops reliability feature',
      'Event contracts across role rows',
      'Live coordination at network scale',
    ],
    status: 'completed',
    date: '2025',
    color: '#FE5934',
  },
];

export function getProjectBySlug(slug: string): Project | undefined {
  return projects.find((p) => p.slug === slug);
}

export function getProjectsByCategory(category: string): Project[] {
  return projects.filter((p) => p.category === category);
}

export function getAllCategories(): string[] {
  return Array.from(new Set(projects.map((p) => p.category)));
}
