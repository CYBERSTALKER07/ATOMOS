'use client';

import React, { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import GlitchText from './GlitchText';

interface SplashScreenProps {
  onComplete?: () => void;
  duration?: number;
}

const SplashScreen: React.FC<SplashScreenProps> = ({ onComplete, duration = 3000 }) => {
  const [isVisible, setIsVisible] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);
  const logoRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      const tl = gsap.timeline();

      // Animate main logo
      tl.from(logoRef.current, {
        scale: 0.8,
        opacity: 0,
        duration: 1.5,
        ease: 'power3.out'
      }, 0.5);

      // Hold for duration
      tl.add(() => {}, `+=${duration / 1000 - 2.3}`);

      // Fade out the entire splash screen
      tl.to(containerRef.current, {
        opacity: 0,
        duration: 0.8,
        ease: 'power2.inOut',
        onComplete: () => {
          setIsVisible(false);
          onComplete?.();
        }
      });
    });

    return () => ctx.revert();
  }, [duration, onComplete]);

  if (!isVisible) return null;

  return (
    <div
      ref={containerRef}
      className="fixed inset-0 z-[9999] bg-black flex items-center justify-center overflow-hidden"
    >
      {/* Main Logo */}
      <div ref={logoRef} className="flex items-center justify-center">
        <img 
          src="/pegasus.jpg" 
          alt="Pegasus Logo" 
          className="max-w-[80vw] max-h-[80vh] object-contain rounded-2xl" 
        />
      </div>
    </div>
  );
};

export default SplashScreen;
