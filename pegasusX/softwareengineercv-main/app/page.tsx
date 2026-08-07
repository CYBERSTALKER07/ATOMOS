import dynamic from 'next/dynamic';
import Hero from './components/Hero';
import About from './components/About';
import Skills from './components/Skills';
import SiteNav from './components/explore/SiteNav';
import LocalizedLaneDivider from './components/layout/LocalizedLaneDivider';
import type { Metadata } from 'next';
import {
  pageMetadata,
  organizationJsonLd,
  websiteJsonLd,
  softwareApplicationJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { translations } from '@/app/lib/i18n/translations';


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

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const dict = translations[lang] ?? translations.en;
  return pageMetadata({
    title: dict.meta_home_title,
    description: dict.meta_home_desc,
    path: '/',
  });
}

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
        <LocalizedLaneDivider index="01" labelKey="home_lane_network" />
        <About />
        {/* <PlatformValue /> */}
        <LocalizedLaneDivider index="02" labelKey="home_lane_signal" />
        {/* <SignalFeatureCards /> */}

        <DispatchVisualSection />
        <PlatformFeatures />
        <PromptDashboardSection />
        <AskPromptSection />
        <EcosystemStats />
        <LogisticsWorkflow />
        <OurApproach />
        <LocalizedLaneDivider index="03" labelKey="home_lane_operations" />
        <Skills />
        <DevelopmentTools />
        <LocalizedLaneDivider index="04" labelKey="home_lane_proof" />
        <PegasusTestimonialsSection />
        <Projects />
        <Companies />
        <Licensing />
        <Footer />
      </div>
    </>
  );
}
