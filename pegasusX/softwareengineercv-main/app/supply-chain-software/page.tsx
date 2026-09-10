import type { Metadata } from 'next';
import Link from 'next/link';
import {
  Boxes,
  Network,
  ShieldCheck,
  Zap,
  ArrowRight,
  ChevronRight,
  Database,
  BarChart2,
  Workflow,
  CheckCircle2,
  Building2,
  Cpu,
  Layers,
  FileCheck2,
} from 'lucide-react';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { getServerLanguage } from '@/app/lib/i18n/server';
import {
  pageMetadata,
  faqPageJsonLd,
  breadcrumbJsonLd,
  softwareApplicationJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';

import DossierHero from '@/app/components/dossier/DossierHero';
import { DOSSIER_PAGE_CONFIGS } from '@/app/components/dossier/dossierPageConfigs';
import {
  TacticalPillarsBento,
  BranchingTimelineSection,
  RadialHubSpokeSection,
  SECTION_PAGE_CONFIGS,
} from '@/app/components/sections';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? 'ПО для цепей поставок и сквозная координация | Pegasus'
    : 'Enterprise Supply Chain Software & Orchestration | Pegasus';

  const description = isRu
    ? 'Трансформируйте цепи поставок с Pegasus: сквозной цикл от заказа до оплаты, учёт остатков, интеграция с ERP/WMS и приложения для 6 ролей сети.'
    : 'Transform B2B supply chains with Pegasus: unified order-to-cash execution, multi-tier inventory visibility, ERP/WMS sync, and 6-role network portals.';

  return pageMetadata({
    title,
    description,
    path: '/supply-chain-software',
    language: lang,
  });
}

const SUPPLY_CHAIN_FAQS = [
  {
    question: 'What differentiates Pegasus from traditional supply chain software (SCM)?',
    answer:
      'Traditional SCM platforms are planning tools separated from execution, requiring manual handoffs across disconnected third-party portals. Pegasus is a closed-loop execution system connecting suppliers, warehouses, factories, drivers, retailers, and treasury into one governed state machine.',
  },
  {
    question: 'How does Pegasus prevent concurrent stock over-allocation across depots?',
    answer:
      'Pegasus enforces strict transactional serializability on warehouse inventory commits. When multiple retailers place high-velocity orders during peak ordering windows, atomic reservation guards ensure zero double-booking or ghost commitments.',
  },
  {
    question: 'How easily does Pegasus integrate with existing enterprise ERPs?',
    answer:
      'Pegasus offers production-tested bi-directional connectors for SAP S/4HANA, Oracle NetSuite, Microsoft Dynamics 365, and 1C:Enterprise. Changes sync in real time via transactional outbox messaging and webhooks.',
  },
  {
    question: 'What roles does the supply chain software provide dedicated apps for?',
    answer:
      'Pegasus delivers tailored desktop, mobile, and web applications for 6 discrete roles: Suppliers (Control Plane), Warehouse Managers (Visual Dispatch), Factory Supervisors (Loading Bay), Drivers (Turn-by-Turn Route Execution), Retailers (B2B Commerce & Tracking), and Gate Controllers (Barcode Seal Verification).',
  },
  {
    question: 'How does Pegasus streamline the Order-to-Cash (O2C) cycle?',
    answer:
      'Every order transitions through governed status transitions: Placed → Loading → In Transit → Arrived → Delivered. Proof of delivery triggers automatic invoice generation, credit line adjustment, and cash-on-delivery reconciliation.',
  },
];

export default async function SupplyChainSoftwarePage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const structuredData = [
    softwareApplicationJsonLd(lang),
    faqPageJsonLd(SUPPLY_CHAIN_FAQS, lang),
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Supply Chain Software', path: '/supply-chain-software' },
    ]),
  ];

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-white selection:text-black">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />
      <SiteNav activeHref="/supply-chain-software" />

      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-28 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-6">
          <Link href="/" className="hover:text-white transition-colors">
            {isRu ? 'Главная' : 'Home'}
          </Link>
          <ChevronRight className="w-3.5 h-3.5 text-white/30" />
          <span className="text-white/80">Supply Chain Software</span>
        </div>

        {/* Dossier Hero Section */}
        <div className="mb-14">
          <DossierHero
            {...DOSSIER_PAGE_CONFIGS['supply-chain-software']}
            tabLabel={isRu ? 'PEGASUS / ЦЕПИ ПОСТАВОК' : 'PEGASUS / SUPPLY CHAIN'}
            className="!px-0"
          />
        </div>

        {/* Modular Sections: Pillars Bento, Branching Timeline, and Radial Hub-Spoke */}
        <TacticalPillarsBento config={SECTION_PAGE_CONFIGS['supply-chain-software'].pillars} />
        <BranchingTimelineSection config={SECTION_PAGE_CONFIGS['supply-chain-software'].timeline} />
        <RadialHubSpokeSection config={SECTION_PAGE_CONFIGS['supply-chain-software'].radial} />

        {/* 6 Roles Section */}
        <section className="mt-24 p-8 sm:p-12 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="mb-10 text-center max-w-2xl mx-auto">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-emerald-400">
              Unified Ecosystem
            </span>
            <h2 className="text-3xl font-black uppercase tracking-tight text-white mt-2">
              Six Purpose-Built Apps. One Operating Core.
            </h2>
            <p className="text-sm text-white/60 font-light mt-2">
              No generic web portals. Each stakeholder interacts through an application optimized specifically for their operating environment.
            </p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {[
              {
                role: 'Supplier Control Plane',
                desc: 'Catalog management, price tiers, retailer vetting, multi-depot topology, and live network analytics.',
                href: '/roles/supplier',
              },
              {
                role: 'Warehouse Operations',
                desc: 'Morning dispatch boards, pallet staging, stock commits, split-load resolution, and fleet management.',
                href: '/roles/warehouse',
              },
              {
                role: 'Factory Loading Bay',
                desc: 'Production request queues, outbound manifest assignment, and dock staging synchronization.',
                href: '/roles/factory',
              },
              {
                role: 'Driver Execution App',
                desc: 'Turn-by-turn CVRP navigation, barcode scan verification, offline cash collection, and digital PoD.',
                href: '/roles/driver',
              },
              {
                role: 'Retailer Commerce Portal',
                desc: 'Multi-supplier ordering catalog, trade credit status, live truck map tracking, and dispute filing.',
                href: '/roles/retailer',
              },
              {
                role: 'Gate & Yard Terminal',
                desc: 'Barcode seal validation, security inspection sign-offs, and automated yard check-in/out.',
                href: '/roles/payload-gate',
              },
            ].map((item, idx) => (
              <div key={idx} className="p-6 rounded-xl bg-white/[0.02] border border-white/5 flex flex-col justify-between">
                <div>
                  <h3 className="text-base font-bold uppercase text-white mb-2">{item.role}</h3>
                  <p className="text-xs text-white/70 leading-relaxed font-light">{item.desc}</p>
                </div>
                <Link
                  href={item.href}
                  className="mt-4 text-xs font-mono text-emerald-400 hover:text-emerald-300 flex items-center gap-1 uppercase tracking-wider"
                >
                  <span>Explore App</span>
                  <ChevronRight className="w-3.5 h-3.5" />
                </Link>
              </div>
            ))}
          </div>
        </section>

        {/* Enterprise Supply Chain FAQ */}
        <section className="mt-24">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-emerald-400">
              FAQ
            </span>
            <h2 className="text-3xl font-black uppercase tracking-tight text-white mt-2">
              Supply Chain Software Architecture FAQ
            </h2>
          </div>

          <div className="space-y-6">
            {SUPPLY_CHAIN_FAQS.map((faq, idx) => (
              <div
                key={idx}
                className="p-6 sm:p-8 rounded-2xl bg-white/[0.02] border border-white/10"
              >
                <h3 className="text-lg font-bold text-white mb-3">
                  {faq.question}
                </h3>
                <p className="text-sm text-white/70 leading-relaxed font-light">
                  {faq.answer}
                </p>
              </div>
            ))}
          </div>
        </section>

        {/* Bottom CTA */}
        <section className="mt-24 p-10 sm:p-14 rounded-3xl bg-gradient-to-b from-white/[0.06] to-transparent border border-white/15 text-center">
          <h2 className="text-3xl sm:text-5xl font-black uppercase tracking-tight text-white">
            Upgrade to Unified Supply Chain Execution
          </h2>
          <p className="mt-4 text-base sm:text-lg text-white/70 max-w-2xl mx-auto font-light">
            Eliminate dropped orders, inventory discrepancies, and fragmented communications with Pegasus enterprise supply chain software.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/join"
              className="px-8 py-4 rounded-xl bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors shadow-xl"
            >
              Request Live Enterprise Demo
            </Link>
            <Link
              href="/contact"
              className="px-8 py-4 rounded-xl bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Talk with an Integration Specialist
            </Link>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
