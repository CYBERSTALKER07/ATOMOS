'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Link from 'next/link';
import StaggeredMenu from '@/components/StaggeredMenu';
import ContentCard, { EDITORIAL_IMAGES } from '../components/ContentCard';

gsap.registerPlugin(ScrollTrigger);

export default function DesktopAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const app3Ref = useRef<HTMLDivElement>(null);
  const app4Ref = useRef<HTMLDivElement>(null);
  const app5Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);
  const feature1Ref = useRef<HTMLDivElement>(null);
  const feature2Ref = useRef<HTMLDivElement>(null);
  const feature3Ref = useRef<HTMLDivElement>(null);
  const feature4Ref = useRef<HTMLDivElement>(null);

  const menuItems = [
    { label: 'Home', ariaLabel: 'Go to home page', link: '/' },
    { label: 'About', ariaLabel: 'Learn about me', link: '/#about' },
    { label: 'Projects', ariaLabel: 'View all projects', link: '/#projects' },
    { label: 'Web Apps', ariaLabel: 'View web applications', link: '/web-apps' },
    { label: 'Mobile Apps', ariaLabel: 'View mobile applications', link: '/mobile-apps' },
    { label: 'Desktop Apps', ariaLabel: 'View desktop applications', link: '/desktop-apps' },
    { label: 'Contact', ariaLabel: 'Get in touch', link: '/#contact' }
  ];

  const socialItems = [
    { label: 'GitHub', link: 'https://github.com' },
    { label: 'LinkedIn', link: 'https://linkedin.com' },
    { label: 'Twitter', link: 'https://twitter.com' }
  ];

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

      // App card animations
      const appRefs = [app1Ref, app2Ref, app3Ref, app4Ref, app5Ref];
      
      appRefs.forEach((ref, index) => {
        gsap.from(ref.current, {
          scrollTrigger: {
            trigger: ref.current,
            start: 'top 80%',
            toggleActions: 'play none none reverse',
          },
          x: index % 2 === 0 ? -100 : 100,
          opacity: 0,
          duration: 1,
          ease: 'power3.out',
        });
      });

      // Features section animation - individual cards
      const featureRefs = [feature1Ref, feature2Ref, feature3Ref, feature4Ref];
      
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
      {/* StaggeredMenu */}
      <StaggeredMenu
        position="right"
        items={menuItems}
        socialItems={socialItems}
        displaySocials={true}
        displayItemNumbering={true}
        menuButtonColor="#FFFFFF"
        openMenuButtonColor="#000000"
        changeMenuColorOnOpen={true}
        colors={['#0D0D0D', '#1a1a1a', '#000000']}
        accentColor="#A9EBF9"
        isFixed={true}
        onMenuOpen={() => console.log('Menu opened')}
        onMenuClose={() => console.log('Menu closed')}
      />

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl font-bold mb-8"
          >
            Retailer Desktop
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Native desktop apps for retailer teams — catalog, checkout, and live delivery tracking at the counter
          </p>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8">
        <div className="max-w-7xl mx-auto editorial-grid grid grid-cols-1 lg:grid-cols-2">
          <div ref={app1Ref} className="lg:col-span-2">
            <ContentCard variant="featured" tone="light" eyebrow="DISCOVER THE PLATFORM" title="Retailer Desktop (Tauri)" description="Fast native desktop app for retailer ordering — catalog with zone checks, scheduled delivery windows, checkout, and live shipment tracking from the store counter." image="/original-1137dbc252460b3d33f86e46095e1b12.webp" href="/#contact" ctaLabel="REQUEST DEMO" ctaStyle="button" />
          </div>
          <div ref={app2Ref}>
            <ContentCard variant="split" tag="Supplier" title="Supplier Ops Dashboard" description="Executive view across the network — order vetting queues, dispatch preview, treasury reconciliation, and topology management for multi-site suppliers." image="/original-2096796ee8410886b847d3c0a93c3d4e.webp" href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app3Ref}>
            <ContentCard variant="split" tone="light" tag="Warehouse" title="Warehouse Dispatch Console" description="Desktop-grade dispatch boards for depot managers — visual load planning, fleet map, and gate coordination during peak hours." image="/original-50941087b949fb6cd5a51dc535e09879.webp" href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app4Ref}>
            <ContentCard variant="vertical" tag="Tracking" title="Fleet Command Center" description="Large-screen fleet telemetry with planned-vs-actual routes and deviation alerts for ops teams." image={EDITORIAL_IMAGES[0]} href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app5Ref}>
            <ContentCard variant="vertical" tone="light" tag="Treasury" title="Treasury Workstation" description="Supplier finance teams reconcile card, cash-on-delivery, and driver collections with audit-ready records." image={EDITORIAL_IMAGES[1]} href="/#contact" ctaLabel="READ MORE" />
          </div>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8 bg-black">
        <div className="max-w-5xl mx-auto">
          <ContentCard variant="featured" tone="light" eyebrow="NEXT STEP" title="Desktop Parity for Retailer Teams" description="Give store counters a fast native app with the same catalog, checkout, and tracking as mobile." image={EDITORIAL_IMAGES[2]} href="/join" ctaLabel="REQUEST DEMO" ctaStyle="button" />
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 px-4 md:px-8 border-t-2 border-white">
        <div className="max-w-7xl mx-auto text-center">
          <p className="text-[#C0C0C0]">
            © 2025 Pegasus. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
