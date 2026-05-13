'use client';

import type { CSSProperties } from 'react';
import { motion, useReducedMotion } from 'framer-motion';

export function LoadingGlyph({ className = '' }: { className?: string }) {
  const shouldReduceMotion = useReducedMotion() ?? false;

  return (
    <svg
      viewBox="0 0 120 120"
      aria-hidden="true"
      className={className}
    >
      <motion.circle
        cx="60"
        cy="60"
        r="45"
        fill="none"
        stroke="var(--desk-border)"
        strokeWidth="4"
        initial={false}
        animate={
          shouldReduceMotion
            ? { opacity: 0.4, scale: 1 }
            : { opacity: [0.2, 0.5, 0.2], scale: [0.98, 1.02, 0.98] }
        }
        transition={{ duration: 2.8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.path
        d="M18 79C35 66 54 60 76 62C89 64 98 62 106 56"
        fill="none"
        stroke="var(--desk-accent)"
        strokeOpacity="0.22"
        strokeWidth="3"
        strokeLinecap="round"
        initial={false}
        animate={
          shouldReduceMotion
            ? { opacity: 0.22, pathLength: 1 }
            : { opacity: [0.1, 0.3, 0.1], pathLength: [0.7, 1, 0.7] }
        }
        transition={{ duration: 2.4, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.rect
        x="28"
        y="34"
        width="64"
        height="44"
        rx="14"
        fill="var(--desk-surface-subtle)"
        stroke="var(--desk-border)"
        strokeWidth="2"
        initial={false}
        animate={shouldReduceMotion ? { y: 0 } : { y: [0, -2, 0] }}
        transition={{ duration: 2.2, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.rect
        x="40"
        y="48"
        width="36"
        height="6"
        rx="3"
        fill="var(--desk-border-strong)"
        initial={false}
        animate={shouldReduceMotion ? { opacity: 0.9 } : { opacity: [0.45, 1, 0.45] }}
        transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.rect
        x="40"
        y="60"
        width="24"
        height="5"
        rx="2.5"
        fill="var(--desk-border)"
        initial={false}
        animate={shouldReduceMotion ? { opacity: 0.7 } : { opacity: [0.3, 0.7, 0.3] }}
        transition={{ duration: 1.8, repeat: Infinity, delay: 0.12, ease: 'easeInOut' }}
      />
      {[0, 1, 2].map((index) => (
        <motion.circle
          key={index}
          cx={47 + index * 13}
          cy="25"
          r="3.5"
          fill="var(--desk-accent)"
          initial={false}
          animate={
            shouldReduceMotion
              ? { opacity: 0.75, scale: 1 }
              : { opacity: [0.35, 1, 0.35], scale: [1, 1.18, 1] }
          }
          transition={{ duration: 1.7, repeat: Infinity, delay: index * 0.14, ease: 'easeInOut' }}
        />
      ))}
    </svg>
  );
}

function PageSkeletonLead({
  sublineWidth,
}: {
  sublineWidth: string;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0 flex-1 space-y-3">
        <div className="skeleton skeleton-header" />
        <div className="skeleton" style={{ height: 14, width: sublineWidth, borderRadius: 6 }} />
      </div>
      <LoadingGlyph className="hidden h-16 w-16 shrink-0 md:block" />
    </div>
  );
}

export function Skeleton({ className = '', style }: { className?: string; style?: CSSProperties }) {
  return <div className={`skeleton-shimmer md-shape-md ${className}`} style={style} />;
}

export function SkeletonText({ lines = 3, className = '' }: { lines?: number; className?: string }) {
  return (
    <div className={className}>
      {Array.from({ length: lines }).map((_, i) => (
        <div
          key={i}
          className="skeleton-shimmer md-skeleton-text md-shape-sm"
          style={i === lines - 1 ? { width: '60%' } : undefined}
        />
      ))}
    </div>
  );
}

export function SkeletonCard({ className = '' }: { className?: string }) {
  return <div className={`skeleton-shimmer md-skeleton-card md-shape-lg ${className}`} />;
}

/** Full-page loading skeleton matching common dashboard layouts */

/**
 * M3 Page-level loading skeleton.
 * Used by loading.tsx files across all routes.
 */
export function PageSkeleton({
  variant = 'dashboard',
}: {
  variant?: 'dashboard' | 'table' | 'form' | 'map';
}) {
  if (variant === 'map') {
    return (
      <div className="page-skeleton flex items-center justify-center" style={{ padding: '0', height: '100%' }}>
        <div className="relative flex h-full min-h-screen w-full items-center justify-center overflow-hidden">
          <div className="skeleton absolute inset-0" style={{ borderRadius: 0 }} />
          <LoadingGlyph className="relative z-10 h-24 w-24" />
        </div>
      </div>
    );
  }

  if (variant === 'form') {
    return (
      <div className="page-skeleton" style={{ maxWidth: 640, margin: '0 auto' }}>
        <PageSkeletonLead sublineWidth="60%" />
        <div className="flex flex-col gap-4 mt-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2">
              <div className="skeleton" style={{ height: 12, width: 100, borderRadius: 4 }} />
              <div className="skeleton" style={{ height: 48, borderRadius: 4 }} />
            </div>
          ))}
        </div>
        <div className="skeleton" style={{ height: 40, width: 140, borderRadius: 9999, marginTop: 16 }} />
      </div>
    );
  }

  if (variant === 'table') {
    return (
      <div className="page-skeleton">
        <PageSkeletonLead sublineWidth="45%" />
        <div className="skeleton-kpi-grid mt-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="skeleton skeleton-kpi" />
          ))}
        </div>
        <div className="skeleton skeleton-table mt-2" />
      </div>
    );
  }

  // Default: dashboard
  return (
    <div className="page-skeleton">
      <PageSkeletonLead sublineWidth="50%" />
      <div className="skeleton-kpi-grid mt-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="skeleton skeleton-kpi" />
        ))}
      </div>
      <div className="skeleton" style={{ height: 280, borderRadius: 12 }} />
      <div className="skeleton skeleton-table" />
    </div>
  );
}
