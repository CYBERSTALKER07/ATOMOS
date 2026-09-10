'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import {
  Cpu,
  Shield,
  Zap,
  Truck,
  Barcode,
  FileCheck,
  Satellite,
  Database,
  Layers,
  ArrowRight,
  ExternalLink,
  Activity,
  Workflow,
  Network,
} from 'lucide-react';
import type { RadialHubSpokeConfig, RadialSatellite } from './sectionConfigs';

const ICON_MAP = {
  cpu: Cpu,
  shield: Shield,
  zap: Zap,
  truck: Truck,
  barcode: Barcode,
  'file-check': FileCheck,
  satellite: Satellite,
  database: Database,
  layers: Layers,
  network: Network,
  workflow: Workflow,
};

type RadialHubSpokeSectionProps = {
  config: RadialHubSpokeConfig;
  className?: string;
};

export default function RadialHubSpokeSection({
  config,
  className = '',
}: RadialHubSpokeSectionProps) {
  const [hoveredPod, setHoveredPod] = useState<string | null>(null);

  // Hexagonal radial angles for the 6 pods (in degrees)
  // 0: top (270°), 1: top-right (330°), 2: bottom-right (30°), 3: bottom (90°), 4: bottom-left (150°), 5: top-left (210°)
  const podAngles = [270, 330, 30, 90, 150, 210];

  return (
    <section className={`w-full py-16 sm:py-24 text-white overflow-hidden ${className}`}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-8 items-center">
          {/* Left Column: Narrative Copy & CTAs (5 cols) */}
          <div className="lg:col-span-5 flex flex-col justify-center">
            <div className="inline-flex items-center gap-2 px-3.5 py-1 rounded-full border border-white/15 bg-white/[0.04] text-[11px] font-mono uppercase tracking-[0.2em] text-emerald-400 mb-6 w-fit">
              <Activity className="w-3.5 h-3.5" />
              <span>{config.badge}</span>
            </div>

            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-black uppercase tracking-tight text-white leading-tight">
              {config.title}
            </h2>

            <p className="mt-5 text-sm sm:text-base text-white/65 leading-relaxed font-light">
              {config.description}
            </p>

            {/* Action CTAs */}
            <div className="mt-8 flex flex-wrap items-center gap-4">
              <Link
                href={config.primaryBtn.href}
                className="inline-flex items-center gap-2 px-6 py-3.5 rounded-xl bg-white text-black font-semibold text-xs tracking-wider uppercase hover:bg-white/90 transition-all shadow-lg hover:shadow-white/10"
              >
                <span>{config.primaryBtn.label}</span>
                <ArrowRight className="w-4 h-4" />
              </Link>
              <Link
                href={config.secondaryBtn.href}
                className="inline-flex items-center gap-2 px-6 py-3.5 rounded-xl border border-white/20 bg-white/[0.03] text-white font-semibold text-xs tracking-wider uppercase hover:bg-white/10 hover:border-white/40 transition-all"
              >
                <span>{config.secondaryBtn.label}</span>
                <ExternalLink className="w-3.5 h-3.5 text-white/60" />
              </Link>
            </div>

            {/* Tech Stack Partner Badges */}
            <div className="mt-12 pt-8 border-t border-white/10">
              <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 block mb-3">
                INTEGRATED INFRASTRUCTURE BUS
              </span>
              <div className="flex flex-wrap gap-2">
                {config.techBadges.map((badge, idx) => (
                  <span
                    key={idx}
                    className="px-2.5 py-1 rounded-md border border-white/10 bg-white/[0.02] text-[10px] font-mono text-white/60 uppercase"
                  >
                    {badge}
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* Right Column: 3D Isometric Radial Hub-and-Spoke Canvas (7 cols) */}
          <div className="lg:col-span-7 relative flex items-center justify-center min-h-[540px]">
            {/* Ambient Radial Background Glow */}
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <div className="w-[500px] h-[500px] bg-emerald-500/10 rounded-full blur-3xl" />
              <div className="w-[300px] h-[300px] bg-cyan-500/10 rounded-full blur-2xl" />
            </div>

            {/* Isometric Perspective Container */}
            <div className="relative w-full max-w-[560px] aspect-square flex items-center justify-center">
              {/* Concentric Orbital Rings */}
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="w-[85%] h-[85%] rounded-full border border-dashed border-white/15" />
                <div className="w-[58%] h-[58%] rounded-full border border-white/10" />
                <div className="w-[38%] h-[38%] rounded-full border border-emerald-500/20" />
              </div>

              {/* Connecting SVG Curved Conduits */}
              <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 600 600">
                <defs>
                  <linearGradient id="spokeGlow" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stopColor="#10B981" stopOpacity="0.8" />
                    <stop offset="100%" stopColor="#06B6D4" stopOpacity="0.2" />
                  </linearGradient>
                </defs>

                {/* 6 Radiating Conduits from center (300, 300) to pod positions */}
                {podAngles.map((deg, i) => {
                  const rad = (deg * Math.PI) / 180;
                  const radius = 220;
                  const x = 300 + radius * Math.cos(rad);
                  const y = 300 + radius * Math.sin(rad);

                  // Curve control points
                  const midRadius = 120;
                  const cx = 300 + midRadius * Math.cos(rad + 0.2);
                  const cy = 300 + midRadius * Math.sin(rad + 0.2);

                  const isHovered = hoveredPod === config.satellites[i]?.id;

                  return (
                    <g key={i}>
                      <path
                        d={`M 300 300 Q ${cx} ${cy}, ${x} ${y}`}
                        fill="none"
                        stroke={isHovered ? '#10B981' : 'url(#spokeGlow)'}
                        strokeWidth={isHovered ? 2.5 : 1.5}
                        strokeDasharray={isHovered ? 'none' : '4 4'}
                        className="transition-all duration-300"
                      />
                      <circle cx={x} cy={y} r="3" fill="#10B981" />
                    </g>
                  );
                })}
              </svg>

              {/* Central Glowing 3D Cylinder / Core Node */}
              <div className="relative z-20 flex flex-col items-center justify-center w-36 h-36 rounded-full bg-[#08080C] border-2 border-emerald-500/40 shadow-[0_0_40px_rgba(16,185,129,0.25)] text-center p-3">
                <div className="w-10 h-10 rounded-xl bg-emerald-500/20 border border-emerald-400/40 flex items-center justify-center text-emerald-300 mb-1">
                  <Workflow className="w-5 h-5 animate-spin" style={{ animationDuration: '16s' }} />
                </div>
                <span className="font-mono text-[10px] font-black text-white uppercase tracking-wider leading-tight">
                  {config.centerNode.title}
                </span>
                <span className="text-[8px] font-mono text-emerald-400 mt-0.5">
                  {config.centerNode.latency}
                </span>
              </div>

              {/* 6 Surrounding Satellite Pods (Absolute Positioned via Math) */}
              {config.satellites.map((sat, i) => {
                const deg = podAngles[i];
                const rad = (deg * Math.PI) / 180;
                // Percent positions from center (50%, 50%)
                const radiusPct = 40; // 40% out from center
                const leftPct = 50 + radiusPct * Math.cos(rad);
                const topPct = 50 + radiusPct * Math.sin(rad);

                const Icon = ICON_MAP[sat.iconName] || Zap;
                const isHovered = hoveredPod === sat.id;

                const accentBorder =
                  sat.accent === 'cyan'
                    ? 'hover:border-cyan-400 group-hover:text-cyan-300'
                    : sat.accent === 'emerald'
                    ? 'hover:border-emerald-400 group-hover:text-emerald-300'
                    : sat.accent === 'amber'
                    ? 'hover:border-amber-400 group-hover:text-amber-300'
                    : sat.accent === 'purple'
                    ? 'hover:border-purple-400 group-hover:text-purple-300'
                    : 'hover:border-blue-400 group-hover:text-blue-300';

                return (
                  <div
                    key={sat.id}
                    onMouseEnter={() => setHoveredPod(sat.id)}
                    onMouseLeave={() => setHoveredPod(null)}
                    style={{
                      left: `${leftPct}%`,
                      top: `${topPct}%`,
                      transform: 'translate(-50%, -50%)',
                    }}
                    className={`absolute z-30 group cursor-pointer transition-all duration-300 ${
                      isHovered ? 'scale-110' : 'scale-100'
                    }`}
                  >
                    <div
                      className={`w-36 sm:w-40 p-3 rounded-xl border bg-[#0B0C10]/95 backdrop-blur-md transition-all shadow-xl ${
                        isHovered
                          ? 'border-white/60 bg-[#12131C] shadow-emerald-500/10'
                          : 'border-white/15'
                      } ${accentBorder}`}
                    >
                      <div className="flex items-center justify-between mb-1.5">
                        <div className="w-6 h-6 rounded-lg bg-white/[0.05] border border-white/10 flex items-center justify-center text-white/80 group-hover:text-emerald-400 transition-colors">
                          <Icon className="w-3.5 h-3.5" />
                        </div>
                        <span className="text-[8px] font-mono px-1.5 py-0.5 rounded bg-white/5 border border-white/10 text-emerald-400 font-semibold uppercase">
                          {sat.status}
                        </span>
                      </div>
                      <div className="font-mono text-[11px] font-bold text-white truncate group-hover:text-emerald-300 transition-colors">
                        {sat.name}
                      </div>
                      <div className="text-[9px] text-white/50 truncate font-light mt-0.5">
                        {sat.subtitle}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
