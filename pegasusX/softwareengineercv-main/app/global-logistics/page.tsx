import type { Metadata } from 'next';
import Link from 'next/link';
import {
  Globe2,
  ShieldCheck,
  Zap,
  ArrowRight,
  Server,
  Layers,
  CheckCircle2,
  ChevronRight,
  BarChart3,
  Truck,
  RefreshCw,
  Radio,
  Building2,
  Sparkles,
} from 'lucide-react';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { getServerLanguage } from '@/app/lib/i18n/server';
import {
  pageMetadata,
  faqPageJsonLd,
  breadcrumbJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? 'Глобальная логистика: платформа управления и сеть | Pegasus'
    : 'Global Logistics Software & Network Platform | Pegasus';

  const description = isRu
    ? 'Управляйте глобальной логистикой через мультирегиональные облачные ячейки, кросс-бордер фрахт и автоматизацию автопарка на единой платформе Pegasus.'
    : 'Orchestrate global logistics across suppliers, multi-region cells, cross-border freight, and automated carrier fleets on one real-time cloud platform.';

  return pageMetadata({
    title,
    description,
    path: '/global-logistics',
    language: lang,
  });
}

const GLOBAL_LOGISTICS_FAQS = [
  {
    question: 'What is a global logistics operating system?',
    answer:
      'A global logistics operating system is an enterprise cloud architecture that unifies multi-country supplier networks, multi-depot warehouse fulfillment, cross-border freight forwarding, and carrier fleet tracking into a single synchronized state machine, eliminating manual spreadsheet handoffs and latency.',
  },
  {
    question: 'How does Pegasus handle multi-region global logistics latency?',
    answer:
      'Pegasus deploys distributed multi-region cell architecture (e.g. cell-uz, cell-eu, cell-us) anchored by Google Cloud Spanner distributed transactions and Maglev consistent hashing, providing sub-millisecond local reads and ACID-compliant distributed cross-border order commits.',
  },
  {
    question: 'How does Pegasus support multi-currency and cross-border settlement?',
    answer:
      'Pegasus integrates dual-entry ledger accounting with real-time foreign exchange rate anchoring, enabling shippers, carriers, and retailers to invoice and settle in local fiat currencies or international trade credits with automated tax and tariff reconciliation.',
  },
  {
    question: 'Can Pegasus integrate with legacy international freight systems?',
    answer:
      'Yes. Pegasus exposes bi-directional REST, gRPC, and Kafka event streaming connectors that interface with enterprise ERPs (SAP S/4HANA, Oracle NetSuite, 1C:Enterprise) and legacy customs EDI message formats.',
  },
  {
    question: 'How does Pegasus optimize multi-stop vehicle routes across global borders?',
    answer:
      'Pegasus utilizes custom mathematical solvers integrated with Google OR-Tools CVRP (Capacitated Vehicle Routing Problem) algorithms, taking into account border wait windows, vehicle weight limits (GVW), driver rest hours, and multi-depot staging.',
  },
];

export default async function GlobalLogisticsPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const structuredData = [
    faqPageJsonLd(GLOBAL_LOGISTICS_FAQS, lang),
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Global Logistics', path: '/global-logistics' },
    ]),
  ];

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-white selection:text-black">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />
      <SiteNav activeHref="/global-logistics" />

      <main className="flex-1 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/" className="hover:text-white transition-colors">
            {isRu ? 'Главная' : 'Home'}
          </Link>
          <ChevronRight className="w-3.5 h-3.5 text-white/30" />
          <span className="text-white/80">Global Logistics</span>
        </div>

        {/* Hero Header */}
        <div className="border-b border-white/10 pb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <Globe2 className="w-3.5 h-3.5 text-blue-400" />
            <span>Global Scale Enterprise Infrastructure</span>
          </div>

          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-white uppercase leading-[1.05]">
            Global Logistics Operating System
          </h1>

          <p className="mt-6 text-xl sm:text-2xl text-white/70 leading-relaxed font-light max-w-3xl">
            Orchestrate multi-region supplier networks, automated cross-border carrier fleets, and real-time inventory fulfillment on one synchronized cloud platform.
          </p>

          <div className="mt-10 flex flex-wrap gap-4">
            <Link
              href="/join"
              className="px-7 py-3.5 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors flex items-center gap-2"
            >
              <span>Schedule Global Architecture Demo</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link
              href="/platform"
              className="px-7 py-3.5 rounded-lg bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Explore Cloud Architecture
            </Link>
          </div>
        </div>

        {/* Key Operational Pillars */}
        <section className="mt-20">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-blue-400">
              Distributed Infrastructure
            </span>
            <h2 className="text-3xl sm:text-4xl font-black uppercase tracking-tight text-white mt-2">
              Architected for Global Scale & Sovereignty
            </h2>
            <p className="text-sm text-white/60 font-light mt-2 max-w-2xl">
              Eliminate cross-continental data lag and regulatory friction with sovereign, cloned cell clusters that communicate over high-speed distributed consensus pipelines.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 mb-6">
                <Server className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold uppercase text-white mb-3">
                Multi-Region Cell Architecture
              </h3>
              <p className="text-sm text-white/70 leading-relaxed font-light">
                Isolated geographic cells (e.g. EU, US, Central Asia) deliver zero-latency localized operations with automated cross-cell synchronization via Google Cloud Spanner and Kafka Outbox.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 mb-6">
                <Radio className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold uppercase text-white mb-3">
                Cross-Border Fleet Telemetry
              </h3>
              <p className="text-sm text-white/70 leading-relaxed font-light">
                Continuous high-frequency GPS tracking and dead-reckoning algorithms maintain real-time vehicle visibility even through cellular dead-zones, international borders, and transit tunnels.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400 mb-6">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold uppercase text-white mb-3">
                Digital Chain of Custody
              </h3>
              <p className="text-sm text-white/70 leading-relaxed font-light">
                Cryptographically validated gate barcode scans, automated tamper-evident seals, and immutable state machines guarantee complete physical accountability from factory to store shelf.
              </p>
            </div>
          </div>
        </section>

        {/* Global Logistics Architecture Comparison */}
        <section className="mt-24 p-8 sm:p-12 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="mb-8">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-blue-400">
              Technical Comparison
            </span>
            <h2 className="text-2xl sm:text-3xl font-black uppercase tracking-tight text-white mt-1">
              Pegasus Global Logistics OS vs Traditional Systems
            </h2>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-sm font-mono">
              <thead>
                <tr className="border-b border-white/10 text-white/50 text-[11px] uppercase tracking-wider">
                  <th className="py-4 pr-6">Capability</th>
                  <th className="py-4 px-6 text-white font-bold bg-white/[0.04]">Pegasus Global OS</th>
                  <th className="py-4 px-6">Legacy Global TMS</th>
                  <th className="py-4 pl-6">Freight Forwarder Portals</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-white/80">
                <tr>
                  <td className="py-4 pr-6 font-bold text-white">Database Foundation</td>
                  <td className="py-4 px-6 bg-white/[0.04] text-emerald-400 font-bold">Google Cloud Spanner (ACID, 99.999%)</td>
                  <td className="py-4 px-6 text-white/50">Legacy Relational / On-Prem</td>
                  <td className="py-4 pl-6 text-white/50">Fragmented 3rd-Party APIs</td>
                </tr>
                <tr>
                  <td className="py-4 pr-6 font-bold text-white">Cross-Border Routing</td>
                  <td className="py-4 px-6 bg-white/[0.04] text-emerald-400 font-bold">Google OR-Tools CVRP + Customs Windows</td>
                  <td className="py-4 px-6 text-white/50">Static Distance Tables</td>
                  <td className="py-4 pl-6 text-white/50">Manual Broker Estimates</td>
                </tr>
                <tr>
                  <td className="py-4 pr-6 font-bold text-white">Realtime Latency</td>
                  <td className="py-4 px-6 bg-white/[0.04] text-emerald-400 font-bold">&lt; 100ms WebSocket & Redis Streams</td>
                  <td className="py-4 px-6 text-white/50">15–60 min Batch Polling</td>
                  <td className="py-4 pl-6 text-white/50">Daily EDI Status Updates</td>
                </tr>
                <tr>
                  <td className="py-4 pr-6 font-bold text-white">Multi-Role Native Apps</td>
                  <td className="py-4 px-6 bg-white/[0.04] text-emerald-400 font-bold">6 Dedicated Roles (Mobile & Desktop)</td>
                  <td className="py-4 px-6 text-white/50">Single Complex Web Portal</td>
                  <td className="py-4 pl-6 text-white/50">Email & PDF Invoices</td>
                </tr>
                <tr>
                  <td className="py-4 pr-6 font-bold text-white">Payment & Treasury</td>
                  <td className="py-4 px-6 bg-white/[0.04] text-emerald-400 font-bold">Automated Point-of-Delivery COD & Invoicing</td>
                  <td className="py-4 px-6 text-white/50">Manual 60-day Net Billing</td>
                  <td className="py-4 pl-6 text-white/50">Manual Factor Invoicing</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Global Logistics FAQ */}
        <section className="mt-24">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-blue-400">
              Common Questions
            </span>
            <h2 className="text-3xl font-black uppercase tracking-tight text-white mt-2">
              Global Logistics Architecture FAQ
            </h2>
          </div>

          <div className="space-y-6">
            {GLOBAL_LOGISTICS_FAQS.map((faq, idx) => (
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
            Ready to Modernize Your Global Logistics?
          </h2>
          <p className="mt-4 text-base sm:text-lg text-white/70 max-w-2xl mx-auto font-light">
            Deploy the modern logistics operating system built for high-throughput enterprise networks, automated fleets, and frictionless cross-border trade.
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
              Speak with a Solutions Architect
            </Link>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
