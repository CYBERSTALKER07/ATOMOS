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

const DEFAULT_STATS_RU: O9ValueStat[] = [
  {
    value: '60%',
    label: 'Снижение дефицита',
    context: 'Точный учет запасов в реальном времени на узлах поставщика и склада.',
  },
  {
    value: '+53%',
    label: 'Своевременность доставки',
    context: 'Координация диспетчеризации и автопарка на единой истине заказа.',
  },
  {
    value: '6',
    label: 'Ролей подключено',
    context: 'Поставщик, склад, фабрика, водитель, ритейлер и КПП в единой записи.',
  },
  {
    value: '<2s',
    label: 'Живые обновления',
    context: 'Доски и карты синхронизируются автоматически без ручного обновления.',
  },
];

const HUB_TABS_EN: Record<string, O9ValueTab[]> = {
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

const HUB_TABS_RU: Record<string, O9ValueTab[]> = {
  platform: [
    { id: 'network', label: 'Сеть', stats: DEFAULT_STATS_RU },
    { id: 'reliability', label: 'Надежность', stats: DEFAULT_STATS_RU },
  ],
  capabilities: [
    { id: 'dispatch', label: 'Диспетчеризация', stats: DEFAULT_STATS_RU },
    { id: 'treasury', label: 'Казначейство', stats: DEFAULT_STATS_RU },
  ],
  operations: [
    { id: 'warehouse', label: 'Склад', stats: DEFAULT_STATS_RU },
    { id: 'fleet', label: 'Автопарк', stats: DEFAULT_STATS_RU },
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

export const DEFAULT_TESTIMONIALS_RU: O9Testimonial[] = [
  {
    company: 'Региональная сеть поставщиков',
    quote:
      'Pegasus дал нам единую диспетчерскую доску и единую истину по платежам — мы перестали сверять три таблицы каждое утро.',
    name: 'Руководитель операций',
    title: 'Панель управления поставщика',
  },
  {
    company: 'Мульти-складской комплекс',
    quote:
      'От пломбы на КПП до отслеживания ритейлером — всё на одном ID заказа. Водители и админы склада наконец видят один статус.',
    name: 'Администратор склада',
    title: 'Диспетчеризация и автопарк',
  },
  {
    company: 'Сеть ритейла',
    quote:
      'Ответ при закрытом магазине и оплата при доставке работают одинаково на десктопе и в мобильном — никаких дублирующих процессов.',
    name: 'Операции ритейла',
    title: 'Доставка последней мили',
  },
];

export function getTestimonials(lang = 'en'): O9Testimonial[] {
  return lang === 'ru' ? DEFAULT_TESTIMONIALS_RU : DEFAULT_TESTIMONIALS;
}

export function getBusinessValueTabs(hubId?: string, outcomes?: string[], lang = 'en'): O9ValueTab[] {
  const tabsMap = lang === 'ru' ? HUB_TABS_RU : HUB_TABS_EN;
  const defaultStats = lang === 'ru' ? DEFAULT_STATS_RU : DEFAULT_STATS;

  if (hubId && tabsMap[hubId]) return tabsMap[hubId];

  if (outcomes && outcomes.length >= 4) {
    return [
      {
        id: 'outcomes',
        label: lang === 'ru' ? 'Результаты' : 'Outcomes',
        stats: outcomes.slice(0, 4).map((o, i) => ({
          value: `${i + 1}`,
          label: lang === 'ru' ? 'Ключевой результат' : 'Key outcome',
          context: o,
        })),
      },
    ];
  }

  return [{ id: 'default', label: lang === 'ru' ? 'Сеть' : 'Network', stats: defaultStats }];
}
