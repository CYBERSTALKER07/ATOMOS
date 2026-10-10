'use client';

import { motion } from 'framer-motion';
import Link from 'next/link';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { O9SectionLabel } from './O9Sections';
import { useLanguage } from '@/app/context/LanguageContext';
import Image from 'next/image';

const FADE_UP = {
  hidden: { opacity: 0, y: 30 },
  visible: { 
    opacity: 1, 
    y: 0, 
    transition: { duration: 1, ease: [0.16, 1, 0.3, 1] as const } 
  }
};

const STAGGER_CONTAINER = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 }
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
      <motion.h2 variants={FADE_UP} className="text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white max-w-5xl leading-[1.05] mb-24">
        {t('sec_related_use_cases', 'Related use cases on the Pegasus platform')}
      </motion.h2>
      
      <motion.div variants={STAGGER_CONTAINER} className="flex flex-col">
        {siblings.slice(0, 6).map((s, i) => {
          const content = s.content[language] || s.content.en;
          const imgSrc = imageFor(flow, i);
          return (
            <motion.div key={s.slug} variants={FADE_UP} className="grid grid-cols-1 md:grid-cols-12 gap-8 md:gap-12 py-12 border-t border-white/10 group">
              <div className="md:col-span-4 relative h-[240px] md:h-full min-h-[200px] w-full overflow-hidden bg-[#050505]">
                <Image 
                  src={imgSrc} 
                  alt={content.title}
                  fill
                  className="object-cover transition-transform duration-[1.5s] ease-[cubic-bezier(0.16,1,0.3,1)] group-hover:scale-105"
                />
              </div>

              <div className="md:col-span-8 flex flex-col justify-center">
                <span className="font-mono text-[10px] uppercase tracking-widest text-white/30 block mb-4">
                  {categoryLabel}
                </span>
                <Link href={s.href} className="inline-block">
                  <h3 className="text-3xl lg:text-4xl font-medium text-white mb-6 group-hover:text-white/80 transition-colors">
                    {content.title}
                  </h3>
                </Link>
                <p className="text-xl font-light text-white/60 leading-relaxed max-w-3xl mb-8">
                  {content.summary}
                </p>
                <Link href={s.href} className="inline-flex items-center gap-4 text-white hover:text-white/70 transition-colors">
                  <span className="text-xs font-mono uppercase tracking-widest">
                    {t('btn_read_more', 'Read more')}
                  </span>
                  <span className="text-lg leading-none mt-[-2px]">›</span>
                </Link>
              </div>
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
      <div className="w-full bg-[#050505] py-24 lg:py-32 flex flex-col items-center justify-center text-center border-y border-white/10">
        <O9SectionLabel>{t('licensing_tour_tag', 'Take a tour')}</O9SectionLabel>
        <motion.h2 variants={FADE_UP} className="mt-8 max-w-3xl text-4xl sm:text-5xl lg:text-7xl font-medium tracking-tight text-white leading-[1.05]">
          {t('licensing_demo_title', 'Experience the platform')}
        </motion.h2>
        <motion.p variants={FADE_UP} className="mt-8 max-w-2xl text-lg sm:text-xl font-light text-white/50 leading-relaxed">
          {t('licensing_demo_desc', 'Schedule a demo to see how Pegasus can transform your supply chain.')}
        </motion.p>
        
        <motion.div variants={FADE_UP} className="mt-12 flex flex-col sm:flex-row items-center gap-4">
          <Link 
            href="/join" 
            className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white text-black hover:bg-white/90 rounded-none"
          >
            <span>{t('nav_demo', 'Request Demo')}</span>
            <span className="text-lg leading-none mt-[-2px]">›</span>
          </Link>
          
          {relatedProjectSlug ? (
            <Link 
              href={`/projects/${relatedProjectSlug}`} 
              className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white/5 text-white hover:bg-white/10 border border-white/10 rounded-none"
            >
              <span>{t('nav_modules', 'View Modules')}</span>
              <span className="text-lg leading-none mt-[-2px]">›</span>
            </Link>
          ) : (
            <Link 
              href="/platform" 
              className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white/5 text-white hover:bg-white/10 border border-white/10 rounded-none"
            >
              <span>{t('nav_tour', 'Platform Tour')}</span>
              <span className="text-lg leading-none mt-[-2px]">›</span>
            </Link>
          )}
        </motion.div>
      </div>
    </motion.section>
  );
}
