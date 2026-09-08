'use client';

import { useEffect, useRef } from 'react';
import { useReducedMotion } from '../hooks/useDevice';

export default function IsometricTerrain() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const prefersReducedMotion = useReducedMotion();

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let width = 0;
    let height = 0;

    // Mouse tracking for parallax tilt
    let mouseX = 0;
    let mouseY = 0;
    let targetTiltX = 0;
    let targetTiltY = 0;
    let currentTiltX = 0;
    let currentTiltY = 0;

    const handleMouseMove = (e: MouseEvent) => {
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width - 0.5;
      const y = (e.clientY - rect.top) / rect.height - 0.5;
      targetTiltX = y * 0.15;
      targetTiltY = x * 0.2;
    };

    const handleMouseLeave = () => {
      targetTiltX = 0;
      targetTiltY = 0;
    };

    window.addEventListener('mousemove', handleMouseMove);

    const resize = () => {
      if (!canvas || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      width = rect.width;
      height = rect.height;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.scale(dpr, dpr);
    };

    resize();
    window.addEventListener('resize', resize);

    // Terrain grid definition
    const COLS = 26;
    const ROWS = 26;
    const ISO_ANGLE = Math.PI / 6; // 30 degrees
    const COS_A = Math.cos(ISO_ANGLE);
    const SIN_A = Math.sin(ISO_ANGLE);

    let time = 0;

    // Peak centers and characteristics (matching the screenshot)
    // The screenshot has a prominent mountain peak on the top-left/center and a secondary ridge
    const elevation = (u: number, v: number, t: number): number => {
      // u, v in range [-1, 1]
      // Peak 1: prominent sharp peak
      const dx1 = u - (-0.22);
      const dy1 = v - (-0.12);
      const d1 = Math.sqrt(dx1 * dx1 + dy1 * dy1);
      const p1 = Math.exp(-d1 * 4.2) * 115;

      // Peak 2: ridge near center-right
      const dx2 = u - (0.18);
      const dy2 = v - (0.08);
      const d2 = Math.sqrt(dx2 * dx2 + dy2 * dy2);
      const p2 = Math.exp(-d2 * 3.6) * 88;

      // Peak 3: smaller ridge
      const dx3 = u - (-0.38);
      const dy3 = v - (0.32);
      const d3 = Math.sqrt(dx3 * dx3 + dy3 * dy3);
      const p3 = Math.exp(-d3 * 4.8) * 52;

      // Subtle dynamic harmonic wave
      const wave = Math.sin(u * 5 + t * 0.0015) * Math.cos(v * 4 + t * 0.001) * 7;

      return p1 + p2 + p3 + wave;
    };

    const project = (
      x: number,
      y: number,
      z: number,
      originX: number,
      originY: number,
      tiltX: number,
      tiltY: number
    ) => {
      // Apply subtle tilt
      const rx = x * Math.cos(tiltY) - y * Math.sin(tiltY);
      const ry = (x * Math.sin(tiltY) + y * Math.cos(tiltY)) * Math.cos(tiltX) - z * Math.sin(tiltX);
      const rz = z * Math.cos(tiltX) + (x * Math.sin(tiltY) + y * Math.cos(tiltY)) * Math.sin(tiltX);

      // Isometric projection
      const screenX = originX + (rx - ry) * COS_A;
      const screenY = originY + (rx + ry) * SIN_A - rz;

      return { x: screenX, y: screenY };
    };

    const render = () => {
      if (!ctx || width === 0 || height === 0) return;

      time += 16;
      currentTiltX += (targetTiltX - currentTiltX) * 0.05;
      currentTiltY += (targetTiltY - currentTiltY) * 0.05;

      ctx.clearRect(0, 0, width, height);

      const originX = width * 0.52;
      const originY = height * 0.44;

      const scale = Math.min(width, height) * 0.64;
      const stepU = 2 / (COLS - 1);
      const stepV = 2 / (ROWS - 1);

      // 1. BOTTOM LAYER: Stacked horizontal contour lines / striated block volume
      // In the screenshot, there are 10-14 horizontal slices descending downwards
      const NUM_BLOCK_LAYERS = 12;
      const BLOCK_DEPTH = 110;

      ctx.save();
      for (let layer = NUM_BLOCK_LAYERS; layer >= 1; layer--) {
        const layerZ = -BLOCK_DEPTH * (layer / NUM_BLOCK_LAYERS) - 15;
        const alpha = 0.12 + (1 - layer / NUM_BLOCK_LAYERS) * 0.22;

        ctx.strokeStyle = `rgba(255, 255, 255, ${alpha.toFixed(3)})`;
        ctx.lineWidth = 0.75;

        // Draw outer contour ring of the block
        const p00 = project(-scale * 0.5, -scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
        const p10 = project(scale * 0.5, -scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
        const p11 = project(scale * 0.5, scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
        const p01 = project(-scale * 0.5, scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);

        ctx.beginPath();
        ctx.moveTo(p00.x, p00.y);
        ctx.lineTo(p10.x, p10.y);
        ctx.lineTo(p11.x, p11.y);
        ctx.lineTo(p01.x, p01.y);
        ctx.closePath();
        ctx.stroke();

        // Draw dotted matrix points on the top block layer
        if (layer === 1) {
          ctx.fillStyle = 'rgba(255, 255, 255, 0.45)';
          for (let i = 0; i < COLS; i += 2) {
            for (let j = 0; j < ROWS; j += 2) {
              const u = -1 + i * stepU;
              const v = -1 + j * stepV;
              const px = u * (scale * 0.5);
              const py = v * (scale * 0.5);
              const pt = project(px, py, layerZ, originX, originY, currentTiltX, currentTiltY);
              ctx.beginPath();
              ctx.arc(pt.x, pt.y, 0.85, 0, Math.PI * 2);
              ctx.fill();
            }
          }
        }
      }

      // Vertical corner edges of the lower block
      const topZ = -15;
      const botZ = -BLOCK_DEPTH - 15;
      const corners = [
        [-scale * 0.5, -scale * 0.5],
        [scale * 0.5, -scale * 0.5],
        [scale * 0.5, scale * 0.5],
        [-scale * 0.5, scale * 0.5],
      ];
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.25)';
      ctx.lineWidth = 0.8;
      for (const [cx, cy] of corners) {
        const pTop = project(cx, cy, topZ, originX, originY, currentTiltX, currentTiltY);
        const pBot = project(cx, cy, botZ, originX, originY, currentTiltX, currentTiltY);
        ctx.beginPath();
        ctx.moveTo(pTop.x, pTop.y);
        ctx.lineTo(pBot.x, pBot.y);
        ctx.stroke();
      }

      // 2. MIDDLE LAYER: Floating Dot Plane (as seen in screenshot)
      const DOT_PLANE_Z = 12;
      ctx.fillStyle = 'rgba(255, 255, 255, 0.55)';
      for (let i = 0; i < COLS; i++) {
        for (let j = 0; j < ROWS; j++) {
          const u = -1 + i * stepU;
          const v = -1 + j * stepV;
          const px = u * (scale * 0.5);
          const py = v * (scale * 0.5);
          const pt = project(px, py, DOT_PLANE_Z, originX, originY, currentTiltX, currentTiltY);
          ctx.beginPath();
          ctx.arc(pt.x, pt.y, 0.75, 0, Math.PI * 2);
          ctx.fill();
        }
      }

      // 3. TOP LAYER: 3D Wireframe Surface Mesh
      const grid: { x: number; y: number; z: number; sx: number; sy: number }[][] = [];

      for (let i = 0; i < COLS; i++) {
        grid[i] = [];
        const u = -1 + i * stepU;
        for (let j = 0; j < ROWS; j++) {
          const v = -1 + j * stepV;
          const px = u * (scale * 0.5);
          const py = v * (scale * 0.5);
          const pz = elevation(u, v, prefersReducedMotion ? 0 : time);
          const pt = project(px, py, pz, originX, originY, currentTiltX, currentTiltY);
          grid[i][j] = { x: px, y: py, z: pz, sx: pt.x, sy: pt.y };
        }
      }

      // Draw wireframe lines along columns (U direction)
      ctx.lineWidth = 1.0;
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.85)';

      for (let i = 0; i < COLS; i++) {
        ctx.beginPath();
        for (let j = 0; j < ROWS; j++) {
          const pt = grid[i][j];
          if (j === 0) {
            ctx.moveTo(pt.sx, pt.sy);
          } else {
            ctx.lineTo(pt.sx, pt.sy);
          }
        }
        ctx.stroke();
      }

      // Draw wireframe lines along rows (V direction)
      for (let j = 0; j < ROWS; j++) {
        ctx.beginPath();
        for (let i = 0; i < COLS; i++) {
          const pt = grid[i][j];
          if (i === 0) {
            ctx.moveTo(pt.sx, pt.sy);
          } else {
            ctx.lineTo(pt.sx, pt.sy);
          }
        }
        ctx.stroke();
      }

      // Vertical drop lines from peaks down to dot plane for high points
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
      ctx.lineWidth = 0.5;
      for (let i = 0; i < COLS; i += 2) {
        for (let j = 0; j < ROWS; j += 2) {
          const pt = grid[i][j];
          if (pt.z > 35) {
            const basePt = project(pt.x, pt.y, DOT_PLANE_Z, originX, originY, currentTiltX, currentTiltY);
            ctx.beginPath();
            ctx.moveTo(pt.sx, pt.sy);
            ctx.lineTo(basePt.x, basePt.y);
            ctx.stroke();
          }
        }
      }

      ctx.restore();

      if (!prefersReducedMotion) {
        animationFrameId = requestAnimationFrame(render);
      }
    };

    render();

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('resize', resize);
      if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
      }
    };
  }, [prefersReducedMotion]);

  return (
    <div ref={containerRef} className="w-full h-full min-h-[360px] sm:min-h-[440px] lg:min-h-[520px] relative flex items-center justify-center overflow-hidden">
      <canvas ref={canvasRef} className="block w-full h-full cursor-grab active:cursor-grabbing" />
    </div>
  );
}
