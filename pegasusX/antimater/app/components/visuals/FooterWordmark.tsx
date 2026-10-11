'use client';

import { useEffect, useRef, type CSSProperties } from 'react';

export interface FooterWordmarkProps {
  text?: string;
  className?: string;
  style?: CSSProperties;
  dotColor?: string;
  ringColor?: string;
  centerDotColor?: string;
  repelRadius?: number;
}

type DotParticle = {
  x: number;
  y: number;
  homeX: number;
  homeY: number;
  size: number;
  seed: number;
};

export default function FooterWordmark({
  text = 'PEGASUS',
  className = '',
  style,
  dotColor = '#ffffff',
  ringColor = 'rgba(255, 255, 255, 0.88)',
  centerDotColor = '#34d399',
  repelRadius = 38,
}: FooterWordmarkProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    if (typeof window === 'undefined') return undefined;

    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return undefined;

    const ctx = canvas.getContext('2d');
    if (!ctx) return undefined;

    let particles: DotParticle[] = [];
    let animationFrame: number | null = null;
    let resizeFrame: number | null = null;
    let width = 0;
    let height = 0;
    let dpr = 1;
    let isVisible = true;
    let isRunning = false;

    const pointer = {
      active: false,
      x: -1000,
      y: -1000,
      smoothX: -1000,
      smoothY: -1000,
    };

    const render = (): void => {
      if (!isVisible) {
        isRunning = false;
        animationFrame = null;
        return;
      }

      ctx.clearRect(0, 0, width, height);

      // Smooth cursor reticle motion
      if (pointer.active) {
        if (pointer.smoothX < -500) {
          pointer.smoothX = pointer.x;
          pointer.smoothY = pointer.y;
        } else {
          pointer.smoothX += (pointer.x - pointer.smoothX) * 0.32;
          pointer.smoothY += (pointer.y - pointer.smoothY) * 0.32;
        }
      }

      let anyMoving = false;
      const ringOuterLimit = repelRadius * 1.35;

      // Update particle positions with perimeter repulsion physics
      for (let i = 0; i < particles.length; i++) {
        const p = particles[i];
        let targetX = p.homeX;
        let targetY = p.homeY;

        if (pointer.active && pointer.smoothX > -100 && pointer.smoothX < width + 100) {
          const dx = p.homeX - pointer.smoothX;
          const dy = p.homeY - pointer.smoothY;
          const dist = Math.hypot(dx, dy);

          if (dist < repelRadius) {
            // Displace outside circle circumference leaving center empty
            const theta = dist > 0.0001 ? Math.atan2(dy, dx) : p.seed * 6.28318;
            const targetR = repelRadius + 2.0 + (1.0 - dist / repelRadius) * 8.0 + (p.seed - 0.5) * 4.0;
            targetX = pointer.smoothX + Math.cos(theta) * targetR;
            targetY = pointer.smoothY + Math.sin(theta) * targetR;
          } else if (dist < ringOuterLimit) {
            // Subtle density bulge right along the outside of the ring
            const theta = Math.atan2(dy, dx);
            const factor = 1.0 - (dist - repelRadius) / (repelRadius * 0.35);
            const push = factor * 5.0;
            targetX = p.homeX + Math.cos(theta) * push;
            targetY = p.homeY + Math.sin(theta) * push;
          }
        }

        // Spring follow interpolation
        const diffX = targetX - p.x;
        const diffY = targetY - p.y;

        if (Math.abs(diffX) > 0.05 || Math.abs(diffY) > 0.05) {
          p.x += diffX * 0.22;
          p.y += diffY * 0.22;
          anyMoving = true;
        } else {
          p.x = targetX;
          p.y = targetY;
        }
      }

      // Draw all dot matrix particles in a high-efficiency batched path
      if (particles.length > 0) {
        ctx.fillStyle = dotColor;
        ctx.beginPath();
        for (let i = 0; i < particles.length; i++) {
          const p = particles[i];
          ctx.moveTo(p.x + p.size, p.y);
          ctx.arc(p.x, p.y, p.size, 0, 6.28318);
        }
        ctx.fill();
      }

      // Draw interactive cursor reticle ( • ) when hovering over wordmark
      if (pointer.active && pointer.smoothX > -50 && pointer.smoothX < width + 50) {
        ctx.save();

        // 1. Outer circumference ring
        ctx.beginPath();
        ctx.arc(pointer.smoothX, pointer.smoothY, repelRadius, 0, 6.28318);
        ctx.strokeStyle = ringColor;
        ctx.lineWidth = 1.4;
        ctx.stroke();

        // 2. High-precision center cursor dot with subtle emerald bloom
        ctx.shadowColor = '#10b981';
        ctx.shadowBlur = 5;
        ctx.beginPath();
        ctx.arc(pointer.smoothX, pointer.smoothY, 2.2, 0, 6.28318);
        ctx.fillStyle = centerDotColor;
        ctx.fill();

        ctx.restore();
      }

      // Continue animation loop while interacting or while particles are settling
      if (pointer.active || anyMoving) {
        animationFrame = window.requestAnimationFrame(render);
      } else {
        isRunning = false;
        animationFrame = null;
      }
    };

    const wakeRenderLoop = (): void => {
      if (isVisible && !isRunning) {
        isRunning = true;
        animationFrame = window.requestAnimationFrame(render);
      }
    };

    const buildWordmark = (): void => {
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

      // Offscreen canvas for typography rasterization
      const offscreen = document.createElement('canvas');
      const offCtx = offscreen.getContext('2d', { willReadFrequently: true });
      if (!offCtx) return;

      offscreen.width = width;
      offscreen.height = height;

      // Calculate typography size & tracking to span proudly across container
      let fontSize = Math.floor(height * 0.85);
      const fontStack = "900 " + fontSize + "px system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif";
      offCtx.font = fontStack;

      const letters = text.split('');
      let tracking = fontSize * 0.08;

      let letterWidths = letters.map((ch) => offCtx.measureText(ch).width);
      let totalTextWidth = letterWidths.reduce((a, b) => a + b, 0) + (letters.length - 1) * tracking;

      // If text exceeds 92% of width, scale down font proportionally
      if (totalTextWidth > width * 0.92) {
        const scale = (width * 0.92) / totalTextWidth;
        fontSize = Math.floor(fontSize * scale);
        offCtx.font = "900 " + fontSize + "px system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif";
        tracking = fontSize * 0.08;
        letterWidths = letters.map((ch) => offCtx.measureText(ch).width);
        totalTextWidth = letterWidths.reduce((a, b) => a + b, 0) + (letters.length - 1) * tracking;
      }

      offCtx.fillStyle = '#ffffff';
      offCtx.textAlign = 'center';
      offCtx.textBaseline = 'middle';

      let startX = (width - totalTextWidth) / 2;
      const centerY = height / 2;

      for (let i = 0; i < letters.length; i++) {
        const ch = letters[i];
        const lw = letterWidths[i];
        offCtx.fillText(ch, startX + lw / 2, centerY);
        startX += lw + tracking;
      }

      const imageData = offCtx.getImageData(0, 0, width, height);
      const newParticles: DotParticle[] = [];

      // Regular Cartesian grid step for strict dot matrix typography
      const gridStep = Math.max(3.8, Math.min(5.0, width / 260));
      const dotRadius = Math.max(1.15, Math.min(1.4, gridStep * 0.3));

      for (let y = Math.floor(gridStep / 2); y < height; y += gridStep) {
        for (let x = Math.floor(gridStep / 2); x < width; x += gridStep) {
          const sampleX = Math.floor(x);
          const sampleY = Math.floor(y);
          const idx = (sampleY * width + sampleX) * 4;
          const alpha = imageData.data[idx + 3];

          if (alpha > 70) {
            const seed = ((sampleX * 37 + sampleY * 97) % 1000) / 1000;
            newParticles.push({
              x: sampleX,
              y: sampleY,
              homeX: sampleX,
              homeY: sampleY,
              size: dotRadius,
              seed,
            });
          }
        }
      }

      particles = newParticles;
      wakeRenderLoop();
    };

    const queueBuild = (): void => {
      if (resizeFrame) window.cancelAnimationFrame(resizeFrame);
      resizeFrame = window.requestAnimationFrame(buildWordmark);
    };

    const handlePointerMove = (e: PointerEvent): void => {
      const rect = canvas.getBoundingClientRect();
      pointer.x = e.clientX - rect.left;
      pointer.y = e.clientY - rect.top;
      pointer.active = true;
      wakeRenderLoop();
    };

    const handlePointerLeave = (): void => {
      pointer.active = false;
      pointer.x = -1000;
      pointer.y = -1000;
      pointer.smoothX = -1000;
      pointer.smoothY = -1000;
      wakeRenderLoop();
    };

    canvas.addEventListener('pointermove', handlePointerMove, { passive: true });
    canvas.addEventListener('pointerdown', handlePointerMove, { passive: true });
    canvas.addEventListener('pointerenter', handlePointerMove, { passive: true });
    canvas.addEventListener('pointerleave', handlePointerLeave);
    canvas.addEventListener('pointerup', handlePointerLeave);
    canvas.addEventListener('pointercancel', handlePointerLeave);

    const observer = new ResizeObserver(() => {
      queueBuild();
    });
    observer.observe(container);

    let intersectionObserver: IntersectionObserver | null = null;
    if ('IntersectionObserver' in window) {
      intersectionObserver = new IntersectionObserver(([entry]) => {
        isVisible = entry?.isIntersecting ?? true;
        if (isVisible) {
          wakeRenderLoop();
        } else if (animationFrame !== null) {
          window.cancelAnimationFrame(animationFrame);
          animationFrame = null;
          isRunning = false;
        }
      });
      intersectionObserver.observe(container);
    }

    queueBuild();

    return () => {
      observer.disconnect();
      intersectionObserver?.disconnect();
      canvas.removeEventListener('pointermove', handlePointerMove);
      canvas.removeEventListener('pointerdown', handlePointerMove);
      canvas.removeEventListener('pointerenter', handlePointerMove);
      canvas.removeEventListener('pointerleave', handlePointerLeave);
      canvas.removeEventListener('pointerup', handlePointerLeave);
      canvas.removeEventListener('pointercancel', handlePointerLeave);
      if (animationFrame !== null) window.cancelAnimationFrame(animationFrame);
      if (resizeFrame !== null) window.cancelAnimationFrame(resizeFrame);
    };
  }, [text, dotColor, ringColor, centerDotColor, repelRadius]);

  return (
    <div
      ref={containerRef}
      className={`relative w-full h-[120px] sm:h-[160px] md:h-[200px] lg:h-[240px] select-none overflow-hidden cursor-none ${className}`}
      style={style}
      role="img"
      aria-label={`${text} interactive brandmark`}
    >
      <canvas ref={canvasRef} className="absolute inset-0 block w-full h-full pointer-events-auto" />
    </div>
  );
}
