import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
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

const OrderCycleVisualSection = dynamic(() => import('./components/OrderCycleVisualSection'));
const DispatchVisualSection = dynamic(() => import('./components/DispatchVisualSection'));
const PlatformFeatures = dynamic(() => import('./components/PlatformFeatures'));
const PromptDashboardSection = dynamic(() => import('./components/PromptDashboardSection'));
const AskPromptSection = dynamic(() => import('./components/ask-prompt/AskPromptSection'));
const EcosystemStats = dynamic(() => import('./components/EcosystemStats'));
const LogisticsWorkflow = dynamic(() => import('./components/LogisticsWorkflow'));
const OurApproach = dynamic(() => import('./components/OurApproach'));
const DevelopmentTools = dynamic(() => import('./components/DevelopmentTools'));
const Projects = dynamic(() => import('./components/Projects'));
const Companies = dynamic(() => import('./components/Companies'));
const PegasusTestimonialsSection = dynamic(() => import('./components/PegasusTestimonialsSection').then((mod) => mod.PegasusTestimonialsSection));
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
        {/* <PlatformValue /> */}
        <LaneDivider index="02" label="Signal" />
        {/* <SignalFeatureCards /> */}
        <OrderCycleVisualSection />
        <DispatchVisualSection />
        <PlatformFeatures />
        <PromptDashboardSection />
        <AskPromptSection />
        <EcosystemStats />
        <LogisticsWorkflow />
        <OurApproach />
        <LaneDivider index="03" label="Operations" />
        <Skills />
        <DevelopmentTools />
        <LaneDivider index="04" label="Proof" />
        <PegasusTestimonialsSection />
        <Projects />
        <Companies />
        <Licensing />
        <Footer />
      </div>
    </>
  );
}
