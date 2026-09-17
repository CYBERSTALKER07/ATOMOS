'use client';

import React, { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { SITE_IMAGES } from '@/app/lib/siteAssets';
import { useReducedMotion } from '@/app/hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

export interface OrbitItem {
  id: string;
  image: string;
  title: string;
  category: string;
  href: string;
}

const ORBIT_ITEMS: OrbitItem[] = [
  {
    id: 'dispatch-hub',
    image: SITE_IMAGES.truckTerminal,
    title: 'Smart Dispatch Hub',
    category: 'TERMINAL DISPATCH',
    href: '/platform',
  },
  {
    id: 'operations-board',
    image: SITE_IMAGES.logisticsPlatformUi,
    title: 'Operations Control Board',
    category: 'CROSS-DOCK ORCHESTRATION',
    href: '/capabilities',
  },
  {
    id: 'fleet-telematics',
    image: SITE_IMAGES.multimodalHub,
    title: 'Multi-Tenant Fleet Telematics',
    category: 'REALTIME GPS & CAN-BUS',
    href: '/capabilities/live-fleet-tracking',
  },
  {
    id: 'warehouse-staging',
    image: SITE_IMAGES.warehouseAutomation,
    title: 'Warehouse Staging & Automation',
    category: 'WMS & INVENTORY FLOW',
    href: '/roles/warehouse',
  },
  {
    id: 'supplier-network',
    image: SITE_IMAGES.pegasusContainer,
    title: 'Sovereign Supplier Network',
    category: 'B2B SOURCING MESH',
    href: '/roles/supplier',
  },
  {
    id: 'telemetry-analytics',
    image: SITE_IMAGES.operationsTeam,
    title: 'Real-Time Telemetry Analytics',
    category: 'TACTICAL CONTROL TOWER',
    href: '/operations',
  },
  {
    id: 'fulfillment-wave',
    image: SITE_IMAGES.warehouseWireframe,
    title: 'Fulfillment Wave Control',
    category: 'MEIO & LOAD BALANCING',
    href: '/platform',
  },
  {
    id: 'treasury-clearing',
    image: SITE_IMAGES.deliveryDrone,
    title: 'Payment & Treasury Clearing',
    category: 'DOUBLE-ENTRY LEDGER',
    href: '/capabilities/payment-confidence',
  },
  {
    id: 'intermodal-lanes',
    image: SITE_IMAGES.containerShip,
    title: 'Intermodal Lane Capacity',
    category: 'GLOBAL MIDDLE-MILE',
    href: '/projects',
  },
  {
    id: 'zero-trust-architecture',
    image: SITE_IMAGES.terminalArchitecture,
    title: 'Spanner Zero-Trust Architecture',
    category: 'ACID DISTRIBUTED DATA',
    href: '/technology',
  },
  {
    id: 'factory-gate',
    image: SITE_IMAGES.portCraneScene,
    title: 'Factory Weighbridge & Gate',
    category: 'SCALE & RFID ACCESS',
    href: '/roles/payload-gate',
  },
  {
    id: 'retailer-portal',
    image: SITE_IMAGES.lastMileDelivery,
    title: 'Retailer Procurement Portal',
    category: 'ORDER-TO-CASH EPOD',
    href: '/demo/retailer',
  },
  {
    id: 'network-mesh',
    image: SITE_IMAGES.fleekHeroNew,
    title: 'Ecosystem Network Mesh',
    category: 'MULTI-ROLE CONVERGENCE',
    href: '/platform',
  },
  {
    id: 'cvrp-optimization',
    image: '/Unknown-10.jpg',
    title: 'Google OR-Tools CVRP Dispatch',
    category: 'ROUTE OPTIMIZATION',
    href: '/solutions/fleet-visibility',
  },
];

export default function ShowcaseWall() {
  const prefersReduced = useReducedMotion();
  const sectionRef = useRef<HTMLDivElement>(null);
  const pinRef = useRef<HTMLDivElement>(null);
  const cardElementsRef = useRef<(HTMLDivElement | null)[]>([]);
  const activeIndexRef = useRef(0);
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    if (!sectionRef.current || !pinRef.current) return;

    const N = ORBIT_ITEMS.length;
    const alpha = (22 * Math.PI) / 180; // Tilt angle (around X axis)
    const beta = (-6 * Math.PI) / 180; // Roll angle (around Z axis)
    const D = 1000; // Perspective distance

    // Continuous idle drift rotation angle + scroll-driven rotation angle
    const state = {
      scrollRotation: 0,
      idleRotation: 0,
    };

    let rafId: number;

    const updateCards = () => {
      if (!pinRef.current) return;

      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      // Responsive radii matching viewport
      const Rx = Math.min(viewportWidth * 0.38, 540);
      const Ry = Math.min(viewportHeight * 0.20, 165);
      const Rz = 240;

      const totalAngle = state.scrollRotation + state.idleRotation;

      let highestZ = -Infinity;
      let frontIdx = 0;

      for (let i = 0; i < N; i++) {
        const cardEl = cardElementsRef.current[i];
        if (!cardEl) continue;

        const theta = (i / N) * 2 * Math.PI + totalAngle;

        const x = Rx * Math.cos(theta);
        const y = Ry * Math.sin(theta);
        const z = Rz * Math.sin(theta);

        // 3D rotations: tilt around X then roll around Z
        const y_prime = y * Math.cos(alpha) - z * Math.sin(alpha);
        const z_prime = y * Math.sin(alpha) + z * Math.cos(alpha);
        const x_double = x * Math.cos(beta) - y_prime * Math.sin(beta);
        const y_double = x * Math.sin(beta) + y_prime * Math.cos(beta);

        // Perspective projection
        const factor = D / (D - z_prime);
        const sx = x_double * factor;
        const sy = y_double * factor;
        const scale = factor;
        const opacity = Math.min(Math.max(0.35 + 0.65 * ((z_prime + Rz) / (2 * Rz)), 0.25), 1.0);
        const zIndex = Math.round(z_prime + 500);

        // Track front-most card for active index
        if (z_prime > highestZ) {
          highestZ = z_prime;
          frontIdx = i;
        }

        // Direct DOM write for 60fps GPU acceleration
        cardEl.style.transform = `translate3d(calc(-50% + ${sx}px), calc(-50% + ${sy}px), 0px) scale(${scale.toFixed(3)})`;
        cardEl.style.opacity = opacity.toFixed(2);
        cardEl.style.zIndex = `${zIndex}`;
      }

      // Update state only when active index shifts
      if (frontIdx !== activeIndexRef.current) {
        activeIndexRef.current = frontIdx;
        setActiveIndex(frontIdx);
      }
    };

    // Smooth idle drift animation
    const animateIdle = () => {
      state.idleRotation += prefersReduced ? 0 : 0.0015;
      updateCards();
      rafId = requestAnimationFrame(animateIdle);
    };

    rafId = requestAnimationFrame(animateIdle);

    // GSAP ScrollTrigger context
    const ctx = gsap.context(() => {
      gsap.to(state, {
        scrollRotation: Math.PI * 2.8, // 1.4 full rotations across scroll
        ease: 'none',
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top top',
          end: '+=220%',
          pin: pinRef.current,
          scrub: 1.2,
          anticipatePin: 1,
          onUpdate: () => {
            updateCards();
          },
        },
      });
    }, sectionRef);

    const handleResize = () => updateCards();
    window.addEventListener('resize', handleResize);

    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', handleResize);
      ctx.revert();
    };
  }, [prefersReduced]);

  const activeItem = ORBIT_ITEMS[activeIndex] || ORBIT_ITEMS[0];

  return (
    <div ref={sectionRef} className="relative w-full bg-black text-white select-none overflow-visible">
      
      {/* Pinned 100vh Fullscreen Viewport */}
      <div
        ref={pinRef}
        className="h-screen w-full sticky top-0 flex flex-col justify-between overflow-hidden bg-black relative"
      >
        {/* Subtle dot matrix grid terrain background matching reference */}
        <div
          className="absolute inset-0 pointer-events-none opacity-20"
          style={{
            backgroundImage: 'radial-gradient(circle, rgba(255, 255, 255, 0.4) 1px, transparent 1px)',
            backgroundSize: '24px 24px',
          }}
        />

        {/* Ambient radial spotlight in the center */}
        <div className="absolute inset-0 pointer-events-none bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.04)_0%,transparent_70%)]" />

        {/* ── Center 3D Orbital Arena ── */}
        <div className="relative w-full flex-1 flex items-center justify-center overflow-visible">

          {/* 3D Orbiting Cards Loop */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            {ORBIT_ITEMS.map((item, idx) => {
              const isFront = idx === activeIndex;

              return (
                <div
                  key={item.id}
                  ref={(el) => {
                    cardElementsRef.current[idx] = el;
                  }}
                  className="absolute top-1/2 left-1/2 will-change-transform pointer-events-auto transition-shadow duration-300"
                  style={{
                    transform: 'translate3d(-50%, -50%, 0px)',
                  }}
                >
                  <Link
                    href={item.href}
                    className={`block relative group overflow-hidden bg-[#0c0c0e] transition-all duration-300 ${
                      isFront
                        ? 'shadow-[0_0_35px_rgba(255,255,255,0.2)]'
                        : ''
                    } w-32 h-22 sm:w-44 sm:h-30 md:w-52 md:h-36 lg:w-60 lg:h-40`}
                  >
                    {/* Image */}
                    <Image
                      src={item.image}
                      alt={item.title}
                      fill
                      sizes="(max-width: 640px) 130px, (max-width: 1024px) 210px, 240px"
                      className="object-cover grayscale contrast-125 brightness-95 group-hover:scale-105 transition-transform duration-500"
                    />
                  </Link>
                </div>
              );
            })}
          </div>

        </div>

        {/* ── Bottom Telemetry Bar (Reference match: "001  EVIDENS DE BEAUTÉ  DISCOVER MORE") ── */}
        <footer className="px-6 sm:px-12 py-6 flex items-center justify-between z-30 font-mono text-xs tracking-wider border-t border-zinc-900 bg-black/70 backdrop-blur-md">
          {/* Left: 3-digit index */}
          <div className="text-white font-bold text-sm">
            {String(activeIndex + 1).padStart(3, '0')}
          </div>

          {/* Center: Active Title */}
          <div className="text-center px-4 truncate max-w-md">
            <span className="text-zinc-500 uppercase tracking-widest text-[11px] hidden sm:inline mr-2">
              NODE //
            </span>
            <span className="text-zinc-100 font-semibold tracking-wider text-xs sm:text-sm uppercase">
              {activeItem.title}
            </span>
          </div>

          {/* Right: Discover More Link */}
          <div className="text-right">
            <Link
              href={activeItem.href}
              className="inline-flex items-center gap-1.5 text-zinc-400 hover:text-white transition-colors group text-xs uppercase"
            >
              <span>DISCOVER MORE</span>
              <span className="group-hover:translate-x-1 transition-transform font-bold">→</span>
            </Link>
          </div>
        </footer>

      </div>

    </div>
  );
}
