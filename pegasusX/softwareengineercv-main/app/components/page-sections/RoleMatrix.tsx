'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';
import { motion, AnimatePresence } from 'framer-motion';
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

type RoleRow = { role: string; touchpoint: string };

type RoleMatrixProps = {
  crossRole: RoleRow[];
  variant?: 'tabs' | 'table';
};

export default function RoleMatrix({ crossRole, variant = 'tabs' }: RoleMatrixProps) {
  const [active, setActive] = useState(0);
  const { t } = useLanguage();
  const current = crossRole[active];

  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <div className="flex flex-col lg:flex-row gap-16 lg:gap-24">
        {/* Left Side: Header and Controls */}
        <div className="w-full lg:w-5/12 flex flex-col">
          <O9SectionLabel>{t('sec_roles_eyebrow', 'Ecosystem Roles')}</O9SectionLabel>
          <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white leading-[1.05] mb-12">
            {t('sec_roles_title', 'Multi-Party Orchestration')}
          </motion.h2>

          {/* Interactive Pills */}
          <motion.div variants={FADE_UP} className="flex flex-wrap gap-3">
            {crossRole.map((row, i) => (
              <button
                key={row.role}
                type="button"
                onClick={() => setActive(i)}
                className={cn(
                  'relative rounded-full px-6 py-3 text-[11px] font-mono uppercase tracking-[0.15em] transition-all duration-500 overflow-hidden group',
                  i === active
                    ? 'text-black bg-white scale-105'
                    : 'text-white/60 bg-white/[0.03] border border-white/10 hover:bg-white/[0.08] hover:text-white'
                )}
              >
                <span className="relative z-10">{row.role}</span>
              </button>
            ))}
          </motion.div>
        </div>

        {/* Right Side: Active Panel (Double-Bezel Card) */}
        <div className="w-full lg:w-7/12 pt-8 lg:pt-0">
          <motion.div variants={FADE_UP} className="h-full">
            <div className="p-[6px] rounded-[2rem] bg-white/[0.03] border border-white/10 group transition-all duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] hover:bg-white/[0.08] hover:border-white/20 h-full">
              <div className="relative h-full w-full rounded-[calc(2rem-6px)] bg-[#050505] overflow-hidden p-10 lg:p-16 transition-transform duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:scale-[0.98] flex flex-col justify-center min-h-[300px]">
                <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50 z-10" />
                
                <AnimatePresence mode="wait">
                  <motion.div
                    key={active}
                    initial={{ opacity: 0, y: 20, filter: 'blur(4px)' }}
                    animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
                    exit={{ opacity: 0, y: -20, filter: 'blur(4px)' }}
                    transition={{ duration: 0.5, ease: [0.32, 0.72, 0, 1] }}
                  >
                    <p className="font-mono text-xs uppercase tracking-[0.2em] text-white/30 mb-6">
                      // {current.role} Touchpoint
                    </p>
                    <p className="text-2xl sm:text-3xl lg:text-4xl font-light text-white leading-[1.3]">
                      {current.touchpoint}
                    </p>
                  </motion.div>
                </AnimatePresence>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </motion.section>
  );
}
