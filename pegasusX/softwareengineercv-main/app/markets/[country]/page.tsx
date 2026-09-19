import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import {
  Globe2,
  ShieldCheck,
  Zap,
  ArrowRight,
  Server,
  ChevronRight,
  CheckCircle2,
  AlertCircle,
  BarChart3,
  Cpu,
  Lock,
} from 'lucide-react';
import SiteNav from '@/app/components/explore/SiteNav';
import SubpageHero from '@/app/components/SubpageHero';
import Footer from '@/app/components/Footer';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { MARKETS_DATA, getMarketBySlug } from '@/app/data/marketsData';
import {
  pageMetadata,
  faqPageJsonLd,
  breadcrumbJsonLd,
  jsonLdGraphScript,
} from '@/app/lib/seo';

type Props = {
  params: Promise<{ country: string }>;
};

export async function generateStaticParams() {
  return MARKETS_DATA.map((m) => ({
    country: m.slug,
  }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { country: slug } = await params;
  const market = getMarketBySlug(slug);
  const lang = await getServerLanguage();

  if (!market) {
    return { title: 'Market Not Found | Pegasus' };
  }

  const title = `${market.name} Logistics & Supply Chain Software | Pegasus`;
  const description = `${market.summary.slice(0, 155)}...`;

  return pageMetadata({
    title,
    description,
    path: `/markets/${market.slug}`,
    language: lang,
  });
}

export default async function MarketCountryPage({ params }: Props) {
  const { country: slug } = await params;
  const market = getMarketBySlug(slug);
  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  if (!market) {
    notFound();
  }

  const structuredData = [
    faqPageJsonLd(market.faqs, lang),
    breadcrumbJsonLd([
      { name: 'Home', path: '/' },
      { name: 'Markets', path: '/markets' },
      { name: market.name, path: `/markets/${market.slug}` },
    ]),
    {
      '@context': 'https://schema.org',
      '@type': 'Place',
      name: market.name,
      alternateName: market.nativeName,
      address: {
        '@type': 'PostalAddress',
        addressCountry: market.name,
      },
    },
  ];

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-white selection:text-black">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdGraphScript(structuredData)}
      />
      <SiteNav activeHref="/markets" />

      <SubpageHero
        badge={`${market.region.toUpperCase()} CORRIDOR · ${market.currency}`}
        badgeIcon={<span className="text-sm">{market.flag}</span>}
        title={market.headline}
        summary={market.summary}
        primaryCta={{
          label: isRu ? `Демо для ${market.name}` : `Request ${market.name} Demo`,
          href: '/join',
        }}
        secondaryCta={{
          label: isRu ? 'Глобальная архитектура' : 'Global Architecture',
          href: '/global-logistics',
        }}
        widget={{
          title: market.name.toUpperCase(),
          description: `${market.cellCluster} · ${market.latency} latency`,
          href: '/global-logistics',
        }}
        breadcrumb={{
          homeLabel: isRu ? 'Главная' : 'Home',
          categoryLabel: isRu ? 'Рынки' : 'Markets',
          categoryHref: '/markets',
          currentPage: market.name,
        }}
      />

      <main className="flex-1 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 pt-10 pb-24 w-full">

        {/* Infrastructure Specs */}
        <section className="mt-14 p-6 rounded-none bg-white/[0.02] border border-white/10 grid grid-cols-1 sm:grid-cols-3 gap-6 font-mono text-xs">
          <div>
            <span className="text-white/40 uppercase tracking-wider block mb-1">Local Cell Cluster</span>
            <span className="text-white font-bold">{market.cellCluster}</span>
          </div>
          <div>
            <span className="text-white/40 uppercase tracking-wider block mb-1">Regional Latency</span>
            <span className="text-white font-mono font-bold">{market.latency}</span>
          </div>
          <div>
            <span className="text-white/40 uppercase tracking-wider block mb-1">Primary Settlement Currency</span>
            <span className="text-white font-bold">{market.currency} ({market.primaryLanguage})</span>
          </div>
        </section>

        {/* Benchmark Stats */}
        <section className="mt-12 grid grid-cols-1 sm:grid-cols-3 gap-6">
          {market.stats.map((stat, idx) => (
            <div
              key={idx}
              className="p-6 rounded-none bg-white/[0.03] border border-white/10 text-center"
            >
              <span className="text-3xl sm:text-4xl font-black text-white font-mono block">
                {stat.value}
              </span>
              <span className="text-xs font-mono text-white/60 mt-2 block uppercase tracking-wider">
                {stat.label}
              </span>
            </div>
          ))}
        </section>

        {/* Market Overview */}
        <section className="mt-20">
          <div className="mb-6">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              Logistics Landscape in {market.name}
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              Regional operating conditions, trade corridors, and regulatory frameworks
            </p>
          </div>

          <div className="p-8 rounded-none bg-white/[0.02] border border-white/10">
            <p className="text-base text-white/80 leading-relaxed font-light mb-6">
              {market.marketOverview}
            </p>

            <h3 className="text-xs font-mono uppercase tracking-widest text-white/40 mb-3">
              Enforced Regulatory & Compliance Standards:
            </h3>
            <div className="flex flex-wrap gap-2">
              {market.complianceFrameworks.map((comp, idx) => (
                <span
                  key={idx}
                  className="px-3 py-1.5 rounded-none bg-white/5 border border-white/10 text-xs font-mono text-white/80 flex items-center gap-1.5"
                >
                  <ShieldCheck className="w-3.5 h-3.5 text-white/80" />
                  <span>{comp}</span>
                </span>
              ))}
            </div>
          </div>
        </section>

        {/* Bottlenecks vs Solutions */}
        <section className="mt-20">
          <div className="mb-8">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              Operating Bottlenecks & Pegasus Solutions
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              How Pegasus addresses specific distribution friction in {market.name}
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {/* Bottlenecks */}
            <div className="p-8 rounded-none bg-white/[0.02] border border-white/10">
              <h3 className="text-lg font-bold uppercase text-white mb-6 flex items-center gap-2">
                <AlertCircle className="w-5 h-5 text-zinc-400" />
                <span>Regional Bottlenecks</span>
              </h3>
              <ul className="space-y-4">
                {market.logisticsBottlenecks.map((b, idx) => (
                  <li key={idx} className="flex items-start gap-3 text-sm text-white/70 leading-relaxed font-light">
                    <span className="text-zinc-500 font-mono text-xs mt-0.5">•</span>
                    <span>{b}</span>
                  </li>
                ))}
              </ul>
            </div>

            {/* Pegasus Solution */}
            <div className="p-8 rounded-none bg-white/[0.02] border border-white/10">
              <h3 className="text-lg font-bold uppercase text-white mb-6 flex items-center gap-2">
                <CheckCircle2 className="w-5 h-5 text-white" />
                <span>Pegasus Deployment Solution</span>
              </h3>
              <ul className="space-y-4">
                {market.pegasusSolution.map((s, idx) => (
                  <li key={idx} className="flex items-start gap-3 text-sm text-white/80 leading-relaxed font-light">
                    <CheckCircle2 className="w-4 h-4 text-white shrink-0 mt-0.5" />
                    <span>{s}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </section>

        {/* FAQs */}
        <section className="mt-20">
          <div className="mb-8">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              Frequently Asked Questions · {market.name}
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              Specific deployment questions for {market.name} operators
            </p>
          </div>

          <div className="space-y-4">
            {market.faqs.map((faq, idx) => (
              <div
                key={idx}
                className="p-6 rounded-none bg-white/[0.02] border border-white/10"
              >
                <h3 className="text-base font-bold text-white mb-2">
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
        <section className="mt-20 p-8 sm:p-12 rounded-none bg-gradient-to-b from-white/[0.06] to-transparent border border-white/15 text-center">
          <h2 className="text-2xl sm:text-4xl font-black uppercase tracking-tight text-white">
            Deploy Pegasus in {market.name}
          </h2>
          <p className="mt-3 text-sm sm:text-base text-white/70 max-w-xl mx-auto font-light">
            Empower your supplier, warehouse, carrier, and retail network with the world's most advanced logistics operating system.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/join"
              className="px-8 py-3.5 rounded-none bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors shadow-xl"
            >
              Schedule {market.name} Demo
            </Link>
            <Link
              href="/markets"
              className="px-8 py-3.5 rounded-none bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              View All Global Markets
            </Link>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
