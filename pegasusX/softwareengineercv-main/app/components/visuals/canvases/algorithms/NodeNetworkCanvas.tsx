'use client';

import React, { useEffect, useRef } from 'react';

export default function NodeNetworkCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) return;

    let animationFrameId: number;
    let width = 0;
    let height = 0;
    let time = 0;

    const routeNodes = Array.from({ length: 30 }, () => ({
      x: Math.random(),
      y: Math.random(),
      vx: (Math.random() - 0.5) * 0.0015,
      vy: (Math.random() - 0.5) * 0.0015,
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
      time++;
      
      ctx.fillStyle = '#030303';
      ctx.fillRect(0, 0, width, height);

      // Update nodes
      routeNodes.forEach(n => {
        n.x += n.vx;
        n.y += n.vy;
        if (n.x < 0 || n.x > 1) n.vx *= -1;
        if (n.y < 0 || n.y > 1) n.vy *= -1;
      });

      const pathPhase = Math.floor(time * 0.05) % routeNodes.length;
      
      // Draw faint connections
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.03)';
      ctx.lineWidth = 1;
      for (let i = 0; i < routeNodes.length; i++) {
        for (let j = i + 1; j < routeNodes.length; j++) {
          const dx = routeNodes[i].x - routeNodes[j].x;
          const dy = routeNodes[i].y - routeNodes[j].y;
          if (dx*dx + dy*dy < 0.06) {
            ctx.beginPath();
            ctx.moveTo(routeNodes[i].x * width, routeNodes[i].y * height);
            ctx.lineTo(routeNodes[j].x * width, routeNodes[j].y * height);
            ctx.stroke();
          }
        }
      }

      // Draw glowing optimal path
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.4)';
      ctx.lineWidth = 2;
      ctx.beginPath();
      for (let i = 0; i < 8; i++) {
        const idx = (pathPhase + i * 3) % routeNodes.length;
        const n = routeNodes[idx];
        if (i === 0) ctx.moveTo(n.x * width, n.y * height);
        else ctx.lineTo(n.x * width, n.y * height);
      }
      ctx.stroke();

      // Draw nodes
      routeNodes.forEach((n, i) => {
        const isActive = ((pathPhase + 7 * 3) % routeNodes.length) === i;
        ctx.fillStyle = isActive ? '#fff' : 'rgba(255,255,255,0.2)';
        ctx.beginPath();
        ctx.arc(n.x * width, n.y * height, isActive ? 4 : 2, 0, Math.PI * 2);
        ctx.fill();
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
