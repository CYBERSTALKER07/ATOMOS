import type { Metadata } from 'next';
import { OG_IMAGE } from '@/app/lib/siteAssets';

export const SITE_NAME = 'Pegasus';
export const SITE_URL = (process.env.NEXT_PUBLIC_SITE_URL ?? 'https://pegasus.io').replace(
  /\/$/,
  ''
);

const DEFAULT_DESCRIPTION =
  'Pegasus is the logistics operating system for supplier-led networks — dispatch, fleet tracking, payments, and realtime coordination across six roles.';

export function absoluteUrl(path = ''): string {
  if (!path || path === '/') return SITE_URL;
  return `${SITE_URL}${path.startsWith('/') ? path : `/${path}`}`;
}

export function absoluteAsset(assetPath: string): string {
  return absoluteUrl(assetPath.startsWith('/') ? assetPath : `/${assetPath}`);
}

type PageMetadataInput = {
  title: string;
  description?: string;
  path?: string;
  image?: string;
  imageAlt?: string;
  noIndex?: boolean;
};

/** Per-page metadata with canonical, Open Graph, and Twitter cards. */
export function pageMetadata({
  title,
  description = DEFAULT_DESCRIPTION,
  path = '',
  image = OG_IMAGE,
  imageAlt = 'Pegasus logistics platform — container crane illustration',
  noIndex = false,
}: PageMetadataInput): Metadata {
  const canonical = absoluteUrl(path);
  const fullTitle = title === 'Home' ? `${SITE_NAME} | Logistics Operating System` : title;

  return {
    title,
    description,
    alternates: { canonical },
    openGraph: {
      type: 'website',
      locale: 'en_US',
      url: canonical,
      siteName: SITE_NAME,
      title: fullTitle,
      description,
      images: [
        {
          url: absoluteAsset(image),
          width: 1200,
          height: 630,
          alt: imageAlt,
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: fullTitle,
      description,
      images: [absoluteAsset(image)],
    },
    ...(noIndex
      ? { robots: { index: false, follow: false } }
      : {}),
  };
}

export function organizationJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: SITE_NAME,
    description: DEFAULT_DESCRIPTION,
    url: SITE_URL,
    logo: absoluteAsset('/pegasus.jpg'),
    image: absoluteAsset(OG_IMAGE),
    sameAs: ['https://linkedin.com/company/pegasus'],
    knowsAbout: [
      'Logistics Software',
      'Fleet Management',
      'Dispatch Operations',
      'Supply Chain',
      'Payment Reconciliation',
      'Warehouse Management',
      'Last-Mile Delivery',
    ],
  };
}

export function websiteJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: SITE_NAME,
    description: DEFAULT_DESCRIPTION,
    url: SITE_URL,
    inLanguage: 'en-US',
    publisher: {
      '@type': 'Organization',
      name: SITE_NAME,
      logo: absoluteAsset('/pegasus.jpg'),
    },
  };
}

export function softwareApplicationJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: SITE_NAME,
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web, Windows, macOS, Android, iOS',
    description:
      'Supplier-led logistics operating system with dispatch boards, fleet live maps, treasury reconciliation, and role-specific apps for warehouse, retailer, driver, factory, and gate teams.',
    offers: {
      '@type': 'Offer',
      price: '0',
      priceCurrency: 'USD',
      description: 'Contact for enterprise licensing',
    },
    url: SITE_URL,
  };
}

export function breadcrumbJsonLd(items: { name: string; path: string }[]) {
  return {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: items.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: item.name,
      item: absoluteUrl(item.path),
    })),
  };
}

export function jsonLdScript(data: Record<string, unknown>) {
  return { __html: JSON.stringify(data) };
}

export function jsonLdGraphScript(nodes: Record<string, unknown>[]) {
  return {
    __html: JSON.stringify({
      '@context': 'https://schema.org',
      '@graph': nodes.map(({ '@context': _c, ...node }) => node),
    }),
  };
}
