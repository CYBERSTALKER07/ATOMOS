'use client';

import React, { useEffect, useRef } from 'react';
import { useTheme } from '@/app/context/ThemeContext';
import { usePerfProfile } from '@/app/hooks/useDevice';

// 4x4 Bayer dithering matrix for authentic ordered halftoning
const BAYER_4X4 = [
  [ 0 / 16, 8 / 16, 2 / 16, 10 / 16],
  [12 / 16, 4 / 16, 14 / 16, 6 / 16],
  [ 3 / 16, 11 / 16, 1 / 16, 9 / 16],
  [15 / 16, 7 / 16, 13 / 16, 5 / 16],
];

interface AnimatedDitherFieldProps {
  className?: string;
  dotSpacing?: number;
  speed?: number;
}

interface DitherDot {
  x: number;
  y: number;
  vignette: number;
  threshold: number; // bayerThreshold * 0.7
  phase1: number;    // x * 0.045 + y * 0.018
  phase2: number;    // x * 0.025 + y * 0.05
}

interface PaletteItem {
  color: string;
  radius: number;
}

const PALETTE_STEPS = 16;

const createPalette = (isLight: boolean): PaletteItem[] => {
  const baseR = isLight ? 124 : 167;
  const baseG = isLight ? 58 : 139;
  const baseB = isLight ? 237 : 250;

  const peakR = isLight ? 99 : 224;
  const peakG = isLight ? 102 : 231;
  const peakB = isLight ? 241 : 255;

  const list: PaletteItem[] = [];
  for (let i = 0; i <= PALETTE_STEPS; i++) {
    const level = i / PALETTE_STEPS;
    const alpha = Math.max(0.12, Math.min(0.95, 0.2 + level * 0.75));
    const radius = 0.75 + level * 0.85;

    const red = Math.round(baseR + (peakR - baseR) * level);
    const green = Math.round(baseG + (peakG - baseG) * level);
    const blue = Math.round(baseB + (peakB - baseB) * level);

    list.push({
      color: `rgba(${red}, ${green}, ${blue}, ${alpha.toFixed(2)})`,
      radius,
    });
  }
  return list;
};

export default function AnimatedDitherField({
  className = '',
  dotSpacing = 11,
  speed = 0.9,
}: AnimatedDitherFieldProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const { resolvedTheme } = useTheme();
  const { prefersReducedMotion, isLowEnd, isMobile } = usePerfProfile();
  const isLight = resolvedTheme === 'light';

  // Mouse interaction coordinates (relative to canvas)
  const mouseRef = useRef<{ x: number; y: number; active: boolean }>({
    x: -1000,
    y: -1000,
    active: false,
  });

  useEffect(() => {
    const canvas = canvasRef.current;
    const container = containerRef.current;
    if (!canvas || !container) return;

    const ctx = canvas.getContext('2d', { alpha: true });
    if (!ctx) return;

    let animationFrameId = 0;
    let isVisible = false;
    let width = 0;
    let height = 0;
    let dpr = 1;
    let lastDrawTime = 0;

    const spacing = isMobile ? Math.max(12, dotSpacing) : Math.max(10, dotSpacing);
    const palette = createPalette(isLight);
    let dots: DitherDot[] = [];

    // Responsive canvas sizing and LUT precomputation
    const updateSize = () => {
      const rect = container.getBoundingClientRect();
      width = Math.max(rect.width, 100);
      height = Math.max(rect.height, 80);
      dpr = Math.min(window.devicePixelRatio || 1, 1.5);

      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;

      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      const cols = Math.ceil(width / spacing);
      const rows = Math.ceil(height / spacing);
      const newDots: DitherDot[] = [];

      for (let c = 0; c < cols; c++) {
        const x = c * spacing;
        const normX = x / width;
        const edgeFadeX = Math.sin(Math.PI * Math.min(Math.max(normX, 0), 1));

        for (let r = 0; r < rows; r++) {
          const y = r * spacing;
          const normY = y / height;
          const edgeFadeY = Math.sin(Math.PI * Math.min(Math.max(normY, 0), 1));
          const vignette = Math.pow(edgeFadeX * edgeFadeY, 0.45);

          // Discard dots that fade out completely at borders
          if (vignette < 0.02) continue;

          const bayerThreshold = BAYER_4X4[c % 4][r % 4];
          newDots.push({
            x,
            y,
            vignette,
            threshold: bayerThreshold * 0.7,
            phase1: x * 0.045 + y * 0.018,
            phase2: x * 0.025 + y * 0.05,
          });
        }
      }

      dots = newDots;
    };

    updateSize();

    // Render single frame of animated dither - completely zero allocations in hot loop
    const drawFrame = (timeSeconds: number) => {
      ctx.clearRect(0, 0, width, height);

      const t = timeSeconds * speed;
      const tPhase1 = t * 2.6;
      const tPhase2 = t * 1.5;
      const mouse = mouseRef.current;
      const hasMouse = mouse.active;
      const mx = mouse.x;
      const my = mouse.y;

      const len = dots.length;
      for (let i = 0; i < len; i++) {
        const dot = dots[i];

        // 1. Primary data flow wave: travels left to right
        const primaryWave = Math.sin(dot.phase1 - tPhase1);

        // 2. Secondary diagonal harmonic shimmer
        const crossWave = Math.cos(dot.phase2 - tPhase2);

        // 3. Mouse proximity ripple
        let mouseFactor = 0;
        if (hasMouse) {
          const dx = dot.x - mx;
          const dy = dot.y - my;
          const distSq = dx * dx + dy * dy;
          if (distSq < 10000) {
            mouseFactor = Math.exp(-distSq / 3200) * 0.55;
          }
        }

        // Combine wave intensities with precomputed static vignette
        const rawIntensity = 0.42 + 0.32 * primaryWave + 0.18 * crossWave + mouseFactor;
        const intensity = rawIntensity * dot.vignette;

        // 4. Ordered Bayer dither quantization
        if (intensity > dot.threshold) {
          const level = Math.min(1, (intensity - dot.threshold) / 0.6);
          const step = Math.min(PALETTE_STEPS, Math.max(0, (level * PALETTE_STEPS) | 0));
          const p = palette[step];

          ctx.fillStyle = p.color;
          const radius = p.radius;
          ctx.fillRect(dot.x - radius, dot.y - radius, radius * 2, radius * 2);
        }
      }
    };

    // Static render for users who prefer reduced motion or low-end
    if (prefersReducedMotion || isLowEnd) {
      drawFrame(1.5);
      return;
    }

    let startTime: number | null = null;

    const loop = (timestamp: number) => {
      if (!isVisible) {
        animationFrameId = 0;
        return;
      }

      animationFrameId = requestAnimationFrame(loop);

      // Throttle to 30 FPS for silky performance and minimal CPU usage
      if (timestamp - lastDrawTime < 33) return;
      lastDrawTime = timestamp;

      if (!startTime) startTime = timestamp;
      const elapsed = (timestamp - startTime) / 1000;
      drawFrame(elapsed);
    };

    const resizeObserver = new ResizeObserver(() => {
      updateSize();
      if (!isVisible) drawFrame(1.5);
    });
    resizeObserver.observe(container);

    // Pause animation when scrolled out of view to preserve battery & CPU
    const intersectionObserver = new IntersectionObserver(
      ([entry]) => {
        const wasVisible = isVisible;
        isVisible = entry?.isIntersecting ?? false;

        if (isVisible && !wasVisible) {
          if (animationFrameId === 0) {
            lastDrawTime = performance.now();
            animationFrameId = requestAnimationFrame(loop);
          }
        } else if (!isVisible && wasVisible) {
          if (animationFrameId !== 0) {
            cancelAnimationFrame(animationFrameId);
            animationFrameId = 0;
          }
        }
      },
      { rootMargin: '80px', threshold: 0.05 }
    );
    intersectionObserver.observe(container);

    return () => {
      if (animationFrameId !== 0) cancelAnimationFrame(animationFrameId);
      resizeObserver.disconnect();
      intersectionObserver.disconnect();
    };
  }, [dotSpacing, speed, prefersReducedMotion, isLowEnd, isMobile, isLight]);

  // Pointer event listeners for interactive cursor ripple
  const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const container = containerRef.current;
    if (!container) return;
    const rect = container.getBoundingClientRect();
    mouseRef.current = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
      active: true,
    };
  };

  const handlePointerLeave = () => {
    mouseRef.current.active = false;
  };

  return (
    <div
      ref={containerRef}
      onPointerMove={handlePointerMove}
      onPointerLeave={handlePointerLeave}
      className={`relative w-full h-full overflow-hidden pointer-events-auto ${className}`}
      aria-hidden="true"
    >
      <canvas
        ref={canvasRef}
        className="block w-full h-full pointer-events-none"
      />
    </div>
  );
}
