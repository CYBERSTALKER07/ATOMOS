'use client';

import React from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { ArrowRight, CheckCircle2, ShieldCheck } from 'lucide-react';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';

export interface O9SplitTourCTAProps {
  relatedProjectSlug?: string;
  demoHref?: string;
  tourHref?: string;
  className?: string;
}

/**
 * o9-style dual conversion banner:
 * Card 1: "Request Architecture Demo"
 * Card 2: "Explore Interactive Platform Tour"
 * Rendered at the foot of every Fleek page via O9FleekPageLayout.
 */
export default function O9SplitTourCTA({
  relatedProjectSlug,
  demoHref = '/join',
  tourHref,
  className,
}: O9SplitTourCTAProps) {
  const { language, t } = useLanguage();
  const resolvedTourHref =
    tourHref ?? (relatedProjectSlug ? `/projects/${relatedProjectSlug}` : '/platform');

  const trustMarkers =
    language === 'ru'
      ? ['Аудит безопасности SOC-2', 'SLA доступности 99.99%', 'Без риска замены существующих ERP']
      : ['SOC-2 Type II Certified', '99.99% Uptime SLA', 'Zero Rip-and-Replace'];

  return (
    <section
      className={cn(
        'w-full pt-12 md:pt-16 border-t border-zinc-200 dark:border-white/10 transition-colors',
        className
      )}
      aria-label="Discover the platform"
    >
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Card 1: Request Architecture Demo */}
        <article className="relative overflow-hidden bg-zinc-900 text-white border border-zinc-700/60 dark:border-white/10 hover:border-emerald-500/60 transition-all duration-300 flex flex-col justify-between group shadow-lg">
          {/* Tactical Corner Brackets */}
          <span
            className="absolute -top-1 -left-1 w-3 h-3 border-t-2 border-l-2 border-emerald-500 z-20 pointer-events-none group-hover:border-emerald-400 transition-colors"
            aria-hidden
          />
          <span
            className="absolute -bottom-1 -right-1 w-3 h-3 border-b-2 border-r-2 border-emerald-500 z-20 pointer-events-none group-hover:border-emerald-400 transition-colors"
            aria-hidden
          />

          {/* Background Image Container */}
          <div className="relative aspect-[16/9] sm:aspect-[21/9] w-full overflow-hidden bg-black">
            <Image
              src={EDITORIAL_IMAGES[1]}
              alt=""
              fill
              className="object-cover opacity-40 group-hover:scale-105 transition-transform duration-700"
              sizes="(max-width: 900px) 100vw, 50vw"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-zinc-900 via-zinc-900/60 to-transparent pointer-events-none" />
            <div className="absolute top-4 left-4 z-10 inline-flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
              <span className="font-mono text-[10px] sm:text-[11px] tracking-widest text-emerald-400 uppercase bg-emerald-950/80 border border-emerald-500/40 px-2.5 py-1">
                {t('licensing_demo_tag', 'ENTERPRISE ARCHITECTURE DEMO')}
              </span>
            </div>
          </div>

          {/* Content Body */}
          <div className="p-6 sm:p-8 flex flex-col flex-1 justify-between gap-6 -mt-8 relative z-10">
            <div>
              <h3 className="text-xl sm:text-2xl lg:text-3xl font-semibold tracking-tight text-white leading-tight">
                {t('licensing_demo_title', 'Schedule a Live Control Plane Walkthrough')}
              </h3>
              <p className="mt-3 text-sm sm:text-base leading-relaxed text-zinc-300">
                {t(
                  'licensing_demo_desc',
                  'Experience multi-role dispatch boards, atomic freeze-lock state machines, and real-time Kafka outbox synchronization tailored to your network topology.'
                )}
              </p>
            </div>

            <div className="pt-4 border-t border-white/10 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <Link
                href={demoHref}
                className="inline-flex items-center justify-center gap-2 px-6 py-3.5 bg-emerald-500 hover:bg-emerald-400 text-black font-mono text-xs uppercase tracking-wider font-semibold transition-all duration-200 shadow-[0_0_20px_rgba(16,185,129,0.25)] rounded-none"
              >
                <span>{t('nav_demo', 'REQUEST DEMO')}</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
              <span className="font-mono text-[11px] text-zinc-400 flex items-center gap-1.5">
                <ShieldCheck className="w-4 h-4 text-emerald-400" />
                <span>NDA & Architecture Walkthrough</span>
              </span>
            </div>
          </div>
        </article>

        {/* Card 2: Interactive Platform Tour */}
        <article className="relative overflow-hidden bg-zinc-900 text-white border border-zinc-700/60 dark:border-white/10 hover:border-emerald-500/60 transition-all duration-300 flex flex-col justify-between group shadow-lg">
          {/* Tactical Corner Brackets */}
          <span
            className="absolute -top-1 -left-1 w-3 h-3 border-t-2 border-l-2 border-emerald-500 z-20 pointer-events-none group-hover:border-emerald-400 transition-colors"
            aria-hidden
          />
          <span
            className="absolute -bottom-1 -right-1 w-3 h-3 border-b-2 border-r-2 border-emerald-500 z-20 pointer-events-none group-hover:border-emerald-400 transition-colors"
            aria-hidden
          />

          {/* Background Image Container */}
          <div className="relative aspect-[16/9] sm:aspect-[21/9] w-full overflow-hidden bg-black">
            <Image
              src={EDITORIAL_IMAGES[2]}
              alt=""
              fill
              className="object-cover opacity-40 group-hover:scale-105 transition-transform duration-700"
              sizes="(max-width: 900px) 100vw, 50vw"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-zinc-900 via-zinc-900/60 to-transparent pointer-events-none" />
            <div className="absolute top-4 left-4 z-10 inline-flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
              <span className="font-mono text-[10px] sm:text-[11px] tracking-widest text-emerald-400 uppercase bg-emerald-950/80 border border-emerald-500/40 px-2.5 py-1">
                {t('licensing_tour_tag', 'SELF-GUIDED PLATFORM TOUR')}
              </span>
            </div>
          </div>

          {/* Content Body */}
          <div className="p-6 sm:p-8 flex flex-col flex-1 justify-between gap-6 -mt-8 relative z-10">
            <div>
              <h3 className="text-xl sm:text-2xl lg:text-3xl font-semibold tracking-tight text-white leading-tight">
                {t('licensing_tour_title', 'Explore the 6 Network Roles in Action')}
              </h3>
              <p className="mt-3 text-sm sm:text-base leading-relaxed text-zinc-300">
                {t(
                  'licensing_tour_desc',
                  'Test real-world operational flows from supplier placement, warehouse picking, driver routing, to retailer pay-at-delivery across web and mobile surfaces.'
                )}
              </p>
            </div>

            <div className="pt-4 border-t border-white/10 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <Link
                href={resolvedTourHref}
                className="inline-flex items-center justify-center gap-2 px-6 py-3.5 bg-white hover:bg-zinc-200 text-black font-mono text-xs uppercase tracking-wider font-semibold transition-all duration-200 rounded-none"
              >
                <span>{t('nav_tour', 'TAKE PLATFORM TOUR')}</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
              <span className="font-mono text-[11px] text-zinc-400">
                Instant Access · No Credit Card
              </span>
            </div>
          </div>
        </article>
      </div>

      {/* Trust Markers Bar */}
      <div className="mt-8 pt-6 border-t border-zinc-200 dark:border-white/10 flex flex-wrap items-center justify-center sm:justify-between gap-4 text-xs font-mono text-zinc-500 dark:text-zinc-400">
        <span className="uppercase tracking-wider">
          {language === 'ru' ? 'СТАНДАРТЫ НАДЕЖНОСТИ ПЛАТФОРМЫ:' : 'ENTERPRISE ASSURANCE:'}
        </span>
        <div className="flex flex-wrap items-center gap-6">
          {trustMarkers.map((marker, idx) => (
            <div key={idx} className="inline-flex items-center gap-2">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" />
              <span>{marker}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
