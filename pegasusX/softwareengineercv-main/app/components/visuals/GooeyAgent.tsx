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
      // Idle breathing for blobs
      gsap.to('.blob-center', {
        scale: 1.05,
        duration: 2,
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });
      gsap.to('.blob-orbit-1', {
        rotation: 360,
        transformOrigin: '100px 100px',
        duration: 8,
        repeat: -1,
        ease: 'none'
      });
      gsap.to('.blob-orbit-2', {
        rotation: -360,
        transformOrigin: '100px 100px',
        duration: 12,
        repeat: -1,
        ease: 'none'
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

      let mouseTimeout: NodeJS.Timeout;

      const handleMouseMove = (e: MouseEvent) => {
        const rect = container.getBoundingClientRect();
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;
        
        const deltaX = e.clientX - centerX;
        const deltaY = e.clientY - centerY;
        
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        const maxDist = 400; 
        const pull = Math.min(distance / maxDist, 1);
        
        const angle = Math.atan2(deltaY, deltaX);

        const eyeMax = 25;
        const blobPullMax = 40;
        
        gsap.to('.eyes-container', {
          x: Math.cos(angle) * pull * eyeMax,
          y: Math.sin(angle) * pull * eyeMax,
          duration: 0.4,
          ease: 'power2.out'
        });

        // Pull the orbital blobs aggressively toward the mouse
        gsap.to(['.blob-orbit-1-inner', '.blob-orbit-2-inner'], {
          x: Math.cos(angle) * pull * blobPullMax,
          y: Math.sin(angle) * pull * blobPullMax,
          duration: 0.6,
          ease: 'power3.out'
        });
        
        gsap.to('.blob-center', {
          x: Math.cos(angle) * pull * (blobPullMax * 0.5),
          y: Math.sin(angle) * pull * (blobPullMax * 0.5),
          duration: 0.7,
          ease: 'power2.out'
        });

        clearTimeout(mouseTimeout);
        mouseTimeout = setTimeout(() => {
          returnToCenter();
        }, 2000);
      };

      const returnToCenter = () => {
        gsap.to(['.eyes-container', '.blob-orbit-1-inner', '.blob-orbit-2-inner', '.blob-center'], {
          x: 0,
          y: 0,
          duration: 1.5,
          ease: 'elastic.out(1, 0.4)'
        });
      };

      const handleMouseLeave = () => {
        clearTimeout(mouseTimeout);
        returnToCenter();
      };

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
        viewBox="-50 -50 300 300"
        width="100%"
        height="100%"
        xmlns="http://www.w3.org/2000/svg"
        className="overflow-visible"
      >
        <defs>
          {/* Expanded filter bounds to prevent clipping glitches */}
          <filter id="gooey-effect" x="-100%" y="-100%" width="300%" height="300%">
            <feGaussianBlur in="SourceGraphic" stdDeviation="12" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="
                1 0 0 0 0  
                0 1 0 0 0  
                0 0 1 0 0  
                0 0 0 25 -10"
              result="gooey"
            />
            <feComposite in="SourceGraphic" in2="gooey" operator="atop" />
          </filter>
        </defs>

        <g filter="url(#gooey-effect)" fill="#ffffff">
          <circle cx="100" cy="100" r="50" className="blob-center" />
          
          <g className="blob-orbit-1">
            <circle cx="100" cy="65" r="25" className="blob-orbit-1-inner" />
          </g>
          
          <g className="blob-orbit-2">
            <circle cx="65" cy="120" r="20" className="blob-orbit-2-inner" />
          </g>
        </g>

        <g className="eyes-container" fill="#000000">
          <g style={{ transformOrigin: '100px 100px', transform: 'rotate(25deg)' }}>
            <rect x="74" y="80" width="16" height="36" rx="8" className="eye" />
            <rect x="110" y="80" width="16" height="36" rx="8" className="eye" />
          </g>
        </g>
      </svg>
    </div>
  );
}
