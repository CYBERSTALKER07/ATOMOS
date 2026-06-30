import type { CapabilitySlug } from "@/lib/constants";

export type CapabilityContent = {
  slug: CapabilitySlug;
  title: string;
  summary: string;
  whoItsFor: string;
  benefits: string[];
  steps: string[];
};

export const capabilityContent: Record<CapabilitySlug, CapabilityContent> = {
  dispatch: {
    slug: "dispatch",
    title: "Smarter dispatch",
    summary:
      "Match orders to trucks using location, capacity, and eligibility — with warehouse teams always in control of the final call.",
    whoItsFor: "Warehouse managers and dispatch leads running daily load planning",
    benefits: [
      "Visual board to pick trucks and orders",
      "Orders grouped by delivery area automatically",
      "Truck capacity checked before confirmation",
      "Oversized loads split across available trucks",
    ],
    steps: [
      "Orders arrive from retailers and pre-order schedules",
      "System checks payment, stock, and eligibility",
      "Orders grouped by delivery area",
      "Matched to truck capacity with room to spare",
      "Routes planned and manifests prepared",
      "Warehouse confirms — gate seals before departure",
      "Drivers notified and retailers see tracking live",
    ],
  },
  outbox: {
    slug: "outbox",
    title: "Reliable updates",
    summary:
      "When something changes in your network, every app gets the update at the same time — no half-finished states or mismatched screens.",
    whoItsFor: "Operations leaders who need every team seeing the same truth",
    benefits: [
      "Status changes and notifications happen together",
      "No 'the board says X but the driver sees Y' moments",
      "Safe during peak hours when volume spikes",
      "Audit trail for every state change",
    ],
    steps: [
      "A team member makes a change — dispatch, payment, or status",
      "The update and notification are saved together",
      "Every connected app receives the new state",
      "Dispatch boards, driver apps, and retailer tracking all agree",
    ],
  },
  payments: {
    slug: "payments",
    title: "Payment confidence",
    summary:
      "Checkout, cash collection, and supplier reconciliation — with duplicate charges blocked and every transaction traceable.",
    whoItsFor: "Finance teams and suppliers managing multi-payment-type networks",
    benefits: [
      "Card, cash, and scheduled payment paths",
      "Duplicate payment attempts blocked automatically",
      "Driver cash collection tracked per stop",
      "Supplier treasury with dispute records",
    ],
    steps: [
      "Retailer completes checkout with their payment method",
      "Payment status tracked through fulfillment",
      "Driver confirms collection at delivery if cash",
      "Supplier treasury reflects actual collections",
      "Disputes resolved with delivery proof attached",
    ],
  },
  telemetry: {
    slug: "telemetry",
    title: "Live fleet tracking",
    summary:
      "See where every truck is, compare progress to the planned route, and catch deviations before customers do.",
    whoItsFor: "Ops teams and retailers who need honest delivery visibility",
    benefits: [
      "Live fleet map with clear on-time vs. delayed status",
      "Planned route set at dispatch, tracked in real time",
      "Deviation alerts for ops before customer complaints",
      "Retailer self-serve tracking without support calls",
    ],
    steps: [
      "Route planned when dispatch confirms the manifest",
      "Driver app reports location as they move",
      "Ops fleet map updates without manual refresh",
      "Retailers see live progress on their tracking screen",
      "Alerts fire if a truck strays from the planned route",
    ],
  },
  realtime: {
    slug: "realtime",
    title: "Instant coordination",
    summary:
      "Dispatch boards, driver apps, and retailer screens update as work happens — no refreshing, no overnight syncs.",
    whoItsFor: "Any team tired of waiting for 'the system to catch up'",
    benefits: [
      "Changes visible across all apps within seconds",
      "Warehouse sees driver progress without calling",
      "Retailers track deliveries without refreshing",
      "Works across web, desktop, and mobile",
    ],
    steps: [
      "A change happens anywhere in the network",
      "Connected apps receive the update immediately",
      "Dispatch boards, maps, and tracking screens refresh",
      "Teams coordinate without phone tag or email chains",
    ],
  },
  topology: {
    slug: "topology",
    title: "One connected network",
    summary:
      "Suppliers define the network. Warehouses dispatch. Factories fulfill. Retailers order within their zone. Everyone shares the same structure.",
    whoItsFor: "Supplier operations teams managing multi-site networks",
    benefits: [
      "Define warehouses, factories, and service areas once",
      "Changes flow to every downstream team",
      "Retailer orders respect delivery zones",
      "Gate seals enforce handoff accountability",
    ],
    steps: [
      "Supplier sets up warehouses, factories, and service areas",
      "Warehouses manage fleet and dispatch capacity",
      "Factories fulfill supply requests from the network",
      "Retailers order within their delivery zone",
      "Gate teams seal loads before drivers depart",
    ],
  },
};
