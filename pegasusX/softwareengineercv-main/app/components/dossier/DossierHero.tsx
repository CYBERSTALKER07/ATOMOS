'use client';

import React from 'react';
import DossierFrame, { type DossierNavPill } from './DossierFrame';
import DossierIsometricStack, { type IsometricTopNode } from './DossierIsometricStack';
import { ArrowDown } from 'lucide-react';

export interface DossierHeroProps {
  tabLabel?: string;
  tabSublabel?: string;
  pills?: DossierNavPill[];
  highlightText: string;
  headlineRest: string;
  missionTitle: string;
  missionBody: string;
  topNodes: [IsometricTopNode, IsometricTopNode, IsometricTopNode, IsometricTopNode];
  midTierLabel?: string;
  baseTierLabel?: string;
  accentColor?: string;
  scrollTargetId?: string;
  className?: string;
}

/**
 * DossierHero — Universal Dossier Hero Section
 * Fully modular section implementing the exact layout from the reference design:
 * 1. Dossier envelope frame with folder tab & nav pills
 * 2. Highlighted headline badge with blinking terminal cursor
 * 3. 4-corner bracketed mission box with domain authority copy
 * 4. Hairline circular down-arrow scroll button
 * 5. 3D layered isometric technical wireframe visual with 4 domain nodes
 */
export default function DossierHero({
  tabLabel,
  tabSublabel,
  pills,
  highlightText,
  headlineRest,
  missionTitle,
  missionBody,
  topNodes,
  midTierLabel,
  baseTierLabel,
  accentColor = '#CEFF00',
  scrollTargetId,
  className = '',
}: DossierHeroProps) {
  const handleScrollDown = () => {
    if (scrollTargetId) {
      const target = document.getElementById(scrollTargetId);
      if (target) {
        target.scrollIntoView({ behavior: 'smooth' });
        return;
      }
    }
    // Fallback: smooth scroll down 650px
    window.scrollBy({ top: 650, behavior: 'smooth' });
  };

  return (
    <DossierFrame
      tabLabel={tabLabel}
      tabSublabel={tabSublabel}
      pills={pills}
      className={className}
    >
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-8 items-center">
        {/* =============================================================== */}
        {/* LEFT COLUMN: TYPOGRAPHY, BRACKETED DOSSIER BOX & ACTIONS        */}
        {/* =============================================================== */}
        <div className="lg:col-span-6 xl:col-span-7 flex flex-col justify-center">
          {/* Main Headline with High-Contrast Highlight Badge */}
          <div className="text-3xl sm:text-4xl md:text-5xl lg:text-[3.25rem] leading-[1.12] tracking-tight font-sans font-medium text-white mb-2">
            <span className="inline-block bg-[#131E33] border border-[#223659] text-[#7DD3FC] px-3.5 py-1 rounded-sm mr-2 mb-2 font-mono text-2xl sm:text-3xl md:text-4xl">
              {highlightText}
            </span>
            <span className="block text-white/95 mt-1 font-sans">
              {headlineRest}
              <span className="inline-block w-2.5 h-8 sm:h-10 ml-1.5 bg-[#7DD3FC] align-middle animate-pulse" />
            </span>
          </div>

          {/* 4-Corner Hairline Bracketed Mission Box */}
          <div className="relative p-6 sm:p-7 max-w-xl my-6 bg-white/[0.02] border border-white/5 backdrop-blur-sm">
            {/* Top-Left Bracket */}
            <span className="absolute top-0 left-0 w-3.5 h-3.5 border-t-2 border-l-2 border-white/40" />
            {/* Top-Right Bracket */}
            <span className="absolute top-0 right-0 w-3.5 h-3.5 border-t-2 border-r-2 border-white/40" />
            {/* Bottom-Left Bracket */}
            <span className="absolute bottom-0 left-0 w-3.5 h-3.5 border-b-2 border-l-2 border-white/40" />
            {/* Bottom-Right Bracket */}
            <span className="absolute bottom-0 right-0 w-3.5 h-3.5 border-b-2 border-r-2 border-white/40" />

            <h3 className="text-base sm:text-lg font-normal text-white mb-2.5 font-serif sm:font-sans tracking-tight">
              {missionTitle}
            </h3>
            <p className="text-xs sm:text-sm text-white/65 leading-relaxed font-light font-mono sm:font-sans">
              {missionBody}
            </p>
          </div>

          {/* Bottom Hairline Circular Down-Arrow Button */}
          <div className="pt-2">
            <button
              type="button"
              onClick={handleScrollDown}
              className="group inline-flex items-center justify-center w-10 h-10 rounded-full border border-white/20 hover:border-white/60 bg-white/[0.03] hover:bg-white/[0.08] text-white/60 hover:text-white transition-all duration-200 cursor-pointer focus:outline-none focus:ring-2 focus:ring-white/40"
              aria-label="Scroll to next section"
              title="Explore overview"
            >
              <ArrowDown className="w-4 h-4 group-hover:translate-y-0.5 transition-transform" />
            </button>
          </div>
        </div>

        {/* =============================================================== */}
        {/* RIGHT COLUMN: 3D LAYERED ISOMETRIC WIREFRAME VISUAL            */}
        {/* =============================================================== */}
        <div className="lg:col-span-6 xl:col-span-5 flex items-center justify-center">
          <DossierIsometricStack
            topNodes={topNodes}
            midTierLabel={midTierLabel}
            baseTierLabel={baseTierLabel}
            accentColor={accentColor}
          />
        </div>
      </div>
    </DossierFrame>
  );
}
