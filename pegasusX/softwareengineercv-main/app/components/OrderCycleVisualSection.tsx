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
      className="relative w-full aspect-[16/9] min-h-[540px] sm:min-h-[620px] lg:min-h-[720px] max-h-[92vh] flex items-center overflow-hidden bg-black select-none"
      aria-label={isRu ? 'Логистическая сеть Pegasus' : 'Pegasus Logistics Network'}
    >
      {/* Background Image — 4K Geometric Logistics Architecture (Full width edge-to-edge, centered) */}
      <div className="absolute inset-0 z-0 w-full h-full bg-black">
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
          className="object-cover object-center w-full h-full"
        />

        {/* Transparent ambient overlay preserving the entire full-width image from edge to edge */}
        <div className="absolute inset-0 bg-black/20 pointer-events-none" />

        {/* Soft top and bottom edge transitions */}
        <div className="absolute inset-x-0 top-0 h-24 sm:h-32 bg-gradient-to-b from-black/80 via-black/40 to-transparent pointer-events-none" />
        <div className="absolute inset-x-0 bottom-0 h-24 sm:h-32 bg-gradient-to-t from-black/80 via-black/40 to-transparent pointer-events-none" />
      </div>

      {/* Content Container positioned right matching reference screenshot */}
      <div className="relative z-20 w-full max-w-[1500px] mx-auto px-6 sm:px-10 lg:px-16 py-16 sm:py-20 lg:py-24">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
          <div
            ref={contentRef}
            className="lg:col-span-6 lg:col-start-7 xl:col-span-6 xl:col-start-7 flex flex-col justify-center text-left"
          >
            {/* Main Headline with drop shadow for crisp clarity over the full-width image */}
            <h2 className="text-3xl sm:text-4xl md:text-5xl lg:text-[3.5rem] xl:text-[4rem] font-light text-white leading-[1.08] tracking-tight drop-shadow-[0_2px_16px_rgba(0,0,0,0.95)]">
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
            <p className="mt-4 sm:mt-6 text-base sm:text-lg lg:text-xl text-white/80 font-light leading-relaxed max-w-lg drop-shadow-[0_2px_12px_rgba(0,0,0,0.95)]">
              {isRu
                ? 'Объединение визуальной диспетчеризации, телеметрии автопарка и координации шести ролей в единой системе.'
                : 'Combining visual dispatch, fleet telemetry, and multi-role coordination all under one roof.'}
            </p>

            {/* CTA Button matching reference */}
            <div className="mt-7 sm:mt-9">
              <Link
                href="/platform"
                className="inline-flex items-center justify-center gap-2.5 px-6 sm:px-7 py-3 sm:py-3.5 bg-white text-black text-sm sm:text-base font-medium tracking-wide rounded-none hover:bg-zinc-200 transition-colors drop-shadow-md"
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
