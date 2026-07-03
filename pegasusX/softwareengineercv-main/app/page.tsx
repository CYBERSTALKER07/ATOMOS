import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
import PlatformValue from './components/PlatformValue';
import Skills from './components/Skills';
import SiteNav from './components/explore/SiteNav';
import LaneDivider from './components/layout/LaneDivider';
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
const PaymentFlowSection = dynamic(() => import('./components/PaymentFlowSection'));
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
        <LaneDivider index="01" label="Network" />
        <About />
        <PlatformValue />
        <LaneDivider index="02" label="Signal" />
        <SignalFeatureCards />
        <OrderCycleVisualSection />
        <DispatchVisualSection />
        <PaymentFlowSection />
        <LaneDivider index="03" label="Operations" />
        <Skills />
        <LogisticsSolutions />
        <DevelopmentTools />
        <LaneDivider index="04" label="Proof" />
        <Projects />
        <Companies />
        <Licensing />
        <Footer />
      </div>
    </>
  );
}
