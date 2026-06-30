export type SpecRow = { key: string; value: string };

export const pegasusPositioning = {
  headline: "Logistics software for teams that move physical goods.",
  subheadline:
    "Pegasus connects suppliers, warehouses, factories, drivers, and retailers — so every order moves from checkout to delivery without dropped handoffs.",
  bullets: [
    "Dedicated apps for every team in your network",
    "Live updates across dispatch boards, driver apps, and retailer tracking",
    "Smart dispatch suggestions with warehouse teams always in control",
    "Instant coordination — no overnight syncs or phone tag",
    "Secure access — every action tied to the right person and role",
  ],
};

export const mutatingHandlerContract: SpecRow[] = [
  { key: "1. Verify", value: "Confirm who is making the change and what they can access" },
  { key: "2. Validate", value: "Check the order, payment, or dispatch is eligible" },
  { key: "3. Save", value: "Update status and notify connected apps together" },
  { key: "4. Refresh", value: "Dispatch boards and tracking screens update immediately" },
  { key: "5. Notify", value: "Drivers, retailers, and ops teams see the change live" },
];

export const ecosystemComparison: SpecRow[] = [
  { key: "Pegasus", value: "Full supplier network — multiple operators, shared infrastructure" },
  { key: "Single-site", value: "One supplier, same apps and workflows" },
  { key: "Shared", value: "Same order flow, tracking, and payment experience" },
  { key: "Growth", value: "Start with one site, expand to a full network when ready" },
];

export const ecosystemBullets = [
  "Every team sees orders, stock, and dispatch from the same source",
  "Requests route reliably even during peak morning dispatch",
  "Retailers discover and order from suppliers in their delivery zone",
  "Drivers, factories, and gate teams always know which supplier they serve",
];

export const reliabilitySpecs: SpecRow[] = [
  { key: "Status accuracy", value: "Order updates and notifications happen together" },
  { key: "Payment safety", value: "Duplicate charges blocked automatically" },
  { key: "Manual override", value: "Ops can pause auto-suggestions during busy windows" },
  { key: "Rate protection", value: "System stays responsive during volume spikes" },
  { key: "Live data", value: "Every screen reflects the current state — not yesterday's export" },
];

export const reliabilityBullets = [
  "Clear order progress from placed to delivered",
  "Payment records that match what drivers actually collected",
  "Route tracking that shows planned vs. actual progress",
  "Role-based access so teams only see what they need",
];

export const pegasusXFootnote =
  "Pegasus scales from a single supplier operation to a full multi-site network — same apps, same workflows.";

export const trustStrip = [
  "Live coordination",
  "Secure access",
  "Payment protection",
  "Accurate tracking",
  "Peak-hour ready",
  "Human override",
];
