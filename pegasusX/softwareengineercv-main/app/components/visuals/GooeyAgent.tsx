'use client';

import React, { useEffect, useRef } from 'react';
import { gsap } from '@/app/lib/gsap';

interface GooeyAgentProps {
  className?: string;
  color?: string;
  size?: number;
}

export default function GooeyAgent({
  className = '',
  color = '#ffffff',
  size = 120,
}: GooeyAgentProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const blob1Ref = useRef<HTMLDivElement>(null);
  const blob2Ref = useRef<HTMLDivElement>(null);
  const blob3Ref = useRef<HTMLDivElement>(null);
  const eyeLeftRef = useRef<HTMLDivElement>(null);
  const eyeRightRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const ctx = gsap.context(() => {
      // Idle floating animation for the gooey blobs
      gsap.to(blob1Ref.current, {
        x: 'random(-10, 10)',
        y: 'random(-10, 10)',
        scale: 'random(0.95, 1.05)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
      });

      gsap.to(blob2Ref.current, {
        x: 'random(-15, 15)',
        y: 'random(-15, 15)',
        scale: 'random(0.85, 1.15)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
        delay: 0.5,
      });

      gsap.to(blob3Ref.current, {
        x: 'random(-10, 10)',
        y: 'random(-20, 20)',
        scale: 'random(0.85, 1.15)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
        delay: 1,
      });

      // Blinking animation for the eyes (blink by scaling Y of the entire eye container? 
      // Actually grok-like big eyes blinking is cute)
      const blinkTl = gsap.timeline({ repeat: -1, repeatDelay: 3.5 });
      blinkTl.to('.grok-eye', { scaleY: 0.1, duration: 0.1, ease: 'power2.in' })
             .to('.grok-eye', { scaleY: 1, duration: 0.1, ease: 'power2.out' });

      // Hover reaction (pupils follow pointer slightly)
      const handleMouseMove = (e: MouseEvent) => {
        if (!containerRef.current) return;
        const bounds = containerRef.current.getBoundingClientRect();
        const mouseX = e.clientX - bounds.left - bounds.width / 2;
        const mouseY = e.clientY - bounds.top - bounds.height / 2;
        
        // Normalize for pupils
        const moveX = (mouseX / bounds.width) * 20; // 20px max movement
        const moveY = (mouseY / bounds.height) * 20;

        gsap.to([eyeLeftRef.current, eyeRightRef.current], {
          x: moveX,
          y: moveY,
          duration: 0.3,
          ease: 'power2.out'
        });
      };

      const handleMouseLeave = () => {
        gsap.to([eyeLeftRef.current, eyeRightRef.current], {
          x: 0,
          y: 0,
          duration: 0.5,
          ease: 'power2.out'
        });
      };

      containerRef.current?.addEventListener('mousemove', handleMouseMove);
      containerRef.current?.addEventListener('mouseleave', handleMouseLeave);

      return () => {
        containerRef.current?.removeEventListener('mousemove', handleMouseMove);
        containerRef.current?.removeEventListener('mouseleave', handleMouseLeave);
      };
    }, containerRef);

    return () => ctx.revert();
  }, []);

  // Calculate proportional sizes based on the 'size' prop
  const eyeWidth = size * 0.28;
  const eyeHeight = size * 0.4;
  const pupilSize = size * 0.12;

  return (
    <div
      ref={containerRef}
      className={`relative flex items-center justify-center cursor-pointer ${className}`}
      style={{ width: size, height: size }}
    >
      {/* SVG Filter for the gooey effect */}
      <svg className="absolute w-0 h-0">
        <defs>
          <filter id="gooey-agent-filter">
            <feGaussianBlur in="SourceGraphic" stdDeviation={size * 0.05} result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 19 -9"
              result="goo"
            />
            <feBlend in="SourceGraphic" in2="goo" />
          </filter>
        </defs>
      </svg>

      {/* Container applying the gooey filter */}
      <div
        className="absolute inset-0 flex items-center justify-center"
        style={{ filter: 'url(#gooey-agent-filter)' }}
      >
        {/* Blob 1 (Main body) */}
        <div
          ref={blob1Ref}
          className="absolute rounded-full"
          style={{
            width: size * 0.65,
            height: size * 0.65,
            backgroundColor: color,
          }}
        />
        {/* Blob 2 (Orbiting blob) */}
        <div
          ref={blob2Ref}
          className="absolute rounded-full"
          style={{
            width: size * 0.5,
            height: size * 0.5,
            backgroundColor: color,
            marginLeft: -size * 0.2,
          }}
        />
        {/* Blob 3 (Orbiting blob) */}
        <div
          ref={blob3Ref}
          className="absolute rounded-full"
          style={{
            width: size * 0.4,
            height: size * 0.4,
            backgroundColor: color,
            marginTop: size * 0.3,
          }}
        />
      </div>

      {/* Big Eyes overlay (Grok-like) */}
      <div className="absolute z-10 flex gap-[10%] pointer-events-none mb-[5%] w-full justify-center">
        {/* Left Eye */}
        <div 
          className="grok-eye relative rounded-full bg-[#111111] overflow-hidden flex shadow-lg"
          style={{ width: eyeWidth, height: eyeHeight }}
        >
          {/* Pupil */}
          <div 
            ref={eyeLeftRef} 
            className="absolute rounded-full bg-white shadow-[0_0_10px_rgba(255,255,255,0.8)]" 
            style={{ 
              width: pupilSize, 
              height: pupilSize, 
              top: '20%', 
              left: '45%',
              transform: 'translateX(-50%)'
            }} 
          />
        </div>

        {/* Right Eye */}
        <div 
          className="grok-eye relative rounded-full bg-[#111111] overflow-hidden flex shadow-lg"
          style={{ width: eyeWidth, height: eyeHeight }}
        >
          {/* Pupil */}
          <div 
            ref={eyeRightRef} 
            className="absolute rounded-full bg-white shadow-[0_0_10px_rgba(255,255,255,0.8)]" 
            style={{ 
              width: pupilSize, 
              height: pupilSize, 
              top: '20%', 
              left: '45%',
              transform: 'translateX(-50%)'
            }} 
          />
        </div>
      </div>
    </div>
  );
}
