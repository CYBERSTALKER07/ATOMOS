import type { Metadata } from 'next';
import Link from 'next/link';
import {
  Globe2,
  ArrowRight,
  ShieldCheck,
  Server,
  Zap,
  MapPin,
  ChevronRight,
  CheckCircle2,
} from 'lucide-react';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { MARKETS_DATA } from '@/app/data/marketsData';
import {
  pageMetadata,
  breadcrumbJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const title = isRu
    ? 'Глобальные рынки логистики и коридоры поставок | Pegasus'
    : 'Global Logistics & Supply Chain Markets (2026) | Pegasus';

  const description = isRu
    ? 'Обзор развёртывания логистической операционной системы Pegasus в 16 странах и международных транспортных коридорах с локальными облачными ячейками.'
    : 'Explore Pegasus global logistics operating system deployments across 16 countries and international freight corridors with localized cell clusters.';

  return pageMetadata({
    title,
    description,
    path: '/markets',
    language: lang,
  });
}

export default async function MarketsHubPage() {
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const structuredData = [
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Markets', path: '/markets' },
    ]),
  ];

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-white selection:text-black">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />
      <SiteNav activeHref="/markets" />

      <main className="flex-1 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/" className="hover:text-white transition-colors">
            {isRu ? 'Главная' : 'Home'}
          </Link>
          <ChevronRight className="w-3.5 h-3.5 text-white/30" />
          <span className="text-white/80">Markets</span>
        </div>

        {/* Hero */}
        <div className="border-b border-white/10 pb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <Globe2 className="w-3.5 h-3.5 text-blue-400" />
            <span>Worldwide Coverage · 16 Strategic Corridors</span>
          </div>

          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight text-white uppercase leading-[1.05]">
            Global Logistics & Supply Chain Markets
          </h1>

          <p className="mt-6 text-xl sm:text-2xl text-white/70 leading-relaxed font-light max-w-3xl">
            Sovereign, cloned cell architectures and low-latency cloud clusters delivering real-time logistics automation in any country and language.
          </p>

          <div className="mt-10 flex flex-wrap gap-4">
            <Link
              href="/join"
              className="px-7 py-3.5 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors flex items-center gap-2"
            >
              <span>Request Regional Deployment Demo</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link
              href="/global-logistics"
              className="px-7 py-3.5 rounded-lg bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Global Architecture Overview
            </Link>
          </div>
        </div>

        {/* Regional Cell Clusters Summary */}
        <section className="mt-16 grid grid-cols-1 sm:grid-cols-4 gap-4">
          <div className="p-6 rounded-2xl bg-white/[0.02] border border-white/10">
            <span className="text-[10px] font-mono uppercase tracking-widest text-blue-400 block mb-1">
              Americas
            </span>
            <span className="text-lg font-bold text-white block">cell-us & cell-latam</span>
            <span className="text-xs text-white/50 font-mono mt-1 block">&lt; 15ms continental</span>
          </div>
          <div className="p-6 rounded-2xl bg-white/[0.02] border border-white/10">
            <span className="text-[10px] font-mono uppercase tracking-widest text-emerald-400 block mb-1">
              Europe
            </span>
            <span className="text-lg font-bold text-white block">cell-eu (Frankfurt/Paris/London)</span>
            <span className="text-xs text-white/50 font-mono mt-1 block">&lt; 8ms DACH / EU</span>
          </div>
          <div className="p-6 rounded-2xl bg-white/[0.02] border border-white/10">
            <span className="text-[10px] font-mono uppercase tracking-widest text-amber-400 block mb-1">
              Middle East & Eurasia
            </span>
            <span className="text-lg font-bold text-white block">cell-me & cell-tr</span>
            <span className="text-xs text-white/50 font-mono mt-1 block">&lt; 12ms GCC & Eurasia</span>
          </div>
          <div className="p-6 rounded-2xl bg-white/[0.02] border border-white/10">
            <span className="text-[10px] font-mono uppercase tracking-widest text-purple-400 block mb-1">
              Asia & Silk Road
            </span>
            <span className="text-lg font-bold text-white block">cell-uz & cell-apac</span>
            <span className="text-xs text-white/50 font-mono mt-1 block">2ms TAS-IX / 5ms SG</span>
          </div>
        </section>

        {/* Markets Directory Grid */}
        <section className="mt-20">
          <div className="mb-10">
            <span className="text-[11px] font-mono uppercase tracking-[0.2em] text-blue-400">
              Country Solutions
            </span>
            <h2 className="text-3xl sm:text-4xl font-black uppercase tracking-tight text-white mt-2">
              Select Your Country Corridor
            </h2>
            <p className="text-sm text-white/60 font-light mt-2 max-w-2xl">
              Each market profile includes localized regulatory compliance, currency support, and regional dispatch benchmarks.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {MARKETS_DATA.map((market) => (
              <Link
                key={market.slug}
                href={`/markets/${market.slug}`}
                className="p-6 rounded-2xl bg-white/[0.02] border border-white/10 hover:border-white/25 hover:bg-white/[0.04] transition-all flex flex-col justify-between group"
              >
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2.5">
                      <span className="text-2xl">{market.flag}</span>
                      <div>
                        <h3 className="text-lg font-bold text-white group-hover:text-blue-300 transition-colors">
                          {market.name}
                        </h3>
                        <span className="text-[11px] font-mono text-white/50">
                          {market.nativeName} · {market.region}
                        </span>
                      </div>
                    </div>
                  </div>

                  <p className="text-xs text-white/70 leading-relaxed font-light line-clamp-3 mb-4">
                    {market.summary}
                  </p>

                  <div className="flex flex-wrap gap-1.5 mb-4">
                    {market.complianceFrameworks.slice(0, 2).map((comp, idx) => (
                      <span
                        key={idx}
                        className="px-2 py-0.5 rounded bg-white/5 text-[10px] font-mono text-white/60"
                      >
                        {comp}
                      </span>
                    ))}
                  </div>
                </div>

                <div className="pt-4 border-t border-white/5 flex items-center justify-between text-xs font-mono text-white/60">
                  <span>{market.currency}</span>
                  <span className="text-blue-400 group-hover:translate-x-1 transition-transform flex items-center gap-1">
                    Explore Corridor <ChevronRight className="w-3.5 h-3.5" />
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </section>

        {/* Bottom CTA */}
        <section className="mt-24 p-10 sm:p-14 rounded-3xl bg-gradient-to-b from-white/[0.06] to-transparent border border-white/15 text-center">
          <h2 className="text-3xl sm:text-5xl font-black uppercase tracking-tight text-white">
            Need a Sovereign Cluster in Your Region?
          </h2>
          <p className="mt-4 text-base sm:text-lg text-white/70 max-w-2xl mx-auto font-light">
            Pegasus deploys turnkey sovereign clusters with dedicated local database residency, in-country payment rails, and zero data cross-contamination.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/join"
              className="px-8 py-4 rounded-xl bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors shadow-xl"
            >
              Request Sovereign Deployment
            </Link>
            <Link
              href="/contact"
              className="px-8 py-4 rounded-xl bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Contact Global Infrastructure Team
            </Link>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
