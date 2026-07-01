import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import ClientLayout from "@/components/ClientLayout";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || 'https://pegasus.io'),
  title: {
    default: 'Pegasus | Logistics Operating System',
    template: '%s | Pegasus'
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
  authors: [{ name: 'Pegasus' }],
  creator: 'Pegasus',
  publisher: 'Pegasus',
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://pegasus.io',
    title: 'Pegasus | Logistics Operating System',
    description: 'Run supplier-led logistics from one platform — dispatch, tracking, payments, and coordination across every team in your network.',
    siteName: 'Pegasus',
    images: [
      {
        url: '/og-image.png',
        width: 1200,
        height: 630,
        alt: 'Pegasus Logistics Platform',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Pegasus | Logistics Operating System',
    description: 'Dispatch, fleet tracking, payments, and realtime coordination for supplier-led logistics networks.',
    creator: '@pegasus',
    images: ['/og-image.png'],
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
    icon: '/atom.jpeg',
    shortcut: '/atom.jpeg',
    apple: '/atom.jpeg',
  },
  manifest: '/manifest.json',
  verification: {
    google: 'your-google-verification-code',
    // yandex: 'your-yandex-verification-code',
    // bing: 'your-bing-verification-code',
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        <link rel="canonical" href="https://pegasus.io" />
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
