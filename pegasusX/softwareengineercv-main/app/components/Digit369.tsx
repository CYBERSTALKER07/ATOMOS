'use client';

import React, { useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { usePerfProfile } from '@/app/hooks/useDevice';

const FILL_CHARS = '0123456789#$%@&*+=<>/\\|[];:^~IOXZPEGASUS';

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

function LitePegasus({ color }: { color: string }) {
  return (
    <div className="relative flex h-full min-h-[240px] sm:min-h-[320px] w-full items-center justify-center bg-black px-6 py-10 overflow-hidden">
      <div className="relative w-48 h-48 sm:w-64 sm:h-64 flex items-center justify-center">
        <div className="absolute inset-0 bg-blue-600/15 rounded-full blur-3xl" />
        <Image
          src="/pegasus.jpg"
          alt="Pegasus Emblem"
          width={240}
          height={240}
          className="object-contain filter brightness-110 contrast-125 select-none"
        />
      </div>
      <div className="absolute bottom-5 left-5 font-mono text-[0.65rem] uppercase tracking-[0.28em] text-white/40 flex items-center space-x-2">
        <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
        <span>PEGASUS · LOGISTICS OS</span>
      </div>
    </div>
  );
}

/**
 * Interactive Pegasus Emblem Matrix.
 * Replaces the 369 figures with the official Pegasus winged shape composed of dynamic digital glyphs.
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

    // Preload Pegasus logo image for canvas silhouette sampling
    const pegasusImg = new window.Image();
    pegasusImg.src = '/pegasus.jpg';

    const buildMask = () => {
      if (cols <= 0 || rows <= 0) return;

      const off = document.createElement('canvas');
      off.width = cols;
      off.height = rows;
      const octx = off.getContext('2d');
      if (!octx) return;

      mask = new Int8Array(cols * rows).fill(-1);

      octx.clearRect(0, 0, cols, rows);
      octx.fillStyle = '#000';
      octx.fillRect(0, 0, cols, rows);

      if (pegasusImg.complete && pegasusImg.naturalWidth > 0) {
        // Fit the Pegasus logo into the grid with responsive margin
        const padX = cols * 0.12;
        const padY = rows * 0.12;
        const maxW = cols - padX * 2;
        const maxH = rows - padY * 2;
        const imgAspect = pegasusImg.naturalWidth / pegasusImg.naturalHeight;

        let drawW = maxW;
        let drawH = drawW / imgAspect;
        if (drawH > maxH) {
          drawH = maxH;
          drawW = drawH * imgAspect;
        }

        const drawX = (cols - drawW) / 2;
        const drawY = (rows - drawH) / 2;

        octx.drawImage(pegasusImg, drawX, drawY, drawW, drawH);

        const data = octx.getImageData(0, 0, cols, rows).data;
        for (let i = 0; i < cols * rows; i++) {
          const r = data[i * 4];
          const g = data[i * 4 + 1];
          const b = data[i * 4 + 2];
          // Sample luminance of the white Pegasus shape on black background
          const lum = 0.299 * r + 0.587 * g + 0.114 * b;
          if (lum > 40) {
            mask[i] = 0; // In Pegasus shape
          }
        }
      }

      chars = Array.from({ length: cols * rows }, pick);
      bgChars = Array.from({ length: cols * rows }, pick);
    };

    pegasusImg.onload = () => {
      buildMask();
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

    let last = 0;
    const tick = (now: number) => {
      if (!running) {
        raf = 0;
        return;
      }
      raf = requestAnimationFrame(tick);
      if (now - last < 40) return;
      last = now;

      const hover = hoveredRef.current;
      const t = now * 0.001;
      mouse.x += (mouse.tx - mouse.x) * (hover ? 0.16 : 0.06);
      mouse.y += (mouse.ty - mouse.y) * (hover ? 0.16 : 0.06);

      // Scramble active characters in the Pegasus shape
      const glyphRate = hover ? 0.14 : 0.035;
      const glyphUpdates = Math.max(6, Math.floor(chars.length * glyphRate));
      for (let n = 0; n < glyphUpdates; n++) {
        const idx = Math.floor(Math.random() * chars.length);
        if (mask[idx]! < 0) continue;
        chars[idx] = pick();
      }

      if (hover) {
        const bgUpdates = Math.max(6, Math.floor(bgChars.length * 0.05));
        for (let n = 0; n < bgUpdates; n++) {
          const idx = Math.floor(Math.random() * bgChars.length);
          if (mask[idx]! >= 0) continue;
          bgChars[idx] = pick();
        }
      }

      // Background clear
      ctx.fillStyle = `rgb(${bg.r},${bg.g},${bg.b})`;
      ctx.fillRect(0, 0, cssW, cssH);

      // Tactical grid lines on hover
      if (hover) {
        ctx.strokeStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.03)`;
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
          const cursorGlow = hover ? Math.max(0, 1 - dist * 2.5) : 0;

          // Outside Pegasus shape
          if (fig < 0) {
            if (!hover) continue;
            if ((col + row) % 3 !== 0 && cursorGlow < 0.3) continue;
            const bgAlpha = Math.min(0.22, 0.03 + cursorGlow * 0.28);
            if (bgAlpha < 0.04) continue;
            ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},${bgAlpha})`;
            ctx.fillText(bgChars[i] ?? '0', (col + 0.5) * cellW, (row + 0.5) * cellH);
            continue;
          }

          // Inside Pegasus shape
          let alpha = 0.72 + cursorGlow * 0.28;
          let r = rgb.r;
          let g = rgb.g;
          let b = rgb.b;

          // Electric blue highlight on hover near cursor
          if (cursorGlow > 0.4) {
            r = Math.min(255, Math.floor(r * 0.7 + 59 * 0.3));
            g = Math.min(255, Math.floor(g * 0.7 + 130 * 0.3));
            b = 255;
            alpha = Math.min(1, alpha + 0.15);
          }

          ctx.fillStyle = `rgba(${r},${g},${b},${alpha})`;
          ctx.fillText(chars[i] ?? 'P', (col + 0.5) * cellW, (row + 0.5) * cellH);
        }
      }
    };

    raf = requestAnimationFrame(tick);

    return () => {
      running = false;
      cancelAnimationFrame(raf);
      wrap.removeEventListener('mousemove', onMove);
      window.removeEventListener('resize', resize);
    };
  }, [animate, color, backgroundColor, resolvedCell]);

  if (!animate) {
    return <LitePegasus color={color} />;
  }

  return (
    <div
      ref={wrapRef}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      className="relative flex h-full min-h-[240px] sm:min-h-[300px] md:min-h-[360px] lg:min-h-full w-full items-center justify-center bg-black overflow-hidden select-none cursor-crosshair group"
      style={{
        backgroundColor,
      }}
    >
      {/* Background ambient lighting */}
      <div className="absolute inset-0 pointer-events-none opacity-40 bg-[radial-gradient(circle_at_center,_rgba(37,99,235,0.15)_0%,_transparent_75%)]" />

      {/* Dynamic Cursor Glow */}
      <div
        ref={glowRef}
        className="pointer-events-none absolute -inset-20 opacity-0 transition-opacity duration-300 group-hover:opacity-100"
        style={{
          background:
            'radial-gradient(400px circle at var(--mx, 50%) var(--my, 50%), rgba(37, 99, 235, 0.18), transparent 70%)',
        }}
      />

      {/* Interactive Matrix Canvas */}
      <canvas ref={canvasRef} className="relative z-10 block" />

      {/* Bottom Status Metadata Line */}
      <div className="absolute bottom-5 left-5 z-20 font-mono text-[0.65rem] uppercase tracking-[0.28em] text-white/40 flex items-center space-x-2 pointer-events-none">
        <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
        <span>PEGASUS · LOGISTICS OS</span>
      </div>
    </div>
  );
}
