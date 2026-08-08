import type { Metadata } from 'next';
import { BRAND_LOGO, OG_IMAGE } from '@/app/lib/siteAssets';
import type { Language } from '@/app/lib/i18n/translations';

export const SITE_NAME = 'Pegasus';

function resolveSiteUrl(): string {
  const candidate =
    process.env.NEXT_PUBLIC_SITE_URL ??
    process.env.URL ??
    process.env.DEPLOY_PRIME_URL ??
    'https://pegasus.io';
  return candidate.replace(/\/$/, '');
}

export const SITE_URL = resolveSiteUrl();

const DEFAULT_DESCRIPTION =
  'Pegasus is the logistics operating system for supplier-led networks — dispatch, fleet tracking, payments, and realtime coordination across six roles.';

export function absoluteUrl(path = ''): string {
  if (!path || path === '/') return SITE_URL;
  return `${SITE_URL}${path.startsWith('/') ? path : `/${path}`}`;
}

export function absoluteAsset(assetPath: string): string {
  return absoluteUrl(assetPath.startsWith('/') ? assetPath : `/${assetPath}`);
}

/** Language-variant URLs for hreflang (cookie + ?lang=). */
export function languageAlternates(path = ''): Record<string, string> {
  const base = absoluteUrl(path);
  const join = base.includes('?') ? '&' : '?';
  return {
    en: `${base}${join}lang=en`,
    ru: `${base}${join}lang=ru`,
    'x-default': `${base}${join}lang=en`,
  };
}

type PageMetadataInput = {
  title: string;
  description?: string;
  path?: string;
  image?: string;
  imageAlt?: string;
  noIndex?: boolean;
  language?: Language;
};

/** Per-page metadata with canonical, hreflang, Open Graph, and Twitter cards. */
export function pageMetadata({
  title,
  description = DEFAULT_DESCRIPTION,
  path = '',
  image = OG_IMAGE,
  imageAlt = 'Pegasus — logistics operating system',
  noIndex = false,
  language = 'en',
}: PageMetadataInput): Metadata {
  const canonical = absoluteUrl(path);
  const fullTitle =
    title === 'Home' || title === 'Главная'
      ? `${SITE_NAME} | Logistics Operating System`
      : title;
  const locale = language === 'ru' ? 'ru_RU' : 'en_US';
  const localeAlternate = language === 'ru' ? 'en_US' : 'ru_RU';
  const ogImage = absoluteAsset(image);
  const isDefaultOg = image === OG_IMAGE;

  return {
    title,
    description,
    alternates: {
      canonical,
      languages: languageAlternates(path),
    },
    openGraph: {
      type: 'website',
      locale,
      alternateLocale: [localeAlternate],
      url: canonical,
      siteName: SITE_NAME,
      title: fullTitle,
      description,
      images: [
        {
          url: isDefaultOg ? absoluteUrl('/opengraph-image') : ogImage,
          width: isDefaultOg ? 1200 : 512,
          height: isDefaultOg ? 630 : 512,
          alt: imageAlt,
          type: isDefaultOg ? 'image/png' : 'image/jpeg',
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: fullTitle,
      description,
      images: [isDefaultOg ? absoluteUrl('/opengraph-image') : ogImage],
    },
    ...(noIndex
      ? { robots: { index: false, follow: false, nocache: true } }
      : {
          robots: {
            index: true,
            follow: true,
            'max-image-preview': 'large' as const,
            'max-snippet': -1,
            'max-video-preview': -1,
          },
        }),
  };
}

export function organizationJsonLd(language: Language = 'en') {
  const description =
    language === 'ru'
      ? 'Pegasus — операционная система логистики для сетей под управлением поставщика: диспетчеризация, мониторинг автопарка, платежи и реалтайм-координация шести ролей.'
      : DEFAULT_DESCRIPTION;

  return {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    '@id': `${SITE_URL}/#organization`,
    name: SITE_NAME,
    legalName: SITE_NAME,
    description,
    url: SITE_URL,
    logo: {
      '@type': 'ImageObject',
      url: absoluteAsset('/pegasus-icon-512.png'),
      width: 512,
      height: 512,
    },
    image: absoluteAsset(BRAND_LOGO),
    sameAs: ['https://t.me/DominusMunerum'],
    contactPoint: [
      {
        '@type': 'ContactPoint',
        contactType: 'sales',
        email: 'cyberstalkerx7@gmail.com',
        url: absoluteUrl('/contact'),
        availableLanguage: ['English', 'Russian'],
      },
    ],
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

export function websiteJsonLd(language: Language = 'en') {
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    '@id': `${SITE_URL}/#website`,
    name: SITE_NAME,
    description: DEFAULT_DESCRIPTION,
    url: SITE_URL,
    inLanguage: language === 'ru' ? ['ru-RU', 'en-US'] : ['en-US', 'ru-RU'],
    publisher: { '@id': `${SITE_URL}/#organization` },
    potentialAction: {
      '@type': 'CommunicateAction',
      target: absoluteUrl('/contact'),
      name: language === 'ru' ? 'Связаться с Pegasus' : 'Contact Pegasus',
    },
  };
}

export function softwareApplicationJsonLd(language: Language = 'en') {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    '@id': `${SITE_URL}/#software`,
    name: SITE_NAME,
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web, Windows, macOS, Android, iOS',
    inLanguage: language === 'ru' ? 'ru' : 'en',
    description:
      language === 'ru'
        ? 'Операционная система логистики под управлением поставщика: диспетчерские доски, карты автопарка, сверка казначейства и приложения для склада, ритейлера, водителя, завода и ворот.'
        : 'Supplier-led logistics operating system with dispatch boards, fleet live maps, treasury reconciliation, and role-specific apps for warehouse, retailer, driver, factory, and gate teams.',
    offers: {
      '@type': 'Offer',
      price: '0',
      priceCurrency: 'USD',
      description:
        language === 'ru'
          ? 'Свяжитесь для enterprise-лицензирования'
          : 'Contact for enterprise licensing',
      url: absoluteUrl('/join'),
    },
    url: SITE_URL,
    provider: { '@id': `${SITE_URL}/#organization` },
  };
}

export function contactPageJsonLd(language: Language = 'en') {
  return {
    '@context': 'https://schema.org',
    '@type': 'ContactPage',
    '@id': `${SITE_URL}/contact#webpage`,
    name: language === 'ru' ? 'Контакты Pegasus' : 'Contact Pegasus',
    description:
      language === 'ru'
        ? 'Свяжитесь с Pegasus для демо, вопросов по развёртыванию или enterprise-логистике.'
        : 'Contact Pegasus for a live demo, deployment questions, or enterprise logistics inquiries.',
    url: absoluteUrl('/contact'),
    isPartOf: { '@id': `${SITE_URL}/#website` },
    about: { '@id': `${SITE_URL}/#organization` },
    inLanguage: language === 'ru' ? 'ru-RU' : 'en-US',
  };
}

export function faqPageJsonLd(
  faqs: { question: string; answer: string }[],
  language: Language = 'en'
) {
  return {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    inLanguage: language === 'ru' ? 'ru-RU' : 'en-US',
    mainEntity: faqs.map((faq) => ({
      '@type': 'Question',
      name: faq.question,
      acceptedAnswer: {
        '@type': 'Answer',
        text: faq.answer,
      },
    })),
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
