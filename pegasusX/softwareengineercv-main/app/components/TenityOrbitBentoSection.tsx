'use client';

import React from 'react';
import Image from 'next/image';
import Link from 'next/link';
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
      className={`relative w-full bg-[#09090B] py-16 sm:py-24 px-4 sm:px-6 md:px-8 lg:px-12 select-none overflow-hidden ${className}`}
    >
      <div className="w-full max-w-[1440px] mx-auto flex flex-col gap-5">
        {/* Floating Tenity-Style Pill Bar */}
        <div className="w-full flex items-center justify-between bg-white text-black rounded-2xl sm:rounded-full px-6 sm:px-8 py-3 shadow-md mb-2">
          {/* Logo */}
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 fill-black" viewBox="0 0 24 24">
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
            </svg>
            <span className="font-title text-xl font-bold tracking-tight">Tenity</span>
          </div>

          {/* Nav Links (Desktop) */}
          <nav className="hidden lg:flex items-center gap-8 text-xs sm:text-[13px] font-medium tracking-normal text-black/90">
            <Link href="/solutions" className="hover:text-black transition-colors">
              {isRu ? 'Венчурный капитал' : 'Venture Capital'}
            </Link>
            <Link href="/operations" className="hover:text-black transition-colors">
              {isRu ? 'Инновационные сервисы' : 'Innovation Services'}
            </Link>
            <Link href="/platform" className="hover:text-black transition-colors">
              {isRu ? 'Стартапы' : 'Startups'}
            </Link>
            <Link href="/about" className="hover:text-black transition-colors">
              {isRu ? 'О платформе' : 'About Tenity'}
            </Link>
            <Link href="/demo" className="hover:text-black transition-colors font-semibold">
              Orbit
            </Link>
          </nav>

          {/* Right Action Buttons */}
          <div className="flex items-center gap-4 sm:gap-6">
            <Link
              href="/join"
              className="text-xs sm:text-[13px] font-semibold underline underline-offset-4 hover:opacity-75"
            >
              {isRu ? 'Войти' : 'Login'}
            </Link>
            <Link
              href="/contact"
              className="bg-[#FF3E60] text-white text-xs sm:text-[13px] font-semibold px-5 sm:px-6 py-2.5 rounded-full hover:bg-[#e63554] transition-colors"
            >
              {isRu ? 'Связаться' : 'Contact'}
            </Link>
          </div>
        </div>

        {/* Top Grid Row: Green Pill Card + Dark Events Circular Portals + Office Photo */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch min-h-[340px]">
          {/* 1. Green Card: The Latest from our Orbit */}
          <div className="md:col-span-3 bg-[#3BDD8F] text-black rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-10 flex flex-col justify-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1">
            <h3 className="font-title text-3xl sm:text-4xl lg:text-[42px] font-normal leading-[1.08] tracking-tight text-black max-w-[220px]">
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

          {/* 2. Center Card: Dark Card with 2 Circular Spotlight Windows */}
          <div className="md:col-span-6 bg-[#0E0E12] border border-white/10 rounded-3xl p-6 sm:p-8 flex items-center justify-around gap-4 relative overflow-hidden">
            {/* Left Circle: Microphones + Upcoming Events */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-white/20 shadow-2xl flex-shrink-0">
              <Image
                src="/tenity/mics.png"
                alt="Upcoming Events Microphones"
                fill
                className="object-cover transition-transform duration-500 group-hover/circle:scale-110"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent" />
              <span className="absolute bottom-5 left-0 right-0 text-center font-title text-white text-base sm:text-lg font-medium tracking-tight">
                {isRu ? 'Предстоящие события' : 'Upcoming Events'}
              </span>
            </div>

            {/* Right Circle: Neon Rocket */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-white/20 shadow-2xl flex-shrink-0">
              <Image
                src="/tenity/rocket.png"
                alt="Neon Rocket Portal"
                fill
                className="object-cover transition-transform duration-500 group-hover/circle:scale-110"
              />
              <div className="absolute inset-0 bg-purple-900/10 pointer-events-none" />
            </div>
          </div>

          {/* 3. Right Card: Tech Workspace / Founders Photo */}
          <div className="md:col-span-3 relative rounded-3xl overflow-hidden min-h-[260px] md:min-h-[340px] border border-white/10 group">
            <Image
              src="/tenity/office.png"
              alt="Founders and Tech Workspace"
              fill
              className="object-cover transition-transform duration-500 group-hover:scale-105"
            />
            <div className="absolute inset-0 bg-black/15 group-hover:bg-transparent transition-colors" />
          </div>
        </div>

        {/* Bottom Grid Row: Giant Coral Pill with 100+ Metric & 3D Dotted Globe + Blue AI Card */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch min-h-[360px]">
          {/* 4. Giant Coral Capsule: 100+ PoCs + 3D Dotted Globe Semicircle */}
          <div className="md:col-span-9 bg-[#FF3E60] text-black rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-12 lg:p-14 flex flex-col md:flex-row justify-between items-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1">
            {/* Left side: Metric & Subtext */}
            <div className="flex flex-col justify-between h-full z-10 w-full md:w-1/2">
              <div className="font-title text-7xl sm:text-8xl md:text-9xl lg:text-[140px] font-light tracking-[-0.045em] text-black leading-none">
                100+
              </div>
              <p className="font-normal text-sm sm:text-base md:text-lg text-black/90 leading-[1.35] max-w-sm mt-8 md:mt-12">
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
                className="object-contain drop-shadow-2xl animate-spin-slow"
                style={{ animationDuration: '60s' }}
              />
            </div>
          </div>

          {/* 5. Periwinkle Blue Card: One of Denmark's largest AI rounds */}
          <div className="md:col-span-3 bg-[#90B8F6] text-black rounded-3xl md:rounded-l-3xl md:rounded-r-full p-8 sm:p-10 flex flex-col justify-center relative overflow-hidden transition-transform duration-300 hover:-translate-y-1">
            <h3 className="font-title text-2xl sm:text-3xl lg:text-[34px] font-normal leading-[1.15] tracking-tight text-black">
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
