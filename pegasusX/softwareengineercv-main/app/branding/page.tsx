'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import PegasusSciFiLogo from '../components/visuals/PegasusSciFiLogo';
import PartnerBrandStrip from '../components/visuals/PartnerBrandStrip';

export default function BrandingShowcasePage() {
  const [retracted, setRetracted] = useState(false);
  const [activeTab, setActiveTab] = useState<'all' | 'solid' | 'neon' | 'hero'>('all');

  return (
    <div className="min-h-screen bg-[#06070a] text-zinc-100 font-sans selection:bg-cyan-500/20 selection:text-cyan-200">
      {/* Top Banner Navigation */}
      <header className="sticky top-0 z-50 border-b border-white/[0.08] bg-[#06070a]/90 backdrop-blur-xl px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/" className="text-xs uppercase tracking-widest text-zinc-400 hover:text-white transition-colors">
            ← Return to PegasusX
          </Link>
          <span className="text-zinc-600">/</span>
          <span className="text-xs uppercase font-mono tracking-widest text-cyan-400 font-medium">
            Sci-Fi Typography & Wordmark System
          </span>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab('all')}
            className={`px-3 py-1 text-xs rounded transition-colors ${
              activeTab === 'all' ? 'bg-white text-black font-medium' : 'text-zinc-400 hover:text-white'
            }`}
          >
            All Displays
          </button>
          <button
            onClick={() => setActiveTab('solid')}
            className={`px-3 py-1 text-xs rounded transition-colors ${
              activeTab === 'solid' ? 'bg-white text-black font-medium' : 'text-zinc-400 hover:text-white'
            }`}
          >
            Solid Vector
          </button>
          <button
            onClick={() => setActiveTab('neon')}
            className={`px-3 py-1 text-xs rounded transition-colors ${
              activeTab === 'neon' ? 'bg-white text-black font-medium' : 'text-zinc-400 hover:text-white'
            }`}
          >
            Cyberpunk Neon
          </button>
          <button
            onClick={() => setActiveTab('hero')}
            className={`px-3 py-1 text-xs rounded transition-colors ${
              activeTab === 'hero' ? 'bg-white text-black font-medium' : 'text-zinc-400 hover:text-white'
            }`}
          >
            Hero Layout
          </button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-12 space-y-16">
        {/* Intro Header */}
        <div className="space-y-4 max-w-3xl">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-cyan-500/20 bg-cyan-500/10 text-cyan-300 text-xs font-mono tracking-wide">
            <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse" />
            TERAFAB-MATCHED GEOMETRIC BRAND SYSTEM
          </div>
          <h1 className="text-4xl sm:text-5xl font-light tracking-tight text-white">
            Pegasus Brandmark & Typography
          </h1>
          <p className="text-base text-zinc-400 font-light leading-relaxed">
            Recreated with mathematical fidelity from the <span className="text-white font-normal">Terafab</span> reference.
            Featuring the signature chevron apex &apos;A&apos; without crossbar, uniform 7.6-unit squircle strokes, and aerospace letterform geometry.
          </p>
        </div>

        {/* Section 1: Solid Wordmark Display */}
        {(activeTab === 'all' || activeTab === 'solid') && (
          <section className="space-y-6">
            <div className="flex items-center justify-between border-b border-white/[0.08] pb-4">
              <div>
                <h2 className="text-lg font-medium text-white">1. Master Display Wordmark (Solid White)</h2>
                <p className="text-xs text-zinc-400 font-mono tracking-wide mt-1">
                  CORRESPONDS TO TOP RED CIRCLE IN REFERENCE SCREENSHOT
                </p>
              </div>
              <span className="text-xs font-mono text-zinc-500">Vector SVG • Scalable • Crisp</span>
            </div>

            <div className="relative rounded-2xl border border-white/[0.08] bg-[#090b10] p-12 sm:p-20 overflow-hidden flex flex-col items-center justify-center">
              {/* Subtle background grid */}
              <div 
                className="absolute inset-0 opacity-[0.03] pointer-events-none"
                style={{
                  backgroundImage: `linear-gradient(#fff 1px, transparent 1px), linear-gradient(90deg, #fff 1px, transparent 1px)`,
                  backgroundSize: '32px 32px'
                }}
              />
              
              <div className="w-full max-w-4xl py-6 flex items-center justify-center">
                <PegasusSciFiLogo variant="solid" color="#ffffff" className="w-full max-w-3xl drop-shadow-[0_2px_24px_rgba(255,255,255,0.18)]" />
              </div>

              <div className="mt-8 flex flex-wrap items-center justify-center gap-6 text-xs font-mono text-zinc-500">
                <span>WIDTH: 772.03 UNITS</span>
                <span>•</span>
                <span>HEIGHT: 55.51 UNITS</span>
                <span>•</span>
                <span>STROKE: 7.60 UNITS</span>
                <span>•</span>
                <span>SQUIRCLE RADIUS: 17.48</span>
              </div>
            </div>
          </section>
        )}

        {/* Section 2: Cyberpunk Neon Luminescence Display */}
        {(activeTab === 'all' || activeTab === 'neon') && (
          <section className="space-y-6">
            <div className="flex items-center justify-between border-b border-white/[0.08] pb-4">
              <div>
                <h2 className="text-lg font-medium text-white">2. High-Lumen Cyberpunk Neon Luminescence</h2>
                <p className="text-xs text-zinc-400 font-mono tracking-wide mt-1">
                  MATCHES THE GLOWING BRIDGE EMBLEM OVER THE RACETRACK IN SCREENSHOT
                </p>
              </div>
              <span className="text-xs font-mono text-cyan-400">Atmospheric Glow • Multi-Tier Blur</span>
            </div>

            <div className="relative rounded-2xl border border-cyan-500/20 bg-[#07090e] p-12 sm:p-20 overflow-hidden flex flex-col items-center justify-center">
              <div className="absolute inset-0 bg-radial-gradient from-cyan-950/20 via-transparent to-transparent pointer-events-none" />
              
              <div className="w-full max-w-4xl py-6 flex items-center justify-center">
                <PegasusSciFiLogo variant="neon" glowColor="#38bdf8" className="w-full max-w-3xl" />
              </div>

              <p className="mt-8 text-xs font-mono text-cyan-400/70 tracking-widest uppercase">
                Emissive Photometric Luminance: #38bdf8 / Cyan 400
              </p>
            </div>
          </section>
        )}

        {/* Section 3: Interactive Retractable Nav Logo */}
        {(activeTab === 'all' || activeTab === 'solid') && (
          <section className="space-y-6">
            <div className="flex items-center justify-between border-b border-white/[0.08] pb-4">
              <div>
                <h2 className="text-lg font-medium text-white">3. Retractable Header Logo (Interactive Scroll Simulation)</h2>
                <p className="text-xs text-zinc-400 font-mono tracking-wide mt-1">
                  REPRODUCES TERAFAB.AI INTERACTION: FULL WORDMARK AT REST → COLLAPSES TO &apos;P&apos; MONOGRAM ON SCROLL
                </p>
              </div>
              <button
                onClick={() => setRetracted(!retracted)}
                className="px-4 py-1.5 rounded-lg border border-white/20 bg-white/10 text-xs font-mono hover:bg-white/20 transition-all text-white"
              >
                {retracted ? '↺ Expand to PEGASUS' : '↳ Collapse to [P]'}
              </button>
            </div>

            <div className="rounded-2xl border border-white/[0.08] bg-[#090b10] p-8 space-y-6">
              <div className="flex items-center justify-between p-4 rounded-xl bg-black/60 border border-white/10 backdrop-blur-md">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg transition-all duration-300 ${retracted ? 'bg-white text-black' : 'bg-transparent text-white'}`}>
                    <PegasusSciFiLogo
                      variant="retractable"
                      retracted={retracted}
                      color={retracted ? '#0a0a0b' : '#ffffff'}
                      height="22px"
                    />
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <span className="text-xs text-zinc-400 font-medium">Watch Demo</span>
                  <span className="px-3 py-1 rounded bg-white text-black text-xs font-medium">
                    Join Network
                  </span>
                </div>
              </div>

              <p className="text-xs text-zinc-500 font-mono">
                Click the toggle button above to simulate how the logo contracts seamlessly when the user scrolls down the page.
              </p>
            </div>
          </section>
        )}

        {/* Section 4: Full Hero Layout Composition */}
        {(activeTab === 'all' || activeTab === 'hero') && (
          <section className="space-y-6">
            <div className="flex items-center justify-between border-b border-white/[0.08] pb-4">
              <div>
                <h2 className="text-lg font-medium text-white">4. Complete Hero Layout & Partner Rail</h2>
                <p className="text-xs text-zinc-400 font-mono tracking-wide mt-1">
                  MATCHES THE COMPOSITION: TITLE + SUBTITLE + TWIN BUTTONS + BOTTOM PARTNER MARKS
                </p>
              </div>
              <span className="text-xs font-mono text-zinc-500">Neo-Grotesque Display</span>
            </div>

            <div className="rounded-2xl border border-white/[0.08] bg-[#07080c] overflow-hidden relative min-h-[500px] flex flex-col justify-between p-8 sm:p-14">
              {/* Header inside mockup */}
              <div className="flex items-center justify-between pb-8">
                <PegasusSciFiLogo variant="solid" color="#ffffff" height="20px" />
                <div className="flex items-center gap-3">
                  <span className="px-4 py-2 rounded-md bg-white/[0.06] border border-white/10 text-xs font-medium text-white hover:bg-white/10 cursor-pointer">
                    Watch Demo
                  </span>
                  <span className="px-4 py-2 rounded-md bg-white text-black text-xs font-medium hover:bg-white/90 cursor-pointer">
                    Join Us
                  </span>
                </div>
              </div>

              {/* Center-Left Content */}
              <div className="max-w-xl space-y-6 my-auto py-12">
                <h1 className="text-5xl sm:text-6xl font-normal tracking-[-0.035em] text-white leading-[1.05]">
                  Pegasus
                </h1>
                <p className="text-xl sm:text-2xl font-light text-zinc-300 leading-snug tracking-[-0.015em]">
                  The most resilient autonomous freight & supply-chain operating system ever.
                </p>

                {/* Twin Action Buttons */}
                <div className="flex flex-wrap items-center gap-4 pt-2">
                  <button className="inline-flex items-center gap-2 px-6 py-3 rounded-md bg-white text-black text-sm font-medium hover:bg-zinc-200 transition-colors">
                    <span>Watch Now</span>
                    <span className="text-base">›</span>
                  </button>
                  <button className="inline-flex items-center gap-2 px-6 py-3 rounded-md bg-white/[0.08] border border-white/15 text-white text-sm font-medium hover:bg-white/15 backdrop-blur-sm transition-colors">
                    <span>Explore Architecture</span>
                    <span className="text-base">›</span>
                  </button>
                </div>
              </div>

              {/* Bottom Partner Strip (Matches bottom circle in screenshot) */}
              <div className="pt-8">
                <PartnerBrandStrip
                  showDivider={true}
                  label="PARTNER & ENTERPRISE INFRASTRUCTURE ECOSYSTEM"
                />
              </div>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}
