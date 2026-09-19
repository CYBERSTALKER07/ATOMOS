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
      // Orbiting blobs
      gsap.to('.blob-orb-1', {
        x: 'random(-20, 20)',
        y: 'random(-20, 20)',
        scale: 'random(0.8, 1.3)',
        duration: 'random(1.5, 3)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      gsap.to('.blob-orb-2', {
        x: 'random(-25, 25)',
        y: 'random(-25, 25)',
        scale: 'random(0.8, 1.4)',
        duration: 'random(2, 3.5)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      gsap.to('.blob-orb-3', {
        x: 'random(-15, 15)',
        y: 'random(-15, 15)',
        scale: 'random(0.7, 1.2)',
        duration: 'random(1.8, 2.8)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      gsap.to('.blob-orb-4', {
        x: 'random(-30, 30)',
        y: 'random(-10, 10)',
        scale: 'random(0.9, 1.5)',
        duration: 'random(2.5, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      // Center blob breathing
      gsap.to('.blob-center', {
        scale: 1.1,
        duration: 2,
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      });

      // Eyes shifting slightly
      gsap.to('.eyes-container', {
        x: 'random(-4, 4)',
        y: 'random(-3, 3)',
        duration: 'random(1, 3)',
        repeat: -1,
        yoyo: true,
        ease: 'power1.inOut'
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
          <circle cx="100" cy="100" r="35" className="blob-orb-2 origin-center" />
          <circle cx="100" cy="100" r="25" className="blob-orb-3 origin-center" />
          <circle cx="100" cy="100" r="28" className="blob-orb-4 origin-center" />
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
