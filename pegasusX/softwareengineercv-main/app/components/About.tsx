'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ASCIIText from './ASCIIText';
import TextType from './TextType';
import { useIsMobile } from '../hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

export default function About() {
  const { isMobile } = useIsMobile();
  const aboutRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const asciiRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (aboutRef.current && imageRef.current && contentRef.current && asciiRef.current) {
      
      // Skip GSAP animations on mobile - simple fade-in only
      if (isMobile) {
        gsap.set([asciiRef.current, imageRef.current, contentRef.current], { 
          opacity: 1, 
          x: 0,
          scale: 1
        });
        return;
      }

      // Desktop: Use scroll-triggered GSAP animations
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: aboutRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse'
        }
      });

      timeline
        .fromTo(asciiRef.current,
          { opacity: 0, scale: 0.9 }, 
          { opacity: 1, scale: 1, duration: 1.2 }
        )
        .fromTo(imageRef.current, 
          { opacity: 0, x: -50 }, 
          { opacity: 1, x: 0, duration: 1 },
          '-=0.8'
        )
        .fromTo(contentRef.current, 
          { opacity: 0, x: 50 }, 
          { opacity: 1, x: 0, duration: 1 }, 
          '-=0.7'
        );
    }
  }, [isMobile]);

  return (
    <section id="about" ref={aboutRef} className="py-20 bg-white text-black">
      <div className="container mx-auto px-4">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center max-w-6xl mx-auto">
          {/* Visual Side - Disable 3D ASCII on mobile */}
          <div ref={imageRef} className="relative">
            <div className="relative rounded-3xl overflow-hidden shadow-2xl border-2 border-black">
              <div ref={asciiRef} className="max-w-7xl mx-auto">
                <div className="relative h-[400px] md:h-[500px] rounded-2xl overflow-hidden border-2 border-black bg-black">
                  {/* Mobile: Simple static content, Desktop: ASCIIText 3D */}
                  {isMobile ? (
                    <div className="w-full h-full flex items-center justify-center bg-gradient-to-br from-gray-900 to-black">
                      <div className="text-white text-9xl font-black opacity-20">X</div>
                    </div>
                  ) : (
                    <ASCIIText
                      text="X"
                      asciiFontSize={20}
                      textFontSize={200}
                      textColor="#ffffff"
                      planeBaseHeight={20}
                      enableWaves={true}
                    />
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Content Side */}
          <div ref={contentRef} className="space-y-6">
            <div>
              <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-4 text-black">
                About Pegasus
              </h2>
              <div className="w-20 h-1 bg-black rounded-full mb-6" />
              
              <div className="mb-6">
                {/* TextType works on mobile */}
                <TextType
                  text={[
                    "Supplier-led logistics networks",
                    "Dispatch, tracking, and payments",
                    "Six roles on one platform",
                    "Operations teams stay aligned"
                  ]}
                  typingSpeed={60}
                  pauseDuration={2000}
                  deletingSpeed={40}
                  showCursor={true}
                  cursorCharacter="_"
                  loop={true}
                  textColors={['#000000', '#C0C0C0']}
                  className="text-xl md:text-2xl font-bold text-black"
                  cursorClassName="text-black"
                  startOnVisible={true}
                />
              </div>
            </div>
            
            <p className="text-lg md:text-xl text-gray-800 leading-relaxed">
              Pegasus is the logistics operating system for supplier-led networks. From morning
              dispatch to live fleet tracking and payment reconciliation, every team — supplier,
              warehouse, factory, driver, retailer, and gate — works from the same source of truth.
            </p>
            
            <div className="flex flex-wrap gap-4">
              <div className="px-6 py-3 bg-black text-white rounded-2xl border-2 border-black hover:bg-white hover:text-black transition-all duration-300">
                Dispatch & Fleet
              </div>
              <div className="px-6 py-3 bg-black text-white rounded-2xl border-2 border-black hover:bg-white hover:text-black transition-all duration-300">
                Payments & Treasury
              </div>
              <div className="px-6 py-3 bg-black text-white rounded-2xl border-2 border-black hover:bg-white hover:text-black transition-all duration-300">
                Realtime Coordination
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
