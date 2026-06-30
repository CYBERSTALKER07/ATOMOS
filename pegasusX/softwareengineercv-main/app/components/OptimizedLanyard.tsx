'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';

// Dynamically import Lanyard only when needed
const Lanyard = dynamic(() => import('./Lanyard'), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full bg-black flex items-center justify-center">
      <div className="border-2 border-white rounded-2xl p-8 bg-[#0D0D0D]">
        <div className="animate-pulse text-white">Loading 3D...</div>
      </div>
    </div>
  ),
});

interface OptimizedLanyardProps {
  position?: [number, number, number];
  gravity?: [number, number, number];
  fov?: number;
  transparent?: boolean;
}

export default function OptimizedLanyard(props: OptimizedLanyardProps) {
  const [shouldRender, setShouldRender] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [isInView, setIsInView] = useState(false);

  useEffect(() => {
    // Detect mobile device
    const checkMobile = () => {
      const mobile = window.innerWidth < 1024 || 
                     /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
      setIsMobile(mobile);
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);

    // Use Intersection Observer to only render when in view
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
        }
      },
      { threshold: 0.1 }
    );

    const element = document.querySelector('#lanyard-container');
    if (element) {
      observer.observe(element);
    }

    return () => {
      window.removeEventListener('resize', checkMobile);
      observer.disconnect();
    };
  }, []);

  useEffect(() => {
    if (isInView) {
      const timer = setTimeout(() => {
        setShouldRender(true);
      }, isMobile ? 500 : 100);
      
      return () => clearTimeout(timer);
    }
  }, [isInView, isMobile]);

  // Fallback for mobile - simple image instead of 3D
  if (isMobile) {
    return (
      <div id="lanyard-container" className="w-full h-full bg-black flex items-center justify-center border-2 border-white rounded-2xl">
        <img 
          src="/atom.jpeg" 
          alt="Logo" 
          className="w-48 h-48 object-cover rounded-2xl border-2 border-white shadow-lg"
          loading="lazy"
        />
      </div>
    );
  }

  if (!shouldRender) {
    return (
      <div id="lanyard-container" className="w-full h-full bg-black flex items-center justify-center border-2 border-white rounded-2xl">
        <div className="text-white">Preparing 3D...</div>
      </div>
    );
  }

  return (
    <div id="lanyard-container" className="w-full h-full">
      <Lanyard {...props} />
    </div>
  );
}
