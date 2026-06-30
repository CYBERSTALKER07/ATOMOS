'use client';

import dynamic from 'next/dynamic';
import { useIsMobile } from '../hooks/useDevice';

// Lazy load 3D components with no SSR
const LaserFlowComponent = dynamic(() => import('./LaserFlow').then(mod => ({ default: mod.LaserFlow })), {
  ssr: false,
  loading: () => <div className="w-full h-full bg-black" />
});

const LanyardComponent = dynamic(() => import('./Lanyard'), {
  ssr: false,
  loading: () => <div className="w-full h-full bg-black" />
});

export const LaserFlowOptimized = (props: any) => {
  const { isMobile, isTablet } = useIsMobile();
  
  // Skip on very small screens or low-end devices
  if (typeof window !== 'undefined' && isMobile) {
    const isLowEnd = /iPhone [5-8]|iPad2|iPad3|iPad4|iPod|Android [4-5]/i.test(navigator.userAgent);
    if (isLowEnd) {
      return <div className="w-full h-full bg-black" />;
    }
  }

  return <LaserFlowComponent {...props} />;
};

export const LanyardOptimized = (props: any) => {
  const { isMobile } = useIsMobile();
  
  // Reduce quality on mobile
  const optimizedProps = {
    ...props,
    gravity: isMobile ? [0, -20, 0] : props.gravity || [0, -40, 0],
    fov: isMobile ? 30 : props.fov || 20,
  };

  return <LanyardComponent {...optimizedProps} />;
};