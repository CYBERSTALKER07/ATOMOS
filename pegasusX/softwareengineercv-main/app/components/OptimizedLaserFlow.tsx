'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';

// Dynamically import LaserFlow only when needed
const LaserFlow = dynamic(() => import('./LaserFlow'), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full bg-black flex items-center justify-center">
      <div className="border-2 border-white rounded-2xl p-8">
        <div className="animate-pulse text-white">Loading...</div>
      </div>
    </div>
  ),
});

interface OptimizedLaserFlowProps {
  color?: string;
  horizontalBeamOffset?: number;
  verticalBeamOffset?: number;
  flowSpeed?: number;
  wispDensity?: number;
  fogIntensity?: number;
}

export default function OptimizedLaserFlow(props: OptimizedLaserFlowProps) {
  const [shouldRender, setShouldRender] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    // Detect mobile device
    const checkMobile = () => {
      const mobile = window.innerWidth < 768 || 
                     /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
      setIsMobile(mobile);
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);

    // Delay rendering on mobile for better initial load
    const timer = setTimeout(() => {
      setShouldRender(true);
    }, isMobile ? 1000 : 0);

    return () => {
      clearTimeout(timer);
      window.removeEventListener('resize', checkMobile);
    };
  }, [isMobile]);

  // Don't render heavy 3D on very slow devices
  if (!shouldRender) {
    return (
      <div className="w-full h-full bg-black flex items-center justify-center border-2 border-white rounded-2xl">
        <div className="text-white text-xl">Loading Experience...</div>
      </div>
    );
  }

  // Simplified version for mobile
  if (isMobile) {
    return (
      <div className="w-full h-full bg-black relative overflow-hidden rounded-2xl border-2 border-white">
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="w-px h-full bg-white opacity-20 animate-pulse" />
          <div className="w-px h-full bg-white opacity-10 animate-pulse delay-100 ml-8" />
          <div className="w-px h-full bg-white opacity-30 animate-pulse delay-200 ml-8" />
        </div>
      </div>
    );
  }

  // Full LaserFlow for desktop
  return <LaserFlow {...props} dpr={1} />;
}
