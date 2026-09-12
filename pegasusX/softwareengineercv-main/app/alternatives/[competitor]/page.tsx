import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import Link from 'next/link';
import { COMPETITORS_DATA, getCompetitorBySlug } from '@/app/data/competitorsData';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { breadcrumbJsonLd, faqPageJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { Check, X, ArrowRight, ShieldCheck, HelpCircle, Layers, ArrowLeftRight } from 'lucide-react';

export function generateStaticParams() {
  return COMPETITORS_DATA.map((c) => ({ competitor: c.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ competitor: string }>;
}): Promise<Metadata> {
  const { competitor: slug } = await params;
  const competitor = getCompetitorBySlug(slug);
  if (!competitor) return {};

  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? `${competitor.name}: альтернатива для B2B логистики | Pegasus TMS`
    : `${competitor.name} Alternative for B2B Fleet Dispatch | Pegasus TMS`;

  const description = isRu
    ? `Ищете замену ${competitor.name}? Откройте для себя Pegasus: сквозная диспетчеризация, живой мониторинг и платежи без привязки к оборудованию.`
    : `Looking for a ${competitor.name} alternative without hardware lock-in? Discover Pegasus for multi-role dispatch, fleet tracking, and automated payments.`;

  return pageMetadata({
    title,
    description: description.slice(0, 160),
    path: `/alternatives/${slug}`,
    language: lang,
  });
}

export default async function CompetitorAlternativePage({
  params,
}: {
  params: Promise<{ competitor: string }>;
}) {
  const { competitor: slug } = await params;
  const competitor = getCompetitorBySlug(slug);
  if (!competitor) notFound();

  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const breadcrumbs = [
    { name: isRu ? 'Главная' : 'Home', path: '/' },
    { name: isRu ? 'Альтернативы' : 'Alternatives', path: '/alternatives' },
    { name: `${competitor.name} Alternative`, path: `/alternatives/${slug}` },
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
        {/* Navigation Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/alternatives" className="hover:text-white transition-colors">
            {isRu ? '← Все альтернативы' : '← All Alternatives'}
          </Link>
          <span>/</span>
          <span className="text-white/80">{competitor.name}</span>
        </div>

        {/* Hero Section */}
        <div>
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <ArrowLeftRight className="w-3.5 h-3.5 text-white" />
            {isRu ? `Сравнение: ${competitor.name} против Pegasus` : `${competitor.name} vs. Pegasus`}
          </div>

          <h1 className="text-3xl sm:text-5xl lg:text-6xl font-black tracking-tight text-white uppercase">
            {isRu ? `Лучшая альтернатива ${competitor.name}` : `The Leading ${competitor.name} Alternative`}
          </h1>

          <p className="mt-6 text-lg sm:text-xl text-white/70 leading-relaxed font-light">
            {isRu
              ? `Устали от ограничений ${competitor.name}? Узнайте, почему операторы и поставщики переходят на Pegasus для сквозного управления автопарком, складом и расчетами.`
              : `Evaluating options beyond ${competitor.name}? Discover why private fleets, manufacturers, and regional distributors choose Pegasus for modern, hardware-independent logistics orchestration.`}
          </p>

          <div className="mt-8 flex flex-wrap gap-4">
            <Link
              href="/join"
              className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors"
            >
              {isRu ? 'Запросить демо' : 'Request Migration Demo'}
            </Link>
            <Link
              href={`/compare/pegasus-vs-${competitor.slug}`}
              className="px-6 py-3 rounded-lg bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              {isRu ? 'Таблица сравнения' : 'View Head-to-Head'}
            </Link>
          </div>
        </div>

        {/* TL;DR Summary Box */}
        <section className="mt-16 p-8 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <Layers className="w-4 h-4 text-white" />
            <span>TL;DR Executive Summary</span>
          </div>
          <p className="text-base text-white/90 font-light leading-relaxed">
            <strong className="font-semibold text-white">{competitor.name}</strong>{' '}
            {competitor.marketPosition.toLowerCase()}. However, {competitor.limitations[0].toLowerCase()}.{' '}
            <strong className="font-semibold text-white">Pegasus</strong> delivers an end-to-end B2B logistics operating system connecting 6 roles (suppliers, warehouses, factories, drivers, retailers, gate security) with zero proprietary hardware lock-in.
          </p>
        </section>

        {/* Why Switch Section */}
        <section className="mt-16">
          <h2 className="text-2xl font-black uppercase tracking-tight text-white mb-6">
            {isRu ? `Почему операторы переходят с ${competitor.name} на Pegasus:` : `Key Reasons Teams Switch from ${competitor.name}:`}
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {competitor.whySwitchToPegasus.map((reason, idx) => (
              <div
                key={idx}
                className="p-6 rounded-xl bg-white/[0.02] border border-white/10 flex items-start gap-3"
              >
                <div className="w-6 h-6 rounded-full bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center shrink-0 mt-0.5">
                  <Check className="w-3.5 h-3.5 text-emerald-400" />
                </div>
                <p className="text-sm text-white/80 font-light leading-relaxed">{reason}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Objective Comparison Table */}
        <section className="mt-20">
          <div className="mb-6">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              {isRu ? 'Детальное сравнение возможностей' : 'Feature-by-Feature Comparison'}
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              {isRu ? 'Честный анализ возможностей без маркетинговых штампов' : 'Transparent evaluation based on verified operational architectures'}
            </p>
          </div>

          <div className="overflow-x-auto border border-white/10 rounded-xl bg-white/[0.01]">
            <table className="w-full text-left text-xs font-mono">
              <thead>
                <tr className="border-b border-white/10 bg-white/5 text-white/60">
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'Возможность' : 'Capability'}</th>
                  <th className="p-4 uppercase tracking-wider text-white">Pegasus TMS</th>
                  <th className="p-4 uppercase tracking-wider">{competitor.name}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-white/80">
                <tr>
                  <td className="p-4 font-semibold text-white">6-Role Ecosystem (Supplier, Warehouse, Driver, Retailer, Gate)</td>
                  <td className="p-4 text-emerald-400">Full native support across all 6 roles</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.multiRoleDispatch.note}</td>
                </tr>
                <tr>
                  <td className="p-4 font-semibold text-white">Hardware Requirements</td>
                  <td className="p-4 text-emerald-400">Hardware-agnostic (iOS, Android, open GPS)</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.hardwareIndependence.note}</td>
                </tr>
                <tr>
                  <td className="p-4 font-semibold text-white">B2B Trade Credit & Driver Cash Reconciliation</td>
                  <td className="p-4 text-emerald-400">Integrated double-entry treasury ledger</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.integratedB2BPayments.note}</td>
                </tr>
                <tr>
                  <td className="p-4 font-semibold text-white">Warehouse Gate Terminal & Seal Verification</td>
                  <td className="p-4 text-emerald-400">Native yard & gate security terminal</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.warehouseGateTerminal.note}</td>
                </tr>
                <tr>
                  <td className="p-4 font-semibold text-white">Route Optimization & Sequencing</td>
                  <td className="p-4 text-emerald-400">Google OR-Tools multi-capacity CVRP engine</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.routeOptimization.note}</td>
                </tr>
                <tr>
                  <td className="p-4 font-semibold text-white">Real-Time State Synchronization</td>
                  <td className="p-4 text-emerald-400">Google Cloud Spanner + WebSockets (&lt;100ms)</td>
                  <td className="p-4 text-white/60">{competitor.featureRatings.realtimeEventSync.note}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Who Should Choose Whom */}
        <section className="mt-20 grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10">
            <h3 className="text-lg font-bold uppercase tracking-wide text-white mb-4">
              {isRu ? `Кому подходит ${competitor.name}:` : `Who Should Choose ${competitor.name}:`}
            </h3>
            <ul className="space-y-3">
              {competitor.idealFor.map((item, idx) => (
                <li key={idx} className="flex items-start gap-2.5 text-xs text-white/70 font-light">
                  <Check className="w-3.5 h-3.5 text-white/40 shrink-0 mt-0.5" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="p-8 rounded-2xl bg-white/[0.04] border border-white/20">
            <h3 className="text-lg font-bold uppercase tracking-wide text-white mb-4">
              {isRu ? 'Кому подходит Pegasus:' : 'Who Should Choose Pegasus:'}
            </h3>
            <ul className="space-y-3">
              <li className="flex items-start gap-2.5 text-xs text-white/80 font-light">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Suppliers and manufacturers operating their own private truck fleets and warehouses</span>
              </li>
              <li className="flex items-start gap-2.5 text-xs text-white/80 font-light">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Companies needing seamless B2B retailer ordering and live order vetting</span>
              </li>
              <li className="flex items-start gap-2.5 text-xs text-white/80 font-light">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Operations seeking zero hardware lock-in and rapid cloud deployment</span>
              </li>
              <li className="flex items-start gap-2.5 text-xs text-white/80 font-light">
                <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                <span>Logistics teams requiring driver cash-on-delivery reconciliation with warehouse treasury</span>
              </li>
            </ul>
          </div>
        </section>

        {/* Migration Path */}
        <section className="mt-20 p-8 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-4">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <span>{isRu ? 'План миграции' : 'Migration & Onboarding Roadmap'}</span>
          </div>
          <h3 className="text-xl font-bold uppercase tracking-tight text-white mb-3">
            {isRu ? `Переход с ${competitor.name} за ${competitor.migrationNotes.timeframe}` : `Switch from ${competitor.name} in ${competitor.migrationNotes.timeframe}`}
          </h3>
          <p className="text-sm text-white/70 font-light leading-relaxed mb-6">
            {competitor.migrationNotes.migrationSupport}
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="p-4 rounded-lg bg-white/5 border border-white/5">
              <span className="text-[10px] font-mono text-white/40 block uppercase">
                {isRu ? 'Переносимые данные' : 'Transferred Data'}
              </span>
              <span className="text-xs font-mono text-white/80 mt-1 block">
                {competitor.migrationNotes.transferredData.join(', ')}
              </span>
            </div>
            <div className="p-4 rounded-lg bg-white/5 border border-white/5">
              <span className="text-[10px] font-mono text-white/40 block uppercase">
                {isRu ? 'Сложность перехода' : 'Migration Complexity'}
              </span>
              <span className="text-xs font-mono text-emerald-400 mt-1 block font-bold">
                {competitor.migrationNotes.difficulty} Complexity ({competitor.migrationNotes.timeframe})
              </span>
            </div>
          </div>
        </section>

        {/* FAQs */}
        <section className="mt-20">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <HelpCircle className="w-4 h-4 text-white" />
            <span>{isRu ? 'Часто задаваемые вопросы' : 'Frequently Asked Questions'}</span>
          </div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white mb-8">
            {isRu ? `Вопросы о замене ${competitor.name}` : `Common Questions: Switching from ${competitor.name}`}
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
            {isRu ? `Готовы перейти с ${competitor.name}?` : `Ready to switch from ${competitor.name}?`}
          </h2>
          <p className="mt-3 text-sm text-white/70 max-w-xl mx-auto font-light leading-relaxed">
            {isRu
              ? 'Закажите персональный обзор миграции и узнайте, как быстро перенести данные автопарка и настроить систему под ваш бизнес.'
              : 'Schedule a tailored migration consultation to evaluate timelines, data transfer, and cost savings for your fleet.'}
          </p>
          <div className="mt-8 flex justify-center gap-4">
            <Link
              href="/join"
              className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors"
            >
              {isRu ? 'Запросить консультацию' : 'Schedule Migration Call'}
            </Link>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}
