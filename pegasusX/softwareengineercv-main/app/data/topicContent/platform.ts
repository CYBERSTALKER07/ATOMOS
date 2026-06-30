import { seedContent, defaultHowItWorks } from './helpers';

export const platformTopics = {
  'atomos-control-plane': seedContent({
    title: 'ATOMOS Control Plane',
    summary: 'One platform. Six roles. Zero blind spots across supplier-led logistics networks.',
    problem:
      'Operations teams juggle spreadsheets, radios, and siloed apps while orders move through six handoffs. Tribal knowledge lives in people, not systems.',
    outcomes: [
      'Single live picture from order to payment',
      'Role-scoped apps sharing one transactional core',
      'Supplier oversight without micromanaging every warehouse',
      'Topology and treasury in the same control plane',
    ],
    howItWorks: defaultHowItWorks([
      ['Connect every role', 'Supplier, warehouse, factory, driver, retailer, and gate each get purpose-built surfaces wired to the same backend.'],
      ['Centralize truth', 'Spanner holds order, fleet, and payment state — mutations flow through guarded handlers.'],
      ['Coordinate in real time', 'Outbox events fan out to Kafka, cache, and WebSocket rooms so screens stay aligned.'],
    ]),
    flow: 'controlPlane',
    relatedProjectSlug: 'supplier-control-plane',
    crossRole: [
      { role: 'Supplier', touchpoint: 'Network topology and vetting before fulfillment' },
      { role: 'Warehouse', touchpoint: 'Dispatch boards fed by live order state' },
    ],
  }),
  'how-pegasus-works': seedContent({
    title: 'How Pegasus Works',
    summary: 'Supplier-led networks from retailer order through dispatch, transit, delivery, and settlement.',
    problem:
      'Physical logistics breaks when each team optimizes locally — retailers call warehouses, warehouses radio drivers, finance reconciles weeks later.',
    outcomes: [
      'End-to-end visibility without phone-tag',
      'Pay-at-delivery with treasury reconciliation',
      'Gate accountability before trucks depart',
      'Exception paths for real field conditions',
    ],
    howItWorks: defaultHowItWorks([
      ['Retailer orders', 'Catalog, stock checks, and zone validation at checkout.'],
      ['Supplier vets', 'Orders wait for approval before warehouse dispatch begins.'],
      ['Warehouse loads', 'Visual dispatch, seal at gate, driver manifest issued.'],
      ['Driver delivers', 'Geofenced arrival, cash or card collection, proof attached.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 2 },
  }),
  'order-lifecycle': seedContent({
    title: 'Order Lifecycle',
    summary: 'Placed → Vetted → Loaded → In transit → Arrived → Paid → Completed — one canonical state machine.',
    problem:
      'When order status means different things to different teams, support calls multiply and payments misfire.',
    outcomes: [
      'Every role reads the same status labels',
      'Terminal states trigger inventory and ledger side effects',
      'Cancel paths release stock and notify downstream roles',
      'Tracking updates without manual refresh',
    ],
    howItWorks: defaultHowItWorks([
      ['Placed & vetted', 'Retailer checkout creates the order; supplier approves or rejects before dispatch.'],
      ['Loaded & sealed', 'Warehouse assigns trucks; payload seals each load before departure.'],
      ['In transit & arrived', 'Driver telemetry feeds fleet maps; geofence gates arrival and payment.'],
      ['Paid & completed', 'COD or card at door; treasury reconciles against supplier earnings.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 4 },
    relatedProjectSlug: 'supplier-control-plane',
  }),
  'supplier-control-plane': seedContent({
    title: 'Supplier Control Plane',
    summary: 'Vetting, topology, treasury, and dispatch preview from one supplier portal.',
    problem:
      'Supplier CEOs cannot see network performance, pending vetting, or treasury exposure without exporting CSVs from five tools.',
    outcomes: [
      'Approve orders before they hit the warehouse floor',
      'Preview dispatch assignments with override authority',
      'Manage catalog, zones, and fleet settings centrally',
      'Treasury and dispute visibility in one dashboard',
    ],
    howItWorks: defaultHowItWorks([
      ['Vet incoming orders', 'Review eligibility, stock, and retailer history before fulfillment.'],
      ['Shape the network', 'Topology changes propagate to warehouse dispatch rules.'],
      ['Preview dispatch', 'See how warehouses load trucks; override when strategy demands.'],
    ]),
    flow: 'controlPlane',
    flowConfig: { roles: ['Supplier', 'Warehouse', 'Retailer', 'Driver', 'Factory', 'Payload'] },
    relatedProjectSlug: 'supplier-control-plane',
  }),
  'mutating-handler-contract': seedContent({
    title: 'Mutating Handler Contract',
    summary: 'Verify → Validate → Save → Refresh → Notify — every state change follows the same guarded pipeline.',
    problem:
      'Ad-hoc API handlers skip cache invalidation or emit events outside transactions, leaving clients showing stale data.',
    outcomes: [
      'Consistent mutation path across all role routes',
      'Outbox writes in the same transaction as row updates',
      'Cache keys invalidated post-commit',
      'WebSocket fanout after durable persistence',
    ],
    howItWorks: defaultHowItWorks([
      ['Verify & validate', 'Auth claims, idempotency keys, and business rules before any write.'],
      ['Save in one transaction', 'Spanner read-write with outbox row in the same commit.'],
      ['Refresh & notify', 'Kafka consumers bust Redis cache and push WS envelopes to role rooms.'],
    ]),
    flow: 'mutatingHandler',
    flowConfig: { highlightStep: 2 },
  }),
  'reliable-updates': seedContent({
    title: 'Reliable Updates',
    summary: 'Transactional outbox — no mismatched screens after mutations.',
    problem:
      'Dual-write failures leave apps showing orders as dispatched when the database still says loading.',
    outcomes: [
      'Events emitted only after commit succeeds',
      'Idempotent consumers safe under replay',
      'Silent WS refresh on connected clients',
      'Cross-role parity on order and fleet state',
    ],
    howItWorks: defaultHowItWorks([
      ['Write once', 'Row update and outbox insert share a Spanner transaction.'],
      ['Publish async', 'Kafka picks up outbox events with ordering guarantees per aggregate.'],
      ['Invalidate & push', 'Redis keys drop; WebSocket hubs fan out role-scoped envelopes.'],
    ]),
    flow: 'realtimePipeline',
    relatedProjectSlug: 'realtime-coordination',
  }),
  'network-topology': seedContent({
    title: 'Network Topology',
    summary: 'Warehouses, factories, zones, and gate seals — the graph your dispatch rules run on.',
    problem:
      'Delivery zones drawn on paper do not stop retailers outside service areas from checking out — until it is too late.',
    outcomes: [
      'H3 zone validation at checkout',
      'Warehouse-factory co-location for internal transfers',
      'Fleet and service area tied to topology nodes',
      'Clear zone-miss errors for retailers and suppliers',
    ],
    howItWorks: defaultHowItWorks([
      ['Model the network', 'Suppliers define warehouses, factories, zones, and fleet home bases.'],
      ['Enforce at checkout', 'Retailers outside zones see actionable errors, not silent failures.'],
      ['Drive dispatch rules', 'Eligibility, fees, and routing respect topology edges.'],
    ]),
    flow: 'topologyMap',
    relatedProjectSlug: 'network-topology',
  }),
  'trust-reliability': seedContent({
    title: 'Trust & Reliability',
    summary: 'Status accuracy, payment safety, and human override when automation is not enough.',
    problem:
      'Operators lose trust when AI suggestions cannot be overridden or when payment state disagrees with delivery proof.',
    outcomes: [
      'Human override on dispatch and supplier decisions',
      'Pay-at-delivery with geofenced collection',
      'Audit trails on force-dispatch and reassignment',
      'Explain banners in plain language per role',
    ],
    howItWorks: defaultHowItWorks([
      ['Accurate status', 'Single lifecycle state machine every client reads.'],
      ['Safe payments', 'Cash and card collection gated on arrival events.'],
      ['Override with accountability', 'Force actions logged; downstream impact previewed.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 5 },
  }),
};
