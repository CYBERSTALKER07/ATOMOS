'use client';

import { motion } from 'framer-motion';
import type { TopicCard } from '@/app/data/topicTypes';
import { cn } from '@/app/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';

const FADE_UP = {
  hidden: { opacity: 0, y: 30 },
  visible: { 
    opacity: 1, 
    y: 0,
    transition: {
      duration: 1,
      ease: [0.16, 1, 0.3, 1] as const,
    }
  }
};

const STAGGER_CONTAINER = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    }
  }
};

export function O9SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-8 flex items-center">
      <div className="h-1.5 w-1.5 bg-white mr-3"></div>
      <span className="font-mono text-[10px] uppercase tracking-widest text-white/50">
        {children}
      </span>
    </div>
  );
}

// --- Wide Stacked Layout (Replacing Cards) ---
export function O9CapabilitiesGrid({
  capabilities,
  title
}: {
  capabilities: TopicCard[];
  title?: string;
}) {
  const { t } = useLanguage();
  if (!capabilities.length) return null;
  const resolvedLabel = t('sec_core_capabilities_label');
  const resolvedTitle = title || t('sec_core_capabilities_title');

  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{resolvedLabel}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {resolvedTitle}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {capabilities.map((cap, i) => (
          <motion.div 
            key={cap.title} 
            variants={FADE_UP}
            className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 md:py-16 border-t border-white/10"
          >
            <div className="md:col-span-1">
              <span className="font-mono text-[10px] uppercase tracking-widest text-white/30">
                {String(i + 1).padStart(2, '0')} //
              </span>
            </div>
            <div className="md:col-span-4">
              <h3 className="font-medium text-white text-3xl lg:text-4xl leading-tight">
                {cap.title}
              </h3>
            </div>
            <div className="md:col-span-7">
              <p className="font-light text-white/60 leading-relaxed text-xl max-w-3xl">
                {cap.description}
              </p>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

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
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {t('sec_key_differentiators_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {items.map((item, i) => (
          <motion.div key={`${item.title}-${i}`} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10">
            <div className="md:col-span-5">
              <h3 className="text-3xl lg:text-4xl font-medium text-white">{item.title}</h3>
            </div>
            <div className="md:col-span-7">
              <p className="text-xl font-light leading-relaxed text-white/60">{item.description}</p>
            </div>
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
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {t('sec_how_it_works_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {steps.map((step, i) => (
          <motion.div key={step.title} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10">
            <div className="md:col-span-1">
              <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-none border border-white/20 bg-white/5 font-mono text-xs text-white/70">
                {String(i + 1).padStart(2, '0')}
              </span>
            </div>
            <div className="md:col-span-4">
              <h3 className="text-3xl font-medium text-white">{step.title}</h3>
            </div>
            <div className="md:col-span-7">
              <p className="text-xl font-light leading-relaxed text-white/60">{step.description}</p>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

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
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {t('sec_edge_cases_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {items.map((item, i) => (
          <motion.div key={item.title} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10">
            <div className="md:col-span-1">
              <div className="mt-2 h-2 w-2 rounded-full bg-red-500/50 shadow-[0_0_15px_rgba(239,68,68,0.6)]" />
            </div>
            <div className="md:col-span-4">
              <h3 className="text-3xl font-medium text-white">{item.title}</h3>
            </div>
            <div className="md:col-span-7">
              <p className="text-xl font-light leading-relaxed text-white/60">{item.description}</p>
            </div>
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
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {t('sec_ai_data_title')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {items.map((item, i) => (
          <motion.div key={item.title} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10">
            <div className="md:col-span-1">
               <div className="mt-2 h-4 w-4 rounded-none border border-white/20 bg-white/5 flex items-center justify-center">
                 <div className="h-1 w-1 bg-white" />
               </div>
            </div>
            <div className="md:col-span-4">
               <h3 className="text-3xl font-medium text-white">{item.title}</h3>
            </div>
            <div className="md:col-span-7">
               <p className="text-xl font-light text-white/60 leading-relaxed">
                 {item.description}
               </p>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </motion.section>
  );
}

export function O9WhyItMatters({
  why,
  problemFallback
}: {
  why?: string[];
  problemFallback?: string;
}) {
  const { t } = useLanguage();
  if (!why?.length && !problemFallback) return null;
  
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_why_it_matters_label', 'Why it matters')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-24">
        {t('sec_why_it_matters_title', 'The impact of optimization')}
      </motion.h2>

      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {why && why.length > 0 ? why.map((item, i) => (
          <motion.div key={i} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10">
            <div className="md:col-span-1">
              <span className="font-mono text-[10px] uppercase tracking-widest text-white/30">
                {String(i + 1).padStart(2, '0')} //
              </span>
            </div>
            <div className="md:col-span-11">
              <p className="text-2xl lg:text-3xl font-light text-white/80 leading-relaxed max-w-4xl">
                {item}
              </p>
            </div>
          </motion.div>
        )) : (
          <motion.div variants={FADE_UP} className="py-12 border-t border-white/10">
             <p className="text-2xl lg:text-3xl font-light text-white/80 leading-relaxed max-w-4xl">
                {problemFallback}
             </p>
          </motion.div>
        )}
      </motion.div>
    </motion.section>
  );
}
