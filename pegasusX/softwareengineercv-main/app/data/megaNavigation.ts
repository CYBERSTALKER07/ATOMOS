export type MegaNavBadge = 'NEW';

export type MegaNavLink = {
  label: string;
  description?: string;
  href: string;
  badge?: MegaNavBadge;
};

export type MegaNavPromo = {
  title: string;
  body: string;
  primaryLabel: string;
  primaryHref: string;
  secondaryLabel?: string;
  secondaryHref?: string;
};

export type MegaNavCategory = {
  id: string;
  label: string;
  links: MegaNavLink[];
  viewAllHref: string;
  viewAllLabel?: string;
  promo?: MegaNavPromo;
};

export const DEFAULT_MEGA_PROMO: MegaNavPromo = {
  title: 'Step inside Pegasus',
  body: 'Explore dispatch, fleet tracking, payments, and coordination across six roles — one platform, every site connected.',
  primaryLabel: 'TAKE PLATFORM TOUR',
  primaryHref: '/#solutions',
  secondaryLabel: 'REQUEST DEMO',
  secondaryHref: '/join',
};

export const MEGA_NAV_FOOTER_LINKS = [
  { label: 'About', href: '/#about' },
  { label: 'Request Demo', href: '/join' },
  { label: 'Contact', href: '/contact' },
  { label: 'All Modules', href: '/projects' },
] as const;

export const MEGA_NAV_CATEGORIES: MegaNavCategory[] = [
  {
    id: 'platform',
    label: 'Platform',
    viewAllHref: '/#platform-value',
    links: [
      { label: 'ATOMOS Control Plane', description: 'One platform. Six roles. Zero blind spots.', href: '/#platform-value' },
      { label: 'How Pegasus Works', description: 'Supplier-led networks from order to payment.', href: '/#about' },
      { label: 'Order Lifecycle', description: 'Placed → Loaded → In transit → Arrived → Completed.', href: '/projects/supplier-control-plane' },
      { label: 'Supplier Control Plane', description: 'Vetting, topology, treasury, and dispatch preview.', href: '/projects/supplier-control-plane' },
      { label: 'Mutating Handler Contract', description: 'Verify → Validate → Save → Refresh → Notify.', href: '/#skills' },
      { label: 'Reliable Updates', description: 'Transactional outbox — no mismatched screens.', href: '/projects/realtime-coordination' },
      { label: 'Network Topology', description: 'Warehouses, factories, zones, and gate seals.', href: '/projects/network-topology' },
      { label: 'Trust & Reliability', description: 'Status accuracy, payment safety, human override.', href: '/#licensing' },
    ],
    promo: {
      title: 'The ATOMOS Control Plane',
      body: 'Digitize tribal knowledge, connect every function, and run one agile planning and execution model end-to-end.',
      primaryLabel: 'EXPLORE PLATFORM',
      primaryHref: '/#platform-value',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'solutions',
    label: 'Solutions',
    viewAllHref: '/#solutions',
    links: [
      { label: 'Dispatch the Right Load', description: 'Peak-window misloads eliminated with visual boards.', href: '/projects/dispatch-engine' },
      { label: 'Visual Dispatch Engine', description: 'Match trucks to orders with warehouse override.', href: '/projects/dispatch-engine' },
      { label: 'Fleet Visibility', description: 'See your fleet as it moves — planned vs actual.', href: '/projects/fleet-telemetry' },
      { label: 'Live Fleet Tracking', description: 'Deviation alerts and retailer tracking views.', href: '/projects/fleet-telemetry' },
      { label: 'Payment Confidence', description: 'Checkout, COD, treasury, and disputes.', href: '/projects/payment-integrity' },
      { label: 'Treasury Integrity', description: 'Close the books without surprises.', href: '/projects/payment-integrity' },
      { label: 'Network Coordination', description: 'Fragmented truth replaced by one live platform.', href: '/projects/realtime-coordination' },
      { label: 'Warehouse Operations', description: 'Pre-order hub, stock commitments, fleet CRUD.', href: '/projects/warehouse-operations' },
      { label: 'Factory Loading', description: 'Supply requests, manifest lifecycle, loading bay.', href: '/projects/factory-loading' },
    ],
    promo: {
      title: 'Proven value for physical logistics',
      body: 'Dispatch accuracy, fleet visibility, payment confidence, and network scale — built for operators who cannot afford downtime.',
      primaryLabel: 'SEE SOLUTIONS',
      primaryHref: '/#solutions',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'roles',
    label: 'Roles',
    viewAllHref: '/#companies',
    links: [
      { label: 'Supplier', description: 'Run your entire network from one place.', href: '/projects/supplier-control-plane' },
      { label: 'Warehouse', description: 'Dispatch with confidence, every morning.', href: '/projects/warehouse-operations' },
      { label: 'Factory', description: 'Keep production and loading in sync.', href: '/projects/factory-loading' },
      { label: 'Driver', description: 'Clear routes. Simple stops. On-time delivery.', href: '/projects/driver-execution-app' },
      { label: 'Retailer', description: 'Order, pay, and track — without phone calls.', href: '/projects/retailer-commerce' },
      { label: 'Payload / Gate', description: 'Gate control that keeps every load accountable.', href: '/projects/payload-gate-control' },
      { label: 'Order Vetting', description: 'Supplier approves before warehouse dispatch.', href: '/projects/supplier-control-plane' },
      { label: 'Cash Collection', description: 'Driver COD flows with treasury reconciliation.', href: '/projects/payment-integrity' },
      { label: 'Role Parity Matrix', description: 'Portal, mobile, and desktop for every team.', href: '/#companies' },
    ],
    promo: {
      title: 'Six roles. One source of truth.',
      body: 'Supplier, warehouse, factory, driver, retailer, and gate — every team works from the same live data.',
      primaryLabel: 'MEET THE ROLES',
      primaryHref: '/#companies',
      secondaryLabel: 'VIEW APPS',
      secondaryHref: '/mobile-apps',
    },
  },
  {
    id: 'capabilities',
    label: 'Capabilities',
    viewAllHref: '/projects',
    links: [
      { label: 'Smarter Dispatch', description: 'Match orders to trucks; warehouse always in control.', href: '/projects/dispatch-engine' },
      { label: 'Reliable Updates', description: 'Outbox pattern — consistent state across apps.', href: '/projects/realtime-coordination' },
      { label: 'Payment Confidence', description: 'Card, COD, treasury, and dispute handling.', href: '/projects/payment-integrity' },
      { label: 'Live Fleet Tracking', description: 'Telemetry with planned vs actual routes.', href: '/projects/fleet-telemetry' },
      { label: 'Instant Coordination', description: 'WebSocket refresh across web and mobile.', href: '/projects/realtime-coordination' },
      { label: 'Connected Network', description: 'Topology, zones, and service areas.', href: '/projects/network-topology' },
      { label: 'Returns & Barcode Gate', description: 'Inbound returns with accountability.', href: '/projects/payload-gate-control' },
      { label: 'Dispatch Preview', description: 'Supplier override before trucks roll.', href: '/projects/supplier-control-plane' },
    ],
  },
  {
    id: 'technology',
    label: 'Technology',
    viewAllHref: '/#skills',
    links: [
      { label: 'Go Backend Platform', description: 'Modular monolith with role-scoped routes.', href: '/#skills' },
      { label: 'Cloud Spanner', description: 'Transactional datastore with outbox pattern.', href: '/projects/realtime-coordination' },
      { label: 'Redis & Kafka', description: 'Cache invalidation and event bus fanout.', href: '/projects/realtime-coordination' },
      { label: 'WebSocket Hubs', description: 'Per-role live coordination rooms.', href: '/projects/realtime-coordination' },
      { label: 'OSRM Routing', description: 'Route geometry and turn-by-turn.', href: '/projects/fleet-telemetry' },
      { label: 'Firebase OTP', description: 'Phone auth for driver, factory, payload.', href: '/projects/driver-execution-app' },
      { label: 'Next.js Surfaces', description: 'Supplier portal, marketing, and ops boards.', href: '/projects/pegasus-marketing-site' },
      { label: 'Native Mobile & Desktop', description: 'Android, iOS, and Tauri retailer desktop.', href: '/mobile-apps' },
    ],
    promo: {
      title: 'Built for real-world complexity',
      body: 'Spanner transactions, Kafka events, Redis cache, and WebSocket fanout — production-grade from day one.',
      primaryLabel: 'VIEW TECH STACK',
      primaryHref: '/#skills',
      secondaryLabel: 'ALL MODULES',
      secondaryHref: '/projects',
    },
  },
  {
    id: 'ai-vision',
    label: 'AI & Vision',
    viewAllHref: '/projects/dispatch-engine',
    links: [
      { label: 'Smart Dispatch Assist', description: 'Optional auto-suggestions; warehouse override.', href: '/projects/dispatch-engine' },
      { label: 'AI Worker / VRP', description: 'Route optimization via Kafka consumer.', href: '/projects/dispatch-engine', badge: 'NEW' },
      { label: 'AI Recommendations', description: 'Supplier ops suggestions from live data.', href: '/projects/supplier-control-plane', badge: 'NEW' },
      { label: 'Pulse Timeline', description: 'Live event stream across the order lifecycle.', href: '/projects/realtime-coordination' },
      { label: 'Explain Status Banners', description: 'Plain-language status for every role.', href: '/projects/realtime-coordination' },
      { label: 'Exception Weather Map', description: 'Network-wide exception visibility.', href: '/projects/fleet-telemetry', badge: 'NEW' },
      { label: 'Override Impact Preview', description: 'See downstream effects before you act.', href: '/projects/supplier-control-plane' },
      { label: 'Future Operating Model', description: 'Self-learning plan-vs-execution at scale.', href: '/#platform-value' },
    ],
    promo: {
      title: 'AI that respects the floor',
      body: 'Smart assist with human override — dispatch suggestions, recommendations, and exception detection without black boxes.',
      primaryLabel: 'DISCOVER AI',
      primaryHref: '/projects/dispatch-engine',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'operations',
    label: 'Operations',
    viewAllHref: '/projects',
    links: [
      { label: 'Zone Miss Handling', description: 'Retailer outside delivery zone — clear errors.', href: '/projects/network-topology' },
      { label: 'Concurrent Stock Reject', description: 'Atomic reservation when stock runs out.', href: '/projects/retailer-commerce' },
      { label: 'Truck Too Small', description: 'Capacity overflow with partial dispatch recovery.', href: '/projects/dispatch-engine' },
      { label: 'Partial Dispatch Commit', description: 'Warehouse recovers from split loads.', href: '/projects/warehouse-operations' },
      { label: 'Wrong Truck Sealed', description: 'Per-truck seal and driver gate accountability.', href: '/projects/payload-gate-control' },
      { label: 'Driver Reassignment', description: 'Mid-load sick driver — capacity-safe replay.', href: '/projects/payload-gate-control' },
      { label: 'Shop Closed at Delivery', description: 'Driver, retailer, and supplier coordination.', href: '/projects/driver-execution-app' },
      { label: 'Cash at Door / COD', description: 'Payment exceptions and webhook replay safety.', href: '/projects/payment-integrity' },
      { label: 'Returns Wrong Barcode', description: 'Gate, warehouse, and supplier handoff.', href: '/projects/payload-gate-control' },
      { label: 'Live Tracking Expectations', description: 'Loss-tolerant telemetry and support flows.', href: '/projects/fleet-telemetry' },
    ],
    promo: {
      title: 'Built for the war stories',
      body: 'Every edge case from the field — zone misses, partial loads, shop closed, COD disputes — has a guarded path.',
      primaryLabel: 'EXPLORE MODULES',
      primaryHref: '/projects',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'apps-deploy',
    label: 'Apps & Deploy',
    viewAllHref: '/#licensing',
    links: [
      { label: 'Mobile Apps', description: 'Driver, warehouse, factory, retailer, payload.', href: '/mobile-apps' },
      { label: 'Desktop Apps', description: 'Retailer Tauri desktop for store counters.', href: '/desktop-apps' },
      { label: 'Web Apps', description: 'Supplier and warehouse portals.', href: '/web-apps' },
      { label: 'Dispatch & Fleet', description: 'Visual load planning at peak hours.', href: '/#licensing' },
      { label: 'Payments & Treasury', description: 'Financial integrity across the network.', href: '/projects/payment-integrity' },
      { label: 'Realtime Coordination', description: 'Live updates across every surface.', href: '/projects/realtime-coordination' },
      { label: 'Enterprise Rollout', description: 'Multi-site networks and deployment pillars.', href: '/#licensing' },
      { label: 'Request Demo', description: 'Live walkthrough with the Pegasus team.', href: '/join' },
    ],
    promo: {
      title: 'Deploy across your network',
      body: 'Portal, desktop, mobile, and terminal — six roles with parity on every surface that matters.',
      primaryLabel: 'VIEW APPS',
      primaryHref: '/mobile-apps',
      secondaryLabel: 'GET IN TOUCH',
      secondaryHref: '/join',
    },
  },
];
