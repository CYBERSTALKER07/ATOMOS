'use client';

import { useEffect, useRef } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import { usePerfProfile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';
import { SITE_IMAGES } from '@/app/lib/siteAssets';
import PageSection from './layout/PageSection';

export default function LastMileSection() {
  const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
  const { t } = useLanguage();
  const sectionRef = useRef<HTMLElement>(null);
  const imageRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sectionRef.current) return;

    const ctx = gsap.context(() => {
      if (isMobile || isLowEnd || prefersReducedMotion) {
        gsap.set([imageRef.current, contentRef.current], { opacity: 1, y: 0 });
        return;
      }

      gsap
        .timeline({
          scrollTrigger: {
            trigger: sectionRef.current,
            start: 'top 78%',
            once: true,
          },
        })
        .fromTo(imageRef.current, { opacity: 0, y: 28 }, { opacity: 1, y: 0, duration: 0.9, ease: 'pegasus' })
        .fromTo(
          contentRef.current,
          { opacity: 0, y: 28 },
          { opacity: 1, y: 0, duration: 0.9, ease: 'pegasus' },
          '-=0.55',
        );
    }, sectionRef);

    return () => ctx.revert();
  }, [isMobile, isLowEnd, prefersReducedMotion]);

  return (
    <PageSection id="last-mile" ref={sectionRef} className="border-t border-white/10 relative">
      
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-center">
        <div ref={imageRef} className="relative order-1 lg:col-span-7">
          <div className="relative aspect-[3/2] w-full overflow-hidden bg-black border border-white/10">
            <Image
              src={SITE_IMAGES.lastMileDelivery}
              alt={t(
                'last_mile_image_alt',
                'Driver handing Pegasus packages to a retailer at the door',
              )}
              fill
              className="object-cover object-center"
              sizes="(max-width: 1024px) 100vw, 60vw"
              priority={false}
            />
          </div>
        </div>

        <div ref={contentRef} className="order-2 space-y-6 max-w-xl lg:col-span-5">
          <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">
            {t('last_mile_eyebrow', 'Delivery')}
          </p>
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-light tracking-tight text-white">
            {t('last_mile_title', 'Track every single order')}
          </h2>
          <div className="w-20 h-px bg-white" />
          <p className="text-base md:text-lg font-extralight text-white/70 leading-relaxed">
            {t(
              'last_mile_desc',
              'Built for retailers and suppliers — on every OS they use. Live status from warehouse to door so both sides see the same stop, without phone calls or guesswork.',
            )}
          </p>
          <div className="flex flex-col sm:flex-row gap-3 pt-2">
            <Link
              href="/capabilities/live-fleet-tracking"
              className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] bg-white text-black hover:bg-white/90 transition-colors"
            >
              {t('last_mile_cta_primary', 'See live tracking')}
            </Link>
            <Link
              href="/roles/retailer"
              className="inline-flex items-center justify-center min-h-11 px-5 text-xs font-semibold uppercase tracking-[0.1em] border border-white/30 text-white hover:border-white/60 transition-colors"
            >
              {t('last_mile_cta_secondary', 'Retailer experience')}
            </Link>
          </div>
        </div>
      </div>
    </PageSection>
  );
}
