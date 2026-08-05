export type FleekTickerItem = {
  text: string;
  href?: string;
};

export type FleekStat = {
  label: string;
  value: string;
};

export type FleekBlobStat = {
  value: string;
  label: string;
  highlight?: boolean;
};

export type FleekImpactMetric = {
  client: string;
  title: string;
  description: string;
  value: number;
  unit?: string;
};

export const DEFAULT_TICKER: FleekTickerItem[] = [
  { text: 'LIVE DISPATCH BOARDS' },
  { text: 'SPANNER SYSTEM OF RECORD' },
  { text: 'SIX ROLE SURFACES' },
  { text: 'REQUEST DEMO' },
];

export const HUB_TICKERS: Record<string, FleekTickerItem[]> = {
  platform: [
    { text: 'ORDER LIFECYCLE' },
    { text: 'OUTBOX FANOUT' },
    { text: 'ONE CONTROL PLANE' },
  ],
  technology: [
    { text: 'SPANNER + KAFKA + WS' },
    { text: 'MUTATING HANDLERS' },
    { text: 'CACHE INVALIDATION' },
  ],
  operations: [
    { text: 'VISUAL DISPATCH' },
    { text: 'SMART FIT OVERFLOW' },
    { text: 'GATE SEAL WORKFLOW' },
  ],
  capabilities: [
    { text: 'SMARTER DISPATCH' },
    { text: 'LIVE FLEET MAP' },
    { text: 'PAY AT DELIVERY' },
  ],
  'ai-vision': [
    { text: 'GOVERNED AI ASSIST' },
    { text: 'DETERMINISTIC FALLBACK' },
    { text: 'HUMAN IN THE LOOP' },
  ],
  'apps-deploy': [
    { text: 'PORTAL + MOBILE + DESKTOP' },
    { text: 'ROLE ROW PARITY' },
    { text: 'SILENT WS REFRESH' },
  ],
  roles: [
    { text: 'SUPPLIER · WAREHOUSE · DRIVER' },
    { text: 'RETAILER · FACTORY · GATE' },
    { text: 'ONE ORDER TRUTH' },
  ],
};

export const DEFAULT_AXIOM_STATS: FleekStat[] = [
  { label: 'CONNECTED ROLES', value: '6_' },
  { label: 'INTENTS PROCESSED', value: '2.4M_' },
  { label: 'NETWORK SITES', value: 'Live_' },
];

export const DEFAULT_BLOB_STATS: FleekBlobStat[] = [
  { value: '12+', label: 'dispatch surfaces including portal and native apps' },
  { value: '100+', label: 'route registrations mapped per role row', highlight: true },
  { value: '200+', label: 'API endpoints across supplier-led logistics' },
];

export const DEFAULT_IMPACT_METRIC: FleekImpactMetric = {
  client: 'NOVA',
  title: 'Dispatch fill rate',
  description: 'Performance score for morning load commits across eligible trucks.',
  value: 72,
  unit: '%',
};

export function getTickerForHub(hubId?: string): FleekTickerItem[] {
  if (hubId && HUB_TICKERS[hubId]) return HUB_TICKERS[hubId];
  return DEFAULT_TICKER;
}

export const FLEEK_STACK_FEATURES = [
  'HIGH PERFORMANCE',
  'LOW LATENCY',
  'GEO-AWARE',
  'HUMAN IN THE LOOP',
  'IDEMPOTENT WRITES',
  'SILENT WS REFRESH',
] as const;
