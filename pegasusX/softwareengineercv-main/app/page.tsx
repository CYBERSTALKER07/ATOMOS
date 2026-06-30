import Hero from './components/Hero';
import About from './components/About';
import Skills from './components/Skills';
import DevelopmentTools from './components/DevelopmentTools';
import Projects from './components/Projects';
import Companies from './components/Companies';
import Licensing from './components/Licensing';
import Contact from './components/Contact';
import Footer from './components/Footer';
import PillNav from './components/PillNav';
import Link from 'next/link';
import type { Metadata } from 'next';

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
  // Structured Data (JSON-LD) for SEO
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
      {/* Structured Data */}
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
          items={[
            { label: 'Home', href: '#' },
            { label: 'About', href: '#about' },
            { label: 'Skills', href: '#skills' },
            { label: 'Tools', href: '#tools' },
            { label: 'Projects', href: '#projects' },
            { label: 'Mobile Apps', href: '/mobile-apps' },
            { label: 'Web Apps', href: '/web-apps' },
            { label: 'Desktop Apps', href: '/desktop-apps' },
            { label: 'Roles', href: '#companies' },
            { label: 'Deploy', href: '#licensing' },
            { label: 'Contact', href: '#contact' }
          ]}
          activeHref="#"
          baseColor="#000000"
          pillColor="#ffffff"
          hoveredPillTextColor="#ffffff"
          pillTextColor="#000000"
        />
        
        {/* Floating "Join Us" Button */}
        <Link 
          href="/join"
          className="fixed bottom-8 right-8 z-50 px-8 py-4 bg-white text-black border-2 border-white hover:bg-black hover:text-white transition-all duration-300 font-bold text-lg shadow-lg rounded-3xl focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-black outline-none"
          aria-label="Request a Pegasus demo"
        >
          Request Demo →
        </Link>
        
        <Hero />
        <About />
        <Skills />
        <DevelopmentTools />
        <Projects />
        <Companies />
        <Licensing />
        <Contact />
        <Footer />
      </div>
    </>
  );
}
