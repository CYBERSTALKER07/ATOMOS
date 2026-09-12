import { seedContent, defaultHowItWorks, cards, DEFAULT_PROOF, DEFAULT_AI_DATA } from './helpers';

/** Role topics grounded in role feature map + ECOSYSTEM_FEATURES_BY_ROLE. */
export const rolesTopics = {
  supplier: seedContent({
    title: 'Supplier',
    summary:
      'ADMIN control plane — order vetting, dispatch preview, topology, treasury, and exception ops from supplier-portal (and Android).',
    problem:
      'Without a single supplier oversight view, orders jump from retailer checkout to warehouse floors with no eligibility gate, and finance reconciles from spreadsheets.',
    outcomes: [
      'Vet / reject / negotiate before warehouse dispatch',
      'Dispatch preview and reassign without leaving the portal',
      'Topology, catalog, pricing, and treasury on one shared order truth',
      'Shop-closed resolve, early-complete, and chargeback workflows',
    ],
    howItWorks: defaultHowItWorks([
      ['Review new orders', 'Retailer places order; supplier vet / negotiate / payment-bypass paths gate eligibility.'],
      ['Oversee dispatch', 'Preview warehouse loads; recommend-reassign and fleet tools when a truck is wrong.'],
      ['Settle the network', 'Ledger, treasury, earnings, chargebacks, and reconciliation in finance packs.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Supplier', 'Vet orders', 'Preview dispatch', 'Treasury'] },
    relatedProjectSlug: 'supplier-control-plane',
    crossRole: [
      { role: 'Retailer', touchpoint: 'Orders wait for vet / negotiate before floor work' },
      { role: 'Warehouse', touchpoint: 'Topology and zones shape the dispatch board' },
    ],
    capabilities: cards([
      ['Order vetting & negotiations', 'supplierroutes: vet, negotiate/resolve, payment-bypass, receipts.'],
      ['Dispatch & fleet oversight', 'Dispatch preview, recommend-reassign, org-fleet, ops map.'],
      ['Topology & catalog', 'Factories, warehouses, zones, lanes, pricing, promotions, inventory import.'],
      ['Treasury & compliance', 'Ledger, payments, chargebacks, reconciliation, tax regimes, credit.'],
    ]),
    differentiators: cards([
      ['Single-tenant ADMIN scope', 'JWT ADMIN + SupplierID — ResolveSupplierID seeds the authenticated supplier.'],
      ['Portal + Android parity', 'SupplierShell and SupplierSection.kt share the same route domains.'],
      ['Exception ownership', 'Shop-closed resolve and early-complete approval stay with the supplier.'],
    ]),
    whyItMatters: {
      headline: 'The network owner needs one place to gate fulfillment',
      body: 'Supplier is the commercial and topological authority. role feature map maps ADMIN to supplier-portal surfaces spanning orders, fleet, topology, and finance — not a thin “approve” button.',
      insights: [
        {
          title: 'Gate before pick',
          body: 'Vetting holds orders out of warehouse eligibility until the supplier acts — floor teams never pick surprise loads.',
        },
        {
          title: 'Finance on the same truth',
          body: 'Treasury and chargebacks read the same order and payment records the driver completed at the door.',
        },
      ],
    },
    edgeCases: cards([
      ['Shop closed mid-route', 'Supplier shop-closed active/resolve coordinates with retailer respond and driver hold.'],
      ['Early complete', 'Supplier must approve early-complete before the route is treated as done.'],
      ['Reassign under capacity', 'recommend-reassign / reassign-order keep capacity checks on the target truck.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
    proofItems: [
      { label: 'Role', value: 'ADMIN (JWT)' },
      { label: 'Clients', value: 'Portal · Android' },
      { label: 'System of record', value: 'shared system of record' },
      { label: 'Surfaces', value: 'Orders · Fleet · Treasury' },
    ],
  }),

  warehouse: seedContent({
    title: 'Warehouse',
    summary:
      'WAREHOUSE_ADMIN / WAREHOUSE — visual dispatch board, fleet assign, stock commitments, and live map after seal.',
    problem:
      'Peak dispatch windows leave no room for radio-driven load plans; misloads and capacity overflows show up only after the gate.',
    outcomes: [
      'Visual truck selector with order checkboxes',
      'Optional Smart Fit / AI suggestions — human always commits',
      'Live fleet map after trucks leave the gate',
      'Stock reservations stay consistent with confirmed saves',
    ],
    howItWorks: defaultHowItWorks([
      ['Open dispatch board', 'Eligible orders filtered by payment, zone, and stock.'],
      ['Load and commit', 'Assign orders; Smart Fit handles overflow across trucks.'],
      ['Track departure', 'After payload seal, fleet map shows live progress vs routing plan.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Warehouse', 'Open board', 'Load trucks', 'Track fleet'] },
    relatedProjectSlug: 'warehouse-operations',
    capabilities: cards([
      ['Dispatch board', 'Truck-and-order matching with capacity and zone eligibility.'],
      ['Fleet CRUD & assign', 'Trucks, drivers, and assign guards before departure.'],
      ['Stock commitments', 'Reservations released on reject / cancel paths.'],
      ['Live ops map', 'Planned geometry + telemetry after seal.'],
    ]),
    differentiators: cards([
      ['Human-in-the-loop', 'AI suggestions are opt-in; force-dispatch is audited.'],
      ['Portal + mobile floor', 'warehouse-portal and native apps share contracts.'],
      ['Same live role updates', 'Board silent-refreshes when supplier vets or retailer cancels.'],
    ]),
    whyItMatters: {
      headline: 'Morning dispatch is the highest-stakes minute',
      body: 'Warehouse owns load planning. AUTO_DISPATCH_IMPROVEMENT_PLAN and FEATURES docs keep suggestions advisory — the commit path is always a warehouse lead action.',
      insights: [
        { title: 'Eligibility first', body: 'Payment state, zone, and stock filter the board before any load packing runs.' },
        { title: 'Overflow is a playbook', body: 'Smart Fit / truck-too-small paths split loads instead of silent overfill.' },
      ],
    },
    edgeCases: cards([
      ['Truck too small', 'Overflow to next eligible truck with operator confirmation.'],
      ['Freeze-lock', 'Catalog/price freezes block mutation mid-pick until unlocked.'],
      ['Fiscal hard-gate', 'ADR-009: fiscal failure blocks dispatch commit.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
    proofItems: DEFAULT_PROOF,
  }),

  factory: seedContent({
    title: 'Factory',
    summary:
      'FACTORY / FACTORY_ADMIN — supply request ACK → production → FULFILL, with manifest sealing and co-locate transfers.',
    problem:
      'Loading bay teams lose track of which supply request and manifest belong on which truck when production and warehouse are loosely coupled.',
    outcomes: [
      'Supply request ACK → IN_PRODUCTION → READY → FULFILL',
      'Manifest sealing before payload handoff',
      'Co-locate mode for internal transfers',
      'Factory portal and mobile for bay teams',
    ],
    howItWorks: defaultHowItWorks([
      ['Acknowledge supply', 'Warehouse supply requests appear on the factory queue.'],
      ['Produce & stage', 'Move supply through production states to READY.'],
      ['Fulfill & manifest', 'Create manifest; payload seals at gate.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Factory', 'Ack supply', 'Stage goods', 'Seal manifest'] },
    relatedProjectSlug: 'factory-loading',
    capabilities: cards([
      ['Supply lifecycle', 'ACK and production state machine tied to warehouse demand.'],
      ['Manifest creation', 'Stage loads for payload gate match.'],
      ['Co-locate transfers', 'Internal node transfers without fake retailer orders.'],
      ['Bay-ready clients', 'factory-portal + Android/iOS.'],
    ]),
    differentiators: cards([
      ['Explicit supply states', 'No silent jump from request to loaded truck.'],
      ['Gate handoff', 'Payload role seals — factory does not fake departure.'],
    ]),
    whyItMatters: {
      headline: 'Production must speak the same order language',
      body: 'Factory apps sit on the same shared order record so warehouse boards and payload terminals never disagree about which SKUs are READY.',
      insights: [
        { title: 'ACK is the contract', body: 'Warehouse demand is acknowledged before production capacity is assumed.' },
        { title: 'Manifest before wheels', body: 'Seal ownership stays with payload/gate after factory stages.' },
      ],
    },
    edgeCases: cards([
      ['Partial fulfill', 'Split supply across manifests when capacity is short.'],
      ['Co-locate exception', 'Internal transfer skips retailer payment paths.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
  }),

  driver: seedContent({
    title: 'Driver',
    summary:
      'DRIVER — mission view with routing geometry, geofenced arrival, shop-closed flows, and COD/card at the door. No portal; Android/iOS only.',
    problem:
      'Drivers need stop-by-stop guidance and simple cash collection — not another ERP screen that fails offline at the curb.',
    outcomes: [
      'Stop-by-stop mission with planned route geometry',
      'Geofenced arrival and shop-closed hold',
      'Cash and card collection at the door',
      'Loss-tolerant telemetry posts',
    ],
    howItWorks: defaultHowItWorks([
      ['Receive manifest', 'Assignment appears after gate seal.'],
      ['Execute route', 'Navigate stops; report GPS telemetry.'],
      ['Complete delivery', 'Arrive, collect payment, attach proof.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Driver', 'Get manifest', 'Run route', 'Collect & complete'] },
    relatedProjectSlug: 'driver-execution-app',
    capabilities: cards([
      ['Mission execution', 'Sealed manifest, ordered stops, delivery confirmation.'],
      ['Geofence gates', 'Arrival and collect-cash require location truth.'],
      ['Pay-at-delivery', 'Cash entry and card-at-door sessions.'],
      ['Telemetry', 'HTTP posts designed for lossy field networks.'],
    ]),
    differentiators: cards([
      ['No driver portal', 'Native-only — role feature map lists driver apps without a web shell.'],
      ['Payment at curb', 'No pre-pay at checkout; obligation clears on collection.'],
    ]),
    whyItMatters: {
      headline: 'Execution quality is the customer experience',
      body: 'Retailer tracking and supplier treasury only stay honest if the driver app enforces geofence and payment states on the same order aggregate.',
      insights: [
        { title: 'Plan vs actual', body: 'routing geometry on the manifest makes “delayed” measurable.' },
        { title: 'Shop closed is a state', body: 'SHOP_CLOSED_PENDING holds the stop until retailer/supplier resolve.' },
      ],
    },
    edgeCases: cards([
      ['Shop closed', 'Hold stop; notify retailer and supplier resolve paths.'],
      ['Partial delivery', 'Claims and exception SOPs after incomplete drop.'],
      ['Cash short', 'Exception path into treasury / dispute workflows.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
    proofItems: [
      { label: 'Role', value: 'DRIVER' },
      { label: 'Clients', value: 'Android · iOS' },
      { label: 'Payment', value: 'Pay-at-delivery' },
      { label: 'Realtime', value: 'Telemetry · WS notify' },
    ],
  }),

  retailer: seedContent({
    title: 'Retailer',
    summary:
      'RETAILER — catalog, checkout, tracking, shop-closed respond, POS/stock, and AI assist across desktop, Android, and iOS.',
    problem:
      'Store managers waste hours calling warehouses for status while checkout, dock receiving, and POS live in disconnected tools.',
    outcomes: [
      'Unified checkout with zone and stock validation',
      'Live tracking with plain-language status',
      'Pay-at-delivery — cash or card at the door',
      'Retail OS: stock, POS, shifts, sections, HQ',
    ],
    howItWorks: defaultHowItWorks([
      ['Browse and order', 'Catalog from attached suppliers; checkout validates zone.'],
      ['Track live', 'Pulse / tracking without calling support.'],
      ['Receive and pay', 'Confirm delivery; pay driver on arrival.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Retailer', 'Place order', 'Track delivery', 'Pay at door'] },
    relatedProjectSlug: 'retailer-commerce',
    capabilities: cards([
      ['Procurement', 'Suppliers, cart sync, quote, cash/card/unified checkout.'],
      ['Orders & delivery', 'Cancel, preorder, delivery proposals, shop-closed respond.'],
      ['Tracking & pulse', 'Tracking, active fulfillment, control-tower pulse.'],
      ['Retail OS', 'Stock, local SKUs, POS, shifts, sections, HQ, reports.'],
    ]),
    differentiators: cards([
      ['Desktop + mobile parity', 'RetailerShell.tsx and RetailerNavigation.kt cover procurement through POS.'],
      ['No pre-pay at checkout', 'Obligation creates; payment completes at door.'],
      ['AI assist opt-in', 'Predictions and auto-order settings with confirm/reject.'],
    ]),
    whyItMatters: {
      headline: 'The store is both buyer and last-mile customer',
      body: 'role feature map lists 145+ retailer route registrations — procurement, claims, credit, and retail OS — on the same shared order record.',
      insights: [
        { title: 'Status without calls', body: 'Plain-language banners map lifecycle codes to store-manager copy.' },
        { title: 'Shop-closed respond', body: 'Retailer can answer a mid-route closed-shop hold without phone tag.' },
      ],
    },
    edgeCases: cards([
      ['Shop-closed respond', 'Accept reschedule / cancel paths from the hold state.'],
      ['Delivery proposal reject', 'Retailer rejects proposed window; order returns to planning.'],
      ['Claim after delivery', 'Claims eligibility and file routes on the order aggregate.'],
    ]),
    aiAndData: [
      ...DEFAULT_AI_DATA.slice(0, 2),
      {
        title: 'Retailer AI paths',
        description:
          'Auto-order run, AI predictions confirm/reject, reorder suggestions — always human-gated.',
      },
    ],
  }),

  'payload-gate': seedContent({
    title: 'Payload / Gate',
    summary:
      'PAYLOAD — per-truck seal with driver gate match, manifest lifecycle on terminal and tablet, reassign with capacity checks.',
    problem:
      'Wrong truck sealed means the wrong driver leaves with the wrong manifest — and every downstream screen lies.',
    outcomes: [
      'Per-truck seal with driver gate match',
      'Manifest lifecycle through terminal and tablet',
      'Reassign and override with capacity checks',
      'Departure notifies driver and warehouse WS rooms',
    ],
    howItWorks: defaultHowItWorks([
      ['Scan manifest', 'Verify truck, driver, and order list.'],
      ['Seal load', 'Lock manifest; notify driver and warehouse.'],
      ['Handle exceptions', 'Reassign driver; rebalance across manifests.'],
    ]),
    flow: 'roleJourney',
    flowConfig: { roles: ['Payload', 'Verify manifest', 'Seal truck', 'Gate release'] },
    relatedProjectSlug: 'payload-gate-control',
    capabilities: cards([
      ['Gate match', 'Truck + driver identity before seal.'],
      ['Manifest seal', 'Locks load; starts driver mission.'],
      ['Exception reassign', 'Capacity-checked rebalance.'],
      ['Terminal + mobile', 'payload-terminal and native apps.'],
    ]),
    differentiators: cards([
      ['Seal is the departure event', 'Fleet map and driver mission unlock only after seal.'],
      ['Physical accountability', 'Gate operators own the last check before wheels roll.'],
    ]),
    whyItMatters: {
      headline: 'The gate is the system boundary between plan and reality',
      body: 'Payload seal turns warehouse load plans into immutable manifests for drivers — change events fan that truth to every role.',
      insights: [
        { title: 'No silent departure', body: 'Without seal, tracking and COD paths should not start.' },
        { title: 'Reassign is audited', body: 'Override paths keep capacity and driver match consistent.' },
      ],
    },
    edgeCases: cards([
      ['Driver mismatch', 'Block seal until gate match passes.'],
      ['Partial load', 'Rebalance orders across manifests before seal.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
  }),

  finance: seedContent({
    title: 'Finance & Treasury',
    summary:
      'Pay-at-delivery, driver collection, supplier treasury, chargebacks, and reconciliation on the same shared order truth.',
    problem:
      'Finance cannot close books when cash-at-door, card sessions, and disputes live outside the order system.',
    outcomes: [
      'No pre-pay at checkout — obligation clears on collection',
      'Geofenced cash and card-at-door tied to the order row',
      'Supplier treasury, earnings, chargebacks, reconciliation',
      'Dispute paths leave an audit trail operators can reconstruct',
    ],
    howItWorks: defaultHowItWorks([
      ['Create obligation', 'Checkout creates payment state without charging the retailer.'],
      ['Collect at door', 'Driver tools for cash entry or card session after geofence arrival.'],
      ['Settle & dispute', 'Treasury and chargeback packs reconcile against the completed order.'],
    ]),
    flow: 'paymentFlow',
    flowConfig: { highlightStep: 2 },
    relatedProjectSlug: 'payment-integrity',
    crossRole: [
      { role: 'Driver', touchpoint: 'Collects cash/card at the stop' },
      { role: 'Retailer', touchpoint: 'Confirms delivery / confirm-cash when required' },
      { role: 'Supplier', touchpoint: 'Sees treasury, chargebacks, and reconciliation' },
    ],
    capabilities: cards([
      ['Pay-at-delivery', 'Cash and card collection only after arrival — never at catalog checkout.'],
      ['Treasury hub', 'Supplier ledger, earnings, payments, and export views.'],
      ['Chargebacks & disputes', 'Exception quarantine with audited resolve paths.'],
      ['Reconciliation', 'Driver collections matched to expected invoices on the shared record rows.'],
    ]),
    differentiators: cards([
      ['Same order aggregate', 'Finance reads the delivery/payment state drivers and retailers already committed.'],
      ['Integer money', 'Money handled as integer minor units — no float drift in settlement.'],
      ['Live role updates', 'Payment mutations notify supplier and retailer WS rooms in the same write path.'],
    ]),
    whyItMatters: {
      headline: 'Settlement has to share the delivery state machine',
      body: 'role feature map maps supplier finance packs (ledger, treasury, chargebacks, reconciliation) to the same order record that drivers complete at the curb. Split systems create end-of-day ghosts.',
      insights: [
        {
          title: 'Obligation ≠ payment',
          body: 'Checkout creates the commercial obligation; cash/card at the door completes it under geofence rules.',
        },
        {
          title: 'Disputes stay on the order',
          body: 'Chargebacks and partials quarantine against the order id — not a side spreadsheet.',
        },
      ],
    },
    edgeCases: cards([
      ['Cash short', 'Exception into dispute / chargeback before treasury marks settled.'],
      ['Card-at-door failure', 'Retry or fall back to cash with audited state transitions.'],
      ['Early complete', 'Supplier approval required before finance treats the route as done.'],
    ]),
    aiAndData: DEFAULT_AI_DATA,
    proofItems: [
      { label: 'Payment model', value: 'Pay-at-delivery' },
      { label: 'Source of truth', value: 'Shared order record' },
      { label: 'Surfaces', value: 'Supplier treasury · Driver collect' },
      { label: 'Integrity', value: 'Integer money · audited disputes' },
    ],
  }),

  'order-vetting': seedContent({
    title: 'Order Vetting',
    summary: 'Supplier approves before warehouse dispatch — no surprise loads on the floor.',
    problem: 'Warehouses discover ineligible orders only after picking has started.',
    outcomes: [
      'Orders held until supplier vet acts',
      'Reject path releases stock reservation',
      'Warehouse board shows only approved orders',
    ],
    howItWorks: defaultHowItWorks([
      ['Retailer submits', 'Order created pending vetting.'],
      ['Supplier reviews', 'Approve, reject, or negotiate.'],
      ['Warehouse dispatches', 'Only vetted orders appear on the board.'],
    ]),
    flow: 'orderLifecycle',
    flowConfig: { highlightStep: 1 },
    relatedProjectSlug: 'supplier-control-plane',
    capabilities: cards([
      ['Vet gate', 'supplier orders/vet before eligibility.'],
      ['Negotiate', 'Pending negotiations with resolve path.'],
      ['Stock release', 'Reject frees reservation in the same write path.'],
    ]),
    edgeCases: cards([
      ['Payment bypass', 'Audited supplier bypass when commercial exception applies.'],
      ['Negotiate loop', 'Retailer and supplier iterate without losing the order id.'],
    ]),
  }),

  'cash-collection': seedContent({
    title: 'Cash Collection',
    summary: 'Driver COD flows with geofence, card-at-door, and treasury reconciliation.',
    problem: 'Cash at the door without system tracking creates end-of-day reconciliation nightmares.',
    outcomes: [
      'Geofenced collect-cash on arrival',
      'Card-at-door sessions when preferred',
      'Treasury ties collection to order and driver',
    ],
    howItWorks: defaultHowItWorks([
      ['Arrive at stop', 'Geofence confirms driver is at retailer.'],
      ['Collect payment', 'Cash entry or card session at door.'],
      ['Reconcile', 'Supplier treasury reflects completed collection.'],
    ]),
    flow: 'paymentFlow',
    flowConfig: { highlightStep: 3 },
    relatedProjectSlug: 'payment-integrity',
    edgeCases: cards([
      ['Cash short', 'Exception into dispute / chargeback paths.'],
      ['Confirm cash', 'Retailer confirm-cash route for doorstep acknowledgment.'],
    ]),
  }),

  'role-parity-matrix': seedContent({
    title: 'Role Parity Matrix',
    summary: 'Portal, mobile, and desktop for every team — same contracts, every surface.',
    problem: 'Warehouse gets a great app while retailers are stuck on a broken mobile web wrapper.',
    outcomes: [
      'Shared types and API client across clients',
      'Silent live refresh on every platform',
      'Role-row parity tracked in ROLE_ROW_PARITY_MATRIX',
    ],
    howItWorks: defaultHowItWorks([
      ['Define contracts', 'packages/types and api-client lead every feature.'],
      ['Ship per role', 'Portal, Android, iOS, desktop as applicable.'],
      ['Verify parity', 'Release checks gate cross-role flows.'],
    ]),
    flow: 'appsMatrix',
    crossRole: [{ role: 'All roles', touchpoint: 'Each row ships on every required client' }],
    capabilities: cards([
      ['Contract-first', 'Shared types before UI.'],
      ['Surface matrix', 'Documented in ROLE_ROW_PARITY_MATRIX.'],
      ['WS everywhere', 'Same envelope contract on portal and native.'],
    ]),
    proofItems: [
      { label: 'Roles', value: '6 connected' },
      { label: 'Parity doc', value: 'ROLE_ROW_PARITY_MATRIX' },
      { label: 'Realtime', value: 'live sync after every change' },
      { label: 'Surfaces', value: 'Portal · Mobile · Desktop' },
    ],
  }),
};
