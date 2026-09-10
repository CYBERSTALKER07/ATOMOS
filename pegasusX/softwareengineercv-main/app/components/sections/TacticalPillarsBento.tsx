'use client';

import React from 'react';
import Link from 'next/link';
import {
  Database,
  Cpu,
  Shield,
  Network,
  Layers,
  Lock,
  GitBranch,
  Zap,
  Truck,
  Workflow,
  ArrowRight,
  CheckCircle2,
  Activity,
  FileCheck,
  Terminal,
} from 'lucide-react';
import type { TacticalPillarsConfig } from './sectionConfigs';

const ICON_MAP = {
  database: Database,
  cpu: Cpu,
  shield: Shield,
  network: Network,
  layers: Layers,
  lock: Lock,
  'git-branch': GitBranch,
  zap: Zap,
  truck: Truck,
  workflow: Workflow,
};

type TacticalPillarsBentoProps = {
  config: TacticalPillarsConfig;
  className?: string;
};

export default function TacticalPillarsBento({ config, className = '' }: TacticalPillarsBentoProps) {
  return (
    <section className={`w-full py-16 sm:py-24 text-white ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Main Obsidian Bento Frame */}
        <div className="relative rounded-3xl border border-white/15 bg-[#09090B] p-6 sm:p-10 lg:p-14 overflow-hidden shadow-2xl">
          {/* Subtle Tactical Ambient Grid & Glow */}
          <div
            className="absolute inset-0 pointer-events-none opacity-[0.03]"
            style={{
              backgroundImage: `radial-gradient(circle at 1px 1px, white 1px, transparent 0)`,
              backgroundSize: '24px 24px',
            }}
          />
          <div className="absolute -top-32 -right-32 w-96 h-96 bg-blue-600/10 rounded-full blur-3xl pointer-events-none" />
          <div className="absolute -bottom-32 -left-32 w-96 h-96 bg-emerald-600/10 rounded-full blur-3xl pointer-events-none" />

          {/* Top Section: Split Layout (Headline vs 2x2 Bento) */}
          <div className="relative z-10 grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-14 items-start">
            {/* Left Column (5 cols) */}
            <div className="lg:col-span-5 flex flex-col justify-between h-full">
              <div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-[11px] font-mono uppercase tracking-[0.2em] text-emerald-400 mb-6">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                  {config.eyebrow}
                </div>

                <h2 className="text-2xl sm:text-3xl lg:text-4xl font-black uppercase tracking-tight text-white leading-tight">
                  {config.title}
                </h2>

                <p className="mt-5 text-sm sm:text-base text-white/65 leading-relaxed font-light">
                  {config.description}
                </p>
              </div>

              {/* Technical Spec Summary Footer */}
              <div className="mt-8 pt-6 border-t border-white/10">
                <div className="flex items-center gap-2 text-[11px] font-mono text-white/45 tracking-wider uppercase">
                  <Terminal className="w-3.5 h-3.5 text-emerald-400" />
                  <span>{config.specSummary}</span>
                </div>
              </div>
            </div>

            {/* Right Column (7 cols): 2x2 Bento Grid */}
            <div className="lg:col-span-7">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-px bg-white/10 rounded-2xl overflow-hidden border border-white/10">
                {config.items.map((item, idx) => {
                  const Icon = ICON_MAP[item.iconName] || Cpu;
                  return (
                    <div
                      key={idx}
                      className="bg-[#09090B] p-6 sm:p-7 flex flex-col justify-between hover:bg-[#111116] transition-colors group relative"
                    >
                      {/* Corner Tech Accent */}
                      <div className="absolute top-2 right-2 font-mono text-[9px] text-white/20 uppercase">
                        #{`0${idx + 1}`}
                      </div>

                      <div>
                        {/* Double-Framed Icon Container from Image 1 */}
                        <div className="w-11 h-11 rounded-xl border border-white/20 p-1 flex items-center justify-center bg-white/[0.02] group-hover:border-emerald-400/50 transition-colors">
                          <div className="w-full h-full rounded-lg border border-white/10 flex items-center justify-center bg-white/[0.04] text-white group-hover:text-emerald-400 transition-colors">
                            <Icon className="w-5 h-5" />
                          </div>
                        </div>

                        <h3 className="text-base sm:text-lg font-bold text-white mt-5 mb-2 group-hover:text-emerald-300 transition-colors">
                          {item.title}
                        </h3>

                        <p className="text-xs sm:text-sm text-white/60 leading-relaxed font-light">
                          {item.description}
                        </p>
                      </div>

                      <div className="mt-6 flex items-center gap-2 text-[10px] font-mono text-white/35 uppercase tracking-wider">
                        <span className="w-1 h-1 rounded-full bg-emerald-400/70" />
                        <span>VERIFIED SPEC</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Bottom Section: Protocol Split Block (Image 1 Bottom Half) */}
          <div className="relative z-10 mt-12 sm:mt-16 pt-10 sm:pt-12 border-t border-white/10">
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12 items-center">
              {/* Left Protocol Explanation */}
              <div className="lg:col-span-5">
                <span className="font-mono text-[10px] uppercase tracking-[0.22em] text-cyan-400 block mb-2">
                  {config.protocolBlock.tag}
                </span>
                <h3 className="text-xl sm:text-2xl font-bold uppercase tracking-tight text-white mb-3">
                  {config.protocolBlock.title}
                </h3>
                <p className="text-xs sm:text-sm text-white/60 leading-relaxed font-light mb-5">
                  {config.protocolBlock.description}
                </p>
                <Link
                  href={config.protocolBlock.whitepaperHref}
                  className="inline-flex items-center gap-2 text-xs font-mono font-medium text-emerald-400 hover:text-emerald-300 transition-colors group"
                >
                  <span>{config.protocolBlock.whitepaperText}</span>
                  <ArrowRight className="w-3.5 h-3.5 transition-transform group-hover:translate-x-1" />
                </Link>
              </div>

              {/* Right Cryptographic / Connected Circuit Matrix */}
              <div className="lg:col-span-7">
                <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-5 relative overflow-hidden">
                  {/* Subtle Background SVG Circuit Lines */}
                  <svg
                    className="absolute inset-0 w-full h-full pointer-events-none opacity-20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <line x1="16%" y1="30%" x2="50%" y2="30%" stroke="#10B981" strokeWidth="1" strokeDasharray="4 4" />
                    <line x1="50%" y1="30%" x2="84%" y2="30%" stroke="#3B82F6" strokeWidth="1" strokeDasharray="4 4" />
                    <line x1="50%" y1="30%" x2="50%" y2="70%" stroke="#06B6D4" strokeWidth="1" />
                    <line x1="16%" y1="70%" x2="50%" y2="70%" stroke="#10B981" strokeWidth="1" strokeDasharray="4 4" />
                    <line x1="50%" y1="70%" x2="84%" y2="70%" stroke="#F59E0B" strokeWidth="1" strokeDasharray="4 4" />
                  </svg>

                  <div className="relative z-10 grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {config.protocolBlock.nodes.map((node) => (
                      <div
                        key={node.id}
                        className="rounded-xl border border-white/10 bg-[#0C0D12] p-3.5 flex flex-col justify-between hover:border-emerald-500/40 transition-all group"
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-mono text-[10px] font-bold text-white/90 group-hover:text-emerald-400 transition-colors">
                            {node.label}
                          </span>
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              node.status === 'active' || node.status === 'verified'
                                ? 'bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]'
                                : 'bg-cyan-400 shadow-[0_0_6px_rgba(6,182,212,0.8)]'
                            }`}
                          />
                        </div>
                        <span className="text-[11px] text-white/50 font-light truncate">
                          {node.sublabel}
                        </span>
                        <div className="mt-2 pt-2 border-t border-white/5 flex items-center justify-between text-[9px] font-mono text-white/30">
                          <span>NODE::{node.id}</span>
                          <span className="text-emerald-400/80 uppercase">{node.status}</span>
                        </div>
                      </div>
                    ))}
                  </div>

                  {/* Micro Live Metrics Strip */}
                  <div className="mt-4 pt-3 border-t border-white/10 flex flex-wrap items-center justify-between text-[10px] font-mono text-white/40">
                    <div className="flex items-center gap-2">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                      <span>STREAM: ATOMIC_OUTBOX</span>
                    </div>
                    <span>CONSISTENCY: 100% LINEARIZABLE</span>
                    <span>RTT: 0.04ms</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
