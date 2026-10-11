'use client';

import { useEffect, useRef, type CSSProperties } from 'react';
import {
  WORLD_MAP_SVG_PATH,
  WORLD_MAP_WIDTH,
  WORLD_MAP_HEIGHT,
} from './visuals/worldMapData';

export interface ParticleWorldMapProps {
  variant?: 'light' | 'dark';
  particleSize?: number;
  density?: number;
  scatter?: number;
  gatherDuration?: number;
  stagger?: number;
  pointerRepel?: number;
  repelRadius?: number;
  idleDrift?: number;
  showHubs?: boolean;
  className?: string;
  style?: CSSProperties;
}

// Pegasus Global Transit Grid Nodes
const PEGASUS_HUBS = [
  { id: 'TAS', name: 'Tashkent HQ', code: 'TAS // HQ', lon: 69.24, lat: 41.29, active: true },
  { id: 'ORD', name: 'Chicago Hub', code: 'ORD // HUB', lon: -87.62, lat: 41.87, active: true },
  { id: 'FRA', name: 'Frankfurt Hub', code: 'FRA // CELL', lon: 8.68, lat: 50.11, active: true },
  { id: 'DXB', name: 'Dubai Port', code: 'DXB // PORT', lon: 55.27, lat: 25.20, active: true },
  { id: 'SIN', name: 'Singapore Gateway', code: 'SIN // DOCK', lon: 103.81, lat: 1.35, active: true },
  { id: 'TYO', name: 'Tokyo Node', code: 'TYO // GATE', lon: 139.69, lat: 35.69, active: true },
  { id: 'LON', name: 'London Node', code: 'LON // NODE', lon: -0.12, lat: 51.50, active: false },
  { id: 'SYD', name: 'Sydney Port', code: 'SYD // DOCK', lon: 151.20, lat: -33.86, active: false },
  { id: 'GRU', name: 'São Paulo', code: 'GRU // CELL', lon: -46.63, lat: -23.55, active: false },
];

const min_lat = -58.0;
const max_lat = 80.0;
const projectCoord = (lon: number, lat: number) => {
  const x = ((lon + 180.0) / 360.0) * WORLD_MAP_WIDTH;
  const clampedLat = Math.max(min_lat, Math.min(max_lat, lat));
  const y = ((max_lat - clampedLat) / (max_lat - min_lat)) * WORLD_MAP_HEIGHT;
  return { x: Math.round(x), y: Math.round(y) };
};

type Particle = {
  x: number;
  y: number;
  startX: number;
  startY: number;
  targetX: number;
  targetY: number;
  size: number;
  color: string;
  seed: number;
  depth: number;
  delay: number;
};

const clamp = (value: number, min: number, max: number): number => Math.min(Math.max(value, min), max);
const easeOutCubic = (t: number): number => 1 - Math.pow(1 - t, 3);

export default function ParticleWorldMap({
  variant = 'dark',
  particleSize = 1.6,
  density = 3.6,
  scatter = 80,
  gatherDuration = 1000,
  stagger = 200,
  pointerRepel = 32,
  repelRadius = 65,
  idleDrift = 0.22,
  showHubs = false,
  className = '',
  style,
}: ParticleWorldMapProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  const isLight = variant === 'light';

  useEffect(() => {
    if (typeof window === 'undefined') return undefined;

    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return undefined;

    const ctx = canvas.getContext('2d');
    if (!ctx) return undefined;

    let particles: Particle[] = [];
    let scaledHubs: Array<{
      id: string;
      name: string;
      code: string;
      active: boolean;
      screenX: number;
      screenY: number;
    }> = [];
    let animationFrame: number | null = null;
    let resizeFrame: number | null = null;
    let width = 0;
    let height = 0;
    let dpr = 1;
    let isVisible = true;
    let gatherStart = 0;
    let gathering = false;

    const pointer = {
      active: false,
      x: 0,
      y: 0,
      smoothX: 0,
      smoothY: 0,
    };

    const dotColor = isLight ? '#d4d4d8' : '#10b981';
    const hubActiveColor = isLight ? '#1d4ed8' : '#34d399';

    const render = (now: number): void => {
      if (!isVisible) {
        animationFrame = null;
        return;
      }

      ctx.clearRect(0, 0, width, height);

      pointer.smoothX += (pointer.x - pointer.smoothX) * 0.18;
      pointer.smoothY += (pointer.y - pointer.smoothY) * 0.18;

      let complete = true;

      // Subtle bloom glow in dark mode
      if (!isLight) {
        ctx.shadowBlur = 3;
        ctx.shadowColor = '#10b981';
      } else {
        ctx.shadowBlur = 0;
      }

      // Render world continent particles
      particles.forEach((particle) => {
        let baseX = particle.targetX;
        let baseY = particle.targetY;

        if (gathering) {
          const local = (now - gatherStart - particle.delay) / Math.max(1, gatherDuration);
          const progress = clamp(local, 0, 1);
          const eased = easeOutCubic(progress);
          baseX = particle.startX + (particle.targetX - particle.startX) * eased;
          baseY = particle.startY + (particle.targetY - particle.startY) * eased;
          if (progress < 1) complete = false;
        } else if (idleDrift > 0) {
          const driftTime = now * 0.001;
          baseX += Math.sin(driftTime * 0.8 + particle.seed * 8) * idleDrift;
          baseY += Math.cos(driftTime * 0.6 + particle.depth * 8) * idleDrift;
        }

        if (pointer.active && pointerRepel > 0 && repelRadius > 0) {
          const dx = baseX - pointer.smoothX;
          const dy = baseY - pointer.smoothY;
          const distance = Math.hypot(dx, dy);
          if (distance > 0 && distance < repelRadius) {
            const force = Math.pow(1 - distance / repelRadius, 2) * pointerRepel;
            baseX += (dx / distance) * force;
            baseY += (dy / distance) * force;
          }
        }

        particle.x += (baseX - particle.x) * 0.22;
        particle.y += (baseY - particle.y) * 0.22;

        ctx.fillStyle = particle.color;
        ctx.beginPath();
        ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
        ctx.fill();
      });

      // Render transit hubs (clean static dots without pulsing animation)
      if (showHubs && scaledHubs.length > 0) {
        scaledHubs.forEach((hub) => {
          if (hub.active) {
            // Active hub node - solid crisp dot
            ctx.beginPath();
            ctx.arc(hub.screenX, hub.screenY, 2.5, 0, Math.PI * 2);
            ctx.fillStyle = hubActiveColor;
            ctx.fill();
          } else {
            // Subtle secondary node
            ctx.beginPath();
            ctx.arc(hub.screenX, hub.screenY, 1.6, 0, Math.PI * 2);
            ctx.fillStyle = isLight ? '#a1a1aa' : '#52525b';
            ctx.fill();
          }
        });
      }

      ctx.shadowBlur = 0;

      if (gathering && complete) {
        gathering = false;
      }

      animationFrame = window.requestAnimationFrame(render);
    };

    const ensureRenderLoop = (): void => {
      if (isVisible && animationFrame === null) {
        animationFrame = window.requestAnimationFrame(render);
      }
    };

    const sampleMap = async (): Promise<void> => {
      const rect = container.getBoundingClientRect();
      width = Math.floor(rect.width);
      height = Math.floor(rect.height);

      if (width <= 0 || height <= 0) return;

      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.max(1, Math.floor(width * dpr));
      canvas.height = Math.max(1, Math.floor(height * dpr));
      canvas.style.width = '100%';
      canvas.style.height = '100%';
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      const offscreen = document.createElement('canvas');
      const offCtx = offscreen.getContext('2d', { willReadFrequently: true });
      if (!offCtx) return;

      const scale = Math.min(width / WORLD_MAP_WIDTH, height / WORLD_MAP_HEIGHT);
      const renderW = Math.ceil(WORLD_MAP_WIDTH * scale);
      const renderH = Math.ceil(WORLD_MAP_HEIGHT * scale);
      const offsetX = (width - renderW) / 2;
      const offsetY = (height - renderH) / 2;

      offscreen.width = renderW;
      offscreen.height = renderH;
      offCtx.clearRect(0, 0, offscreen.width, offscreen.height);

      offCtx.save();
      offCtx.scale(scale, scale);
      offCtx.fillStyle = '#000000';

      if (typeof Path2D !== 'undefined') {
        const path = new Path2D(WORLD_MAP_SVG_PATH);
        offCtx.fill(path, 'evenodd');
      }
      offCtx.restore();

      const imageData = offCtx.getImageData(0, 0, offscreen.width, offscreen.height);
      const targets: Array<{ x: number; y: number }> = [];
      const step = Math.max(2.8, density);

      for (let y = 0; y < offscreen.height; y += step) {
        const pixelY = Math.min(offscreen.height - 1, Math.floor(y));
        for (let x = 0; x < offscreen.width; x += step) {
          const pixelX = Math.min(offscreen.width - 1, Math.floor(x));
          const alpha = imageData.data[(pixelY * offscreen.width + pixelX) * 4 + 3];
          if (alpha > 40) {
            targets.push({
              x: offsetX + x,
              y: offsetY + y,
            });
          }
        }
      }

      scaledHubs = PEGASUS_HUBS.map((hub) => {
        const p = projectCoord(hub.lon, hub.lat);
        return {
          id: hub.id,
          name: hub.name,
          code: hub.code,
          active: hub.active,
          screenX: offsetX + p.x * scale,
          screenY: offsetY + p.y * scale,
        };
      });

      particles = targets.map((target, index) => {
        const seed = ((index * 9301 + 49297) % 233280) / 233280;
        const depth = 0.5 + (((index * 233 + 97) % 1000) / 1000) * 0.5;

        return {
          x: target.x,
          y: target.y,
          startX: target.x,
          startY: target.y,
          targetX: target.x,
          targetY: target.y,
          size: particleSize,
          color: dotColor,
          seed,
          depth,
          delay: seed * stagger,
        };
      });

      pointer.x = width / 2;
      pointer.y = height / 2;
      pointer.smoothX = pointer.x;
      pointer.smoothY = pointer.y;

      gatherStart = performance.now();
      gathering = true;
      ensureRenderLoop();
    };

    const queueSample = (): void => {
      if (resizeFrame) window.cancelAnimationFrame(resizeFrame);
      resizeFrame = window.requestAnimationFrame(sampleMap);
    };

    const handlePointerMove = (event: PointerEvent): void => {
      const rect = canvas.getBoundingClientRect();
      pointer.x = event.clientX - rect.left;
      pointer.y = event.clientY - rect.top;
      pointer.active = true;
      ensureRenderLoop();
    };

    const handlePointerLeave = (): void => {
      pointer.active = false;
    };

    canvas.addEventListener('pointermove', handlePointerMove, { passive: true });
    canvas.addEventListener('pointerleave', handlePointerLeave);

    const observer = new ResizeObserver(() => {
      queueSample();
    });
    observer.observe(container);

    let intersectionObserver: IntersectionObserver | null = null;
    if ('IntersectionObserver' in window) {
      intersectionObserver = new IntersectionObserver(([entry]) => {
        isVisible = entry?.isIntersecting ?? true;
        if (isVisible) {
          ensureRenderLoop();
        } else if (animationFrame !== null) {
          window.cancelAnimationFrame(animationFrame);
          animationFrame = null;
        }
      });
      intersectionObserver.observe(container);
    }

    queueSample();

    return () => {
      observer.disconnect();
      intersectionObserver?.disconnect();
      canvas.removeEventListener('pointermove', handlePointerMove);
      canvas.removeEventListener('pointerleave', handlePointerLeave);
      if (animationFrame !== null) window.cancelAnimationFrame(animationFrame);
      if (resizeFrame !== null) window.cancelAnimationFrame(resizeFrame);
    };
  }, [variant, particleSize, density, scatter, gatherDuration, stagger, pointerRepel, repelRadius, idleDrift, showHubs, isLight]);

  return (
    <div
      ref={containerRef}
      className={`relative w-full aspect-[16/9] min-h-[140px] max-h-[190px] select-none overflow-hidden cursor-crosshair ${className}`}
      style={style}
      aria-label="Pegasus Autonomous Transit World Mesh"
    >
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full pointer-events-auto" />
    </div>
  );
}
