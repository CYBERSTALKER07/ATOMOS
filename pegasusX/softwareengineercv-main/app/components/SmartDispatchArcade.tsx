'use client';

import { useEffect, useRef } from 'react';
import { gsap } from '@/app/lib/gsap';
import PageSection from './layout/PageSection';
import { usePerfProfile } from '@/app/hooks/useDevice';
import GooeyAgent from './visuals/GooeyAgent';

export default function SmartDispatchArcade() {
 const sectionRef = useRef<HTMLElement>(null);
 const containerRef = useRef<HTMLDivElement>(null);
 const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
 const reduced = prefersReducedMotion || isLowEnd;

 useEffect(() => {
   if (!sectionRef.current || !containerRef.current) return;

   if (reduced) {
     gsap.set(containerRef.current.children, { opacity: 1, y: 0 });
     return;
   }

   const ctx = gsap.context(() => {
     gsap.fromTo(
       containerRef.current?.children ? Array.from(containerRef.current.children) : [],
       { opacity: 0, scale: 0.9 },
       {
         opacity: 1,
         scale: 1,
         duration: 1.2,
         ease: 'pegasus',
         scrollTrigger: { trigger: sectionRef.current, start: 'top 85%', fastScrollEnd: true },
       }
     );
   }, sectionRef);

   return () => ctx.revert();
 }, [reduced]);

 return (
   <PageSection ref={sectionRef}>
     <div ref={containerRef} className="max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
       {/* Full Width Centered GooeyAgent (Grok Bot) */}
       <div className="p-8 md:p-12 bg-[#000000] flex items-center justify-center w-full">
         <GooeyAgent size={400}  />
       </div>
     </div>
   </PageSection>
 );
}
