'use client';

import React from 'react';
import type { TopicCard, WhyItMatters } from '@/app/data/topicTypes';
import { cn } from '@/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';
import { motion } from 'framer-motion';

// --- Awwwards Tier Fluid Dynamics ---
const FADE_UP = {
  hidden: { opacity: 0, y: 60, filter: 'blur(10px)' },
  visible: { 
    opacity: 1, 
    y: 0, 
    filter: 'blur(0px)',
    transition: { duration: 1, ease: [0.32, 0.72, 0, 1] as const } 
  }
};

const STAGGER_CONTAINER = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.15 }
  }
};

// --- Haptic Micro-Aesthetics: Double-Bezel Architecture ---
const DoubleBezelCard = ({ children, className, innerClassName }: { children: React.ReactNode, className?: string, innerClassName?: string }) => (
  <div 
    className={cn(
      "p-[6px] rounded-[2rem] bg-white/[0.03] border border-white/10 group transition-all duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] hover:bg-white/[0.08] hover:border-white/20",
      className
    )}
  >
    <div 
      className={cn(
        "relative h-full w-full rounded-[calc(2rem-6px)] bg-[#050505] overflow-hidden p-8 lg:p-10 transition-transform duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:scale-[0.98]",
        innerClassName
      )}
    >
      {/* Subtle top inner highlight */}
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50" />
      <div className="relative z-10 h-full flex flex-col">
        {children}
      </div>
    </div>
  </div>
);

// --- Eyebrow Pill ---
export function O9SectionLabel({ children }: { children: string }) {
  return (
    <motion.div variants={FADE_UP} className="inline-flex items-center rounded-full px-4 py-1.5 text-[10px] uppercase tracking-[0.2em] font-medium border border-white/10 bg-white/5 text-white/60 mb-8">
      {children}
    </motion.div>
  );
}

// --- The Editorial Split ---
export function O9WhyItMatters({
  why,
  problemFallback,
}: {
  why?: WhyItMatters;
  problemFallback: string;
}) {
  const { t } = useLanguage();
  const headline = why?.headline ?? t('sec_why_it_matters_title');
  const body = why?.body ?? problemFallback;
  const insights = why?.insights ?? [];

  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <div className="flex flex-col lg:flex-row gap-16 lg:gap-32">
        {/* Left side: Massive Typography */}
        <div className="w-full lg:w-1/2 flex flex-col items-start">
          <O9SectionLabel>{t('sec_why_it_matters_label')}</O9SectionLabel>
          <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white leading-[1.05]">
            {headline}
          </motion.h2>
          <motion.p variants={FADE_UP} className="mt-8 text-lg sm:text-xl font-light leading-relaxed text-white/50 max-w-lg">
            {body}
          </motion.p>
        </div>
        
        {/* Right side: Stacked Double-Bezel Cards */}
        {insights.length > 0 && (
          <motion.div variants={STAGGER_CONTAINER} className="w-full lg:w-1/2 flex flex-col gap-6 pt-12 lg:pt-0">
            {insights.map((insight, i) => (
              <motion.div key={insight.title} variants={FADE_UP}>
                <DoubleBezelCard className="w-full">
                  <div className="flex items-center gap-4 mb-6">
                    <div className="h-8 w-8 rounded-full border border-white/10 bg-white/5 flex items-center justify-center font-mono text-[10px] text-white/50">
                      0{i + 1}
                    </div>
                    <h3 className="text-xl sm:text-2xl font-medium text-white">
                      {insight.title}
                    </h3>
                  </div>
                  <p className="text-sm sm:text-base font-light leading-relaxed text-white/50 pl-12">
                    {insight.body}
                  </p>
                </DoubleBezelCard>
              </motion.div>
            ))}
          </motion.div>
        )}
      </div>
    </motion.section>
  );
}

// --- The Asymmetrical Bento Grid ---
export function O9CapabilityGrid({
  capabilities,
  label,
  title,
}: {
  capabilities: TopicCard[];
  label?: string;
  title?: string;
}) {
  const { t } = useLanguage();
  if (!capabilities.length) return null;

  const resolvedLabel = label ?? 'CORE CAPABILITIES';
  const resolvedTitle = title ?? 'What this solution enables';

  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{resolvedLabel}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {resolvedTitle}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {capabilities.map((cap, i) => (
          <motion.div 
            key={cap.title} 
            variants={FADE_UP}
            className={cn(
              "flex",
              i === 0 && capabilities.length >= 3 ? "md:col-span-2 lg:col-span-2 lg:row-span-2" : "col-span-1"
            )}
          >
            <DoubleBezelCard 
              className="w-full flex flex-col"
              innerClassName={cn(
                "flex flex-col justify-end",
                i === 0 && capabilities.length >= 3 ? "p-12 lg:p-16" : "p-8 lg:p-10"
              )}
            >
              <div className="mb-auto pb-12">
                <span className="font-mono text-[10px] uppercase tracking-widest text-white/30">
                  {String(i + 1).padStart(2, '0')} // CAPABILITY
                </span>
              </div>
              <h3 className={cn("font-medium text-white mb-4", i === 0 && capabilities.length >= 3 ? "text-3xl lg:text-4xl" : "text-xl lg:text-2xl")}>
                {cap.title}
              </h3>
              <p className={cn("font-light text-white/50 leading-relaxed", i === 0 && capabilities.length >= 3 ? "text-lg lg:text-xl max-w-xl" : "text-sm sm:text-base")}>
                {cap.description}
              </p>
            </DoubleBezelCard>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

// --- Soft Structuralism List ---
export function O9DifferentiatorList({ items }: { items: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items.length) return null;
  
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_key_differentiators_label')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_key_differentiators_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {items.map((item, i) => (
          <motion.div key={`${item.title}-${i}`} variants={FADE_UP}>
            <DoubleBezelCard>
              <h3 className="text-2xl font-medium text-white mb-4">{item.title}</h3>
              <p className="text-base font-light leading-relaxed text-white/50">{item.description}</p>
            </DoubleBezelCard>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

export function O9HowItWorks({
  steps
}: {
  steps: { title: string; description: string }[];
}) {
  const { t } = useLanguage();
  if (!steps.length) return null;
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_how_it_works_label')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_how_it_works_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {steps.map((step, i) => (
          <motion.div key={step.title} variants={FADE_UP}>
            <DoubleBezelCard className="h-full">
              <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/5 font-mono text-xs text-white/70 mb-8">
                {String(i + 1).padStart(2, '0')}
              </span>
              <h3 className="text-xl font-medium text-white mb-4">{step.title}</h3>
              <p className="text-sm font-light leading-relaxed text-white/50">{step.description}</p>
            </DoubleBezelCard>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

// --- Asymmetrical Bento Grid 2 ---
export function O9EdgeCaseGrid({ items }: { items?: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items?.length) return null;
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_edge_cases_label')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_edge_cases_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {items.map((item, i) => (
          <motion.div 
            key={item.title} 
            variants={FADE_UP}
            className={cn(
              "flex",
              i === 0 ? "md:col-span-2 lg:col-span-2" : "col-span-1"
            )}
          >
            <DoubleBezelCard 
              className="w-full flex flex-col"
              innerClassName={cn(
                "flex flex-col justify-end",
                i === 0 ? "p-10 lg:p-14" : "p-8"
              )}
            >
              <div className="mb-auto pb-12">
                <div className="h-2 w-2 rounded-full bg-red-500/50 shadow-[0_0_15px_rgba(239,68,68,0.6)]" />
              </div>
              <h3 className={cn("font-medium text-white mb-4", i === 0 ? "text-3xl lg:text-4xl" : "text-xl")}>
                {item.title}
              </h3>
              <p className={cn("font-light text-white/50 leading-relaxed", i === 0 ? "text-lg max-w-2xl" : "text-sm")}>
                {item.description}
              </p>
            </DoubleBezelCard>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

export function O9AiDataPanel({ items }: { items?: TopicCard[] }) {
  const { t } = useLanguage();
  if (!items?.length) return null;
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_ai_data_label')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_ai_data_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {items.map((item, i) => (
          <motion.div key={item.title} variants={FADE_UP}>
            <DoubleBezelCard className="h-full">
               <div className="h-10 w-10 rounded-full border border-white/10 bg-white/5 flex items-center justify-center mb-12">
                 <div className="h-1.5 w-1.5 rounded-full bg-white shadow-[0_0_12px_rgba(255,255,255,1)]" />
               </div>
               <h3 className="text-2xl font-medium text-white mb-4">{item.title}</h3>
               <p className="text-base font-light text-white/50 leading-relaxed">
                 {item.description}
               </p>
            </DoubleBezelCard>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}
