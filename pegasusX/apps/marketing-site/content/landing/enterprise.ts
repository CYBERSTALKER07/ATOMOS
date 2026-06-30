export const PLATFORM_BRAND = "The ATOMOS Control Plane";

export const customerStrip = {
  headline: "Built for operators who can't afford downtime",
  subline:
    "From regional distributors to national supplier networks — teams use Pegasus to run dispatch, warehouses, and last-mile delivery every day.",
  labels: [
    "Regional distributors",
    "Food & beverage suppliers",
    "Building materials networks",
    "Consumer goods wholesalers",
    "Cold chain operators",
    "Multi-warehouse fleets",
    "Factory-to-retail supply",
    "Cash-on-delivery markets",
    "Scheduled delivery programs",
    "High-volume dispatch windows",
  ],
};

export const platformThesis = {
  eyebrow: PLATFORM_BRAND,
  headline: "One platform. Six roles. Zero blind spots.",
  body:
    "Stop chasing updates across phone calls, spreadsheets, and separate apps. Pegasus gives every team in your network the same live picture — from the moment a retailer places an order to the moment it arrives at the door.",
};

export const experiencePaths = [
  {
    id: "tour",
    title: "Explore the platform",
    description:
      "Walk through how orders, dispatch, tracking, and payments connect — in plain language, with real workflows.",
    cta: "See how it works",
    href: "/platform" as const,
  },
  {
    id: "demo",
    title: "Talk to our team",
    description:
      "Get a guided walkthrough tailored to your network — dispatch, warehouses, drivers, or the full picture.",
    cta: "Request a demo",
    href: "/contact" as const,
  },
] as const;

export const trustProof = {
  tag: "Why teams trust Pegasus",
  title: "Built for accuracy at scale",
  summary:
    "Every app in your network reads from the same source of truth — so a warehouse dispatch, a driver update, and a retailer tracking screen always agree.",
  highlights: [
    { key: "Every role covered", value: "Dedicated apps for suppliers, warehouses, factories, drivers, retailers, and gate teams" },
    { key: "Updates you can trust", value: "Status changes and notifications happen together — no mismatched data" },
    { key: "Payment safety", value: "Duplicate charges blocked; every transaction traceable" },
    { key: "Live coordination", value: "Dispatch boards and fleet maps refresh as work happens" },
  ],
  cta: { label: "Read about reliability", href: "/#reliability" as const },
};
