import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import Link from 'next/link';
import { INDUSTRIES_DATA, getIndustryBySlug } from '@/app/data/industriesData';
import SiteNav from '@/app/components/explore/SiteNav';
import Footer from '@/app/components/Footer';
import { breadcrumbJsonLd, faqPageJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import { getServerLanguage } from '@/app/lib/i18n/server';
import { Check, ArrowRight, Shield, TrendingUp, HelpCircle, Layers, Building2 } from 'lucide-react';

export function generateStaticParams() {
  return INDUSTRIES_DATA.map((ind) => ({ vertical: ind.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ vertical: string }>;
}): Promise<Metadata> {
  const { vertical: slug } = await params;
  const industry = getIndustryBySlug(slug);
  if (!industry) return {};

  const lang = await getServerLanguage();

  return pageMetadata({
    title: industry.metaTitle,
    description: industry.metaDescription,
    path: `/solutions/industry/${slug}`,
    language: lang,
  });
}

export default async function IndustrySolutionDetailPage({
  params,
}: {
  params: Promise<{ vertical: string }>;
}) {
  const { vertical: slug } = await params;
  const industry = getIndustryBySlug(slug);
  if (!industry) notFound();

  const lang = await getServerLanguage();
  const isRu = lang === 'ru';

  const breadcrumbs = [
    { name: isRu ? 'Главная' : 'Home', path: '/' },
    { name: isRu ? 'Решения' : 'Solutions', path: '/solutions' },
    { name: industry.name, path: `/solutions/industry/${slug}` },
  ];

  return (
    <div className="min-h-screen bg-[#070709] text-white flex flex-col font-sans selection:bg-white/20">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(breadcrumbJsonLd(breadcrumbs))}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={jsonLdScript(faqPageJsonLd(industry.faqs, lang))}
      />
      <SiteNav />

      <main className="flex-1 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 pt-32 pb-24 w-full">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-xs font-mono text-white/50 mb-8">
          <Link href="/solutions" className="hover:text-white transition-colors">
            {isRu ? '← Все решения' : '← All Solutions'}
          </Link>
          <span>/</span>
          <span className="text-white/80">{industry.name}</span>
        </div>

        {/* Hero */}
        <div>
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-[11px] font-mono tracking-wider uppercase text-white/70 mb-6">
            <Building2 className="w-3.5 h-3.5 text-white" />
            <span>Industry Playbook · {industry.name}</span>
          </div>

          <h1 className="text-3xl sm:text-5xl lg:text-6xl font-black tracking-tight text-white uppercase">
            {industry.headline}
          </h1>

          <p className="mt-6 text-lg sm:text-xl text-white/70 leading-relaxed font-light">
            {industry.summary}
          </p>

          <div className="mt-8 flex flex-wrap gap-4">
            <Link
              href="/join"
              className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs hover:bg-white/90 transition-colors"
            >
              Request Industry Demo
            </Link>
            <Link
              href="/platform"
              className="px-6 py-3 rounded-lg bg-white/5 text-white font-bold uppercase tracking-wider text-xs hover:bg-white/10 transition-colors border border-white/15"
            >
              Platform Overview
            </Link>
          </div>
        </div>

        {/* ROI Metrics */}
        <section className="mt-16 grid grid-cols-1 sm:grid-cols-3 gap-6">
          {industry.metrics.map((metric, idx) => (
            <div
              key={idx}
              className="p-6 rounded-2xl bg-white/[0.03] border border-white/10 text-center"
            >
              <span className="text-3xl sm:text-4xl font-black text-white font-mono block">
                {metric.value}
              </span>
              <span className="text-xs font-mono text-white/60 mt-2 block uppercase tracking-wider">
                {metric.label}
              </span>
            </div>
          ))}
        </section>

        {/* Key Challenges vs How Pegasus Solves */}
        <section className="mt-20">
          <div className="mb-8">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              Industry Challenges & Solutions
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              How Pegasus addresses specific distribution friction
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {/* Challenges */}
            <div className="p-8 rounded-2xl bg-white/[0.02] border border-white/10">
              <h3 className="text-lg font-bold uppercase text-rose-300 mb-6 flex items-center gap-2">
                <span>Core Operating Bottlenecks</span>
              </h3>
              <ul className="space-y-4">
                {industry.keyChallenges.map((challenge, idx) => (
                  <li key={idx} className="flex items-start gap-3 text-xs text-white/70 font-light leading-relaxed">
                    <span className="w-1.5 h-1.5 rounded-full bg-rose-400 shrink-0 mt-1.5" />
                    <span>{challenge}</span>
                  </li>
                ))}
              </ul>
            </div>

            {/* How Pegasus Solves */}
            <div className="p-8 rounded-2xl bg-white/[0.04] border border-white/20">
              <h3 className="text-lg font-bold uppercase text-emerald-300 mb-6 flex items-center gap-2">
                <span>The Pegasus Solution</span>
              </h3>
              <div className="space-y-6">
                {industry.howPegasusSolves.map((sol, idx) => (
                  <div key={idx}>
                    <h4 className="text-sm font-bold text-white mb-1">{sol.title}</h4>
                    <p className="text-xs text-white/70 font-light leading-relaxed">{sol.description}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* Operational Workflow Steps */}
        <section className="mt-20">
          <div className="mb-8">
            <h2 className="text-2xl font-black uppercase tracking-tight text-white">
              Standard Operating Procedure
            </h2>
            <p className="text-xs font-mono text-white/50 mt-1">
              End-to-end execution lifecycle in Pegasus
            </p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {industry.workflowSteps.map((ws, idx) => (
              <div
                key={idx}
                className="p-6 rounded-xl bg-white/[0.02] border border-white/10 flex flex-col justify-between"
              >
                <div>
                  <span className="text-xs font-mono text-emerald-400 font-bold block mb-2">
                    {ws.step}
                  </span>
                  <p className="text-xs text-white/80 font-light leading-relaxed">{ws.detail}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* FAQs */}
        <section className="mt-20">
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-wider text-white/50 mb-3">
            <HelpCircle className="w-4 h-4 text-white" />
            <span>Industry Specific FAQ</span>
          </div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white mb-8">
            Frequently Asked Questions
          </h2>
          <div className="space-y-4">
            {industry.faqs.map((faq, idx) => (
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
            Schedule an Industry Walkthrough
          </h2>
          <p className="mt-3 text-sm text-white/70 max-w-xl mx-auto font-light leading-relaxed">
            See how Pegasus configures {industry.name.toLowerCase()} workflows, prevents dropped handoffs, and unlocks real-time operational visibility.
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
