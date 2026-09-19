'use client';

import { useEffect, useRef } from 'react';
import { gsap } from '@/app/lib/gsap';
import PageSection from './layout/PageSection';
import { usePerfProfile } from '@/app/hooks/useDevice';
import { useLanguage } from '@/app/context/LanguageContext';
import GooeyAgent from './visuals/GooeyAgent';

export default function SmartDispatchArcade() {
 const sectionRef = useRef<HTMLElement>(null);
 const containerRef = useRef<HTMLDivElement>(null);
 const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
 const reduced = prefersReducedMotion || isLowEnd;
 const { t } = useLanguage();

 useEffect(() => {
   if (!sectionRef.current || !containerRef.current) return;

   if (reduced) {
     gsap.set(containerRef.current.children, { opacity: 1, y: 0 });
     return;
   }

   const ctx = gsap.context(() => {
     gsap.fromTo(
       containerRef.current?.children ? Array.from(containerRef.current.children) : [],
       { opacity: 0, y: 36 },
       {
         opacity: 1,
         y: 0,
         duration: 0.9,
         stagger: 0.1,
         ease: 'pegasus',
         scrollTrigger: { trigger: sectionRef.current, start: 'top 75%', fastScrollEnd: true },
       }
     );
   }, sectionRef);

   return () => ctx.revert();
 }, [reduced]);

 return (
   <PageSection ref={sectionRef}>
     <div ref={containerRef} className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-6xl mx-auto items-center">
       {/* Left Side: Bullet Points */}
       <div className="p-8 md:p-12 bg-[#000000] transition-colors rounded-none flex flex-col justify-center min-h-[300px]">
         <h3 className="text-2xl font-light mb-6">{t('dispatch_key_capabilities', 'Key Capabilities')}</h3>
         <ul className="space-y-4 text-white/70">
           <li className="flex items-start gap-3">
             <span className="text-white mt-1">✓</span>
             <span className="text-base">{t('dispatch_cap_realtime', 'Real-time visibility into supply chain operations')}</span>
           </li>
           <li className="flex items-start gap-3">
             <span className="text-white mt-1">✓</span>
             <span className="text-base">{t('dispatch_cap_predictive', 'Predictive analytics powered by AI/ML')}</span>
           </li>
           <li className="flex items-start gap-3">
             <span className="text-white mt-1">✓</span>
             <span className="text-base">{t('dispatch_cap_erp', 'Seamless integration with existing ERP systems')}</span>
           </li>
         </ul>
       </div>
       
       {/* Right Side: GooeyAgent (Grok Bot) */}
       <div className="p-8 md:p-12 bg-[#000000] flex items-center justify-center min-h-[300px]">
         <GooeyAgent size={280}  />
       </div>
     </div>
   </PageSection>
 );
}
