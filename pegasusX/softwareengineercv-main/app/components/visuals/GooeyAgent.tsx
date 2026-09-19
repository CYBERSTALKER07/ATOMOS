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
        x: 'random(-15, 15)',
        y: 'random(-15, 15)',
        scale: 'random(0.9, 1.1)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
      });

      gsap.to(blob2Ref.current, {
        x: 'random(-20, 20)',
        y: 'random(-20, 20)',
        scale: 'random(0.8, 1.2)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
        delay: 0.5,
      });

      gsap.to(blob3Ref.current, {
        x: 'random(-10, 10)',
        y: 'random(-25, 25)',
        scale: 'random(0.8, 1.2)',
        duration: 'random(2, 4)',
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut',
        delay: 1,
      });

      // Blinking animation for the eyes
      gsap.to([eyeLeftRef.current, eyeRightRef.current], {
        scaleY: 0.1,
        duration: 0.1,
        repeat: -1,
        repeatDelay: 3,
        yoyo: true,
      });

      // Hover reaction (eyes follow pointer slightly)
      const handleMouseMove = (e: MouseEvent) => {
        const bounds = containerRef.current!.getBoundingClientRect();
        const mouseX = e.clientX - bounds.left - bounds.width / 2;
        const mouseY = e.clientY - bounds.top - bounds.height / 2;
        
        // Normalize
        const moveX = (mouseX / bounds.width) * 10;
        const moveY = (mouseY / bounds.height) * 10;

        gsap.to([eyeLeftRef.current, eyeRightRef.current], {
          x: moveX,
          y: moveY,
          duration: 0.5,
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

      containerRef.current.addEventListener('mousemove', handleMouseMove);
      containerRef.current.addEventListener('mouseleave', handleMouseLeave);

      return () => {
        containerRef.current?.removeEventListener('mousemove', handleMouseMove);
        containerRef.current?.removeEventListener('mouseleave', handleMouseLeave);
      };
    }, containerRef);

    return () => ctx.revert();
  }, []);

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
            <feGaussianBlur in="SourceGraphic" stdDeviation="8" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 18 -7"
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
            width: size * 0.6,
            height: size * 0.6,
            backgroundColor: color,
          }}
        />
        {/* Blob 2 (Orbiting blob) */}
        <div
          ref={blob2Ref}
          className="absolute rounded-full"
          style={{
            width: size * 0.45,
            height: size * 0.45,
            backgroundColor: color,
            marginLeft: -size * 0.2,
          }}
        />
        {/* Blob 3 (Orbiting blob) */}
        <div
          ref={blob3Ref}
          className="absolute rounded-full"
          style={{
            width: size * 0.35,
            height: size * 0.35,
            backgroundColor: color,
            marginTop: size * 0.3,
          }}
        />
      </div>

      {/* Eyes overlay (not gooey, remains sharp) */}
      <div className="absolute z-10 flex gap-2 pointer-events-none mb-1">
        <div
          ref={eyeLeftRef}
          className="w-1.5 h-1.5 rounded-full bg-black"
          style={{ transformOrigin: 'center' }}
        />
        <div
          ref={eyeRightRef}
          className="w-1.5 h-1.5 rounded-full bg-black"
          style={{ transformOrigin: 'center' }}
        />
      </div>
    </div>
  );
}
