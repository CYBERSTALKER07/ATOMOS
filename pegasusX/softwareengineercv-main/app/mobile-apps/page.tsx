'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Link from 'next/link';
import { StaggeredMenu } from '@/components/StaggeredMenu';
import ContentCard, { EDITORIAL_IMAGES } from '../components/ContentCard';

gsap.registerPlugin(ScrollTrigger);

export default function MobileAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);
  const feature1Ref = useRef<HTMLDivElement>(null);
  const feature2Ref = useRef<HTMLDivElement>(null);
  const feature3Ref = useRef<HTMLDivElement>(null);
  const ctaButtonRef = useRef<HTMLAnchorElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      // Hero animations
      gsap.from(titleRef.current, {
        y: 100,
        opacity: 0,
        duration: 1,
        ease: 'power4.out',
      });

      gsap.from(subtitleRef.current, {
        y: 50,
        opacity: 0,
        duration: 1,
        delay: 0.2,
        ease: 'power4.out',
      });

      // CTA Button animation
      gsap.from(ctaButtonRef.current, {
        y: -50,
        opacity: 0,
        duration: 0.8,
        delay: 0.3,
        ease: 'power4.out',
      });

      // App card animations
      gsap.from(app1Ref.current, {
        scrollTrigger: {
          trigger: app1Ref.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        x: -100,
        opacity: 0,
        duration: 1,
        ease: 'power3.out',
      });

      gsap.from(app2Ref.current, {
        scrollTrigger: {
          trigger: app2Ref.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        x: 100,
        opacity: 0,
        duration: 1,
        ease: 'power3.out',
      });

      // Features section animation - individual cards
      const featureRefs = [feature1Ref, feature2Ref, feature3Ref];
      
      featureRefs.forEach((ref, index) => {
        if (ref.current) {
          gsap.from(ref.current, {
            scrollTrigger: {
              trigger: ref.current,
              start: 'top 85%',
              toggleActions: 'play none none reverse',
            },
            y: 50,
            opacity: 0,
            duration: 0.8,
            delay: index * 0.15,
            ease: 'power3.out',
          });
        }
      });
    });

    return () => ctx.revert();
  }, []);

  return (
    <div className="min-h-screen bg-black text-white">
      {/* Fixed Header with CTA Button - positioned to not overlap with menu */}
      <div className="fixed top-8 right-24 z-40 hidden md:block">
        <Link
          ref={ctaButtonRef}
          href="/join"
          className="editorial-btn editorial-btn--shadow"
        >
          Request Demo
        </Link>
      </div>

      {/* StaggeredMenu */}
      <StaggeredMenu
        position="right"
        colors={['#0D0D0D', '#000000']}
        items={[
          { label: 'Home', ariaLabel: 'Go to home page', link: '/' },
          { label: 'Projects', ariaLabel: 'View projects', link: '/#projects' },
          { label: 'About', ariaLabel: 'About me', link: '/#about' },
          { label: 'Contact', ariaLabel: 'Contact me', link: '/#contact' },
        ]}
        socialItems={[
          { label: 'GitHub', link: 'https://github.com' },
          { label: 'LinkedIn', link: 'https://linkedin.com' },
          { label: 'Twitter', link: 'https://twitter.com' },
        ]}
        displaySocials={true}
        displayItemNumbering={true}
        menuButtonColor="#FFFFFF"
        openMenuButtonColor="#000000"
        accentColor="#FFA500"
        isFixed={true}
        changeMenuColorOnOpen={true}
      />

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl xl:text-9xl font-bold mb-8"
          >
            Field Mobile Apps
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl lg:text-3xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Native apps for drivers, warehouse floor teams, and gate operators — built for the field
          </p>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8 lg:px-12">
        <div className="max-w-7xl mx-auto editorial-grid grid grid-cols-1">
          <div ref={app1Ref}>
            <ContentCard
              variant="featured"
              tone="light"
              eyebrow="DISCOVER THE PLATFORM"
              title="Driver Execution"
              description="Route execution stop by stop — sealed manifests, delivery confirmation, cash collection, and live progress back to ops and retailers."
              image="/d43ad1fb95964cffaa808bf7a0364307.webp"
              href="/#contact"
              ctaLabel="REQUEST DEMO"
              ctaStyle="button"
            />
          </div>
          <div ref={app2Ref}>
            <ContentCard
              variant="split"
              tag="Mobile"
              title="Warehouse & Gate"
              description="Android apps for warehouse floor teams and gate operators — dispatch boards, manifest scanning, seal workflows, and live fleet visibility after departure."
              image="/original-66e7443f7182936b9a9732295fbfc121.webp"
              href="/#contact"
              ctaLabel="READ MORE"
              ctaStyle="link"
            />
          </div>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8 lg:px-12 bg-black">
        <div className="max-w-7xl mx-auto">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold text-center mb-16">
            Why Pegasus Mobile?
          </h2>
          <div
            ref={featuresRef}
            className="editorial-grid grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3"
          >
            <div ref={feature1Ref}>
              <ContentCard
                variant="vertical"
                tag="Field Ops"
                title="Field-First"
                description="Clear stop-by-stop workflows for drivers and floor teams under time pressure at peak dispatch hours."
                image={EDITORIAL_IMAGES[3]}
                href="/#contact"
                ctaLabel="READ MORE"
              />
            </div>
            <div ref={feature2Ref}>
              <ContentCard
                variant="vertical"
                tone="light"
                tag="Parity"
                title="Role Parity"
                description="Android and iOS apps share the same contracts — driver, warehouse, and gate teams stay aligned with portal ops."
                image={EDITORIAL_IMAGES[4]}
                href="/#contact"
                ctaLabel="READ MORE"
              />
            </div>
            <div ref={feature3Ref}>
              <ContentCard
                variant="vertical"
                tag="Realtime"
                title="Live Sync"
                description="Status changes propagate within seconds — no manual refresh when dispatch, routes, or payments update."
                image={EDITORIAL_IMAGES[5]}
                href="/#contact"
                ctaLabel="READ MORE"
              />
            </div>
          </div>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8 lg:px-12">
        <div className="max-w-5xl mx-auto">
          <ContentCard
            variant="featured"
            tone="light"
            eyebrow="NEXT STEP"
            title="Ready to Run Field Ops on Pegasus?"
            description="See how driver, warehouse, and gate apps work together on one network."
            image={EDITORIAL_IMAGES[6]}
            href="/join"
            ctaLabel="REQUEST DEMO"
            ctaStyle="button"
          />
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 md:px-8 lg:px-12 border-t-2 border-white">
        <div className="max-w-7xl mx-auto text-center">
          <p className="text-base md:text-lg text-[#C0C0C0]">
            © 2025 Pegasus. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
