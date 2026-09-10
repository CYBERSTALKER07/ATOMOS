'use client';

import React from 'react';
import { Activity, ShieldCheck, Zap } from 'lucide-react';

export type TacticalInstrumentsProps = {
  eyebrow?: string;
  title?: string;
  description?: string;
  instruments?: [
    {
      title: string;
      description: string;
      metricLabel?: string;
    },
    {
      title: string;
      description: string;
      cipherLines?: string[];
    },
    {
      title: string;
      description: string;
      metricLabel?: string;
    }
  ];
  className?: string;
};

export default function TacticalInstrumentsSection({
  eyebrow = 'THE SOLUTION',
  title = 'The Deterministic Algorithm for Logistics',
  description = 'Combining sub-second vehicle telemetry, mathematical CVRP route optimization, and atomic double-entry ledger reconciliation to maximize fleet throughput and eliminate operational friction.',
  instruments = [
    {
      title: 'Blazingly Fast Connection',
      description:
        'Sub-10 millisecond event fanout via Redis Streams and direct TAS-IX fiber peering ensures zero delay between driver actions and warehouse dispatch boards.',
      metricLabel: '3.4ms RTT',
    },
    {
      title: 'Everlasting Ledger Audit',
      description:
        'Cryptographically signed append-only event streams protect every stock allocation, route diversion, and cash payment forever with zero data drift.',
      cipherLines: [
        '0x7F...spanner.ReadWriteTx',
        'tx.BufferWrite(OutboxEvent)',
        'ledger.Debit(tiyin, minorUnits)',
        'ACID_INVARIANT: BALANCED',
      ],
    },
    {
      title: 'Adaptive Growth Framework',
      description:
        'Google OR-Tools metaheuristics dynamically absorb sudden demand spikes and driver absences, recalculating multi-stop routes in under 3 seconds.',
      metricLabel: '+94.2% Capacity',
    },
  ],
  className = '',
}: TacticalInstrumentsProps) {
  return (
    <section className={`w-full py-16 sm:py-24 text-white overflow-hidden ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Frame with Corner Crosshairs (Image 2 style) */}
        <div className="relative rounded-3xl border border-white/10 bg-[#070709] p-8 sm:p-12 lg:p-14 shadow-2xl">
          {/* Top-Left and Bottom-Right Tactical Crosshairs */}
          <div className="absolute top-4 left-4 font-mono text-xs text-white/30 select-none">+</div>
          <div className="absolute top-4 right-4 font-mono text-xs text-white/30 select-none">+</div>
          <div className="absolute bottom-4 left-4 font-mono text-xs text-white/30 select-none">+</div>
          <div className="absolute bottom-4 right-4 font-mono text-xs text-white/30 select-none">+</div>

          {/* Header Block */}
          <div className="max-w-3xl mb-14">
            <span className="font-mono text-[11px] uppercase tracking-[0.25em] text-pink-400 block mb-3 font-semibold">
              {eyebrow}
            </span>
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black uppercase tracking-tight text-white leading-tight">
              {title}
            </h2>
            <p className="mt-5 text-sm sm:text-base text-white/60 leading-relaxed font-light">
              {description}
            </p>
          </div>

          {/* 3 Tactical Instruments Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 lg:gap-12 pt-8 border-t border-white/10">
            {/* Instrument 1: Speedometer Dial Gauge */}
            <div className="flex flex-col justify-between">
              <div>
                {/* SVG Semi-Circular Speedometer Gauge */}
                <div className="relative w-full h-44 flex items-center justify-center mb-6">
                  <svg className="w-48 h-36" viewBox="0 0 200 130">
                    <defs>
                      <linearGradient id="gaugeArc" x1="0%" y1="0%" x2="100%" y2="0%">
                        <stop offset="0%" stopColor="#3B82F6" />
                        <stop offset="60%" stopColor="#EC4899" />
                        <stop offset="100%" stopColor="#F43F5E" />
                      </linearGradient>
                      <filter id="gaugeGlow">
                        <feGaussianBlur stdDeviation="3" result="glow" />
                        <feMerge>
                          <feMergeNode in="glow" />
                          <feMergeNode in="SourceGraphic" />
                        </feMerge>
                      </filter>
                    </defs>

                    {/* Background Arc Ticks */}
                    {Array.from({ length: 21 }).map((_, i) => {
                      const angle = 180 + (i * 180) / 20;
                      const rad = (angle * Math.PI) / 180;
                      const r1 = 70;
                      const r2 = i % 5 === 0 ? 84 : 77;
                      const x1 = 100 + r1 * Math.cos(rad);
                      const y1 = 110 + r1 * Math.sin(rad);
                      const x2 = 100 + r2 * Math.cos(rad);
                      const y2 = 110 + r2 * Math.sin(rad);
                      const isHot = i > 12;
                      return (
                        <line
                          key={i}
                          x1={x1}
                          y1={y1}
                          x2={x2}
                          y2={y2}
                          stroke={isHot ? '#EC4899' : '#334155'}
                          strokeWidth={i % 5 === 0 ? 2 : 1}
                          strokeLinecap="round"
                        />
                      );
                    })}

                    {/* Speedometer Main Needle (Pointing to low latency ~75% angle) */}
                    <g filter="url(#gaugeGlow)">
                      <line
                        x1="100"
                        y1="110"
                        x2="155"
                        y2="60"
                        stroke="#F43F5E"
                        strokeWidth="3"
                        strokeLinecap="round"
                      />
                      {/* Pivot Center Knob */}
                      <circle cx="100" cy="110" r="10" fill="#FFFFFF" />
                      <circle cx="100" cy="110" r="14" fill="#F43F5E" opacity="0.4" />
                    </g>
                  </svg>

                  {/* Micro Readout Pill */}
                  <div className="absolute bottom-2 font-mono text-[10px] uppercase tracking-wider px-2 py-0.5 rounded bg-white/5 border border-white/10 text-pink-300">
                    {instruments[0].metricLabel || '3.4ms RTT'}
                  </div>
                </div>

                <h3 className="text-lg font-bold text-white mb-2">
                  {instruments[0].title}
                </h3>
                <p className="text-xs sm:text-sm text-white/60 leading-relaxed font-light">
                  {instruments[0].description}
                </p>
              </div>
            </div>

            {/* Instrument 2: Cryptographic Cipher Text Stream */}
            <div className="flex flex-col justify-between border-t md:border-t-0 md:border-l border-white/10 pt-8 md:pt-0 md:pl-8 lg:pl-12">
              <div>
                {/* Cipher Box */}
                <div className="relative w-full h-44 rounded-2xl border border-white/10 bg-[#0C0D12] p-4 font-mono text-[11px] leading-relaxed mb-6 overflow-hidden flex flex-col justify-center">
                  {/* Subtle Background Cipher Code Blur */}
                  <div className="absolute inset-0 p-3 opacity-25 filter blur-[0.5px] select-none text-[9px] text-white/40 overflow-hidden pointer-events-none">
                    0x7Fa8B941c99021884bA12eFc841
                    SpannerTransactionCommitHash::0019
                    LedgerDebitBalance::2958100UZS
                    OutboxEventFanoutRedisStreams
                    KafkaZeroLossMirrorMakerOK
                    TLS13_ECDHE_RSA_AES256_GCM
                  </div>

                  {/* Highlighting Foreground Code */}
                  <div className="relative z-10 space-y-1.5">
                    <div className="text-white/40 text-[10px]">
                      // CRYPTOGRAPHIC_INVARIANT
                    </div>
                    <div className="text-pink-400 font-bold">
                      &gt; spanner.ReadWriteTx()
                    </div>
                    <div className="text-emerald-400">
                      + tx.BufferWrite(OutboxEvent)
                    </div>
                    <div className="text-cyan-300">
                      + ledger.Debit(tiyin, exact)
                    </div>
                    <div className="text-white/70 text-[10px] pt-1 border-t border-white/5 flex items-center justify-between">
                      <span>STATUS: 100% ACID</span>
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]" />
                    </div>
                  </div>
                </div>

                <h3 className="text-lg font-bold text-white mb-2">
                  {instruments[1].title}
                </h3>
                <p className="text-xs sm:text-sm text-white/60 leading-relaxed font-light">
                  {instruments[1].description}
                </p>
              </div>
            </div>

            {/* Instrument 3: Adaptive Waveform / Area Chart */}
            <div className="flex flex-col justify-between border-t md:border-t-0 md:border-l border-white/10 pt-8 md:pt-0 md:pl-8 lg:pl-12">
              <div>
                {/* SVG Area Waveform with Peak Marker */}
                <div className="relative w-full h-44 flex items-center justify-center mb-6">
                  <svg className="w-full h-36" viewBox="0 0 240 120" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="waveGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                        <stop offset="0%" stopColor="#EC4899" stopOpacity="0.45" />
                        <stop offset="100%" stopColor="#EC4899" stopOpacity="0.0" />
                      </linearGradient>
                      <linearGradient id="waveStroke" x1="0%" y1="0%" x2="100%" y2="0%">
                        <stop offset="0%" stopColor="#F43F5E" />
                        <stop offset="70%" stopColor="#EC4899" />
                        <stop offset="100%" stopColor="#8B5CF6" />
                      </linearGradient>
                    </defs>

                    {/* Area Fill */}
                    <path
                      d="M 10 100 Q 40 90, 70 80 T 130 65 T 180 30 L 220 50 L 230 100 Z"
                      fill="url(#waveGradient)"
                    />

                    {/* Waveform Spline Line */}
                    <path
                      d="M 10 100 Q 40 90, 70 80 T 130 65 T 180 30 L 220 50"
                      fill="none"
                      stroke="url(#waveStroke)"
                      strokeWidth="2.5"
                      strokeLinecap="round"
                    />

                    {/* Peak Vertical Hairline Indicator */}
                    <line x1="180" y1="30" x2="180" y2="100" stroke="#FFFFFF" strokeWidth="1" strokeDasharray="3 3" opacity="0.6" />

                    {/* Apex Peak Marker Heart/Pin Node */}
                    <g transform="translate(174, 18)">
                      <circle cx="6" cy="6" r="8" fill="#EC4899" opacity="0.3" className="animate-ping" />
                      <circle cx="6" cy="6" r="4" fill="#FFFFFF" />
                    </g>
                  </svg>

                  {/* Top-right Peak Stat Badge */}
                  <div className="absolute top-2 right-2 font-mono text-[9px] uppercase tracking-wider px-2 py-0.5 rounded bg-pink-500/10 border border-pink-500/30 text-pink-300">
                    {instruments[2].metricLabel || 'CVRP OPTIMAL'}
                  </div>
                </div>

                <h3 className="text-lg font-bold text-white mb-2">
                  {instruments[2].title}
                </h3>
                <p className="text-xs sm:text-sm text-white/60 leading-relaxed font-light">
                  {instruments[2].description}
                </p>
              </div>
            </div>
          </div>

          {/* Bottom Micro Loader Accent (Image 2 style) */}
          <div className="mt-10 pt-6 border-t border-white/5 flex items-center justify-between text-[10px] font-mono text-white/40">
            <div className="flex items-center gap-2">
              <span className="w-1.5 h-1.5 rounded-full bg-pink-400" />
              <span>DETERMINISTIC KERNEL: ACTIVE</span>
            </div>
            {/* Spinning Loader Ring */}
            <div className="w-4 h-4 rounded-full border-2 border-pink-500/30 border-t-pink-400 animate-spin" />
          </div>
        </div>
      </div>
    </section>
  );
}
