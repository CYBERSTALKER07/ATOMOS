import type { Metadata } from "next";
import "./globals.css";
import ClientLayout from "@/components/ClientLayout";
import { BRAND_LOGO } from "@/app/lib/siteAssets";
import { absoluteUrl, languageAlternates, SITE_NAME, SITE_URL } from "@/app/lib/seo";
import { getServerLanguage } from "@/app/lib/i18n/server";
import { translations } from "@/app/lib/i18n/translations";

const geistSans = {
  variable: "font-sans",
};

const geistMono = {
  variable: "font-mono",
};

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const dict = translations[lang] ?? translations.en;
  const titleSuffix = dict.meta_root_title;
  const description = dict.meta_root_desc;
  const ogDescription = dict.meta_og_desc;
  const twitterDescription = dict.meta_twitter_desc;
  const locale = lang === 'ru' ? 'ru_RU' : 'en_US';

  return {
    metadataBase: new URL(SITE_URL),
    title: {
      default: `${SITE_NAME} | ${titleSuffix}`,
      template: `%s | ${SITE_NAME}`,
    },
    description,
    keywords:
      lang === 'ru'
        ? [
            'Pegasus',
            'Pegasus Logistics',
            'Глобальная логистика',
            'ПО для цепей поставок',
            'Автоматизация логистики',
            'Transportation Management System',
            'TMS система',
            'система диспетчеризации',
            'мониторинг автопарка',
            'оптимизация маршрутов',
            'управление цепочками поставок',
            'управление складом',
            'отслеживание доставки',
            'сверка платежей',
            'last mile delivery',
            'Pegasus TMS',
            'логистическая платформа',
            'Samsara аналоги',
          ]
        : [
            'Pegasus',
            'Pegasus Logistics',
            'Global Logistics',
            'Supply Chain Software',
            'Logistics Automation',
            'Transportation Management System',
            'TMS software',
            'Fleet dispatch software',
            'Route optimization software',
            'Supply chain management',
            'Delivery management system',
            'logistics software',
            'dispatch system',
            'fleet tracking',
            'supplier network',
            'warehouse management',
            'payment reconciliation',
            'last mile delivery',
            'Pegasus TMS',
            'Samsara alternative',
            'Rose Rocket alternative',
            'Motive alternative',
            'Turvo alternative',
            'Onfleet alternative',
          ],
    authors: [{ name: SITE_NAME }],
    creator: SITE_NAME,
    publisher: SITE_NAME,
    formatDetection: {
      email: false,
      address: false,
      telephone: false,
    },
    alternates: {
      canonical: SITE_URL,
      languages: languageAlternates(''),
    },
    openGraph: {
      type: 'website',
      locale,
      alternateLocale: [lang === 'ru' ? 'en_US' : 'ru_RU'],
      url: SITE_URL,
      title: `${SITE_NAME} | ${titleSuffix}`,
      description: ogDescription,
      siteName: SITE_NAME,
      images: [
        {
          url: absoluteUrl('/opengraph-image'),
          width: 1200,
          height: 630,
          alt: 'Pegasus — Logistics Operating System',
          type: 'image/png',
        },
      ],
    },
    twitter: {
      card: 'summary_large_image',
      title: `${SITE_NAME} | ${titleSuffix}`,
      description: twitterDescription,
      creator: '@pegasus',
      images: [absoluteUrl('/opengraph-image')],
    },
    robots: {
      index: true,
      follow: true,
      googleBot: {
        index: true,
        follow: true,
        'max-video-preview': -1,
        'max-image-preview': 'large',
        'max-snippet': -1,
      },
    },
    icons: {
      icon: [
        { url: '/favicon.ico', sizes: 'any' },
        { url: '/pegasus-icon-192.png', sizes: '192x192', type: 'image/png' },
        { url: '/pegasus-icon-512.png', sizes: '512x512', type: 'image/png' },
        { url: BRAND_LOGO, type: 'image/jpeg' },
      ],
      shortcut: '/favicon.ico',
      apple: [{ url: '/pegasus-icon-192.png', sizes: '180x180', type: 'image/png' }],
    },
    ...(process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION
      ? {
          verification: {
            google: process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION,
          },
        }
      : {}),
  };
}

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const lang = await getServerLanguage();
  const dict = translations[lang] ?? translations.en;
  const skipLabel = dict.skip_to_content || 'Skip to content';
  return (
    <html lang={lang} dir={lang === 'ar' ? 'rtl' : 'ltr'}>
      <head>
        <meta name="theme-color" content="#000000" />
        <script
          dangerouslySetInnerHTML={{
            __html: `
              try {
                if (sessionStorage.getItem('hasSeenSplash')) {
                  document.documentElement.classList.add('splash-done');
                }
                const openuiDevtoolsFlag = Symbol.for('openui.devtools.autoMount');
                window[openuiDevtoolsFlag] = true;
              } catch (e) {}
            `,
          }}
        />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased relative bg-black`}
      >
        <div
          id="app-splash-screen"
          className="fixed inset-0 z-[10006] bg-black flex items-center justify-center overflow-hidden pointer-events-auto"
          aria-label="Pegasus Loading"
        >
          <img
            src="/pegasus.jpg"
            alt="Pegasus Logo"
            className="w-14 h-14 sm:w-16 sm:h-16 object-contain select-none"
          />
        </div>
        <a
          href="#main-content"
          className="fixed top-4 left-4 z-[10001] bg-white text-black px-4 py-2 rounded-md transition-transform -translate-y-20 focus:translate-y-0 font-light border-2 border-black"
        >
          {skipLabel}
        </a>
        <ClientLayout initialLanguage={lang}>
          <main id="main-content" tabIndex={-1} className="outline-none">
            {children}
          </main>
        </ClientLayout>
      </body>
    </html>
  );
}
