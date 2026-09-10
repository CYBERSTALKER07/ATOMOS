'use client';

import React from 'react';
import { Layers, Terminal, Sparkles, ArrowRight } from 'lucide-react';

export type BrandEcosystemProps = {
  title?: string;
  badge?: string;
  description?: string;
  className?: string;
};

export default function BrandEcosystemSpec({
  title = '_ Pegasus Ecosystem',
  badge = '6+ RUNTIMES',
  description = 'The engineering team has unified the supply ecosystem by delivering dedicated, high-performance client runtimes for every participant: Supplier Web, Warehouse Desktop, Driver Mission, and Retailer Telegram. Ensuring zero divergence from the shared transactional core.',
  className = '',
}: BrandEcosystemProps) {
  const products = [
    {
      quarter: '2023 Q2',
      name: 'Pegasus Core',
      role: 'DISTRIBUTED GO ENGINE',
      iconText: 'CORE',
      accent: 'border-white/20 text-white',
    },
    {
      quarter: '2023 Q3',
      name: 'Pegasus Desktop',
      role: 'WAREHOUSE TAURI V2',
      iconText: 'TAURI',
      accent: 'border-white/20 text-white',
    },
    {
      quarter: '2024 Q2',
      name: 'Pegasus Driver',
      role: 'NATIVE KOTLIN & SWIFT',
      iconText: 'MOBILE',
      accent: 'border-white/20 text-white',
    },
    {
      quarter: '2024 Q3',
      name: 'Pegasus Telegram',
      role: 'RETAILER MINI APP',
      iconText: 'TG-BOT',
      accent: 'border-orange-500/40 text-orange-400 bg-orange-500/10',
    },
  ];

  const colorSwatches = [
    { hex: '#F8FAFC', label: 'LIGHT INK', bg: 'bg-[#F8FAFC]', text: 'text-black' },
    { hex: '#94A3B8', label: 'MUTED BORDER', bg: 'bg-[#94A3B8]', text: 'text-black' },
    { hex: '#1E293B', label: 'TACTICAL SLATE', bg: 'bg-[#1E293B]', text: 'text-white' },
    { hex: '#09090B', label: 'OBSIDIAN CORE', bg: 'bg-[#09090B] border border-white/20', text: 'text-white' },
    { hex: '#FF7A1A', label: 'SAFETY ORANGE', bg: 'bg-[#FF7A1A]', text: 'text-black' },
    { hex: '#E2FD52', label: 'TACTICAL LIME', bg: 'bg-[#E2FD52]', text: 'text-black' },
  ];

  return (
    <section className={`w-full py-16 sm:py-24 text-white font-sans ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Main Spec Frame with Hairline Grid Borders */}
        <div className="border border-white/15 bg-[#08080B] rounded-3xl overflow-hidden shadow-2xl">
          {/* Top Section: Title & Mission Description (Split) */}
          <div className="grid grid-cols-1 lg:grid-cols-12 border-b border-white/15">
            <div className="lg:col-span-6 p-8 sm:p-12 border-b lg:border-b-0 lg:border-r border-white/15 flex flex-col justify-between">
              <div>
                <div className="flex items-center gap-2 mb-4">
                  <span className="w-1.5 h-1.5 rounded-full bg-orange-500 animate-pulse" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em] text-white/50">
                    ECOSYSTEM SPECIFICATION
                  </span>
                </div>
                <h2 className="text-3xl sm:text-4xl font-black uppercase tracking-tight text-white font-mono">
                  {title}
                </h2>
              </div>
              <div className="mt-8 font-mono text-[10px] tracking-widest text-white/40 border border-white/10 px-2.5 py-1 rounded inline-block w-fit">
                [ {badge} ]
              </div>
            </div>

            <div className="lg:col-span-6 p-8 sm:p-12 flex items-center bg-white/[0.01]">
              <p className="font-mono text-xs sm:text-sm text-white/70 leading-relaxed font-light">
                {description}
              </p>
            </div>
          </div>

          {/* 4-Product Timeline Row (Image 1 Style) */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 border-b border-white/15 divide-y sm:divide-y-0 sm:divide-x divide-white/15">
            {products.map((p, idx) => (
              <div key={idx} className="p-6 sm:p-8 flex flex-col justify-between bg-[#08080B] hover:bg-[#0E0F14] transition-colors group">
                <span className="font-mono text-[10px] text-white/40 mb-6 uppercase tracking-wider">
                  {p.quarter}
                </span>

                <div className="my-6">
                  {/* Styled Monospace Logo Badge */}
                  <div className={`w-12 h-12 rounded-xl border flex items-center justify-center font-mono font-bold text-xs mb-4 ${p.accent}`}>
                    {p.iconText}
                  </div>
                  <h3 className="text-base font-bold text-white group-hover:text-orange-400 transition-colors">
                    {p.name}
                  </h3>
                  <span className="font-mono text-[10px] text-white/50 uppercase tracking-wider block mt-1">
                    {p.role}
                  </span>
                </div>

                <div className="pt-4 border-t border-white/5 text-[9px] font-mono text-white/30 uppercase">
                  VERIFIED RUNTIME
                </div>
              </div>
            ))}
          </div>

          {/* Color Palette Ribbon */}
          <div className="p-6 sm:p-8 border-b border-white/15 bg-[#060608]">
            <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-white/40 block mb-4">
              DESIGN SYSTEM COLOR TOKENS
            </span>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
              {colorSwatches.map((swatch, idx) => (
                <div key={idx} className="rounded-xl border border-white/10 bg-[#0C0D12] p-2.5 flex flex-col justify-between">
                  <div className={`w-full h-8 rounded-lg mb-2 ${swatch.bg}`} />
                  <div className="flex items-center justify-between text-[10px] font-mono">
                    <span className="font-bold text-white/80">{swatch.hex}</span>
                    <span className="text-[8px] text-white/40 uppercase truncate ml-1">{swatch.label}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Giant Stylized Wireframe Typography Section */}
          <div className="p-8 sm:p-14 border-b border-white/15 bg-gradient-to-b from-[#0A0B10] to-[#060608] flex items-center justify-center overflow-hidden relative">
            {/* Massive Stylized Vector Outlines */}
            <div className="select-none font-black text-6xl sm:text-8xl lg:text-9xl tracking-tighter text-transparent uppercase text-center"
                 style={{
                   WebkitTextStroke: '1px rgba(255, 122, 26, 0.4)',
                   letterSpacing: '0.05em',
                 }}>
              PEGASUS
            </div>
          </div>

          {/* Typeface Pairings & Pixel Art Footer */}
          <div className="grid grid-cols-1 lg:grid-cols-12 divide-y lg:divide-y-0 lg:divide-x divide-white/15">
            {/* Display Typeface */}
            <div className="lg:col-span-5 p-8 sm:p-10 bg-[#08080B]">
              <span className="font-mono text-[10px] uppercase tracking-wider text-white/40 block mb-3">
                PRIMARY DISPLAY TYPEFACE
              </span>
              <div className="text-2xl sm:text-3xl font-black uppercase tracking-tight text-white leading-tight font-mono">
                BACKING TOMORROW
              </div>
              <div className="text-xl sm:text-2xl font-bold uppercase tracking-tight text-white/70 mt-1 font-mono">
                DETERMINISTIC SCALE
              </div>
              <span className="text-[10px] font-mono text-white/30 uppercase block mt-4">
                GEIST / ROBOTO MONO DISPLAY
              </span>
            </div>

            {/* Monospace Body Typeface */}
            <div className="lg:col-span-4 p-8 sm:p-10 bg-[#08080B]">
              <span className="font-mono text-[10px] uppercase tracking-wider text-white/40 block mb-3">
                MONOSPACE BODY &amp; CAPTIONS
              </span>
              <p className="font-mono text-sm sm:text-base text-white/90 leading-relaxed font-semibold">
                Growth Funding &amp; End-to-End Orchestration Support.
              </p>
              <span className="text-[10px] font-mono text-white/30 uppercase block mt-4">
                TABULAR NUMERALS &amp; MICRO-LABELS
              </span>
            </div>

            {/* Pixel Glyph & Narrative */}
            <div className="lg:col-span-3 p-8 sm:p-10 bg-[#060608] flex flex-col justify-between">
              <div className="flex items-center gap-3 mb-4 font-mono text-xl text-orange-400">
                <span>[◉_◉]</span>
                <span>➔</span>
              </div>
              <p className="font-mono text-[11px] text-white/60 leading-relaxed font-light">
                Every component adheres to crisp 1px hairline borders, tabular monospace arithmetic, and zero generic SaaS gray templates.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
