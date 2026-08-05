'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ContentCardEyebrow } from '@/app/components/ContentCard';
import { cn } from '@/lib/utils';

type HeroEditorialProps = {
  backHref?: string;
  backLabel?: string;
  eyebrow: string;
  title: string;
  summary: string;
  badge?: string;
  className?: string;
};

export default function HeroEditorial({
  backHref,
  backLabel,
  eyebrow,
  title,
  summary,
  badge,
  className,
}: HeroEditorialProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return;
    gsap.fromTo(ref.current, { opacity: 0, y: 40 }, { opacity: 1, y: 0, duration: 0.9, ease: 'power3.out' });
  }, []);

  return (
    <div ref={ref} className={cn(className)}>
      {backHref ? (
        <Link href={backHref} className="editorial-btn editorial-btn--sm mb-8">
          ← {backLabel}
        </Link>
      ) : null}
      <ContentCardEyebrow>{eyebrow}</ContentCardEyebrow>
      <h1 className="mt-4 max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">
        {title}
        {badge ? (
          <span className="ml-3 align-middle font-mono text-xs tracking-widest text-white/60">
            [{badge}]
          </span>
        ) : null}
      </h1>
      <p className="mt-6 max-w-3xl text-lg leading-relaxed text-white/70">{summary}</p>
    </div>
  );
}
