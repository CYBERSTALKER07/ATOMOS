'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Link from 'next/link';
import { FiArrowLeft } from 'react-icons/fi';
import ContentCard, { EDITORIAL_IMAGES } from '../components/ContentCard';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

gsap.registerPlugin(ScrollTrigger);

export default function WebAppsPage() {
  const heroRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLParagraphElement>(null);
  const app1Ref = useRef<HTMLDivElement>(null);
  const app2Ref = useRef<HTMLDivElement>(null);
  const app3Ref = useRef<HTMLDivElement>(null);
  const app4Ref = useRef<HTMLDivElement>(null);
  const app5Ref = useRef<HTMLDivElement>(null);
  const featuresRef = useRef<HTMLDivElement>(null);

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

      // Features section animation
      gsap.from(featuresRef.current?.children || [], {
        scrollTrigger: {
          trigger: featuresRef.current,
          start: 'top 80%',
          toggleActions: 'play none none reverse',
        },
        y: 50,
        opacity: 0,
        duration: 0.8,
        stagger: 0.2,
        ease: 'power3.out',
      });
    });

    return () => ctx.revert();
  }, []);

  return (
    <div className="min-h-screen bg-black text-white">
      {/* Back Button */}
      <Link href="/" className="editorial-btn editorial-btn--shadow fixed top-8 left-8 z-50">
        <FiArrowLeft size={20} />
        Back
      </Link>

      {/* Hero Section */}
      <section ref={heroRef} className="min-h-screen flex items-center justify-center px-4 md:px-8 pt-24 md:pt-0">
        <div className="max-w-6xl mx-auto text-center">
          <h1
            ref={titleRef}
            className="text-5xl md:text-7xl lg:text-8xl font-light mb-8"
          >
            Operations Portals
          </h1>
          <p
            ref={subtitleRef}
            className="text-xl md:text-2xl text-[#C0C0C0] max-w-3xl mx-auto"
          >
            Web portals for suppliers, warehouses, factories, and retailers — dispatch boards, treasury, and live ops
          </p>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8">
        <div className="max-w-7xl mx-auto editorial-grid grid grid-cols-1 lg:grid-cols-2">
          <div ref={app1Ref} className="lg:col-span-2">
            <ContentCard
              variant="featured"
              tone="light"
              eyebrow="DISCOVER THE PLATFORM"
              title="Supplier Control Plane"
              description="Network oversight for suppliers — order vetting, dispatch preview, topology management, and treasury views across warehouses, factories, and retailers."
              image={SITE_IMAGES.logisticsPlatformUi}
              href="/#contact"
              ctaLabel="REQUEST DEMO"
              ctaStyle="button"
            />
          </div>
          <div ref={app2Ref}>
            <ContentCard variant="split" tag="Warehouse" title="Warehouse Dispatch Board" description="Visual morning dispatch with truck-and-order matching, capacity planning, gate seal workflow, and live fleet map after departure." image={SITE_IMAGES.warehouseAutomation} href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app3Ref}>
            <ContentCard variant="split" tone="light" tag="Retailer" title="Retailer Commerce Portal" description="Catalog browsing, checkout, delivery scheduling, and live order tracking — self-serve for retailer teams without calling support." image={SITE_IMAGES.multimodalHub} href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app4Ref}>
            <ContentCard variant="vertical" tag="Fleet" title="Fleet Telemetry Map" description="Live fleet map with planned-vs-actual routes, deviation alerts for ops, and retailer self-serve tracking when deliveries run late." image={EDITORIAL_IMAGES[0]} href="/#contact" ctaLabel="READ MORE" />
          </div>
          <div ref={app5Ref}>
            <ContentCard variant="vertical" tag="Finance" title="Payment Integrity" description="Checkout through driver collection to supplier treasury — duplicate protection and a clear audit trail across the network." image={EDITORIAL_IMAGES[1]} href="/#contact" ctaLabel="READ MORE" />
          </div>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8 bg-black">
        <div className="max-w-7xl mx-auto">
          <h2 className="text-4xl md:text-5xl font-light text-center mb-16">Why Pegasus Portals?</h2>
          <div ref={featuresRef} className="editorial-grid grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4">
            <ContentCard variant="vertical" tag="Performance" title="High Performance" description="Optimized ops boards with realtime refresh and efficient caching during peak dispatch." image={EDITORIAL_IMAGES[2]} href="/#contact" ctaLabel="READ MORE" />
            <ContentCard variant="vertical" tone="light" tag="Design" title="Role-Ready UX" description="Interfaces tuned for warehouse, supplier, and retailer teams — not generic admin templates." image={EDITORIAL_IMAGES[3]} href="/#contact" ctaLabel="READ MORE" />
            <ContentCard variant="vertical" tag="Architecture" title="Shared Contracts" description="Portal and mobile apps read the same network truth — dispatch, tracking, and payments stay aligned." image={EDITORIAL_IMAGES[4]} href="/#contact" ctaLabel="READ MORE" />
            <ContentCard variant="vertical" tone="light" tag="Scale" title="Multi-Site Networks" description="Built for suppliers running many warehouses, factories, and delivery zones on one platform." image={EDITORIAL_IMAGES[5]} href="/#contact" ctaLabel="READ MORE" />
          </div>
        </div>
      </section>

      <section className="py-20 px-4 md:px-8">
        <div className="max-w-5xl mx-auto">
          <ContentCard variant="featured" tone="light" eyebrow="NEXT STEP" title="Run Your Network from One Control Plane" description="See supplier, warehouse, and retailer portals working together on Pegasus." image={EDITORIAL_IMAGES[6]} href="/join" ctaLabel="REQUEST DEMO" ctaStyle="button" />
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
