'use client';

import { useEffect, useRef } from 'react';
import { gsap, smoothScrollTo } from '@/app/lib/gsap';
import ParticleText from './ParticleText';
import TextType from './TextType';
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
 const videoRef = useRef<HTMLVideoElement>(null);

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
     [titleRef.current, subtitleRef.current, descRef.current, ctaRef.current],
     { opacity: 1, x: 0, y: 0 }
    );
    return;
   }

   const timeline = gsap.timeline({ defaults: { ease: 'pegasus' } });

   gsap.set(titleRef.current, { opacity: 1, y: 0 });

   timeline
    .fromTo(
     subtitleRef.current,
     { opacity: 0, y: 16 },
     { opacity: 1, y: 0, duration: 0.45 },
     0.08
    )
    .fromTo(
     descRef.current,
     { opacity: 0, y: 12 },
     { opacity: 1, y: 0, duration: 0.45 },
     0.18
    )
    .fromTo(
     ctaRef.current,
     { opacity: 0, y: 12 },
     { opacity: 1, y: 0, duration: 0.45 },
     0.28
    );
  });

  return () => ctx.revert();
 }, [isMobile, isLowEnd, prefersReducedMotion]);

 // Autoplay video when component mounts
 useEffect(() => {
  const video = videoRef.current;
  if (video) {
   video.play().catch(() => {
    // Autoplay blocked — silent fallback to poster
   });
  }
 }, []);

 const scrollToNext = () => {
  smoothScrollTo('#about', { offsetY: 64, duration: 1.1, ease: 'pegasus' });
 };

 return (
  <section
   id="hero"
   className="min-h-screen relative flex flex-col justify-end bg-[#000000] overflow-hidden"
  >
   {/* ── Video Background ── */}
   <div className="absolute inset-0 z-0">
   <video
    ref={videoRef}
    className="absolute inset-0 w-full h-full object-cover"
    src="/videos/hero-bg.mp4"
    muted
    loop
    playsInline
    preload="auto"
    poster="/images/topics/control_plane.jpg"
   />
   {/* Cinematic overlays for text readability */}
   <div className="absolute inset-0 bg-gradient-to-t from-black via-black/50 to-black/30 pointer-events-none" />
   <div className="absolute inset-0 bg-gradient-to-r from-black/70 via-black/30 to-transparent pointer-events-none" />
   <div className="absolute inset-0 bg-black/20 pointer-events-none" />
   </div>

   {/* ── Content Overlay ── */}
   <div className="w-full relative z-10">
   <div
    ref={textRef}
    className="flex flex-col justify-end p-8 sm:p-12 lg:p-16 xl:p-24 min-h-[80vh] lg:min-h-[85vh]"
   >
    <div className="space-y-4 max-w-3xl">
    {/* Primary Headline with Interactive ParticleText */}
    <div className="space-y-2">
     <div ref={titleRef} className="w-full h-20 sm:h-24 md:h-28 xl:h-32">
     <span className="sr-only">{t('hero_title')}</span>
     <ParticleText
      text="Pegasus"
      particleSize={2.2}
      density={4}
      color="#f8fafc"
      highlightColor="#10B981"
      scatter={190}
      gatherDuration={1600}
      stagger={420}
      pointerRepel={42}
      repelRadius={120}
      idleDrift={0.8}
      trigger="mount"
      fontSize="clamp(3.2rem, 6.2vw, 5.8rem)"
      fontWeight={800}
      fontFamily="inherit"
      textAlign="left"
      glow
     />
     </div>

     <h1
      ref={subtitleRef}
      className="block text-2xl sm:text-3xl md:text-4xl lg:text-5xl font-light min-h-[1.25em] tracking-tight leading-[1.12] text-white"
     >
      <TextType
       key={`${language}-hero-video`}
       text={typedPhrases}
       startFull={true}
       typingSpeed={isMobile ? 90 : 70}
       pauseDuration={2400}
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
     className="text-sm sm:text-base md:text-lg font-light leading-relaxed max-w-lg pt-1 text-white/60"
    >
     {t('hero_desc')}
    </p>
    </div>

    {/* Action Buttons */}
    <div ref={ctaRef} className="pt-6 sm:pt-8 flex flex-wrap items-center gap-4">
    <button
     onClick={scrollToNext}
     className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white text-black hover:bg-white/90"
    >
     <span>{t('hero_explore')}</span>
     <span className="text-lg leading-none mt-[-2px]">›</span>
    </button>

    <a
     href="/join"
     className="inline-flex items-center justify-center gap-2 px-8 py-3 transition-all text-sm sm:text-base font-medium bg-white/10 text-white hover:bg-white/20 backdrop-blur-sm"
    >
     <span>{t('hero_demo')}</span>
     <span className="text-lg leading-none mt-[-2px]">›</span>
    </a>
    </div>
   </div>
   </div>

   {/* ── Bottom scroll indicator ── */}
   <div className="absolute bottom-8 left-1/2 -translate-x-1/2 z-10 flex flex-col items-center gap-2 opacity-60">
   <div className="w-[1px] h-8 bg-gradient-to-b from-transparent to-white/60" />
   <button
    onClick={scrollToNext}
    className="text-white/50 hover:text-white transition-colors"
    aria-label="Scroll down"
   >
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
    <path d="M7 13l5 5 5-5M7 6l5 5 5-5" />
    </svg>
   </button>
   </div>
  </section>
 );
}
