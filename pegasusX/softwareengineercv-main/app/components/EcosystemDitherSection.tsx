'use client';

import React, { useState, useRef, useEffect } from 'react';
import dynamic from 'next/dynamic';

// Dynamically import the 3D WebGL Dither Stage with SSR disabled
const EcosystemDitherStage = dynamic(
  () => import('./visuals/EcosystemDitherStage'),
  {
    ssr: false,
    loading: () => (
      <div className="w-full h-full flex items-center justify-center bg-[#09090B]">
        <div className="w-2 h-2 bg-white/40 animate-ping" />
      </div>
    ),
  }
);

export default function EcosystemDitherSection() {
  const [activeStep, setActiveStep] = useState<number>(0);
  const sectionRef = useRef<HTMLDivElement>(null);

  // Scroll listener synchronizing normalized progress across the scroll track
  useEffect(() => {
    const el = sectionRef.current;
    if (!el) return;

    let ticking = false;
    const handleScroll = () => {
      if (!ticking) {
        window.requestAnimationFrame(() => {
          if (!el) return;
          const rect = el.getBoundingClientRect();
          const maxScroll = rect.height - window.innerHeight;

          if (maxScroll > 0) {
            const p = Math.min(1, Math.max(0, -rect.top / maxScroll));
            const step = Math.min(3, Math.floor(p * 4));
            setActiveStep(step);
          }
          ticking = false;
        });
        ticking = true;
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll();

    return () => {
      window.removeEventListener('scroll', handleScroll);
    };
  }, []);

  return (
    <section
      id="ecosystem-dither"
      className="relative w-full bg-[#09090B] text-white p-0 m-0 overflow-visible select-none"
    >
      {/* Scroll track for smooth camera progression across the 4 stages */}
      <div ref={sectionRef} className="relative w-full h-[260vh]">
        {/* Sticky Fullscreen 3D Stage - ONLY ANIMATION, ZERO SIDEBARS OR UI */}
        <div className="sticky top-0 w-full h-screen overflow-hidden bg-[#09090B]">
          <EcosystemDitherStage
            activeStep={activeStep}
            theme="dark"
            className="w-full h-full"
          />
        </div>
      </div>
    </section>
  );
}
