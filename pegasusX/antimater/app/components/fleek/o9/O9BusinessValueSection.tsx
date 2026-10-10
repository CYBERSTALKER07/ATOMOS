'use client';

import React, { useState, useId } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { TrendingUp } from 'lucide-react';
import type { O9ValueTab, O9ValueStat } from '@/app/data/o9FleekDefaults';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';

export interface BusinessValueStatItem extends O9ValueStat {
  delta?: string;
  subtext?: string;
  trend?: string;
}

export interface BusinessValueTabItem {
  id: string;
  label: string;
  stats: BusinessValueStatItem[];
}

export interface O9BusinessValueSectionProps {
  tabs?: (O9ValueTab | BusinessValueTabItem)[];
  kicker?: string;
  title?: string;
  description?: string;
  className?: string;
}

export default function O9BusinessValueSection({
  tabs = [],
  kicker,
  title,
  description,
  className,
}: O9BusinessValueSectionProps) {
  const { language } = useLanguage();
  const tablistId = useId();

  const [activeId, setActiveId] = useState<string>(() => tabs[0]?.id ?? '');
  const activeTab = tabs.find((tab) => tab.id === activeId) ?? tabs[0];

  if (!tabs.length || !activeTab) return null;

  const resolvedKicker =
    kicker ??
    (language === 'ru' ? 'ИЗМЕРИМЫЙ БИЗНЕС-ЭФФЕКТ' : 'QUANTIFIED BUSINESS VALUE');

  const resolvedTitle =
    title ??
    (language === 'ru'
      ? 'Измеримые результаты внедрения на платформе Pegasus'
      : 'Measurable Operational & Capital Outcomes');

  const resolvedDescription =
    description ??
    (language === 'ru'
      ? 'Проверенные показатели производительности сети, сокращения дефицита и высвобождения оборотного капитала.'
      : 'Verified benchmarks across network velocity, stockout prevention, touchless dispatch, and working capital release.');

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

      {/* Interactive Tabs Row */}
      {tabs.length > 1 && (
        <div
          id={tablistId}
          role="tablist"
          aria-label="Business value tabs"
          className="mt-8 flex flex-wrap items-center gap-2 border-b border-zinc-200 dark:border-white/10 pb-4"
        >
          {tabs.map((tab) => {
            const isSelected = tab.id === activeTab.id;
            return (
              <button
                key={tab.id}
                type="button"
                role="tab"
                aria-selected={isSelected}
                aria-controls={`value-tabpanel-${tab.id}`}
                onClick={() => setActiveId(tab.id)}
                className={cn(
                  'px-3 sm:px-4 py-1.5 sm:py-2 font-mono text-xs sm:text-sm uppercase tracking-wider transition-all duration-200 cursor-pointer relative border rounded-none',
                  isSelected
                    ? 'border-emerald-500 bg-emerald-500/10 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 font-semibold shadow-sm'
                    : 'border-transparent text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white hover:border-zinc-300 dark:hover:border-white/20'
                )}
              >
                {isSelected && (
                  <span
                    className="absolute -top-1 -left-1 w-1.5 h-1.5 border-t border-l border-emerald-500"
                    aria-hidden
                  />
                )}
                {tab.label}
              </button>
            );
          })}
        </div>
      )}

      {/* Tab Panel Content: 4-Column Stat Bento Cards */}
      <AnimatePresence mode="wait">
        <motion.div
          key={activeTab.id}
          id={`value-tabpanel-${activeTab.id}`}
          role="tabpanel"
          aria-labelledby={tablistId}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -10 }}
          transition={{ duration: 0.24, ease: [0.16, 1, 0.3, 1] }}
          className="mt-8 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6"
        >
          {activeTab.stats.map((stat, idx) => {
            return (
              <article
                key={`${stat.label}-${idx}`}
                className="relative p-5 sm:p-6 md:p-7 bg-white dark:bg-black/60 border border-zinc-200 dark:border-white/10 hover:border-emerald-500/40 dark:hover:border-emerald-500/40 transition-all duration-300 flex flex-col justify-between group shadow-sm dark:shadow-none"
              >
                {/* Tactical Corner Brackets */}
                <span
                  className="absolute -top-1 -left-1 w-2 h-2 border-t border-l border-emerald-500/70 pointer-events-none group-hover:border-emerald-400 transition-colors"
                  aria-hidden
                />
                <span
                  className="absolute -bottom-1 -right-1 w-2 h-2 border-b border-r border-emerald-500/70 pointer-events-none group-hover:border-emerald-400 transition-colors"
                  aria-hidden
                />

                {/* Context Description on top */}
                {stat.context && (
                  <p className="text-xs sm:text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed min-h-[3rem] break-words">
                    {stat.context}
                  </p>
                )}

                {/* Big Metric Stat and Delta */}
                <div className="mt-6 pt-4 border-t border-zinc-100 dark:border-white/5">
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="font-mono text-2xl sm:text-3xl lg:text-4xl xl:text-5xl font-semibold tracking-tight text-emerald-600 dark:text-emerald-400 break-words">
                      {stat.value}
                    </span>

                    {(stat as BusinessValueStatItem).delta && (
                      <span className="font-mono text-[10px] text-emerald-600 dark:text-emerald-400 flex items-center gap-0.5 bg-emerald-500/10 px-1.5 py-0.5 shrink-0">
                        <TrendingUp className="w-3 h-3 shrink-0" />
                        <span>{(stat as BusinessValueStatItem).delta}</span>
                      </span>
                    )}
                  </div>

                  {/* Metric Label */}
                  <p className="mt-2 font-mono text-[11px] sm:text-xs uppercase tracking-wider text-zinc-500 dark:text-zinc-400 font-medium break-words">
                    {stat.label}
                  </p>

                  {(stat as BusinessValueStatItem).subtext && (
                    <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400 leading-normal break-words">
                      {(stat as BusinessValueStatItem).subtext}
                    </p>
                  )}
                </div>
              </article>
            );
          })}
        </motion.div>
      </AnimatePresence>
    </section>
  );
}
