export type CustomerStory = {
  slug: string;
  company: string;
  industry: string;
  headline: string;
  summary: string;
  challenge: string;
  result: string;
  metrics: { label: string; value: string }[];
  quote: { text: string; author: string; role: string };
};

export const customerStories: CustomerStory[] = [
  {
    slug: "northline-distribution",
    company: "Northline Distribution",
    industry: "Food & beverage wholesale",
    headline: "Cut morning dispatch time in half",
    summary:
      "A regional distributor with 40 trucks replaced radio dispatch and spreadsheets with Pegasus warehouse boards.",
    challenge:
      "Peak dispatch happened in a 90-minute window. Misloads and missed orders were costing them retailer relationships.",
    result:
      "Warehouse teams now confirm loads on a visual board. Drivers leave with sealed manifests. Retailers track deliveries without calling in.",
    metrics: [
      { label: "Dispatch window", value: "45 min avg." },
      { label: "Misload rate", value: "Down 68%" },
      { label: "Retailer calls", value: "Down 40%" },
    ],
    quote: {
      text: "Our warehouse lead used to run dispatch from memory and a whiteboard. Now the whole team sees the same board — and trucks leave on time.",
      author: "Elena Vasquez",
      role: "Head of Operations",
    },
  },
  {
    slug: "meridian-building-supply",
    company: "Meridian Building Supply",
    industry: "Construction materials",
    headline: "One map for 120 daily deliveries",
    summary:
      "A multi-warehouse supplier needed live fleet visibility across urban and suburban routes.",
    challenge:
      "Ops spent hours each day calling drivers for location updates. Retailers complained they couldn't track heavy-material deliveries.",
    result:
      "Live fleet map with planned-vs-actual routes. Retailers self-serve tracking. Ops gets deviation alerts before customers do.",
    metrics: [
      { label: "Daily deliveries", value: "120+" },
      { label: "Support calls", value: "Down 35%" },
      { label: "On-time rate", value: "94%" },
    ],
    quote: {
      text: "We stopped being a call center for 'where's my order?' Retailers check tracking themselves. Our ops team handles exceptions, not status checks.",
      author: "James Okonkwo",
      role: "Logistics Director",
    },
  },
  {
    slug: "pacific-cold-chain",
    company: "Pacific Cold Chain",
    industry: "Temperature-controlled logistics",
    headline: "Payments and delivery proof in one flow",
    summary:
      "A cold-chain operator running cash-on-delivery needed reconciliation that matched what drivers collected.",
    challenge:
      "End-of-day cash reconciliation took hours. Disputes with retailers over payment status were common.",
    result:
      "Drivers confirm payment at each stop. Treasury sees collections in real time. Disputes resolved with delivery proof attached.",
    metrics: [
      { label: "Reconciliation time", value: "Same day" },
      { label: "Payment disputes", value: "Down 52%" },
      { label: "COD stops / day", value: "800+" },
    ],
    quote: {
      text: "Cash-on-delivery used to mean end-of-day headaches. Now finance closes the day knowing exactly what each driver collected.",
      author: "Priya Nair",
      role: "Finance Controller",
    },
  },
];

export const customersPageContent = {
  headline: "Teams running real operations on Pegasus",
  summary:
    "From regional distributors to national supplier networks — operators use Pegasus to dispatch faster, track deliveries live, and close the books with confidence.",
  cta: { label: "Talk to our team", href: "/contact" as const },
};
