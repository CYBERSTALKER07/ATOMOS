import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
import PlatformValue from './components/PlatformValue';
import Skills from './components/Skills';
import SiteNav from './components/explore/SiteNav';
import type { Metadata } from 'next';
import { OG_IMAGE } from '@/app/lib/siteAssets';

const LogisticsSolutions = dynamic(() => import('./components/LogisticsSolutions'));
const SignalFeatureCards = dynamic(() => import('./components/SignalFeatureCards'));
const OrderCycleVisualSection = dynamic(() => import('./components/OrderCycleVisualSection'));
const DispatchVisualSection = dynamic(() => import('./components/DispatchVisualSection'));
const DevelopmentTools = dynamic(() => import('./components/DevelopmentTools'));
const Projects = dynamic(() => import('./components/Projects'));
const Companies = dynamic(() => import('./components/Companies'));
const Licensing = dynamic(() => import('./components/Licensing'));
const Footer = dynamic(() => import('./components/Footer'));

export const metadata: Metadata = {
  title: 'Home',
  description: 'Pegasus — the logistics operating system for supplier-led networks. Dispatch, tracking, payments, and coordination across every team.',
  openGraph: {
    title: 'Pegasus | Logistics Operating System',
    description: 'Run supplier-led logistics from one platform — dispatch, tracking, payments, and coordination across every team in your network.',
    url: 'https://pegasus.io',
    images: [
      {
        url: OG_IMAGE,
        width: 1200,
        height: 630,
        alt: 'Pegasus Logistics Platform',
      },
    ],
  },
};

export default function Home() {
  const structuredData = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: 'Pegasus',
    description: 'Logistics operating system for supplier-led networks — dispatch, tracking, payments, and coordination',
    url: 'https://pegasus.io',
    sameAs: [
      'https://linkedin.com/company/pegasus',
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

  const websiteStructuredData = {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: 'Pegasus',
    description: 'Logistics operating system for supplier-led networks — dispatch, fleet tracking, payments, and realtime coordination',
    url: 'https://pegasus.io',
    publisher: {
      '@type': 'Organization',
      name: 'Pegasus',
    },
    inLanguage: 'en-US',
  };

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(websiteStructuredData) }}
      />

      <div>
        <SiteNav activeHref="/" />

        <Hero />
        <About />
        <PlatformValue />
        <SignalFeatureCards />
        <OrderCycleVisualSection />
        <DispatchVisualSection />
        <Skills />
        <LogisticsSolutions />
        <DevelopmentTools />
        <Projects />
        <Companies />
        <Licensing />
        <Footer />
      </div>
    </>
  );
}
