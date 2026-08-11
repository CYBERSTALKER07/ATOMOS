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
  faqPageJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { translations } from '@/app/lib/i18n/translations';


const SignalFeatureCards = dynamic(() => import('./components/SignalFeatureCards'));
const DispatchVisualSection = dynamic(() => import('./components/DispatchVisualSection'));
const LastMileSection = dynamic(() => import('./components/LastMileSection'));
const PlatformFeatures = dynamic(() => import('./components/PlatformFeatures'));
const PromptDashboardSection = dynamic(() => import('./components/PromptDashboardSection'));
const AskPromptSection = dynamic(() => import('./components/ask-prompt/AskPromptSection'));
const EcosystemStats = dynamic(() => import('./components/EcosystemStats'));
const LogisticsWorkflow = dynamic(() => import('./components/LogisticsWorkflow'));
const OurApproach = dynamic(() => import('./components/OurApproach'));
const DevelopmentTools = dynamic(() => import('./components/DevelopmentTools'));
const ShowcaseWall = dynamic(() => import('./components/ShowcaseWall'));
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
    language: lang,
  });
}

export default async function Home() {
  const lang = await getServerLanguage();
  const faqs =
    lang === 'ru'
      ? [
          {
            question: 'Что такое Pegasus?',
            answer:
              'Pegasus — операционная система логистики для сетей под управлением поставщика. Она объединяет диспетчеризацию, мониторинг автопарка, платежи и координацию шести ролей в одной системе состояний.',
          },
          {
            question: 'Какие роли поддерживает платформа?',
            answer:
              'Поставщик, склад, ритейлер, водитель, завод и ворота — у каждой роли свои приложения с общей правдой статуса заказа.',
          },
          {
            question: 'Как запросить демо?',
            answer:
              'Откройте страницу Request Demo (/join) или Contact и оставьте заявку — команда свяжется в течение рабочего дня.',
          },
        ]
      : [
          {
            question: 'What is Pegasus?',
            answer:
              'Pegasus is the logistics operating system for supplier-led networks. It unifies dispatch, fleet tracking, payments, and coordination across six roles in one governed state machine.',
          },
          {
            question: 'Which roles does the platform support?',
            answer:
              'Supplier, warehouse, retailer, driver, factory, and gate — each role gets purpose-built apps that share one order-status truth.',
          },
          {
            question: 'How do I request a demo?',
            answer:
              'Open the Request Demo page (/join) or Contact and submit the form — the team typically responds within one business day.',
          },
        ];

  const structuredData = [
    organizationJsonLd(lang),
    websiteJsonLd(lang),
    softwareApplicationJsonLd(lang),
    faqPageJsonLd(faqs, lang),
  ];

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />

      <div className="relative">
        <SiteNav activeHref="/" />

        <section id="section-overview">
          <Hero />
        </section>

        <LocalizedLaneDivider index="01" labelKey="home_lane_network" />

        <section id="section-platform">
          <About />
          <DispatchVisualSection />
        </section>

        <LocalizedLaneDivider index="02" labelKey="home_lane_last_mile" />

        <section id="section-last-mile">
          <LastMileSection />
        </section>

        <LocalizedLaneDivider index="03" labelKey="home_lane_signal" />

        <section id="section-telemetry">
          <PlatformFeatures />
          <PromptDashboardSection />
          <AskPromptSection />
        </section>

        <section id="section-workflow">
          <EcosystemStats />
          <LogisticsWorkflow />
          <OurApproach />
          <LocalizedLaneDivider index="04" labelKey="home_lane_operations" />
          <Skills />
          <DevelopmentTools />
        </section>

        <LocalizedLaneDivider index="05" labelKey="home_lane_proof" />

        <section id="section-showcase">
          <ShowcaseWall />
          <PegasusTestimonialsSection />
          <Projects />
          <Companies />
        </section>

        <section id="section-deploy">
          <Licensing />
          <Footer />
        </section>
      </div>
    </>
  );
}
