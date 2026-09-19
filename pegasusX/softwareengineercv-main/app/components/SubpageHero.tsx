'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';

export type SubpageHeroProps = {
  categoryLabel?: string;
  categoryHref?: string;
  badge?: string;
  badgeIcon?: ReactNode;
  title: string;
  summary: string;
  primaryCta?: {
    label: string;
    href: string;
  };
  secondaryCta?: {
    label: string;
    href: string;
  };
  widget?: {
    title: string;
    description: string;
    href?: string;
  };
  nodes?: {
    left?: {
      tag?: string;
      value?: string;
      symbol?: string;
    };
    center?: {
      tag?: string;
      value?: string;
    };
    right?: {
      tag?: string;
      value?: string;
      symbol?: string;
    };
  };
  breadcrumb?: {
    homeLabel?: string;
    categoryLabel?: string;
    categoryHref?: string;
    currentPage?: string;
  };
};

export default function SubpageHero({
  categoryLabel,
  categoryHref,
  badge,
  badgeIcon,
  title,
  summary,
  primaryCta,
  secondaryCta,
  widget,
  nodes,
  breadcrumb,
}: SubpageHeroProps) {
  const eyebrowBadge = badge || (categoryLabel ? `PEGASUS OS // ${categoryLabel.toUpperCase()}` : 'PEGASUS OPERATIONAL SYSTEM');
  
  const primaryButton = primaryCta || {
    label: 'REQUEST DEMO',
    href: '/join',
  };

  const secondaryButton = secondaryCta || {
    label: 'EXPLORE STACK',
    href: categoryHref || '/platform',
  };

  const widgetData = widget || {
    title: 'INTRODUCING PEGASUS OS',
    description: 'Autonomous orchestration across all supply chain roles.',
    href: '/platform',
  };

  const leftNode = nodes?.left || {
    tag: 'DISPATCHING',
    value: '6.54K UNITS',
    symbol: 'T',
  };

  const centerNode = nodes?.center || {
    tag: 'PROCESSING',
    value: 'BLOCK 164...674',
  };

  const rightNode = nodes?.right || {
    tag: 'SETTLED',
    value: 'ZERO LATENCY',
    symbol: 'ETH',
  };

  return (
    <section className="relative w-full bg-black text-white pt-24 pb-16 sm:pt-28 sm:pb-20 md:pt-32 md:pb-24 px-4 sm:px-6 lg:px-8 overflow-hidden border-b border-white/[0.08] select-none">
      
      {/* ========================================================================= */}
      {/* SQUIRCLE MATRIX CANVAS & GLOWING TRACE BACKGROUND */}
      {/* ========================================================================= */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        
        {/* Soft radial ambient glow */}
        <div 
          className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[900px] h-[550px] bg-gradient-to-b from-white/[0.035] via-white/[0.015] to-transparent rounded-full blur-3xl"
        />

        {/* Squircle Matrix Grid */}
        <div className="absolute inset-0 flex items-center justify-center opacity-70">
          <div className="grid grid-cols-6 sm:grid-cols-8 md:grid-cols-10 gap-2.5 sm:gap-3 p-4 w-full max-w-7xl">
            {Array.from({ length: 40 }).map((_, i) => (
              <div
                key={i}
                className="aspect-square rounded-2xl sm:rounded-[20px] bg-black border border-white/[0.05] transition-colors"
              />
            ))}
          </div>
        </div>

        {/* Illuminated Tile Accents (matching reference image) */}
        <div className="absolute inset-0 max-w-7xl mx-auto hidden md:block">
          {/* Top-center illuminated border tile */}
          <div className="absolute top-[8%] left-[46%] w-24 h-24 rounded-[20px] border border-white/30 shadow-[0_0_15px_rgba(255,255,255,0.1)] pointer-events-none" />
          
          {/* Subtle horizontal highlight trace on right side */}
          <div className="absolute top-[28%] right-[12%] w-44 h-[1px] bg-gradient-to-r from-transparent via-white/30 to-transparent pointer-events-none" />
          <div className="absolute top-[32%] right-[10%] w-32 h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent pointer-events-none" />
        </div>

        {/* Glowing Curved Energy Track (SVG curve flowing through the grid) */}
        <svg
          className="absolute inset-0 w-full h-full pointer-events-none hidden md:block"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="curveGlow" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#ffffff" stopOpacity="0.0" />
              <stop offset="30%" stopColor="#ffffff" stopOpacity="0.4" />
              <stop offset="60%" stopColor="#ffffff" stopOpacity="0.8" />
              <stop offset="100%" stopColor="#ffffff" stopOpacity="0.1" />
            </linearGradient>
            <filter id="blurFilter" x="-20%" y="-20%" width="140%" height="140%">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          {/* Smooth S-curve connecting top-center toward center card */}
          <path
            d="M 540 50 Q 560 140, 520 180 T 480 230"
            fill="none"
            stroke="url(#curveGlow)"
            strokeWidth="1.5"
            filter="url(#blurFilter)"
          />
        </svg>

        {/* Bottom fading mask */}
        <div className="absolute inset-x-0 bottom-0 h-44 bg-gradient-to-t from-black via-black/80 to-transparent pointer-events-none" />
      </div>

      {/* ========================================================================= */}
      {/* 3 FLOATING INTERACTIVE SYSTEM CARDS (LEFT, CENTER, RIGHT) */}
      {/* ========================================================================= */}
      <div className="relative max-w-7xl mx-auto z-10 hidden sm:block">
        <div className="relative h-44 sm:h-52 md:h-64 w-full">
          
          {/* Card 1: Left Node (e.g. Swapping / Ingestion) */}
          <div className="absolute top-2 left-[4%] sm:left-[8%] md:left-[12%] w-28 sm:w-32 md:w-36 h-32 sm:h-36 md:h-40 rounded-2xl sm:rounded-[22px] bg-black border border-white/[0.14] p-3.5 flex flex-col justify-between items-center shadow-2xl backdrop-blur-md hover:border-white/30 transition-all group">
            {/* Top-right rotating indicator badge */}
            <div className="w-full flex justify-end">
              <div className="w-5 h-5 rounded-full bg-white/[0.06] border border-white/15 flex items-center justify-center">
                <div className="w-2.5 h-2.5 rounded-full border-t border-r border-white animate-spin" />
              </div>
            </div>

            {/* Glyph Icon */}
            <div className="w-9 h-9 flex items-center justify-center text-white font-serif text-2xl font-bold tracking-tighter opacity-90 group-hover:scale-105 transition-transform">
              {leftNode.symbol === 'T' ? (
                <svg viewBox="0 0 24 24" className="w-7 h-7 stroke-current fill-none stroke-[1.5]">
                  <path d="M4 6h16M12 6v14M8 6l1-2h6l1 2" strokeLinecap="round" />
                </svg>
              ) : (
                <span className="font-mono font-bold text-lg">{leftNode.symbol}</span>
              )}
            </div>

            {/* Bottom Monospace Telemetry */}
            <div className="w-full text-center">
              <span className="text-[9px] font-mono tracking-widest uppercase text-zinc-500 block">
                {leftNode.tag}
              </span>
              <span className="text-[10px] sm:text-[11px] font-mono font-medium text-white block truncate">
                {leftNode.value}
              </span>
              <span className="text-[9px] font-mono text-zinc-600 block mt-0.5">//</span>
            </div>
          </div>

          {/* Card 2: Center Node (Processing Isometric 3D Cube) */}
          <div className="absolute top-16 sm:top-14 md:top-16 left-[44%] sm:left-[46%] -translate-x-1/2 w-28 sm:w-32 md:w-36 h-32 sm:h-36 md:h-40 rounded-2xl sm:rounded-[22px] bg-black border border-white/[0.18] p-3.5 flex flex-col justify-between items-center shadow-[0_0_40px_rgba(0,0,0,1)] backdrop-blur-md hover:border-white/40 transition-all group z-10">
            <div className="w-full h-4" />

            {/* 3D Wireframe Isometric Cube Glyph */}
            <div className="w-10 h-10 flex items-center justify-center group-hover:scale-110 transition-transform">
              <svg viewBox="0 0 24 24" className="w-9 h-9 text-white stroke-current fill-none stroke-[1.4]">
                <path d="M12 2L3 7v10l9 5 9-5V7l-9-5z" strokeLinejoin="round" />
                <path d="M12 12L3 7" strokeLinejoin="round" />
                <path d="M12 12l9-5" strokeLinejoin="round" />
                <path d="M12 12v10" strokeLinejoin="round" />
              </svg>
            </div>

            {/* Bottom Monospace Telemetry */}
            <div className="w-full text-center">
              <span className="text-[9px] font-mono tracking-widest uppercase text-zinc-500 flex items-center justify-center gap-1.5">
                <span>{centerNode.tag}</span>
                <span className="w-1.5 h-1.5 rounded-full bg-white animate-ping inline-block" />
              </span>
              <span className="text-[10px] sm:text-[11px] font-mono font-medium text-white block truncate">
                {centerNode.value}
              </span>
              <span className="text-[9px] font-mono text-zinc-600 block mt-0.5">//</span>
            </div>
          </div>

          {/* Card 3: Right Node (e.g. Swapped / Settled Crystal) */}
          <div className="absolute top-2 right-[4%] sm:right-[8%] md:right-[12%] w-28 sm:w-32 md:w-36 h-32 sm:h-36 md:h-40 rounded-2xl sm:rounded-[22px] bg-black border border-white/[0.14] p-3.5 flex flex-col justify-between items-center shadow-2xl backdrop-blur-md hover:border-white/30 transition-all group">
            {/* Top-right checkmark circle badge */}
            <div className="w-full flex justify-end">
              <div className="w-5 h-5 rounded-full bg-white/[0.08] border border-white/20 flex items-center justify-center text-[10px] text-white font-bold">
                ✓
              </div>
            </div>

            {/* Geometric Crystal / Ethereum Diamond Glyph */}
            <div className="w-9 h-9 flex items-center justify-center text-white group-hover:scale-105 transition-transform">
              <svg viewBox="0 0 24 24" className="w-7 h-7 stroke-current fill-none stroke-[1.5]">
                <path d="M12 2L5 12l7 4 7-4-7-10z" strokeLinejoin="round" />
                <path d="M5 12l7 10 7-10-7 4-7-4z" strokeLinejoin="round" />
              </svg>
            </div>

            {/* Bottom Monospace Telemetry */}
            <div className="w-full text-center">
              <span className="text-[9px] font-mono tracking-widest uppercase text-zinc-500 block">
                {rightNode.tag}
              </span>
              <span className="text-[10px] sm:text-[11px] font-mono font-medium text-zinc-200 block truncate">
                {rightNode.value}
              </span>
              <span className="text-[9px] font-mono text-zinc-600 block mt-0.5">//</span>
            </div>
          </div>

        </div>
      </div>

      {/* ========================================================================= */}
      {/* FOREGROUND CONTENT SECTION: HEADLINE, DESCRIPTION, CTAS & WIDGET */}
      {/* ========================================================================= */}
      <div className="relative max-w-7xl mx-auto z-20 mt-4 sm:mt-6 md:mt-10">
        
        {/* Optional Breadcrumb Nav */}
        {breadcrumb && (
          <nav aria-label="Breadcrumb" className="mb-6 flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-zinc-500">
            <Link href="/" className="hover:text-white transition-colors">
              {breadcrumb.homeLabel || 'Home'}
            </Link>
            {breadcrumb.categoryLabel && (
              <>
                <span aria-hidden>/</span>
                <Link href={breadcrumb.categoryHref || '#'} className="hover:text-white transition-colors">
                  {breadcrumb.categoryLabel}
                </Link>
              </>
            )}
            {breadcrumb.currentPage && (
              <>
                <span aria-hidden>/</span>
                <span className="text-zinc-300" aria-current="page">
                  {breadcrumb.currentPage}
                </span>
              </>
            )}
          </nav>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-end">
          
          {/* Left Column: Eyebrow, Massive Title, Summary, and Dual CTA Buttons (7-8 cols) */}
          <div className="lg:col-span-8 flex flex-col items-start text-left">
            
            {/* Pill Eyebrow Badge */}
            <div className="inline-flex items-center gap-2.5 px-3.5 py-1.5 rounded-full bg-white/[0.06] border border-white/15 text-xs font-mono tracking-wider uppercase text-zinc-300 mb-5 backdrop-blur-md">
              {badgeIcon || (
                <span className="w-4 h-4 rounded bg-white/10 border border-white/20 text-[10px] flex items-center justify-center font-bold text-white">
                  T
                </span>
              )}
              <span>{eyebrowBadge}</span>
            </div>

            {/* Main Headline (H1) - Industrial Uppercase Sans */}
            <h1 className="text-3xl sm:text-5xl md:text-6xl lg:text-[62px] font-black uppercase tracking-tight text-white leading-[1.06] font-sans">
              {title}
            </h1>

            {/* Subtitle */}
            <p className="mt-5 text-sm sm:text-base md:text-lg text-zinc-400 max-w-2xl leading-relaxed font-normal">
              {summary}
            </p>

            {/* Dual CTA Buttons */}
            <div className="mt-8 flex flex-wrap items-center gap-3.5">
              <Link
                href={primaryButton.href}
                className="px-6 py-3 rounded-lg bg-white text-black font-bold uppercase tracking-wider text-xs sm:text-sm hover:bg-zinc-200 transition-all shadow-[0_0_25px_rgba(255,255,255,0.18)] active:scale-95"
              >
                {primaryButton.label}
              </Link>
              <Link
                href={secondaryButton.href}
                className="px-6 py-3 rounded-lg bg-white/[0.05] hover:bg-white/[0.1] text-white font-semibold uppercase tracking-wider text-xs sm:text-sm border border-white/15 hover:border-white/30 transition-all active:scale-95"
              >
                {secondaryButton.label}
              </Link>
            </div>

          </div>

          {/* Right Column: Bottom-Right Feature / Status Widget (4 cols) */}
          <div className="lg:col-span-4 flex justify-start lg:justify-end w-full">
            <Link
              href={widgetData.href || '/platform'}
              className="w-full max-w-sm rounded-2xl bg-black border border-white/15 p-3.5 sm:p-4 flex items-center gap-3.5 backdrop-blur-md shadow-2xl hover:border-white/35 hover:bg-zinc-950 transition-all group"
            >
              {/* Dotted Matrix Icon Thumbnail */}
              <div className="w-10 h-10 rounded-xl bg-white/[0.04] border border-white/10 flex items-center justify-center flex-shrink-0 text-zinc-400 group-hover:text-white transition-colors">
                <svg className="w-5 h-5 fill-current" viewBox="0 0 24 24">
                  <rect x="3" y="3" width="3" height="3" rx="0.5" />
                  <rect x="10.5" y="3" width="3" height="3" rx="0.5" />
                  <rect x="18" y="3" width="3" height="3" rx="0.5" />
                  <rect x="3" y="10.5" width="3" height="3" rx="0.5" />
                  <rect x="10.5" y="10.5" width="3" height="3" rx="0.5" opacity="0.4" />
                  <rect x="18" y="10.5" width="3" height="3" rx="0.5" />
                  <rect x="3" y="18" width="3" height="3" rx="0.5" />
                  <rect x="10.5" y="18" width="3" height="3" rx="0.5" />
                  <rect x="18" y="18" width="3" height="3" rx="0.5" />
                </svg>
              </div>

              {/* Title & Description */}
              <div className="flex-1 min-w-0">
                <div className="text-xs font-bold uppercase tracking-wider text-white group-hover:text-zinc-100 transition-colors truncate">
                  {widgetData.title}
                </div>
                <div className="text-[11px] text-zinc-400 font-normal truncate mt-0.5">
                  {widgetData.description}
                </div>
              </div>

              {/* Arrow Circle Indicator */}
              <div className="w-7 h-7 rounded-full bg-white/[0.08] border border-white/10 flex items-center justify-center text-zinc-400 group-hover:text-white group-hover:bg-white/20 transition-all flex-shrink-0">
                <span className="text-xs font-bold">›</span>
              </div>
            </Link>
          </div>

        </div>

      </div>

    </section>
  );
}
