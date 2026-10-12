'use client';

import React, { useRef, useCallback, useEffect } from 'react';
import { cn } from '@/lib/utils';

export interface SmoothCursorProps {
  className?: string;
  pointsCount?: number;
  lineWidth?: number;
  springStrength?: number;
  dampening?: number;
  color?: string;
  blur?: number;
  mixBlendMode?: GlobalCompositeOperation;
  velocityScale?: boolean;
  trailOpacity?: number;
  smoothFactor?: number;
}

interface TrailPoint {
  x: number;
  y: number;
  vx: number;
  vy: number;
}

export default function SmoothCursor({
  className = '',
  pointsCount = 40,
  lineWidth = 0.3,
  springStrength = 0.4,
  dampening = 0.5,
  color = '#000000',
  blur = 0,
  mixBlendMode = 'source-over',
  velocityScale = false,
  trailOpacity = 1,
  smoothFactor = 1
}: SmoothCursorProps) {
  const friction = Math.min(Math.max(dampening, 0.1), 0.99);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const ctxRef = useRef<CanvasRenderingContext2D | null>(null);
  const pointsRef = useRef<TrailPoint[]>([]);
  const targetPos = useRef<{ x: number; y: number }>({ x: 0, y: 0 });
  const isInside = useRef<boolean>(true);
  const presence = useRef<number>(1);
  const rafId = useRef<number | null>(null);
  const renderFrameRef = useRef<(() => void) | null>(null);

  const requestFrame = useCallback(() => {
    rafId.current = requestAnimationFrame(() => {
      renderFrameRef.current?.();
    });
  }, []);

  useEffect(() => {
    const el = canvasRef.current;
    if (el) {
      el.style.filter = blur > 0 ? `blur(${blur}px)` : 'none';
    }
  }, [blur]);

  const initPoints = useCallback(() => {
    const el = canvasRef.current;
    if (!el) return;
    el.width = el.offsetWidth;
    el.height = el.offsetHeight;
    targetPos.current = { x: 0.5 * el.offsetWidth, y: 0.5 * el.offsetHeight };
    pointsRef.current = Array.from({ length: pointsCount }, () => ({
      x: targetPos.current.x,
      y: targetPos.current.y,
      vx: 0,
      vy: 0
    }));
  }, [pointsCount]);

  const setTarget = useCallback((x: number, y: number) => {
    targetPos.current.x = x;
    targetPos.current.y = y;
  }, []);

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      const el = canvasRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      setTarget(e.clientX - rect.left, e.clientY - rect.top);
    },
    [setTarget]
  );

  const onTouchMove = useCallback(
    (e: TouchEvent) => {
      const el = canvasRef.current;
      if (!el || e.touches.length === 0) return;
      const rect = el.getBoundingClientRect();
      setTarget(e.touches[0].clientX - rect.left, e.touches[0].clientY - rect.top);
    },
    [setTarget]
  );

  const onClick = useCallback(
    (e: MouseEvent) => {
      const el = canvasRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      setTarget(e.clientX - rect.left, e.clientY - rect.top);
    },
    [setTarget]
  );

  const onMouseEnter = useCallback(() => {
    isInside.current = true;
  }, []);

  const onMouseLeave = useCallback(() => {
    isInside.current = false;
  }, []);

  useEffect(() => {
    renderFrameRef.current = () => {
      const canvas = canvasRef.current;
      const ctx = ctxRef.current;
      if (!canvas || !ctx) return;

      ctx.clearRect(0, 0, canvas.width, canvas.height);

      const pts = pointsRef.current;
      pts.forEach((p, idx) => {
        const leader = idx === 0 ? targetPos.current : pts[idx - 1];
        const strength = idx === 0 ? 0.4 * springStrength : springStrength;
        p.vx += (leader.x - p.x) * strength;
        p.vy += (leader.y - p.y) * strength;
        p.vx *= friction;
        p.vy *= friction;
        p.x += p.vx;
        p.y += p.vy;
      });

      const targetVal = isInside.current ? 1 : 0;
      presence.current += (targetVal - presence.current) * 0.15;

      ctx.globalCompositeOperation = mixBlendMode;
      ctx.globalAlpha = presence.current * trailOpacity;
      ctx.strokeStyle = color;
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';

      if (pts.length > 2) {
        ctx.beginPath();
        ctx.moveTo(pts[0].x, pts[0].y);

        for (let i = 1; i < pts.length - 1; i++) {
          let vScale = 1;
          if (velocityScale && i > 0) {
            const p = pts[i];
            vScale = 1 + Math.min(0.5 * Math.sqrt(p.vx * p.vx + p.vy * p.vy), 2);
          }
          const midX = 0.5 * (pts[i].x + pts[i + 1].x) * smoothFactor + pts[i].x * (1 - smoothFactor);
          const midY = 0.5 * (pts[i].y + pts[i + 1].y) * smoothFactor + pts[i].y * (1 - smoothFactor);

          ctx.quadraticCurveTo(pts[i].x, pts[i].y, midX, midY);
          ctx.lineWidth = lineWidth * (pointsCount - i) * vScale;
          ctx.stroke();
        }

        ctx.lineTo(pts[pts.length - 1].x, pts[pts.length - 1].y);
        ctx.stroke();
      }

      ctx.globalAlpha = 1;
      ctx.globalCompositeOperation = 'source-over';
      requestFrame();
    };
  }, [
    pointsCount,
    lineWidth,
    springStrength,
    friction,
    color,
    blur,
    mixBlendMode,
    velocityScale,
    trailOpacity,
    smoothFactor,
    requestFrame
  ]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    ctxRef.current = ctx;
    initPoints();
    requestFrame();

    window.addEventListener('mousemove', onMouseMove, { passive: true });
    window.addEventListener('touchmove', onTouchMove, { passive: true });
    window.addEventListener('click', onClick, { passive: true });
    window.addEventListener('resize', initPoints);
    document.addEventListener('mouseenter', onMouseEnter);
    document.addEventListener('mouseleave', onMouseLeave);

    return () => {
      if (rafId.current !== null) cancelAnimationFrame(rafId.current);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('touchmove', onTouchMove);
      window.removeEventListener('click', onClick);
      window.removeEventListener('resize', initPoints);
      document.removeEventListener('mouseenter', onMouseEnter);
      document.removeEventListener('mouseleave', onMouseLeave);
    };
  }, [onMouseMove, onTouchMove, onClick, initPoints, onMouseEnter, onMouseLeave, requestFrame]);

  return (
    <div className={cn('fixed inset-0 w-full h-full pointer-events-none z-9999', className)}>
      <canvas ref={canvasRef} className="w-full h-full block" />
    </div>
  );
}
