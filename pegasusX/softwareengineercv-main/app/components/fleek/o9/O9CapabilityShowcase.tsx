'use client';

import React from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { ArrowUpRight } from 'lucide-react';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';

export interface O9CapabilityCard {
  id?: string;
  title: string;
  description: string;
  href?: string;
  image?: string;
  tag?: string;
  workflowSteps?: string[];
  sla?: string;
}

export interface O9CapabilityShowcaseProps {
  items?: O9CapabilityCard[];
  kicker?: string;
  title?: string;
  label?: string;
  description?: string;
  className?: string;
}

export default function O9CapabilityShowcase({
  items = [],
  kicker,
  title,
  label,
  description,
  className,
}: O9CapabilityShowcaseProps) {
  const { language, t } = useLanguage();
  if (!items.length) return null;

  const resolvedKicker =
    kicker ??
    label ??
    (language === 'ru' ? 'КЛЮЧЕВЫЕ ВОЗМОЖНОСТИ' : 'CORE CAPABILITIES');

  const resolvedTitle =
    title ??
    (language === 'ru'
      ? 'Что обеспечивает платформа Pegasus'
      : 'What Pegasus Capabilities Enable');

  const resolvedDescription =
    description ??
    (language === 'ru'
      ? 'Модульные операционные компоненты для диспетчеризации, контроля загрузки и сквозной прослеживаемости.'
      : 'Modular operational capabilities built for intelligent dispatch, volumetric load-packing, and immutable visibility.');

  return (
    <section
      className={cn(
        'w-full pt-12 md:pt-16 border-t border-zinc-200 dark:border-white/10 transition-colors',
        className
      )}
      aria-label={resolvedTitle}
    >
      {/* Eyebrow & Title */}
      <div className="flex flex-col gap-4 max-w-4xl">
        <div className="inline-flex items-center gap-2">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
          <span className="font-mono text-[10px] sm:text-[11px] uppercase tracking-widest text-emerald-600 dark:text-emerald-400 font-medium">
            {resolvedKicker}
          </span>
        </div>

        <h2 className="text-2xl sm:text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-zinc-900 dark:text-white leading-[1.12]">
          {resolvedTitle}
        </h2>

        {resolvedDescription && (
          <p className="text-sm sm:text-base text-zinc-600 dark:text-zinc-400 leading-relaxed max-w-3xl">
            {resolvedDescription}
          </p>
        )}
      </div>

      {/* 3-Column Responsive Grid */}
      <div className="mt-8 sm:mt-10 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {items.slice(0, 6).map((item, idx) => {
          const imageSrc = item.image ?? EDITORIAL_IMAGES[idx % EDITORIAL_IMAGES.length];
          const itemTag = item.tag ?? (language === 'ru' ? 'Возможность' : 'Capability');
          const href = item.href ?? '#';

          return (
            <article
              key={item.title + idx}
              className="relative bg-white dark:bg-black/60 border border-zinc-200 dark:border-white/10 hover:border-emerald-500/50 dark:hover:border-emerald-500/50 transition-all duration-300 flex flex-col justify-between group shadow-sm dark:shadow-none overflow-hidden"
            >
              {/* Tactical Corner Brackets */}
              <span
                className="absolute -top-1 -left-1 w-2.5 h-2.5 border-t-2 border-l-2 border-emerald-500/80 pointer-events-none group-hover:border-emerald-400 transition-colors z-20"
                aria-hidden
              />
              <span
                className="absolute -bottom-1 -right-1 w-2.5 h-2.5 border-b-2 border-r-2 border-emerald-500/80 pointer-events-none group-hover:border-emerald-400 transition-colors z-20"
                aria-hidden
              />

              {/* Media Container */}
              <div className="relative aspect-[16/10] w-full overflow-hidden bg-zinc-900">
                <Image
                  src={imageSrc}
                  alt={item.title}
                  fill
                  className="object-cover transition-transform duration-500 group-hover:scale-105 opacity-90 dark:opacity-80"
                  sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent pointer-events-none" />

                {/* Tag Overlay */}
                <div className="absolute top-3 left-3 z-10">
                  <span className="font-mono text-[10px] uppercase tracking-wider px-2 py-0.5 bg-black/80 backdrop-blur-md border border-white/20 text-white font-medium">
                    [{itemTag}]
                  </span>
                </div>

                {/* Optional SLA Badge */}
                {item.sla && (
                  <div className="absolute bottom-3 left-3 z-10 max-w-[calc(100%-1.5rem)]">
                    <span className="font-mono text-[9px] sm:text-[10px] uppercase tracking-wider px-2 py-0.5 bg-emerald-950/80 backdrop-blur-md border border-emerald-500/40 text-emerald-400 truncate inline-block max-w-full">
                      SLA: {item.sla}
                    </span>
                  </div>
                )}
              </div>

              {/* Body Content */}
              <div className="p-5 sm:p-6 flex flex-col flex-1 justify-between gap-6">
                <div>
                  <h3 className="text-lg sm:text-xl font-semibold tracking-tight text-zinc-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors break-words">
                    {item.title}
                  </h3>

                  <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-300 leading-relaxed line-clamp-3 break-words">
                    {item.description}
                  </p>

                  {/* Optional Workflow Steps Preview */}
                  {item.workflowSteps && item.workflowSteps.length > 0 && (
                    <div className="mt-4 pt-3 border-t border-zinc-100 dark:border-white/5 flex flex-wrap gap-1">
                      {item.workflowSteps.map((step, sIdx) => {
                        const cleanStep = step.replace(/^\d+[\s.:-]*\s*/, '');
                        return (
                          <span
                            key={sIdx}
                            className="font-mono text-[9px] uppercase tracking-wider px-1.5 py-0.5 bg-zinc-100 dark:bg-white/5 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-white/5 break-words"
                          >
                            {String(sIdx + 1).padStart(2, '0')} {cleanStep}
                          </span>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* Card Footer Link */}
                <div className="pt-4 border-t border-zinc-100 dark:border-white/5 flex flex-wrap items-center justify-between gap-2">
                  <Link
                    href={href}
                    className="inline-flex items-center gap-1.5 font-mono text-xs uppercase tracking-wider text-emerald-600 dark:text-emerald-400 font-semibold group-hover:translate-x-0.5 transition-transform shrink-0"
                  >
                    <span>{t('btn_read_more', 'READ MORE')}</span>
                    <ArrowUpRight className="w-3.5 h-3.5" />
                  </Link>

                  <span className="font-mono text-[10px] text-zinc-400 dark:text-zinc-600 shrink-0">
                    [{String(idx + 1).padStart(2, '0')}]
                  </span>
                </div>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
