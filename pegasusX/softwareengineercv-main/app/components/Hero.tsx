'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import Image from 'next/image';
import { LaserFlowOptimized } from './Optimized3D';
import CurvedLoop from './CurvedLoop';
import TextType from './TextType';
import ChamferButton from './ChamferButton';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';

export default function Hero() {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  
  const textRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const descRef = useRef<HTMLParagraphElement>(null);
  const ctaRef = useRef<HTMLDivElement>(null);
  const visualRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Skip GSAP animations on mobile - just use simple fade-in
    if (isMobile || prefersReducedMotion) {
      gsap.set([titleRef.current, subtitleRef.current, descRef.current, ctaRef.current, visualRef.current], { 
        opacity: 1,
        x: 0,
        y: 0
      });
      return;
    }

    // Desktop animations only
    const timeline = gsap.timeline({ defaults: { ease: 'power3.out' } });

    timeline
      .fromTo(visualRef.current,
        { opacity: 0, x: 100 }, 
        { opacity: 1, x: 0, duration: 1.2 }
      )
      .fromTo(titleRef.current,
        { opacity: 0, y: 50 },
        { opacity: 1, y: 0, duration: 1 },
        '-=0.8'
      )
      .fromTo(subtitleRef.current,
        { opacity: 0, y: 30 },
        { opacity: 1, y: 0, duration: 0.8 },
        '-=0.6'
      )
      .fromTo(descRef.current,
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.8 },
        '-=0.5'
      )
      .fromTo(ctaRef.current,
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.8 },
        '-=0.3'
      );
  }, [isMobile, prefersReducedMotion]);

  const scrollToNext = () => {
    const nextSection = document.querySelector('#about');
    if (nextSection) {
      nextSection.scrollIntoView({ behavior: 'smooth' });
    }
  };

  return (
    <section id="hero" className="min-h-screen relative flex items-center bg-black overflow-hidden pt-[4.5rem] md:pt-20">
      {/* Corner Loops - Hidden on mobile */}
      {!isMobile && (
        <>
          <div className="absolute top-0 left-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={900}
              direction="right"
              interactive={false}
              className="fill-white"
            />
          </div>

          <div className="absolute top-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={500}
              direction="left"
              interactive={false}
              className="fill-white"
            />
          </div>

          <div className="absolute bottom-0 right-0 w-64 md:w-80 h-20 md:h-24 pointer-events-none opacity-30 z-10 rotate-180 scale-x-[-1]">
            <CurvedLoop
              marqueeText="PEGASUS  "
              speed={1.5}
              curveAmount={200}
              direction="right"
              interactive={false}
              className="fill-white"
            />
          </div>
        </>
      )}

      <div className="container mx-auto px-4 py-20 relative z-20">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center max-w-7xl mx-auto">
          {/* Content Side - Left */}
          <div ref={textRef} className="space-y-8 order-2 lg:order-1">
            <div>
              <h1 
                ref={titleRef} 
                className="text-5xl md:text-6xl lg:text-7xl xl:text-8xl font-bold mb-4 text-white"
              >
                Pegasus
              </h1>
              
              <div ref={subtitleRef} className="mb-6">
                <TextType
                  text={["Logistics Platform", "Dispatch System", "Fleet Tracking", "Payment Confidence"]}
                  typingSpeed={isMobile ? 100 : 75}
                  pauseDuration={1500}
                  deletingSpeed={isMobile ? 70 : 50}
                  showCursor={true}
                  cursorCharacter="|"
                  loop={true}
                  textColors={['#FFFFFF', '#C0C0C0']}
                  className="text-2xl md:text-3xl lg:text-4xl font-light text-white"
                  cursorClassName="text-white font-light"
                />
              </div>
              
              <div className="w-90 h-[0.5px] bg-white mb-6" />
              
              <p 
                ref={descRef} 
                className="text-base md:text-lg font-extralight lg:text-xl text-white leading-relaxed max-w-xl"
              >
                Run supplier-led logistics from one platform — dispatch, tracking, payments,
                and coordination across every team in your network.
              </p>
            </div>

            {/* CTA Buttons */}
            <div ref={ctaRef} className="flex flex-col sm:flex-row gap-3">
              <ChamferButton onClick={scrollToNext} variant="fill">
                Explore Platform
              </ChamferButton>
              <ChamferButton href="/join" variant="ghost">
                Request Demo
              </ChamferButton>
            </div>
          </div>

          {/* Visual Side - Mobile: atom.jpeg, Desktop: LaserFlow */}
          <div ref={visualRef} className="relative order-1 lg:order-2">
            <div className="relative h-[400px] md:h-[500px] lg:h-[600px] rounded-2xl overflow-hidden shadow-2xl bg-black border-2 border-white">
              {isMobile ? (
                /* Mobile: atom.jpeg image */
                <div className="absolute inset-0 flex items-center justify-center p-8">
                  <div className="relative w-full h-full">
                    <Image
                      src="/atom.jpeg"
                      alt="Atom Logo"
                      fill
                      className="object-contain transition-all duration-500 hover:scale-110"
                      priority
                    />
                  </div>
                </div>
              ) : (
                /* Desktop: LaserFlow 3D animation */
                <div className="absolute inset-0">
                  <LaserFlowOptimized
                    color="#FFFFFF"
                    horizontalBeamOffset={0.1}
                    verticalBeamOffset={0.0}
                    flowSpeed={0.1}
                    wispDensity={1.2}
                    fogIntensity={0.5}
                  />
                </div>
              )}
              
              {/* Decorative border overlay - show on mobile only */}
              {isMobile && (
                <div className="absolute inset-0 pointer-events-none">
                  <div className="absolute top-0 left-0 w-20 h-20 border-t-2 border-l-2 border-white" />
                  <div className="absolute top-0 right-0 w-20 h-20 border-t-2 border-r-2 border-white" />
                  <div className="absolute bottom-0 left-0 w-20 h-20 border-b-2 border-l-2 border-white" />
                  <div className="absolute bottom-0 right-0 w-20 h-20 border-b-2 border-r-2 border-white" />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Scroll Indicator */}
      <button
        className="absolute bottom-8 left-1/2 transform -translate-x-1/2 cursor-pointer z-10 hidden md:block group focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-black outline-none rounded-lg p-2"
        onClick={scrollToNext}
        aria-label="Scroll to next section"
      >
        <div className="flex flex-col items-center gap-2 text-white group-hover:text-[#FBFF63] transition-colors duration-300">
          <span className="text-sm font-light tracking-widest">SCROLL</span>
          <svg 
            className="w-6 h-6 animate-bounce" 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path 
              strokeLinecap="round" 
              strokeLinejoin="round" 
              strokeWidth={2} 
              d="M19 14l-7 7m0 0l-7-7m7 7V3"
            />
          </svg>
        </div>
      </button>
    </section>
  );
}
