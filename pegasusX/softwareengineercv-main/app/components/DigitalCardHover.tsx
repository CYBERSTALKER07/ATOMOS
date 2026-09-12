'use client';

import { useEffect, useRef } from 'react';

type DigitalCardHoverProps = {
  active: boolean;
  className?: string;
  characters?: string;
  color?: string;
};

const DEFAULT_CHARS = '0123456789#$%@&*+=<>/\\|[];:^~IOXZ01';

/**
 * Full-bleed digit/symbol field. Animates only while `active` (hover).
 */
export default function DigitalCardHover({
  active,
  className = '',
  characters = DEFAULT_CHARS,
  color = '#9a9695',
}: DigitalCardHoverProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!active) return;

    const canvas = canvasRef.current;
    const wrap = wrapRef.current;
    if (!canvas || !wrap) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const charset = Array.from(characters);
    const cellW = 11;
    const cellH = 16;
    const mouse = { x: 0.5, y: 0.5 };

    let cols = 0;
    let rows = 0;
    let chars: string[] = [];
    let cssW = 0;
    let cssH = 0;
    let raf = 0;

    const pick = () => charset[Math.floor(Math.random() * charset.length)] ?? '0';

    const resize = () => {
      const rect = wrap.getBoundingClientRect();
      cssW = rect.width;
      cssH = rect.height;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      cols = Math.max(1, Math.ceil(cssW / cellW));
      rows = Math.max(1, Math.ceil(cssH / cellH));
      canvas.width = Math.floor(cssW * dpr);
      canvas.height = Math.floor(cssH * dpr);
      canvas.style.width = `${cssW}px`;
      canvas.style.height = `${cssH}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      chars = Array.from({ length: cols * rows }, pick);
    };

    const onMove = (e: MouseEvent) => {
      const rect = wrap.getBoundingClientRect();
      mouse.x = (e.clientX - rect.left) / Math.max(rect.width, 1);
      mouse.y = (e.clientY - rect.top) / Math.max(rect.height, 1);
    };

    wrap.addEventListener('mousemove', onMove);
    window.addEventListener('resize', resize);
    resize();

    let last = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      if (now - last < 40) return;
      last = now;

      // Scramble ~8% of cells each frame
      const updates = Math.max(4, Math.floor(chars.length * 0.08));
      for (let i = 0; i < updates; i++) {
        const idx = Math.floor(Math.random() * chars.length);
        chars[idx] = pick();
      }

      ctx.clearRect(0, 0, cssW, cssH);
      ctx.font = `600 ${cellW}px ui-monospace, SFMono-Regular, Menlo, monospace`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const i = row * cols + col;
          const cx = (col + 0.5) / cols;
          const cy = (row + 0.5) / rows;
          const dx = cx - mouse.x;
          const dy = cy - mouse.y;
          const dist = Math.sqrt(dx * dx + dy * dy);
          // Bright near cursor, dim toward edges
          const cursorGlow = Math.max(0, 1 - dist * 2.2);
          const edgeFade = Math.min(cx, 1 - cx, cy, 1 - cy) * 4;
          const alpha = Math.min(0.55, 0.12 + cursorGlow * 0.45) * Math.min(1, 0.35 + edgeFade);

          ctx.fillStyle = color.startsWith('#')
            ? `rgba(${parseInt(color.slice(1, 3), 16)},${parseInt(color.slice(3, 5), 16)},${parseInt(color.slice(5, 7), 16)},${alpha})`
            : color;
          ctx.fillText(chars[i] ?? '0', (col + 0.5) * cellW, (row + 0.5) * cellH);
        }
      }
    };

    raf = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(raf);
      wrap.removeEventListener('mousemove', onMove);
      window.removeEventListener('resize', resize);
    };
  }, [active, characters, color]);

  return (
    <div
      ref={wrapRef}
      className={`absolute inset-0 overflow-hidden pointer-events-none ${className}`}
      aria-hidden
    >
      {active && <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full" />}
      {/* Keep copy readable */}
      <div className="absolute inset-0 bg-gradient-to-b from-black/45 via-black/25 to-black/70" />
    </div>
  );
}
