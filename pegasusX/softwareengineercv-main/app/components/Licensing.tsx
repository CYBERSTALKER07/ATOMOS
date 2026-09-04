'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import GlitchText from './GlitchText';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';

gsap.registerPlugin(ScrollTrigger);

export default function Licensing() {
  const { isMobile } = useIsMobile();
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
      ctaLabel: t('nav_tour').toUpperCase(),
    },
    {
      tone: 'light' as const,
      tag: t('licensing_demo_tag'),
      title: t('licensing_demo_title'),
      description: t('licensing_demo_desc'),
      image: EDITORIAL_IMAGES[1],
      href: '/join',
      ctaLabel: t('nav_demo').toUpperCase(),
    },
  ];

  useEffect(() => {
    if (sectionRef.current && titleRef.current && contentRef.current && cardsRef.current) {
      if (isMobile) {
        gsap.set([titleRef.current, contentRef.current, cardsRef.current], {
          opacity: 1,
          y: 0,
          scale: 1,
        });
        return;
      }

      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse',
        },
      });

      timeline
        .fromTo(titleRef.current, { opacity: 0, scale: 0.9 }, { opacity: 1, scale: 1, duration: 1.2 })
        .fromTo(contentRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 }, '-=0.4')
        .fromTo(
          cardsRef.current.children,
          { opacity: 0, y: 30 },
          { opacity: 1, y: 0, duration: 0.6, stagger: 0.15 },
          '-=0.3'
        );
    }
  }, [isMobile]);

  return (
    <section
      ref={sectionRef}
      id="deployment"
      className="min-h-screen py-20 bg-black text-white relative overflow-hidden flex items-center"
    >
      <div className="container mx-auto px-4 relative z-10">
        <div className="max-w-6xl mx-auto">
          <div ref={titleRef} className="text-center mb-12 md:mb-16">
            {isMobile ? (
              <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-6 text-white">{t('deployment_heading')}</h2>
            ) : (
              <GlitchText speed={1} enableShadows={true} enableOnHover={true} className="mb-6">
                {t('deployment_heading')}
              </GlitchText>
            )}
            <div className="w-20 h-1 bg-white rounded-full mx-auto" />
          </div>

          <div ref={contentRef} className="text-center mb-12 md:mb-16 max-w-3xl mx-auto">
            <p className="text-lg md:text-xl text-gray-300 leading-relaxed">
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
                className="deployment-card min-h-[28rem]"
                hoverLabel={card.href === '/platform' ? t('licensing_hover_view') : t('licensing_hover_demo')}
              />
            ))}
          </div>
        </div>
      </div>

      <div className="absolute top-10 left-10 w-32 h-32 border-2 border-white opacity-10 rounded-2xl" />
      <div className="absolute bottom-10 right-10 w-40 h-40 border-2 border-white opacity-10 rounded-2xl" />
    </section>
  );
}
