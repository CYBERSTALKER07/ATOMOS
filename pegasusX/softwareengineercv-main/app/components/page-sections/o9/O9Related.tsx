'use client';

import Link from 'next/link';
import { motion } from 'framer-motion';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { O9SectionLabel } from './O9Sections';
import { useLanguage } from '@/app/context/LanguageContext';
import { cn } from '@/lib/utils';
import Image from 'next/image';

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

const FLEET_FLOWS: FlowVariant[] = ['fleetMap', 'dispatchBoard'];

function imageFor(flow: FlowVariant, index: number) {
  if (FLEET_FLOWS.includes(flow)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

export function O9RelatedUseCases({
  siblings,
  categoryLabel,
  flow,
}: {
  siblings: TopicPage[];
  categoryLabel: string;
  flow: FlowVariant;
}) {
  const { language, t } = useLanguage();
  if (!siblings.length) return null;
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <O9SectionLabel>{t('sec_use_cases', 'use cases')}</O9SectionLabel>
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-5xl leading-[1.05] mb-16 lg:mb-24">
        {t('sec_related_use_cases', 'Related use cases on the Pegasus platform')}
      </motion.h2>
      
      <motion.div variants={STAGGER_CONTAINER} className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {siblings.slice(0, 6).map((s, i) => {
          const content = s.content[language] || s.content.en;
          const imgSrc = imageFor(flow, i);
          return (
            <motion.div key={s.slug} variants={FADE_UP} className="h-full">
              <Link href={s.href} className="group block h-full">
                <div className="p-[6px] rounded-[2rem] bg-white/[0.03] border border-white/10 transition-all duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:bg-white/[0.08] group-hover:border-white/20 h-full">
                  <div className="relative h-full w-full rounded-[calc(2rem-6px)] bg-[#050505] overflow-hidden transition-transform duration-700 ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:scale-[0.98] flex flex-col">
                    <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50 z-10" />
                    
                    {/* Image Area */}
                    <div className="relative w-full aspect-[4/3] overflow-hidden bg-[#111]">
                      <div className="absolute inset-0 bg-black/20 z-10 transition-opacity duration-700 group-hover:opacity-0" />
                      <Image 
                        src={imgSrc} 
                        alt={content.title}
                        fill
                        className="object-cover transition-transform duration-[1.5s] ease-[cubic-bezier(0.32,0.72,0,1)] group-hover:scale-105"
                      />
                    </div>

                    {/* Content Area */}
                    <div className="flex-1 p-8 lg:p-10 flex flex-col justify-between">
                      <div>
                        <span className="font-mono text-[10px] uppercase tracking-widest text-white/30 block mb-4">
                          {categoryLabel}
                        </span>
                        <h3 className="text-2xl font-medium text-white mb-4">
                          {content.title}
                        </h3>
                        <p className="text-sm font-light text-white/50 leading-relaxed">
                          {content.summary}
                        </p>
                      </div>
                      <div className="mt-8 flex items-center gap-4">
                        <div className="h-10 w-10 rounded-full border border-white/10 flex items-center justify-center transition-colors duration-700 group-hover:bg-white group-hover:border-white">
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-white group-hover:text-black transition-colors duration-700">
                            <path d="M5 12h14M12 5l7 7-7 7" strokeLinecap="round" strokeLinejoin="round"/>
                          </svg>
                        </div>
                        <span className="text-xs font-mono uppercase tracking-widest text-white/70 group-hover:text-white transition-colors duration-700">
                          {t('btn_read_more', 'Read more')}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </Link>
            </motion.div>
          );
        })}
      </motion.div>
    </motion.section>
  );
}

export function O9TourCTA({
  relatedProjectSlug,
}: {
  relatedProjectSlug?: string;
}) {
  const { t } = useLanguage();
  return (
    <motion.section 
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-10%" }}
      className="py-24 lg:py-40 px-4 sm:px-6 lg:px-8 w-full max-w-[1560px] mx-auto border-t border-white/10"
    >
      <div className="p-[6px] rounded-[2rem] bg-white/[0.03] border border-white/10 relative overflow-hidden group">
        <div className="relative h-full w-full rounded-[calc(2rem-6px)] bg-[#050505] p-12 lg:p-24 text-center overflow-hidden flex flex-col items-center justify-center">
          <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50 z-10" />
          
          <O9SectionLabel>{t('licensing_tour_tag')}</O9SectionLabel>
          <motion.h2 variants={FADE_UP} className="mt-8 max-w-3xl text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white leading-[1.05]">
            {t('licensing_demo_title')}
          </motion.h2>
          <motion.p variants={FADE_UP} className="mt-8 max-w-2xl text-lg sm:text-xl font-light text-white/50 leading-relaxed">
            {t('licensing_demo_desc')}
          </motion.p>
          
          <motion.div variants={FADE_UP} className="mt-12 flex flex-col sm:flex-row items-center gap-6">
            <Link 
              href="/join" 
              className="group/btn relative inline-flex items-center justify-center rounded-full bg-white px-8 py-4 text-sm font-medium text-black transition-all hover:scale-[0.98] hover:bg-white/90"
            >
              <span className="relative z-10">{t('nav_demo')}</span>
            </Link>
            
            {relatedProjectSlug ? (
              <Link 
                href={`/projects/${relatedProjectSlug}`} 
                className="group/btn relative inline-flex items-center justify-center rounded-full bg-white/[0.05] border border-white/10 px-8 py-4 text-sm font-medium text-white transition-all hover:bg-white/[0.1] hover:scale-[0.98]"
              >
                <span className="relative z-10">{t('nav_modules')}</span>
              </Link>
            ) : (
              <Link 
                href="/platform" 
                className="group/btn relative inline-flex items-center justify-center rounded-full bg-white/[0.05] border border-white/10 px-8 py-4 text-sm font-medium text-white transition-all hover:bg-white/[0.1] hover:scale-[0.98]"
              >
                <span className="relative z-10">{t('nav_tour')}</span>
              </Link>
            )}
          </motion.div>
        </div>
      </div>
    </motion.section>
  );
}
