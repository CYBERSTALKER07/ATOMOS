'use client';

import React, { useEffect, useRef } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { gsap, ScrollTrigger } from '@/app/lib/gsap';
import { SITE_IMAGES } from '@/app/lib/siteAssets';
import { usePerfProfile } from '@/app/hooks/useDevice';

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
 const { isLowEnd, prefersReducedMotion } = usePerfProfile();
 const prefersReduced = prefersReducedMotion || isLowEnd;
 const sectionRef = useRef<HTMLDivElement>(null);
 const pinRef = useRef<HTMLDivElement>(null);
 const cardElementsRef = useRef<(HTMLDivElement | null)[]>([]);
 const activeIndexRef = useRef(9); // Default to featured card 010 (Spanner Zero-Trust Architecture)

 useEffect(() => {
 if (!sectionRef.current || !pinRef.current) return;

 const N = ORBIT_ITEMS.length;
 const alpha = (33 * Math.PI) / 180; // Tilt angle (around X axis) - elevates back cards nicely above
 const beta = (-4 * Math.PI) / 180; // Subtle roll angle (around Z axis) - matching reference slant
 const D = 1100; // Perspective distance

 // Continuous idle drift rotation angle + scroll-driven rotation angle
 const state = {
 scrollRotation: 0,
 idleRotation: 0,
 };

 let rafId: number | null = null;

 const updateCards = () => {
 if (!pinRef.current) return;

 const viewportWidth = window.innerWidth;
 const viewportHeight = window.innerHeight;

 // Responsive radii: balanced 3D ground plane orbit
 const Rx = Math.min(viewportWidth * 0.36, 520);
 const Rz = Math.min(viewportHeight * 0.33, Rx * 0.72, 340);

 const totalAngle = state.scrollRotation + state.idleRotation;

 let highestZ = -Infinity;
 let frontIdx = 0;

 for (let i = 0; i < N; i++) {
 const el = cardElementsRef.current[i];
 if (!el) continue;

 // Position on elliptic perimeter
 const theta = (i / N) * 2 * Math.PI + totalAngle;
 const x = Rx * Math.sin(theta);
 const z = Rz * Math.cos(theta);

 // Ground-plane 3D projection
 const yRot = x * Math.sin(beta);
 const zRot = x * Math.cos(beta);
 const yProj = yRot * Math.cos(alpha) - z * Math.sin(alpha);
 const zProj = yRot * Math.sin(alpha) + z * Math.cos(alpha);

 // Track closest card to viewer
 if (zProj > highestZ) {
 highestZ = zProj;
 frontIdx = i;
 }

 // Perspective scale & depth
 const scale = D / (D + zProj * 0.85);
 const depthNorm = (zProj + Rz) / (2 * Rz); // 0 (back) to 1 (front)

 // Visual depth cues
 const opacity = 0.28 + depthNorm * 0.72;
 const blurAmount = Math.max(0, (1 - depthNorm) * 3);
 const zIndex = Math.round(depthNorm * 100);

 // Smooth GPU transform
 el.style.transform = `translate3d(calc(-50% + ${x}px), calc(-50% + ${yProj}px), 0px) scale(${scale})`;
 el.style.opacity = `${opacity}`;
 el.style.filter = blurAmount > 0.5 ? `blur(${blurAmount}px)` : 'none';
 el.style.zIndex = `${zIndex}`;
 }

 activeIndexRef.current = frontIdx;
 };

 // Initial positioning
 updateCards();

 // Smooth idle drift animation (skipped on reduced motion & low-end devices to conserve CPU)
 if (!prefersReduced) {
 const animateIdle = () => {
 state.idleRotation += 0.0015;
 updateCards();
 rafId = requestAnimationFrame(animateIdle);
 };
 rafId = requestAnimationFrame(animateIdle);
 }

 // GSAP ScrollTrigger context (reduced scroll distance for effortless browsing)
 const ctx = gsap.context(() => {
 gsap.to(state, {
 scrollRotation: Math.PI * 1.5, // Responsive orbital rotation
 ease: 'none',
 scrollTrigger: {
 trigger: sectionRef.current,
 start: 'top top',
 end: '+=90%', // Significantly less scroll distance (from 220% down to 90%)
 pin: pinRef.current,
 scrub: 0.7,
 anticipatePin: 1,
 fastScrollEnd: true,
 onUpdate: () => {
 updateCards();
 },
 },
 });
 }, sectionRef);

 const handleResize = () => updateCards();
 window.addEventListener('resize', handleResize);

 return () => {
 if (rafId !== null) cancelAnimationFrame(rafId);
 window.removeEventListener('resize', handleResize);
 ctx.revert();
 };
 }, [prefersReduced]);

 return (
 <div ref={sectionRef} className="relative w-full bg-black text-white select-none overflow-visible">
 
 {/* Pinned 100vh Fullscreen Viewport */}
 <div
 ref={pinRef}
 className="h-screen w-full sticky top-0 flex flex-col justify-between overflow-hidden bg-black relative"
 >
 {/* Subtle dot matrix grid terrain background matching reference */}
 <div
 className="absolute inset-0 pointer-events-none opacity-[0.06]"
 style={{
 backgroundImage: 'radial-gradient(circle, rgba(255, 255, 255, 0.4) 1px, transparent 1px)',
 backgroundSize: '32px 32px',
 }}
 />

 {/* ── Center 3D Orbital Arena ── */}
 <div className="relative w-full flex-1 flex items-center justify-center overflow-visible bg-black">

 {/* 3D Orbiting Cards Loop */}
 <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
 {ORBIT_ITEMS.map((item, idx) => (
 <div
 key={item.id}
 ref={(el) => {
 cardElementsRef.current[idx] = el;
 }}
 className="absolute top-1/2 left-1/2 will-change-transform pointer-events-auto"
 style={{
 transform: 'translate3d(-50%, -50%, 0px)',
 }}
 >
 <Link
 href={item.href}
 className="block relative group overflow-hidden bg-black w-32 h-22 sm:w-44 sm:h-30 md:w-52 md:h-36 lg:w-60 lg:h-40"
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
 ))}
 </div>

 </div>

 {/* ── Bottom Telemetry Bar (Reference match: "010 NODE // SPANNER ZERO-TRUST ARCHITECTURE DISCOVER MORE") ── */}
 <footer className="px-6 sm:px-12 py-6 flex items-center justify-between z-30 font-mono text-xs tracking-wider bg-black border-t-0">
 {/* Left: 3-digit index */}
 <div className="text-white font-bold text-sm">
 010
 </div>

 {/* Center: Node Title */}
 <div className="text-center px-4 truncate max-w-md">
 <span className="text-zinc-500 uppercase tracking-widest text-[11px] hidden sm:inline mr-2">
 NODE //
 </span>
 <span className="text-zinc-100 font-semibold tracking-wider text-xs sm:text-sm uppercase">
 SPANNER ZERO-TRUST ARCHITECTURE
 </span>
 </div>

 {/* Right: Discover More Link */}
 <div className="text-right">
 <Link
 href="/technology"
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
