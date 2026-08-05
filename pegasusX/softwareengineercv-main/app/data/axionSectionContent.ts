import { SITE_IMAGES } from '@/app/lib/siteAssets';

export type AxionSolutionCard = {
  title: string;
  description?: string;
  image: string;
  href: string;
  size?: 'large' | 'small';
};

export type AxionIndustryCard = {
  title: string;
  description?: string;
  icon: 'retail' | 'health' | 'tech' | 'manufacturing' | 'fleet' | 'warehouse';
  href?: string;
  highlight?: boolean;
};

export type AxionTechFeature = {
  title: string;
  description: string;
  href: string;
};

export const DEFAULT_SOLUTIONS: AxionSolutionCard[] = [
  {
    title: 'International Shipping',
    image: SITE_IMAGES.containerShip,
    href: '/operations',
    size: 'large',
  },
  {
    title: 'Warehousing & Distribution',
    image: SITE_IMAGES.warehouseAutomation,
    href: '/roles/warehouse',
    size: 'large',
  },
  {
    title: 'Last-Mile Delivery',
    image: SITE_IMAGES.truckTerminal,
    href: '/capabilities/smarter-dispatch',
    size: 'small',
  },
  {
    title: 'Supply Chain Optimization',
    image: SITE_IMAGES.multimodalHub,
    href: '/platform/how-pegasus-works',
    size: 'small',
  },
  {
    title: 'Customs Clearance',
    image: SITE_IMAGES.operationsTeam,
    href: '/operations',
    size: 'small',
  },
];

export const DEFAULT_INDUSTRIES: AxionIndustryCard[] = [
  { title: 'Retail', icon: 'retail', href: '/roles/retailer' },
  { title: 'Healthcare', icon: 'health', href: '/capabilities' },
  { title: 'Technology', icon: 'tech', href: '/technology' },
  { title: 'Manufacturing', icon: 'manufacturing', href: '/roles/factory', highlight: true },
];

export const DEFAULT_TECH_FEATURES: AxionTechFeature[] = [
  {
    title: 'Real-Time Tracking',
    description: 'Live fleet and order state on one shared order truth — portal, mobile, and gate.',
    href: '/technology/redis-kafka',
  },
  {
    title: 'Data Analytics',
    description: 'Pricing trends, fill rates, and lane performance without spreadsheet drift.',
    href: '/platform/atomos-control-plane',
  },
  {
    title: 'Automated Updates',
    description: 'live sync after every change refresh so every role sees the same status.',
    href: '/platform/reliable-updates',
  },
  {
    title: 'Secure Portal',
    description: 'Claims-scoped APIs and role-ready surfaces for supplier-led networks.',
    href: '/platform/supplier-control-plane',
  },
];

export function mapTopicsToSolutions(
  topics: { label: string; href: string; description?: string }[],
  images: string[]
): AxionSolutionCard[] {
  return topics.slice(0, 5).map((t, i) => ({
    title: t.label,
    description: t.description,
    image: images[i % images.length],
    href: t.href,
    size: i < 2 ? 'large' : 'small',
  }));
}
