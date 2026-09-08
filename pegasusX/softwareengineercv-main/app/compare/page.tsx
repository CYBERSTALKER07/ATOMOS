import type { Metadata } from 'next';
import Link from 'next/link';
import { COMPETITORS_DATA } from '@/app/data/competitorsData';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { breadcrumbJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { ArrowRight, GitCompare, ShieldCheck } from 'lucide-react';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  return pageMetadata({
    title: lang === 'ru'
      ? 'Сравнение TMS и систем управления автопарком | Pegasus'
      : 'TMS Software Comparison: Head-to-Head Guides | Pegasus',
    description: lang === 'ru'
      ? 'Детальные сравнения Pegasus с Samsara, Rose Rocket, Motive, Turvo и Onfleet. Честный разбор архитектуры, оборудования и цен.'
      : 'Compare leading transportation management systems side by side. Unbiased breakdowns of features, pricing, architecture, and deployment workflows.',
    path: '/compare',
    language: lang,
  });
}

export default async function CompareHubPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const breadcrumbs = [
    { name: isRu ? 'Главная' : 'Home', path: '/' },
    { name: isRu ? 'Сравнения' : 'Comparisons', path: '/compare' },
  ];

  return (
    <div className="min-h-screen bg-[#070709] text-white flex flex-col font-sans selection:bg-white/20">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(breadcrumbJsonLd(breadcrumbs))}
      />
      <SiteNav />

      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Header */}
        <div className="max-w-3xl">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <GitCompare className="w-3.5 h-3.5 text-white" />
            {isRu ? 'Аналитика и сравнение логистического ПО' : 'Objective Software Analysis'}
          </div>
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-black tracking-tight text-white uppercase">
            {isRu ? 'Сравнение систем управления логистикой' : 'Head-to-Head TMS Comparisons'}
          </h1>
          <p className="mt-6 text-lg sm:text-xl text-white/70 leading-relaxed font-light">
            {isRu
              ? 'Честное и прямое сравнение Pegasus с ведущими телематическими, брокерскими и экспедиторскими платформами на рынке.'
              : 'Direct, architectural comparisons of Pegasus against leading telematics, brokerage, and dispatch platforms. Learn which platform aligns with your operational profile.'}
          </p>
        </div>

        {/* Comparison Cards */}
        <div className="mt-16 grid grid-cols-1 md:grid-cols-2 gap-6">
          {COMPETITORS_DATA.map((competitor) => (
            <div
              key={competitor.slug}
              className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/25 transition-all flex flex-col justify-between"
            >
              <div>
                <div className="flex items-center justify-between mb-4">
                  <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40">
                    {competitor.category}
                  </span>
                  <span className="text-xs font-mono text-emerald-400">
                    Pegasus vs. {competitor.name}
                  </span>
                </div>

                <h3 className="text-2xl font-bold uppercase tracking-tight text-white mb-2">
                  Pegasus vs {competitor.name}
                </h3>
                <p className="text-sm text-white/70 font-light leading-relaxed mb-6">
                  {competitor.tagline}. {isRu ? 'Сравните ключевые отличия в оборудовании, диспетчеризации и расчетах.' : 'Compare critical differences in hardware lock-in, multi-role dispatch, and financial reconciliation.'}
                </p>
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-white/5">
                <span className="text-xs font-mono text-white/50">
                  {competitor.hardwareRequired ? 'Hardware Required' : 'Software-First'}
                </span>
                <Link
                  href={`/compare/pegasus-vs-${competitor.slug}`}
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-white text-black text-xs font-bold uppercase tracking-wider hover:bg-white/90 transition-colors"
                >
                  <span>{isRu ? 'Читать сравнение' : 'Read Guide'}</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </Link>
              </div>
            </div>
          ))}
        </div>

        {/* Methodology Note */}
        <section className="mt-20 p-8 rounded-2xl bg-white/[0.02] border border-white/10">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <ShieldCheck className="w-4 h-4 text-white" />
            <span>{isRu ? 'Наша методология' : 'Our Comparison Methodology'}</span>
          </div>
          <p className="text-sm text-white/70 font-light leading-relaxed">
            {isRu
              ? 'Мы верим, что честность строит доверие. Мы открыто указываем сильные стороны каждого конкурента (например, зрелость камер безопасности Samsara или брокерский интерфейс Rose Rocket) и четко разграничиваем, кому подходит каждый инструмент.'
              : 'We believe honesty builds lasting partnerships. We acknowledge where competitors excel (such as Samsara’s mature dashcam coaching or Rose Rocket’s freight brokerage workflows) and clearly articulate when an alternative may be a better fit than Pegasus.'}
          </p>
        </section>
      </main>

      <Footer />
    </div>
  );
}
