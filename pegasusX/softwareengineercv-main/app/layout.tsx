import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import ClientLayout from "@/components/ClientLayout";
import { BRAND_LOGO, OG_IMAGE } from "@/app/lib/siteAssets";
import { absoluteAsset, SITE_NAME, SITE_URL } from "@/app/lib/seo";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: `${SITE_NAME} | Logistics Operating System`,
    template: `%s | ${SITE_NAME}`,
  },
  description: 'Pegasus is the logistics operating system for supplier-led networks. Dispatch, fleet tracking, payments, and realtime coordination across six roles.',
  keywords: [
    'logistics software',
    'dispatch system',
    'fleet tracking',
    'supplier network',
    'warehouse management',
    'delivery tracking',
    'payment reconciliation',
    'last mile delivery',
    'Pegasus',
    'logistics platform',
    'supply chain operations',
    'cash on delivery',
  ],
  authors: [{ name: SITE_NAME }],
  creator: SITE_NAME,
  publisher: SITE_NAME,
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: SITE_URL,
    title: `${SITE_NAME} | Logistics Operating System`,
    description: 'Run supplier-led logistics from one platform — dispatch, tracking, payments, and coordination across every team in your network.',
    siteName: SITE_NAME,
    images: [
      {
        url: absoluteAsset(OG_IMAGE),
        width: 512,
        height: 512,
        alt: 'Pegasus logo',
        type: 'image/jpeg',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Pegasus | Logistics Operating System',
    description: 'Dispatch, fleet tracking, payments, and realtime coordination for supplier-led logistics networks.',
    creator: '@pegasus',
    images: [absoluteAsset(OG_IMAGE)],
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
    icon: BRAND_LOGO,
    shortcut: BRAND_LOGO,
    apple: '/web-app-manifest-192x192.png',
  },
  manifest: '/manifest.json',
  ...(process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION
    ? {
        verification: {
          google: process.env.NEXT_PUBLIC_GOOGLE_SITE_VERIFICATION,
        },
      }
    : {}),
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        <meta name="theme-color" content="#000000" />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased relative bg-black`}
      >
        <a
          href="#main-content"
          className="fixed top-4 left-4 z-[10001] bg-white text-black px-4 py-2 rounded-md transition-transform -translate-y-20 focus:translate-y-0 font-light border-2 border-black"
        >
          Skip to content
        </a>
        <ClientLayout>
          <main id="main-content" tabIndex={-1} className="outline-none">
            {children}
          </main>
        </ClientLayout>
      </body>
    </html>
  );
}
