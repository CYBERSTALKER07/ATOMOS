"use client";

import { useEffect, useRef } from 'react';

interface EvilEyeProps {
  eyeColor?: string;
  intensity?: number;
  pupilSize?: number;
  irisWidth?: number;
  glowIntensity?: number;
  scale?: number;
  noiseScale?: number;
  pupilFollow?: number;
  flameSpeed?: number;
  backgroundColor?: string;
  /** Character cell size in CSS pixels. Default 11. */
  pixelSize?: number;
  scanlineStrength?: number;
  /** Digits & symbols used to draw the eye. */
  characters?: string;
}

const DEFAULT_CHARS = '0123456789#$%@&*+=<>/\\|[];:^~IOXZ';

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const h = hex.replace('#', '');
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

type Zone = 'void' | 'lid' | 'sclera' | 'iris' | 'pupil';

type Cell = {
  char: string;
  brightness: number;
  targetBrightness: number;
  scramble: number;
  zone: Zone;
};

export default function EvilEye({
  eyeColor = '#d4d0cf',
  intensity = 1.25,
  pupilSize = 0.7,
  irisWidth = 0.42,
  glowIntensity = 0.55,
  scale = 0.78,
  pupilFollow = 0.75,
  flameSpeed = 1.4,
  backgroundColor = '#000000',
  pixelSize = 10,
  characters = DEFAULT_CHARS,
}: EvilEyeProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const charset = Array.from(characters);
    const digitset = Array.from('0123456789');
    const symbolset = Array.from('#$%@&*+=<>/\\|[];:^~');
    const eyeRgb = hexToRgb(eyeColor);
    const bgRgb = hexToRgb(backgroundColor);

    let cols = 0;
    let rows = 0;
    let cells: Cell[] = [];
    let cellW = pixelSize;
    let cellH = pixelSize * 1.55;
    let cssW = 0;
    let cssH = 0;

    const mouse = { x: 0, y: 0, tx: 0, ty: 0 };

    const pick = (set: string[]) => set[Math.floor(Math.random() * set.length)] ?? '0';
    const randChar = () => pick(charset);

    /**
     * Classic Sauron silhouette:
     * - sharp horizontal almond lid
     * - bright dense iris ring
     * - dark horizontal slit pupil that tracks the cursor
     * - sparse sclera so the iris reads clearly
     */
    const eyeMask = (
      nx: number,
      ny: number,
      pupilOx: number,
      pupilOy: number,
      t: number
    ): { bright: number; zone: Zone } => {
      const sx = nx / scale;
      const sy = ny / scale;

      // Almond eyelid (wide, flat) — hard cut outside
      const lid = Math.hypot(sx * 0.48, sy * 1.55);
      if (lid > 1.0) return { bright: 0, zone: 'void' };

      // Soft lid rim (bright edge of the eye)
      const lidRim = lid > 0.88;

      const irisR = Math.hypot(sx * 1.05, sy * 1.05);
      const irisInner = 0.16 + pupilSize * 0.12;
      const irisOuter = irisInner + 0.22 + irisWidth * 0.38;

      // Horizontal slit pupil (Sauron-style), tracks mouse
      const px = sx - pupilOx;
      const py = sy - pupilOy;
      const pupilDist = Math.hypot(px * 3.6, py * 1.05);
      const pupilEdge = 0.18 + pupilSize * 0.16;

      const angle = Math.atan2(sy, sx);
      const swirl = 0.5 + 0.5 * Math.sin(angle * 8 + t * flameSpeed * 2.4 + irisR * 12);
      const pulse = 0.7 + 0.3 * Math.sin(t * flameSpeed * 3.2 + irisR * 16);

      // Pupil = near-black void (almost no glyphs)
      if (pupilDist < pupilEdge) {
        const fade = pupilDist / pupilEdge;
        return {
          bright: fade < 0.55 ? 0 : 0.08 * swirl,
          zone: 'pupil',
        };
      }

      // Dense bright iris ring of symbols
      if (irisR >= irisInner && irisR <= irisOuter) {
        const mid = (irisInner + irisOuter) * 0.5;
        const half = (irisOuter - irisInner) * 0.5;
        const ring = 1 - Math.abs(irisR - mid) / Math.max(half, 0.001);
        const spokes = 0.55 + 0.45 * Math.pow(Math.abs(Math.sin(angle * 10 + t * flameSpeed)), 0.35);
        const bright = (0.55 + ring * 0.7) * intensity * spokes * pulse * (0.85 + swirl * 0.2);
        return { bright: Math.min(1.55, bright), zone: 'iris' };
      }

      // Lid rim highlight
      if (lidRim) {
        return {
          bright: 0.55 * intensity * glowIntensity * (0.8 + swirl * 0.2),
          zone: 'lid',
        };
      }

      // Sparse sclera / fill inside the almond
      const field = Math.pow(Math.max(0, 1 - lid), 1.2);
      const sparse = field * glowIntensity * intensity * 0.22 * (0.5 + swirl * 0.5);
      return { bright: sparse, zone: 'sclera' };
    };

    const rebuild = () => {
      const rect = container.getBoundingClientRect();
      cssW = rect.width;
      cssH = rect.height;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);

      cellW = Math.max(8, pixelSize);
      cellH = cellW * 1.55;
      cols = Math.max(1, Math.ceil(cssW / cellW));
      rows = Math.max(1, Math.ceil(cssH / cellH));

      canvas.width = Math.floor(cssW * dpr);
      canvas.height = Math.floor(cssH * dpr);
      canvas.style.width = `${cssW}px`;
      canvas.style.height = `${cssH}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      cells = Array.from({ length: cols * rows }, () => ({
        char: randChar(),
        brightness: 0,
        targetBrightness: 0,
        scramble: Math.random(),
        zone: 'void' as Zone,
      }));
    };

    const onMove = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      mouse.tx = ((e.clientX - rect.left) / rect.width) * 2 - 1;
      mouse.ty = -(((e.clientY - rect.top) / rect.height) * 2 - 1);
    };
    const onLeave = () => {
      mouse.tx = 0;
      mouse.ty = 0;
    };

    container.addEventListener('mousemove', onMove);
    container.addEventListener('mouseleave', onLeave);
    window.addEventListener('resize', rebuild);
    rebuild();

    let raf = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      const t = now * 0.001;

      mouse.x += (mouse.tx - mouse.x) * 0.07;
      mouse.y += (mouse.ty - mouse.y) * 0.07;

      const pupilOx = mouse.x * pupilFollow * 0.2;
      const pupilOy = mouse.y * pupilFollow * 0.12;

      ctx.fillStyle = `rgb(${bgRgb.r},${bgRgb.g},${bgRgb.b})`;
      ctx.fillRect(0, 0, cssW, cssH);
      ctx.font = `600 ${Math.floor(cellW * 0.92)}px ui-monospace, SFMono-Regular, Menlo, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      const aspect = cssW / Math.max(cssH, 1);

      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const i = row * cols + col;
          const cell = cells[i];
          if (!cell) continue;

          const nx = ((col + 0.5) / cols - 0.5) * 2 * aspect;
          const ny = ((row + 0.5) / rows - 0.5) * 2;

          const { bright, zone } = eyeMask(nx, ny, pupilOx, pupilOy, t);
          cell.targetBrightness = bright;
          cell.zone = zone;
          cell.brightness += (cell.targetBrightness - cell.brightness) * 0.28;

          // Density control: skip most sclera cells so iris reads clearly
          if (zone === 'sclera' && ((col + row) % 3 !== 0)) {
            continue;
          }
          if (zone === 'pupil' && cell.brightness < 0.06) {
            continue;
          }
          if (cell.brightness < 0.05) continue;

          const scrambleRate =
            zone === 'iris' ? 0.28 : zone === 'lid' ? 0.12 : zone === 'sclera' ? 0.08 : 0.03;
          cell.scramble += scrambleRate * flameSpeed;
          if (cell.scramble >= 1 || Math.random() < scrambleRate * 0.4) {
            if (zone === 'iris') cell.char = pick(symbolset);
            else if (zone === 'pupil' || zone === 'lid') cell.char = pick(digitset);
            else cell.char = randChar();
            cell.scramble = 0;
          }

          const a = Math.min(1, cell.brightness);
          const boost = zone === 'iris' ? 1.25 : zone === 'lid' ? 1.05 : zone === 'pupil' ? 0.4 : 0.75;
          const r = Math.min(255, Math.round(eyeRgb.r * a * boost));
          const g = Math.min(255, Math.round(eyeRgb.g * a * boost));
          const b = Math.min(255, Math.round(eyeRgb.b * a * boost));

          ctx.fillStyle = `rgba(${r},${g},${b},${Math.min(1, 0.4 + a * 0.6)})`;
          ctx.fillText(cell.char, (col + 0.5) * cellW, (row + 0.5) * cellH);
        }
      }
    };

    raf = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('resize', rebuild);
      container.removeEventListener('mousemove', onMove);
      container.removeEventListener('mouseleave', onLeave);
    };
  }, [
    eyeColor,
    intensity,
    pupilSize,
    irisWidth,
    glowIntensity,
    scale,
    pupilFollow,
    flameSpeed,
    backgroundColor,
    pixelSize,
    characters,
  ]);

  return (
    <div ref={containerRef} className="relative w-full h-full overflow-hidden bg-black">
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full" aria-hidden />
    </div>
  );
}
