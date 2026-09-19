'use client';

import { useLanguage } from '@/app/context/LanguageContext';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import { O9SectionLabel } from './o9/O9Sections';

const FADE_UP = {
  hidden: { opacity: 0, y: 60, filter: 'blur(10px)' },
  visible: { 
    opacity: 1, 
    y: 0, 
    filter: 'blur(0px)',
    transition: { duration: 1, ease: [0.32, 0.72, 0, 1] } 
  }
};

const STAGGER_CONTAINER = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 }
  }
};

type Spec = { label: string; value: string };

type SpecPanelProps = {
  specs: Spec[];
  variant?: 'terminal' | 'grid';
};

export default function SpecPanel({ specs, variant = 'terminal' }: SpecPanelProps) {
  const { t } = useLanguage();

  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_specs_eyebrow', 'Technical Specifications')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-4xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_specs_title', 'System Architecture & Capabilities')}
      </motion.h2>

      <motion.div variants={FADE_UP}>
        <div className="p-[6px] rounded-[2rem] bg-white/[0.03] border border-white/10 group transition-all duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] hover:bg-white/[0.08] hover:border-white/20">
          <div className="relative h-full w-full rounded-[calc(2rem-6px)] bg-[#050505] overflow-hidden p-8 lg:p-12 transition-transform duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:scale-[0.98]">
            <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50 z-10" />
            
            {/* Terminal Header */}
            <div className="flex items-center gap-3 border-b border-white/10 pb-6 mb-6">
              <div className="flex gap-2">
                <span className="h-3 w-3 rounded-full bg-white/20" />
                <span className="h-3 w-3 rounded-full bg-white/40" />
                <span className="h-3 w-3 rounded-full bg-white/60" />
              </div>
              <span className="ml-4 font-mono text-xs text-white/40 uppercase tracking-widest">
                pegasus.sys.spec
              </span>
            </div>

            {/* Spec List */}
            <motion.dl variants={STAGGER_CONTAINER} className="grid grid-cols-1 md:grid-cols-2 gap-x-12 gap-y-6">
              {specs.map((spec) => (
                <motion.div key={spec.label} variants={FADE_UP} className="flex flex-col sm:flex-row sm:items-baseline justify-between border-b border-white/5 pb-4">
                  <dt className="font-mono text-[11px] uppercase tracking-widest text-white/40 mb-2 sm:mb-0">
                    {spec.label}
                  </dt>
                  <dd className="font-mono text-sm text-white/90 text-right">
                    {spec.value}
                  </dd>
                </motion.div>
              ))}
            </motion.dl>
          </div>
        </div>
      </motion.div>
    </motion.section>
  );
}
