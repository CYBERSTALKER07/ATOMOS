'use client';

import React, { useState } from 'react';
import {
  GitBranch,
  GitCommit,
  GitMerge,
  Clock,
  CheckCircle2,
  AlertTriangle,
  Layers,
  ArrowRight,
  ShieldCheck,
  Zap,
} from 'lucide-react';
import type { BranchingTimelineConfig, TimelineBranch } from './sectionConfigs';

type BranchingTimelineSectionProps = {
  config: BranchingTimelineConfig;
  className?: string;
};

export default function BranchingTimelineSection({
  config,
  className = '',
}: BranchingTimelineSectionProps) {
  const [activeBranchIndex, setActiveBranchIndex] = useState<number>(0);
  const activeBranch = config.branches[activeBranchIndex];

  return (
    <section className={`w-full py-16 sm:py-24 text-white ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header Block */}
        <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 gap-6">
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-[11px] font-mono uppercase tracking-[0.2em] text-cyan-400 mb-4">
              <GitBranch className="w-3.5 h-3.5" />
              <span>{config.eyebrow}</span>
            </div>
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black uppercase tracking-tight text-white leading-tight">
              {config.title}
            </h2>
            <p className="mt-4 text-sm sm:text-base text-white/65 max-w-2xl font-light leading-relaxed">
              {config.description}
            </p>
          </div>

          {/* Quick Metric Pill */}
          <div className="p-4 rounded-2xl border border-white/15 bg-[#0C0D12] flex items-center gap-4 shrink-0 shadow-lg">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 font-mono font-bold text-sm">
              {config.summaryStat.value}
            </div>
            <div>
              <div className="text-xs font-bold text-white uppercase tracking-wider">
                {config.summaryStat.label}
              </div>
              <div className="text-[10px] font-mono text-white/40">
                AUDITED IN REAL-TIME
              </div>
            </div>
          </div>
        </div>

        {/* Tactical Branching Canvas Frame (Image 2) */}
        <div className="relative rounded-3xl border border-white/15 bg-[#09090B] p-6 sm:p-10 overflow-hidden shadow-2xl">
          {/* Subtle Grid Pattern */}
          <div
            className="absolute inset-0 pointer-events-none opacity-[0.03]"
            style={{
              backgroundImage: `linear-gradient(to right, white 1px, transparent 1px), linear-gradient(to bottom, white 1px, transparent 1px)`,
              backgroundSize: '32px 32px',
            }}
          />

          {/* Top Timestamps Ruler */}
          <div className="relative z-10 grid grid-cols-5 gap-2 pb-6 border-b border-white/10 text-center">
            {config.timestamps.map((time, idx) => (
              <div key={idx} className="flex flex-col items-center">
                <span className="font-mono text-[10px] text-white/40 uppercase tracking-wider">
                  STEP 0{idx + 1}
                </span>
                <span className="font-mono text-xs font-semibold text-white/80 mt-1 truncate max-w-full">
                  {time}
                </span>
              </div>
            ))}
          </div>

          {/* Git-Style Branching DAG Visualizer (Desktop & Mobile) */}
          <div className="relative z-10 mt-8">
            {/* Main Trunk Line Bar */}
            <div className="flex items-center justify-between p-3.5 rounded-xl bg-white/[0.03] border border-white/10 mb-8">
              <div className="flex items-center gap-3">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" />
                <span className="font-mono text-xs font-bold text-emerald-400 uppercase tracking-wider">
                  TRUNK: {config.mainBranchLabel}
                </span>
                <span className="hidden sm:inline-block px-2 py-0.5 rounded text-[10px] font-mono bg-white/10 text-white/60">
                  HEAD: commit #4a88f1c
                </span>
              </div>
              <div className="flex items-center gap-2 text-[10px] font-mono text-white/50">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                <span className="hidden sm:inline">LIVE STATE ATOMIC WRITE</span>
                <span className="text-emerald-400 font-bold">100% ACID</span>
              </div>
            </div>

            {/* SVG Curved Branching Splines */}
            <div className="relative w-full h-32 hidden md:block">
              <svg className="w-full h-full" preserveAspectRatio="none" viewBox="0 0 1000 120">
                <defs>
                  <linearGradient id="mainGlow" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" stopColor="#10B981" stopOpacity="0.8" />
                    <stop offset="50%" stopColor="#06B6D4" stopOpacity="0.8" />
                    <stop offset="100%" stopColor="#3B82F6" stopOpacity="0.8" />
                  </linearGradient>
                  <linearGradient id="branchGlow1" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" stopColor="#10B981" />
                    <stop offset="100%" stopColor="#F59E0B" />
                  </linearGradient>
                  <linearGradient id="branchGlow2" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" stopColor="#06B6D4" />
                    <stop offset="100%" stopColor="#3B82F6" />
                  </linearGradient>
                </defs>

                {/* Main Horizontal Spine */}
                <line x1="20" y1="20" x2="980" y2="20" stroke="url(#mainGlow)" strokeWidth="3" strokeLinecap="round" />

                {/* Trunk Checkpoint Circles */}
                <circle cx="100" cy="20" r="5" fill="#10B981" />
                <circle cx="300" cy="20" r="5" fill="#10B981" />
                <circle cx="500" cy="20" r="5" fill="#06B6D4" />
                <circle cx="700" cy="20" r="5" fill="#3B82F6" />
                <circle cx="900" cy="20" r="5" fill="#10B981" />

                {/* Branch 1 Spline: Departs at x=100, drops to y=70, returns at x=500 */}
                <path
                  d="M 100 20 C 150 20, 180 70, 240 70 L 400 70 C 450 70, 470 20, 500 20"
                  fill="none"
                  stroke="url(#branchGlow1)"
                  strokeWidth="2"
                  strokeDasharray="4 4"
                  className={activeBranchIndex === 0 ? 'stroke-[3px] opacity-100' : 'opacity-40'}
                />

                {/* Branch 2 Spline: Departs at x=300, drops to y=100, returns at x=700 */}
                <path
                  d="M 300 20 C 350 20, 380 100, 450 100 L 600 100 C 650 100, 680 20, 700 20"
                  fill="none"
                  stroke="url(#branchGlow2)"
                  strokeWidth="2"
                  strokeDasharray="4 4"
                  className={activeBranchIndex === 1 ? 'stroke-[3px] opacity-100' : 'opacity-40'}
                />

                {/* Branch 3 Spline: Departs at x=500, drops to y=60, merges at x=900 */}
                <path
                  d="M 500 20 C 550 20, 580 60, 650 60 L 800 60 C 850 60, 880 20, 900 20"
                  fill="none"
                  stroke="#10B981"
                  strokeWidth="2"
                  strokeDasharray="4 4"
                  className={activeBranchIndex === 2 ? 'stroke-[3px] opacity-100' : 'opacity-40'}
                />

                {/* Dynamic Active Merge Dot Indicator */}
                <circle
                  cx={activeBranchIndex === 0 ? 320 : activeBranchIndex === 1 ? 525 : 725}
                  cy={activeBranchIndex === 0 ? 70 : activeBranchIndex === 1 ? 100 : 60}
                  r="6"
                  fill="#FFF"
                  className="animate-pulse"
                />
              </svg>
            </div>

            {/* 3 Interactive Branch Detail Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
              {config.branches.map((branch, idx) => {
                const isActive = activeBranchIndex === idx;
                const statusColorClass =
                  branch.statusColor === 'emerald'
                    ? 'text-emerald-400 border-emerald-500/30 bg-emerald-500/10'
                    : branch.statusColor === 'cyan'
                    ? 'text-cyan-400 border-cyan-500/30 bg-cyan-500/10'
                    : branch.statusColor === 'amber'
                    ? 'text-amber-400 border-amber-500/30 bg-amber-500/10'
                    : 'text-blue-400 border-blue-500/30 bg-blue-500/10';

                return (
                  <button
                    key={idx}
                    type="button"
                    onClick={() => setActiveBranchIndex(idx)}
                    className={`text-left rounded-2xl p-5 border transition-all relative overflow-hidden ${
                      isActive
                        ? 'border-white/40 bg-[#12131A] shadow-xl ring-1 ring-white/20'
                        : 'border-white/10 bg-[#0C0D12] hover:border-white/20 hover:bg-[#0F1017]'
                    }`}
                  >
                    {/* Active Glow Accent */}
                    {isActive && (
                      <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-emerald-400 via-cyan-400 to-blue-500" />
                    )}

                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <GitBranch className="w-3.5 h-3.5 text-white/50" />
                        <span className="font-mono text-xs font-bold text-white truncate max-w-[160px]">
                          {branch.name}
                        </span>
                      </div>
                      <span className={`px-2 py-0.5 rounded text-[9px] font-mono uppercase font-bold border ${statusColorClass}`}>
                        {branch.status}
                      </span>
                    </div>

                    <p className="text-xs text-white/70 font-light leading-relaxed mb-4 min-h-[36px]">
                      {branch.message}
                    </p>

                    <div className="pt-3 border-t border-white/5 flex items-center justify-between text-[10px] font-mono text-white/40">
                      <span>{branch.author}</span>
                      <span className="text-white/80 font-bold">{branch.metric}</span>
                    </div>

                    <div className="mt-2 flex items-center justify-between text-[9px] font-mono text-white/30">
                      <span>COMMIT #{branch.commitHash}</span>
                      <span className="flex items-center gap-1 text-emerald-400">
                        <GitMerge className="w-2.5 h-2.5" />
                        AUTO-MERGE OK
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>

            {/* Bottom Safe Execution Guarantee */}
            <div className="mt-8 p-4 rounded-xl border border-white/10 bg-white/[0.02] flex flex-col sm:flex-row items-center justify-between gap-4 text-xs">
              <div className="flex items-center gap-3">
                <div className="w-7 h-7 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
                  <ShieldCheck className="w-4 h-4" />
                </div>
                <div className="text-white/70">
                  <span className="font-bold text-white">Isolated Execution Guarantee:</span> All branch simulations run on read-only Spanner shadow instances without polluting active driver routes or financial ledgers.
                </div>
              </div>
              <div className="font-mono text-[10px] text-white/40 whitespace-nowrap">
                SAFE SIMULATION MODE: ACTIVE
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
