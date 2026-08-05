import type { FlowVariant } from './topicTypes';
import { topicHref } from './topicTypes';

export type MegaNavBadge = 'NEW';

export type MegaNavLink = {
  slug: string;
  label: string;
  description?: string;
  href: string;
  badge?: MegaNavBadge;
  flow?: FlowVariant;
  relatedProjectSlug?: string;
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
  primaryHref: '/platform',
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
    viewAllHref: '/platform',
    links: [
      { slug: 'atomos-control-plane', label: 'Control Plane', description: 'One platform. Six roles. Zero blind spots.', href: topicHref('platform', 'atomos-control-plane'), flow: 'controlPlane', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'how-pegasus-works', label: 'How Pegasus Works', description: 'Supplier-led networks from order to payment.', href: topicHref('platform', 'how-pegasus-works'), flow: 'orderLifecycle' },
      { slug: 'order-lifecycle', label: 'Order Lifecycle', description: 'Placed → Loaded → In transit → Arrived → Completed.', href: topicHref('platform', 'order-lifecycle'), flow: 'orderLifecycle', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'supplier-control-plane', label: 'Supplier Control Plane', description: 'Vetting, topology, treasury, and dispatch preview.', href: topicHref('platform', 'supplier-control-plane'), flow: 'controlPlane', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'mutating-handler-contract', label: 'Safe Updates', description: 'Check → Confirm → Save → Refresh → Notify.', href: topicHref('platform', 'mutating-handler-contract'), flow: 'mutatingHandler' },
      { slug: 'reliable-updates', label: 'Reliable Updates', description: 'Every app stays aligned after each change.', href: topicHref('platform', 'reliable-updates'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'network-topology', label: 'Network Topology', description: 'Warehouses, factories, zones, and gate seals.', href: topicHref('platform', 'network-topology'), flow: 'topologyMap', relatedProjectSlug: 'network-topology' },
      { slug: 'trust-reliability', label: 'Trust & Reliability', description: 'Status accuracy, payment safety, human override.', href: topicHref('platform', 'trust-reliability'), flow: 'orderLifecycle' },
    ],
    promo: {
      title: 'The Pegasus Control Plane',
      body: 'Digitize tribal knowledge, connect every function, and run one agile planning and execution model end-to-end.',
      primaryLabel: 'EXPLORE PLATFORM',
      primaryHref: '/platform',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'solutions',
    label: 'Solutions',
    viewAllHref: '/solutions',
    links: [
      { slug: 'dispatch-the-right-load', label: 'Dispatch the Right Load', description: 'Peak-window misloads eliminated with visual boards.', href: '/capabilities/smarter-dispatch', flow: 'dispatchBoard', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'visual-dispatch-engine', label: 'Visual Dispatch Engine', description: 'Match trucks to orders with warehouse override.', href: '/capabilities/smarter-dispatch', flow: 'dispatchBoard', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'fleet-visibility', label: 'Fleet Visibility', description: 'See your fleet as it moves — planned vs actual.', href: '/capabilities/live-fleet-tracking', flow: 'fleetMap', relatedProjectSlug: 'fleet-telemetry' },
      { slug: 'live-fleet-tracking', label: 'Live Fleet Tracking', description: 'Deviation alerts and retailer tracking views.', href: '/capabilities/live-fleet-tracking', flow: 'fleetMap', relatedProjectSlug: 'fleet-telemetry' },
      { slug: 'payment-confidence', label: 'Payment Confidence', description: 'Checkout, COD, treasury, and disputes.', href: '/capabilities/payment-confidence', flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'treasury-integrity', label: 'Treasury Integrity', description: 'Close the books without surprises.', href: '/roles/finance', flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'network-coordination', label: 'Network Coordination', description: 'Fragmented truth replaced by one live platform.', href: '/capabilities/instant-coordination', flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'warehouse-operations', label: 'Warehouse Operations', description: 'Pre-order hub, stock commitments, fleet CRUD.', href: '/roles/warehouse', flow: 'dispatchBoard', relatedProjectSlug: 'warehouse-operations' },
      { slug: 'factory-loading', label: 'Factory Loading', description: 'Supply requests, manifest lifecycle, loading bay.', href: '/roles/factory', flow: 'orderLifecycle', relatedProjectSlug: 'factory-loading' },
    ],
    promo: {
      title: 'Proven value for physical logistics',
      body: 'Dispatch accuracy, fleet visibility, payment confidence, and network scale — built for operators who cannot afford downtime.',
      primaryLabel: 'SEE SOLUTIONS',
      primaryHref: '/solutions',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'roles',
    label: 'Roles',
    viewAllHref: '/roles',
    links: [
      { slug: 'supplier', label: 'Supplier', description: 'Run your entire network from one place.', href: '/roles/supplier', flow: 'roleJourney', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'warehouse', label: 'Warehouse', description: 'Dispatch with confidence, every morning.', href: '/roles/warehouse', flow: 'roleJourney', relatedProjectSlug: 'warehouse-operations' },
      { slug: 'factory', label: 'Factory', description: 'Keep production and loading in sync.', href: '/roles/factory', flow: 'roleJourney', relatedProjectSlug: 'factory-loading' },
      { slug: 'driver', label: 'Driver', description: 'Clear routes. Simple stops. On-time delivery.', href: '/roles/driver', flow: 'roleJourney', relatedProjectSlug: 'driver-execution-app' },
      { slug: 'retailer', label: 'Retailer', description: 'Order, pay, and track — without phone calls.', href: '/roles/retailer', flow: 'roleJourney', relatedProjectSlug: 'retailer-commerce' },
      { slug: 'finance', label: 'Finance & Treasury', description: 'Close the books without surprises.', href: '/roles/finance', flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'payload-gate', label: 'Payload / Gate', description: 'Gate control that keeps every load accountable.', href: '/roles/payload-gate', flow: 'roleJourney', relatedProjectSlug: 'payload-gate-control' },
    ],
    promo: {
      title: 'Six roles. One source of truth.',
      body: 'Supplier, warehouse, factory, driver, retailer, and gate — every team works from the same live data.',
      primaryLabel: 'MEET THE ROLES',
      primaryHref: '/roles',
      secondaryLabel: 'VIEW APPS',
      secondaryHref: '/apps-deploy/mobile-apps',
    },
  },
  {
    id: 'capabilities',
    label: 'Capabilities',
    viewAllHref: '/capabilities',
    links: [
      { slug: 'smarter-dispatch', label: 'Smarter Dispatch', description: 'Match orders to trucks; warehouse always in control.', href: topicHref('capabilities', 'smarter-dispatch'), flow: 'dispatchBoard', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'reliable-updates', label: 'Reliable Updates', description: 'Consistent state across every app after each change.', href: topicHref('capabilities', 'reliable-updates'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'payment-confidence', label: 'Payment Confidence', description: 'Card, COD, treasury, and dispute handling.', href: topicHref('capabilities', 'payment-confidence'), flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'live-fleet-tracking', label: 'Live Fleet Tracking', description: 'Planned vs actual routes with live tracking.', href: topicHref('capabilities', 'live-fleet-tracking'), flow: 'fleetMap', relatedProjectSlug: 'fleet-telemetry' },
      { slug: 'instant-coordination', label: 'Instant Coordination', description: 'Live refresh across web and mobile.', href: topicHref('capabilities', 'instant-coordination'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'connected-network', label: 'Connected Network', description: 'Topology, zones, and service areas.', href: topicHref('capabilities', 'connected-network'), flow: 'topologyMap', relatedProjectSlug: 'network-topology' },
      { slug: 'returns-barcode-gate', label: 'Returns & Barcode Gate', description: 'Inbound returns with accountability.', href: topicHref('capabilities', 'returns-barcode-gate'), flow: 'exceptionPlaybook', relatedProjectSlug: 'payload-gate-control' },
      { slug: 'dispatch-preview', label: 'Dispatch Preview', description: 'Supplier override before trucks roll.', href: topicHref('capabilities', 'dispatch-preview'), flow: 'controlPlane', relatedProjectSlug: 'supplier-control-plane' },
    ],
  },
  {
    id: 'technology',
    label: 'Technology',
    viewAllHref: '/technology',
    links: [
      { slug: 'go-backend-platform', label: 'Unified Platform Core', description: 'One backend for every role and surface.', href: topicHref('technology', 'go-backend-platform'), flow: 'techStack' },
      { slug: 'cloud-spanner', label: 'Shared System of Record', description: 'Reliable order truth for the whole network.', href: topicHref('technology', 'cloud-spanner'), flow: 'techStack', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'redis-kafka', label: 'Live Sync & Events', description: 'Keep screens fresh after every change.', href: topicHref('technology', 'redis-kafka'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'websocket-hubs', label: 'Live Coordination', description: 'Instant updates by role across the network.', href: topicHref('technology', 'websocket-hubs'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'osrm-routing', label: 'Route Planning', description: 'Planned paths and turn-by-turn guidance.', href: topicHref('technology', 'osrm-routing'), flow: 'fleetMap', relatedProjectSlug: 'fleet-telemetry' },
      { slug: 'firebase-otp', label: 'Phone Sign-In', description: 'SMS login for driver, factory, and gate teams.', href: topicHref('technology', 'firebase-otp'), flow: 'techStack', relatedProjectSlug: 'driver-execution-app' },
      { slug: 'next-js-surfaces', label: 'Web Portals', description: 'Supplier portal, marketing, and ops boards.', href: topicHref('technology', 'next-js-surfaces'), flow: 'techStack', relatedProjectSlug: 'pegasus-marketing-site' },
      { slug: 'native-mobile-desktop', label: 'Native Mobile & Desktop', description: 'Android, iOS, and retailer desktop apps.', href: topicHref('technology', 'native-mobile-desktop'), flow: 'appsMatrix' },
    ],
    promo: {
      title: 'Built for real-world complexity',
      body: 'Reliable writes, live sync, and instant updates across every role — built for peak operations.',
      primaryLabel: 'VIEW TECH STACK',
      primaryHref: '/technology',
      secondaryLabel: 'ALL MODULES',
      secondaryHref: '/projects',
    },
  },
  {
    id: 'ai-vision',
    label: 'AI & Vision',
    viewAllHref: '/ai-vision',
    links: [
      { slug: 'smart-dispatch-assist', label: 'Smart Dispatch Assist', description: 'Optional auto-suggestions; warehouse override.', href: topicHref('ai-vision', 'smart-dispatch-assist'), flow: 'aiAssist', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'AI assist-vrp', label: 'Smart Route Assist', description: 'Background route suggestions for dispatch.', href: topicHref('ai-vision', 'AI assist-vrp'), flow: 'aiAssist', badge: 'NEW', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'ai-recommendations', label: 'AI Recommendations', description: 'Supplier ops suggestions from live data.', href: topicHref('ai-vision', 'ai-recommendations'), flow: 'aiAssist', badge: 'NEW', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'pulse-timeline', label: 'Pulse Timeline', description: 'Live event stream across the order lifecycle.', href: topicHref('ai-vision', 'pulse-timeline'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'explain-status-banners', label: 'Explain Status Banners', description: 'Plain-language status for every role.', href: topicHref('ai-vision', 'explain-status-banners'), flow: 'orderLifecycle', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'exception-weather-map', label: 'Exception Weather Map', description: 'Network-wide exception visibility.', href: topicHref('ai-vision', 'exception-weather-map'), flow: 'fleetMap', badge: 'NEW', relatedProjectSlug: 'fleet-telemetry' },
      { slug: 'override-impact-preview', label: 'Override Impact Preview', description: 'See downstream effects before you act.', href: topicHref('ai-vision', 'override-impact-preview'), flow: 'aiAssist', relatedProjectSlug: 'supplier-control-plane' },
      { slug: 'future-operating-model', label: 'Future Operating Model', description: 'Self-learning plan-vs-execution at scale.', href: topicHref('ai-vision', 'future-operating-model'), flow: 'controlPlane' },
    ],
    promo: {
      title: 'AI that respects the floor',
      body: 'Smart assist with human override — dispatch suggestions, recommendations, and exception detection without black boxes.',
      primaryLabel: 'DISCOVER AI',
      primaryHref: '/ai-vision',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'operations',
    label: 'Operations',
    viewAllHref: '/operations',
    links: [
      { slug: 'zone-miss-handling', label: 'Zone Miss Handling', description: 'Retailer outside delivery zone — clear errors.', href: topicHref('operations', 'zone-miss-handling'), flow: 'topologyMap', relatedProjectSlug: 'network-topology' },
      { slug: 'concurrent-stock-reject', label: 'Concurrent Stock Reject', description: 'Atomic reservation when stock runs out.', href: topicHref('operations', 'concurrent-stock-reject'), flow: 'exceptionPlaybook', relatedProjectSlug: 'retailer-commerce' },
      { slug: 'truck-too-small', label: 'Truck Too Small', description: 'Capacity overflow with partial dispatch recovery.', href: topicHref('operations', 'truck-too-small'), flow: 'dispatchBoard', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'partial-dispatch-commit', label: 'Partial Dispatch Commit', description: 'Warehouse recovers from split loads.', href: topicHref('operations', 'partial-dispatch-commit'), flow: 'dispatchBoard', relatedProjectSlug: 'warehouse-operations' },
      { slug: 'wrong-truck-sealed', label: 'Wrong Truck Sealed', description: 'Per-truck seal and driver gate accountability.', href: topicHref('operations', 'wrong-truck-sealed'), flow: 'exceptionPlaybook', relatedProjectSlug: 'payload-gate-control' },
      { slug: 'driver-reassignment', label: 'Driver Reassignment', description: 'Mid-load sick driver — capacity-safe replay.', href: topicHref('operations', 'driver-reassignment'), flow: 'exceptionPlaybook', relatedProjectSlug: 'payload-gate-control' },
      { slug: 'shop-closed-at-delivery', label: 'Shop Closed at Delivery', description: 'Driver, retailer, and supplier coordination.', href: topicHref('operations', 'shop-closed-at-delivery'), flow: 'exceptionPlaybook', relatedProjectSlug: 'driver-execution-app' },
      { slug: 'cash-at-door-cod', label: 'Cash at Door / COD', description: 'Payment exceptions and safe payment retries.', href: topicHref('operations', 'cash-at-door-cod'), flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'returns-wrong-barcode', label: 'Returns Wrong Barcode', description: 'Gate, warehouse, and supplier handoff.', href: topicHref('operations', 'returns-wrong-barcode'), flow: 'exceptionPlaybook', relatedProjectSlug: 'payload-gate-control' },
      { slug: 'live-tracking-expectations', label: 'Live Tracking Expectations', description: 'Loss-tolerant telemetry and support flows.', href: topicHref('operations', 'live-tracking-expectations'), flow: 'fleetMap', relatedProjectSlug: 'fleet-telemetry' },
    ],
    promo: {
      title: 'Built for the war stories',
      body: 'Every edge case from the field — zone misses, partial loads, shop closed, COD disputes — has a guarded path.',
      primaryLabel: 'EXPLORE OPERATIONS',
      primaryHref: '/operations',
      secondaryLabel: 'REQUEST DEMO',
      secondaryHref: '/join',
    },
  },
  {
    id: 'apps-deploy',
    label: 'Apps & Deploy',
    viewAllHref: '/apps-deploy',
    links: [
      { slug: 'mobile-apps', label: 'Mobile Apps', description: 'Driver, warehouse, factory, retailer, payload.', href: topicHref('apps-deploy', 'mobile-apps'), flow: 'appsMatrix' },
      { slug: 'desktop-apps', label: 'Desktop Apps', description: 'Retailer desktop app for store counters.', href: topicHref('apps-deploy', 'desktop-apps'), flow: 'appsMatrix' },
      { slug: 'web-apps', label: 'Web Apps', description: 'Supplier and warehouse portals.', href: topicHref('apps-deploy', 'web-apps'), flow: 'appsMatrix' },
      { slug: 'dispatch-fleet', label: 'Dispatch & Fleet', description: 'Visual load planning at peak hours.', href: topicHref('apps-deploy', 'dispatch-fleet'), flow: 'dispatchBoard', relatedProjectSlug: 'dispatch-engine' },
      { slug: 'payments-treasury', label: 'Payments & Treasury', description: 'Financial integrity across the network.', href: topicHref('apps-deploy', 'payments-treasury'), flow: 'paymentFlow', relatedProjectSlug: 'payment-integrity' },
      { slug: 'realtime-coordination', label: 'Realtime Coordination', description: 'Live updates across every surface.', href: topicHref('apps-deploy', 'realtime-coordination'), flow: 'realtimePipeline', relatedProjectSlug: 'realtime-coordination' },
      { slug: 'enterprise-rollout', label: 'Enterprise Rollout', description: 'Multi-site networks and deployment pillars.', href: topicHref('apps-deploy', 'enterprise-rollout'), flow: 'controlPlane' },
      { slug: 'request-demo', label: 'Request Demo', description: 'Live walkthrough with the Pegasus team.', href: topicHref('apps-deploy', 'request-demo'), flow: 'orderLifecycle' },
    ],
  },
];
