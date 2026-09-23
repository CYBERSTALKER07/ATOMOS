'use client';

import React from 'react';

/**
 * Partner & Ecosystem Brand Strip
 * Matches the bottom circle in the Terafab reference screenshot (Tesla, SpaceX style wide-spaced brandmarks).
 */

export interface PartnerBrandStripProps {
  className?: string;
  showDivider?: boolean;
  label?: string;
}

export const PARTNERS = [
  { name: 'TESLA', tracking: 'tracking-[0.35em]' },
  { name: 'SPACEX', tracking: 'tracking-[0.3em]' },
  { name: 'SPANNER', tracking: 'tracking-[0.32em]' },
  { name: 'ORBITAL', tracking: 'tracking-[0.3em]' },
  { name: 'MAERSK', tracking: 'tracking-[0.35em]' },
  { name: 'NVIDIA', tracking: 'tracking-[0.32em]' },
];

export default function PartnerBrandStrip({
  className = '',
  showDivider = true,
  label,
}: PartnerBrandStripProps) {
  return (
    <div className={`w-full ${className}`}>
      {showDivider && (
        <div className="w-full h-[1px] bg-white/[0.08] mb-6" />
      )}
      
      {label && (
        <p className="text-[10px] uppercase font-mono tracking-[0.25em] text-white/40 mb-4">
          {label}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-x-8 gap-y-4 sm:gap-x-12">
        {PARTNERS.map((partner) => (
          <span
            key={partner.name}
            className={`font-mono text-xs sm:text-sm font-light text-white/40 hover:text-white/90 transition-colors cursor-default select-none ${partner.tracking}`}
          >
            {partner.name}
          </span>
        ))}
      </div>
    </div>
  );
}
