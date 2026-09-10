'use client';

import React from 'react';
import {
  Globe2,
  Server,
  TrendingUp,
  ShieldCheck,
  Radio,
  Zap,
  Layers,
  ArrowRight,
} from 'lucide-react';

export type TickerItem = {
  id: string;
  name: string;
  symbol: string;
  value: string;
  metric: string;
  isPositive?: boolean;
};

export type DiamondMatrixTickerProps = {
  eyebrow?: string;
  title?: string;
  description?: string;
  items?: TickerItem[];
  className?: string;
};

export default function DiamondMatrixTickerSection({
  eyebrow = 'GLOBAL CORRIDOR TELEMETRY',
  title = 'Planetary Settlement & Live Corridor Rates',
  description = 'Real-time telemetry, freight rates, and cross-border currency conversions calculated across all sovereign regional cells and transit nodes.',
  items = [
    { id: '1', name: 'Tashkent Sovereign', symbol: 'TAS-IX', value: '2.14ms', metric: '$3,112,170 Vol' },
    { id: '2', name: 'Samarkand Depot', symbol: 'M39-SAM', value: '7.24ms', metric: '$25,143,587 Vol' },
    { id: '3', name: 'Fergana Valley', symbol: 'SILK-RD', value: '8.45ms', metric: '$18,901,230 Vol' },
    { id: '4', name: 'Frankfurt Core', symbol: 'CELL-EU', value: '$2,965.61', metric: '$356,225,807 Vol' },
    { id: '5', name: 'Dubai Transit Hub', symbol: 'DXB-AIR', value: '$145.29', metric: '$65,169,112 Vol' },
    { id: '6', name: 'Virginia Ingress', symbol: 'CELL-US', value: '$33.07', metric: '$12,616,497 Vol' },
  ],
  className = '',
}: DiamondMatrixTickerProps) {
  const leftItems = items.slice(0, 3);
  const rightItems = items.slice(3, 6);

  // Diamond mosaic coordinates (arranged in a symmetrical diamond shape like Image 3)
  // Rows:
  // Row 1: [0]
  // Row 2: [-1, 1]
  // Row 3: [-2, 0, 2] (or open center ring)
  // Row 4: [-1, 1]
  // Row 5: [0]
  const diamondTiles = [
    { row: 0, col: 0 },
    { row: 1, col: -1 },
    { row: 1, col: 1 },
    { row: 2, col: -2 },
    { row: 2, col: 2 },
    { row: 3, col: -1 },
    { row: 3, col: 1 },
    { row: 4, col: 0 },
  ];

  return (
    <section className={`w-full py-16 sm:py-24 text-white overflow-hidden ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header Block */}
        <div className="text-center max-w-3xl mx-auto mb-14">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-pink-500/20 bg-pink-500/10 text-[11px] font-mono uppercase tracking-[0.2em] text-pink-400 mb-4">
            <Radio className="w-3.5 h-3.5 animate-pulse" />
            <span>{eyebrow}</span>
          </div>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black uppercase tracking-tight text-white leading-tight">
            {title}
          </h2>
          <p className="mt-4 text-sm sm:text-base text-white/60 leading-relaxed font-light">
            {description}
          </p>
        </div>

        {/* Central Stage: Diamond Matrix Core with Floating Ticker Cards */}
        <div className="relative rounded-3xl border border-white/10 bg-[#08080C] p-6 sm:p-12 lg:p-16 shadow-2xl overflow-hidden min-h-[560px] flex items-center justify-center">
          {/* Subtle Background Radial Atmosphere */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div className="w-[450px] h-[450px] bg-pink-600/10 rounded-full blur-3xl" />
          </div>

          {/* Hand-Drawn Style Annotations (Image 3 Style) */}
          <div className="absolute top-8 left-10 font-mono text-2xl text-white/20 select-none tracking-widest hidden md:block">
            * * *
          </div>
          <div className="absolute top-8 right-12 font-mono text-2xl text-white/20 select-none hidden md:block">
            ₿
          </div>
          <div className="absolute bottom-10 left-12 font-mono text-xs uppercase tracking-widest text-white/25 border border-white/10 px-3 py-1 rounded -rotate-6 select-none hidden md:block">
            SAFE PARK
          </div>
          <div className="absolute bottom-10 right-14 font-mono text-xl text-white/25 select-none hidden md:block">
            ⟳
          </div>

          {/* Grid Layout: Left Tickers (3) — Center Diamond Matrix — Right Tickers (3) */}
          <div className="relative z-10 w-full grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
            {/* Left 3 Ticker Cards */}
            <div className="lg:col-span-4 space-y-4">
              {leftItems.map((item) => (
                <div
                  key={item.id}
                  className="rounded-2xl border border-white/15 bg-[#0D0E14]/90 backdrop-blur-md p-4 sm:p-5 flex items-center justify-between hover:border-pink-500/40 hover:bg-[#12131C] transition-all shadow-xl group"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-xl bg-white/[0.05] border border-white/10 flex items-center justify-center text-white/80 group-hover:text-pink-400 transition-colors">
                      <Server className="w-4 h-4" />
                    </div>
                    <div>
                      <div className="text-sm font-bold text-white group-hover:text-pink-300 transition-colors">
                        {item.name}
                      </div>
                      <div className="font-mono text-[10px] text-white/40 uppercase">
                        {item.symbol}
                      </div>
                    </div>
                  </div>

                  <div className="text-right">
                    <div className="font-mono text-sm sm:text-base font-bold text-white tabular-nums">
                      {item.value}
                    </div>
                    <div className="font-mono text-[10px] text-white/40 tabular-nums">
                      {item.metric}
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Center Neon Pink Diamond Matrix (Image 3) */}
            <div className="lg:col-span-4 flex items-center justify-center my-6 lg:my-0">
              <div className="relative w-48 h-64 flex items-center justify-center">
                {/* SVG Connecting Diamond Perimeter */}
                <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 200 260">
                  <polygon
                    points="100,20 180,130 100,240 20,130"
                    fill="none"
                    stroke="#EC4899"
                    strokeWidth="1"
                    strokeDasharray="4 4"
                    opacity="0.3"
                  />
                </svg>

                {/* Arranged Neon Magenta Tiles */}
                <div className="relative w-40 h-56 flex flex-col items-center justify-between">
                  {/* Row 0: 1 tile at top */}
                  <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />

                  {/* Row 1: 2 tiles */}
                  <div className="flex justify-between w-28">
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                  </div>

                  {/* Row 2: 2 tiles spread wide */}
                  <div className="flex justify-between w-40">
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                  </div>

                  {/* Row 3: 2 tiles */}
                  <div className="flex justify-between w-28">
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                    <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                  </div>

                  {/* Row 4: 1 tile at bottom */}
                  <div className="w-9 h-9 rounded-lg bg-[#FF45B5] shadow-[0_0_20px_rgba(255,69,181,0.65)] border border-white/30" />
                </div>
              </div>
            </div>

            {/* Right 3 Ticker Cards */}
            <div className="lg:col-span-4 space-y-4">
              {rightItems.map((item) => (
                <div
                  key={item.id}
                  className="rounded-2xl border border-white/15 bg-[#0D0E14]/90 backdrop-blur-md p-4 sm:p-5 flex items-center justify-between hover:border-pink-500/40 hover:bg-[#12131C] transition-all shadow-xl group"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-xl bg-white/[0.05] border border-white/10 flex items-center justify-center text-white/80 group-hover:text-pink-400 transition-colors">
                      <Globe2 className="w-4 h-4" />
                    </div>
                    <div>
                      <div className="text-sm font-bold text-white group-hover:text-pink-300 transition-colors">
                        {item.name}
                      </div>
                      <div className="font-mono text-[10px] text-white/40 uppercase">
                        {item.symbol}
                      </div>
                    </div>
                  </div>

                  <div className="text-right">
                    <div className="font-mono text-sm sm:text-base font-bold text-white tabular-nums">
                      {item.value}
                    </div>
                    <div className="font-mono text-[10px] text-white/40 tabular-nums">
                      {item.metric}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Bottom Telemetry Ticker Strip */}
          <div className="absolute bottom-3 left-0 right-0 px-8 flex items-center justify-between text-[10px] font-mono text-white/30">
            <span>SPANNER REGIONAL CELLS: 6 ONLINE</span>
            <span className="hidden sm:inline">ZERO-DRIFT CONSENSUS</span>
            <span>SYNC: 100% LIVE</span>
          </div>
        </div>
      </div>
    </section>
  );
}
