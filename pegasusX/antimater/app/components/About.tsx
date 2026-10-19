'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap, SplitText } from '@/app/lib/gsap';
import TextType from './TextType';
import Dither from './visuals/Dither';
import { usePerfProfile } from '../hooks/useDevice';
import PageSection from './layout/PageSection';
import { useLanguage } from '../context/LanguageContext';

export default function About() {
 const { isMobile, isLowEnd, prefersReducedMotion } = usePerfProfile();
 const { t, language } = useLanguage();
 const aboutRef = useRef<HTMLElement>(null);
 const imageRef = useRef<HTMLDivElement>(null);
 const contentRef = useRef<HTMLDivElement>(null);

 useEffect(() => {
 if (!aboutRef.current) return;

 const ctx = gsap.context(() => {
 if (isMobile || isLowEnd || prefersReducedMotion) {
 gsap.set([imageRef.current, contentRef.current], {
 opacity: 1,
 x: 0,
 scale: 1,
 });
 return;
 }

 const heading = contentRef.current?.querySelector('h2');
 let splitHeading: SplitText | null = null;
 if (heading) {
   splitHeading = new SplitText(heading, { type: 'lines' });
 }

 const tl = gsap.timeline({
 scrollTrigger: {
 trigger: aboutRef.current,
 start: 'top 80%',
 end: 'bottom 20%',
 toggleActions: 'play none none reverse',
 fastScrollEnd: true,
 },
 });

 tl.fromTo(imageRef.current, 
  { opacity: 0, scale: 0.8, filter: 'blur(10px)' }, 
  { opacity: 1, scale: 1, filter: 'blur(0px)', duration: 1.2, ease: 'power3.out' }
 );

 const textElements = contentRef.current ? Array.from(contentRef.current.children) : [];
 
 if (splitHeading && splitHeading.lines) {
   tl.fromTo(splitHeading.lines, 
     { opacity: 0, y: 30 }, 
     { opacity: 1, y: 0, duration: 0.8, stagger: 0.1, ease: 'power2.out' },
     '-=0.8'
   );
 }

 tl.fromTo(textElements.filter(el => el !== heading?.parentElement), 
   { opacity: 0, x: 30 }, 
   { opacity: 1, x: 0, duration: 0.8, stagger: 0.1, ease: 'power2.out' },
   '-=0.6'
 );

 gsap.to(imageRef.current, {
   yPercent: 15,
   ease: 'none',
   scrollTrigger: {
     trigger: aboutRef.current,
     start: 'top bottom',
     end: 'bottom top',
     scrub: 1,
   }
 });
 }, aboutRef);

 return () => ctx.revert();
 }, [isMobile, isLowEnd, prefersReducedMotion]);

 return (
 <PageSection id="about" ref={aboutRef}>
 <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 md:gap-10 lg:gap-12 items-center">
 <div ref={imageRef} className="relative">
 <div className="relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px] overflow-hidden bg-black rounded-lg" style={{ 
   width: '100%',
   WebkitMaskImage: 'url(/productionlogo-mask.png?v=2)',
   WebkitMaskSize: 'contain',
   WebkitMaskRepeat: 'no-repeat',
   WebkitMaskPosition: 'center',
   maskImage: 'url(/productionlogo-mask.png?v=2)',
   maskSize: 'contain',
   maskRepeat: 'no-repeat',
   maskPosition: 'center'
}}>
   <Dither
     waveColor={[0.06274509803921569,0.7254901960784313,0.5058823529411764]}
     disableAnimation={false}
     enableMouseInteraction={true}
     mouseRadius={0.3}
     colorNum={4}
     waveAmplitude={0.3}
     waveFrequency={3}
     waveSpeed={0.05}
     backgroundColor={[0,0,0]}
   />
 </div>
 </div>

 <div ref={contentRef} className="space-y-6">
 <div>
 <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-4 text-zinc-900 dark:text-white">
 {t('about_title')}
 </h2>
 <div className="w-20 h-[0.5px] bg-zinc-900 dark:bg-white rounded-none mb-6" />

 <div className="mb-6">
 <TextType
 key={language}
 text={[
 t('about_type_1'),
 t('about_type_2'),
 t('about_type_3'),
 t('about_type_4'),
 ]}
 typingSpeed={60}
 pauseDuration={2000}
 deletingSpeed={40}
 showCursor={true}
 cursorCharacter="_"
 loop={true}
 className="text-xl md:text-2xl font-light text-zinc-900 dark:text-white"
 cursorClassName="text-zinc-900 dark:text-white"
 startOnVisible={true}
 />
 </div>
 </div>

 <p className="text-lg md:text-xl text-zinc-600 dark:text-white/65 leading-relaxed font-light">
 {t('about_desc')}
 </p>

 <div className="flex flex-wrap gap-4">
 <Link href="/join" className="editorial-btn editorial-btn--sm">
 {t('about_btn_join_us', 'Join Us')}
 </Link>
 </div>
 </div>
 </div>
 </PageSection>
 );
}
