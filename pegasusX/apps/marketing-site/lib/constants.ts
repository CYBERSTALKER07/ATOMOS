export const LANDING_SECTIONS = [
  { id: "hero", label: "Overview" },
  { id: "experience", label: "Get started" },
  { id: "control-plane", label: "Platform" },
  { id: "ecosystem", label: "Your network" },
  { id: "six-roles", label: "Who it's for" },
  { id: "dispatch-engine", label: "Dispatch" },
  { id: "live-telemetry", label: "Tracking" },
  { id: "financial-integrity", label: "Payments" },
  { id: "reliability", label: "Trust" },
  { id: "component-system", label: "Solutions" },
  { id: "cta", label: "Contact" },
] as const;

export type LandingSectionId = (typeof LANDING_SECTIONS)[number]["id"];

export const ROLES = [
  {
    slug: "supplier",
    name: "Supplier",
    tagline: "Run your network from one dashboard",
    surfaces: ["Web portal", "Desktop app", "Mobile"],
  },
  {
    slug: "warehouse",
    name: "Warehouse",
    tagline: "Dispatch trucks and manage stock with confidence",
    surfaces: ["Web portal", "Mobile"],
  },
  {
    slug: "factory",
    name: "Factory",
    tagline: "Keep production and loading lanes in sync",
    surfaces: ["Web portal", "Mobile"],
  },
  {
    slug: "driver",
    name: "Driver",
    tagline: "Clear routes, simple stops, on-time delivery",
    surfaces: ["Mobile app"],
  },
  {
    slug: "retailer",
    name: "Retailer",
    tagline: "Order, pay, and track deliveries in one place",
    surfaces: ["Desktop app", "Mobile"],
  },
  {
    slug: "payload",
    name: "Payload",
    tagline: "Gate control that keeps every load accountable",
    surfaces: ["Terminal", "Mobile"],
  },
] as const;

export type RoleSlug = (typeof ROLES)[number]["slug"];

export const ORDER_LIFECYCLE = [
  "Order placed",
  "Loaded",
  "In transit",
  "Arrived",
  "Completed",
] as const;

export const CAPABILITIES = [
  { slug: "dispatch", title: "Smarter dispatch" },
  { slug: "outbox", title: "Reliable updates" },
  { slug: "payments", title: "Payment confidence" },
  { slug: "telemetry", title: "Live fleet tracking" },
  { slug: "realtime", title: "Instant coordination" },
  { slug: "topology", title: "One connected network" },
] as const;

export type CapabilitySlug = (typeof CAPABILITIES)[number]["slug"];

export const SOLUTIONS = [
  {
    slug: "dispatch-accuracy",
    title: "Dispatch the right load, every time",
    summary: "Match orders to trucks without spreadsheets or guesswork.",
  },
  {
    slug: "fleet-visibility",
    title: "See your fleet as it moves",
    summary: "Live location and route progress for every delivery on the road.",
  },
  {
    slug: "payment-confidence",
    title: "Close the books without surprises",
    summary: "Payments, cash collection, and reconciliation that stay in sync.",
  },
  {
    slug: "network-coordination",
    title: "Keep every partner aligned",
    summary: "Suppliers, warehouses, factories, and retailers on the same page.",
  },
] as const;

export type SolutionSlug = (typeof SOLUTIONS)[number]["slug"];

export const SITE_NAME = "Pegasus";
export const PLATFORM_BRAND = "The ATOMOS Control Plane";
export const SITE_TAGLINE =
  "The logistics operating system for teams that move physical goods.";

export const ASSET_SLOTS = {
  hero: "/models/object-a-hero.glb",
  controlPlane: "/models/object-b-layers.glb",
  roles: "/models/object-c-roles.glb",
  heroVideo: "/video/hero-loop.mp4",
  ctaVideo: "/video/cta-horizon.mp4",
} as const;
