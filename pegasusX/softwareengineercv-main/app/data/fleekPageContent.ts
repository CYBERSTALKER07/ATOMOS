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
  { text: 'ONE SOURCE OF TRUTH' },
  { text: 'SIX ROLE SURFACES' },
  { text: 'REQUEST DEMO' },
];

export const DEFAULT_TICKER_RU: FleekTickerItem[] = [
  { text: 'ЖИВЫЕ ДОСКИ ДИСПЕТЧЕРИЗАЦИИ' },
  { text: 'ОДИН ИСТОЧНИК ПРАВДЫ' },
  { text: 'ШЕСТЬ РОЛЕВЫХ ПОВЕРХНОСТЕЙ' },
  { text: 'ЗАПРОСИТЬ ДЕМО' },
];

export const HUB_TICKERS: Record<string, FleekTickerItem[]> = {
  platform: [
    { text: 'ORDER LIFECYCLE' },
    { text: 'LIVE ROLE UPDATES' },
    { text: 'ONE CONTROL PLANE' },
  ],
  technology: [
    { text: 'LIVE SYNC ACROSS APPS' },
    { text: 'GUARDED UPDATES' },
    { text: 'ALWAYS-FRESH SCREENS' },
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
    { text: 'PROVEN FALLBACK RULES' },
    { text: 'HUMAN IN THE LOOP' },
  ],
  'apps-deploy': [
    { text: 'PORTAL + MOBILE + DESKTOP' },
    { text: 'SAME FEATURES EVERYWHERE' },
    { text: 'AUTO SCREEN REFRESH' },
  ],
  roles: [
    { text: 'SUPPLIER · WAREHOUSE · DRIVER' },
    { text: 'RETAILER · FACTORY · GATE' },
    { text: 'ONE ORDER TRUTH' },
  ],
};

export const HUB_TICKERS_RU: Record<string, FleekTickerItem[]> = {
  platform: [
    { text: 'ЖИЗНЕННЫЙ ЦИКЛ ЗАКАЗА' },
    { text: 'ЖИВЫЕ ОБНОВЛЕНИЯ РОЛЕЙ' },
    { text: 'ЕДИНАЯ ПАНЕЛЬ УПРАВЛЕНИЯ' },
  ],
  technology: [
    { text: 'ЖИВАЯ СИНХРОНИЗАЦИЯ ПРИЛОЖЕНИЙ' },
    { text: 'ЗАЩИЩЁННЫЕ ОБНОВЛЕНИЯ' },
    { text: 'ВСЕГДА АКТУАЛЬНЫЕ ЭКРАНЫ' },
  ],
  operations: [
    { text: 'ВИЗУАЛЬНАЯ ДИСПЕТЧЕРИЗАЦИЯ' },
    { text: 'УМНЫЙ ПЕРЕПОЛНЕНИЕ FIT' },
    { text: 'ПЛОМБИРОВАНИЕ НА ВОРОТАХ' },
  ],
  capabilities: [
    { text: 'УМНАЯ ДИСПЕТЧЕРИЗАЦИЯ' },
    { text: 'ЖИВАЯ КАРТА АВТОПАРКА' },
    { text: 'ОПЛАТА ПРИ ДОСТАВКЕ' },
  ],
  'ai-vision': [
    { text: 'УПРАВЛЯЕМЫЙ ИИ-АССИСТЕНТ' },
    { text: 'ПРОВЕРЕННЫЕ ПРАВИЛА ОТКАТА' },
    { text: 'ЧЕЛОВЕК В КОНТУРЕ' },
  ],
  'apps-deploy': [
    { text: 'ПОРТАЛ + МОБИЛЬНЫЕ + ДЕСКТОП' },
    { text: 'ОДНИ ФУНКЦИИ ВЕЗДЕ' },
    { text: 'АВТООБНОВЛЕНИЕ ЭКРАНА' },
  ],
  roles: [
    { text: 'ПОСТАВЩИК · СКЛАД · ВОДИТЕЛЬ' },
    { text: 'РИТЕЙЛЕР · ЗАВОД · ВОРОТА' },
    { text: 'ОДНА ПРАВДА ЗАКАЗА' },
  ],
};

export const DEFAULT_AXIOM_STATS: FleekStat[] = [
  { label: 'CONNECTED ROLES', value: '6_' },
  { label: 'INTENTS PROCESSED', value: '2.4M_' },
  { label: 'NETWORK SITES', value: 'Live_' },
];

export const DEFAULT_AXIOM_STATS_RU: FleekStat[] = [
  { label: 'СВЯЗАННЫЕ РОЛИ', value: '6_' },
  { label: 'ОБРАБОТАНО ИНТЕНТОВ', value: '2.4M_' },
  { label: 'ПЛОЩАДКИ СЕТИ', value: 'Live_' },
];

export const DEFAULT_BLOB_STATS: FleekBlobStat[] = [
  { value: '12+', label: 'dispatch surfaces including portal and native apps' },
  { value: '100+', label: 'route registrations mapped per role', highlight: true },
  { value: '200+', label: 'workflows across supplier-led logistics' },
];

export const DEFAULT_BLOB_STATS_RU: FleekBlobStat[] = [
  { value: '12+', label: 'поверхностей диспетчеризации, включая портал и нативные приложения' },
  { value: '100+', label: 'регистраций маршрутов, сопоставленных по ролям', highlight: true },
  { value: '200+', label: 'процессов в логистике под управлением поставщика' },
];

export const DEFAULT_IMPACT_METRIC: FleekImpactMetric = {
  client: 'NOVA',
  title: 'Dispatch fill rate',
  description: 'Performance score for morning load commits across eligible trucks.',
  value: 72,
  unit: '%',
};

export const DEFAULT_IMPACT_METRIC_RU: FleekImpactMetric = {
  client: 'NOVA',
  title: 'Заполнение диспетчеризации',
  description: 'Показатель утренних коммитов загрузки по подходящим грузовикам.',
  value: 72,
  unit: '%',
};

export function getTickerForHub(hubId?: string, lang: 'en' | 'ru' = 'en'): FleekTickerItem[] {
  const map = lang === 'ru' ? HUB_TICKERS_RU : HUB_TICKERS;
  const fallback = lang === 'ru' ? DEFAULT_TICKER_RU : DEFAULT_TICKER;
  if (hubId && map[hubId]) return map[hubId];
  return fallback;
}

export const FLEEK_STACK_FEATURES = [
  'HIGH PERFORMANCE',
  'FAST RESPONSE',
  'GEO-AWARE',
  'HUMAN IN THE LOOP',
  'SAFE RETRIES',
  'AUTO SCREEN REFRESH',
] as const;

export const FLEEK_STACK_FEATURES_RU = [
  'ВЫСОКАЯ ПРОИЗВОДИТЕЛЬНОСТЬ',
  'БЫСТРЫЙ ОТКЛИК',
  'ГЕО-ОСВЕДОМЛЁННОСТЬ',
  'ЧЕЛОВЕК В КОНТУРЕ',
  'БЕЗОПАСНЫЕ ПОВТОРЫ',
  'АВТООБНОВЛЕНИЕ ЭКРАНА',
] as const;

export function getAxiomStats(lang: 'en' | 'ru' = 'en'): FleekStat[] {
  return lang === 'ru' ? DEFAULT_AXIOM_STATS_RU : DEFAULT_AXIOM_STATS;
}

export function getBlobStats(lang: 'en' | 'ru' = 'en'): FleekBlobStat[] {
  return lang === 'ru' ? DEFAULT_BLOB_STATS_RU : DEFAULT_BLOB_STATS;
}

export function getImpactMetric(lang: 'en' | 'ru' = 'en'): FleekImpactMetric {
  return lang === 'ru' ? DEFAULT_IMPACT_METRIC_RU : DEFAULT_IMPACT_METRIC;
}

export function getStackFeatures(lang: 'en' | 'ru' = 'en'): string[] {
  return [...(lang === 'ru' ? FLEEK_STACK_FEATURES_RU : FLEEK_STACK_FEATURES)];
}
