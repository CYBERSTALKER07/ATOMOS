'use client';

import { type ReactNode } from 'react';
import { motion } from 'framer-motion';

interface BentoGridProps {
  children: ReactNode;
  className?: string;
  theme?: 'brutalist' | 'apple';
  staggerChildren?: number;
}

export function BentoGrid({
  children,
  className = '',
  theme = 'apple',
  staggerChildren = 0.05,
}: BentoGridProps) {
  const themeClass = theme === 'apple' ? 'bento-apple' : '';

  return (
    <motion.div
      initial="hidden"
      whileInView="show"
      viewport={{ once: true, margin: '-50px' }}
      variants={{
        hidden: {},
        show: {
          transition: {
            staggerChildren,
          },
        },
      }}
      className={`bento-grid ${themeClass} ${className}`}
    >
      {children}
    </motion.div>
  );
}

type BentoSize = 'stat' | 'anchor' | 'list' | 'control' | 'wide' | 'full';

interface BentoCardProps {
  children: ReactNode;
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
  interactive?: boolean;
  delay?: number;
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = '',
  interactive = true,
  delay = 0,
}: BentoCardProps) {
  const sizeClass = size ? `bento-${size}` : `bento-span-${span}${rowSpan ? ' bento-row-2' : ''}`;

  return (
    <motion.div
      variants={{
        hidden: { opacity: 0, y: 12 },
        show: {
          opacity: 1,
          y: 0,
          transition: {
            type: 'spring',
            stiffness: 260,
            damping: 24,
            delay: delay / 1000,
          },
        },
      }}
      className={`bento-card ${interactive ? 'active-press' : ''} ${sizeClass} ${className} relative overflow-hidden`}
    >
      <div className="relative h-full w-full overflow-hidden rounded-[inherit] z-10">
        {children}
      </div>
    </motion.div>
  );
}

interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}

export function BentoSkeleton({ size, span = 1, rowSpan = false, className = '' }: BentoSkeletonProps) {
  const sizeClass = size ? `bento-${size}` : `bento-span-${span}${rowSpan ? ' bento-row-2' : ''}`;

  return (
    <div
      className={`bento-skeleton animate-pulse ${sizeClass} ${className}`}
      style={{
        background: 'var(--desk-canvas)',
        border: '1px solid var(--desk-border)',
        borderRadius: 'var(--radius-lg)',
      }}
    />
  );
}
