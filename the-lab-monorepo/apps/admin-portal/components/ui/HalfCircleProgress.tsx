'use client';

import React from 'react';

/**
 * A visually appealing Half Circle Progress Ring for Dashboards
 */
export default function HalfCircleProgress({
  value,
  max = 100,
  label,
  subtitle,
  color = 'var(--accent)',
}: {
  value: number;
  max?: number;
  label: string;
  subtitle?: string;
  color?: string;
}) {
  const percentage = Math.min(value / max, 1);
  const radius = 100;
  const strokeWidth = 24;
  
  // Circumference of semi-circle: pi * r
  const circumference = Math.PI * radius;
  const offset = circumference - percentage * circumference;

  return (
    <div className="relative flex flex-col items-center justify-center p-6 md-card-elevated bg-[var(--surface)]">
      <svg
        viewBox="0 0 240 130"
        className="w-full max-w-[240px] overflow-visible"
        style={{ transform: 'rotateY(180deg)' }}
      >
        {/* Background Track */}
        <path
          d={`M ${strokeWidth}, 120 A ${radius}, ${radius} 0 0, 1 ${240 - strokeWidth}, 120`}
          fill="none"
          stroke="var(--surface)"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />

        {/* Progress Track */}
        <path
          d={`M ${strokeWidth}, 120 A ${radius}, ${radius} 0 0, 1 ${240 - strokeWidth}, 120`}
          fill="none"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          style={{ transition: 'stroke-dashoffset 1s cubic-bezier(0.2, 0, 0, 1)' }}
        />
      </svg>

      <div className="absolute bottom-6 flex flex-col items-center text-center fade-in-up delay-100">
        <span className="text-4xl font-bold text-[var(--foreground)]">
          {value.toLocaleString()}
          <span className="text-xl text-[var(--muted)] font-normal ml-1">
            / {max.toLocaleString()}
          </span>
        </span>
        <span className="text-sm font-medium tracking-wide text-[var(--accent)] mt-1 uppercase">
          {label}
        </span>
        {subtitle && (
          <span className="text-xs text-[var(--muted)] mt-1 max-w-[150px]">
            {subtitle}
          </span>
        )}
      </div>
    </div>
  );
}