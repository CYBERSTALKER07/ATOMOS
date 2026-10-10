'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { ArrowUpRight, ChevronRight } from 'lucide-react';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import PartnerBrandStrip from '@/app/components/visuals/PartnerBrandStrip';
import TopicCanvas from '@/app/components/visuals/TopicCanvas';

export interface BreadcrumbItem {
  categoryLabel?: string;
  categoryHref?: string;
  subCategoryLabel?: string;
  subCategoryHref?: string;
  currentPage: string;
  telemetryCode?: string;
}

export interface MetricItem {
  value: string;
  label: string;
  sublabel?: string;
}

export interface CapabilityModuleItem {
  id?: string;
  tag?: string;
  title: string;
  description: string;
  href: string;
  imageSrc?: string;
  badge?: string;
}

export interface FaqItem {
  question: string;
  answer: string;
}

export interface BrandedSubpageLayoutProps {
  activeHref?: string;
  // Tier 1 & 2: Breadcrumbs & Hero
  breadcrumb: BreadcrumbItem;
  badge?: string;
  title: string;
  summary: string;
  primaryCta?: {
    label: string;
    href: string;
  };
  secondaryCta?: {
    label: string;
    href: string;
  };
  heroVisual?: ReactNode;
  heroImageSrc?: string;
  heroImageAlt?: string;
  topicSlug?: string;

  // Tier 3: Proof / SLA Metric Strip
  metrics?: MetricItem[];

  // Tier 4: Core Capabilities Bento Grid
  capabilitiesTitle?: string;
  capabilitiesKicker?: string;
  capabilitiesDescription?: string;
  capabilities?: CapabilityModuleItem[];

  // Tier 5: Technical Details / Systems
  technicalDetails?: ReactNode;

  // Tier 5.5: Enterprise FAQ
  faqs?: FaqItem[];
  faqsTitle?: string;
  faqsKicker?: string;

  // Tier 6: Conversion Band
  conversionBand?: {
    kicker?: string;
    title: string;
    description: string;
    primaryCta?: {
      label: string;
      href: string;
    };
    secondaryCta?: {
      label: string;
      href: string;
    };
  };
  showPartnerStrip?: boolean;
}

export default function BrandedSubpageLayout({
  activeHref,
  breadcrumb,
  badge,
  title,
  summary,
  primaryCta = { label: 'Request Demonstration', href: '/join' },
  secondaryCta = { label: 'Explore Architecture', href: '/platform' },
  heroVisual,
  heroImageSrc,
  heroImageAlt,
  topicSlug,
  metrics,
  capabilitiesTitle = 'Autonomous Capabilities',
  capabilitiesKicker = 'CORE ENGINES',
  capabilitiesDescription = 'Deterministic algorithmic systems operating in real time across the distributed network.',
  capabilities = [],
  technicalDetails,
  faqs = [],
  faqsTitle = 'Technical Specifications & FAQ',
  faqsKicker = 'SYSTEM PROTOCOLS',
  conversionBand,
  showPartnerStrip = true,
}: BrandedSubpageLayoutProps) {
  const currentPath = activeHref || breadcrumb.categoryHref || '/';

  return (
    <FleekPageShell activeHref={currentPath}>
      <div className="min-h-screen bg-[#06070a] text-zinc-100 selection:bg-cyan-500/20 selection:text-cyan-200">
        
        {/* =========================================================================
            TIER 1: Sticky Telemetry Navigation & Monospace Breadcrumbs
            ========================================================================= */}
        <section className="border-b border-white/[0.08] bg-[#06070a]/90 backdrop-blur-xl px-5 sm:px-8 py-3.5 sticky top-[4.5rem] md:top-20 z-30">
          <div className="max-w-7xl mx-auto flex items-center justify-between">
            <nav aria-label="Breadcrumb" className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.2em] text-zinc-400">
              <Link href="/" className="hover:text-white transition-colors">
                PEGASUS
              </Link>
              <span className="text-zinc-600">/</span>
              {breadcrumb.categoryLabel && (
                <>
                  <Link
                    href={breadcrumb.categoryHref || '/'}
                    className="hover:text-white transition-colors"
                  >
                    {breadcrumb.categoryLabel}
                  </Link>
                  <span className="text-zinc-600">/</span>
                </>
              )}
              {breadcrumb.subCategoryLabel && (
                <>
                  <Link
                    href={breadcrumb.subCategoryHref || '#'}
                    className="hover:text-white transition-colors text-zinc-400"
                  >
                    {breadcrumb.subCategoryLabel}
                  </Link>
                  <span className="text-zinc-600">/</span>
                </>
              )}
              <span className="text-cyan-400 font-medium truncate max-w-[200px] sm:max-w-none">
                {breadcrumb.currentPage}
              </span>
            </nav>

            <div className="hidden sm:flex items-center gap-3 font-mono text-[10px] uppercase tracking-widest text-zinc-500">
              <span className="inline-flex items-center gap-1.5 text-emerald-400">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                {breadcrumb.telemetryCode || 'SYSTEM ONLINE'}
              </span>
              <span className="text-zinc-700">|</span>
              <span>OR-TOOLS v4.8</span>
            </div>
          </div>
        </section>

        {/* =========================================================================
            TIER 2: Split Architectural Hero
            ========================================================================= */}
        <section className="relative overflow-hidden border-b border-white/[0.08] bg-[#06070a] pt-10 sm:pt-14 pb-14 sm:pb-20">
          {/* Subtle geometric background grid */}
          <div
            className="absolute inset-0 opacity-[0.025] pointer-events-none"
            style={{
              backgroundImage: `linear-gradient(#fff 1px, transparent 1px), linear-gradient(90deg, #fff 1px, transparent 1px)`,
              backgroundSize: '40px 40px',
            }}
          />

          <div className="max-w-7xl mx-auto px-5 sm:px-8 relative z-10">
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center">
              
              {/* Left Column: Eyebrow + Display Headline + Summary + Twin CTAs */}
              <div className="lg:col-span-7 space-y-6 sm:space-y-8">
                {badge && (
                  <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-cyan-500/20 bg-cyan-500/10 text-cyan-300 text-[10px] sm:text-[11px] font-mono tracking-[0.22em] uppercase">
                    <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse" />
                    {badge}
                  </div>
                )}

                <h1 className="text-4xl sm:text-5xl lg:text-6xl font-light tracking-[-0.035em] text-white leading-[1.06]">
                  {title}
                </h1>

                <p className="text-base sm:text-lg font-light text-zinc-400 leading-relaxed max-w-2xl">
                  {summary}
                </p>

                {/* Twin Action CTAs */}
                <div className="flex flex-wrap items-center gap-4 pt-2">
                  <Link
                    href={primaryCta.href}
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-md bg-white text-black text-sm font-medium hover:bg-zinc-200 transition-colors shadow-lg shadow-white/5"
                  >
                    <span>{primaryCta.label}</span>
                    <ChevronRight className="w-4 h-4" />
                  </Link>

                  <Link
                    href={secondaryCta.href}
                    className="inline-flex items-center gap-2 px-6 py-3 rounded-md bg-white/[0.06] border border-white/15 text-white text-sm font-medium hover:bg-white/10 backdrop-blur-sm transition-colors"
                  >
                    <span>{secondaryCta.label}</span>
                    <ArrowUpRight className="w-4 h-4" />
                  </Link>
                </div>
              </div>

              {/* Right Column: Visual Telemetry Canvas / 3D Asset */}
              <div className="lg:col-span-5 relative">
                {heroVisual ? (
                  heroVisual
                ) : heroImageSrc ? (
                  <div className="relative rounded-2xl border border-white/15 overflow-hidden aspect-[16/11] bg-[#090b10] group shadow-2xl">
                    <Image
                      src={heroImageSrc}
                      alt={heroImageAlt || title}
                      fill
                      priority
                      className="object-cover transition-transform duration-700 group-hover:scale-105"
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent pointer-events-none" />
                    <div className="absolute bottom-4 left-4 right-4 flex items-center justify-between font-mono text-[10px] text-zinc-400 tracking-wider">
                      <span className="uppercase text-cyan-400">TELEMETRY // LIVE HUD</span>
                      <span>SLA: 99.999%</span>
                    </div>
                  </div>
                ) : (
                  <div className="relative rounded-2xl border border-white/15 overflow-hidden aspect-[16/11] bg-[#090b10]">
                    <TopicCanvas slug={topicSlug || 'control_plane'} />
                  </div>
                )}
              </div>

            </div>
          </div>
        </section>

        {/* =========================================================================
            TIER 3: Proof & SLA Benchmark Strip
            ========================================================================= */}
        {metrics && metrics.length > 0 && (
          <section className="border-b border-white/[0.08] bg-black">
            <div className="max-w-7xl mx-auto px-5 sm:px-8">
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 border-x border-white/[0.08] divide-y sm:divide-y-0 sm:divide-x divide-white/[0.08]">
                {metrics.map((metric, idx) => (
                  <div key={idx} className="p-6 sm:p-8 bg-[#06070a]/50 hover:bg-white/[0.02] transition-colors">
                    <span className="font-mono text-3xl sm:text-4xl font-normal text-white block tracking-tight">
                      {metric.value}
                    </span>
                    <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-zinc-400 mt-2 block">
                      {metric.label}
                    </span>
                    {metric.sublabel && (
                      <span className="text-xs font-light text-zinc-500 mt-1 block">
                        {metric.sublabel}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </section>
        )}

        {/* =========================================================================
            TIER 4: Core Capabilities Bento Grid (Standardized Hover-Reveal Module Cards)
            ========================================================================= */}
        {capabilities && capabilities.length > 0 && (
          <section className="py-20 sm:py-24 max-w-7xl mx-auto px-5 sm:px-8 border-b border-white/[0.08]">
            <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 gap-4">
              <div>
                <span className="font-mono text-[10px] uppercase tracking-[0.22em] text-cyan-400 block mb-2">
                  {capabilitiesKicker}
                </span>
                <h2 className="text-3xl sm:text-4xl font-light tracking-[-0.025em] text-white">
                  {capabilitiesTitle}
                </h2>
              </div>
              <p className="text-xs sm:text-sm font-light text-zinc-400 max-w-md">
                {capabilitiesDescription}
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {capabilities.map((item, idx) => (
                <Link
                  key={idx}
                  href={item.href}
                  className="module-card group min-h-[360px] flex flex-col justify-between p-8"
                >
                  {/* Background image unblurring on hover */}
                  {item.imageSrc && (
                    <Image
                      src={item.imageSrc}
                      alt={item.title}
                      fill
                      className="module-card__image"
                    />
                  )}
                  <div className="module-card__overlay" />

                  {/* Top card metadata */}
                  <div className="relative z-10 flex items-center justify-between">
                    <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-400 bg-black/50 border border-white/10 px-2.5 py-1 rounded-sm backdrop-blur-sm">
                      {item.tag || `ENGINE 0${idx + 1}`}
                    </span>
                    {item.badge && (
                      <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-400">
                        {item.badge}
                      </span>
                    )}
                  </div>

                  {/* Bottom card content & action */}
                  <div className="relative z-10 pt-16">
                    <h3 className="text-xl font-normal text-white group-hover:text-cyan-200 transition-colors">
                      {item.title}
                    </h3>
                    <p className="text-xs text-zinc-400 font-light mt-2 line-clamp-3 leading-relaxed">
                      {item.description}
                    </p>

                    <div className="pt-6">
                      <span className="module-card__btn inline-flex items-center gap-1.5 px-4 py-2 rounded-md text-xs font-medium">
                        <span>Explore Engine</span>
                        <ArrowUpRight className="w-3.5 h-3.5" />
                      </span>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          </section>
        )}

        {/* =========================================================================
            TIER 5: Technical Details / Systems
            ========================================================================= */}
        {technicalDetails && (
          <section className="py-20 max-w-7xl mx-auto px-5 sm:px-8 border-b border-white/[0.08]">
            {technicalDetails}
          </section>
        )}

        {/* =========================================================================
            TIER 5.5: Enterprise FAQ Grid
            ========================================================================= */}
        {faqs && faqs.length > 0 && (
          <section className="py-20 sm:py-24 max-w-7xl mx-auto px-5 sm:px-8 border-b border-white/[0.08]">
            <div className="mb-12">
              <span className="font-mono text-[10px] uppercase tracking-[0.22em] text-cyan-400 block mb-2">
                {faqsKicker}
              </span>
              <h2 className="text-3xl sm:text-4xl font-light tracking-tight text-white">
                {faqsTitle}
              </h2>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {faqs.map((faq, idx) => (
                <div
                  key={idx}
                  className="rounded-2xl border border-white/[0.08] bg-[#090b10] p-8 space-y-3 hover:border-white/20 transition-colors"
                >
                  <h3 className="text-base font-normal text-white flex items-start gap-3">
                    <span className="font-mono text-xs text-cyan-400 mt-0.5">[{idx < 9 ? `0${idx + 1}` : idx + 1}]</span>
                    <span>{faq.question}</span>
                  </h3>
                  <p className="text-xs sm:text-sm text-zinc-400 font-light leading-relaxed pl-7">
                    {faq.answer}
                  </p>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* =========================================================================
            TIER 6: Chamfered Conversion Band & Enterprise Partner Rail
            ========================================================================= */}
        <section className="py-20 max-w-7xl mx-auto px-5 sm:px-8">
          <div className="rounded-3xl border border-white/15 bg-gradient-to-b from-[#090b10] to-[#06070a] p-10 sm:p-16 relative overflow-hidden flex flex-col lg:flex-row items-center justify-between gap-10 shadow-2xl">
            {/* Grid accent backdrop */}
            <div
              className="absolute inset-0 opacity-[0.03] pointer-events-none"
              style={{
                backgroundImage: `linear-gradient(#fff 1px, transparent 1px), linear-gradient(90deg, #fff 1px, transparent 1px)`,
                backgroundSize: '32px 32px',
              }}
            />

            <div className="space-y-4 max-w-2xl relative z-10">
              <span className="font-mono text-[10px] tracking-[0.22em] text-emerald-400 uppercase inline-flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                {conversionBand?.kicker || 'PRODUCTION-GRADE ORCHESTRATION'}
              </span>
              <h2 className="text-3xl sm:text-4xl lg:text-5xl font-light tracking-[-0.03em] text-white leading-tight">
                {conversionBand?.title || 'Accelerate Fleet & Logistics Operations Today'}
              </h2>
              <p className="text-sm sm:text-base font-light text-zinc-400 leading-relaxed">
                {conversionBand?.description ||
                  'Deploy Pegasus into your logistics network with sub-second CVRP calculation, automated gate verification, and real-time ledger accounting.'}
              </p>
            </div>

            <div className="flex flex-col sm:flex-row items-center gap-4 relative z-10 w-full sm:w-auto">
              <Link
                href={conversionBand?.primaryCta?.href || '/join'}
                className="w-full sm:w-auto text-center px-8 py-3.5 rounded-md bg-white text-black text-sm font-medium hover:bg-zinc-200 transition-colors shadow-lg"
              >
                {conversionBand?.primaryCta?.label || 'Request Live Demo →'}
              </Link>
              {conversionBand?.secondaryCta && (
                <Link
                  href={conversionBand.secondaryCta.href}
                  className="w-full sm:w-auto text-center px-6 py-3.5 rounded-md bg-white/[0.06] border border-white/15 text-white text-sm font-medium hover:bg-white/10 transition-colors"
                >
                  {conversionBand.secondaryCta.label}
                </Link>
              )}
            </div>
          </div>

          {/* Bottom Partner Strip */}
          {showPartnerStrip && (
            <div className="mt-16">
              <PartnerBrandStrip
                showDivider={true}
                label="ENTERPRISE INFRASTRUCTURE & CARRIER COMPATIBILITY"
              />
            </div>
          )}
        </section>

      </div>
    </FleekPageShell>
  );
}
