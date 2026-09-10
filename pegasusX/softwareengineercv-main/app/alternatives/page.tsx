import type { Metadata } from 'next';
import Link from 'next/link';
import { COMPETITORS_DATA } from '@/app/data/competitorsData';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { breadcrumbJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { Check, X, Minus, ArrowRight, Shield, Zap, RefreshCw, Cpu } from 'lucide-react';
import DossierHero from '@/app/components/dossier/DossierHero';
import { DOSSIER_PAGE_CONFIGS } from '@/app/components/dossier/dossierPageConfigs';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  return pageMetadata({
    title: lang === 'ru' 
      ? 'Лучшие альтернативы TMS и систем диспетчеризации (2026)' 
      : 'Best TMS & Fleet Management Alternatives (2026)',
    description: lang === 'ru'
      ? 'Сравнение ведущих систем управления транспортом и диспетчеризации автопарка. Выберите лучшую альтернативу Samsara, Rose Rocket, Motive и Turvo.'
      : 'Compare the top transportation management systems and fleet dispatch software. Find the best alternative to Samsara, Rose Rocket, Motive, and Turvo.',
    path: '/alternatives',
    language: lang,
  });
}

export default async function AlternativesPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const breadcrumbs = [
    { name: isRu ? 'Главная' : 'Home', path: '/' },
    { name: isRu ? 'Альтернативы TMS' : 'TMS Alternatives', path: '/alternatives' },
  ];

  return (
    <div className="min-h-screen bg-[#070709] text-white flex flex-col font-sans selection:bg-white/20">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(breadcrumbJsonLd(breadcrumbs))}
      />
      <SiteNav />

      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-28 pb-24 w-full">
        {/* Dossier Hero Section */}
        <div className="mb-14">
          <DossierHero
            {...DOSSIER_PAGE_CONFIGS['alternatives']}
            tabLabel={isRu ? 'PEGASUS / АЛЬТЕРНАТИВЫ' : 'PEGASUS / ALTERNATIVES'}
            className="!px-0"
          />
        </div>

        {/* Evaluation Pillars */}
        <section className="mt-16 grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="p-6 rounded-xl bg-white/[0.03] border border-white/10">
            <div className="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center mb-4">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <h3 className="text-lg font-bold uppercase tracking-wide mb-2">
              {isRu ? 'Независимость от "железа"' : 'Hardware Independence'}
            </h3>
            <p className="text-sm text-white/60 leading-relaxed font-light">
              {isRu
                ? 'Избегайте многолетней аренды проприетарных черных ящиков. Современные системы работают на смартфонах водителей и открытых датчиках.'
                : 'Avoid 36-60 month proprietary black-box hardware leases. Modern platforms leverage driver mobile devices and open telematics APIs.'}
            </p>
          </div>

          <div className="p-6 rounded-xl bg-white/[0.03] border border-white/10">
            <div className="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center mb-4">
              <RefreshCw className="w-5 h-5 text-white" />
            </div>
            <h3 className="text-lg font-bold uppercase tracking-wide mb-2">
              {isRu ? 'Сквозной B2B контур (6 ролей)' : 'Full 6-Role Closed Loop'}
            </h3>
            <p className="text-sm text-white/60 leading-relaxed font-light">
              {isRu
                ? 'Объединение поставщика, склада, фабрики, водителя, ритейлера и ворот в единой системе состояний с нулевыми потерями данных.'
                : 'Connect suppliers, warehouses, factories, drivers, retailers, and security gates into one shared transactional state without dropped handoffs.'}
            </p>
          </div>

          <div className="p-6 rounded-xl bg-white/[0.03] border border-white/10">
            <div className="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center mb-4">
              <Shield className="w-5 h-5 text-white" />
            </div>
            <h3 className="text-lg font-bold uppercase tracking-wide mb-2">
              {isRu ? 'Финансовая сверка' : 'Financial Reconciliation'}
            </h3>
            <p className="text-sm text-white/60 leading-relaxed font-light">
              {isRu
                ? 'Прямая сверка наложенных платежей (COD), банковских переводов и доказательств доставки в казначействе день в день.'
                : 'Same-day point-of-delivery cash-on-delivery reconciliation and instant credit limit validation directly within the operations core.'}
            </p>
          </div>
        </section>

        {/* Competitor Cards Grid */}
        <section className="mt-20">
          <div className="flex items-center justify-between mb-8 pb-4 border-b border-white/10">
            <div>
              <h2 className="text-2xl font-black uppercase tracking-tight text-white">
                {isRu ? 'Обзор платформ и альтернатив' : 'Evaluated Software Platforms'}
              </h2>
              <p className="text-sm text-white/50 font-mono mt-1">
                {isRu ? 'Сравнение архитектур, цен и сценариев применения' : 'Architectural breakdown, pricing models, and key tradeoffs'}
              </p>
            </div>
            <Link
              href="/compare"
              className="text-xs font-mono tracking-wider uppercase text-white/70 hover:text-white transition-colors flex items-center gap-1.5"
            >
              {isRu ? 'Сравнение лицом к лицу →' : 'Head-to-head guides →'}
            </Link>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {COMPETITORS_DATA.map((competitor) => (
              <article
                key={competitor.slug}
                className="p-8 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/20 transition-all flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-start justify-between gap-4 mb-4">
                    <div>
                      <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40">
                        {competitor.category}
                      </span>
                      <h3 className="text-2xl font-bold uppercase tracking-tight text-white mt-1">
                        {competitor.name}
                      </h3>
                    </div>
                    <span className="text-xs font-mono px-2.5 py-1 rounded bg-white/5 border border-white/10 text-white/80">
                      {competitor.hardwareRequired ? (isRu ? 'Требует датчики' : 'Hardware required') : (isRu ? 'Только софт' : 'Software-first')}
                    </span>
                  </div>

                  <p className="text-sm text-white/70 font-light mb-6">
                    {competitor.tagline} — {competitor.marketPosition}.
                  </p>

                  <div className="space-y-3 mb-6 pb-6 border-b border-white/5">
                    <p className="text-xs font-mono uppercase tracking-wider text-white/40">
                      {isRu ? 'Почему переходят на Pegasus:' : 'Why teams switch to Pegasus:'}
                    </p>
                    <ul className="space-y-2">
                      {competitor.whySwitchToPegasus.slice(0, 3).map((reason, idx) => (
                        <li key={idx} className="flex items-start gap-2.5 text-xs text-white/80 font-light">
                          <Check className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                          <span>{reason}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>

                <div className="flex flex-wrap items-center justify-between gap-4 pt-2">
                  <div>
                    <span className="text-[10px] font-mono text-white/40 block">
                      {isRu ? 'Оценка стоимости:' : 'Pricing estimate:'}
                    </span>
                    <span className="text-xs font-mono text-white/80">{competitor.pricingEstimate}</span>
                  </div>

                  <div className="flex items-center gap-3">
                    <Link
                      href={`/compare/pegasus-vs-${competitor.slug}`}
                      className="text-xs font-mono text-white/60 hover:text-white transition-colors"
                    >
                      {isRu ? 'Vs Pegasus' : 'Vs Pegasus'}
                    </Link>
                    <Link
                      href={`/alternatives/${competitor.slug}`}
                      className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-white text-black text-xs font-bold uppercase tracking-wider hover:bg-white/90 transition-colors"
                    >
                      <span>{isRu ? 'Полный анализ' : 'Alternative guide'}</span>
                      <ArrowRight className="w-3.5 h-3.5" />
                    </Link>
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>

        {/* Comparison Summary Table */}
        <section className="mt-24">
          <div className="mb-8">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              {isRu ? 'Сводная матрица возможностей' : 'Feature Comparison Matrix'}
            </h2>
            <p className="text-sm text-white/50 font-mono mt-1">
              {isRu ? 'Технические различия по ключевым модулям' : 'Technical capabilities across critical B2B logistics layers'}
            </p>
          </div>

          <div className="overflow-x-auto border border-white/10 rounded-xl bg-white/[0.01]">
            <table className="w-full text-left text-xs font-mono">
              <thead>
                <tr className="border-b border-white/10 bg-white/5 text-white/60">
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'Платформа' : 'Platform'}</th>
                  <th className="p-4 uppercase tracking-wider">{isRu ? '6-ролевая сеть' : '6-Role Network'}</th>
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'Без "железа"' : 'Hardware Free'}</th>
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'B2B Платежи' : 'B2B Payments'}</th>
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'Ворота/Терминал' : 'Gate Terminal'}</th>
                  <th className="p-4 uppercase tracking-wider">{isRu ? 'Маршрутизация CVRP' : 'CVRP Optimization'}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-white/80">
                <tr className="bg-white/10 font-bold text-white">
                  <td className="p-4 flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-400" />
                    Pegasus TMS
                  </td>
                  <td className="p-4 text-emerald-400">Full (6 roles)</td>
                  <td className="p-4 text-emerald-400">Yes (Mobile apps)</td>
                  <td className="p-4 text-emerald-400">Full (COD + Audit)</td>
                  <td className="p-4 text-emerald-400">Yes (Native app)</td>
                  <td className="p-4 text-emerald-400">Google OR-Tools</td>
                </tr>
                {COMPETITORS_DATA.map((c) => (
                  <tr key={c.slug} className="hover:bg-white/[0.02]">
                    <td className="p-4 font-semibold text-white/90">{c.name}</td>
                    <td className="p-4">
                      {c.featureRatings.multiRoleDispatch.rating === 'full' ? (
                        <span className="text-emerald-400">Full</span>
                      ) : c.featureRatings.multiRoleDispatch.rating === 'partial' ? (
                        <span className="text-amber-400">Partial</span>
                      ) : (
                        <span className="text-rose-400">None</span>
                      )}
                    </td>
                    <td className="p-4">
                      {c.featureRatings.hardwareIndependence.rating === 'full' ? (
                        <span className="text-emerald-400">Yes</span>
                      ) : (
                        <span className="text-rose-400">Hardware locked</span>
                      )}
                    </td>
                    <td className="p-4">
                      {c.featureRatings.integratedB2BPayments.rating === 'full' ? (
                        <span className="text-emerald-400">Full</span>
                      ) : c.featureRatings.integratedB2BPayments.rating === 'partial' ? (
                        <span className="text-amber-400">Partial</span>
                      ) : (
                        <span className="text-rose-400">None</span>
                      )}
                    </td>
                    <td className="p-4">
                      {c.featureRatings.warehouseGateTerminal.rating === 'full' ? (
                        <span className="text-emerald-400">Yes</span>
                      ) : c.featureRatings.warehouseGateTerminal.rating === 'partial' ? (
                        <span className="text-amber-400">Partial</span>
                      ) : (
                        <span className="text-rose-400">None</span>
                      )}
                    </td>
                    <td className="p-4">
                      {c.featureRatings.routeOptimization.rating === 'full' ? (
                        <span className="text-emerald-400">Advanced</span>
                      ) : c.featureRatings.routeOptimization.rating === 'partial' ? (
                        <span className="text-amber-400">Basic</span>
                      ) : (
                        <span className="text-rose-400">None</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* CTA */}
        <div className="mt-24 p-10 rounded-2xl bg-gradient-to-b from-white/10 to-white/5 border border-white/15 text-center">
          <h2 className="text-3xl font-black uppercase tracking-tight text-white">
            {isRu ? 'Готовы оптимизировать диспетчеризацию?' : 'Ready to upgrade your logistics operations?'}
          </h2>
          <p className="mt-3 text-sm text-white/70 max-w-xl mx-auto font-light leading-relaxed">
            {isRu
              ? 'Запросите демонстрацию Pegasus и узнайте, как связать поставщиков, склады, фабрики, водителей и ритейлеров без лишних затрат на оборудование.'
              : 'Book a tailored demonstration of Pegasus to see how unified multi-role dispatch and automated reconciliation transform physical goods distribution.'}
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/join"
              className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors"
            >
              {isRu ? 'Запросить демо' : 'Request Live Demo'}
            </Link>
            <Link
              href="/platform"
              className="px-6 py-3 rounded-lg bg-white/10 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/15 transition-colors border border-white/20"
            >
              {isRu ? 'Обзор платформы' : 'Explore Platform'}
            </Link>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}
