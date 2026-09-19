'use client';

import React, { useEffect, useRef } from 'react';

type TopicCanvasProps = {
  slug: string;
};

export default function TopicCanvas({ slug }: TopicCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let width = 0;
    let height = 0;
    let time = 0;

    const resize = () => {
      const parent = canvas.parentElement;
      if (parent) {
        width = parent.clientWidth;
        height = parent.clientHeight;
        // Handle high DPI displays
        const dpr = window.devicePixelRatio || 1;
        canvas.width = width * dpr;
        canvas.height = height * dpr;
        ctx.scale(dpr, dpr);
        canvas.style.width = `${width}px`;
        canvas.style.height = `${height}px`;
      }
    };

    window.addEventListener('resize', resize);
    resize();

    // -- SIMULATIONS -- //

    // 1. Radar Sweep for 'live-fleet-tracking'
    const blips = Array.from({ length: 15 }, () => ({
      x: Math.random(),
      y: Math.random(),
      life: Math.random() * 100,
    }));

    const drawRadar = () => {
      const cx = width / 2;
      const cy = height / 2;
      const radius = Math.min(width, height) * 0.4;
      const angle = time * 0.02;

      // Draw grid
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
      ctx.lineWidth = 1;
      for (let i = 0; i < 5; i++) {
        ctx.beginPath();
        ctx.arc(cx, cy, radius * (i / 4), 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.beginPath();
      ctx.moveTo(cx - radius, cy);
      ctx.lineTo(cx + radius, cy);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(cx, cy - radius);
      ctx.lineTo(cx, cy + radius);
      ctx.stroke();

      // Draw sweep
      ctx.save();
      ctx.translate(cx, cy);
      ctx.rotate(angle);
      const gradient = ctx.createConicGradient(0, 0, 0);
      gradient.addColorStop(0, 'rgba(255, 255, 255, 0.2)');
      gradient.addColorStop(0.1, 'rgba(255, 255, 255, 0)');
      gradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
      
      ctx.fillStyle = gradient;
      ctx.beginPath();
      ctx.moveTo(0, 0);
      ctx.arc(0, 0, radius, 0, Math.PI / 2);
      ctx.lineTo(0, 0);
      ctx.fill();
      
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.5)';
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(0, 0);
      ctx.lineTo(radius, 0);
      ctx.stroke();
      ctx.restore();

      // Draw blips
      blips.forEach(blip => {
        blip.life += 1;
        if (blip.life > 100) {
          blip.x = Math.random();
          blip.y = Math.random();
          blip.life = 0;
        }
        
        const bx = cx - radius + (blip.x * radius * 2);
        const by = cy - radius + (blip.y * radius * 2);
        
        // Only draw if inside circle
        const dist = Math.hypot(bx - cx, by - cy);
        if (dist < radius) {
          const opacity = Math.sin((blip.life / 100) * Math.PI);
          ctx.fillStyle = `rgba(255, 255, 255, ${opacity * 0.8})`;
          ctx.beginPath();
          ctx.arc(bx, by, 3, 0, Math.PI * 2);
          ctx.fill();
          
          // Radar ping rings
          ctx.strokeStyle = `rgba(255, 255, 255, ${opacity * 0.3})`;
          ctx.beginPath();
          ctx.arc(bx, by, (blip.life % 20), 0, Math.PI * 2);
          ctx.stroke();
        }
      });
    };

    // 2. Nodes & Routing for 'dynamic-route-optimization', 'smarter-dispatch'
    const routeNodes = Array.from({ length: 25 }, () => ({
      x: Math.random(),
      y: Math.random(),
      vx: (Math.random() - 0.5) * 0.002,
      vy: (Math.random() - 0.5) * 0.002,
      active: false
    }));

    const drawRouting = () => {
      // Update nodes
      routeNodes.forEach(n => {
        n.x += n.vx;
        n.y += n.vy;
        if (n.x < 0 || n.x > 1) n.vx *= -1;
        if (n.y < 0 || n.y > 1) n.vy *= -1;
      });

      // Find "active" path (simulate TSP/Routing)
      const pathPhase = Math.floor(time * 0.05) % routeNodes.length;
      
      // Draw all connections faint
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.03)';
      ctx.lineWidth = 1;
      for (let i = 0; i < routeNodes.length; i++) {
        for (let j = i + 1; j < routeNodes.length; j++) {
          const dx = routeNodes[i].x - routeNodes[j].x;
          const dy = routeNodes[i].y - routeNodes[j].y;
          if (dx*dx + dy*dy < 0.05) {
            ctx.beginPath();
            ctx.moveTo(routeNodes[i].x * width, routeNodes[i].y * height);
            ctx.lineTo(routeNodes[j].x * width, routeNodes[j].y * height);
            ctx.stroke();
          }
        }
      }

      // Draw optimal path glowing
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
        const isActive = ((pathPhase + 7 * 3) % routeNodes.length) === i; // just random highlight
        ctx.fillStyle = isActive ? '#fff' : 'rgba(255,255,255,0.2)';
        ctx.beginPath();
        ctx.arc(n.x * width, n.y * height, isActive ? 4 : 2, 0, Math.PI * 2);
        ctx.fill();
      });
    };

    // 3. Warehouse Grid for 'warehouse', 'fulfillment'
    const gridSize = 10;
    const boxes = Array.from({ length: 40 }, () => ({
      gx: Math.floor(Math.random() * gridSize),
      gy: Math.floor(Math.random() * gridSize),
      targetGx: Math.floor(Math.random() * gridSize),
      targetGy: Math.floor(Math.random() * gridSize),
      progress: 0,
      active: false
    }));

    const drawWarehouse = () => {
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
      boxes.forEach((b, i) => {
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
    };

    // 4. Default: Data Mesh
    const particles = Array.from({ length: 60 }, () => ({
      x: Math.random(),
      y: Math.random(),
      vx: (Math.random() - 0.5) * 0.005,
      vy: (Math.random() - 0.5) * 0.005,
      size: Math.random() * 2 + 1
    }));

    const drawDefault = () => {
      particles.forEach(p => {
        p.x += p.vx;
        p.y += p.vy;
        if (p.x < 0 || p.x > 1) p.vx *= -1;
        if (p.y < 0 || p.y > 1) p.vy *= -1;
        
        ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
        ctx.beginPath();
        ctx.arc(p.x * width, p.y * height, p.size, 0, Math.PI * 2);
        ctx.fill();
      });

      ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
      for (let i = 0; i < particles.length; i++) {
        for (let j = i + 1; j < particles.length; j++) {
          const dx = particles[i].x - particles[j].x;
          const dy = particles[i].y - particles[j].y;
          const dist = Math.sqrt(dx*dx + dy*dy);
          
          if (dist < 0.15) {
            ctx.lineWidth = 1 - (dist / 0.15);
            ctx.beginPath();
            ctx.moveTo(particles[i].x * width, particles[i].y * height);
            ctx.lineTo(particles[j].x * width, particles[j].y * height);
            ctx.stroke();
          }
        }
      }
    };

    // Render loop
    const render = () => {
      time++;
      
      // Clear background
      ctx.clearRect(0, 0, width, height);

      if (slug.includes('fleet') || slug.includes('tracking')) {
        drawRadar();
      } else if (slug.includes('route') || slug.includes('dispatch') || slug.includes('network')) {
        drawRouting();
      } else if (slug.includes('warehouse') || slug.includes('fulfillment') || slug.includes('inventory')) {
        drawWarehouse();
      } else {
        drawDefault();
      }

      animationFrameId = requestAnimationFrame(render);
    };

    render();

    return () => {
      window.removeEventListener('resize', resize);
      cancelAnimationFrame(animationFrameId);
    };
  }, [slug]);

  return (
    <div className="absolute inset-0 w-full h-full bg-[#030303] flex items-center justify-center">
      {/* Subtle overlay gradient */}
      <div className="absolute inset-0 bg-gradient-to-tr from-black via-transparent to-white/5 pointer-events-none z-10" />
      <canvas
        ref={canvasRef}
        className="block w-full h-full opacity-60"
        style={{ imageRendering: 'pixelated' }}
      />
    </div>
  );
}
