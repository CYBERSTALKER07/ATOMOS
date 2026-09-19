'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import PageSection from './layout/PageSection';
import { usePerfProfile } from '@/app/hooks/useDevice';
import GooeyAgent from './visuals/GooeyAgent';

export default function SmartDispatchArcade() {
 const sectionRef = useRef<HTMLElement>(null);
 const containerRef = useRef<HTMLDivElement>(null);
 const { isLowEnd, prefersReducedMotion } = usePerfProfile();
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
       { opacity: 0, y: 40 },
       {
         opacity: 1,
         y: 0,
         duration: 1,
         stagger: 0.1,
         ease: 'power3.out',
         scrollTrigger: { trigger: sectionRef.current, start: 'top 85%', fastScrollEnd: true },
       }
     );
   }, sectionRef);

   return () => ctx.revert();
 }, [reduced]);

 return (
   <PageSection ref={sectionRef} className="py-24 bg-[#0a0a0a]">
     <div className="w-full max-w-[1200px] mx-auto px-6">
       <div 
         ref={containerRef} 
         className="w-full bg-[#1c1c1c] rounded-[32px] overflow-hidden relative flex flex-col md:flex-row min-h-[500px]"
       >
         
         {/* Left half: Text */}
         <div className="p-10 md:p-16 lg:p-20 flex flex-col justify-center w-full md:w-[55%] z-10 relative">
           <h2 className="text-[32px] sm:text-[40px] md:text-[48px] font-medium tracking-tight text-white leading-[1.1] mb-6">
             Message Bots like teammates
           </h2>
           <p className="text-[17px] md:text-[20px] text-white/60 font-light leading-[1.5]">
             Give tasks to Bots like you would a teammate on desktop or iOS. Your AI teammates take projects from start to end, keep context on how you work and get smarter over time, and come back when your approval is needed.
           </p>
         </div>

         {/* Right half: The Agent (positioned at bottom right) */}
         <div className="w-full md:w-[45%] h-[300px] md:h-auto relative z-0">
           {/* Position the agent to crop off the bottom right */}
           <div className="absolute -bottom-[20%] -right-[20%] md:-bottom-[40%] md:-right-[30%] w-[120%] h-[120%] flex items-center justify-center">
             <GooeyAgent size={700} />
           </div>
         </div>

       </div>
     </div>
   </PageSection>
 );
}
