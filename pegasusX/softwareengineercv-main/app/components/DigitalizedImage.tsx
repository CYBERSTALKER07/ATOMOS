'use client';

import { useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { usePerfProfile } from '@/app/hooks/useDevice';

const FILL_CHARS = '0123456789#$%@&*+=<>/\\|[];:^~IOXZ';

type DigitalizedImageProps = {
  src: string;
  alt?: string;
  color?: string;
  backgroundColor?: string;
  cellSize?: number;
  threshold?: number;
  className?: string;
};

function hexToRgb(hex: string) {
  const h = hex.replace('#', '');
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16),
  };
}

/**
 * Image silhouette filled with digits/symbols.
 * Heavy continuous FX only on capable desktops; mobile/tablet/low-end get a
 * single static digitalized frame (or plain image if reduced-motion).
 */
export default function DigitalizedImage({
  src,
  alt = '',
  color = '#e8e4e3',
  backgroundColor = '#000000',
  cellSize,
  threshold = 0.22,
  className = '',
}: DigitalizedImageProps) {
  const perf = usePerfProfile();
  const resolvedCell = cellSize ?? perf.cellSize;
  const animate = perf.allowHeavyFx;
  const showPlain = perf.prefersReducedMotion;

  const wrapRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const glowRef = useRef<HTMLDivElement>(null);
  const [hovered, setHovered] = useState(false);
  const hoveredRef = useRef(false);

  useEffect(() => {
    hoveredRef.current = hovered;
  }, [hovered]);

  useEffect(() => {
    if (showPlain) return;

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
    let weight: Float32Array = new Float32Array(0);
    let chars: string[] = [];
    let bgChars: string[] = [];
    let raf = 0;
    let imageReady = false;
    let running = true;

    const mouse = { x: 0.5, y: 0.5, tx: 0.5, ty: 0.5 };

    const img = new window.Image();
    img.decoding = 'async';

    const sampleImage = () => {
      if (!img.complete || !img.naturalWidth) return;

      const off = document.createElement('canvas');
      off.width = cols;
      off.height = rows;
      const octx = off.getContext('2d');
      if (!octx) return;

      const scale = Math.min(cols / img.naturalWidth, rows / img.naturalHeight) * 0.88;
      const dw = img.naturalWidth * scale;
      const dh = img.naturalHeight * scale;
      const dx = (cols - dw) / 2;
      const dy = (rows - dh) / 2;

      octx.fillStyle = '#000';
      octx.fillRect(0, 0, cols, rows);
      octx.drawImage(img, dx, dy, dw, dh);

      const data = octx.getImageData(0, 0, cols, rows).data;
      weight = new Float32Array(cols * rows);
      for (let i = 0; i < cols * rows; i++) {
        const r = data[i * 4]!;
        const g = data[i * 4 + 1]!;
        const b = data[i * 4 + 2]!;
        const lum = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
        weight[i] = lum > threshold ? lum : 0;
      }

      chars = Array.from({ length: cols * rows }, pick);
      bgChars = Array.from({ length: cols * rows }, pick);
      imageReady = true;
    };

    const paintFrame = (hover: boolean, scramble: boolean) => {
      if (!imageReady || weight.length === 0) {
        ctx.fillStyle = `rgb(${bg.r},${bg.g},${bg.b})`;
        ctx.fillRect(0, 0, cssW, cssH);
        return;
      }

      if (scramble) {
        const rate = hover ? 0.1 : 0.03;
        const updates = Math.max(4, Math.floor(chars.length * rate));
        for (let n = 0; n < updates; n++) {
          const idx = Math.floor(Math.random() * chars.length);
          if (!(weight[idx]! > 0)) continue;
          chars[idx] = pick();
        }
        if (hover) {
          const bgUpdates = Math.max(4, Math.floor(bgChars.length * 0.04));
          for (let n = 0; n < bgUpdates; n++) {
            const idx = Math.floor(Math.random() * bgChars.length);
            if (weight[idx]! > 0) continue;
            bgChars[idx] = pick();
          }
        }
      }

      ctx.fillStyle = `rgb(${bg.r},${bg.g},${bg.b})`;
      ctx.fillRect(0, 0, cssW, cssH);
      ctx.font = `600 ${Math.floor(cellW * 0.92)}px ui-monospace, Menlo, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const i = row * cols + col;
          const w = weight[i]!;
          const cx = (col + 0.5) / cols;
          const cy = (row + 0.5) / rows;
          const dx = cx - mouse.x;
          const dy = cy - mouse.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          const cursorGlow = hover ? Math.max(0, 1 - dist * 2.5) : 0;

          if (!(w > 0)) {
            if (!hover) continue;
            if ((col + row) % 3 !== 0 && cursorGlow < 0.4) continue;
            const bgAlpha = Math.min(0.2, 0.03 + cursorGlow * 0.25);
            if (bgAlpha < 0.04) continue;
            ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},${bgAlpha})`;
            ctx.fillText(bgChars[i] ?? '0', (col + 0.5) * cellW, (row + 0.5) * cellH);
            continue;
          }

          const alpha = Math.min(1, w * 0.85 + cursorGlow * 0.35 + (hover ? 0.08 : 0));
          ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},${alpha})`;
          ctx.fillText(chars[i] ?? '0', (col + 0.5) * cellW, (row + 0.5) * cellH);
        }
      }
    };

    const resize = () => {
      const rect = wrap.getBoundingClientRect();
      cssW = rect.width;
      cssH = rect.height;
      const dpr = Math.min(window.devicePixelRatio || 1, animate ? 2 : 1.25);
      cellW = Math.max(7, resolvedCell);
      cellH = cellW * 1.45;
      cols = Math.max(1, Math.ceil(cssW / cellW));
      rows = Math.max(1, Math.ceil(cssH / cellH));
      canvas.width = Math.floor(cssW * dpr);
      canvas.height = Math.floor(cssH * dpr);
      canvas.style.width = `${cssW}px`;
      canvas.style.height = `${cssH}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      sampleImage();
      paintFrame(false, false);
    };

    img.onload = () => {
      sampleImage();
      paintFrame(false, false);
    };
    img.src = src;

    const onMove = (e: MouseEvent) => {
      if (!animate) return;
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

    if (!animate) {
      return () => {
        running = false;
        wrap.removeEventListener('mousemove', onMove);
        window.removeEventListener('resize', resize);
        img.onload = null;
      };
    }

    // Pause when off-screen
    const io = new IntersectionObserver(
      ([entry]) => {
        running = !!entry?.isIntersecting;
      },
      { rootMargin: '80px' }
    );
    io.observe(wrap);

    let last = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      if (!running) return;
      if (now - last < 40) return;
      last = now;

      const hover = hoveredRef.current;
      mouse.x += (mouse.tx - mouse.x) * (hover ? 0.16 : 0.06);
      mouse.y += (mouse.ty - mouse.y) * (hover ? 0.16 : 0.06);
      paintFrame(hover, true);
    };
    raf = requestAnimationFrame(tick);

    return () => {
      running = false;
      cancelAnimationFrame(raf);
      io.disconnect();
      wrap.removeEventListener('mousemove', onMove);
      window.removeEventListener('resize', resize);
      img.onload = null;
    };
  }, [src, color, backgroundColor, resolvedCell, threshold, animate, showPlain]);

  if (showPlain) {
    return (
      <div className={`relative w-full h-full bg-black flex items-center justify-center p-8 ${className}`}>
        <Image
          src={src}
          alt={alt}
          width={320}
          height={320}
          className="h-auto w-full max-w-[240px] md:max-w-[280px] object-contain opacity-90"
          priority
        />
      </div>
    );
  }

  return (
    <div
      ref={wrapRef}
      className={`relative w-full h-full overflow-hidden bg-black ${className}`}
      onMouseEnter={() => animate && setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      role="img"
      aria-label={alt}
    >
      {animate && (
        <div
          ref={glowRef}
          className={`pointer-events-none absolute inset-0 z-[1] transition-opacity duration-500 ${
            hovered ? 'opacity-100' : 'opacity-0'
          }`}
          style={{
            background:
              'radial-gradient(circle 220px at var(--mx, 50%) var(--my, 50%), rgba(232,228,227,0.12), transparent 70%)',
          }}
        />
      )}
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full" aria-hidden />
    </div>
  );
}
