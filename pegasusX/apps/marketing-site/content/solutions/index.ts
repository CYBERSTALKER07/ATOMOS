import type { SolutionSlug } from "@/lib/constants";

export type SolutionContent = {
  slug: SolutionSlug;
  title: string;
  summary: string;
  problem: string;
  outcomes: string[];
  howItWorks: { title: string; description: string }[];
  relatedCapabilities: string[];
  relatedRoles: string[];
};

export const solutionContent: Record<SolutionSlug, SolutionContent> = {
  "dispatch-accuracy": {
    slug: "dispatch-accuracy",
    title: "Dispatch the right load, every time",
    summary:
      "Give warehouse teams a clear board to match orders to trucks — with smart suggestions when they want help, and full control when they don't.",
    problem:
      "Dispatch windows are short. Wrong truck assignments mean late deliveries, wasted fuel, and angry retailers. Spreadsheets and radio calls don't scale.",
    outcomes: [
      "Fewer misloaded trucks leaving the yard",
      "Faster dispatch during peak morning windows",
      "Oversized orders split automatically across available capacity",
      "Gate teams confirm every load before departure",
    ],
    howItWorks: [
      {
        title: "See what's ready to go",
        description:
          "Eligible orders appear on the dispatch board with payment and stock status already checked.",
      },
      {
        title: "Pick truck and orders",
        description:
          "Warehouse leads select a truck, check the orders going on it, and preview the route before confirming.",
      },
      {
        title: "Confirm and notify",
        description:
          "Once sealed at the gate, drivers receive their manifest and retailers see tracking go live.",
      },
    ],
    relatedCapabilities: ["dispatch", "topology"],
    relatedRoles: ["warehouse", "driver", "payload"],
  },
  "fleet-visibility": {
    slug: "fleet-visibility",
    title: "See your fleet as it moves",
    summary:
      "Know where every truck is, whether it's on plan, and when deliveries will land — for ops teams and retailers alike.",
    problem:
      "When customers call asking 'where's my order?', your team shouldn't have to phone drivers one by one. You need one live map that tells the truth.",
    outcomes: [
      "Ops sees live vs. delayed status at a glance",
      "Retailers track deliveries without calling support",
      "Route deviations flagged before they become complaints",
      "Planned vs. actual progress on every active route",
    ],
    howItWorks: [
      {
        title: "Routes set at dispatch",
        description:
          "Every manifest ships with a planned route your team and drivers agree on before wheels roll.",
      },
      {
        title: "Live driver updates",
        description:
          "Driver apps report location as they move. The fleet map updates without manual refresh.",
      },
      {
        title: "Alerts when something's off",
        description:
          "If a truck strays from plan, ops gets notified — not after the customer does.",
      },
    ],
    relatedCapabilities: ["telemetry", "realtime"],
    relatedRoles: ["warehouse", "driver", "retailer"],
  },
  "payment-confidence": {
    slug: "payment-confidence",
    title: "Close the books without surprises",
    summary:
      "From retailer checkout to supplier treasury — payments, cash collection, and reconciliation stay aligned.",
    problem:
      "Cash-on-delivery, card payments, and scheduled billing create reconciliation nightmares. One missed webhook or duplicate charge erodes trust fast.",
    outcomes: [
      "Duplicate payment attempts blocked automatically",
      "Clear audit trail from checkout to settlement",
      "Cash collection tracked per stop and per driver",
      "Supplier treasury views that match what actually happened",
    ],
    howItWorks: [
      {
        title: "Checkout that fits your market",
        description:
          "Support card, cash-on-delivery, and scheduled payment flows — retailers pick what works for them.",
      },
      {
        title: "Driver cash collection",
        description:
          "Drivers confirm payment at delivery. Ops and treasury see it without end-of-day spreadsheets.",
      },
      {
        title: "Reconciliation built in",
        description:
          "Supplier dashboards show earnings, disputes, and chargebacks in one place.",
      },
    ],
    relatedCapabilities: ["payments", "outbox"],
    relatedRoles: ["retailer", "driver", "supplier"],
  },
  "network-coordination": {
    slug: "network-coordination",
    title: "Keep every partner aligned",
    summary:
      "Suppliers, warehouses, factories, and retailers work from the same live picture — no more version conflicts between teams.",
    problem:
      "A supplier changes a catalog item. A warehouse dispatches against old stock. A factory fulfills the wrong request. Fragmented tools create fragmented truth.",
    outcomes: [
      "One order status every team agrees on",
      "Supplier topology changes flow to warehouses and factories",
      "Factory supply requests visible to warehouse fulfillment",
      "Retailer orders respect delivery zones and vetting rules",
    ],
    howItWorks: [
      {
        title: "Suppliers set the network",
        description:
          "Define warehouses, factories, and service areas. Every downstream team inherits the same structure.",
      },
      {
        title: "Handoffs with guardrails",
        description:
          "Orders pass through vetting, stock checks, and gate seals — each step visible to the next team.",
      },
      {
        title: "Everyone sees the same status",
        description:
          "When something changes, every app updates. No overnight syncs, no 'check with the other team.'",
      },
    ],
    relatedCapabilities: ["topology", "realtime"],
    relatedRoles: ["supplier", "warehouse", "factory", "retailer"],
  },
};
