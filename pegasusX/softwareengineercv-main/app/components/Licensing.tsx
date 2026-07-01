'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import GlitchText from './GlitchText';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

const DEPLOYMENT_CARDS = [
  {
    tone: 'dark' as const,
    tag: 'DISCOVER THE PLATFORM',
    title: 'Take a Tour',
    description:
      'See how Pegasus unifies dispatch, fleet tracking, payments, and coordination across every role in your network.',
    image: EDITORIAL_IMAGES[4],
    href: '/#solutions',
    ctaLabel: 'TAKE PLATFORM TOUR',
  },
  {
    tone: 'light' as const,
    tag: 'DISCOVER OUR PLATFORM',
    title: 'Live Demo with a Pegasus Expert',
    description:
      'Get a personalized walkthrough and see how to run supplier-led logistics with faster, smarter decisions across your enterprise.',
    image: EDITORIAL_IMAGES[1],
    href: '/join',
    ctaLabel: 'REQUEST DEMO',
  },
];

export default function Licensing() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const cardsRef = useRef<HTMLDivElement>(null);

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
              <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-6 text-white">DEPLOYMENT</h2>
            ) : (
              <GlitchText speed={1} enableShadows={true} enableOnHover={true} className="mb-6">
                DEPLOYMENT
              </GlitchText>
            )}
            <div className="w-20 h-1 bg-white rounded-full mx-auto" />
          </div>

          <div ref={contentRef} className="text-center mb-12 md:mb-16 max-w-3xl mx-auto">
            <p className="text-lg md:text-xl text-gray-300 leading-relaxed">
              Deploy Pegasus across your network with guided onboarding, live demos, and expert
              walkthroughs tailored to how your teams run dispatch and delivery today.
            </p>
          </div>

          <div ref={cardsRef} className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6 max-w-6xl mx-auto">
            {DEPLOYMENT_CARDS.map((card) => (
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
                hoverLabel={card.ctaLabel.includes('TOUR') ? 'VIEW' : 'DEMO'}
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
