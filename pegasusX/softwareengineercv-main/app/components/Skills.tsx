'use client';

import { useEffect, useRef } from 'react';
import dynamic from 'next/dynamic';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import { useInView } from '../hooks/useInView';
import { bentoPlacement, bentoVariant } from '../lib/bento';

const PixelBlast = dynamic(() => import('./PixelBlast'), { ssr: false });

gsap.registerPlugin(ScrollTrigger);

export default function Skills() {
  const { isMobile } = useIsMobile();
  const { ref: skillsRef, isInView } = useInView<HTMLElement>({ exit: true, rootMargin: '0px' });
  const titleRef = useRef<HTMLDivElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!skillsRef.current) return;

    const ctx = gsap.context(() => {
      if (isMobile) {
        gsap.set([titleRef.current, gridRef.current], { opacity: 1, y: 0 });
        return;
      }

      gsap.timeline({
        scrollTrigger: {
          trigger: skillsRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse',
        },
      })
        .fromTo(titleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 })
        .fromTo(gridRef.current, { opacity: 0, y: 50 }, { opacity: 1, y: 0, duration: 1 }, '-=0.4');
    }, skillsRef);

    return () => ctx.revert();
  }, [isMobile]);

  const capabilityCards = [
    {
      title: 'Dispatch Engine',
      description: 'Visual warehouse boards with smart truck-and-order matching at peak hours',
      tag: 'Operations',
    },
    {
      title: 'Fleet Telemetry',
      description: 'Live maps with planned-vs-actual routes and deviation alerts',
      tag: 'Visibility',
    },
    {
      title: 'Payment Integrity',
      description: 'Checkout, cash collection, and supplier reconciliation in one flow',
      tag: 'Finance',
    },
    {
      title: 'Network Topology',
      description: 'One connected structure for sites, zones, and delivery rules',
      tag: 'Network',
    },
    {
      title: 'Realtime Sync',
      description: 'Instant updates across dispatch boards, apps, and tracking',
      tag: 'Live Ops',
    },
    {
      title: 'Role Parity',
      description: 'Six role apps — portal, mobile, and desktop — on shared contracts',
      tag: 'Platform',
    },
  ];

  return (
    <section ref={skillsRef} className="py-20 bg-black text-white relative overflow-hidden" id="skills">
      {!isMobile && isInView ? (
        <div className="absolute inset-0 opacity-25">
          <PixelBlast
            variant="circle"
            pixelSize={5}
            color="#FFFFFF"
            patternScale={2.5}
            patternDensity={1}
            pixelSizeJitter={0.35}
            enableRipples={false}
            speed={0.45}
            edgeFade={0.3}
            transparent
            autoPauseOffscreen
          />
        </div>
      ) : null}

      <div className="container mx-auto px-4 relative z-10">
        <div ref={titleRef} className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-4 text-white">
            Platform Capabilities
          </h2>
          <div className="w-20 h-1 bg-white mx-auto mb-6" />
          <p className="text-lg md:text-xl text-gray-300 max-w-2xl mx-auto">
            Everything supplier-led networks need to dispatch, track, collect, and coordinate
          </p>
        </div>

        <div ref={gridRef} className="editorial-bento max-w-7xl mx-auto">
          {capabilityCards.map((card, index) => (
            <ContentCard
              key={card.title}
              variant={bentoVariant(index)}
              tag={card.tag}
              title={card.title}
              description={card.description}
              image={EDITORIAL_IMAGES[(index + 2) % EDITORIAL_IMAGES.length]}
              href="#projects"
              ctaLabel="READ MORE"
              ctaStyle="link"
              className={bentoPlacement(index)}
            />
          ))}
        </div>
      </div>
    </section>
  );
}
