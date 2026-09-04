"use client";

import { useEffect, useRef } from 'react';

interface DigitTowerProps {
  color?: string;
  backgroundColor?: string;
  /** Character cell size in CSS pixels. */
  pixelSize?: number;
  /** Animation speed. */
  speed?: number;
  /** How much the tower leans toward the cursor (0–1). */
  parallax?: number;
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

type Cell = {
  char: string;
  brightness: number;
  target: number;
  scramble: number;
};

/**
 * Digital control-tower silhouette built from digits & symbols.
 * Shape: wide base → shaft → observation deck → antenna.
 */
export default function DigitTower({
  color = '#e8e4e3',
  backgroundColor = '#000000',
  pixelSize = 10,
  speed = 1.25,
  parallax = 0.35,
  characters = DEFAULT_CHARS,
}: DigitTowerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const charset = Array.from(characters);
    const digits = Array.from('0123456789');
    const symbols = Array.from('#$%@&*+=<>/\\|[];:^~');
    const rgb = hexToRgb(color);
    const bg = hexToRgb(backgroundColor);

    let cols = 0;
    let rows = 0;
    let cells: Cell[] = [];
    let cellW = pixelSize;
    let cellH = pixelSize * 1.55;
    let cssW = 0;
    let cssH = 0;

    const mouse = { x: 0, y: 0, tx: 0, ty: 0 };
    const pick = (set: string[]) => set[Math.floor(Math.random() * set.length)] ?? '0';

    /** Returns brightness 0–1 if (nx,ny) is inside the tower silhouette. */
    const towerMask = (nx: number, ny: number, lean: number, t: number): number => {
      // Normalized: x in ~[-aspect,aspect], y in [-1,1] bottom→top via -ny
      const x = nx - lean;
      const y = -ny; // +1 at top of canvas

      // Remap y from [-1,1] to [0,1] ground→sky
      const h = (y + 1) * 0.5;
      if (h < 0.02 || h > 0.98) return 0;

      const ax = Math.abs(x);
      const pulse = 0.85 + 0.15 * Math.sin(t * speed * 2.2 + h * 14);
      const flicker = 0.7 + 0.3 * Math.sin(t * speed * 5.5 + x * 20 + h * 30);

      // Antenna / mast (top)
      if (h > 0.86) {
        const mastW = 0.035 + (h - 0.86) * 0.02;
        if (ax < mastW) return (0.55 + (h - 0.86) * 2) * pulse * flicker;
        // Beacon tip
        if (h > 0.94 && ax < 0.08 && Math.sin(t * speed * 8) > 0.2) {
          return 0.9 * pulse;
        }
        return 0;
      }

      // Observation deck / control cab
      if (h > 0.72 && h <= 0.86) {
        const deckW = 0.22 + Math.sin((h - 0.72) * Math.PI) * 0.08;
        if (ax > deckW) return 0;
        // Window band
        const windowBand = h > 0.76 && h < 0.82;
        const edge = ax > deckW * 0.82;
        if (edge) return 0.95 * pulse;
        if (windowBand) return (0.35 + 0.45 * flicker) * pulse;
        return 0.7 * pulse * flicker;
      }

      // Mid shaft (tapers upward)
      if (h > 0.28 && h <= 0.72) {
        const taper = 1 - (h - 0.28) / 0.44;
        const shaftW = 0.1 + taper * 0.08;
        if (ax > shaftW) return 0;
        // Vertical rib / elevator shaft
        const rib = ax < 0.03;
        const floorLine = Math.abs((h * 40) % 1 - 0.5) < 0.08;
        if (rib) return 0.85 * pulse;
        if (floorLine) return 0.55 * pulse;
        return (0.4 + (1 - ax / shaftW) * 0.35) * pulse * flicker;
      }

      // Wide base / podium
      if (h >= 0.02 && h <= 0.28) {
        const baseTaper = (h - 0.02) / 0.26;
        const baseW = 0.38 - baseTaper * 0.18;
        if (ax > baseW) return 0;
        const step = Math.abs((h * 28) % 1 - 0.5) < 0.1;
        const core = ax < baseW * 0.35;
        if (step) return 0.75 * pulse;
        if (core) return 0.65 * pulse * flicker;
        return (0.35 + (1 - ax / baseW) * 0.35) * pulse;
      }

      return 0;
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
        char: pick(charset),
        brightness: 0,
        target: 0,
        scramble: Math.random(),
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

      mouse.x += (mouse.tx - mouse.x) * 0.06;
      mouse.y += (mouse.ty - mouse.y) * 0.06;
      const lean = mouse.x * parallax * 0.12;

      ctx.fillStyle = `rgb(${bg.r},${bg.g},${bg.b})`;
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

          const mask = towerMask(nx, ny, lean, t);
          cell.target = mask;
          cell.brightness += (cell.target - cell.brightness) * 0.3;

          if (cell.brightness < 0.06) continue;

          // Scramble faster on deck / antenna
          const h = (-ny + 1) * 0.5;
          const scrambleRate = h > 0.72 ? 0.25 : h > 0.28 ? 0.14 : 0.1;
          cell.scramble += scrambleRate * speed;
          if (cell.scramble >= 1 || Math.random() < scrambleRate * 0.35) {
            cell.char = h > 0.72 ? pick(symbols) : Math.random() < 0.55 ? pick(digits) : pick(charset);
            cell.scramble = 0;
          }

          const a = Math.min(1, cell.brightness);
          const r = Math.min(255, Math.round(rgb.r * a));
          const g = Math.min(255, Math.round(rgb.g * a));
          const b = Math.min(255, Math.round(rgb.b * a));
          ctx.fillStyle = `rgba(${r},${g},${b},${Math.min(1, 0.35 + a * 0.65)})`;
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
  }, [color, backgroundColor, pixelSize, speed, parallax, characters]);

  return (
    <div ref={containerRef} className="relative w-full h-full overflow-hidden bg-black">
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full" aria-hidden />
    </div>
  );
}
