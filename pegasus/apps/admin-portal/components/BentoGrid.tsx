'use client';

import { type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { LoadingGlyph } from './Skeleton';

// ── Bento Grid Container ────────────────────────────────────────────────────

interface BentoGridProps {
  children: ReactNode;
  className?: string;
  /** Visual theme: 'brutalist' = sharp, 'apple' = premium rounded (default) */
  theme?: 'brutalist' | 'apple';
  staggerChildren?: number;
}

export function BentoGrid({ 
  children, 
  className = '', 
  theme = 'apple',
  staggerChildren = 0.05 
}: BentoGridProps) {
  const themeClass = theme === 'apple' ? 'bento-apple' : '';
  
  return (
    <motion.div 
      initial="hidden"
      whileInView="show"
      viewport={{ once: true, margin: "-50px" }}
      variants={{
        hidden: {},
        show: {
          transition: {
            staggerChildren: staggerChildren
          }
        }
      }}
      className={`bento-grid ${themeClass} ${className}`}
    >
      {children}
    </motion.div>
  );
}

// ── Bento Card ──────────────────────────────────────────────────────────────

type BentoSize = 'stat' | 'anchor' | 'list' | 'control' | 'wide' | 'full';

interface BentoCardProps {
  children: ReactNode;
  /** Semantic size: stat (1×1), anchor (2×2), list (1×2), control (2×1), wide (2×1), full (full-width) */
  size?: BentoSize;
  /** Legacy span override (1-4 columns) */
  span?: 1 | 2 | 3 | 4;
  /** Legacy row span */
  rowSpan?: boolean;
  className?: string;
  /** Interaction: enable premium hover lift/glow */
  interactive?: boolean;
  /** Optional stagger delay in ms (legacy prop, accepted for backwards compat) */
  delay?: number;
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = '',
}: BentoCardProps) {
  // Build size class — semantic size takes priority over legacy span/rowSpan
  let sizeClass: string;
  if (size) {
    sizeClass = `bento-${size}`;
  } else {
    sizeClass = `bento-span-${span}${rowSpan ? ' bento-row-2' : ''}`;
  }

  return (
    <motion.div
      variants={{
        hidden: { opacity: 0, y: 12 },
        show: { 
          opacity: 1, 
          y: 0,
          transition: {
            type: "spring",
            stiffness: 260,
            damping: 24
          }
        }
      }}
      className={`bento-card ${sizeClass} ${className} relative overflow-hidden`}
    >
      <div className="relative h-full w-full overflow-hidden rounded-[inherit] z-10">
        {children}
      </div>
    </motion.div>
  );
}

// ── Bento Skeleton (per-cell placeholder) ───────────────────────────────────

interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}

export function BentoSkeleton({ size, span = 1, rowSpan = false, className = '' }: BentoSkeletonProps) {
  let sizeClass: string;
  if (size) {
    sizeClass = `bento-${size}`;
  } else {
    sizeClass = `bento-span-${span}${rowSpan ? ' bento-row-2' : ''}`;
  }

  return (
    <div
      className={`bento-skeleton ${sizeClass} ${className} relative overflow-hidden`}
      style={{
        background: 'var(--desk-canvas)',
        border: '1px solid var(--desk-border)',
        borderRadius: 'var(--radius-lg)',
      }}
    >
      <div className="absolute inset-0 skeleton-shimmer opacity-70" />
      <div className="relative z-10 flex h-full w-full items-center justify-center">
        <LoadingGlyph className="h-16 w-16 opacity-80 md:h-20 md:w-20" />
      </div>
    </div>
  );
}
