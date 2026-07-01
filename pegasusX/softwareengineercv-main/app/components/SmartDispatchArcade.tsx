'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ChamferButton from './ChamferButton';
import { useReducedMotion } from '@/app/hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

export default function SmartDispatchArcade() {
  const sectionRef = useRef<HTMLElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();

  useEffect(() => {
    if (!sectionRef.current || !containerRef.current || reduced) return;
    gsap.fromTo(
      containerRef.current.children,
      { opacity: 0, y: 36 },
      {
        opacity: 1,
        y: 0,
        duration: 0.9,
        stagger: 0.1,
        ease: 'power3.out',
        scrollTrigger: { trigger: sectionRef.current, start: 'top 75%' },
      }
    );
  }, [reduced]);

  return (
    <section ref={sectionRef} className="border-t border-white/10 bg-black py-20 md:py-28 text-white">
      <div className="container mx-auto max-w-7xl px-4">
        <div className="mb-10 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="editorial-eyebrow">Smart dispatch</p>
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Arcade assist — warehouse always in control
            </h2>
            <p className="mt-3 max-w-xl text-sm text-white/55">
              Ranked truck suggestions, never auto-commit. The floor lead confirms every load.
            </p>
          </div>
          <ChamferButton href="/capabilities/smarter-dispatch" variant="ghost">
            Smarter dispatch
          </ChamferButton>
        </div>

        <div ref={containerRef} className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-12 max-w-6xl mx-auto">
          <div className="border border-white/10 p-2 md:p-4 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none">
            <img 
              src="/20260406_0048_Image Generation_remix_01knfjzxcsfwk9h7ty1b7awjkm.png" 
              alt="Smart Dispatch Preview 1" 
              className="w-full h-auto object-cover border border-white/10"
            />
          </div>
          <div className="border border-white/10 p-2 md:p-4 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none">
            <img 
              src="/20260406_0048_Image Generation_remix_01knfjzxcsfwk9h7ty1b7awjkm.png" 
              alt="Smart Dispatch Preview 2" 
              className="w-full h-auto object-cover border border-white/10"
            />
          </div>
        </div>
      </div>
    </section>
  );
}
