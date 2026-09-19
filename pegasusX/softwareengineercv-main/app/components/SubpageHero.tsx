'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';
import TopicCanvas from './visuals/TopicCanvas';
import { useLanguage } from '../context/LanguageContext';

export type SubpageHeroProps = {
  topicSlug?: string;
  categoryLabel?: string;
  categoryHref?: string;
  badge?: string;
  badgeIcon?: ReactNode;
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
  widget?: any;
  nodes?: any;
  breadcrumb?: any;
};

export default function SubpageHero({
  topicSlug,
  categoryLabel,
  categoryHref,
  badge,
  title,
  summary,
  primaryCta,
  secondaryCta,
}: SubpageHeroProps) {
  const { language } = useLanguage();
  const eyebrowBadge = badge || (categoryLabel ? `PEGASUS // ${categoryLabel.toUpperCase()}` : 'PEGASUS OS');
  
  const primaryButton = primaryCta || {
    label: language === 'ru' ? 'Запросить демо' : 'Request Demo',
    href: '/join',
  };

  const secondaryButton = secondaryCta || {
    label: language === 'ru' ? 'Обзор платформы' : 'Explore Stack',
    href: categoryHref || '/platform',
  };

  return (
    <section className="min-h-screen w-full relative flex flex-col justify-center bg-[#000000] overflow-hidden pt-20 sm:pt-24 pb-0">
      <div className="w-full relative z-10">
        <div className="relative grid grid-cols-1 lg:grid-cols-2 bg-[#000000] w-full min-h-[calc(100vh-5rem)]">
          
          {/* LEFT COLUMN: Editorial Headline, Subtitle, Description & Outlined CTA */}
          <div className="flex flex-col justify-end p-8 sm:p-12 lg:p-16 xl:p-24 relative z-10 min-h-[540px] lg:min-h-[640px] xl:min-h-[700px]">
            
            {/* Eyebrow */}
            <div className="mb-8 flex items-center">
              <div className="h-1.5 w-1.5 bg-white mr-3"></div>
              <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-white/50">
                {eyebrowBadge}
              </span>
            </div>

            <div className="space-y-6">
              {/* Primary Headline */}
              <h1 className="text-4xl sm:text-5xl md:text-6xl lg:text-[4.75rem] font-medium tracking-tight text-white leading-[1.05]">
                {title}
              </h1>

              {/* Subtitle Description */}
              <p className="text-base sm:text-lg md:text-xl font-light leading-relaxed max-w-2xl pt-2 text-white/60">
                {summary}
              </p>
            </div>

            {/* Action Buttons */}
            <div className="pt-10 flex flex-wrap items-center gap-4">
              <Link
                href={primaryButton.href}
                className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white text-black hover:bg-white/90 rounded-none"
              >
                <span>{primaryButton.label}</span>
                <span className="text-lg leading-none mt-[-2px]">›</span>
              </Link>

              <Link
                href={secondaryButton.href}
                className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white/5 text-white hover:bg-white/10 border border-white/10 rounded-none"
              >
                <span>{secondaryButton.label}</span>
                <span className="text-lg leading-none mt-[-2px]">›</span>
              </Link>
            </div>
          </div>

          {/* RIGHT COLUMN: Full Wide Generated Topic Visual */}
          <div className="flex flex-col justify-between relative overflow-hidden bg-[#050505] min-h-[450px] lg:min-h-full w-full h-full">
            <TopicCanvas slug={topicSlug || 'default'} />
          </div>
          
        </div>
      </div>
    </section>
  );
}
