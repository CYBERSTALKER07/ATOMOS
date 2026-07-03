'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ChamferButton from './ChamferButton';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { DISPATCH_ARCADE_IMAGE } from '@/app/lib/siteAssets';

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
    <PageSection ref={sectionRef}>
        <div className="mb-10 flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <SectionHeader
            eyebrow="Smart dispatch"
            title="Supplier Control Panel"
            description="Ranked truck suggestions, never auto-commit. The floor lead confirms every load."
            className="mb-0"
          />
          <ChamferButton href="/capabilities/smarter-dispatch" variant="ghost" className="shrink-0">
            Smarter dispatch
          </ChamferButton>
        </div>

        <div ref={containerRef} className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-6xl mx-auto">
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
              src={DISPATCH_ARCADE_IMAGE} 
              alt="Supplier Control Panel Preview" 
              className="w-full h-full object-cover border border-white/10"
            />
          </div>
        </div>
    </PageSection>
  );
}
