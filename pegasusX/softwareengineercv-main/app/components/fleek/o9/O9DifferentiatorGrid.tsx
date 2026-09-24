'use client';

import React from 'react';
import { Box, Layers, Shield, Zap, Cpu, Network, ArrowUpRight, LucideIcon } from 'lucide-react';
import type { TopicCard } from '@/app/data/topicTypes';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';

export interface DifferentiatorCardItem {
  id?: string;
  title: string;
  description: string;
  icon?: LucideIcon | React.ComponentType<{ className?: string }>;
  badge?: string;
  kicker?: string;
  colSpan?: 1 | 2;
  featured?: boolean;
  previewType?: 'graph' | 'telemetry' | 'code' | 'metric';
  previewValue?: string;
  href?: string;
}

export interface O9DifferentiatorGridProps {
  items?: (TopicCard | DifferentiatorCardItem)[];
  cards?: DifferentiatorCardItem[];
  kicker?: string;
  title?: string;
  description?: string;
  className?: string;
}

const DEFAULT_ICONS = [Layers, Zap, Shield, Box, Cpu, Network];

export default function O9DifferentiatorGrid({
  items,
  cards,
  kicker,
  title,
  description,
  className,
}: O9DifferentiatorGridProps) {
  const { language } = useLanguage();

  const resolvedCards: DifferentiatorCardItem[] = cards ?? (items as DifferentiatorCardItem[]) ?? [];
  if (!resolvedCards.length) return null;

  const resolvedKicker =
    kicker ??
    (language === 'ru' ? 'КЛЮЧЕВЫЕ ДИФФЕРЕНЦИАТОРЫ' : 'KEY DIFFERENTIATORS');

  const resolvedTitle =
    title ??
    (language === 'ru'
      ? 'Почему отраслевые лидеры выбирают Pegasus'
      : 'Why Industry Leaders Choose Pegasus');

  const resolvedDescription =
    description ??
    (language === 'ru'
      ? 'Детерминированное ядро исполнения, криптографическая изоляция арендаторов и отсутствие разрывов между шестью ролями.'
      : 'Deterministic state execution, cryptographic tenant isolation, and zero handoff gaps across all six network roles.');

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
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" aria-hidden />
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

      {/* Bento Grid */}
      <div className="mt-8 sm:mt-10 grid grid-cols-1 md:grid-cols-2 gap-4 sm:gap-6">
        {resolvedCards.slice(0, 6).map((card, i) => {
          const Icon = card.icon ?? DEFAULT_ICONS[i % DEFAULT_ICONS.length];
          const indexTag = card.kicker ?? `[0${i + 1}]`;
          const isFeatured = card.featured || card.colSpan === 2;

          return (
            <article
              key={card.title + i}
              className={cn(
                'relative p-6 sm:p-8 bg-white dark:bg-black/60 border border-zinc-200 dark:border-white/10 hover:border-emerald-500/50 dark:hover:border-emerald-500/50 transition-all duration-300 group flex flex-col justify-between shadow-sm dark:shadow-none',
                isFeatured && 'md:col-span-2'
              )}
            >
              {/* Tactical Corner Brackets */}
              <span
                className="absolute -top-1 -left-1 w-2.5 h-2.5 border-t-2 border-l-2 border-emerald-500/80 pointer-events-none group-hover:border-emerald-400 transition-colors"
                aria-hidden
              />
              <span
                className="absolute -bottom-1 -right-1 w-2.5 h-2.5 border-b-2 border-r-2 border-emerald-500/80 pointer-events-none group-hover:border-emerald-400 transition-colors"
                aria-hidden
              />

              <div>
                {/* Header row: Icon & Badges */}
                <div className="flex items-center justify-between gap-4 mb-5">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 flex items-center justify-center bg-emerald-500/10 dark:bg-emerald-950/40 border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 group-hover:scale-105 transition-transform duration-200">
                      <Icon className="w-5 h-5" aria-hidden />
                    </div>
                    <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500 font-medium">
                      {indexTag}
                    </span>
                  </div>

                  {card.badge && (
                    <span className="font-mono text-[10px] uppercase tracking-wider px-2 py-0.5 border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 bg-emerald-500/5 dark:bg-emerald-950/40">
                      {card.badge}
                    </span>
                  )}
                </div>

                {/* Card Title */}
                <h3 className="text-lg sm:text-xl font-semibold tracking-tight text-zinc-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
                  {card.title}
                </h3>

                {/* Card Body */}
                <p className="mt-2.5 text-sm sm:text-base leading-relaxed text-zinc-600 dark:text-zinc-300">
                  {card.description}
                </p>
              </div>

              {/* Optional Telemetry Preview or Link Trigger */}
              <div className="mt-6 pt-4 border-t border-zinc-100 dark:border-white/5 flex items-center justify-between text-xs font-mono">
                {card.previewType === 'telemetry' || card.previewValue ? (
                  <span className="text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-ping" />
                    <span>{card.previewValue ?? 'ACTIVE TELEMETRY: 99.98%'}</span>
                  </span>
                ) : (
                  <span className="text-zinc-400 dark:text-zinc-500 uppercase tracking-wider">
                    {language === 'ru' ? 'ПРОВЕРЕНО НА СЕТИ' : 'VERIFIED IN NETWORK'}
                  </span>
                )}

                {card.href && (
                  <span className="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-semibold group-hover:translate-x-0.5 transition-transform">
                    <span>{language === 'ru' ? 'ПОДРОБНЕЕ' : 'LEARN MORE'}</span>
                    <ArrowUpRight className="w-3.5 h-3.5" />
                  </span>
                )}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
