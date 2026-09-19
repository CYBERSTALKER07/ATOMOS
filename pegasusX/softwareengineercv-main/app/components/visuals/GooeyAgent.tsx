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
    
    const container = containerRef.current;

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
          transformOrigin: '50% 50%',
          onComplete: () => {
            gsap.delayedCall(gsap.utils.random(2, 6), blink);
          }
        });
      };
      gsap.delayedCall(2, blink);

      // Mouse tracking
      let mouseTimeout: NodeJS.Timeout;

      const handleMouseMove = (e: MouseEvent) => {
        const rect = container.getBoundingClientRect();
        
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
        const eyeMax = 18;
        const blobStretch = 15; // How much the main blob stretches
        const blobMove = 10; // How much it moves
        
        // Animate eyes looking at mouse
        gsap.to('.eyes-container', {
          x: Math.cos(angle) * pull * eyeMax,
          y: Math.sin(angle) * pull * eyeMax,
          duration: 0.4,
          ease: 'power2.out'
        });

        // Stretch and pull the main blob slightly toward the mouse (no gooey filter needed, just clean stretching)
        gsap.to('.blob-center', {
          x: Math.cos(angle) * pull * blobMove,
          y: Math.sin(angle) * pull * blobMove,
          scaleX: 1 + (pull * 0.15),
          scaleY: 1 - (pull * 0.05),
          rotation: angle * (180 / Math.PI), // Rotate to face mouse direction
          transformOrigin: '50% 50%',
          duration: 0.7,
          ease: 'power3.out'
        });

        // Debounce returning to center when mouse stops moving
        clearTimeout(mouseTimeout);
        mouseTimeout = setTimeout(() => {
          returnToCenter();
        }, 2000);
      };

      const returnToCenter = () => {
        gsap.to(['.eyes-container', '.blob-center'], {
          x: 0,
          y: 0,
          scaleX: 1,
          scaleY: 1,
          rotation: 0,
          duration: 1.5,
          ease: 'elastic.out(1, 0.4)'
        });
      };

      const handleMouseLeave = () => {
        clearTimeout(mouseTimeout);
        returnToCenter();
      };

      // Only attach to container, not window, so it only follows when mouse is ON the section
      container.addEventListener('mousemove', handleMouseMove);
      container.addEventListener('mouseleave', handleMouseLeave);

      return () => {
        container.removeEventListener('mousemove', handleMouseMove);
        container.removeEventListener('mouseleave', handleMouseLeave);
        clearTimeout(mouseTimeout);
      };

    }, containerRef);

    return () => ctx.revert();
  }, []);

  return (
    <div 
      ref={containerRef}
      className={`relative flex items-center justify-center ${className}`}
      style={{ width: size, height: size, maxWidth: '100%' }}
    >
      <svg
        viewBox="0 0 200 200"
        width="100%"
        height="100%"
        xmlns="http://www.w3.org/2000/svg"
        className="overflow-visible"
      >
        {/* The Crisp White Blob (Stretches smoothly via GSAP instead of glitchy SVG filters) */}
        <g fill="#ffffff">
          <circle cx="100" cy="100" r="55" className="blob-center" />
        </g>

        {/* The Eyes (Slanted like the screenshot) */}
        <g className="eyes-container" fill="#000000">
          <g style={{ transformOrigin: '90px 100px', transform: 'rotate(25deg)' }}>
            <rect x="74" y="80" width="14" height="32" rx="7" className="eye" />
            <rect x="110" y="80" width="14" height="32" rx="7" className="eye" />
          </g>
        </g>
      </svg>
    </div>
  );
}
