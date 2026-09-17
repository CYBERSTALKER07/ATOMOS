'use client';

import React, { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { useLanguage } from '../context/LanguageContext';
import { getTestimonials, type O9Testimonial } from '../data/o9FleekDefaults';

// Custom dither overlay component to give portrait images the dithered matrix halftone aesthetic from the design reference
function DitheredPortrait({ src, alt }: { src: string; alt: string }) {
  return (
    <div className="relative w-full h-full bg-black overflow-hidden flex items-center justify-center group">
      {/* High-contrast grayscale portrait - centered on face and shoulders */}
      <Image
        src={src}
        alt={alt}
        width={800}
        height={1200}
        priority
        className="w-full h-full object-cover object-[50%_32%] grayscale contrast-125 brightness-105 transition-transform duration-700 group-hover:scale-105"
      />

      {/* Subtle Scanlines effect */}
      <div 
        className="absolute inset-0 pointer-events-none opacity-15 bg-repeat"
        style={{
          backgroundImage: `linear-gradient(to bottom, transparent 50%, rgba(0, 0, 0, 0.8) 51%)`,
          backgroundSize: '100% 4px',
        }}
      />
    </div>
  );
}

export function PegasusTestimonialsSection() {
  const { t, language } = useLanguage();
  const testimonials = getTestimonials(language);
  const [activeRole, setActiveRole] = useState<string>('ALL');

  const roleFilters = [
    { id: 'ALL', label: language === 'ru' ? 'ВСЕ РОЛИ' : 'ALL ROLES' },
    { id: 'SUPPLIER', label: language === 'ru' ? 'ПОСТАВЩИК' : 'SUPPLIER' },
    { id: 'WAREHOUSE', label: language === 'ru' ? 'СКЛАД' : 'WAREHOUSE' },
    { id: 'RETAILER', label: language === 'ru' ? 'РИТЕЙЛЕР' : 'RETAILER' },
    { id: 'FACTORY', label: language === 'ru' ? 'ФАБРИКА' : 'FACTORY' },
    { id: 'DRIVER', label: language === 'ru' ? 'ВОДИТЕЛЬ' : 'DRIVER' },
    { id: 'PAYLOAD', label: language === 'ru' ? 'ТЕРМИНАЛ' : 'PAYLOAD' },
  ];

  const filteredList = activeRole === 'ALL'
    ? testimonials
    : testimonials.filter((item) => {
        const badge = (item.roleBadge || '').toUpperCase();
        if (activeRole === 'SUPPLIER') return badge === 'SUPPLIER' || badge === 'ПОСТАВЩИК';
        if (activeRole === 'WAREHOUSE') return badge === 'WAREHOUSE' || badge === 'СКЛАД';
        if (activeRole === 'RETAILER') return badge === 'RETAILER' || badge === 'РИТЕЙЛЕР';
        if (activeRole === 'FACTORY') return badge === 'FACTORY' || badge === 'ФАБРИКА';
        if (activeRole === 'DRIVER') return badge === 'DRIVER' || badge === 'ВОДИТЕЛЬ';
        if (activeRole === 'PAYLOAD') return badge === 'PAYLOAD' || badge === 'ТЕРМИНАЛ';
        return true;
      });

  return (
    <section className="w-full bg-[#080808] text-white py-24 px-4 sm:px-8 flex flex-col items-center font-sans antialiased relative overflow-hidden select-none border-t border-zinc-900">
      
      {/* Subtle grid pattern background */}
      <div 
        className="absolute inset-0 pointer-events-none opacity-10 bg-repeat"
        style={{
          backgroundImage: `linear-gradient(to right, #27272a 1px, transparent 1px), linear-gradient(to bottom, #27272a 1px, transparent 1px)`,
          backgroundSize: '40px 40px',
        }}
      />

      <div className="max-w-6xl w-full mx-auto flex flex-col items-center gap-16 relative z-10">

        {/* Section Header */}
        <div className="flex flex-col items-center text-center max-w-2xl gap-3">

          <h2 className="text-3xl sm:text-4xl md:text-5xl font-bold tracking-tight text-zinc-100">
            {t('testimonials_title') || 'Trusted by operators across the supply chain'}
          </h2>
          <p className="text-zinc-400 text-sm sm:text-base leading-relaxed">
            {t('testimonials_subtitle') || 'Single dispatch board, real-time telemetry, and zero-friction settlement for every role.'}
          </p>
        </div>

        {/* ========================================================================= */}
        {/* HERO FEATURED CARD - EXACT CYBERPUNK / TECHNICAL GRID STYLE FROM REFERENCE */}
        {/* ========================================================================= */}
        <div className="w-full relative my-4">
          
          {/* Extended Outer Grid Alignment Lines */}
          <div className="absolute left-[-30px] right-[-30px] top-0 h-[1px] bg-zinc-800/80 pointer-events-none hidden sm:block" />
          <div className="absolute left-[-30px] right-[-30px] bottom-0 h-[1px] bg-zinc-800/80 pointer-events-none hidden sm:block" />
          <div className="absolute top-[-30px] bottom-[-30px] left-0 w-[1px] bg-zinc-800/80 pointer-events-none hidden sm:block" />
          <div className="absolute top-[-30px] bottom-[-30px] right-0 w-[1px] bg-zinc-800/80 pointer-events-none hidden sm:block" />

          {/* Main Card Frame with Outer Border */}
          <div className="w-full bg-[#0d0d0d] border border-zinc-800 relative">

            {/* Corner Node Handle Squares (□) at frame intersections */}
            <div className="w-2.5 h-2.5 bg-[#0d0d0d] border border-zinc-400 absolute -top-1.25 -left-1.25 z-20" />
            <div className="w-2.5 h-2.5 bg-[#0d0d0d] border border-zinc-400 absolute -top-1.25 -right-1.25 z-20" />
            <div className="w-2.5 h-2.5 bg-[#0d0d0d] border border-zinc-400 absolute -bottom-1.25 -left-1.25 z-20" />
            <div className="w-2.5 h-2.5 bg-[#0d0d0d] border border-zinc-400 absolute -bottom-1.25 -right-1.25 z-20" />

            {/* Top Bar Header Row */}
            <div className="flex items-center justify-between border-b border-zinc-800 px-5 py-2.5 bg-[#0a0a0a]">
              <div className="flex items-center gap-2 font-mono text-xs text-white tracking-wider">
                <span className="w-1.5 h-1.5 bg-white inline-block" />
                <span>CASE STUDY</span>
              </div>
              <div className="font-mono text-xs text-zinc-500 tracking-widest hidden sm:block">
                SYS.REF · PEGASUS_CTO_01
              </div>
            </div>

            {/* Middle Grid Row: Left Content | Right Dithered Portrait */}
            <div className="grid grid-cols-1 lg:grid-cols-12 items-stretch">
              
              {/* Left Column: Content & CTA (7 cols on lg) */}
              <div className="lg:col-span-7 p-5 sm:p-7 flex flex-col justify-between gap-4 border-b lg:border-b-0 lg:border-r border-zinc-800">
                <div className="flex flex-col gap-2.5">
                  <h3 className="text-lg sm:text-xl md:text-2xl font-normal tracking-tight text-zinc-100 leading-snug">
                    {t('cto_quote')}
                  </h3>
                  <p className="text-zinc-400 text-xs sm:text-sm leading-relaxed max-w-xl">
                    {language === 'ru' 
                      ? 'Оцените, как передовые логистические команды поддерживают высокую скорость диспетчеризации и прозрачность расчетов на единой платформе Pegasus.'
                      : 'See how leading logistics operators maintain fast dispatch and rigorous settlement accuracy across all 6 supply chain roles.'}
                  </p>
                </div>

                {/* Monochrome CTA Button */}
                <div className="pt-2">
                  <Link 
                    href="/contact"
                    className="inline-flex items-center gap-2.5 px-5 py-2.5 bg-white hover:bg-zinc-200 text-black font-medium text-xs sm:text-sm transition-colors rounded-none shadow-sm group border border-white"
                  >
                    <span>{t('read_case_study') || 'Read case study'}</span>
                    <span className="group-hover:translate-x-1 transition-transform font-bold">→</span>
                  </Link>
                </div>
              </div>

              {/* Right Column: Dithered Halftone Image & Meta (5 cols on lg) */}
              <div className="lg:col-span-5 flex flex-col justify-between bg-black relative h-full">
                
                {/* Dithered Portrait Container - Properly Proportioned */}
                <div className="w-full flex-1 min-h-[300px] sm:min-h-[340px] lg:min-h-[320px] relative border-b border-zinc-800 overflow-hidden bg-black">
                  <DitheredPortrait 
                    src="/Gemini_Generated_Image_e86uare86uare86u.png" 
                    alt={t('cto_role')}
                  />
                  
                  {/* Grid handle nodes inside portrait container frame */}
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute top-2 left-2 z-20" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute top-2 right-2 z-20" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute bottom-2 left-2 z-20" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute bottom-2 right-2 z-20" />
                </div>

                {/* Bottom Metadata Bar */}
                <div className="px-5 py-2.5 bg-[#0a0a0a] flex items-center justify-between font-mono text-[11px] text-zinc-400 tracking-wider">
                  <span className="truncate">— {t('cto_name').toUpperCase()} · {t('cto_role').toUpperCase()}</span>
                </div>

              </div>

            </div>

          </div>

        </div>

        {/* ========================================================================= */}
        {/* ROLE FILTER TABS & TECHNICAL CARDS GRID (HIDDEN) */}
        {/* ========================================================================= */}
        <div className="hidden" aria-hidden="true">
          {/* Role Filters */}
          <div className="flex flex-wrap items-center justify-center gap-2 max-w-full">
            {roleFilters.map((role) => (
              <button
                key={role.id}
                onClick={() => setActiveRole(role.id)}
                className={`px-4 py-2 text-xs font-mono tracking-wider transition-all border rounded-none ${
                  activeRole === role.id
                    ? 'bg-white text-black border-white font-bold shadow-md'
                    : 'bg-[#0d0d0d] text-zinc-400 border-zinc-800 hover:border-zinc-700 hover:text-zinc-200'
                }`}
              >
                [ {role.label} ]
              </button>
            ))}
          </div>

          {/* Technical Role Testimonials Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 w-full">
            {filteredList.map((item: O9Testimonial, idx: number) => {
              return (
                <div
                  key={`${item.company}-${idx}`}
                  className="w-full bg-[#0d0d0d] border border-zinc-800 hover:border-zinc-700 transition-all duration-300 relative group flex flex-col justify-between"
                >
                  {/* Corner Node Handles (□) */}
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute -top-1 -left-1 z-20 group-hover:border-zinc-300 transition-colors" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute -top-1 -right-1 z-20 group-hover:border-zinc-300 transition-colors" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute -bottom-1 -left-1 z-20 group-hover:border-zinc-300 transition-colors" />
                  <div className="w-2 h-2 bg-[#0d0d0d] border border-zinc-500 absolute -bottom-1 -right-1 z-20 group-hover:border-zinc-300 transition-colors" />

                  {/* Card Top Header */}
                  <div className="flex items-center justify-between border-b border-zinc-800 px-5 py-2.5 bg-[#0a0a0a]">
                    <div className="flex items-center gap-2 font-mono text-[11px] text-zinc-300">
                      <span className="w-1.5 h-1.5 bg-white inline-block" />
                      [ {item.roleBadge || 'ROLE NODE'} ]
                    </div>
                    {item.metric && (
                      <span className="font-mono text-[10px] text-zinc-400 bg-zinc-900 px-2 py-0.5 border border-zinc-800">
                        {item.metric}
                      </span>
                    )}
                  </div>

                  {/* Card Body Quote */}
                  <div className="p-6 flex-1 flex flex-col justify-between gap-4">
                    <p className="text-zinc-300 text-sm leading-relaxed font-sans">
                      &ldquo;{item.quote}&rdquo;
                    </p>
                    <p className="text-xs font-mono font-semibold text-zinc-400 tracking-wide">
                      {item.company}
                    </p>
                  </div>

                  {/* Card Bottom Meta Bar */}
                  <div className="px-5 py-3 border-t border-zinc-800 bg-[#0a0a0a] flex items-center justify-between font-mono text-[11px] text-zinc-400">
                    <div className="flex items-center gap-2 truncate">
                      <span className="w-5 h-5 bg-zinc-800 border border-zinc-700 text-zinc-200 flex items-center justify-center text-[10px] font-bold flex-shrink-0">
                        {item.initials || item.name.substring(0, 2).toUpperCase()}
                      </span>
                      <span className="text-zinc-200 truncate">{item.name}</span>
                    </div>
                  </div>

                </div>
              );
            })}
          </div>
        </div>

      </div>
    </section>
  );
}
