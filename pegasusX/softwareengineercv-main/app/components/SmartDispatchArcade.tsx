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
              Supplier Control Panel
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
          <div className="border border-white/10 p-8 md:p-12 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none flex flex-col justify-center min-h-[300px]">
             <h3 className="text-2xl font-light mb-6">Key Capabilities</h3>
             <ul className="space-y-4 text-white/70">
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span className="text-base">Real-time visibility into supply chain operations</span>
                </li>
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span className="text-base">Predictive analytics powered by AI/ML</span>
                </li>
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span className="text-base">Seamless integration with existing ERP systems</span>
                </li>
             </ul>
          </div>
          <div className="border border-white/10 p-2 md:p-4 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none flex items-center justify-center">
            <img 
              src="/Gemini_Generated_Image_un3te4un3te4un3t.png" 
              alt="Supplier Control Panel Preview" 
              className="w-full h-full object-cover border border-white/10"
            />
          </div>
        </div>
      </div>
    </section>
  );
}
