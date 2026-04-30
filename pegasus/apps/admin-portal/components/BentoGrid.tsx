'use client';

import { useRef, useEffect, useState, type ReactNode } from 'react';

// ── Bento Grid Container ────────────────────────────────────────────────────

interface BentoGridProps {
  children: ReactNode;
  className?: string;
  /** Visual theme: 'brutalist' = 0 radius (default), 'apple' = 24px radius */
  theme?: 'brutalist' | 'apple';
}

export function BentoGrid({ children, className = '', theme = 'brutalist' }: BentoGridProps) {
  const themeClass = theme === 'apple' ? 'bento-apple' : '';
  return (
    <div className={`bento-grid ${themeClass} ${className}`}>
      {children}
    </div>
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
  /** Stagger delay for reveal animation (ms) */
  delay?: number;
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = '',
  delay = 0,
}: BentoCardProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Build size class — semantic size takes priority over legacy span/rowSpan
  let sizeClass: string;
  if (size) {
    sizeClass = `bento-${size}`;
  } else {
    sizeClass = `bento-span-${span}${rowSpan ? ' bento-row-2' : ''}`;
  }

  return (
    <div
      ref={ref}
      className={`bento-card ${sizeClass} ${className}`}
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0) scale(1)' : 'translateY(12px) scale(0.98)',
        transition: `opacity 0.4s cubic-bezier(0.05, 0.7, 0.1, 1) ${delay}ms, transform 0.4s cubic-bezier(0.05, 0.7, 0.1, 1) ${delay}ms`,
      }}
    >
      {children}
    </div>
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

  return <div className={`bento-skeleton ${sizeClass} ${className}`} />;
}
