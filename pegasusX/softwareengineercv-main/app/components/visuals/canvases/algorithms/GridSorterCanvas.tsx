'use client';

import React, { useEffect, useRef } from 'react';

export default function GridSorterCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) return;

    let animationFrameId: number;
    let width = 0;
    let height = 0;

    const gridSize = 12;
    const boxes = Array.from({ length: 45 }, () => ({
      gx: Math.floor(Math.random() * gridSize),
      gy: Math.floor(Math.random() * gridSize),
      targetGx: Math.floor(Math.random() * gridSize),
      targetGy: Math.floor(Math.random() * gridSize),
      progress: 0,
      active: false
    }));

    const resize = () => {
      width = canvas.parentElement?.clientWidth || window.innerWidth;
      height = canvas.parentElement?.clientHeight || window.innerHeight;
      
      const dpr = window.devicePixelRatio || 1;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      ctx.scale(dpr, dpr);
    };

    window.addEventListener('resize', resize);
    resize();

    const render = () => {
      ctx.fillStyle = '#030303';
      ctx.fillRect(0, 0, width, height);

      const cellW = width / gridSize;
      const cellH = height / gridSize;

      // Draw grid floor
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.03)';
      ctx.lineWidth = 1;
      for (let i = 0; i <= gridSize; i++) {
        ctx.beginPath();
        ctx.moveTo(i * cellW, 0);
        ctx.lineTo(i * cellW, height);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(0, i * cellH);
        ctx.lineTo(width, i * cellH);
        ctx.stroke();
      }

      // Update and draw boxes
      boxes.forEach((b) => {
        if (b.progress >= 1) {
          b.gx = b.targetGx;
          b.gy = b.targetGy;
          b.progress = 0;
          if (Math.random() > 0.95) {
            b.targetGx = Math.floor(Math.random() * gridSize);
            b.targetGy = Math.floor(Math.random() * gridSize);
            b.active = true;
          } else {
            b.active = false;
          }
        }

        if (b.active) {
          b.progress += 0.02;
        }

        // Interpolate position
        const currentX = b.gx + (b.targetGx - b.gx) * b.progress;
        const currentY = b.gy + (b.targetGy - b.gy) * b.progress;

        const x = currentX * cellW + cellW * 0.2;
        const y = currentY * cellH + cellH * 0.2;
        const w = cellW * 0.6;
        const h = cellH * 0.6;

        ctx.fillStyle = b.active ? 'rgba(255, 255, 255, 0.8)' : 'rgba(255, 255, 255, 0.1)';
        ctx.fillRect(x, y, w, h);
        
        if (b.active) {
          ctx.strokeStyle = '#fff';
          ctx.strokeRect(x, y, w, h);
        }
      });

      animationFrameId = requestAnimationFrame(render);
    };

    render();

    return () => {
      window.removeEventListener('resize', resize);
      cancelAnimationFrame(animationFrameId);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="block w-full h-full opacity-60"
      style={{ imageRendering: 'pixelated' }}
    />
  );
}
