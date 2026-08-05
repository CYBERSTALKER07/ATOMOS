'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import TextType from './TextType';
import DigitalizedImage from './DigitalizedImage';
import { useIsMobile } from '../hooks/useDevice';
import PageSection from './layout/PageSection';

gsap.registerPlugin(ScrollTrigger);

const PEGASUS_LOGO = '/pegasus.jpg';

export default function About() {
  const { isMobile } = useIsMobile();
  const aboutRef = useRef<HTMLElement>(null);
  const imageRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!aboutRef.current) return;

    const ctx = gsap.context(() => {
      if (isMobile) {
        gsap.set([imageRef.current, contentRef.current], {
          opacity: 1,
          x: 0,
          scale: 1,
        });
        return;
      }

      gsap.timeline({
        scrollTrigger: {
          trigger: aboutRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse',
        },
      })
        .fromTo(imageRef.current, { opacity: 0, x: -50 }, { opacity: 1, x: 0, duration: 1 })
        .fromTo(contentRef.current, { opacity: 0, x: 50 }, { opacity: 1, x: 0, duration: 1 }, '-=0.7');
    }, aboutRef);

    return () => ctx.revert();
  }, [isMobile]);

  return (
    <PageSection id="about" ref={aboutRef}>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 md:gap-10 lg:gap-12 items-center">
        <div ref={imageRef} className="relative">
          <div className="relative h-[240px] sm:h-[320px] md:h-[400px] lg:h-[500px] overflow-hidden border border-white/10 bg-black">
            <DigitalizedImage
              src={PEGASUS_LOGO}
              alt="Pegasus"
              color="#e8e4e3"
              backgroundColor="#000000"
              threshold={0.2}
            />
          </div>
        </div>

        <div ref={contentRef} className="space-y-6">
          <div>
            <h2 className="text-4xl md:text-5xl lg:text-6xl font-light mb-4 text-white">
              About Pegasus
            </h2>
            <div className="w-20 h-[0.5px] bg-white rounded-full mb-6" />

            <div className="mb-6">
              <TextType
                text={[
                  'Supplier-led logistics networks',
                  'Dispatch, tracking, and payments',
                  'Six roles on one platform',
                  'Operations teams stay aligned',
                ]}
                typingSpeed={60}
                pauseDuration={2000}
                deletingSpeed={40}
                showCursor={true}
                cursorCharacter="_"
                loop={true}
                textColors={['#ffffff', '#a3a3a3']}
                className="text-xl md:text-2xl font-light text-white"
                cursorClassName="text-white"
                startOnVisible={true}
              />
            </div>
          </div>

          <p className="text-lg md:text-xl text-white/65 leading-relaxed font-light">
            Pegasus is the logistics operating system for supplier-led networks. From morning
            dispatch to live fleet tracking and payment reconciliation, every team — supplier,
            warehouse, factory, driver, retailer, and gate — works from the same source of truth.
          </p>

          <div className="flex flex-wrap gap-4">
            <Link href="/solutions/visual-dispatch-engine" className="editorial-btn editorial-btn--sm">
              Dispatch & Fleet
            </Link>
            <Link href="/capabilities/payment-confidence" className="editorial-btn editorial-btn--sm">
              Payments & Treasury
            </Link>
            <Link href="/capabilities/instant-coordination" className="editorial-btn editorial-btn--sm">
              Live Coordination
            </Link>
          </div>
        </div>
      </div>
    </PageSection>
  );
}
