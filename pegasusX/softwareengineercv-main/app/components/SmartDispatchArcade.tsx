'use client';

import { useEffect, useRef } from 'react';
import { gsap } from '@/app/lib/gsap';
import PageSection from './layout/PageSection';
import { usePerfProfile } from '@/app/hooks/useDevice';
import GooeyAgent from './visuals/GooeyAgent';
import { useLanguage } from '@/app/context/LanguageContext';

export default function SmartDispatchArcade() {
 const sectionRef = useRef<HTMLElement>(null);
 const containerRef = useRef<HTMLDivElement>(null);
 const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
 const { language } = useLanguage();
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
       { opacity: 0, scale: 0.95 },
       {
         opacity: 1,
         scale: 1,
         duration: 1.2,
         stagger: 0.2,
         ease: 'pegasus',
         scrollTrigger: { trigger: sectionRef.current, start: 'top 85%', fastScrollEnd: true },
       }
     );
   }, sectionRef);

   return () => ctx.revert();
 }, [reduced]);

 return (
   <PageSection ref={sectionRef} className="border-t border-white/10 bg-black overflow-hidden">
     <div ref={containerRef} className="w-full max-w-[1560px] mx-auto min-h-[600px] grid grid-cols-1 lg:grid-cols-2">
       
       {/* Left half: Text */}
       <div className="p-8 sm:p-12 lg:p-24 flex flex-col justify-center border-b lg:border-b-0 lg:border-r border-white/10 z-10 relative bg-black">
         <div className="mb-8 flex items-center">
            <div className="h-1.5 w-1.5 bg-white mr-3"></div>
            <span className="font-mono text-[10px] uppercase tracking-widest text-white/50">
              PEGASUS INTELLIGENCE
            </span>
         </div>
         <h2 className="text-4xl sm:text-5xl lg:text-[5rem] font-medium tracking-tight text-white leading-[1.05] mb-8">
           Autonomous<br/>Supply Chain<br/>Agent
         </h2>
         <p className="text-xl font-light text-white/50 max-w-lg leading-relaxed">
           The engine behind your operations. Grok-powered intelligence dynamically learning, routing, and optimizing your logistics network in real-time.
         </p>
       </div>

       {/* Right half: Massive Gooey Bot */}
       <div className="p-8 md:p-12 bg-[#050505] flex items-center justify-center w-full min-h-[500px] lg:min-h-[800px] relative overflow-hidden">
         <div className="absolute inset-0 bg-gradient-to-br from-white/[0.02] to-transparent pointer-events-none" />
         <GooeyAgent size={700} />
       </div>

     </div>
   </PageSection>
 );
}
