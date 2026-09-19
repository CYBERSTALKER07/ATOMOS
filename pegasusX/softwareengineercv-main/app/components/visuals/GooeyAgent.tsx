'use client';

import React, { useEffect, useRef } from 'react';
import { gsap } from 'gsap';

interface GooeyAgentProps {
  size?: number;
  className?: string;
}

export default function GooeyAgent({ size = 200, className = '' }: GooeyAgentProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const ctx = gsap.context(() => {
      // Breathing only (perfect circle idle)
      gsap.to('.blob-center', {
        scale: 1.05,
        duration: 2,
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      // Blinking animation
      const blink = () => {
        gsap.to('.eye', {
          scaleY: 0.1,
          duration: 0.1,
          yoyo: true,
          repeat: 1,
          onComplete: () => {
            gsap.delayedCall(gsap.utils.random(2, 6), blink);
          }
        });
      };
      gsap.delayedCall(2, blink);

      // Mouse tracking
      let mouseTimeout: NodeJS.Timeout;

      const handleMouseMove = (e: MouseEvent) => {
        if (!containerRef.current) return;
        const rect = containerRef.current.getBoundingClientRect();
        
        // Calculate center of the SVG component
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;
        
        // Distance from center
        const deltaX = e.clientX - centerX;
        const deltaY = e.clientY - centerY;
        
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        // Normalize pull based on screen size (max effect at 400px away)
        const maxDist = 400; 
        const pull = Math.min(distance / maxDist, 1);
        
        const angle = Math.atan2(deltaY, deltaX);

        // Max pixel translation within the 200x200 viewBox
        const eyeMax = 14;
        const orb1Max = 35; // Stretches out far
        const orb2Max = 20; // Stretches medium
        const orb3Max = 10; // Stretches short
        
        // Animate eyes looking at mouse
        gsap.to('.eyes-container', {
          x: Math.cos(angle) * pull * eyeMax,
          y: Math.sin(angle) * pull * eyeMax,
          duration: 0.4,
          ease: 'power2.out'
        });

        // Orbs pulling (creates the gooey stretch)
        gsap.to('.blob-orb-1', {
          x: Math.cos(angle) * pull * orb1Max,
          y: Math.sin(angle) * pull * orb1Max,
          scale: 1 - (pull * 0.2), // gets slightly thinner as it stretches
          duration: 0.7,
          ease: 'power3.out'
        });

        gsap.to('.blob-orb-2', {
          x: Math.cos(angle + 0.15) * pull * orb2Max,
          y: Math.sin(angle + 0.15) * pull * orb2Max,
          duration: 0.9,
          ease: 'power3.out'
        });
        
        gsap.to('.blob-orb-3', {
          x: Math.cos(angle - 0.15) * pull * orb3Max,
          y: Math.sin(angle - 0.15) * pull * orb3Max,
          duration: 0.8,
          ease: 'power3.out'
        });

        // Debounce returning to center when mouse stops
        clearTimeout(mouseTimeout);
        mouseTimeout = setTimeout(() => {
          returnToCenter();
        }, 2000);
      };

      const returnToCenter = () => {
        gsap.to(['.eyes-container', '.blob-orb-1', '.blob-orb-2', '.blob-orb-3'], {
          x: 0,
          y: 0,
          scale: 1,
          duration: 1.5,
          ease: 'elastic.out(1, 0.4)'
        });
      };

      const handleMouseLeave = () => {
        clearTimeout(mouseTimeout);
        returnToCenter();
      };

      window.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseleave', handleMouseLeave);

      return () => {
        window.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseleave', handleMouseLeave);
        clearTimeout(mouseTimeout);
      };

    }, containerRef);

    return () => ctx.revert();
  }, []);

  return (
    <div 
      ref={containerRef}
      className={`relative flex items-center justify-center ${className}`}
      style={{ width: size, height: size }}
    >
      <svg
        viewBox="0 0 200 200"
        width="100%"
        height="100%"
        xmlns="http://www.w3.org/2000/svg"
        className="overflow-visible"
      >
        <defs>
          <filter id="gooey-effect" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur in="SourceGraphic" stdDeviation="12" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="
                1 0 0 0 0  
                0 1 0 0 0  
                0 0 1 0 0  
                0 0 0 22 -9"
              result="gooey"
            />
            <feComposite in="SourceGraphic" in2="gooey" operator="atop" />
          </filter>
        </defs>

        {/* The Morphing Gooey White Blob */}
        <g filter="url(#gooey-effect)" fill="#ffffff">
          <circle cx="100" cy="100" r="45" className="blob-center origin-center" />
          <circle cx="100" cy="100" r="30" className="blob-orb-1 origin-center" />
          <circle cx="100" cy="100" r="38" className="blob-orb-2 origin-center" />
          <circle cx="100" cy="100" r="25" className="blob-orb-3 origin-center" />
        </g>

        {/* The Eyes (No Filter, Crisp Edges) */}
        <g className="eyes-container origin-center" fill="#000000">
          <circle cx="82" cy="92" r="7" className="eye origin-center" />
          <circle cx="118" cy="92" r="7" className="eye origin-center" />
        </g>
      </svg>
    </div>
  );
}
