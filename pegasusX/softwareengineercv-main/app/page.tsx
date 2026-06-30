import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
import PlatformValue from './components/PlatformValue';
import Skills from './components/Skills';
import PillNav from './components/PillNav';
import type { Metadata } from 'next';

const LogisticsSolutions = dynamic(() => import('./components/LogisticsSolutions'));
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
        url: '/og-image.png',
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
        <PillNav
          logo=""
          logoAlt="Pegasus Logo"
          showMenuButton
          items={[
            { label: 'Home', href: '#' },
            { label: 'About', href: '#about' },
            { label: 'Solutions', href: '#solutions' },
            { label: 'Projects', href: '#projects' },
            { label: 'Demo', href: '/join' },
          ]}
          activeHref="#"
          baseColor="#000000"
          pillColor="#ffffff"
          hoveredPillTextColor="#ffffff"
          pillTextColor="#000000"
        />

        <Hero />
        <About />
        <PlatformValue />
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
