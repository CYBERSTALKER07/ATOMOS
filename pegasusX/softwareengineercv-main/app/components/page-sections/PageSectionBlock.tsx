'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { ContentCardEyebrow } from '@/app/components/ContentCard';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { cn } from '@/lib/utils';

gsap.registerPlugin(ScrollTrigger);

type PageSectionBlockProps = {
  eyebrow: string;
  title: string;
  children: React.ReactNode;
  className?: string;
  animate?: boolean;
};

export default function PageSectionBlock({
  eyebrow,
  title,
  children,
  className = '',
  animate = true,
}: PageSectionBlockProps) {
  const ref = useRef<HTMLElement>(null);
  const { prefersReducedMotion } = usePerfProfile();

  useEffect(() => {
    if (!ref.current || !animate || prefersReducedMotion) return;

    const ctx = gsap.context(() => {
      gsap.fromTo(
        ref.current,
        { opacity: 0, y: 28 },
        {
          opacity: 1,
          y: 0,
          duration: 0.7,
          ease: 'power3.out',
          scrollTrigger: { trigger: ref.current, start: 'top 88%', once: true },
        }
      );
    }, ref);

    return () => ctx.revert();
  }, [animate, prefersReducedMotion]);

  return (
    <section ref={ref} className={cn('border-t border-white/10 py-12 md:py-16', className)}>
      <ContentCardEyebrow>{eyebrow}</ContentCardEyebrow>
      <h2 className="mt-3 text-2xl font-semibold tracking-tight md:text-3xl">{title}</h2>
      <div className="mt-6">{children}</div>
    </section>
  );
}
