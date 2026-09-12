import { seedContent, defaultHowItWorks, cards, DEFAULT_PROOF } from './helpers';

/** Platform topics — grounded in ECOSYSTEM_FEATURES_BY_ROLE + ORDER_FLOW docs */
export const platformTopics = {
  'atomos-control-plane': seedContent({
    title: 'Pegasus Control Plane',
    summary:
      'One platform. Six roles. Zero blind spots across supplier-led logistics networks — shared order truth, change events, role-scoped apps.',
    problem:
      'Operations teams juggle spreadsheets, radios, and siloed apps while orders move through six handoffs. Tribal knowledge lives in people, not systems — and every status mismatch becomes a support call.',
    outcomes: [
      'Single live picture from order to payment across supplier, warehouse, factory, driver, retailer, and payload',
      'Role-scoped apps sharing one transactional core with JWT-derived supplier scope',
      'Supplier oversight without micromanaging every warehouse floor decision',
      'Topology, treasury, and dispatch preview in the same control plane',
    ],
    howItWorks: defaultHowItWorks([
      [
        'Connect every role',
        'Supplier, warehouse, factory, driver, retailer, and payload each get purpose-built portal/mobile/desktop surfaces wired to the same platform.',
      ],
      [
        'Centralize truth',
        'shared system of record holds order, fleet, inventory, and payment state. Mutations go through guarded handlers with safe-retry keys.',
      ],
      [
        'Coordinate in real time',
        'Outbox events in the same confirmed save fan out to every role’s boards, maps, and caches so every screen stays aligned.',
      ],
    ]),
    flow: 'controlPlane',
    relatedProjectSlug: 'supplier-control-plane',
    crossRole: [
      { role: 'Supplier', touchpoint: 'Network topology, vetting, treasury, dispatch preview' },
      { role: 'Warehouse', touchpoint: 'Dispatch boards fed by live order + freeze-lock state' },
      { role: 'Retailer', touchpoint: 'Catalog, checkout, tracking, credit on shared order IDs' },
      { role: 'Driver', touchpoint: 'Manifest execution, geofenced arrival, cash collection' },
    ],
    capabilities: cards([
      [
        'Role-row parity',
        'A feature for a role lands on portal, Android, and iOS unless explicitly deferred — one product per role, not a pile of apps.',
      ],
      [
        'Mutation contract',
        'Check → Confirm → Save → Refresh screens → Notify teams. Every critical change follows the same pipeline.',
      ],
      [
        'Multi-tenant perimeter',
        'supplier_id and node scope come from JWT claims — never from request body — so warehouses cannot see another supplier’s fleet.',
      ],
      [
        'End-to-end coherence',
        'Order status, inventory reserves, fiscal receipts, and exceptions stay consistent across apps in near real time.',
      ],
    ]),
    differentiators: cards([
      [
        'Code-grounded lifecycle',
        'Statuses and transitions come from state_machine.go — PENDING through FISCALIZING to COMPLETED — not marketing diagrams.',
      ],
      [
        'Human override with audit',
        'Freeze-locks and force paths keep AI and automation from racing humans; every override is logged.',
      ],
      [
        'Integer money only',
        'All financial amounts are minor units (tiyin/cents). No floats — ledger and fiscal paths stay exact.',
      ],
    ]),
    whyItMatters: {
      headline: 'Global wholesale networks cannot run on tribal knowledge',
      body: 'Wholesale B2B distribution needs one system where suppliers run catalog and fleet, retailers order online, warehouses load correctly, drivers collect cash, and money plus stock stay consistent. Without a control plane, each team optimizes locally and the network pays in exceptions.',
      insights: [
        {
          title: 'From siloed apps to one spine',
          body: 'Client → secure API → confirmed save (domain + change log) → background jobs + live screen updates. One architecture for every role.',
        },
        {
          title: 'Prevention over firefighting',
          body: 'Zone validation, stock reserves, and vetting catch failures before trucks leave — not after retailers call support.',
        },
      ],
    },
    edgeCases: cards([
      [
        'Safe retries',
        'Mutating POSTs accept safe-retry key so retries never double-dispatch or double-charge.',
      ],
      [
        'Stale inventory release',
        'Reserves release on cancel, vet reject, and timeout — stock cannot lock forever after abandoned carts.',
      ],
    ]),
    aiAndData: cards([
      [
        'Shared system of record',
        'Strongly consistent order, inventory, and ledger rows — the only place domain truth may live.',
      ],
      [
        'Outbox → live events → consumers',
        'Fiscal apply, warehouse workers, notification dispatcher, and AI assist consume durable events — never write twice.',
      ],
      [
        'instant updates',
        'Role hubs at live channels push automatic refresh envelopes so boards and maps update without polling storms.',
      ],
    ]),
    proofItems: DEFAULT_PROOF,
  }),

  'how-pegasus-works': seedContent({
    title: 'How Pegasus Works',
    summary:
      'Supplier-led networks from retailer order through dispatch, transit, delivery, fiscalization, and settlement — one canonical flow.',
    problem:
      'Physical logistics breaks when each team optimizes locally — retailers call warehouses, warehouses radio drivers, finance reconciles weeks later, and nobody shares the same status words.',
    outcomes: [
      'End-to-end visibility without phone-tag',
      'Pay-at-delivery with treasury reconciliation and ADR-009 fiscal hard-gate',
      'Gate accountability before trucks depart',
      'Exception paths for shop-closed, zone-miss, and cash collection',
    ],
    howItWorks: defaultHowItWorks([
      ['Retailer orders', 'Catalog, stock checks, zone validation, and checkout quote at the retailer surface.'],
      ['Supplier vets', 'Orders wait for approval (or auto-accept rules) before warehouse dispatch begins.'],
      ['Warehouse loads', 'Visual dispatch, Smart Fit / AI optimizer with freeze-lock, seal at payload gate.'],
      ['Driver delivers', 'Geofenced arrival, cash/card/credit paths, fiscalizing before COMPLETED.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 2 },
    capabilities: cards([
      ['Canonical state machine', 'PENDING → LOADED → IN_TRANSIT → ARRIVED → payment paths → FISCALIZING → COMPLETED.'],
      ['Role handoffs', 'Each transition is triggered by the role that owns the physical action — not a generic admin.'],
      ['Proof at the door', 'Arrival and payment events carry geofence and amount constraints from code.'],
      ['Cancel with release', 'Cancel paths release inventory and notify downstream roles instead of orphaning stock.'],
    ]),
    differentiators: cards([
      ['Fiscal hard-gate (ADR-009)', 'Payment capture enters FISCALIZING; COMPLETED only after OFD success or explicit force.'],
      ['Credit leave path', 'DELIVERED_ON_CREDIT still reaches COMPLETED only via FISCALIZING after money movement.'],
      ['Shop-closed pending', 'ARRIVED can enter SHOP_CLOSED_PENDING without inventing a fake delivery.'],
    ]),
    whyItMatters: {
      headline: 'One flow, six roles, no ambiguous statuses',
      body: 'When “delivered” means different things to finance and drivers, payments misfire and support multiplies. Pegasus makes the state machine the product.',
      insights: [
        { title: 'Atomic retailer super-orders', body: 'Retailer orders stay atomic by default for B2B last-mile — split only via explicit overflow rules.' },
        { title: 'Realtime after commit', body: 'Screens refresh from change events, not optimistic UI guesses.' },
      ],
    },
    edgeCases: cards([
      ['CANCEL_REQUESTED exits', 'Must transition to CANCELLED or return to LOADED/IN_TRANSIT/ARRIVED — never brick the order.'],
      ['RECONCILIATION_REQUIRED', 'CANCELLED can enter reconciliation when money or stock still needs cleanup.'],
      ['Geofence violation', 'Arrival and payment actions reject outside proximity when policy requires it.'],
    ]),
    proofItems: [
      { label: 'Statuses', value: 'Code-defined' },
      { label: 'Roles', value: '6' },
      { label: 'Fiscal', value: 'ADR-009 gated' },
      { label: 'Surfaces', value: 'All clients' },
    ],
  }),

  'order-lifecycle': seedContent({
    title: 'Order Lifecycle',
    summary:
      'Placed → Vetted → Loaded → In transit → Arrived → Paid → Fiscalizing → Completed — one canonical state machine from state_machine.go.',
    problem:
      'When order status means different things to different teams, support calls multiply and payments misfire. Soft ARRIVED→COMPLETED shortcuts leave fiscal and ledger inconsistent.',
    outcomes: [
      'Every role reads the same status labels from one source of truth',
      'Terminal states trigger inventory and ledger side effects',
      'Cancel paths release stock and notify downstream roles',
      'Tracking updates without manual refresh with live alerts',
    ],
    howItWorks: defaultHowItWorks([
      ['Placed & vetted', 'Retailer checkout creates PENDING; supplier approves or rejects before dispatch.'],
      ['Loaded & sealed', 'Warehouse assigns trucks (LOADED); payload seals before IN_TRANSIT.'],
      ['In transit & arrived', 'Driver telemetry feeds maps; geofence gates ARRIVED and payment entry.'],
      ['Paid & completed', 'AWAITING_PAYMENT / PENDING_CASH_COLLECTION / DELIVERED_ON_CREDIT → FISCALIZING → COMPLETED.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 4 },
    relatedProjectSlug: 'supplier-control-plane',
    capabilities: cards([
      ['ValidateStatusTransition', 'Exact allow-list from code — same from==to is idempotent; everything else is deny-by-default.'],
      ['Payment forks', 'ARRIVED opens card, cash collection, credit delivery, or shop-closed pending.'],
      ['Delay & backorder', 'DELAYED, BACKORDERED, SCHEDULED, AUTO_ACCEPTED cover real wholesale patterns.'],
      ['Force with discipline', 'Force-complete paths are forbidden where fiscal integrity would break.'],
    ]),
    differentiators: cards([
      ['No soft complete', 'ADR-009: no ARRIVED → COMPLETED without fiscalizing.'],
      ['Sentinel errors', 'zone_miss, geofence_violation, inventory_exhausted, credit_limit_breached — actionable, not generic 500s.'],
      ['Cross-role parity', 'Portal and mobile read the same labels for the same order ID.'],
    ]),
    whyItMatters: {
      headline: 'Status accuracy is trust',
      body: 'Operators lose trust when AI or automation cannot explain why an order cannot complete. The lifecycle is the contract between field teams and finance.',
      insights: [
        { title: 'From firefighting to structure', body: 'Exceptions like SHOP_CLOSED_PENDING are first-class states, not chat threads.' },
        { title: 'Money follows state', body: 'Ledger and OFD movements attach to fiscalizing — not to informal “marked delivered” clicks.' },
      ],
    },
    edgeCases: cards([
      ['SHOP_CLOSED_PENDING', 'Driver can leave without fake delivery; later reopen to payment or cancel paths.'],
      ['FISCAL_FAILED', 'Retry FISCALIZING or force COMPLETED only through audited paths.'],
      ['CANCEL_REQUESTED mid-route', 'Must resolve to CANCELLED or resume LOADED/IN_TRANSIT/ARRIVED.'],
      ['zone_miss', 'Checkout and dispatch reject destinations outside topology service areas.'],
    ]),
    specs: [
      { label: 'Source of truth', value: 'order/state_machine.go' },
      { label: 'Money unit', value: 'Integer minor units' },
      { label: 'Fiscal', value: 'ADR-009 hard-gate' },
      { label: 'Safe retries', value: 'Required on critical save actions' },
    ],
  }),

  'supplier-control-plane': seedContent({
    title: 'Supplier Control Plane',
    summary: 'Vetting, topology, treasury, and dispatch preview from one supplier portal — the ADMIN role is the supplier.',
    problem:
      'Supplier operators cannot see network performance, pending vetting, or treasury exposure without exporting CSVs from five tools — and “platform admin” is a separate product that drifts from ops reality.',
    outcomes: [
      'Approve orders before they hit the warehouse floor',
      'Preview dispatch assignments with override authority and freeze-lock safety',
      'Manage catalog, zones, and fleet settings centrally',
      'Treasury and dispute visibility in one dashboard',
    ],
    howItWorks: defaultHowItWorks([
      ['Vet incoming orders', 'Review eligibility, stock, and retailer history before fulfillment.'],
      ['Shape the network', 'Topology changes propagate to warehouse dispatch rules and retailer checkout.'],
      ['Preview dispatch', 'See how warehouses load trucks; override when strategy demands — with audit.'],
    ]),
    flow: 'controlPlane',
    flowConfig: { roles: ['Supplier', 'Warehouse', 'Retailer', 'Driver', 'Factory', 'Payload'] },
    relatedProjectSlug: 'supplier-control-plane',
    capabilities: cards([
      ['Supplier = ADMIN', 'There is no separate platform admin — the supplier portal is the control plane.'],
      ['Topology + treasury', 'Network graph and money views live next to order queues.'],
      ['Dispatch preview', 'Supplier sees warehouse Smart Fit outcomes without owning the floor board.'],
      ['Staff + settings', 'Products, pricing, CRM, fleet, and returns from one shell.'],
    ]),
    differentiators: cards([
      ['JWT-scoped perimeter', 'ResolveSupplierID from claims — never trust body supplier_id.'],
      ['Role parity matrix', 'Desktop/portal features tracked against Android/iOS for the same role.'],
      ['Double-entry ledger', 'Money movements post debit and credit with the same confirmed save as business state.'],
    ]),
    whyItMatters: {
      headline: 'Suppliers need a cockpit, not another spreadsheet',
      body: 'Wholesale suppliers run catalog, warehouses, factories, fleet, and finance. Fragmented tools create blind spots exactly when peak dispatch and cash collection collide.',
      insights: [
        { title: 'One product, many surfaces', body: 'supplier-portal, Android, and iOS share contracts — desktop stub redirects to portal where appropriate.' },
        { title: 'Oversight ≠ micromanagement', body: 'Preview and override without replacing warehouse freeze-lock authority.' },
      ],
    },
    edgeCases: cards([
      ['Vet reject releases stock', 'Rejected orders free reserved inventory before warehouse ever sees them.'],
      ['Permission delegation', 'Staff roles cover managers away without sharing root credentials.'],
    ]),
  }),

  'mutating-handler-contract': seedContent({
    title: 'Safe Updates',
    summary: 'Verify → Validate → Save → Refresh → Notify — every state change follows the same guarded pipeline.',
    problem:
      'Ad-hoc API handlers skip screen refresh or emit events outside transactions, leaving clients showing stale data while the shared record disagrees.',
    outcomes: [
      'Consistent mutation path across all role routes',
      'Outbox writes in the same transaction as row updates',
      'Cache keys invalidated post-commit',
      'instant updates after durable persistence',
    ],
    howItWorks: defaultHowItWorks([
      ['Verify & validate', 'Auth claims, safe-retry keys, and business rules before any write.'],
      ['Save in one transaction', 'Confirmed save with the change event in the same step.'],
      ['Refresh & notify', 'background jobs refresh caches and push live updates to each role.'],
    ]),
    flow: 'mutatingHandler',
    flowConfig: { highlightStep: 2 },
    capabilities: cards([
      ['safe-retry key', 'Required on critical save actions so retries never double-run.'],
      ['Same-txn change log', 'No write twice: event durability equals row durability.'],
      ['Post-commit only', 'Cache and WS happen after commit — clients never celebrate rolled-back writes.'],
      ['Role-scoped notify', 'Hubs fan out only to rooms that should see the aggregate.'],
    ]),
    differentiators: cards([
      ['One contract, 400+ routes', 'backend-go mounts share the pattern instead of per-feature inventiveness.'],
      ['Explainable failures', 'Validation errors surface as sentinel codes teams can act on.'],
    ]),
    whyItMatters: {
      headline: 'Stale screens destroy operator trust',
      body: 'If dispatch boards show LOADED while the database still says PENDING, drivers leave empty and finance chases ghosts. The mutation contract is how Pegasus keeps screens honest.',
      insights: [
        { title: 'Events after truth', body: 'Outbox relay publishes only what committed.' },
        { title: 'Replay-safe consumers', body: 'Workers are idempotent under live events redelivery.' },
      ],
    },
    edgeCases: cards([
      ['Partial failure after commit', 'Consumers retry; handlers do not re-insert conflicting domain rows.'],
      ['Missing safe retries key', 'Route rejects rather than risk duplicate side effects.'],
    ]),
    specs: [
      { label: 'DB', value: 'shared system of record RW' },
      { label: 'Events', value: 'Reliable change events' },
      { label: 'Cache', value: 'Cache refresh' },
      { label: 'Push', value: 'live coordination' },
    ],
  }),

  'reliable-updates': seedContent({
    title: 'Reliable Updates',
    summary: 'Reliable change events — no mismatched screens after mutations.',
    problem:
      'Dual-write failures leave apps showing orders as dispatched when the database still says loading — the classic logistics trust killer.',
    outcomes: [
      'Events emitted only after commit succeeds',
      'Safe-retry consumers safe under replay',
      'Silent live refresh on connected clients',
      'Cross-role parity on order and fleet state',
    ],
    howItWorks: defaultHowItWorks([
      ['Write once', 'Row update and change log insert share one confirmed save.'],
      ['Publish async', 'live events picks up change events with per-aggregate ordering.'],
      ['Invalidate & push', 'caches refresh; live updates reach every role.'],
    ]),
    flow: 'realtimePipeline',
    relatedProjectSlug: 'realtime-coordination',
    capabilities: cards([
      ['Outbox relay', 'Workers drain durable events — not in-request fire-and-forget publishes.'],
      ['Cache discipline', 'Invalidation is explicit and post-commit.'],
      ['Silent refresh', 'Clients refresh collections without toast spam.'],
      ['Cross-role sync', 'Supplier preview, warehouse board, and retailer tracking converge on the same IDs.'],
    ]),
    differentiators: cards([
      ['No write twice', 'live updates follow the confirmed save — never write beside it.'],
      ['Ordering per aggregate', 'Order and fleet updates do not leapfrog incorrectly.'],
    ]),
    whyItMatters: {
      headline: 'Realtime only counts if it is true',
      body: 'Logistics UIs that lie under load are worse than slow UIs. Reliable updates make “live” mean committed.',
      insights: [
        { title: 'Replay is normal', body: 'Consumers must tolerate at-least-once delivery.' },
        { title: 'Notify after durable', body: 'Live updates never show unconfirmed status.' },
      ],
    },
    edgeCases: cards([
      ['Consumer lag', 'Boards may be briefly stale but never invent statuses the shared record does not hold.'],
      ['Reconnect storm', 'Clients re-subscribe to hubs; snapshots repair gaps.'],
    ]),
  }),

  'network-topology': seedContent({
    title: 'Network Topology',
    summary: 'Warehouses, factories, zones, and gate seals — the graph your dispatch rules and checkout validation run on.',
    problem:
      'Delivery zones drawn on paper do not stop retailers outside service areas from checking out — until a truck is already loaded for a zone-miss.',
    outcomes: [
      'delivery zones zone validation at checkout',
      'Warehouse-factory co-location for internal transfers',
      'Fleet and service area tied to topology nodes',
      'Clear zone-miss errors for retailers and suppliers',
    ],
    howItWorks: defaultHowItWorks([
      ['Model the network', 'Suppliers define warehouses, factories, zones, and fleet home bases.'],
      ['Enforce at checkout', 'Retailers outside zones see actionable errors, not silent failures.'],
      ['Drive dispatch rules', 'Eligibility, fees, and Smart Fit routing respect topology edges and delivery zones cells.'],
    ]),
    flow: 'topologyMap',
    relatedProjectSlug: 'network-topology',
    capabilities: cards([
      ['delivery zones spatial model', 'Uber delivery zones (res 7 class) underpins zone checks and dispatch clustering.'],
      ['Node types', 'Warehouse, factory, and home-node scoping for drivers and payload.'],
      ['Gate seals', 'Payload terminal confirms load integrity before departure.'],
      ['Service areas', 'Checkout and dispatch share the same geography contract.'],
    ]),
    differentiators: cards([
      ['zone_miss as product', 'First-class error — not a generic validation message buried in logs.'],
      ['Topology drives money', 'Fees and eligibility can attach to edges, not hardcoded city lists.'],
    ]),
    whyItMatters: {
      headline: 'Geography is a business rule, not a map widget',
      body: 'If topology is wrong, every downstream system invents workarounds. Pegasus treats the network graph as operational truth.',
      insights: [
        { title: 'Prevent bad checkouts', body: 'Stop impossible deliveries before payment authorization.' },
        { title: 'Same graph for AI', body: 'Dispatch optimizer and proven Smart Fit rules read the same plan data.' },
      ],
    },
    edgeCases: cards([
      ['zone_miss at checkout', 'Retailer sees why the destination is outside service.'],
      ['Node offline', 'Dispatch eligibility excludes unavailable warehouses without crashing the board.'],
    ]),
  }),

  'trust-reliability': seedContent({
    title: 'Trust & Reliability',
    summary: 'Status accuracy, payment safety, and human override when automation is not enough.',
    problem:
      'Operators lose trust when AI suggestions cannot be overridden or when payment state disagrees with delivery proof.',
    outcomes: [
      'Human override on dispatch and supplier decisions with freeze-lock safety',
      'Pay-at-delivery with geofenced collection',
      'Audit trails on force-dispatch and reassignment',
      'Explain banners in plain language per role',
    ],
    howItWorks: defaultHowItWorks([
      ['Accurate status', 'Single lifecycle state machine every client reads.'],
      ['Safe payments', 'Cash and card collection gated on arrival events and fiscalizing.'],
      ['Override with accountability', 'Force actions logged; downstream impact previewed.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 5 },
    capabilities: cards([
      ['Freeze-lock', 'Manual intervention and AI cannot race — locks are the single source of override truth.'],
      ['Deterministic AI fallback', 'Optimizer timeouts fall back to pure zone-aware  load packing engines.'],
      ['Geofenced money', 'Cash collection requires proximity when policy says so.'],
      ['Auditability', 'Force paths leave trails operators and finance can reconstruct.'],
    ]),
    differentiators: cards([
      ['Automation that yields', 'Pegasus prefers a correct human override over a clever stuck algorithm.'],
      ['Fiscal integrity', 'Trust includes tax receipts — COMPLETED is not a casual click.'],
    ]),
    whyItMatters: {
      headline: 'Trust is an operational KPI',
      body: 'If floor leads disable the system because overrides are impossible, the platform has already failed. Reliability includes governance.',
      insights: [
        { title: 'Explain, then act', body: 'Banners tell roles why a transition is blocked.' },
        { title: 'Safety over speed', body: '2.5s AI timeout with fallback beats waiting forever.' },
      ],
    },
    edgeCases: cards([
      ['force_complete_forbidden', 'Code blocks completes that would skip fiscal integrity.'],
      ['proximity_required', 'Arrival/payment rejected outside geofence when required.'],
      ['AI timeout', 'Dispatch continues on the proven Smart Fit path.'],
    ]),
  }),
};
