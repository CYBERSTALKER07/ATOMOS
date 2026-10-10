'use client';

import React, { useEffect, useRef } from 'react';
import { Timer, Radar, Rocket } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import { gsap, SplitText } from '@/app/lib/gsap';

export default function OurApproach() {
 const { t } = useLanguage();
 const sectionRef = useRef<HTMLElement>(null);
 const svgContainerRef = useRef<HTMLDivElement>(null);
 const headerContentRef = useRef<HTMLDivElement>(null);
 const featuresRef = useRef<HTMLDivElement>(null);

 useEffect(() => {
   if (!sectionRef.current) return;

   const ctx = gsap.context(() => {
     const img = svgContainerRef.current?.querySelector('img');
     if (img) {
       gsap.to(img, {
         yPercent: 15,
         ease: 'none',
         scrollTrigger: {
           trigger: sectionRef.current,
           start: 'top bottom',
           end: 'bottom top',
           scrub: true,
         },
       });
     }

     const title = headerContentRef.current?.querySelector('h2');
     let splitTitle: SplitText | null = null;
     if (title) {
       splitTitle = new SplitText(title, { type: 'lines,words' });
     }

     const tl = gsap.timeline({
       scrollTrigger: {
         trigger: sectionRef.current,
         start: 'top 75%',
         once: true,
       }
     });

     const eyebrow = headerContentRef.current?.querySelector('.eyebrow-container');
     const desc = headerContentRef.current?.querySelector('p');

     if (eyebrow) {
       tl.fromTo(eyebrow, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.8, ease: 'power2.out' });
     }

     if (splitTitle && splitTitle.words) {
       tl.fromTo(
         splitTitle.words,
         { opacity: 0, y: 25, rotateX: -40 },
         { opacity: 1, y: 0, rotateX: 0, duration: 0.8, stagger: 0.05, ease: 'back.out(1.4)' },
         '-=0.4'
       );
     }

     if (desc) {
       tl.fromTo(desc, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.8, ease: 'power2.out' }, '-=0.6');
     }

     const features = featuresRef.current?.children ? Array.from(featuresRef.current.children) : [];
     if (features.length > 0) {
       tl.fromTo(
         features,
         { opacity: 0, y: 30 },
         { opacity: 1, y: 0, duration: 0.8, stagger: 0.15, ease: 'power2.out' },
         '-=0.4'
       );
     }
   }, sectionRef);

   return () => ctx.revert();
 }, []);

 return (
 <section ref={sectionRef} className="w-full flex flex-col lg:flex-row min-h-0 lg:min-h-[640px] xl:min-h-[800px] border-t border-white/10 overflow-hidden">
 <div ref={svgContainerRef} className="flex-1 relative bg-white min-h-[280px] sm:min-h-[300px] md:min-h-[360px] lg:min-h-full overflow-hidden">
 <div className="absolute inset-0 flex items-center justify-center p-8 lg:p-12">
 <img src="/blob-anim.svg" alt="Animated Logo" className="w-full h-full object-contain scale-[1.1]" />
 </div>
 </div>

 <div className="flex-1 flex flex-col bg-black text-white relative border-l border-white/10">
 <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff0a_1px,transparent_1px),linear-gradient(to_bottom,#ffffff0a_1px,transparent_1px)] bg-[size:24px_24px] pointer-events-none" />

 <div className="relative z-10 flex flex-col h-full w-full max-w-[800px]">
 <div ref={headerContentRef} className="p-6 sm:p-8 md:p-12 lg:p-16 border-b border-white/10">
 <div className="eyebrow-container flex items-center gap-4 mb-6 sm:mb-8">
 <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 93.865 16.562" className="h-3 sm:h-4 w-auto fill-white">
 <path d="M 7.998 4 L 80.998 4 L 80.998 13 L 7.998 13 Z" fill="transparent" />
 <path d="M 56.561 14.72 L 40.21 0 L 38.553 1.84 L 54.903 16.561 Z M 64.021 14.72 L 47.672 0 L 46.014 1.84 L 62.364 16.561 Z M 71.482 14.72 L 55.133 0 L 53.475 1.84 L 69.825 16.561 Z M 78.943 14.72 L 62.594 0 L 60.936 1.84 L 77.286 16.561 Z M 86.404 14.72 L 70.055 0 L 68.397 1.84 L 84.747 16.561 Z M 93.865 14.72 L 77.516 0 L 75.858 1.84 L 92.208 16.561 Z M 48.85 14.72 L 32.5 0 L 30.842 1.84 L 47.192 16.561 Z M 41.138 14.72 L 24.79 0 L 23.131 1.84 L 39.481 16.561 Z M 33.428 14.72 L 17.078 0 L 15.421 1.84 L 31.771 16.561 Z M 25.717 14.72 L 9.367 0 L 7.711 1.84 L 24.058 16.562 Z M 18.006 14.72 L 1.656 0 L 0 1.84 L 16.348 16.562 Z" />
 </svg>
 <span className="text-xs tracking-widest font-mono text-white/60 uppercase">
 {t('approach_eyebrow', 'OUR APPROACH')}
 </span>
 </div>

 <h2 className="text-3xl sm:text-4xl md:text-5xl lg:text-6xl font-medium tracking-tight mb-5 sm:mb-8" style={{ perspective: '400px' }}>
 {t('approach_title', 'Built for the long term')}
 </h2>

 <p className="text-base sm:text-lg text-white/70 max-w-xl leading-relaxed">
 {t(
 'approach_desc',
 "We don't just ship software; we build operations infrastructure. Our approach combines field-tested logistics workflows with real-time coordination across six roles."
 )}
 </p>
 </div>

 <div ref={featuresRef} className="flex-1 grid grid-cols-1 md:grid-cols-2">
 <div className="p-6 sm:p-8 md:p-10 border-b md:border-r border-white/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-white/[0.04]">
 <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center ">
 <Timer size={36} className="text-white transition-all duration-300 custom-timer-icon" />
 </div>
 <div>
 <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">
 {t('approach_prime_title', 'SLA-First Dispatch')}
 </h3>
 <p className="text-white/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-white/85">
 {t(
 'approach_prime_desc',
 'We prioritize on-time delivery SLAs to ensure your dispatch engine delivers consistent, measurable results every shift.'
 )}
 </p>
 </div>
 </div>

 <div className="p-6 sm:p-8 md:p-10 border-b border-white/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-white/[0.04]">
 <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center ">
 <Radar size={36} className="text-white transition-all duration-300 custom-radar-icon" />
 </div>
 <div>
 <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">
 {t('approach_clarity_title', 'Total Clarity')}
 </h3>
 <p className="text-white/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-white/85">
 {t(
 'approach_clarity_desc',
 'Full visibility from warehouse dock to retailer doorstep — every order, vehicle, and payment in one live dashboard.'
 )}
 </p>
 </div>
 </div>

 <div className="p-6 sm:p-8 md:p-10 md:border-r border-white/10 flex flex-col gap-4 sm:gap-6 group transition-colors duration-300 hover:bg-white/[0.04]">
 <div className="w-10 h-10 sm:w-12 sm:h-12 flex items-center justify-center ">
 <Rocket size={36} className="text-white transition-all duration-300 custom-rocket-icon" />
 </div>
 <div>
 <h3 className="text-lg sm:text-xl font-medium mb-2 sm:mb-3 font-mono tracking-tight">
 {t('approach_cycles_title', 'Fast Onboarding')}
 </h3>
 <p className="text-white/60 text-sm leading-relaxed transition-colors duration-300 group-hover:text-white/85">
 {t(
 'approach_cycles_desc',
 'Onboard a new warehouse or expand to a new region in weeks, not months, with pre-built role templates and proven workflows.'
 )}
 </p>
 </div>
 </div>

 <div className="p-10 hidden md:block" />
 </div>
 </div>
 </div>
 </section>
 );
}
