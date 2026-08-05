export type O9ValueStat = {
  value: string;
  label: string;
  context?: string;
};

export type O9ValueTab = {
  id: string;
  label: string;
  stats: O9ValueStat[];
};

export type O9Testimonial = {
  company: string;
  quote: string;
  name: string;
  title: string;
};

const DEFAULT_STATS: O9ValueStat[] = [
  {
    value: '60%',
    label: 'Reduction in stock-outs',
    context: 'Maintained accurate, real-time inventory visibility across supplier and warehouse nodes.',
  },
  {
    value: '+53%',
    label: 'On-time delivery',
    context: 'Dispatch and fleet coordination on one order truth — fewer manual handoffs.',
  },
  {
    value: '6',
    label: 'Roles connected',
    context: 'Supplier, warehouse, factory, driver, retailer, and gate on one shared order record.',
  },
  {
    value: '<2s',
    label: 'Live updates',
    context: 'Boards and maps stay aligned automatically — no manual refresh.',
  },
];

const HUB_TABS: Record<string, O9ValueTab[]> = {
  platform: [
    { id: 'network', label: 'Network', stats: DEFAULT_STATS },
    { id: 'reliability', label: 'Reliability', stats: DEFAULT_STATS },
  ],
  capabilities: [
    { id: 'dispatch', label: 'Dispatch', stats: DEFAULT_STATS },
    { id: 'treasury', label: 'Treasury', stats: DEFAULT_STATS },
  ],
  operations: [
    { id: 'warehouse', label: 'Warehouse', stats: DEFAULT_STATS },
    { id: 'fleet', label: 'Fleet', stats: DEFAULT_STATS },
  ],
};

export const DEFAULT_TESTIMONIALS: O9Testimonial[] = [
  {
    company: 'Regional supplier network',
    quote:
      'Pegasus gave us one dispatch board and one payment truth — we stopped reconciling three spreadsheets every morning.',
    name: 'Operations lead',
    title: 'Supplier control plane',
  },
  {
    company: 'Multi-site warehouse',
    quote:
      'Gate seal to retailer tracking stayed on the same order ID. Drivers and warehouse admins finally saw the same state.',
    name: 'Warehouse admin',
    title: 'Dispatch & fleet',
  },
  {
    company: 'Retailer chain',
    quote:
      'Shop-closed respond and pay-at-delivery work the same on desktop and mobile — no duplicate workflows to maintain.',
    name: 'Retail ops',
    title: 'Last-mile delivery',
  },
];

export function getBusinessValueTabs(hubId?: string, outcomes?: string[]): O9ValueTab[] {
  if (hubId && HUB_TABS[hubId]) return HUB_TABS[hubId];

  if (outcomes && outcomes.length >= 4) {
    return [
      {
        id: 'outcomes',
        label: 'Outcomes',
        stats: outcomes.slice(0, 4).map((o, i) => ({
          value: `${i + 1}`,
          label: 'Key outcome',
          context: o,
        })),
      },
    ];
  }

  return [{ id: 'default', label: 'Network', stats: DEFAULT_STATS }];
}
