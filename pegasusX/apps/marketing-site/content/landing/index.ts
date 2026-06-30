import type { SpecRow } from "@/content/pegasus";

export const heroContent = {
  headline: "Run your logistics network from one place.",
  subheadline:
    "Pegasus connects suppliers, warehouses, factories, drivers, and retailers — so every order moves from checkout to delivery without dropped handoffs or blind spots.",
  primaryCta: { label: "Request a demo", href: "/contact" as const },
  secondaryCta: { label: "See how it works", href: "/platform" as const },
  scrollHint: "Scroll to explore",
};

export const heroSpecs: SpecRow[] = [
  { key: "Built for", value: "Multi-supplier logistics networks" },
  { key: "Covers", value: "Supplier · Warehouse · Factory · Driver · Retailer · Gate" },
  { key: "Works on", value: "Web · Desktop · Mobile · Loading terminals" },
  { key: "Tracks", value: "Order placed → Loaded → In transit → Delivered" },
];

export const controlPlaneLayers = [
  {
    id: "surfaces",
    title: "Apps for every role",
    body: "Each person in your network gets tools built for their job — warehouse dispatch, driver routes, retailer ordering, and more.",
    icon: "nextjs" as const,
    specs: [
      { key: "Web & desktop", value: "Operations dashboards and portals" },
      { key: "Mobile", value: "Field apps for drivers and floor teams" },
    ],
  },
  {
    id: "maglev",
    title: "Always available",
    body: "Your operations keep running even as demand spikes — requests route to the right place without manual failover.",
    icon: "kubernetes" as const,
    specs: [
      { key: "Uptime", value: "Built for high-traffic dispatch windows" },
      { key: "Scale", value: "Handles thousands of retailers per supplier" },
    ],
  },
  {
    id: "api",
    title: "One shared platform",
    body: "Orders, dispatch, fleet, payments, and tracking all run on the same foundation — so data stays consistent everywhere.",
    icon: "go" as const,
    specs: [
      { key: "Core areas", value: "Orders · Dispatch · Fleet · Payments · Tracking" },
      { key: "Updates", value: "Every screen sees the same status" },
    ],
  },
  {
    id: "spanner",
    title: "Nothing gets lost",
    body: "When something changes — an order status, a payment, a truck assignment — the update and the notification happen together.",
    icon: "spanner" as const,
    specs: [
      { key: "Accuracy", value: "Status and alerts always match" },
      { key: "Safety", value: "No half-finished updates during busy hours" },
    ],
  },
  {
    id: "redis-kafka",
    title: "Fast when it matters",
    body: "Dispatch boards and fleet maps refresh quickly, even when hundreds of orders are moving at once.",
    icon: "kafka" as const,
    specs: [
      { key: "Speed", value: "Live boards during peak dispatch" },
      { key: "Reach", value: "Updates flow to every app instantly" },
    ],
  },
  {
    id: "ws",
    title: "Live coordination",
    body: "Warehouses, drivers, and retailers see changes as they happen — no refreshing, no waiting on overnight syncs.",
    icon: "websocket" as const,
    specs: [
      { key: "Live updates", value: "Dispatch, tracking, and payments" },
      { key: "Coverage", value: "Every role in your network" },
    ],
  },
];

export const dispatchPipeline = [
  "Orders come in",
  "Eligible loads identified",
  "Grouped by area",
  "Matched to truck capacity",
  "Routes planned",
  "Assignments confirmed",
  "Drivers notified",
  "Deliveries executed",
  "Progress tracked live",
];

export const dispatchStats = [
  { label: "Orders", value: "12" },
  { label: "Areas", value: "3" },
  { label: "Trucks", value: "2" },
  { label: "Manifests", value: "1" },
];

export const dispatchSpecs: SpecRow[] = [
  { key: "Primary workflow", value: "Warehouse teams pick trucks and orders on a visual board" },
  { key: "Smart assist", value: "Optional auto-suggestions based on location and capacity" },
  { key: "Overflow", value: "Large orders split across trucks; nothing left behind" },
  { key: "Gate check", value: "Loads sealed before trucks leave the yard" },
];

export const telemetrySpecs: SpecRow[] = [
  { key: "Planned route", value: "Set when dispatch confirms the load" },
  { key: "Live progress", value: "Driver location compared to plan in real time" },
  { key: "Alerts", value: "Ops notified when a truck goes off route" },
  { key: "Fleet map", value: "Clear live vs. delayed status on every vehicle" },
];

export const financialSpecs: SpecRow[] = [
  { key: "Payments", value: "Duplicate charges prevented automatically" },
  { key: "Ledger", value: "Every payment recorded with a clear audit trail" },
  { key: "Checkout", value: "Card, cash, and scheduled payment paths supported" },
  { key: "Treasury", value: "Supplier reconciliation and dispute records in one view" },
];
