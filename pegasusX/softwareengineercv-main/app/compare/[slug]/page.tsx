import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import Link from 'next/link';
import { COMPETITORS_DATA } from '@/app/data/competitorsData';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { breadcrumbJsonLd, faqPageJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { Check, X, ArrowRight, Layers, HelpCircle, ArrowLeftRight } from 'lucide-react';

export function generateStaticParams() {
  return COMPETITORS_DATA.map((c) => ({ slug: `pegasus-vs-${c.slug}` }));
}

function extractCompetitor(slug: string) {
  const competitorSlug = slug.replace(/^pegasus-vs-/, '');
  return COMPETITORS_DATA.find((c) => c.slug === competitorSlug);
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const competitor = extractCompetitor(slug);
  if (!competitor) return {};

  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? `Pegasus vs ${competitor.name}: Сравнение систем логистики 2026`
    : `Pegasus vs ${competitor.name}: 2026 Fleet & TMS Comparison`;

  const description = isRu
    ? `Подробное сравнение Pegasus и ${competitor.name}. Сравните зависимость от оборудования, оркестрацию диспетчеризации и стоимость владения.`
    : `Detailed comparison of Pegasus and ${competitor.name}. Compare hardware dependency, dispatch orchestration, multi-role access, and total cost of ownership.`;

  return pageMetadata({
    title,
    description: description.slice(0, 160),
    path: `/compare/${slug}`,
    language: lang,
  });
}

export default async function HeadToHeadComparePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const competitor = extractCompetitor(slug);
  if (!competitor) notFound();

  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const breadcrumbs = [
    { name: isRu ? 'Главная' : 'Home', path: '/' },
    { name: isRu ? 'Сравнения' : 'Comparisons', path: '/compare' },
    { name: `Pegasus vs ${competitor.name}`, path: `/compare/${slug}` },
  ];

  return (
    <div className="min-h-screen bg-[#070709] text-white flex flex-col font-sans selection:bg-white/20">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(breadcrumbJsonLd(breadcrumbs))}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(faqPageJsonLd(competitor.faqs, lang))}
      />
      <SiteNav />

      <main className="flex-1 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/compare" className="hover:text-white transition-colors">
            {isRu ? '← Все сравнения' : '← All Comparisons'}
          </Link>
          <span>/</span>
          <span className="text-white/80">Pegasus vs. {competitor.name}</span>
        </div>

        {/* Hero */}
        <div>
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <ArrowLeftRight className="w-3.5 h-3.5 text-white" />
            {isRu ? 'Сравнение лицом к лицу' : 'Direct Comparison'}
          </div>

          <h1 className="text-3xl sm:text-5xl lg:text-6xl font-black tracking-tight text-white uppercase">
            Pegasus vs. {competitor.name}
          </h1>

          <p className="mt-6 text-lg sm:text-xl text-white/70 leading-relaxed font-light">
            {isRu
              ? `Технический и операционный разбор: какая платформа лучше подходит для задач вашего автопарка, склада и поставок?`
              : `A technical and architectural evaluation: how Pegasus compares to ${competitor.name} across dispatch optimization, hardware dependency, and multi-role operations.`}
          </p>
        </div>

        {/* TL;DR */}
        <section className="mt-14 p-8 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <Layers className="w-4 h-4 text-white" />
            <span>At-a-Glance Verdict</span>
          </div>
          <p className="text-base text-white/90 font-light leading-relaxed">
            <strong className="font-semibold text-white">{competitor.name}</strong> is built for{' '}
            {competitor.idealFor[0].toLowerCase()}, with strong focus on {competitor.strengths[0].toLowerCase()}.{' '}
            <strong className="font-semibold text-white">Pegasus</strong> is an end-to-end logistics operating system built for supplier networks and private fleets, providing native apps across 6 operational roles without proprietary hardware leases.
          </p>
        </section>

        {/* Side by side overview */}
        <section className="mt-16 grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="p-8 rounded-2xl bg-white/[0.04] border border-white/20">
            <span className="text-[10px] font-mono text-emerald-400 uppercase tracking-widest block mb-2">
              The Sovereign Solution
            </span>
            <h3 className="text-2xl font-bold uppercase text-white mb-4">Pegasus TMS</h3>
            <ul className="space-y-3 text-xs text-white/80 font-light">
              <li className="flex items-start gap-2.5">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Zero proprietary hardware lock-in (BYOD mobile apps & open GPS)</span>
              </li>
              <li className="flex items-start gap-2.5">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>6 dedicated roles: Supplier, Warehouse, Factory, Driver, Retailer, Gate</span>
              </li>
              <li className="flex items-start gap-2.5">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Integrated B2B trade terms, driver cash collection & treasury reconciliation</span>
              </li>
              <li className="flex items-start gap-2.5">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Google OR-Tools CVRP capacity-balanced route optimization</span>
              </li>
            </ul>
          </div>

          <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10">
            <span className="text-[10px] font-mono text-white/40 uppercase tracking-widest block mb-2">
              Industry Incumbent
            </span>
            <h3 className="text-2xl font-bold uppercase text-white mb-4">{competitor.name}</h3>
            <ul className="space-y-3 text-xs text-white/70 font-light">
              {competitor.strengths.map((str, idx) => (
                <li key={idx} className="flex items-start gap-2.5">
                  <Check className="w-3.5 h-3.5 text-white/40 shrink-0 mt-0.5" />
                  <span>{str}</span>
                </li>
              ))}
              {competitor.limitations.slice(0, 2).map((lim, idx) => (
                <li key={idx} className="flex items-start gap-2.5 text-rose-300/80">
                  <X className="w-3.5 h-3.5 text-rose-400 shrink-0 mt-0.5" />
                  <span>{lim}</span>
                </li>
              ))}
            </ul>
          </div>
        </section>

        {/* Detailed Breakdown by Category */}
        <section className="mt-20 space-y-12">
          <div>
            <h2 className="text-2xl font-black uppercase tracking-tight text-white mb-6">
              Category Breakdown
            </h2>

            {/* Category 1: Hardware & Infrastructure */}
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 mb-6">
              <h3 className="text-lg font-bold uppercase text-white mb-3">1. Hardware & Infrastructure</h3>
              <p className="text-sm text-white/70 font-light leading-relaxed mb-4">
                <strong>{competitor.name}:</strong> {competitor.hardwareRequired ? 'Relies on proprietary vehicle hardware installations and multi-year lease agreements, making fleet updates capital-intensive.' : 'Cloud-based platform with third-party telematics ingestion.'}
              </p>
              <p className="text-sm text-white/70 font-light leading-relaxed">
                <strong>Pegasus:</strong> Completely hardware-independent. Drivers use native iOS/Android apps for high-accuracy telemetry, breadcrumb navigation, and electronic proof of delivery. Existing commercial GPS feeds ingest directly via open REST and Kafka pipelines.
              </p>
            </div>

            {/* Category 2: Operational Breadth */}
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 mb-6">
              <h3 className="text-lg font-bold uppercase text-white mb-3">2. Multi-Role Ecosystem & Staging</h3>
              <p className="text-sm text-white/70 font-light leading-relaxed mb-4">
                <strong>{competitor.name}:</strong> Primarily built for {competitor.marketPosition.toLowerCase()}. Lacks synchronized surfaces for warehouse dock staging, factory loading lanes, or security gate terminals.
              </p>
              <p className="text-sm text-white/70 font-light leading-relaxed">
                <strong>Pegasus:</strong> Features purpose-built desktop and mobile applications for suppliers, warehouse dispatchers, factory supervisors, drivers, retail buyers, and gate security agents. All 6 roles coordinate against the same Google Cloud Spanner distributed transactional state.
              </p>
            </div>

            {/* Category 3: Financial Reconciliation */}
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10">
              <h3 className="text-lg font-bold uppercase text-white mb-3">3. Payments & Treasury Integrity</h3>
              <p className="text-sm text-white/70 font-light leading-relaxed mb-4">
                <strong>{competitor.name}:</strong> Financial handling is outside core product scope, requiring separate ERP or accounting reconciliations.
              </p>
              <p className="text-sm text-white/70 font-light leading-relaxed">
                <strong>Pegasus:</strong> Full double-entry financial ledger support. Point-of-delivery invoice adjustments, cash-on-delivery (COD) driver collection locks, and instant treasury matching eliminate disputed invoices and end-of-day cash reconciliation delays.
              </p>
            </div>
          </div>
        </section>

        {/* FAQs */}
        <section className="mt-20">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <HelpCircle className="w-4 h-4 text-white" />
            <span>Frequently Asked Questions</span>
          </div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white mb-8">
            Pegasus vs. {competitor.name} FAQ
          </h2>
          <div className="space-y-4">
            {competitor.faqs.map((faq, idx) => (
              <div key={idx} className="p-6 rounded-xl bg-white/[0.02] border border-white/10">
                <h3 className="text-base font-bold text-white mb-2">{faq.question}</h3>
                <p className="text-sm text-white/70 font-light leading-relaxed">{faq.answer}</p>
              </div>
            ))}
          </div>
        </section>

        {/* CTA */}
        <div className="mt-24 p-10 rounded-2xl bg-gradient-to-b from-white/10 to-white/5 border border-white/15 text-center">
          <h2 className="text-3xl font-black uppercase tracking-tight text-white">
            See the Difference in Action
          </h2>
          <p className="mt-3 text-sm text-white/70 max-w-xl mx-auto font-light leading-relaxed">
            Experience how Pegasus transforms dispatch efficiency, fleet telemetry, and payment reconciliation without long-term hardware contracts.
          </p>
          <div className="mt-8 flex justify-center gap-4">
            <Link
              href="/join"
              className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors"
            >
              Request Live Demo
            </Link>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}
