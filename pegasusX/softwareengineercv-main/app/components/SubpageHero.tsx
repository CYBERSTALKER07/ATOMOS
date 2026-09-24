'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { usePathname } from 'next/navigation';
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
  widget?: ReactNode | { title?: string; description?: string; href?: string; [key: string]: unknown };
  nodes?: ReactNode | unknown;
  breadcrumb?: {
    homeLabel?: string;
    categoryLabel?: string;
    categoryHref?: string;
    subCategoryLabel?: string;
    subCategoryHref?: string;
    currentPage?: string;
    [key: string]: unknown;
  };
  imageSrc?: string;
  imageAlt?: string;
  footerTickerText?: string;
};

export default function SubpageHero({
  topicSlug,
  categoryLabel,
  categoryHref,
  title,
  summary,
  primaryCta,
  secondaryCta,
  breadcrumb,
  imageSrc,
  imageAlt,
  footerTickerText,
}: SubpageHeroProps) {
  const { language } = useLanguage();
  const pathname = usePathname();
  const pathParts = pathname?.split('/').filter(Boolean) || [];
  const primarySection = pathParts[0] || '';
  const slugFromPath = pathParts[pathParts.length - 1];
  const activeSlug = topicSlug || slugFromPath || 'default';

  // Category mapping fallback
  const defaultCategoryMap: Record<string, { label: string; href: string }> = {
    platform: { label: 'Platform', href: '/platform' },
    technology: { label: 'Technology', href: '/technology' },
    solutions: { label: 'Solutions', href: '/solutions' },
    capabilities: { label: 'Capabilities', href: '/capabilities' },
    roles: { label: 'Roles', href: '/roles' },
    operations: { label: 'Operations', href: '/operations' },
    'ai-vision': { label: 'AI & Vision', href: '/ai-vision' },
    'apps-deploy': { label: 'Apps & Deploy', href: '/apps-deploy' },
    markets: { label: 'Markets', href: '/markets' },
    compare: { label: 'Compare', href: '/compare' },
    'supply-chain-software': { label: 'Solutions', href: '/solutions' },
    'logistics-automation': { label: 'Solutions', href: '/solutions' },
    'global-logistics': { label: 'Solutions', href: '/solutions' },
  };

  const detectedCategory = defaultCategoryMap[primarySection] || {
    label: categoryLabel || 'Solutions',
    href: categoryHref || `/${primarySection || 'solutions'}`,
  };

  const resolvedCategoryLabel = breadcrumb?.categoryLabel || categoryLabel || detectedCategory.label;
  const resolvedCategoryHref = breadcrumb?.categoryHref || categoryHref || detectedCategory.href;
  
  // Subcategory detection (e.g. /solutions/industry/retail or custom)
  const resolvedSubCategory =
    breadcrumb?.subCategoryLabel ||
    (pathParts.length > 2 ? pathParts[1].replace(/-/g, ' ').toUpperCase() : null);
  const resolvedSubCategoryHref =
    breadcrumb?.subCategoryHref ||
    (pathParts.length > 2 ? `/${pathParts[0]}/${pathParts[1]}` : undefined);

  const currentTitle = breadcrumb?.currentPage || title;

  const primaryButton = primaryCta || {
    label: language === 'ru' ? 'Запросить демо' : 'Request Demo',
    href: '/join',
  };

  const secondaryButton = secondaryCta || {
    label: language === 'ru' ? 'Обзор платформы' : 'Take Platform Tour',
    href: resolvedCategoryHref || '/platform',
  };

  return (
    <section className="w-full relative flex flex-col justify-between bg-[#000000] overflow-hidden pt-6 sm:pt-8 lg:pt-10 pb-8 sm:pb-12">
      <div className="w-full flex-1 flex flex-col justify-center">
        <div className="relative grid grid-cols-1 lg:grid-cols-2 bg-[#000000] w-full items-center">
          
          {/* LEFT COLUMN: Breadcrumbs, Title, Summary, CTAs, Gartner Peer Insights Card */}
          <div className="flex flex-col justify-center px-5 sm:px-12 lg:px-16 xl:px-20 py-8 lg:py-16 relative z-10">
            
            {/* Breadcrumbs Navigation */}
            <nav aria-label="Breadcrumb" className="mb-6 sm:mb-8 flex flex-wrap items-center gap-2 font-mono text-[11px] uppercase tracking-wider text-zinc-400">
              <Link
                href="/"
                className="hover:text-white transition-colors flex items-center"
                aria-label="Home"
              >
                <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
                  <polyline points="9 22 9 12 15 12 15 22" />
                </svg>
              </Link>

              <span className="text-zinc-600">/</span>

              {resolvedCategoryLabel ? (
                <>
                  <Link
                    href={resolvedCategoryHref}
                    className="hover:text-white transition-colors"
                  >
                    {resolvedCategoryLabel.toUpperCase()}
                  </Link>
                  <span className="text-zinc-600">/</span>
                </>
              ) : null}

              {resolvedSubCategory ? (
                <>
                  {resolvedSubCategoryHref ? (
                    <Link
                      href={resolvedSubCategoryHref}
                      className="hover:text-white transition-colors"
                    >
                      {resolvedSubCategory.length > 20
                        ? `${resolvedSubCategory.slice(0, 18)}...`
                        : resolvedSubCategory}
                    </Link>
                  ) : (
                    <span>
                      {resolvedSubCategory.length > 20
                        ? `${resolvedSubCategory.slice(0, 18)}...`
                        : resolvedSubCategory}
                    </span>
                  )}
                  <span className="text-zinc-600">/</span>
                </>
              ) : null}

              <span className="text-zinc-300 truncate max-w-[200px] sm:max-w-[320px]">
                {currentTitle.toUpperCase()}
              </span>
            </nav>

            {/* Typography Section */}
            <div className="space-y-4 sm:space-y-5">
              {/* Primary Headline */}
              <h1 className="text-2xl sm:text-4xl md:text-5xl lg:text-[3.25rem] xl:text-[3.75rem] font-medium tracking-tight text-white leading-[1.12] break-words">
                {title}
              </h1>

              {/* Subtitle Description */}
              <p className="text-sm sm:text-base md:text-lg text-zinc-300/80 font-light leading-relaxed max-w-xl">
                {summary}
              </p>
            </div>

            {/* Action Buttons */}
            <div className="pt-6 sm:pt-10 flex flex-col sm:flex-row items-stretch sm:items-center gap-3 sm:gap-4">
              <Link
                href={primaryButton.href}
                className="inline-flex items-center justify-center px-6 sm:px-7 py-3 sm:py-3.5 text-xs sm:text-sm font-semibold tracking-wider uppercase bg-white text-black hover:bg-zinc-200 transition-colors rounded-none text-center"
              >
                {primaryButton.label.toUpperCase()}
              </Link>

              <Link
                href={secondaryButton.href}
                className="inline-flex items-center justify-center px-6 sm:px-7 py-3 sm:py-3.5 text-xs sm:text-sm font-semibold tracking-wider uppercase bg-transparent text-white hover:bg-white/10 border border-white transition-colors rounded-none text-center"
              >
                {secondaryButton.label.toUpperCase()}
              </Link>
            </div>

            {/* Trust / Reviews Card (Gartner Peer Insights) */}
            <div className="mt-8 sm:mt-10 max-w-md">
              <div className="bg-[#0c0c0e] border border-white/10 p-4 sm:p-5">
                <div className="flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <span className="text-2xl sm:text-3xl font-bold text-white tracking-tight leading-none">
                      4.8
                    </span>
                    <div className="flex flex-col">
                      <span className="text-xs font-bold text-white tracking-tight leading-none">
                        Gartner
                      </span>
                      <span className="text-[10px] text-zinc-400 font-medium tracking-wider leading-none mt-1">
                        Peer Insights<span className="text-[8px]">™</span>
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 text-white" aria-label="4.8 out of 5 stars">
                    {[1, 2, 3, 4, 5].map((star) => (
                      <svg
                        key={star}
                        className="w-3.5 h-3.5 fill-white text-white"
                        viewBox="0 0 24 24"
                      >
                        <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
                      </svg>
                    ))}
                  </div>
                </div>
                <p className="mt-2.5 text-[11px] sm:text-xs text-zinc-400 font-light leading-snug">
                  {language === 'ru'
                    ? '202+ отзыва корпоративных клиентов в области решений для планирования цепочек поставок на 23 сентября 2026 г.'
                    : '202+ customer reviews in Supply Chain Planning Solutions as of September 23, 2026'}
                </p>
              </div>
            </div>

          </div>

          {/* RIGHT COLUMN: Clean Full Visual Presentation */}
          <div className="flex items-center justify-center p-6 sm:p-8 lg:p-10 xl:p-12 w-full h-full">
            <div className="relative w-full aspect-[4/3] max-w-2xl overflow-hidden bg-zinc-950">
              {imageSrc ? (
                <Image
                  src={imageSrc}
                  alt={imageAlt || title}
                  fill
                  priority
                  sizes="(max-width: 1024px) 100vw, 50vw"
                  className="object-cover object-center"
                />
              ) : (
                <TopicCanvas slug={activeSlug} />
              )}
            </div>
          </div>
          
        </div>
      </div>
    </section>
  );
}
