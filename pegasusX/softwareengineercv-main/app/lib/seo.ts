import type { Metadata } from 'next';
import { BRAND_LOGO, OG_IMAGE } from '@/app/lib/siteAssets';
import type { Language } from '@/app/lib/i18n/translations';

export const SITE_NAME = 'Pegasus';

function resolveSiteUrl(): string {
  const candidate =
    process.env.NEXT_PUBLIC_SITE_URL ??
    process.env.URL ??
    process.env.DEPLOY_PRIME_URL ??
    'https://xn--pgasus-p3a.org';
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
    es: `${base}${join}lang=es`,
    de: `${base}${join}lang=de`,
    fr: `${base}${join}lang=fr`,
    zh: `${base}${join}lang=zh`,
    ja: `${base}${join}lang=ja`,
    ar: `${base}${join}lang=ar`,
    pt: `${base}${join}lang=pt`,
    tr: `${base}${join}lang=tr`,
    uz: `${base}${join}lang=uz`,
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
    title: typeof title === 'string' && title.includes(SITE_NAME) ? { absolute: title } : title,
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
      ? 'Pegasus — глобальное программное обеспечение для управления цепочками поставок и автоматизации логистики: диспетчеризация, мониторинг автопарка, платежи и реалтайм-координация шести ролей.'
      : 'Pegasus is the global supply chain software and logistics automation platform for supplier-led networks — dispatch, fleet tracking, payments, and coordination across six roles.';

  return {
    '@context': 'https://schema.org',
    '@type': ['Organization', 'Corporation'],
    '@id': `${SITE_URL}/#organization`,
    name: SITE_NAME,
    legalName: 'Pegasus Global Logistics & Supply Chain Technologies',
    alternateName: [
      'Pegasus Global Logistics',
      'Pegasus Supply Chain Software',
      'Pegasus Logistics Automation',
      'Pegasus TMS',
      'Pegasus Logistics Operating System',
    ],
    description,
    url: SITE_URL,
    logo: {
      '@type': 'ImageObject',
      url: absoluteAsset('/pegasus-icon-512.png'),
      width: 512,
      height: 512,
    },
    image: absoluteAsset(BRAND_LOGO),
    sameAs: [
      'https://t.me/DominusMunerum',
      'https://en.wikipedia.org/wiki/Supply_chain_management',
      'https://en.wikipedia.org/wiki/Logistics_automation',
      'https://en.wikipedia.org/wiki/Transportation_management_system',
      'https://en.wikipedia.org/wiki/Logistics',
    ],
    contactPoint: [
      {
        '@type': 'ContactPoint',
        contactType: 'sales',
        email: 'cyberstalkerx7@gmail.com',
        url: absoluteUrl('/contact'),
        availableLanguage: [
          'English',
          'Spanish',
          'German',
          'French',
          'Chinese',
          'Japanese',
          'Arabic',
          'Portuguese',
          'Turkish',
          'Uzbek',
          'Russian',
        ],
      },
    ],
    areaServed: [
      'Worldwide',
      'US',
      'EU',
      'GB',
      'DE',
      'FR',
      'ES',
      'CN',
      'JP',
      'AE',
      'SA',
      'TR',
      'UZ',
      'BR',
      'MX',
      'IN',
      'SG',
      'CA',
      'AU',
    ],
    knowsAbout: [
      'Global Logistics',
      'Supply Chain Software',
      'Logistics Automation',
      'Transportation Management System',
      'Fleet Dispatch Software',
      'Route Optimization Software',
      'Warehouse Automation',
      'Last-Mile Delivery Optimization',
      'B2B Freight Coordination',
      'Payment Reconciliation',
      'Logística Global',
      'Software de Cadena de Suministro',
      'Automatización Logística',
      'Globale Logistik',
      'Supply-Chain-Software',
      'Logistikautomatisierung',
      'Logistique Mondiale',
      'Logiciel Supply Chain',
      '全球物流',
      '供应链软件',
      '物流自动化',
      'グローバル物流',
      'サプライチェーンソフトウェア',
      'الخدمات اللوجستية العالمية',
      'برمجيات سلاسل الإمداد',
      'أتمتة الخدمات اللوجستية',
      'Logística Global',
      'Software de Cadeia de Suprimentos',
      'Global Lojistik',
      'Tedarik Zinciri Yazılımı',
      'Global logistika',
      'Taʼminot zanjiri dasturi',
    ],
  };
}

export function websiteJsonLd(language: Language = 'en') {
  return {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    '@id': `${SITE_URL}/#website`,
    name: 'Pegasus — Global Logistics & Supply Chain Software Automation',
    alternateName: [
      'Pegasus',
      'Pegasus Logistics',
      'Pegasus Global Logistics',
      'Pegasus Supply Chain Software',
      'Pegasus Logistics Automation',
      'Pegasus TMS',
    ],
    description: DEFAULT_DESCRIPTION,
    url: SITE_URL,
    inLanguage: [
      'en',
      'es',
      'de',
      'fr',
      'zh',
      'ja',
      'ar',
      'pt',
      'tr',
      'uz',
      'ru',
    ],
    publisher: { '@id': `${SITE_URL}/#organization` },
    potentialAction: [
      {
        '@type': 'SearchAction',
        target: {
          '@type': 'EntryPoint',
          urlTemplate: `${SITE_URL}/alternatives?q={search_term_string}`,
        },
        'query-input': 'required name=search_term_string',
      },
      {
        '@type': 'CommunicateAction',
        target: absoluteUrl('/contact'),
        name: language === 'ru' ? 'Связаться с Pegasus' : 'Contact Pegasus',
      },
    ],
  };
}

export function softwareApplicationJsonLd(language: Language = 'en') {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    '@id': `${SITE_URL}/#software`,
    name: `${SITE_NAME} — Global Logistics & Supply Chain Software`,
    alternateName: [
      'Pegasus',
      'Pegasus Logistics',
      'Pegasus Global Logistics',
      'Pegasus Supply Chain Software',
      'Pegasus Logistics Automation',
      'Pegasus TMS',
    ],
    applicationCategory: 'BusinessApplication, LogisticsSoftware, SupplyChainSoftware, FleetManagementSoftware',
    applicationSubCategory: 'Global Logistics, Supply Chain Management & Fleet Automation',
    operatingSystem: 'Web, Windows, macOS, Android, iOS',
    inLanguage: language === 'ru' ? 'ru' : 'en',
    description:
      language === 'ru'
        ? 'Глобальная B2B система управления цепочками поставок и автоматизации логистики: умная диспетчеризация автопарка, оптимизация маршрутов CVRP, сверка казначейства и приложения для 6 ролей.'
        : 'Global supply chain software and logistics automation operating system with multi-stop route optimization, real-time fleet telemetry, warehouse gate control, and treasury reconciliation.',
    offers: {
      '@type': 'Offer',
      price: '0',
      priceCurrency: 'USD',
      description:
        language === 'ru'
          ? 'Свяжитесь для enterprise-демонстрации и внедрения'
          : 'Contact for enterprise demo and deployment',
      url: absoluteUrl('/join'),
    },
    featureList: [
      'Global Logistics & Multi-Region Cell Cloud Architecture',
      'Enterprise Supply Chain Management (SCM) & Multi-Enterprise Execution',
      'Logistics Automation & Automated Dispatch Optimization (Google OR-Tools CVRP)',
      'Real-Time Global Fleet Telemetry & GPS Tracking',
      'Multi-Role Supply Chain Orchestration across 6 Roles',
      'Warehouse Gate Terminal & Digital Chain of Custody Barcode Seals',
      'Point-of-Delivery Invoice Settlement & Automated Treasury Reconciliation',
      'Bi-Directional ERP & WMS Integrations (SAP, NetSuite, 1C)',
    ],
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
