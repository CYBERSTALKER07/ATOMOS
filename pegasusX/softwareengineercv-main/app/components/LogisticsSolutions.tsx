'use client';

import { useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { AnimatePresence, motion, useReducedMotion as useFramerReducedMotion } from 'framer-motion';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import { cn } from '@/lib/utils';

gsap.registerPlugin(ScrollTrigger);

type Solution = {
  id: string;
  label: string;
  title: string;
  description: string;
  href: string;
  /** Drop files under /public/solutions/ and set paths here later. */
  image?: string;
  imageAlt?: string;
};

const SOLUTIONS: Solution[] = [
  {
    id: 'network',
    label: 'Network Control',
    title: 'Supplier Control Plane',
    description:
      'Connect vetting, topology, treasury, and dispatch preview on one live platform — so suppliers see the full network without switching tools.',
    href: '/projects/supplier-control-plane',
    // image: '/solutions/network.jpg',
  },
  {
    id: 'dispatch',
    label: 'Dispatch Operations',
    title: 'Visual Dispatch Engine',
    description:
      'Match trucks to orders at peak hours with live warehouse boards, gate seals, and instant updates when plans change on the floor.',
    href: '/solutions/visual-dispatch-engine',
    // image: '/solutions/dispatch.jpg',
  },
  {
    id: 'fleet',
    label: 'Fleet Visibility',
    title: 'Fleet Telemetry',
    description:
      'Planned-vs-actual routes, deviation alerts, and retailer self-serve tracking — one honest picture of where every truck is.',
    href: '/solutions/fleet-visibility',
    // image: '/solutions/fleet.jpg',
  },
  {
    id: 'payments',
    label: 'Payment Integrity',
    title: 'Checkout to Treasury',
    description:
      'From retailer checkout through driver cash collection to supplier reconciliation — duplicate protection and a clear audit trail.',
    href: '/projects/payment-integrity',
    // image: '/solutions/payments.jpg',
  },
  {
    id: 'commerce',
    label: 'Retailer Commerce',
    title: 'Ordering & Tracking',
    description:
      'Catalog, checkout, scheduling, and live order tracking with desktop and mobile parity for retailer teams.',
    href: '/projects/retailer-commerce',
    // image: '/solutions/commerce.jpg',
  },
  {
    id: 'gate',
    label: 'Gate & Seal Control',
    title: 'Payload Accountability',
    description:
      'Seal, scan, and release at the gate — terminal and mobile parity so nothing leaves without a verified manifest.',
    href: '/projects/payload-gate-control',
    // image: '/solutions/gate.jpg',
  },
  {
    id: 'partner',
    label: 'Partner Collaboration',
    title: 'Six-Role Coordination',
    description:
      'Supplier, warehouse, factory, driver, retailer, and gate teams on shared contracts — every role sees the same live state.',
    href: '/roles',
    // image: '/solutions/partner.jpg',
  },
];

const contentVariants = {
  initial: { opacity: 0, y: 14, filter: 'blur(4px)' },
  animate: { opacity: 1, y: 0, filter: 'blur(0px)' },
  exit: { opacity: 0, y: -10, filter: 'blur(4px)' },
};

function SolutionImagePanel({
  title,
  image,
  imageAlt,
}: {
  title: string;
  image?: string;
  imageAlt?: string;
}) {
  return (
    <div className="relative flex h-full min-h-[20rem] flex-col border-l border-white/10 bg-[#141414] overflow-hidden">
      {image ? (
        <Image
          src={image}
          alt={imageAlt ?? title}
          fill
          className="object-cover"
          sizes="(max-width: 768px) 100vw, 40vw"
          priority={false}
        />
      ) : (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-gradient-to-br from-[#1a1a1a] to-[#0a0a0a] p-8 text-center">
          <div className="absolute inset-0 opacity-40 bg-[linear-gradient(to_right,#333_1px,transparent_1px),linear-gradient(to_bottom,#333_1px,transparent_1px)] bg-[size:32px_32px]" />
          <p className="relative z-10 font-mono text-[0.65rem] uppercase tracking-[0.2em] text-white/40">
            Image placeholder
          </p>
          <p className="relative z-10 max-w-[14rem] text-sm text-white/55">{title}</p>
          <p className="relative z-10 font-mono text-[0.6rem] text-white/30">
            Add file → set image path in SOLUTIONS
          </p>
        </div>
      )}
    </div>
  );
}

export default function LogisticsSolutions() {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  const framerReduced = useFramerReducedMotion();
  const reduceMotion = prefersReducedMotion || !!framerReduced;

  const sectionRef = useRef<HTMLElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const navItemRefs = useRef<(HTMLLIElement | null)[]>([]);
  const [activeId, setActiveId] = useState(SOLUTIONS[0].id);
  const [hoveredNav, setHoveredNav] = useState<string | null>(null);

  const active = SOLUTIONS.find((item) => item.id === activeId) ?? SOLUTIONS[0];
  const activeIndex = SOLUTIONS.findIndex((s) => s.id === activeId);

  useEffect(() => {
    if (!sectionRef.current) return;

    if (isMobile || reduceMotion) {
      gsap.set([titleRef.current, panelRef.current, ...navItemRefs.current], {
        opacity: 1,
        y: 0,
        clearProps: 'all',
      });
      return;
    }

    const ctx = gsap.context(() => {
      const timeline = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top 78%',
          toggleActions: 'play none none reverse',
        },
      });

      timeline
        .fromTo(
          titleRef.current,
          { opacity: 0, y: 28 },
          { opacity: 1, y: 0, duration: 0.75, ease: 'power3.out' }
        )
        .fromTo(
          panelRef.current,
          { opacity: 0, y: 36 },
          { opacity: 1, y: 0, duration: 0.85, ease: 'power3.out' },
          '-=0.45'
        )
        .fromTo(
          navItemRefs.current.filter(Boolean),
          { opacity: 0, x: -16 },
          {
            opacity: 1,
            x: 0,
            duration: 0.45,
            stagger: 0.06,
            ease: 'power2.out',
          },
          '-=0.55'
        );
    }, sectionRef);

    return () => ctx.revert();
  }, [isMobile, reduceMotion]);

  return (
    <PageSection ref={sectionRef} id="solutions">
      <div ref={titleRef}>
        <SectionHeader
          eyebrow="The Pegasus Network"
          title="End-to-End Logistics Solutions"
        />
      </div>

      <div
        ref={panelRef}
        className="grid grid-cols-1 lg:grid-cols-[minmax(0,17rem)_1fr] gap-0 border border-white/15 overflow-hidden"
      >
        <nav
          className="relative border-b lg:border-b-0 lg:border-r border-white/15 bg-black/40"
          aria-label="Logistics solutions"
        >
          <ul className="relative">
            {SOLUTIONS.map((solution, index) => {
              const isActive = solution.id === activeId;
              const isHovered = hoveredNav === solution.id;
              return (
                <li
                  key={solution.id}
                  ref={(el) => {
                    navItemRefs.current[index] = el;
                  }}
                  className="border-b border-white/10 last:border-b-0 relative"
                  onMouseEnter={() => setHoveredNav(solution.id)}
                  onMouseLeave={() => setHoveredNav(null)}
                >
                  {isActive && (
                    <motion.span
                      layoutId={reduceMotion ? undefined : 'solutions-active-bar'}
                      className="absolute inset-0 bg-white z-0"
                      transition={{ type: 'spring', stiffness: 380, damping: 34 }}
                    />
                  )}
                  {!isActive && isHovered && !reduceMotion && (
                    <motion.span
                      layoutId="solutions-hover-bar"
                      className="absolute inset-0 bg-white/[0.06] z-0"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                    />
                  )}
                  <button
                    type="button"
                    onClick={() => setActiveId(solution.id)}
                    aria-current={isActive ? 'true' : undefined}
                    className={cn(
                      'relative z-10 w-full text-left px-5 py-4 md:py-5 text-sm md:text-base font-medium flex items-center gap-3 transition-colors duration-200',
                      isActive ? 'text-black' : 'text-white/85 hover:text-white'
                    )}
                  >
                    <span
                      className={cn(
                        'w-2 h-2 shrink-0 transition-all duration-300',
                        isActive
                          ? 'bg-black scale-110'
                          : isHovered
                            ? 'bg-white/70 scale-105'
                            : 'bg-transparent ring-1 ring-white/25'
                      )}
                      aria-hidden="true"
                    />
                    <span className="flex-1">{solution.label}</span>
                    <span
                      className={cn(
                        'font-mono text-[0.65rem] tracking-wider transition-opacity duration-200',
                        isActive
                          ? 'opacity-70 text-black'
                          : isHovered
                            ? 'opacity-70 text-white'
                            : 'opacity-40 text-white'
                      )}
                    >
                      {String(index + 1).padStart(2, '0')}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
          <Link
            href="/projects"
            className="editorial-link inline-block m-5 text-white hover:text-[#FBFF63] hover:translate-x-1 transition-all duration-200"
          >
            ALL MODULES →
          </Link>
        </nav>

        <div className="grid grid-cols-1 md:grid-cols-2 min-h-[22rem]">
          <div className="relative p-8 md:p-10 lg:p-12 flex flex-col justify-between border-b md:border-b-0 md:border-r border-white/10 overflow-hidden">
            <AnimatePresence mode="wait" initial={false}>
              <motion.div
                key={active.id}
                variants={reduceMotion ? undefined : contentVariants}
                initial={reduceMotion ? false : 'initial'}
                animate="animate"
                exit={reduceMotion ? undefined : 'exit'}
                transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
                className="flex flex-col justify-between h-full gap-8"
              >
                <div>
                  <p className="font-mono text-[0.65rem] uppercase tracking-[0.2em] text-white/40 mb-3">
                    Module {String(activeIndex + 1).padStart(2, '0')} /{' '}
                    {String(SOLUTIONS.length).padStart(2, '0')}
                  </p>
                  <h3 className="text-2xl md:text-3xl font-light mb-5">{active.title}</h3>
                  <p className="text-white/70 leading-relaxed max-w-md">{active.description}</p>
                </div>
                <Link
                  href={active.href}
                  className="editorial-btn mt-auto w-fit hover:gap-3 transition-all duration-200"
                >
                  LEARN MORE
                </Link>
              </motion.div>
            </AnimatePresence>
          </div>

          <AnimatePresence mode="wait" initial={false}>
            <motion.div
              key={active.id}
              initial={reduceMotion ? false : { opacity: 0, x: 18 }}
              animate={{ opacity: 1, x: 0 }}
              exit={reduceMotion ? undefined : { opacity: 0, x: -12 }}
              transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
              className="min-h-[20rem] relative"
            >
              <SolutionImagePanel
                title={active.title}
                image={active.image}
                imageAlt={active.imageAlt}
              />
            </motion.div>
          </AnimatePresence>
        </div>
      </div>
    </PageSection>
  );
}
