'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { ArrowRight, Layers, LayoutDashboard } from 'lucide-react';
import ParticleText from './ParticleText';
import TextType from './TextType';
import IsometricTerrain from './IsometricTerrain';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';
import { useLanguage } from '../context/LanguageContext';

export default function Hero() {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  const { t, language } = useLanguage();
  const [visualMode, setVisualMode] = useState<'wireframe' | 'dashboard'>('wireframe');

  const textRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLDivElement>(null);
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
      className="min-h-screen relative flex items-center bg-[#000000] overflow-hidden pt-20 sm:pt-24 pb-12 sm:pb-16"
    >
      <div className="w-full max-w-[1560px] mx-auto px-4 sm:px-6 lg:px-8">
        {/* Tactical Framed Container from reference layout */}
        <div className="border border-white/15 bg-[#000000] shadow-2xl relative grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-white/15">
          {/* LEFT COLUMN: Editorial Headline, Subtitle, Description & Outlined CTA */}
          <div
            ref={textRef}
            className="flex flex-col justify-between p-8 sm:p-12 lg:p-14 xl:p-16 relative z-10"
          >
            <div className="space-y-6">
              {/* Primary Headline */}
              <div>
                <h1
                  ref={titleRef}
                  className="text-4xl sm:text-5xl md:text-6xl xl:text-7xl font-normal tracking-tight text-white leading-[1.08]"
                >
                  <span className="sr-only">{t('hero_title')}</span>
                  <span className="block font-semibold text-white tracking-tight mb-2">
                    Pegasus.
                  </span>
                  <span
                    ref={subtitleRef}
                    className="block text-2xl sm:text-3xl md:text-4xl xl:text-5xl font-light text-white/90 min-h-[1.25em]"
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
                  </span>
                </h1>
              </div>

              {/* Subtitle Description */}
              <p
                ref={descRef}
                className="text-sm sm:text-base md:text-lg text-white/60 font-light leading-relaxed max-w-lg pt-1"
              >
                {t('hero_desc')}
              </p>
            </div>

            {/* Outlined Action Buttons matching reference layout */}
            <div ref={ctaRef} className="pt-8 sm:pt-12 flex flex-wrap items-center gap-4">
              <button
                onClick={scrollToNext}
                className="inline-flex items-center justify-center gap-3 px-8 py-3.5 border border-white/30 hover:border-white hover:bg-white hover:text-black transition-all text-xs sm:text-sm font-medium tracking-widest uppercase text-white group"
              >
                <span>{t('hero_explore')}</span>
                <ArrowRight className="w-4 h-4 transition-transform group-hover:translate-x-1" />
              </button>

              <a
                href="/join"
                className="inline-flex items-center justify-center gap-2 px-7 py-3.5 border border-white/10 bg-white/5 hover:bg-white/10 hover:border-white/30 text-white/70 hover:text-white transition-all text-xs sm:text-sm font-medium tracking-widest uppercase"
              >
                <span>{t('hero_demo')}</span>
              </a>
            </div>
          </div>

          {/* RIGHT COLUMN: 3D Isometric Wireframe Graphic & Bottom Metrics Bar */}
          <div
            ref={visualRef}
            className="flex flex-col justify-between bg-black relative overflow-hidden"
          >
            {/* Mode Switcher pill */}
            <div className="absolute top-4 right-4 z-20 flex items-center gap-1 bg-white/5 border border-white/10 p-1 rounded-none text-[10px] font-mono uppercase tracking-wider text-white/60">
              <button
                onClick={() => setVisualMode('wireframe')}
                className={`flex items-center gap-1.5 px-2.5 py-1 transition-colors ${
                  visualMode === 'wireframe'
                    ? 'bg-white text-black font-semibold'
                    : 'text-white/60 hover:text-white'
                }`}
                aria-label="3D Wireframe View"
              >
                <Layers className="w-3 h-3" />
                <span>Topology</span>
              </button>
              <button
                onClick={() => setVisualMode('dashboard')}
                className={`flex items-center gap-1.5 px-2.5 py-1 transition-colors ${
                  visualMode === 'dashboard'
                    ? 'bg-white text-black font-semibold'
                    : 'text-white/60 hover:text-white'
                }`}
                aria-label="Platform Preview View"
              >
                <LayoutDashboard className="w-3 h-3" />
                <span>Platform</span>
              </button>
            </div>

            {/* Upper Area: Visual Display */}
            <div className="flex-1 min-h-[380px] sm:min-h-[460px] lg:min-h-[500px] relative flex items-center justify-center p-4 sm:p-6 overflow-hidden">
              {visualMode === 'wireframe' ? (
                <IsometricTerrain />
              ) : (
                <div className="relative w-full h-full max-w-xl max-h-[440px] overflow-hidden border border-white/15 bg-black p-2 shadow-2xl">
                  <img
                    src="/EbszSCwA.jpeg"
                    alt="Pegasus Logistics Platform Interface"
                    className="w-full h-full object-cover"
                  />
                </div>
              )}
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
    </section>
  );
}

