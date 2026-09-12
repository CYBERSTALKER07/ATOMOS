'use client';

import React from 'react';
import { Sparkles, Hash, Clock, Calendar } from 'lucide-react';
import { DocArticle } from '@/app/data/docsData';

type DocsHeroBannerProps = {
  article: DocArticle;
};

export default function DocsHeroBanner({ article }: DocsHeroBannerProps) {
  const [wordTop, wordBottom] = article.heroWordmark || ['Pegasus', 'Docs'];

  return (
    <div className="space-y-6">
      {/* Signature 3D Hero Card Banner (Replicating Primer Brand layout in Tactical Dark Aesthetic) */}
      <div className="relative w-full rounded-3xl overflow-hidden bg-gradient-to-br from-[#0D0D14] via-[#11111B] to-[#141828] border border-[#222234] shadow-[0_8px_32px_rgba(0,0,0,0.6)] min-h-[260px] sm:min-h-[320px] flex items-center justify-between p-8 sm:p-12 lg:p-14">
        {/* Ambient Glows */}
        <div className="absolute -top-24 -right-24 w-96 h-96 bg-blue-600/15 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute -bottom-24 right-1/4 w-80 h-80 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute top-1/2 left-1/3 w-64 h-64 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />

        {/* Tactical Grid Coordinates Overlay */}
        <div
          className="absolute inset-0 opacity-[0.04] pointer-events-none"
          style={{
            backgroundImage: `radial-gradient(circle at 1px 1px, #ffffff 1px, transparent 0)`,
            backgroundSize: '24px 24px',
          }}
        />

        {/* Left: Giant Bold Typography Wordmark */}
        <div className="relative z-10 max-w-xl">
          {article.badge && (
            <div className="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-[#181826] border border-[#2F2F44] text-[11px] font-mono font-medium text-[#93C5FD] mb-4 shadow-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse" />
              <span>{article.badge}</span>
            </div>
          )}

          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black tracking-tight leading-[0.95] text-white select-none">
            <span className="block text-white drop-shadow-sm">{wordTop}</span>
            <span className="block bg-gradient-to-r from-[#A5B4FC] via-[#CBD5E1] to-[#64748B] bg-clip-text text-transparent">
              {wordBottom}
            </span>
          </h1>

          <p className="mt-4 text-xs sm:text-sm font-mono text-[#8E8EA8] tracking-wide uppercase">
            Sovereign Physical Distribution Platform · Cluster Node
          </p>
        </div>

        {/* Right: Abstract 3D Luminous Sphere with Orbitals */}
        <div className="hidden md:flex relative z-10 w-64 h-64 lg:w-80 lg:h-80 shrink-0 items-center justify-center">
          {/* Outer Ring */}
          <div className="absolute w-60 h-60 lg:w-72 lg:h-72 rounded-full border border-cyan-500/20 animate-[spin_60s_linear_infinite]" />
          <div className="absolute w-48 h-48 lg:w-56 lg:h-56 rounded-full border border-dashed border-blue-400/25 animate-[spin_40s_linear_infinite_reverse]" />

          {/* Glowing 3D Orb */}
          <div className="relative w-40 h-40 lg:w-48 lg:h-48 rounded-full bg-gradient-to-tr from-[#1E3A8A] via-[#0284C7] to-[#34D399] shadow-[0_0_60px_rgba(2,132,199,0.45)] border border-white/20 flex items-center justify-center overflow-hidden">
            {/* Curving 3D highlight contour lines */}
            <div className="absolute -inset-10 opacity-40 mix-blend-overlay bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-white via-transparent to-black" />
            <svg
              className="absolute inset-0 w-full h-full opacity-30 text-white"
              viewBox="0 0 100 100"
              fill="none"
              stroke="currentColor"
              strokeWidth="0.75"
            >
              <ellipse cx="50" cy="50" rx="45" ry="18" transform="rotate(-25 50 50)" />
              <ellipse cx="50" cy="50" rx="45" ry="32" transform="rotate(-25 50 50)" />
              <ellipse cx="50" cy="50" rx="45" ry="42" transform="rotate(-25 50 50)" />
              <line x1="10" y1="10" x2="90" y2="90" strokeDasharray="2,2" />
            </svg>
            <div className="w-16 h-16 rounded-full bg-white/10 backdrop-blur-md border border-white/30 shadow-inner flex items-center justify-center">
              <Sparkles className="w-7 h-7 text-white drop-shadow" />
            </div>
          </div>
        </div>
      </div>

      {/* Under-Banner Metadata Line (Matching the reference "# v4.0.0 · Last updated...") */}
      <div className="flex flex-wrap items-center gap-y-2 gap-x-4 pt-2 text-xs font-mono text-[#8E8EA0] border-b border-[#1F1F2B] pb-5">
        <div className="flex items-center space-x-1.5 text-[#3B82F6] font-semibold">
          <Hash className="w-3.5 h-3.5" />
          <span>{article.version}</span>
        </div>
        <span className="text-[#3F3F50]">·</span>
        <div className="flex items-center space-x-1.5">
          <Calendar className="w-3.5 h-3.5 text-[#6E6E80]" />
          <span>Last updated {article.lastUpdated}</span>
        </div>
        <span className="text-[#3F3F50]">·</span>
        <div className="flex items-center space-x-1.5">
          <Clock className="w-3.5 h-3.5 text-[#6E6E80]" />
          <span>{article.readTime}</span>
        </div>
      </div>

      {/* Large Lead Sentence */}
      <div className="pt-2">
        <h2 className="text-xl sm:text-2xl lg:text-3xl font-bold tracking-tight text-white leading-snug">
          {article.leadSentence}
        </h2>
        <p className="mt-3 text-sm sm:text-base text-[#9A9AA8] leading-relaxed max-w-4xl">
          {article.summary}
        </p>
      </div>
    </div>
  );
}
