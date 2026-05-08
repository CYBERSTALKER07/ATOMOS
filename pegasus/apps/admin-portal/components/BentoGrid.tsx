'use client';

import { type ReactNode } from 'react';
import { motion } from 'framer-motion';

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
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = '',
  interactive = true,
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
        hidden: { opacity: 0, y: 20, scale: 0.98 },
        show: { 
          opacity: 1, 
          y: 0, 
          scale: 1,
          transition: {
            type: "spring",
            stiffness: 260,
            damping: 20
          }
        }
      }}
      whileHover={interactive ? { 
        y: -4, 
        scale: 1.002,
        transition: { duration: 0.2, ease: "easeOut" }
      } : undefined}
      className={`bento-card glass-premium ${sizeClass} ${interactive ? 'hover:shadow-2xl hover:border-white/20' : ''} ${className} relative overflow-hidden`}
    >
      <div className="relative h-full w-full overflow-hidden rounded-[inherit] z-10">
        {children}
      </div>
      
      {/* Subtle corner highlight */}
      <div className="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-white/5 to-transparent pointer-events-none" />
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
    <div className={`bento-skeleton animate-pulse bg-muted/20 ${sizeClass} ${className} md-shape-lg`} />
  );
}
