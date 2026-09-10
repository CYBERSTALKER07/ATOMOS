'use client';

import React from 'react';
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
        {/* Top Grid Row: Lime Capsule Card + Dark Events Circular Portals + High-Tech Telemetry Card */}
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

          {/* 2. Center Card: Pitch-Black Card with Lime-Accented Circular Tactical Radar & Orbit Portals */}
          <div className="md:col-span-6 bg-[#09090B] border border-[#CEFF00]/25 rounded-3xl p-6 sm:p-8 flex items-center justify-around gap-4 relative overflow-hidden shadow-2xl">
            {/* Left Circle: Upcoming Events Radar Portal */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-[#CEFF00]/50 hover:border-[#CEFF00] bg-black shadow-[0_0_30px_rgba(206,255,0,0.15)] flex-shrink-0 transition-all duration-300 flex flex-col items-center justify-center">
              {/* Tactical Radar Background */}
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none opacity-40">
                <div className="w-3/4 h-3/4 rounded-full border border-dashed border-[#CEFF00]/40 animate-[spin_20s_linear_infinite]" />
                <div className="absolute w-1/2 h-1/2 rounded-full border border-[#CEFF00]/30" />
                <div className="absolute w-full h-[1px] bg-[#CEFF00]/20" />
                <div className="absolute h-full w-[1px] bg-[#CEFF00]/20" />
              </div>

              {/* Animated Radar Sweep */}
              <div className="absolute inset-0 bg-[conic-gradient(from_0deg,transparent_0deg,rgba(206,255,0,0.25)_60deg,transparent_60.1deg)] animate-[spin_4s_linear_infinite] pointer-events-none rounded-full" />

              {/* Live Blip */}
              <div className="relative z-10 flex flex-col items-center gap-2">
                <div className="relative flex items-center justify-center">
                  <span className="absolute w-6 h-6 rounded-full bg-[#CEFF00]/30 animate-ping" />
                  <span className="w-3.5 h-3.5 rounded-full bg-[#CEFF00] shadow-[0_0_12px_#CEFF00]" />
                </div>
                <div className="px-2 py-0.5 rounded bg-[#CEFF00]/10 border border-[#CEFF00]/40 text-[10px] font-mono uppercase tracking-widest text-[#CEFF00]">
                  Live Pulse
                </div>
              </div>

              {/* Label */}
              <span className="absolute bottom-5 left-0 right-0 text-center font-title text-[#CEFF00] text-sm sm:text-base font-bold tracking-tight z-10 drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)]">
                {isRu ? 'Предстоящие события' : 'Upcoming Events'}
              </span>
            </div>

            {/* Right Circle: Orbital Trajectory Portal */}
            <div className="relative group/circle cursor-pointer w-44 h-44 sm:w-56 sm:h-56 rounded-full overflow-hidden border-2 border-[#CEFF00]/50 hover:border-[#CEFF00] bg-black shadow-[0_0_30px_rgba(206,255,0,0.15)] flex-shrink-0 transition-all duration-300 flex flex-col items-center justify-center">
              {/* Concentric Gyroscope Rings */}
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="w-[85%] h-[85%] rounded-full border border-[#CEFF00]/30 animate-[spin_12s_linear_infinite]" />
                <div className="absolute w-[60%] h-[60%] rounded-full border border-dashed border-[#CEFF00]/50 animate-[spin_8s_linear_infinite_reverse]" />
                <div className="absolute w-[35%] h-[35%] rounded-full border border-[#CEFF00]/40" />
                {/* Orbiting Satellite Node */}
                <div className="absolute w-full h-full animate-[spin_6s_linear_infinite]">
                  <div className="w-2.5 h-2.5 rounded-full bg-[#CEFF00] shadow-[0_0_10px_#CEFF00] absolute top-2 left-1/2 -translate-x-1/2" />
                </div>
              </div>

              {/* Center Launch Icon / Core */}
              <div className="relative z-10 flex flex-col items-center gap-1.5">
                <svg
                  className="w-8 h-8 text-[#CEFF00] drop-shadow-[0_0_10px_rgba(206,255,0,0.6)] transform group-hover/circle:-translate-y-1 group-hover/circle:scale-110 transition-transform duration-300"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z" />
                  <path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z" />
                  <path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0" />
                  <path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5" />
                </svg>
                <span className="text-[10px] font-mono tracking-widest text-[#CEFF00]/80 uppercase">
                  Telemetry
                </span>
              </div>

              <span className="absolute bottom-5 left-0 right-0 text-center font-title text-[#CEFF00] text-sm sm:text-base font-bold tracking-tight z-10 drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)]">
                {isRu ? 'Орбитальная Сеть' : 'Orbit Trajectory'}
              </span>
            </div>
          </div>

          {/* 3. Right Card: Tactical Fleet & Hub Telemetry Console */}
          <div className="md:col-span-3 relative rounded-3xl overflow-hidden min-h-[260px] md:min-h-[340px] bg-[#09090B] border border-[#CEFF00]/25 hover:border-[#CEFF00]/60 transition-colors duration-300 p-6 flex flex-col justify-between group">
            {/* Top Badge & Node Indicator */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 rounded-full bg-[#CEFF00] animate-pulse" />
                <span className="text-[11px] font-mono uppercase tracking-wider text-[#CEFF00]">
                  SYS_CORE // OK
                </span>
              </div>
              <span className="text-[10px] font-mono text-[#CEFF00]/60">99.99% SLA</span>
            </div>

            {/* Tactical Waveform / Signal Visual */}
            <div className="my-auto py-4">
              <div className="flex items-end justify-between gap-1.5 h-16 w-full px-2">
                {[40, 65, 30, 85, 95, 45, 75, 55, 90, 60, 100, 70, 80, 50, 65].map((h, i) => (
                  <div
                    key={i}
                    style={{ height: `${h}%` }}
                    className="flex-1 bg-[#CEFF00]/20 group-hover:bg-[#CEFF00]/50 rounded-t-sm transition-all duration-300"
                  />
                ))}
              </div>
              <div className="mt-3 flex justify-between text-[9px] font-mono text-[#CEFF00]/50 border-t border-[#CEFF00]/20 pt-2">
                <span>TAS-IX IXP: 1.2ms</span>
                <span>CELL-UZ: SYNC</span>
              </div>
            </div>

            {/* Bottom Details */}
            <div className="pt-3 border-t border-[#CEFF00]/15 flex items-center justify-between">
              <div>
                <div className="text-xs font-bold text-white uppercase tracking-wider">
                  {isRu ? 'Инновационный Узел' : 'Global Hub Matrix'}
                </div>
                <div className="text-[10px] font-mono text-[#CEFF00]/80 mt-0.5">
                  5,400+ TPS Distributed
                </div>
              </div>
              <div className="w-7 h-7 rounded-full bg-[#CEFF00]/10 border border-[#CEFF00]/40 flex items-center justify-center text-[#CEFF00] group-hover:bg-[#CEFF00] group-hover:text-black transition-colors">
                ↗
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Grid Row: Giant Lime Capsule with 100+ Metric & Halftone Vector Globe + Pure Black/Lime Card */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-5 items-stretch min-h-[360px]">
          {/* 4. Giant Lime Capsule: 100+ PoCs + Pure SVG Halftone Globe */}
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

            {/* Right side: Pure SVG Halftone Globe Graphic (Zero Raster Images) */}
            <div className="relative w-64 h-64 sm:w-80 sm:h-80 md:w-[360px] md:h-[360px] lg:w-[400px] lg:h-[400px] -mr-6 sm:-mr-10 md:-mr-14 flex-shrink-0 flex items-center justify-center">
              {/* Outer Orbit Ring */}
              <div className="absolute inset-2 rounded-full border border-dashed border-black/30 animate-[spin_45s_linear_infinite]" />
              
              {/* Tactical SVG Globe Sphere */}
              <svg
                viewBox="0 0 300 300"
                className="w-full h-full text-black drop-shadow-lg"
                fill="none"
              >
                {/* Globe Sphere Boundary */}
                <circle cx="150" cy="150" r="130" stroke="currentColor" strokeWidth="3" opacity="0.85" />
                
                {/* Latitudes */}
                <ellipse cx="150" cy="150" rx="130" ry="40" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" opacity="0.6" />
                <ellipse cx="150" cy="150" rx="130" ry="85" stroke="currentColor" strokeWidth="1.5" strokeDasharray="4 4" opacity="0.7" />
                <line x1="20" y1="150" x2="280" y2="150" stroke="currentColor" strokeWidth="2" opacity="0.8" />
                
                {/* Longitudes */}
                <ellipse cx="150" cy="150" rx="45" ry="130" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 3" opacity="0.6" />
                <ellipse cx="150" cy="150" rx="90" ry="130" stroke="currentColor" strokeWidth="1.5" strokeDasharray="4 4" opacity="0.7" />
                <line x1="150" y1="20" x2="150" y2="280" stroke="currentColor" strokeWidth="2" opacity="0.8" />

                {/* Halftone Dotted Network Nodes */}
                {[
                  [150, 70], [180, 95], [120, 100], [210, 130], [90, 140],
                  [160, 150], [135, 175], [195, 180], [105, 205], [150, 225],
                  [80, 110], [225, 160], [170, 200], [115, 130], [185, 140]
                ].map(([cx, cy], i) => (
                  <circle
                    key={i}
                    cx={cx}
                    cy={cy}
                    r={i % 3 === 0 ? 5 : 3.5}
                    fill="currentColor"
                    className="animate-pulse"
                    style={{ animationDuration: `${2 + (i % 3)}s`, animationDelay: `${(i * 0.2)}s` }}
                  />
                ))}

                {/* Interconnecting Flight / Routing Arcs */}
                <path
                  d="M120 100 Q 150 120 180 95"
                  stroke="currentColor"
                  strokeWidth="2"
                  opacity="0.8"
                />
                <path
                  d="M90 140 Q 125 155 160 150"
                  stroke="currentColor"
                  strokeWidth="2"
                  opacity="0.8"
                />
                <path
                  d="M160 150 Q 180 165 195 180"
                  stroke="currentColor"
                  strokeWidth="2"
                  opacity="0.8"
                />
                <path
                  d="M135 175 Q 142 200 150 225"
                  stroke="currentColor"
                  strokeWidth="2"
                  opacity="0.8"
                />
              </svg>
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

