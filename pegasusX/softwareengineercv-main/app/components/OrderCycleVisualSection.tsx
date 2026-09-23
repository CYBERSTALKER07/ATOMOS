'use client';

import { useEffect, useRef } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { gsap } from '@/app/lib/gsap';
import { useLanguage } from '@/app/context/LanguageContext';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

export default function OrderCycleVisualSection() {
  const { language } = useLanguage();
  const isRu = language === 'ru';
  const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();

  const sectionRef = useRef<HTMLElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sectionRef.current || !contentRef.current) return;

    if (isMobile || isLowEnd || prefersReducedMotion) {
      gsap.set(contentRef.current, { opacity: 1, y: 0 });
      return;
    }

    const ctx = gsap.context(() => {
      gsap.fromTo(
        contentRef.current,
        { opacity: 0, y: 28 },
        {
          opacity: 1,
          y: 0,
          duration: 0.9,
          ease: 'pegasus',
          scrollTrigger: {
            trigger: sectionRef.current,
            start: 'top 75%',
            once: true,
          },
        }
      );
    }, sectionRef);

    return () => ctx.revert();
  }, [isMobile, isLowEnd, prefersReducedMotion]);

  return (
    <section
      ref={sectionRef}
      className="relative w-full min-h-[580px] sm:min-h-[660px] lg:min-h-[780px] xl:min-h-[860px] flex items-center overflow-hidden bg-black select-none"
      aria-label={isRu ? 'Логистическая сеть Pegasus' : 'Pegasus Logistics Network'}
    >
      {/* Background Image — 4K Geometric Logistics Architecture (preserves user image) */}
      <div className="absolute inset-0 z-0 bg-black">
        <Image
          src={SITE_IMAGES.geometricTerminal}
          alt={
            isRu
              ? 'Автономный логистический терминал и геометрический хаб распределения Pegasus'
              : 'Pegasus autonomous logistics terminal and geometric distribution hub'
          }
          fill
          priority
          sizes="100vw"
          className="object-cover object-[28%_center] lg:object-[24%_center]"
        />

        {/* Atmospheric overlays matching screenshot reference */}
        {/* Right side gradient for optimal text readability while keeping architecture bright on left */}
        <div className="absolute inset-0 bg-gradient-to-r from-black/40 via-black/50 to-black/90 lg:from-transparent lg:via-black/35 lg:to-black/85 pointer-events-none" />

        {/* Top edge soft blend into About section */}
        <div className="absolute inset-x-0 top-0 h-28 sm:h-40 bg-gradient-to-b from-black via-black/60 to-transparent pointer-events-none" />

        {/* Bottom edge soft blend into LastMileSection */}
        <div className="absolute inset-x-0 bottom-0 h-28 sm:h-40 bg-gradient-to-t from-black via-black/60 to-transparent pointer-events-none" />
      </div>

      {/* Content Container positioned right as in reference screenshot */}
      <div className="relative z-20 w-full max-w-[1500px] mx-auto px-6 sm:px-10 lg:px-16 py-20 lg:py-28">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
          <div
            ref={contentRef}
            className="lg:col-span-6 lg:col-start-7 xl:col-span-6 xl:col-start-7 flex flex-col justify-center text-left"
          >
            {/* Main Headline */}
            <h2 className="text-3xl sm:text-4xl md:text-5xl lg:text-[3.5rem] xl:text-[4rem] font-light text-white leading-[1.08] tracking-tight">
              {isRu ? (
                <>
                  Логистическая сеть для <br className="hidden sm:inline" />
                  автономного масштаба
                </>
              ) : (
                <>
                  A logistics network built <br className="hidden sm:inline" />
                  for autonomous scale
                </>
              )}
            </h2>

            {/* Subtitle / Description */}
            <p className="mt-4 sm:mt-6 text-base sm:text-lg lg:text-xl text-white/70 font-light leading-relaxed max-w-lg">
              {isRu
                ? 'Объединение визуальной диспетчеризации, телеметрии автопарка и координации шести ролей в единой системе.'
                : 'Combining visual dispatch, fleet telemetry, and multi-role coordination all under one roof.'}
            </p>

            {/* CTA Button matching reference */}
            <div className="mt-7 sm:mt-9">
              <Link
                href="/platform"
                className="inline-flex items-center justify-center gap-2.5 px-6 sm:px-7 py-3 sm:py-3.5 bg-white text-black text-sm sm:text-base font-medium tracking-wide rounded-none hover:bg-zinc-200 transition-colors"
              >
                <span>{isRu ? 'Подробнее' : 'Learn More'}</span>
                <span className="text-base leading-none">›</span>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
