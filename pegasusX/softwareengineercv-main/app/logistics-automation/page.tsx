import type { Metadata } from 'next';
import Link from 'next/link';
import {
  Cpu,
  Zap,
  ShieldCheck,
  ArrowRight,
  ChevronRight,
  Route,
  ScanLine,
  Receipt,
  CheckCircle2,
  BarChart3,
  Truck,
  Timer,
  RefreshCw,
  Sparkles,
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

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? 'Автоматизация логистики: диспетчеризация и маршруты | Pegasus'
    : 'Logistics Automation & Automated Dispatch Platform | Pegasus';

  const description = isRu
    ? 'Автоматизируйте диспетчеризацию, расчет маршрутов (CVRP), КПП склада, подтверждение доставки и сверку платежей с системой автоматизации логистики Pegasus.'
    : 'Automate dispatch, vehicle routing (CVRP), warehouse gate check-ins, proof of delivery, and payment reconciliation with Pegasus logistics automation.';

  return pageMetadata({
    title,
    description,
    path: '/logistics-automation',
    language: lang,
  });
}

const LOGISTICS_AUTOMATION_FAQS = [
  {
    question: 'What is logistics automation in Pegasus?',
    answer:
      'Logistics automation in Pegasus is the end-to-end algorithmic orchestration of order dispatch, vehicle route optimization (CVRP), warehouse staging, security seal gate checks, and point-of-delivery payment reconciliation without manual spreadsheet coordination.',
  },
  {
    question: 'How does automated CVRP dispatching work?',
    answer:
      'Pegasus integrates Google OR-Tools algorithms to automatically group pending orders by delivery zones, vehicle payload capacities (GVW), volumetric cubic limits, driver working hours, and retailer receiving time windows within seconds.',
  },
  {
    question: 'How does automated gate security verification prevent loading errors?',
    answer:
      'Warehouse gate staff scan tamper-evident digital barcode seals attached to each outbound truck. Pegasus validates the manifest in real time against the assigned driver and destination stops, preventing incorrect trucks from leaving the yard.',
  },
  {
    question: 'How does automated proof-of-delivery (PoD) reduce disputes?',
    answer:
      'When a driver arrives at a retailer, geofencing triggers delivery mode. Drivers scan item barcodes, capture recipient signatures on glass, and record payment collection. The order status, invoice, and inventory ledgers are updated simultaneously in under 100 milliseconds.',
  },
  {
    question: 'What ROI can enterprises expect from logistics automation?',
    answer:
      'Enterprises running Pegasus typically observe a 42% reduction in morning dispatch planning time, a 19% reduction in total fleet fuel consumption through optimal routing, and 100% elimination of dropped handoffs and lost cash collections.',
  },
];

export default async function LogisticsAutomationPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const structuredData = [
    softwareApplicationJsonLd(lang),
    faqPageJsonLd(LOGISTICS_AUTOMATION_FAQS, lang),
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Logistics Automation', path: '/logistics-automation' },
    ]),
  ];

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-white selection:text-black">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />
      <SiteNav activeHref="/logistics-automation" />

      <main className="flex-1 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/" className="hover:text-white transition-colors">
            {isRu ? 'Главная' : 'Home'}
          </Link>
          <ChevronRight className="w-3.5 h-3.5 text-white/30" />
          <span className="text-white/80">Logistics Automation</span>
        </div>

        {/* Hero Section */}
        <div className="border-b border-white/10 pb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <Zap className="w-3.5 h-3.5 text-amber-400" />
            <span>Algorithmic Fleet & Yard Orchestration</span>
          </div>

          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-white uppercase leading-[1.05]">
            Intelligent Logistics Automation
          </h1>

          <p className="mt-6 text-xl sm:text-2xl text-white/70 leading-relaxed font-light max-w-3xl">
            Automate vehicle routing, visual dispatch load balancing, warehouse gate inspections, and treasury reconciliation with zero manual latency.
          </p>

          <div className="mt-10 flex flex-wrap gap-4">
            <Link
              href="/join"
              className="px-7 py-3.5 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors flex items-center gap-2"
            >
              <span>See Automation in Action</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link
              href="/capabilities/smarter-dispatch"
              className="px-7 py-3.5 rounded-lg bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Dispatch Optimization Guide
            </Link>
          </div>
        </div>

        {/* Benchmarks / ROI Bar */}
        <section className="mt-16 grid grid-cols-1 sm:grid-cols-3 gap-6">
          <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 text-center">
            <span className="text-4xl sm:text-5xl font-black text-amber-400 font-mono block">
              42%
            </span>
            <span className="text-xs font-mono text-white/60 mt-2 block uppercase tracking-wider">
              Reduction in Dispatch Prep Time
            </span>
          </div>
          <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 text-center">
            <span className="text-4xl sm:text-5xl font-black text-emerald-400 font-mono block">
              19%
            </span>
            <span className="text-xs font-mono text-white/60 mt-2 block uppercase tracking-wider">
              Fuel Savings via CVRP Routing
            </span>
          </div>
          <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 text-center">
            <span className="text-4xl sm:text-5xl font-black text-blue-400 font-mono block">
              &lt; 100ms
            </span>
            <span className="text-xs font-mono text-white/60 mt-2 block uppercase tracking-wider">
              Realtime State Synchronization
            </span>
          </div>
        </section>

        {/* 5 Automation Pillars */}
        <section className="mt-24">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-amber-400">
              Automation Modules
            </span>
            <h2 className="text-3xl sm:text-4xl font-black uppercase tracking-tight text-white mt-2">
              Five Automated Engines Powering Pegasus
            </h2>
            <p className="text-sm text-white/60 font-light mt-2 max-w-2xl">
              From morning route calculation to evening treasury closure, every operational step is governed by deterministic business logic.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400 mb-6">
                <Route className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                1. Automated CVRP Routing
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Mathematical multi-stop route optimization balancing vehicle weight limits, cubic space, customer time windows, and traffic patterns with Google OR-Tools.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 mb-6">
                <ScanLine className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                2. Automated Staging & Gate Check
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Digital barcode seals verify every pallet and prevent outbound truck dispatch errors before vehicles exit the loading terminal gate.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 mb-6">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                3. Automated Geofence Status
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Live vehicle telemetry triggers automatic arrival status, notifying receiving managers and preparing loading bays the moment a truck enters the perimeter.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400 mb-6">
                <CheckCircle2 className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                4. Automated Proof of Delivery
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Barcode scan verification at the retailer counter captures item-level receipt, glass signature, and photo validation with instant cloud sync.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400 mb-6">
                <Receipt className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                5. Automated Treasury Balancing
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Point-of-delivery cash collection, commercial trade credit, and dispute offsets are immediately posted to the general ledger with zero manual reconciliation.
              </p>
            </div>

            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-colors">
              <div className="w-12 h-12 rounded-xl bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400 mb-6">
                <RefreshCw className="w-6 h-6" />
              </div>
              <h3 className="text-lg font-bold uppercase text-white mb-2">
                Human Override Architecture
              </h3>
              <p className="text-xs text-white/70 leading-relaxed font-light">
                Automation serves the floor. Dispatchers and warehouse managers retain 1-click manual override capability for rush orders and sick driver substitutions.
              </p>
            </div>
          </div>
        </section>

        {/* Logistics Automation FAQ */}
        <section className="mt-24">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-amber-400">
              Automation FAQ
            </span>
            <h2 className="text-3xl font-black uppercase tracking-tight text-white mt-2">
              Logistics Automation Platform FAQ
            </h2>
          </div>

          <div className="space-y-6">
            {LOGISTICS_AUTOMATION_FAQS.map((faq, idx) => (
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
            Automate Your Logistics Operations Today
          </h2>
          <p className="mt-4 text-base sm:text-lg text-white/70 max-w-2xl mx-auto font-light">
            Deploy high-velocity route optimization, automated gate control, and real-time treasury settlement on the Pegasus platform.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/join"
              className="px-8 py-4 rounded-xl bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors shadow-xl"
            >
              Request Live Automation Demo
            </Link>
            <Link
              href="/contact"
              className="px-8 py-4 rounded-xl bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Talk with an Automation Engineer
            </Link>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
