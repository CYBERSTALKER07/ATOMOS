'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ContentCardEyebrow } from '@/app/components/ContentCard';
import ChamferButton from '@/app/components/ChamferButton';
import type { MegaNavPromo } from '@/app/data/megaNavigation';

type HeroSplitProps = {
  eyebrow?: string;
  label: string;
  body?: string;
  promo?: MegaNavPromo;
  visual?: React.ReactNode;
};

export default function HeroSplit({ eyebrow = 'Explore', label, body, promo, visual }: HeroSplitProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return;
    gsap.fromTo(ref.current, { opacity: 0, y: 40 }, { opacity: 1, y: 0, duration: 0.9, ease: 'power3.out' });
  }, []);

  return (
    <div ref={ref} className="grid gap-10 lg:grid-cols-2 lg:gap-16 lg:items-center">
      <div>
        <ContentCardEyebrow>{eyebrow}</ContentCardEyebrow>
        <h1 className="mt-4 text-4xl font-semibold tracking-tight md:text-6xl">{label}</h1>
        {body ? <p className="mt-6 max-w-xl text-lg text-white/70">{body}</p> : null}
        {promo ? (
          <div className="mt-8 border border-white/20 p-6 md:p-8">
            <h2 className="text-xl font-semibold md:text-2xl">{promo.title}</h2>
            <div className="mt-5 flex flex-col gap-3 sm:flex-row">
              <ChamferButton href={promo.primaryHref} variant="fill">
                {promo.primaryLabel}
              </ChamferButton>
              {promo.secondaryLabel && promo.secondaryHref ? (
                <ChamferButton href={promo.secondaryHref} variant="ghost">
                  {promo.secondaryLabel}
                </ChamferButton>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
      {visual ? (
        <div className="relative min-h-[240px] border border-white/15 bg-[#0c0c0c] p-6 md:min-h-[320px]">
          {visual}
        </div>
      ) : null}
    </div>
  );
}
