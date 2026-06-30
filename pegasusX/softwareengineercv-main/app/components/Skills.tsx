'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import MagicBento from './MagicBento';
import PixelBlast from './PixelBlast';
import { useIsMobile } from '../hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

export default function Skills() {
  const { isMobile } = useIsMobile();
  const skillsRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const bentoRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (skillsRef.current && titleRef.current && bentoRef.current) {
      
      // Mobile: Simple fade-in
      if (isMobile) {
        gsap.set([titleRef.current, bentoRef.current], { 
          opacity: 1, 
          y: 0 
        });
        return;
      }

      // Desktop: Scroll-triggered animations
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: skillsRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse'
        }
      });

      timeline
        .fromTo(titleRef.current, 
          { opacity: 0, y: 30 }, 
          { opacity: 1, y: 0, duration: 0.8 }
        )
        .fromTo(bentoRef.current, 
          { opacity: 0, y: 50 }, 
          { opacity: 1, y: 0, duration: 1 }, 
          '-=0.4'
        );
    }
  }, [isMobile]);

  const skillCards = [
    {
      color: '#000000',
      title: 'Dispatch Engine',
      description: 'Visual warehouse boards with smart truck-and-order matching at peak hours',
      label: 'Operations'
    },
    {
      color: '#1a1a1a',
      title: 'Fleet Telemetry',
      description: 'Live maps with planned-vs-actual routes and deviation alerts',
      label: 'Visibility'
    },
    {
      color: '#0a0a0a',
      title: 'Payment Integrity',
      description: 'Checkout, cash collection, and supplier reconciliation in one flow',
      label: 'Finance'
    },
    {
      color: '#000000',
      title: 'Network Topology',
      description: 'One connected structure for sites, zones, and delivery rules',
      label: 'Network'
    },
    {
      color: '#1a1a1a',
      title: 'Realtime Sync',
      description: 'Instant updates across dispatch boards, apps, and tracking',
      label: 'Live Ops'
    },
    {
      color: '#0a0a0a',
      title: 'Role Parity',
      description: 'Six role apps — portal, mobile, and desktop — on shared contracts',
      label: 'Platform'
    }
  ];

  const skills = [
    { 
      name: 'React & Next.js', 
      level: 95, 
      icon: '⚛️',
      color: 'hover-neon-blue'
    },
    { 
      name: 'TypeScript', 
      level: 90, 
      icon: '📘',
      color: 'hover-neon-cyan'
    },
    { 
      name: 'Node.js', 
      level: 88, 
      icon: '🟢',
      color: 'hover-neon-green'
    },
    { 
      name: 'Three.js & WebGL', 
      level: 85, 
      icon: '🎨',
      color: 'hover-neon-purple'
    },
    { 
      name: 'GSAP Animation', 
      level: 92, 
      icon: '✨',
      color: 'hover-neon-yellow'
    },
    { 
      name: 'Tailwind CSS', 
      level: 95, 
      icon: '🎨',
      color: 'hover-neon-pink'
    },
    { 
      name: 'PostgreSQL', 
      level: 80, 
      icon: '🐘',
      color: 'hover-neon-orange'
    },
    { 
      name: 'Docker & AWS', 
      level: 82, 
      icon: '☁️',
      color: 'hover-neon-red'
    }
  ];

  return (
    <section ref={skillsRef} className="py-20 bg-black text-white relative overflow-hidden" id="skills">
      {/* Background PixelBlast - Disabled on mobile */}
      {!isMobile && (
        <div className="absolute inset-0 opacity-30">
          <PixelBlast
            variant="circle"
            pixelSize={6}
            color="#FFFFFF"
            patternScale={3}
            patternDensity={1.2}
            pixelSizeJitter={0.5}
            enableRipples
            rippleSpeed={0.4}
            rippleThickness={0.12}
            rippleIntensityScale={1.5}
            liquid
            liquidStrength={0.12}
            liquidRadius={1.2}
            liquidWobbleSpeed={5}
            speed={0.6}
            edgeFade={0.25}
            transparent
          />
        </div>
      )}

      <div className="container mx-auto px-4 relative z-10">
        <div ref={titleRef} className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-4 text-white">
            Platform Capabilities
          </h2>
          <div className="w-20 h-1 bg-white rounded-full mx-auto mb-6" />
          <p className="text-lg md:text-xl text-gray-300 max-w-2xl mx-auto">
            Everything supplier-led networks need to dispatch, track, collect, and coordinate
          </p>
        </div>

        <div ref={bentoRef}>
          <MagicBento
            textAutoHide={true}
            enableStars={!isMobile}
            enableSpotlight={!isMobile}
            enableBorderGlow={!isMobile}
            enableTilt={!isMobile}
            enableMagnetism={!isMobile}
            clickEffect={!isMobile}
            cards={skillCards}
          />
        </div>
      </div>
    </section>
  );
}
