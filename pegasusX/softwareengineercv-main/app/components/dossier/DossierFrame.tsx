'use client';

import React from 'react';
import Link from 'next/link';

export interface DossierNavPill {
  label: string;
  href?: string;
  active?: boolean;
  icon?: React.ReactNode;
}

export interface DossierFrameProps {
  tabLabel?: string;
  tabSublabel?: string;
  pills?: DossierNavPill[];
  children: React.ReactNode;
  className?: string;
}

/**
 * DossierFrame — Architectural Folder-Tab Envelope
 * Replicating the physical dossier silhouette from the reference design:
 * - Raised top-left dossier tab with smooth chamfer S-curve transition
 * - Top-right horizontal pill navigation bar
 * - 4-corner coordinate crosshairs (+)
 * - Precision 1px hairline border & tactical dark canvas
 */
export default function DossierFrame({
  tabLabel = 'PEGASUS / CORE',
  tabSublabel,
  pills = [],
  children,
  className = '',
}: DossierFrameProps) {
  return (
    <div className={`relative w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 ${className}`}>
      {/* Outer Dossier Container */}
      <div className="relative w-full rounded-2xl bg-[#09090C] border border-white/15 shadow-[0_24px_80px_rgba(0,0,0,0.85)] overflow-hidden">
        {/* Subtle background noise and film grain texture */}
        <div 
          className="absolute inset-0 pointer-events-none opacity-[0.035] mix-blend-screen bg-repeat z-0"
          style={{ backgroundImage: 'radial-gradient(#ffffff 1px, transparent 1px)', backgroundSize: '16px 16px' }}
        />

        {/* Ambient atmospheric corner glows */}
        <div className="absolute top-0 right-0 w-96 h-96 bg-blue-600/5 rounded-full blur-[100px] pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-white/[0.02] rounded-full blur-[90px] pointer-events-none" />

        {/* =============================================================== */}
        {/* TOP DOSSIER TAB & PILL NAVIGATION HEADER                       */}
        {/* =============================================================== */}
        <div className="relative z-10 w-full flex flex-col md:flex-row items-stretch md:items-center justify-between border-b border-white/10">
          {/* Top-Left Folder Tab Silhouette */}
          <div className="relative flex items-center gap-3 px-6 sm:px-8 py-4 sm:py-5 bg-[#101015] border-r border-white/10 md:rounded-br-2xl md:after:content-[''] md:after:absolute md:after:-right-[24px] md:after:top-0 md:after:bottom-0 md:after:w-[24px] md:after:bg-transparent">
            {/* Green glowing status dot */}
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#CEFF00] opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-[#CEFF00]" />
            </span>

            <div className="flex flex-col">
              <span className="font-mono text-xs sm:text-sm font-bold tracking-[0.2em] text-white uppercase select-none">
                {tabLabel}
              </span>
              {tabSublabel ? (
                <span className="font-mono text-[10px] tracking-wider text-white/40 uppercase">
                  {tabSublabel}
                </span>
              ) : null}
            </div>
          </div>

          {/* Top-Right Pill Navigation & Utilities */}
          {pills && pills.length > 0 ? (
            <div className="flex items-center gap-2 px-4 sm:px-6 py-3 overflow-x-auto no-scrollbar">
              {pills.map((pill, idx) => {
                const isAction = idx === pills.length - 1;
                const baseClasses = `inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-mono transition-all duration-200 whitespace-nowrap select-none`;
                const stateClasses = pill.active
                  ? 'bg-white text-black font-semibold shadow-sm'
                  : isAction
                    ? 'bg-white/10 hover:bg-white/20 text-white border border-white/20 font-medium'
                    : 'bg-white/5 hover:bg-white/10 text-white/70 hover:text-white border border-white/10';

                return pill.href ? (
                  <Link key={pill.label} href={pill.href} className={`${baseClasses} ${stateClasses}`}>
                    <span>{pill.label}</span>
                    {pill.icon}
                  </Link>
                ) : (
                  <span key={pill.label} className={`${baseClasses} ${stateClasses} cursor-default`}>
                    <span>{pill.label}</span>
                    {pill.icon}
                  </span>
                );
              })}
            </div>
          ) : null}
        </div>

        {/* =============================================================== */}
        {/* TACTICAL COORDINATE CROSSHAIRS (+) IN CORNERS                  */}
        {/* =============================================================== */}
        {/* Top Left Crosshair */}
        <div className="absolute top-20 left-8 z-20 pointer-events-none select-none font-mono text-sm text-white/25">
          +
        </div>
        {/* Top Right Crosshair */}
        <div className="absolute top-20 right-8 z-20 pointer-events-none select-none font-mono text-sm text-white/25">
          +
        </div>
        {/* Mid Center Left Crosshair */}
        <div className="absolute top-1/2 left-8 -translate-y-1/2 z-20 pointer-events-none select-none font-mono text-sm text-white/15">
          +
        </div>
        {/* Mid Center Right Crosshair */}
        <div className="absolute top-1/2 right-8 -translate-y-1/2 z-20 pointer-events-none select-none font-mono text-sm text-white/15">
          +
        </div>
        {/* Bottom Left Crosshair */}
        <div className="absolute bottom-8 left-8 z-20 pointer-events-none select-none font-mono text-sm text-white/25">
          +
        </div>
        {/* Bottom Right Crosshair */}
        <div className="absolute bottom-8 right-8 z-20 pointer-events-none select-none font-mono text-sm text-white/25">
          +
        </div>

        {/* =============================================================== */}
        {/* MAIN DOSSIER INTERIOR CONTENT                                  */}
        {/* =============================================================== */}
        <div className="relative z-10 p-6 sm:p-10 lg:p-12">
          {children}
        </div>
      </div>
    </div>
  );
}
