'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import PageSection from '../layout/PageSection';
import { usePerfProfile } from '../../hooks/useDevice';
import { useInView } from '../../hooks/useInView';
import { PEGASUS_ASK_PROMPTS } from '@/app/data/askPromptCards';
import type { AskPromptSectionContent } from './types';
import AskPromptTitle from './AskPromptTitle';
import AskPromptMetricsFlow from './AskPromptMetricsFlow';
import { cn } from '@/lib/utils';

gsap.registerPlugin(ScrollTrigger);

export type AskPromptSectionProps = {
  content?: AskPromptSectionContent;
  id?: string;
  className?: string;
};

export default function AskPromptSection({
  content = PEGASUS_ASK_PROMPTS,
  id = 'ask-prompt',
  className,
}: AskPromptSectionProps) {
  const { isMobile, isTablet, prefersReducedMotion, isLowEnd } = usePerfProfile();
  const { ref: sectionRef, isInView } = useInView<HTMLElement>({ rootMargin: '80px' });
  const headerRef = useRef<HTMLDivElement>(null);
  const flowRef = useRef<HTMLDivElement>(null);
  const [chartsAnimated, setChartsAnimated] = useState(false);
  const compact = isMobile || isTablet;

  const metric = content.metric ?? PEGASUS_ASK_PROMPTS.metric!;

  useEffect(() => {
    if (isInView && !prefersReducedMotion && !isLowEnd) setChartsAnimated(true);
  }, [isInView, prefersReducedMotion, isLowEnd]);

  useEffect(() => {
    const section = sectionRef.current;
    if (!section || !headerRef.current || !flowRef.current) return;

    if (prefersReducedMotion || isMobile || isLowEnd) return;

    const panels = flowRef.current.querySelectorAll('.ask-metrics-card');

    const ctx = gsap.context(() => {
      gsap.from(headerRef.current, {
        opacity: 0,
        y: 22,
        duration: 0.6,
        ease: 'power3.out',
        scrollTrigger: { trigger: section, start: 'top 85%', once: true },
      });
      gsap.from(panels, {
        opacity: 0,
        y: 28,
        duration: 0.55,
        stagger: 0.12,
        ease: 'power2.out',
        scrollTrigger: { trigger: section, start: 'top 80%', once: true },
        delay: 0.15,
      });
    }, section);

    return () => ctx.revert();
  }, [prefersReducedMotion, isMobile, isLowEnd, sectionRef]);

  return (
    <PageSection
      ref={sectionRef}
      id={id}
      className={cn('overflow-hidden border-t border-white/5 !py-10 sm:!py-14 md:!py-20', className)}
    >
      <div className="max-w-6xl mx-auto px-4 sm:px-6 md:px-10 lg:px-[70px]">
        <div ref={headerRef} className="text-center mb-10 sm:mb-12 md:mb-14">
          <AskPromptTitle title={content.title} compact={compact} />
          <p className="mt-4 sm:mt-5 text-sm sm:text-base text-white/50 max-w-xl mx-auto leading-relaxed font-light px-1">
            {content.subtitle}
          </p>
        </div>

        <div ref={flowRef}>
          <AskPromptMetricsFlow metric={metric} animate={chartsAnimated} />
        </div>
      </div>
    </PageSection>
  );
}
