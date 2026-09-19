'use client';

import { useEffect, useRef } from 'react';
import { gsap, smoothScrollTo } from '@/app/lib/gsap';
import { ArrowRight } from '@/components/icons';
import ParticleText from './ParticleText';
import CurvedLoop from './CurvedLoop';
import TextType from './TextType';
import IsometricTerrain from './IsometricTerrain';
import { usePerfProfile } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';
import { useTheme } from '../context/ThemeContext';

export default function Hero() {
  const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
  const { t, language } = useLanguage();
  const { resolvedTheme } = useTheme();
  const isLight = resolvedTheme === 'light';

  const textRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const subtitleRef = useRef<HTMLHeadingElement>(null);
  const descRef = useRef<HTMLParagraphElement>(null);
  const ctaRef = useRef<HTMLDivElement>(null);
  const visualRef = useRef<HTMLDivElement>(null);

  const typedPhrases = [
    t('hero_type_1'),
    t('hero_type_2'),
    t('hero_type_3'),
    t('hero_type_4'),
  ];



  useEffect(() => {
    const ctx = gsap.context(() => {
      if (isMobile || isLowEnd || prefersReducedMotion) {
        gsap.set(
          [titleRef.current, subtitleRef.current, descRef.current, ctaRef.current, visualRef.current],
          { opacity: 1, x: 0, y: 0 }
        );
        return;
      }

      const timeline = gsap.timeline({ defaults: { ease: 'pegasus' } });

      timeline
        .fromTo(
          visualRef.current,
          { opacity: 0, scale: 0.96 },
          { opacity: 1, scale: 1, duration: 1.1 }
        )
        .fromTo(
          titleRef.current,
          { opacity: 0, y: 40 },
          { opacity: 1, y: 0, duration: 0.9 },
          '-=0.7'
        )
        .fromTo(
          subtitleRef.current,
          { opacity: 0, y: 24 },
          { opacity: 1, y: 0, duration: 0.7 },
          '-=0.5'
        )
        .fromTo(
          descRef.current,
          { opacity: 0, y: 16 },
          { opacity: 1, y: 0, duration: 0.7 },
          '-=0.5'
        )
        .fromTo(
          ctaRef.current,
          { opacity: 0, y: 16 },
          { opacity: 1, y: 0, duration: 0.7 },
          '-=0.4'
        );
    });

    return () => ctx.revert();
  }, [isMobile, isLowEnd, prefersReducedMotion]);

  const scrollToNext = () => {
    smoothScrollTo('#about', { offsetY: 64, duration: 1.1, ease: 'pegasus' });
  };

  return (
    <section
      id="hero"
      className="min-h-screen relative flex flex-col justify-center bg-[#000000] overflow-hidden pt-20 sm:pt-24 pb-14 sm:pb-16"
    >
      {/* Decorative Curved Loops for desktop / Mac */}
      {!isMobile && (
        <>
          <div className="absolute top-0 left-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-0">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={900}
              direction="right"
              interactive={false}
              className={isLight ? 'fill-black' : 'fill-white'}
            />
          </div>

          <div className="absolute top-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-0 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={500}
              direction="left"
              interactive={false}
              className={isLight ? 'fill-black' : 'fill-white'}
            />
          </div>

          <div className="absolute bottom-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-0 rotate-180 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={200}
              direction="right"
              interactive={false}
              className={isLight ? 'fill-black' : 'fill-white'}
            />
          </div>
        </>
      )}

      <div className="w-full max-w-[1560px] mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Tactical Framed Container */}
        <div className={`border shadow-2xl relative grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x transition-colors duration-200 ${
          isLight
            ? 'border-black/10 bg-white divide-black/10 shadow-[0_8px_30px_rgba(0,0,0,0.06)]'
            : 'border-white/15 bg-[#000000] divide-white/15 shadow-2xl'
        }`}>
          {/* LEFT COLUMN: Editorial Headline, Subtitle, Description & Outlined CTA */}
          <div
            ref={textRef}
            className="flex flex-col justify-end p-6 sm:p-8 lg:p-10 xl:p-12 relative z-10 min-h-[540px] lg:min-h-[640px] xl:min-h-[700px]"
          >
            <div className="space-y-4">
              {/* Primary Headline with Interactive ParticleText */}
              <div className="space-y-2">
                <div ref={titleRef} className="w-full h-20 sm:h-24 md:h-28 xl:h-32">
                  <span className="sr-only">{t('hero_title')}</span>
                  <ParticleText
                    text="Pegasus"
                    particleSize={2.4}
                    density={4}
                    color={isLight ? '#09090b' : '#f8fafc'}
                    highlightColor="#10B981"
                    scatter={160}
                    gatherDuration={1500}
                    stagger={350}
                    pointerRepel={42}
                    repelRadius={120}
                    idleDrift={0.6}
                    trigger="mount"
                    fontSize="clamp(3.2rem, 6.2vw, 5.8rem)"
                    fontWeight={800}
                    textAlign="left"
                    glow={!isLight}
                  />
                </div>

                <h1
                  ref={subtitleRef}
                  className={`block text-2xl sm:text-3xl md:text-4xl lg:text-5xl font-light min-h-[1.25em] tracking-tight leading-[1.12] ${
                    isLight ? 'text-zinc-900' : 'text-white'
                  }`}
                >
                  <TextType
                    key={`${language}-${isLight ? 'light' : 'dark'}`}
                    text={typedPhrases}
                    typingSpeed={isMobile ? 90 : 70}
                    pauseDuration={1600}
                    deletingSpeed={isMobile ? 60 : 45}
                    showCursor={true}
                    cursorCharacter="|"
                    loop={true}
                    textColors={isLight ? ['#09090B', '#475569'] : ['#FFFFFF', '#C0C0C0']}
                    className="font-light"
                    cursorClassName={isLight ? 'text-zinc-900 font-light' : 'text-white font-light'}
                  />
                </h1>
              </div>

              {/* Subtitle Description */}
              <p
                ref={descRef}
                className={`text-sm sm:text-base md:text-lg font-light leading-relaxed max-w-lg pt-1 ${
                  isLight ? 'text-zinc-600' : 'text-white/60'
                }`}
              >
                {t('hero_desc')}
              </p>
            </div>

            {/* Outlined Action Buttons */}
            <div ref={ctaRef} className="pt-6 sm:pt-8 flex flex-wrap items-center gap-4">
              <button
                onClick={scrollToNext}
                className={`inline-flex items-center justify-center gap-3 px-8 py-3.5 border transition-all text-xs sm:text-sm font-semibold tracking-widest uppercase group ${
                  isLight
                    ? 'border-black text-black hover:bg-black hover:text-white'
                    : 'border-white/30 hover:border-white hover:bg-white hover:text-black text-white'
                }`}
              >
                <span>{t('hero_explore')}</span>
                <ArrowRight size={16} />
              </button>

              <a
                href="/join"
                className={`inline-flex items-center justify-center gap-2 px-7 py-3.5 border transition-all text-xs sm:text-sm font-semibold tracking-widest uppercase ${
                  isLight
                    ? 'border-black/20 bg-transparent hover:bg-black/5 hover:border-black/40 text-zinc-800 hover:text-zinc-950'
                    : 'border-white/20 bg-transparent hover:bg-white/10 hover:border-white/40 text-white/80 hover:text-white'
                }`}
              >
                <span>{t('hero_demo')}</span>
              </a>
            </div>
          </div>

          {/* RIGHT COLUMN: 3D Isometric Wireframe Graphic & Bottom Metrics Bar */}
          <div
            ref={visualRef}
            className={`flex flex-col justify-between relative overflow-hidden min-h-[540px] lg:min-h-[640px] xl:min-h-[700px] ${
              isLight ? 'bg-zinc-50/50' : 'bg-black'
            }`}
          >
            {/* Upper Area: Pure 3D Isometric Wireframe Visual positioned high */}
            <div className="flex-1 min-h-[340px] sm:min-h-[420px] lg:min-h-[460px] relative flex items-center justify-center p-4 sm:p-6 overflow-hidden">
              <IsometricTerrain />
            </div>


          </div>
        </div>
      </div>

    </section>
  );
}

