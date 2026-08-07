export type PricingMonth = {
  label: string;
  value: number;
  highlight?: boolean;
};

export type ShippingDestination = {
  from: string;
  to: string;
  price: number;
};

export type LogisticsCityAnalytics = {
  id: string;
  name: string;
  pricingTrends: PricingMonth[];
  destinations: ShippingDestination[];
};

export const LOGISTICS_CITIES: LogisticsCityAnalytics[] = [
  {
    id: 'new-york',
    name: 'New York',
    pricingTrends: [
      { label: 'Feb 2025', value: 11266 },
      { label: 'Mar 2025', value: 11567 },
      { label: 'Apr 2025', value: 9023 },
      { label: 'May 2025', value: 16356 },
      { label: 'Jun 2025', value: 17890 },
      { label: 'Jul 2025', value: 24667, highlight: true },
      { label: 'Aug 2025', value: 10334 },
      { label: 'Sep 2025', value: 11567 },
      { label: 'Oct 2025', value: 23456 },
      { label: 'Nov 2025', value: 42341 },
      { label: 'Dec 2025', value: 47857 },
      { label: 'Jan 2026', value: 14568 },
    ],
    destinations: [
      { from: 'Qingdao', to: 'Havana', price: 4500 },
      { from: 'Mumbai', to: 'New York', price: 3750 },
      { from: 'Mexico City', to: 'New York', price: 1800 },
      { from: 'Vancouver', to: 'Yokohama', price: 1320 },
      { from: 'Busan', to: 'London', price: 3400 },
      { from: 'Istanbul', to: 'Cairo', price: 3050 },
    ],
  },
  {
    id: 'london',
    name: 'London',
    pricingTrends: [
      { label: 'Feb 2025', value: 9840 },
      { label: 'Mar 2025', value: 10220 },
      { label: 'Apr 2025', value: 8650 },
      { label: 'May 2025', value: 14200 },
      { label: 'Jun 2025', value: 15680 },
      { label: 'Jul 2025', value: 22100, highlight: true },
      { label: 'Aug 2025', value: 9900 },
      { label: 'Sep 2025', value: 10840 },
      { label: 'Oct 2025', value: 19800 },
      { label: 'Nov 2025', value: 38900 },
      { label: 'Dec 2025', value: 44200 },
      { label: 'Jan 2026', value: 13200 },
    ],
    destinations: [
      { from: 'Shanghai', to: 'London', price: 2850 },
      { from: 'Rotterdam', to: 'London', price: 980 },
      { from: 'Dubai', to: 'London', price: 2100 },
      { from: 'Singapore', to: 'London', price: 3200 },
      { from: 'Hamburg', to: 'London', price: 1150 },
      { from: 'Casablanca', to: 'London', price: 1680 },
    ],
  },
  {
    id: 'shanghai',
    name: 'Shanghai',
    pricingTrends: [
      { label: 'Feb 2025', value: 8900 },
      { label: 'Mar 2025', value: 9400 },
      { label: 'Apr 2025', value: 7800 },
      { label: 'May 2025', value: 12800 },
      { label: 'Jun 2025', value: 14200 },
      { label: 'Jul 2025', value: 19800, highlight: true },
      { label: 'Aug 2025', value: 8600 },
      { label: 'Sep 2025', value: 9200 },
      { label: 'Oct 2025', value: 17600 },
      { label: 'Nov 2025', value: 35200 },
      { label: 'Dec 2025', value: 40100 },
      { label: 'Jan 2026', value: 11800 },
    ],
    destinations: [
      { from: 'Shanghai', to: 'Los Angeles', price: 1950 },
      { from: 'Shanghai', to: 'Singapore', price: 680 },
      { from: 'Shanghai', to: 'Rotterdam', price: 2400 },
      { from: 'Ningbo', to: 'Shanghai', price: 420 },
      { from: 'Shanghai', to: 'Tokyo', price: 890 },
      { from: 'Shanghai', to: 'Sydney', price: 1650 },
    ],
  },
];

export function getCityAnalytics(cityId?: string): LogisticsCityAnalytics {
  const found = LOGISTICS_CITIES.find((c) => c.id === cityId);
  return found ?? LOGISTICS_CITIES[0];
}

const MONTH_RU: Record<string, string> = {
  Feb: 'Фев',
  Mar: 'Мар',
  Apr: 'Апр',
  May: 'Май',
  Jun: 'Июн',
  Jul: 'Июл',
  Aug: 'Авг',
  Sep: 'Сен',
  Oct: 'Окт',
  Nov: 'Ноя',
  Dec: 'Дек',
  Jan: 'Янв',
};

export function localizeMonthLabel(label: string, lang: 'en' | 'ru' = 'en'): string {
  if (lang !== 'ru') return label;
  const [mon, year] = label.split(' ');
  return `${MONTH_RU[mon] ?? mon} ${year ?? ''}`.trim();
}

export function formatUsd(value: number, lang: 'en' | 'ru' = 'en'): string {
  return value.toLocaleString(lang === 'ru' ? 'ru-RU' : 'en-US');
}

export function getLocalizedCities(lang: 'en' | 'ru' = 'en'): LogisticsCityAnalytics[] {
  if (lang !== 'ru') return LOGISTICS_CITIES;
  const names: Record<string, string> = {
    'new-york': 'Нью-Йорк',
    london: 'Лондон',
    shanghai: 'Шанхай',
  };
  return LOGISTICS_CITIES.map((city) => ({
    ...city,
    name: names[city.id] ?? city.name,
    pricingTrends: city.pricingTrends.map((m) => ({
      ...m,
      label: localizeMonthLabel(m.label, 'ru'),
    })),
  }));
}
