'use client';

import { useEffect, useState } from 'react';

export function usePerformanceMonitor() {
  const [fps, setFps] = useState(60);
  const [isSlowDevice, setIsSlowDevice] = useState(false);

  useEffect(() => {
    let frameCount = 0;
    let lastTime = performance.now();
    let rafId: number;

    const measureFPS = () => {
      frameCount++;
      const currentTime = performance.now();
      
      if (currentTime >= lastTime + 1000) {
        const currentFps = Math.round((frameCount * 1000) / (currentTime - lastTime));
        setFps(currentFps);
        
        // Mark as slow if consistently under 30 FPS
        if (currentFps < 30) {
          setIsSlowDevice(true);
        }
        
        frameCount = 0;
        lastTime = currentTime;
      }
      
      rafId = requestAnimationFrame(measureFPS);
    };

    rafId = requestAnimationFrame(measureFPS);

    return () => cancelAnimationFrame(rafId);
  }, []);

  return { fps, isSlowDevice };
}