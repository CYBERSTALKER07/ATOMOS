import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
import PlatformValue from './components/PlatformValue';
import Skills from './components/Skills';
import SiteNav from './components/explore/SiteNav';
import type { Metadata } from 'next';
import {
  pageMetadata,
  organizationJsonLd,
  websiteJsonLd,
  softwareApplicationJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';

const LogisticsSolutions = dynamic(() => import('./components/LogisticsSolutions'));
const SignalFeatureCards = dynamic(() => import('./components/SignalFeatureCards'));
const OrderCycleVisualSection = dynamic(() => import('./components/OrderCycleVisualSection'));
const DispatchVisualSection = dynamic(() => import('./components/DispatchVisualSection'));
const DevelopmentTools = dynamic(() => import('./components/DevelopmentTools'));
const Projects = dynamic(() => import('./components/Projects'));
const Companies = dynamic(() => import('./components/Companies'));
const Licensing = dynamic(() => import('./components/Licensing'));
const Footer = dynamic(() => import('./components/Footer'));

export const metadata: Metadata = pageMetadata({
  title: 'Home',
  description:
    'Pegasus is the logistics operating system for supplier-led networks — dispatch, fleet tracking, payments, and coordination across supplier, warehouse, retailer, driver, factory, and gate teams.',
  path: '/',
});

export default function Home() {
  const structuredData = [
    organizationJsonLd(),
    websiteJsonLd(),
    softwareApplicationJsonLd(),
  ];

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
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
