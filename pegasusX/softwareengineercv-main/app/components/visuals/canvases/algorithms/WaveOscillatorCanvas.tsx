'use client';

import React, { useEffect, useRef } from 'react';

export default function WaveOscillatorCanvas() {
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
      time += 0.02;
      
      ctx.fillStyle = '#030303';
      ctx.fillRect(0, 0, width, height);

      // Draw sine waves
      for (let i = 0; i < 5; i++) {
        ctx.beginPath();
        const yOffset = (i - 2) * (height * 0.15);
        
        for (let x = 0; x < width; x += 10) {
          const normalizedX = x / width;
          // Complex wave combining multiple sine functions
          const y = Math.sin(normalizedX * 10 + time + i) * 30 
                  + Math.sin(normalizedX * 20 - time * 1.5) * 15
                  + Math.sin(normalizedX * 5 + time * 0.5) * 40;
                  
          const drawY = height / 2 + yOffset + y;
          
          if (x === 0) ctx.moveTo(x, drawY);
          else ctx.lineTo(x, drawY);
        }

        const opacity = 0.5 - Math.abs(i - 2) * 0.15;
        ctx.strokeStyle = `rgba(255, 255, 255, ${opacity})`;
        ctx.lineWidth = 2;
        ctx.stroke();
      }

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
