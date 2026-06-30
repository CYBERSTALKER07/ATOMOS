import { ROLES, type RoleSlug } from "@/lib/constants";

export type RoleContent = {
  slug: RoleSlug;
  headline: string;
  summary: string;
  persona: string;
  painPoints: string[];
  outcomes: string[];
  bullets: string[];
  workflows: { title: string; steps: string[] }[];
  crossRole: { role: string; touchpoint: string }[];
  capabilityLinks: string[];
};

export const roleContent: Record<RoleSlug, RoleContent> = {
  supplier: {
    slug: "supplier",
    headline: "Run your entire network from one place",
    summary:
      "See orders, manage your catalog, preview dispatch, and track earnings — without switching between tools.",
    persona: "Supplier CEO or operations director",
    painPoints: [
      "Orders sit in limbo between retailer requests and warehouse dispatch",
      "No single view of earnings, disputes, and network performance",
      "Topology changes don't reach warehouses fast enough",
    ],
    outcomes: [
      "Approve or reject orders before they hit the warehouse floor",
      "Preview dispatch assignments before trucks roll",
      "Treasury and reconciliation in one dashboard",
    ],
    bullets: [
      "Order vetting before fulfillment begins",
      "Dispatch preview with override when needed",
      "Catalog, topology, and treasury in one portal",
    ],
    workflows: [
      {
        title: "Review new orders",
        steps: [
          "Retailer places an order",
          "You review eligibility and stock",
          "Approve or reject before dispatch",
        ],
      },
      {
        title: "Oversee dispatch",
        steps: [
          "See demand across your network",
          "Preview how warehouses are loading trucks",
          "Step in with overrides when needed",
        ],
      },
    ],
    crossRole: [
      { role: "Retailer", touchpoint: "Orders wait for your approval before fulfillment" },
      { role: "Warehouse", touchpoint: "Your topology and fleet settings shape their dispatch board" },
    ],
    capabilityLinks: ["dispatch", "realtime", "topology"],
  },
  warehouse: {
    slug: "warehouse",
    headline: "Dispatch with confidence, every morning",
    summary:
      "A visual dispatch board, live fleet map, and stock commitments — built for the pace of warehouse operations.",
    persona: "Warehouse manager or dispatch lead",
    painPoints: [
      "Peak dispatch windows leave no room for mistakes",
      "Hard to see which orders are ready and which trucks have capacity",
      "Radio calls and paper lists don't survive a busy Tuesday",
    ],
    outcomes: [
      "Pick trucks and orders on a clear visual board",
      "Optional smart suggestions when you want help",
      "Live fleet map after trucks leave the yard",
    ],
    bullets: [
      "Visual truck selector with order checkboxes",
      "Smart dispatch suggestions — opt in when ready",
      "Stock reserved at order creation — no overselling",
    ],
    workflows: [
      {
        title: "Morning dispatch",
        steps: [
          "Open the dispatch board",
          "Select a truck and check eligible orders",
          "Confirm the manifest — gate team seals before departure",
        ],
      },
      {
        title: "Manage pre-orders",
        steps: [
          "Review retailer schedule requests",
          "Accept or adjust proposed delivery windows",
          "Commit stock for confirmed schedules",
        ],
      },
    ],
    crossRole: [
      { role: "Retailer", touchpoint: "Receives delivery proposals and live tracking" },
      { role: "Gate team", touchpoint: "Seals every manifest before drivers depart" },
    ],
    capabilityLinks: ["dispatch", "telemetry", "realtime"],
  },
  factory: {
    slug: "factory",
    headline: "Keep production and loading in sync",
    summary:
      "Track manifests through loading, sealing, and dispatch — with supply requests flowing cleanly to your warehouse partners.",
    persona: "Factory floor manager",
    painPoints: [
      "Loading lanes get out of sync with warehouse demand",
      "Supply requests lost between factory and warehouse teams",
      "Unclear when a manifest is ready for a driver",
    ],
    outcomes: [
      "Clear manifest lifecycle from loading to departure",
      "Supply requests tracked through fulfillment",
      "Loading bay transfers visible to the whole network",
    ],
    bullets: [
      "Manifest tracking: load → seal → dispatch → complete",
      "Supply requests with warehouse fulfillment status",
      "Loading bay coordination with gate teams",
    ],
    workflows: [
      {
        title: "Load and seal a manifest",
        steps: [
          "Start loading orders onto the truck",
          "Confirm each item on the manifest",
          "Seal when complete — driver can depart",
        ],
      },
    ],
    crossRole: [
      { role: "Warehouse", touchpoint: "Fulfills your supply requests" },
      { role: "Driver", touchpoint: "Can't start route until manifest is sealed" },
    ],
    capabilityLinks: ["dispatch", "realtime"],
  },
  driver: {
    slug: "driver",
    headline: "Clear routes. Simple stops. On-time delivery.",
    summary:
      "Your manifest, your route, and your stops — with live tracking that keeps ops and retailers informed.",
    persona: "Delivery driver",
    painPoints: [
      "Manifests change after leaving the yard",
      "Unclear stop order and delivery instructions",
      "Cash collection tracked on paper at end of day",
    ],
    outcomes: [
      "Sealed manifest before you roll — no mid-route surprises",
      "Step-by-step stop progression with clear instructions",
      "Payment confirmation built into each delivery",
    ],
    bullets: [
      "Route unlocked only after gate seal",
      "Stop-by-stop: arrive → deliver → confirm → complete",
      "Cash collection recorded per stop automatically",
    ],
    workflows: [
      {
        title: "Run your route",
        steps: [
          "Accept your sealed manifest",
          "Follow the planned route stop by stop",
          "Confirm delivery and payment at each location",
        ],
      },
    ],
    crossRole: [
      { role: "Warehouse", touchpoint: "Assigns your manifest and route" },
      { role: "Retailer", touchpoint: "Tracks your progress live on their screen" },
    ],
    capabilityLinks: ["telemetry", "realtime"],
  },
  retailer: {
    slug: "retailer",
    headline: "Order, pay, and track — without phone calls",
    summary:
      "Browse catalog, place orders, choose delivery windows, and track every shipment from checkout to your door.",
    persona: "Retailer buyer or store manager",
    painPoints: [
      "No visibility into when orders will actually arrive",
      "Calling the supplier for every status update",
      "Checkout doesn't fit local payment habits",
    ],
    outcomes: [
      "Self-serve tracking without calling support",
      "Scheduled and on-demand ordering in one catalog",
      "Payment options that match your market",
    ],
    bullets: [
      "Standard and scheduled ordering flows",
      "Delivery proposals you can accept or adjust",
      "Receipt and proof of delivery in one place",
    ],
    workflows: [
      {
        title: "Place and track an order",
        steps: [
          "Browse the supplier catalog",
          "Check delivery availability for your area",
          "Place order and track it live until arrival",
        ],
      },
    ],
    crossRole: [
      { role: "Supplier", touchpoint: "Reviews your order before fulfillment" },
      { role: "Driver", touchpoint: "You see their live location on your tracking screen" },
    ],
    capabilityLinks: ["payments", "telemetry"],
  },
  payload: {
    slug: "payload",
    headline: "Gate control that keeps every load accountable",
    summary:
      "Loading gate operations — scan, load, seal, and release — so nothing leaves the yard unverified.",
    persona: "Gate operator or loading bay supervisor",
    painPoints: [
      "Trucks leaving with incomplete or wrong loads",
      "No clear handoff between loading and driver departure",
      "Exceptions handled on paper with no audit trail",
    ],
    outcomes: [
      "Every manifest scanned and verified before seal",
      "Drivers blocked until gate confirms the load",
      "Exception handling with a clear record",
    ],
    bullets: [
      "Per-truck manifest: load → scan → seal",
      "Batch seal releases drivers for departure",
      "Exception and reassignment handled in-app",
    ],
    workflows: [
      {
        title: "Seal and release",
        steps: [
          "Load orders onto the assigned truck",
          "Scan each item against the manifest",
          "Seal when complete — driver route activates",
        ],
      },
    ],
    crossRole: [
      { role: "Warehouse", touchpoint: "Creates the draft manifest you seal" },
      { role: "Driver", touchpoint: "Can't depart until you confirm the seal" },
    ],
    capabilityLinks: ["dispatch", "realtime"],
  },
};

export const rolesParadeContent = ROLES.map((role) => ({
  ...role,
  bullets: roleContent[role.slug].bullets.slice(0, 2),
}));
