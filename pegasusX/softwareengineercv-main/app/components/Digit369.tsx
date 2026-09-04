'use client';

import { useEffect, useRef, useState } from 'react';
import { usePerfProfile } from '@/app/hooks/useDevice';

const FILL_CHARS = '0123456789#$%@&*+=<>/\\|[];:^~IOXZ';
const FIGURES = ['3', '6', '9'] as const;
const FIG_CENTERS = [0.2, 0.5, 0.8] as const;

type Digit369Props = {
  color?: string;
  backgroundColor?: string;
  cellSize?: number;
};

function hexToRgb(hex: string) {
  const h = hex.replace('#', '');
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

function Lite369({ color }: { color: string }) {
  return (
    <div className="relative flex h-full min-h-[240px] sm:min-h-[320px] w-full items-center justify-center bg-black px-6 py-10">
      <div className="flex w-full max-w-lg items-end justify-between gap-2 sm:gap-4">
        {FIGURES.map((n, i) => (
          <span
            key={n}
            className="font-extralight leading-none text-white/80"
            style={{
              color,
              fontSize: 'clamp(3.5rem, 18vw, 7rem)',
              opacity: 0.55 + i * 0.15,
            }}
          >
            {n}
          </span>
        ))}
      </div>
      <div className="absolute bottom-5 left-5 font-mono text-[0.6rem] uppercase tracking-[0.28em] text-white/35">
        3 · 6 · 9
      </div>
    </div>
  );
}

/**
 * 3 / 6 / 9 filled with digits & symbols.
 * Continuous FX on capable desktops only; lite static type on mobile/tablet/low-end.
 */
export default function Digit369({
  color = '#e8e4e3',
  backgroundColor = '#000000',
  cellSize,
}: Digit369Props) {
  const perf = usePerfProfile();
  const resolvedCell = cellSize ?? perf.cellSize;
  const animate = perf.allowHeavyFx;

  const wrapRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const glowRef = useRef<HTMLDivElement>(null);
  const [hovered, setHovered] = useState(false);
  const hoveredRef = useRef(false);

  useEffect(() => {
    hoveredRef.current = hovered;
  }, [hovered]);

  useEffect(() => {
    if (!animate) return;

    const wrap = wrapRef.current;
    const canvas = canvasRef.current;
    if (!wrap || !canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const rgb = hexToRgb(color);
    const bg = hexToRgb(backgroundColor);
    const charset = Array.from(FILL_CHARS);
    const pick = () => charset[Math.floor(Math.random() * charset.length)] ?? '0';

    let cssW = 0;
    let cssH = 0;
    let cellW = resolvedCell;
    let cellH = resolvedCell * 1.45;
    let cols = 0;
    let rows = 0;
    let mask: Int8Array = new Int8Array(0);
    let chars: string[] = [];
    let bgChars: string[] = [];
    let raf = 0;
    let running = true;

    const mouse = { x: 0.5, y: 0.5, tx: 0.5, ty: 0.5 };

    const buildMask = () => {
      const off = document.createElement('canvas');
      off.width = cols;
      off.height = rows;
      const octx = off.getContext('2d');
      if (!octx) return;

      mask = new Int8Array(cols * rows).fill(-1);
      const fontPx = Math.floor(rows * 0.62);

      for (let id = 0; id < 3; id++) {
        octx.clearRect(0, 0, cols, rows);
        octx.fillStyle = '#000';
        octx.fillRect(0, 0, cols, rows);
        octx.fillStyle = '#fff';
        octx.textAlign = 'center';
        octx.textBaseline = 'middle';
        octx.font = `800 ${fontPx}px ui-monospace, Menlo, monospace`;
        octx.fillText(FIGURES[id], cols * FIG_CENTERS[id]!, rows * 0.5);

        const data = octx.getImageData(0, 0, cols, rows).data;
        for (let i = 0; i < cols * rows; i++) {
          if (data[i * 4]! > 40) mask[i] = id as 0 | 1 | 2;
        }
      }

      chars = Array.from({ length: cols * rows }, pick);
      bgChars = Array.from({ length: cols * rows }, pick);
    };

    const resize = () => {
      const rect = wrap.getBoundingClientRect();
      cssW = rect.width;
      cssH = rect.height;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      cellW = Math.max(8, resolvedCell);
      cellH = cellW * 1.45;
      cols = Math.max(1, Math.ceil(cssW / cellW));
      rows = Math.max(1, Math.ceil(cssH / cellH));
      canvas.width = Math.floor(cssW * dpr);
      canvas.height = Math.floor(cssH * dpr);
      canvas.style.width = `${cssW}px`;
      canvas.style.height = `${cssH}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      buildMask();
    };

    const onMove = (e: MouseEvent) => {
      const rect = wrap.getBoundingClientRect();
      mouse.tx = (e.clientX - rect.left) / Math.max(rect.width, 1);
      mouse.ty = (e.clientY - rect.top) / Math.max(rect.height, 1);
      if (glowRef.current) {
        glowRef.current.style.setProperty('--mx', `${mouse.tx * 100}%`);
        glowRef.current.style.setProperty('--my', `${mouse.ty * 100}%`);
      }
    };

    wrap.addEventListener('mousemove', onMove);
    window.addEventListener('resize', resize);
    resize();

    const io = new IntersectionObserver(
      ([entry]) => {
        running = !!entry?.isIntersecting;
      },
      { rootMargin: '60px' }
    );
    io.observe(wrap);

    let last = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      if (!running) return;
      if (now - last < 40) return;
      last = now;

      const hover = hoveredRef.current;
      const t = now * 0.001;
      mouse.x += (mouse.tx - mouse.x) * (hover ? 0.16 : 0.06);
      mouse.y += (mouse.ty - mouse.y) * (hover ? 0.16 : 0.06);

      let activeFig = -1;
      let best = 1;
      if (hover) {
        for (let f = 0; f < 3; f++) {
          const d = Math.abs(mouse.x - FIG_CENTERS[f]!);
          if (d < best) {
            best = d;
            activeFig = f;
          }
        }
        if (best > 0.18) activeFig = -1;
      }

      const glyphRate = hover ? (activeFig >= 0 ? 0.16 : 0.1) : 0.03;
      const glyphUpdates = Math.max(6, Math.floor(chars.length * glyphRate));
      for (let n = 0; n < glyphUpdates; n++) {
        const idx = Math.floor(Math.random() * chars.length);
        if (mask[idx]! < 0) continue;
        if (activeFig >= 0 && mask[idx] !== activeFig && Math.random() < 0.55) continue;
        chars[idx] = pick();
      }

      if (hover) {
        const bgUpdates = Math.max(6, Math.floor(bgChars.length * 0.045));
        for (let n = 0; n < bgUpdates; n++) {
          const idx = Math.floor(Math.random() * bgChars.length);
          if (mask[idx]! >= 0) continue;
          bgChars[idx] = pick();
        }
      }

      ctx.fillStyle = `rgb(${bg.r},${bg.g},${bg.b})`;
      ctx.fillRect(0, 0, cssW, cssH);

      if (hover) {
        ctx.strokeStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.035)`;
        const grid = cellW * 3;
        const drift = (t * 12) % grid;
        for (let x = -grid + drift; x < cssW; x += grid) {
          ctx.beginPath();
          ctx.moveTo(x, 0);
          ctx.lineTo(x, cssH);
          ctx.stroke();
        }
      }

      ctx.font = `600 ${Math.floor(cellW * 0.92)}px ui-monospace, Menlo, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const i = row * cols + col;
          const fig = mask[i]!;
          const cx = (col + 0.5) / cols;
          const cy = (row + 0.5) / rows;
          const dx = cx - mouse.x;
          const dy = cy - mouse.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          const cursorGlow = hover ? Math.max(0, 1 - dist * 2.6) : 0;

          if (fig < 0) {
            if (!hover) continue;
            if ((col + row) % 3 !== 0 && cursorGlow < 0.35) continue;
            const bgAlpha = Math.min(0.25, 0.04 + cursorGlow * 0.3);
            if (bgAlpha < 0.05) continue;
            ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},${bgAlpha})`;
            ctx.fillText(bgChars[i] ?? '0', (col + 0.5) * cellW, (row + 0.5) * cellH);
            continue;
          }

          const isActive = hover && fig === activeFig;
          const isOther = hover && activeFig >= 0 && fig !== activeFig;
          let base = 0.4 + fig * 0.05 + (hover ? 0.1 : 0);
          if (isActive) base += 0.28;
          if (isOther) base *= 0.55;

          let drawX = (col + 0.5) * cellW;
          let drawY = (row + 0.5) * cellH;
          if (isActive) {
            drawX += (cx - FIG_CENTERS[fig]!) * cellW * cols * 0.04;
            drawY += (cy - 0.5) * cellH * rows * 0.04;
          }

          const alpha = Math.min(1, base + cursorGlow * (isActive ? 0.5 : 0.28));
          ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},${alpha})`;
          ctx.fillText(chars[i] ?? '0', drawX, drawY);
        }
      }
    };

    raf = requestAnimationFrame(tick);

    return () => {
      running = false;
      cancelAnimationFrame(raf);
      io.disconnect();
      wrap.removeEventListener('mousemove', onMove);
      window.removeEventListener('resize', resize);
    };
  }, [color, backgroundColor, resolvedCell, animate]);

  if (!animate) {
    return <Lite369 color={color} />;
  }

  return (
    <div
      ref={wrapRef}
      className="relative w-full h-full min-h-[280px] overflow-hidden bg-black"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div
        ref={glowRef}
        className={`pointer-events-none absolute inset-0 transition-opacity duration-500 ${
          hovered ? 'opacity-100' : 'opacity-0'
        }`}
        style={{
          background:
            'radial-gradient(circle 260px at var(--mx, 50%) var(--my, 50%), rgba(232,228,227,0.14), transparent 70%)',
        }}
      />
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full" aria-hidden />
      <div
        className={`pointer-events-none absolute bottom-5 left-5 font-mono text-[0.6rem] uppercase tracking-[0.28em] transition-opacity duration-500 ${
          hovered ? 'text-white/70' : 'text-white/30'
        }`}
      >
        3 · 6 · 9
      </div>
    </div>
  );
}
