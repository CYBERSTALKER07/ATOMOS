'use client';

import React, { useEffect, useRef } from 'react';
import { decodeNeuralParticles, NeuralPoint } from '../data/neuralFaceData';

interface Particle {
  nx: number;
  ny: number;
  baseX: number;
  baseY: number;
  startX: number;
  startY: number;
  x: number;
  y: number;
  vx: number;
  vy: number;
  size: number;
  idlePhase: number;
  idleRadius: number;
  delay: number;
}

interface ColorBucket {
  color: string;
  isBright: boolean;
  particles: Particle[];
}

interface Ripple {
  x: number;
  y: number;
  radius: number;
  maxRadius: number;
  speed: number;
  strength: number;
}

export default function NeuralParticleCanvas() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d', { alpha: true });
    if (!ctx) return;

    const rawPoints = decodeNeuralParticles();
    if (rawPoints.length === 0) return;

    let animId: number | null = null;
    let isVisible = false;
    let width = 0;
    let height = 0;
    let dpr = 1;

    // Pointer state with smooth lerping
    const pointer = {
      active: false,
      x: -9999,
      y: -9999,
      smoothX: -9999,
      smoothY: -9999,
    };

    // Interactive click shockwaves
    const ripples: Ripple[] = [];

    // Particle storage & color buckets
    let particles: Particle[] = [];
    let buckets: ColorBucket[] = [];
    const bucketMap = new Map<string, Particle[]>();

    // Starter forming animation state
    let isForming = false;
    let formingStart = 0;
    const formingDuration = 1350;
    let hasTriggeredForming = false;

    const setupParticles = (w: number, h: number) => {
      const ORIG_W = 1024;
      const ORIG_H = 612;

      let scale = (h / ORIG_H) * 1.05;
      let offsetX = -10;
      let offsetY = (h - ORIG_H * scale) * 0.5;

      if (w < 768) {
        scale = (h / ORIG_H) * 0.95;
        offsetX = -Math.max(20, ORIG_W * scale * 0.18);
        offsetY = (h - ORIG_H * scale) * 0.5;
      } else if (w < 1024) {
        scale = (h / ORIG_H) * 1.0;
        offsetX = -15;
        offsetY = (h - ORIG_H * scale) * 0.5;
      }

      bucketMap.clear();

      if (particles.length === 0) {
        particles = rawPoints.map((pt: NeuralPoint, idx: number) => {
          const bx = offsetX + pt.nx * ORIG_W * scale;
          const by = offsetY + pt.ny * ORIG_H * scale;

          // Starter animation: dither sweep assembly from left to right
          const seedAngle = (idx * 79.19) * (Math.PI / 180);
          const seedDist = (Math.random() * 70 + 25);
          const startX = bx + Math.cos(seedAngle) * seedDist;
          const startY = by + Math.sin(seedAngle) * seedDist;

          // Sequential left-to-right cybernetic dither sweep
          const delay = (pt.nx * 0.8 + (idx % 8) * 0.025) * 650;

          const p: Particle = {
            nx: pt.nx,
            ny: pt.ny,
            baseX: bx,
            baseY: by,
            startX,
            startY,
            x: startX,
            y: startY,
            vx: 0,
            vy: 0,
            size: pt.size,
            idlePhase: (pt.nx * 24.0 + pt.ny * 16.0 + (idx % 4) * 1.57) % (Math.PI * 2),
            idleRadius: 0.25,
            delay,
          };

          const qr = Math.min(255, Math.round(pt.r / 16) * 16);
          const qg = Math.min(255, Math.round(pt.g / 16) * 16);
          const qb = Math.min(255, Math.round(pt.b / 16) * 16);
          const colorKey = `rgb(${qr},${qg},${qb})`;

          if (!bucketMap.has(colorKey)) {
            bucketMap.set(colorKey, []);
          }
          bucketMap.get(colorKey)!.push(p);

          return p;
        });

        buckets = Array.from(bucketMap.entries()).map(([color, pts]) => {
          const m = color.match(/\d+/g);
          const lum = m ? Number(m[0]) * 0.299 + Number(m[1]) * 0.587 + Number(m[2]) * 0.114 : 0;
          return {
            color,
            isBright: lum > 170,
            particles: pts,
          };
        });
      } else {
        for (let i = 0; i < particles.length; i++) {
          const p = particles[i];
          p.baseX = offsetX + p.nx * ORIG_W * scale;
          p.baseY = offsetY + p.ny * ORIG_H * scale;
        }
      }
    };

    const triggerFormation = () => {
      isForming = true;
      formingStart = performance.now();
      hasTriggeredForming = true;

      // Reset particles to start scattered positions for cinematic assembly
      for (let i = 0; i < particles.length; i++) {
        const p = particles[i];
        p.x = p.startX;
        p.y = p.startY;
        p.vx = 0;
        p.vy = 0;
      }
    };

    const handleResize = () => {
      const rect = canvas.getBoundingClientRect();
      width = rect.width;
      height = rect.height;
      dpr = Math.min(window.devicePixelRatio || 1, 2);

      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);

      setupParticles(width, height);
    };

    handleResize();

    const resizeObserver = new ResizeObserver(() => {
      handleResize();
    });
    if (canvas.parentElement) {
      resizeObserver.observe(canvas.parentElement);
    }

    // Section-level pointer tracking (handles events across the whole section)
    const onPointerMove = (e: MouseEvent | TouchEvent) => {
      const rect = canvas.getBoundingClientRect();
      let clientX = 0;
      let clientY = 0;
      if ('touches' in e && e.touches.length > 0) {
        clientX = e.touches[0].clientX;
        clientY = e.touches[0].clientY;
      } else if ('clientX' in e) {
        clientX = e.clientX;
        clientY = e.clientY;
      }
      pointer.x = clientX - rect.left;
      pointer.y = clientY - rect.top;
      pointer.active = true;
    };

    const onPointerLeave = () => {
      pointer.active = false;
      pointer.x = -9999;
      pointer.y = -9999;
    };

    // Click shockwave ripple
    const onClick = (e: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const clickX = e.clientX - rect.left;
      const clickY = e.clientY - rect.top;

      if (ripples.length < 4) {
        ripples.push({
          x: clickX,
          y: clickY,
          radius: 5,
          maxRadius: 280,
          speed: 7.5,
          strength: 6.5,
        });
      }
    };

    const section = canvas.closest('section') || canvas.parentElement || window;
    section.addEventListener('pointermove', onPointerMove as EventListener);
    section.addEventListener('pointerleave', onPointerLeave);
    section.addEventListener('touchmove', onPointerMove as EventListener, { passive: true });
    section.addEventListener('touchend', onPointerLeave);
    section.addEventListener('click', onClick as EventListener);

    const io = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        const wasVisible = isVisible;
        isVisible = entry.isIntersecting;

        if (isVisible) {
          if (!hasTriggeredForming) {
            triggerFormation();
          }
          if (!animId) {
            animId = requestAnimationFrame(render);
          }
        } else if (wasVisible && !isVisible) {
          if (animId) {
            cancelAnimationFrame(animId);
            animId = null;
          }
        }
      },
      { threshold: 0.08 }
    );
    io.observe(canvas);

    const render = (now: number) => {
      if (!isVisible) {
        animId = null;
        return;
      }

      ctx.save();
      ctx.scale(dpr, dpr);
      ctx.clearRect(0, 0, width, height);

      // Smooth pointer coordinates
      pointer.smoothX += (pointer.x - pointer.smoothX) * 0.18;
      pointer.smoothY += (pointer.y - pointer.smoothY) * 0.18;

      const repelRadius = 115;
      const repelRadiusSq = repelRadius * repelRadius;
      const springRate = isForming ? 0.09 : 0.13;
      const friction = 0.84;

      // Global wave coefficients
      const driftTime = now * 0.0015;
      const waveSin = Math.sin(driftTime * 0.85);
      const waveCos = Math.cos(driftTime * 0.7);

      const hasPointer = pointer.active && pointer.smoothX > -100 && pointer.smoothX < width + 100;

      // Update interactive click shockwave ripples
      for (let r = ripples.length - 1; r >= 0; r--) {
        const rip = ripples[r];
        rip.radius += rip.speed;
        if (rip.radius >= rip.maxRadius) {
          ripples.splice(r, 1);
        }
      }

      // Check forming completion
      let allFormed = true;
      let globalAlpha = 1.0;

      if (isForming) {
        const elapsed = now - formingStart;
        const globalProgress = Math.min(Math.max(elapsed / (formingDuration * 0.8), 0), 1);
        globalAlpha = 0.1 + 0.9 * (1 - Math.pow(1 - globalProgress, 3));

        if (elapsed > formingDuration + 500) {
          isForming = false;
        }
      }

      ctx.globalAlpha = globalAlpha;

      for (let b = 0; b < buckets.length; b++) {
        const bucket = buckets[b];
        ctx.fillStyle = bucket.color;

        ctx.shadowBlur = 0;

        const pts = bucket.particles;
        const count = pts.length;

        for (let i = 0; i < count; i++) {
          const p = pts[i];
          let targetX = p.baseX;
          let targetY = p.baseY;

          if (isForming) {
            const local = (now - formingStart - p.delay) / formingDuration;
            const progress = Math.min(Math.max(local, 0), 1);
            const eased = 1 - Math.pow(1 - progress, 3);
            targetX = p.startX + (p.baseX - p.startX) * eased;
            targetY = p.startY + (p.baseY - p.startY) * eased;
            if (progress < 1) allFormed = false;
          } else {
            // Rigid Bayer dither grid alignment in idle state
            targetX = p.baseX;
            targetY = p.baseY;
          }

          // Cursor repulsion
          if (hasPointer) {
            const dx = p.x - pointer.smoothX;
            const dy = p.y - pointer.smoothY;
            const distSq = dx * dx + dy * dy;

            if (distSq < repelRadiusSq && distSq > 0) {
              const dist = Math.sqrt(distSq);
              const force = Math.pow(1 - dist / repelRadius, 2) * 5.2;
              p.vx += (dx / dist) * force;
              p.vy += (dy / dist) * force;
            }
          }

          // Shockwave ripple repulsion
          if (ripples.length > 0) {
            for (let r = 0; r < ripples.length; r++) {
              const rip = ripples[r];
              const rdx = p.x - rip.x;
              const rdy = p.y - rip.y;
              const rdist = Math.hypot(rdx, rdy);
              const waveDiff = Math.abs(rdist - rip.radius);
              if (waveDiff < 45 && rdist > 0) {
                const waveForce = (1 - waveDiff / 45) * rip.strength;
                p.vx += (rdx / rdist) * waveForce;
                p.vy += (rdy / rdist) * waveForce;
              }
            }
          }

          p.vx *= friction;
          p.vy *= friction;

          p.x += (targetX - p.x) * springRate + p.vx;
          p.y += (targetY - p.y) * springRate + p.vy;

          // Living Bayer dither shimmer: subtle ordered harmonic phase oscillation
          let sz = p.size;
          if (!isForming) {
            const ditherShimmer = Math.sin(now * 0.0032 + p.idlePhase);
            sz = Math.max(0.8, p.size + ditherShimmer * 0.32);
          }

          ctx.fillRect(p.x - sz * 0.5, p.y - sz * 0.5, sz, sz);
        }
      }

      ctx.restore();
      animId = requestAnimationFrame(render);
    };

    return () => {
      if (animId) cancelAnimationFrame(animId);
      resizeObserver.disconnect();
      io.disconnect();
      section.removeEventListener('pointermove', onPointerMove as EventListener);
      section.removeEventListener('pointerleave', onPointerLeave);
      section.removeEventListener('touchmove', onPointerMove as EventListener);
      section.removeEventListener('touchend', onPointerLeave);
      section.removeEventListener('click', onClick as EventListener);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="absolute inset-0 w-full h-full pointer-events-none select-none z-0"
      style={{ display: 'block' }}
      aria-hidden="true"
    />
  );
}
