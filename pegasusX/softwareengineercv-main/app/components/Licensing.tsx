'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import GlitchText from './GlitchText';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import { BENTO_THREE } from '../lib/bento';

gsap.registerPlugin(ScrollTrigger);

export default function Licensing() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const cardsRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (sectionRef.current && titleRef.current && contentRef.current && cardsRef.current) {
      
      // Mobile: Simple fade-in only
      if (isMobile) {
        gsap.set([titleRef.current, contentRef.current, cardsRef.current], {
          opacity: 1,
          y: 0,
          scale: 1
        });
        return;
      }

      // Desktop: Scroll-triggered animations
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top 80%',
          end: 'bottom 20%',
          toggleActions: 'play none none reverse'
        }
      });

      timeline
        .fromTo(titleRef.current, 
          { opacity: 0, scale: 0.9 }, 
          { opacity: 1, scale: 1, duration: 1.2 }
        )
        .fromTo(contentRef.current, 
          { opacity: 0, y: 30 }, 
          { opacity: 1, y: 0, duration: 0.8 }, 
          '-=0.4'
        )
        .fromTo(cardsRef.current.children, 
          { opacity: 0, y: 30 }, 
          { opacity: 1, y: 0, duration: 0.6, stagger: 0.2 }, 
          '-=0.3'
        );
    }
  }, [isMobile]);

  const licenses = [
    {
      title: 'Starter',
      description: 'Single-site dispatch and tracking for growing supplier networks.',
      features: ['Dispatch Board', 'Fleet Map', 'Retailer Tracking', 'Email Support']
    },
    {
      title: 'Professional',
      description: 'Multi-site operations with payments, treasury, and priority onboarding.',
      features: ['All Starter Features', 'Payment Integrity', 'Treasury Views', 'Priority Support']
    },
    {
      title: 'Enterprise',
      description: 'Full network rollout with dedicated success team and custom integrations.',
      features: ['All Professional Features', 'Custom Topology', 'Dedicated Support', 'SLA Guarantee']
    }
  ];

  return (
    <section 
      ref={sectionRef} 
      id="licensing" 
      className="min-h-screen py-20 bg-black text-white relative overflow-hidden flex items-center"
    >
      <div className="container mx-auto px-4 relative z-10">
        <div className="max-w-6xl mx-auto">
          {/* Title Section - Disable GlitchText on mobile */}
          <div ref={titleRef} className="text-center mb-16">
            {isMobile ? (
              <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-6 text-white">
                DEPLOYMENT
              </h2>
            ) : (
              <GlitchText
                speed={1}
                enableShadows={true}
                enableOnHover={true}
                className="mb-6"
              >
                DEPLOYMENT
              </GlitchText>
            )}
            <div className="w-20 h-1 bg-white rounded-full mx-auto" />
          </div>

          {/* Content Description */}
          <div ref={contentRef} className="text-center mb-16 max-w-3xl mx-auto">
            <p className="text-lg md:text-xl text-gray-300 leading-relaxed mb-8">
              Platform pillars built for physical logistics — dispatch accuracy, fleet visibility, payment confidence, and network scale
            </p>
            
            {/* Visual Badge Display */}
            <div className="flex flex-wrap justify-center gap-6 mb-12">
              <div className="bg-white text-black px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#A9EBF9] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Dispatch</div>
                <div className="text-sm">Visual Load Planning</div>
              </div>
              
              <div className="bg-black text-white px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#8DDC96] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Tracking</div>
                <div className="text-sm">Live Fleet Maps</div>
              </div>
              
              <div className="bg-white text-black px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#FFA500] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Payments</div>
                <div className="text-sm">Treasury Integrity</div>
              </div>
              
              <div className="bg-black text-white px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#8DDC96] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Realtime</div>
                <div className="text-sm">Live Coordination</div>
              </div>
              
              <div className="bg-white text-black px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#FFDA6F] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Security</div>
                <div className="text-sm">Claims & Audit</div>
              </div>
              
              <div className="bg-black text-white px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#BDE7FF] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Scale</div>
                <div className="text-sm">Multi-Site Networks</div>
              </div>
              
              <div className="bg-white text-black px-6 py-4 rounded-2xl border-2 border-white hover:bg-[#DABDFF] hover:text-black transition-all duration-300">
                <div className="text-2xl font-bold">Roles</div>
                <div className="text-sm">Six-App Parity</div>
              </div>
            </div>
          </div>

          {/* License Cards */}
          <div ref={cardsRef} className="editorial-bento max-w-6xl mx-auto">
            {licenses.map((license, index) => (
              <ContentCard
                key={license.title}
                variant={index === 1 ? 'split' : 'vertical'}
                tone={index === 1 ? 'light' : 'dark'}
                tag="Deployment"
                title={license.title}
                description={license.description}
                image={EDITORIAL_IMAGES[(index + 4) % EDITORIAL_IMAGES.length]}
                href="/join"
                ctaLabel="REQUEST DEMO"
                ctaStyle="button"
                className={BENTO_THREE[index]}
              >
                <ul className="mt-5 space-y-2 text-sm text-inherit opacity-90">
                  {license.features.map((feature) => (
                    <li key={feature} className="flex gap-2">
                      <span aria-hidden="true">—</span>
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
              </ContentCard>
            ))}
          </div>

          {/* Bottom CTA */}
          <div className="text-center mt-12">
            <p className="text-lg text-gray-300 mb-6">
              Have questions about deployment or enterprise rollout?
            </p>
            <a href="/join" className="editorial-btn editorial-btn--inverted">
              GET IN TOUCH
            </a>
          </div>
        </div>
      </div>

      {/* Decorative elements */}
      <div className="absolute top-10 left-10 w-32 h-32 border-2 border-white opacity-10 rounded-2xl" />
      <div className="absolute bottom-10 right-10 w-40 h-40 border-2 border-white opacity-10 rounded-2xl" />
      <div className="absolute top-1/2 left-1/4 w-2 h-2 bg-white rounded-full opacity-30" />
      <div className="absolute top-1/3 right-1/3 w-2 h-2 bg-white rounded-full opacity-30" />
    </section>
  );
}
