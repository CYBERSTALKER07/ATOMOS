'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ArrowRight } from 'lucide-react';
import ParticleText from './ParticleText';
import CurvedLoop from './CurvedLoop';
import TextType from './TextType';
import IsometricTerrain from './IsometricTerrain';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';

export default function Hero() {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  const { t, language } = useLanguage();

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

  const isRu = language === 'ru';
  const metrics = [
    {
      label: isRu ? 'АКТИВНЫЕ ЯЧЕЙКИ' : 'ACTIVE CELLS',
      value: '16 Nodes',
    },
    {
      label: isRu ? 'ОБЪЁМ ИНТЕНТОВ' : 'DAILY INTENTS',
      value: '2.4M+',
    },
    {
      label: isRu ? 'РОЛЕВЫЕ ПОВЕРХНОСТИ' : 'CONNECTED ROLES',
      value: '6 Surfaces',
    },
    {
      label: isRu ? 'АПТАЙМ СЕТИ' : 'GLOBAL UPTIME',
      value: '99.99%',
    },
  ];

  useEffect(() => {
    const ctx = gsap.context(() => {
      if (isMobile || prefersReducedMotion) {
        gsap.set(
          [titleRef.current, subtitleRef.current, descRef.current, ctaRef.current, visualRef.current],
          { opacity: 1, x: 0, y: 0 }
        );
        return;
      }

      const timeline = gsap.timeline({ defaults: { ease: 'power3.out' } });

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
  }, [isMobile, prefersReducedMotion]);

  const scrollToNext = () => {
    const nextSection = document.querySelector('#about');
    if (nextSection) {
      nextSection.scrollIntoView({ behavior: 'smooth' });
    }
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
              className="fill-white"
            />
          </div>

          <div className="absolute top-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-0 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={500}
              direction="left"
              interactive={false}
              className="fill-white"
            />
          </div>

          <div className="absolute bottom-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-20 z-0 rotate-180 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={200}
              direction="right"
              interactive={false}
              className="fill-white"
            />
          </div>
        </>
      )}

      <div className="w-full max-w-[1560px] mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Tactical Framed Container from reference layout */}
        <div className="border border-white/15 bg-[#000000] shadow-2xl relative grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-white/15">
          {/* LEFT COLUMN: Editorial Headline, Subtitle, Description & Outlined CTA — Positioned lower down matching reference */}
          <div
            ref={textRef}
            className="flex flex-col justify-end p-8 sm:p-12 lg:p-14 xl:p-16 relative z-10 min-h-[520px] lg:min-h-[680px] xl:min-h-[740px] pt-16 sm:pt-24 lg:pt-36 xl:pt-48"
          >
            <div className="space-y-6">
              {/* Primary Headline with Interactive ParticleText */}
              <div className="space-y-3">
                <div ref={titleRef} className="w-full h-32 sm:h-40 md:h-48 xl:h-56">
                  <span className="sr-only">{t('hero_title')}</span>
                  <ParticleText
                    text="Pegasus"
                    particleSize={2.6}
                    density={4}
                    color="#f8fafc"
                    highlightColor="#10B981"
                    scatter={160}
                    gatherDuration={1500}
                    stagger={350}
                    pointerRepel={42}
                    repelRadius={120}
                    idleDrift={0.6}
                    trigger="mount"
                    fontSize="clamp(4.2rem, 8.5vw, 8rem)"
                    fontWeight={800}
                    textAlign="left"
                    glow
                  />
                </div>

                <h1
                  ref={subtitleRef}
                  className="block text-3xl sm:text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-normal text-white min-h-[1.25em] tracking-tight leading-[1.08]"
                >
                  <TextType
                    key={language}
                    text={typedPhrases}
                    typingSpeed={isMobile ? 90 : 70}
                    pauseDuration={1600}
                    deletingSpeed={isMobile ? 60 : 45}
                    showCursor={true}
                    cursorCharacter="|"
                    loop={true}
                    textColors={['#FFFFFF', '#C0C0C0']}
                    className="font-light"
                    cursorClassName="text-white font-light"
                  />
                </h1>
              </div>

              {/* Subtitle Description */}
              <p
                ref={descRef}
                className="text-lg sm:text-xl md:text-2xl text-white/70 font-light leading-relaxed max-w-2xl pt-2"
              >
                {t('hero_desc')}
              </p>
            </div>

            {/* Outlined Action Buttons matching reference layout */}
            <div ref={ctaRef} className="pt-8 sm:pt-10 flex flex-wrap items-center gap-4">
              <button
                onClick={scrollToNext}
                className="inline-flex items-center justify-center gap-3 px-10 py-4 border border-white/35 hover:border-white hover:bg-white hover:text-black transition-all text-sm sm:text-base font-semibold tracking-widest uppercase text-white group"
              >
                <span>{t('hero_explore')}</span>
                <ArrowRight className="w-5 h-5 transition-transform group-hover:translate-x-1" />
              </button>

              <a
                href="/join"
                className="inline-flex items-center justify-center gap-2 px-8 py-4 border border-white/15 bg-white/5 hover:bg-white/10 hover:border-white/35 text-white/80 hover:text-white transition-all text-sm sm:text-base font-semibold tracking-widest uppercase"
              >
                <span>{t('hero_demo')}</span>
              </a>
            </div>
          </div>

          {/* RIGHT COLUMN: 3D Isometric Wireframe Graphic & Bottom Metrics Bar — Starts high up */}
          <div
            ref={visualRef}
            className="flex flex-col justify-between bg-black relative overflow-hidden min-h-[520px] lg:min-h-[680px] xl:min-h-[740px]"
          >
            {/* Upper Area: Pure 3D Isometric Wireframe Visual positioned high */}
            <div className="flex-1 min-h-[360px] sm:min-h-[440px] lg:min-h-[480px] relative flex items-center justify-center p-4 sm:p-6 overflow-hidden">
              <IsometricTerrain />
            </div>

            {/* Bottom Metric Bar: 4-stat row strictly following the reference image */}
            <div className="border-t border-white/15 grid grid-cols-2 sm:grid-cols-4 divide-y sm:divide-y-0 sm:divide-x divide-white/15 bg-black">
              {metrics.map((m, idx) => (
                <div key={idx} className="p-4 sm:px-6 sm:py-5 flex flex-col justify-center">
                  <span className="text-[10px] sm:text-[11px] font-mono tracking-wider text-white/45 uppercase mb-1 truncate">
                    {m.label}
                  </span>
                  <span className="text-base sm:text-lg lg:text-xl font-mono font-medium text-white tracking-tight">
                    {m.value}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Scroll Indicator */}
      <button
        className="absolute bottom-3 left-1/2 transform -translate-x-1/2 cursor-pointer z-20 hidden md:block group focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-black outline-none rounded-lg p-2"
        onClick={scrollToNext}
        aria-label="Scroll to next section"
      >
        <div className="flex flex-col items-center gap-1.5 text-white/60 group-hover:text-[#FBFF63] transition-colors duration-300">
          <span className="text-[10px] font-mono tracking-widest uppercase">{t('hero_scroll')}</span>
          <svg
            className="w-4 h-4 animate-bounce"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 14l-7 7m0 0l-7-7m7 7V3"
            />
          </svg>
        </div>
      </button>
    </section>
  );
}

