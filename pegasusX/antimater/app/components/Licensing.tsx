'use client';

import { useEffect, useRef } from 'react';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import GlitchText from './GlitchText';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { usePerfProfile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';

export default function Licensing() {
 const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
 const { t } = useLanguage();
 const sectionRef = useRef<HTMLDivElement>(null);
 const titleRef = useRef<HTMLDivElement>(null);
 const contentRef = useRef<HTMLDivElement>(null);
 const cardsRef = useRef<HTMLDivElement>(null);

 const deploymentCards = [
 {
 tone: 'dark' as const,
 tag: t('licensing_tour_tag'),
 title: t('licensing_tour_title'),
 description: t('licensing_tour_desc'),
 image: EDITORIAL_IMAGES[4],
 href: '/platform',
 ctaLabel: (t('nav_tour') || 'Take Platform Tour').toUpperCase(),
 },
 {
 tone: 'dark' as const,
 tag: t('licensing_demo_tag'),
 title: t('licensing_demo_title'),
 description: t('licensing_demo_desc'),
 image: EDITORIAL_IMAGES[1],
 href: '/join',
 ctaLabel: (t('nav_demo') || 'Request Demo').toUpperCase(),
 },
 ];

  useEffect(() => {
    if (!sectionRef.current || !titleRef.current || !contentRef.current || !cardsRef.current) return;

    if (isMobile || isLowEnd || prefersReducedMotion) {
      gsap.set([titleRef.current, contentRef.current, cardsRef.current], {
        opacity: 1,
        y: 0,
        scale: 1,
      });
      return;
    }

    const ctx = gsap.context(() => {
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top 95%',
          once: true,
          fastScrollEnd: true,
        },
      });

      timeline
        .fromTo(titleRef.current, { opacity: 0, y: 16 }, { opacity: 1, y: 0, duration: 0.45, ease: 'power2.out' })
        .fromTo(contentRef.current, { opacity: 0, y: 16 }, { opacity: 1, y: 0, duration: 0.45, ease: 'power2.out' }, '-=0.35')
        .fromTo(
          cardsRef.current?.children ? Array.from(cardsRef.current.children) : [],
          { opacity: 0, y: 20 },
          { opacity: 1, y: 0, duration: 0.45, stagger: 0.08, ease: 'power2.out' },
          '-=0.35'
        );
    }, sectionRef);

    ScrollTrigger.refresh();

    return () => ctx.revert();
  }, [isMobile, isLowEnd, prefersReducedMotion]);

 return (
 <section
 ref={sectionRef}
 id="deployment"
 className="py-16 md:py-24 bg-[#F8FAFC] dark:bg-black text-zinc-900 dark:text-white relative overflow-hidden flex items-center transition-colors"
 >
 <div className="container mx-auto px-4 relative z-10">
 <div className="max-w-6xl mx-auto">
 <div ref={titleRef} className="text-center mb-12 md:mb-16">
 {isMobile ? (
 <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-6 text-zinc-900 dark:text-white">{t('deployment_heading')}</h2>
 ) : (
 <GlitchText speed={1} enableShadows={true} enableOnHover={true} className="mb-6">
 {t('deployment_heading')}
 </GlitchText>
 )}
 <div className="w-20 h-1 bg-zinc-900 dark:bg-white rounded-none mx-auto" />
 </div>

 <div ref={contentRef} className="text-center mb-12 md:mb-16 max-w-3xl mx-auto">
 <p className="text-lg md:text-xl text-zinc-600 dark:text-gray-300 leading-relaxed">
 {t('deployment_desc')}
 </p>
 </div>

 <div ref={cardsRef} className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6 max-w-6xl mx-auto">
 {deploymentCards.map((card) => (
 <ContentCard
 key={card.title}
 variant="vertical"
 tone={card.tone}
 tag={card.tag}
 title={card.title}
 description={card.description}
 image={card.image}
 href={card.href}
 ctaLabel={card.ctaLabel}
 ctaStyle="button"
 splitCta
                imagePriority={true}
 className="deployment-card min-h-[28rem]"
 hoverLabel={card.href === '/platform' ? t('licensing_hover_view') : t('licensing_hover_demo')}
 />
 ))}
 </div>
 </div>
 </div>

 </section>
 );
}
