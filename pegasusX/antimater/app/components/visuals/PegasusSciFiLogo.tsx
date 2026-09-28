'use client';

import React from 'react';

/**
 * PEGASUS Futuristic Sci-Fi Vector Wordmark
 * Engineered with the exact geometric visual language of Terafab (Tesla/SpaceX aerospace design system).
 * 
 * Features:
 * - Mathematical squircle curvature (outer radius 17.485, inner radius 9.885)
 * - Uniform stroke weight (7.600 units across 55.506 height grid)
 * - Iconic inverted chevron 'A' without crossbar
 * - Geometric squircle 'P', 'E', 'G', 'S', 'U'
 * - Multiple presentation modes: solid, neon glow (bridge sign style), retractable nav, and standalone badge
 */

export const PEGASUS_GLYPHS = {
  // P (Width: 89.719, Height: 55.506)
  P: 'M0 0H72.234C81.874 0 89.719 7.8455 89.719 17.4853C89.719 27.1251 81.874 34.9707 72.234 34.9707H7.607V55.4987H0V0ZM7.607 7.59995V27.3707H72.234C77.685 27.3707 82.119 22.9363 82.119 17.4853C82.119 12.0344 77.685 7.59995 72.234 7.59995H7.607Z',
  
  // E (Exact normalized from Terafab design system)
  E: 'M17.485 7.59995H89.719V0H17.485C7.845 0 0 7.84554 0 17.4853V38.0134C0 47.6532 7.845 55.4987 17.485 55.4987H89.719V47.8988H17.485C12.034 47.8988 7.6 43.4643 7.6 38.0134V31.5527H76.026V23.9528H7.6V17.4922C7.6 12.0412 12.034 7.60677 17.485 7.60677V7.59995Z',
  
  // G (Squircle with precision mid-height spur)
  G: 'M17.485 7.59995H89.719V0H17.485C7.845 0 0 7.84554 0 17.4853V38.0134C0 47.6532 7.845 55.4987 17.485 55.4987H72.234C81.874 55.4987 89.719 47.6532 89.719 38.0134V23.9528H48.0V31.5527H82.119V38.0134C82.119 43.464 77.685 47.8988 72.234 47.8988H17.485C12.034 47.8988 7.6 43.464 7.6 38.0134V17.4922C7.6 12.0412 12.034 7.60677 17.485 7.60677V7.59995Z',
  
  // A (Signature inverted chevron apex without crossbar)
  A: 'M8.657 55.5055L31.962 12.7576C33.695 9.5784 37.024 7.60678 40.64 7.60678H49.079C52.702 7.60678 56.024 9.5784 57.757 12.7576L81.061 55.5055H89.719L64.436 9.12131C61.373 3.49981 55.485 0.00683594 49.079 0.00683594H40.64C34.234 0.00683594 28.353 3.49981 25.283 9.12131L0 55.5055H8.657Z',
  
  // S (Curvilinear squircle S with balanced horizontal terminals)
  S: 'M17.485 0H89.719V7.59995H17.485C12.034 7.59995 7.6 12.0412 7.6 17.4922C7.6 22.9432 12.034 27.378 17.485 27.378H72.234C81.874 27.378 89.719 33.623 89.719 41.863C89.719 50.400 82.874 55.4987 72.234 55.4987H0V47.8988H72.234C77.685 47.8988 82.119 44.4643 82.119 41.863C82.119 38.0134 77.685 34.978 72.234 34.978H17.485C7.845 34.978 0 28.132 0 17.4922C0 7.84554 7.845 0 17.485 0Z',
  
  // U (Symmetrical squircle capsule)
  U: 'M0 0H7.607V38.0134C7.607 43.4643 12.034 47.8988 17.485 47.8988H72.234C77.685 47.8988 82.119 43.464 82.119 38.0134V0H89.719V38.0134C89.719 47.6532 81.874 55.4987 72.234 55.4987H17.485C7.845 55.4987 0 47.6532 0 38.0134V0Z',
};

// Coordinate offsets for "P E G A S U S"
// Total width = 772.034, height = 55.506 (~56)
export const PEGASUS_LETTER_OFFSETS = [
  { char: 'P', x: 0.0 },
  { char: 'E', x: 115.719 },
  { char: 'G', x: 231.438 },
  { char: 'A', x: 341.157 },
  { char: 'S', x: 450.876 },
  { char: 'U', x: 566.595 },
  { char: 'S', x: 682.314 },
];

export interface PegasusSciFiLogoProps {
  variant?: 'solid' | 'neon' | 'badge' | 'retractable';
  className?: string;
  glowColor?: string;
  color?: string;
  height?: number | string;
  retracted?: boolean;
}

export default function PegasusSciFiLogo({
  variant = 'solid',
  className = '',
  glowColor = '#38bdf8',
  color = 'currentColor',
  height = '1.75rem',
  retracted = false,
}: PegasusSciFiLogoProps) {
  // 1. Standalone Badge ('P' in square rounded badge)
  if (variant === 'badge') {
    return (
      <svg
        viewBox="0 0 90 56"
        fill={color}
        className={className}
        style={{ height }}
        xmlns="http://www.w3.org/2000/svg"
        aria-label="Pegasus P Badge"
      >
        <path d={PEGASUS_GLYPHS.P} fill={color} />
      </svg>
    );
  }

  // 2. Retractable Nav Logo (P is fixed, EGASUS collapses smoothly)
  if (variant === 'retractable') {
    return (
      <div
        className={`inline-flex items-center overflow-hidden transition-all duration-500 ease-out ${className}`}
        style={{ height }}
        aria-label="Pegasus"
      >
        {/* P glyph */}
        <svg
          viewBox="0 0 90 56"
          fill={color}
          className="shrink-0 transition-transform duration-300"
          style={{ height: '100%', aspectRatio: '90 / 56' }}
          xmlns="http://www.w3.org/2000/svg"
        >
          <path d={PEGASUS_GLYPHS.P} fill={color} />
        </svg>

        {/* EGASUS Track */}
        <div
          className="overflow-hidden transition-all duration-500 ease-[cubic-bezier(0.22,1,0.36,1)]"
          style={{
            maxWidth: retracted ? '0px' : '300px',
            opacity: retracted ? 0 : 1,
            marginLeft: retracted ? '0px' : '8px',
          }}
        >
          <svg
            viewBox="115 0 657 56"
            fill={color}
            className="shrink-0"
            style={{ height: '100%', aspectRatio: '657 / 56' }}
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d={PEGASUS_GLYPHS.E} transform="translate(115.719, 0)" fill={color} />
            <path d={PEGASUS_GLYPHS.G} transform="translate(231.438, 0)" fill={color} />
            <path d={PEGASUS_GLYPHS.A} transform="translate(341.157, 0)" fill={color} />
            <path d={PEGASUS_GLYPHS.S} transform="translate(450.876, 0)" fill={color} />
            <path d={PEGASUS_GLYPHS.U} transform="translate(566.595, 0)" fill={color} />
            <path d={PEGASUS_GLYPHS.S} transform="translate(682.314, 0)" fill={color} />
          </svg>
        </div>
      </div>
    );
  }

  // 3. Cyberpunk Neon Luminescence (Matching bridge sign in reference screenshot)
  if (variant === 'neon') {
    return (
      <div className={`relative inline-block ${className}`} style={{ height }}>
        <svg
          viewBox="0 0 773 56"
          className="w-full h-full drop-shadow-[0_0_12px_rgba(56,189,248,0.7)] drop-shadow-[0_0_36px_rgba(56,189,248,0.35)]"
          xmlns="http://www.w3.org/2000/svg"
          aria-label="Pegasus Neon Logo"
        >
          {PEGASUS_LETTER_OFFSETS.map(({ char, x }) => (
            <path
              key={`${char}-${x}`}
              d={PEGASUS_GLYPHS[char as keyof typeof PEGASUS_GLYPHS]}
              transform={`translate(${x}, 0)`}
              fill={glowColor}
            />
          ))}
        </svg>
      </div>
    );
  }

  // 4. Default: Pristine Sci-Fi Vector Wordmark (Solid White / currentColor)
  return (
    <svg
      viewBox="0 0 773 56"
      fill={color}
      className={`inline-block shrink-0 ${className}`}
      style={{ height }}
      xmlns="http://www.w3.org/2000/svg"
      aria-label="Pegasus Sci-Fi Logo"
    >
      {PEGASUS_LETTER_OFFSETS.map(({ char, x }) => (
        <path
          key={`${char}-${x}`}
          d={PEGASUS_GLYPHS[char as keyof typeof PEGASUS_GLYPHS]}
          transform={`translate(${x}, 0)`}
          fill={color}
        />
      ))}
    </svg>
  );
}
