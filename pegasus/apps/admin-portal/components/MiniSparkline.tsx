'use client';

import { useMemo } from 'react';

interface MiniSparklineProps {
  data: number[];
  width?: number;
  height?: number;
  color?: string;
  className?: string;
}

export default function MiniSparkline({ data, width = 80, height = 32, color, className = '' }: MiniSparklineProps) {
  const parsedWidth = typeof width === 'number' ? width : Number(width);
  const parsedHeight = typeof height === 'number' ? height : Number(height);
  const safeWidth = Number.isFinite(parsedWidth) && parsedWidth > 0 ? parsedWidth : 80;
  const safeHeight = Number.isFinite(parsedHeight) && parsedHeight > 0 ? parsedHeight : 32;

  const path = useMemo(() => {
    if (!data.length) return '';
    const max = Math.max(...data);
    const min = Math.min(...data);
    const range = max - min || 1;
    const step = safeWidth / Math.max(data.length - 1, 1);
    return data
      .map((v, i) => {
        const x = i * step;
        const y = safeHeight - ((v - min) / range) * (safeHeight - 4) - 2;
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  }, [data, safeWidth, safeHeight]);

  if (!data.length) return null;

  return (
    <svg width={safeWidth} height={safeHeight} className={className} style={{ overflow: 'visible' }}>
      <path
        d={path}
        fill="none"
        stroke={color || 'var(--muted)'}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
