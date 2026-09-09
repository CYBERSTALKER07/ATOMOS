'use client';

import React from 'react';
import Image from 'next/image';
import { useLanguage } from '@/app/context/LanguageContext';

type TenityOrbitBentoSectionProps = {
  className?: string;
};

export default function TenityOrbitBentoSection({ className = '' }: TenityOrbitBentoSectionProps) {
  const { language } = useLanguage();
  const isRu = language === 'ru';

  return (
    <section
      aria-label="Tenity Orbit Innovation Grid"
      className={`relative w-full bg-[#000000] py-16 sm:py-24 px-4 sm:px-6 md:px-8 lg:px-12 select-none overflow-hidden ${className}`}
    >
      <div className="w-full max-w-[1440px] mx-auto flex flex-col gap-5">
        {/* Top Grid Row: Lime Capsule Card + Dark Events Circular Portals + High-Contrast Office Photo */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch min-h-[340px]">
          {/* 1. Lime Card: The Latest from our Orbit */}
          <div className="md:col-span-3 bg-[#CEFF00] text-black rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-10 flex flex-col justify-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1 shadow-[0_10px_40px_rgba(206,255,0,0.18)]">
            <h3 className="font-title text-3xl sm:text-4xl lg:text-[42px] font-bold leading-[1.08] tracking-tight text-black max-w-[220px]">
              {isRu ? (
                <>
                  Последнее <br />
                  из нашего <br />
                  Orbit
                </>
              ) : (
                <>
                  The Latest <br />
                  from our <br />
                  Orbit
                </>
              )}
            </h3>
          </div>

          {/* 2. Center Card: Pitch-Black Card with Lime-Accented Circular Spotlight Windows */}
          <div className="md:col-span-6 bg-[#09090B] border border-[#CEFF00]/25 rounded-3xl p-6 sm:p-8 flex items-center justify-around gap-4 relative overflow-hidden shadow-2xl">
            {/* Left Circle: Upcoming Events */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-[#CEFF00]/50 hover:border-[#CEFF00] shadow-[0_0_30px_rgba(206,255,0,0.2)] flex-shrink-0 transition-colors duration-300">
              <Image
                src="/tenity/mics.png"
                alt="Upcoming Events Microphones"
                fill
                className="object-cover grayscale contrast-125 transition-all duration-500 group-hover/circle:scale-110 group-hover/circle:grayscale-0"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent" />
              <span className="absolute bottom-5 left-0 right-0 text-center font-title text-[#CEFF00] text-base sm:text-lg font-bold tracking-tight drop-shadow-[0_2px_10px_rgba(0,0,0,0.8)]">
                {isRu ? 'Предстоящие события' : 'Upcoming Events'}
              </span>
            </div>

            {/* Right Circle: Neon Rocket Portal */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-[#CEFF00]/50 hover:border-[#CEFF00] shadow-[0_0_30px_rgba(206,255,0,0.2)] flex-shrink-0 transition-colors duration-300">
              <Image
                src="/tenity/rocket.png"
                alt="Neon Rocket Portal"
                fill
                className="object-cover transition-transform duration-500 group-hover/circle:scale-110"
              />
              <div className="absolute inset-0 bg-black/30 pointer-events-none group-hover/circle:bg-transparent transition-colors" />
            </div>
          </div>

          {/* 3. Right Card: Tech Workspace / Founders Photo (Tactical Monochrome) */}
          <div className="md:col-span-3 relative rounded-3xl overflow-hidden min-h-[260px] md:min-h-[340px] border border-[#CEFF00]/25 hover:border-[#CEFF00]/60 transition-colors duration-300 group">
            <Image
              src="/tenity/office.png"
              alt="Founders and Tech Workspace"
              fill
              className="object-cover grayscale contrast-125 transition-all duration-500 group-hover:scale-105 group-hover:grayscale-0"
            />
            <div className="absolute inset-0 bg-black/40 group-hover:bg-black/10 transition-colors" />
          </div>
        </div>

        {/* Bottom Grid Row: Giant Lime Capsule with 100+ Metric & 3D Dotted Globe + Pure Black/Lime Card */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch min-h-[360px]">
          {/* 4. Giant Lime Capsule: 100+ PoCs + 3D Dotted Globe Semicircle */}
          <div className="md:col-span-9 bg-[#CEFF00] text-black rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-12 lg:p-14 flex flex-col md:flex-row justify-between items-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1 shadow-[0_15px_50px_rgba(206,255,0,0.22)]">
            {/* Left side: Metric & Subtext */}
            <div className="flex flex-col justify-between h-full z-10 w-full md:w-1/2">
              <div className="font-title text-7xl sm:text-8xl md:text-9xl lg:text-[140px] font-bold tracking-[-0.045em] text-black leading-none">
                100+
              </div>
              <p className="font-semibold text-sm sm:text-base md:text-lg text-black/95 leading-[1.35] max-w-sm mt-8 md:mt-12">
                {isRu
                  ? 'Пилотных проектов (PoC), реализованных через Visa Innovation Program Europe'
                  : 'PoCs facilitated through the Visa Innovation Program Europe'}
              </p>
            </div>

            {/* Right side: Halftone Dotted Globe Sphere */}
            <div className="relative w-72 h-72 sm:w-80 sm:h-80 md:w-[380px] md:h-[380px] lg:w-[420px] lg:h-[420px] -mr-8 sm:-mr-12 md:-mr-16 flex-shrink-0">
              <Image
                src="/tenity/globe.png"
                alt="3D Halftone Dotted Globe"
                fill
                className="object-contain drop-shadow-2xl animate-spin-slow invert"
                style={{ animationDuration: '60s' }}
              />
            </div>
          </div>

          {/* 5. Pure Black & Lime Card: One of Denmark's largest AI rounds */}
          <div className="md:col-span-3 bg-[#09090B] border border-[#CEFF00]/30 text-white rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-10 flex flex-col justify-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1 hover:border-[#CEFF00] shadow-2xl">
            <h3 className="font-title text-2xl sm:text-3xl lg:text-[34px] font-bold leading-[1.15] tracking-tight text-[#CEFF00]">
              {isRu ? (
                <>
                  Один из <br />
                  крупнейших <br />
                  раундов в <br />
                  сфере ИИ <br />
                  в Дании
                </>
              ) : (
                <>
                  One of <br />
                  Denmark&apos;s <br />
                  largest <br />
                  AI/Deeptech <br />
                  rounds
                </>
              )}
            </h3>
          </div>
        </div>
      </div>
    </section>
  );
}
