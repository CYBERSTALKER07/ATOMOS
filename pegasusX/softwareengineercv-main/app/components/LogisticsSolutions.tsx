'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useIsMobile, useReducedMotion } from '../hooks/useDevice';
import FleetVisualPanel from './fleet/FleetVisualPanel';

gsap.registerPlugin(ScrollTrigger);

type Solution = {
  id: string;
  label: string;
  title: string;
  description: string;
  href: string;
  chartLabel: string;
  bars: number[];
  line: number[];
};

const QUARTERS = ['Q1', 'Q2', 'Q3', 'Q4'];

const SOLUTIONS: Solution[] = [
  {
    id: 'network',
    label: 'Network Control',
    title: 'Supplier Control Plane',
    description:
      'Connect vetting, topology, treasury, and dispatch preview on one live platform — so suppliers see the full network without switching tools.',
    href: '/projects/supplier-control-plane',
    chartLabel: 'ORDER VOLUME',
    bars: [42, 58, 71, 88],
    line: [38, 52, 68, 82],
  },
  {
    id: 'dispatch',
    label: 'Dispatch Operations',
    title: 'Visual Dispatch Engine',
    description:
      'Match trucks to orders at peak hours with live warehouse boards, gate seals, and instant updates when plans change on the floor.',
    href: '/solutions/visual-dispatch-engine',
    chartLabel: 'DISPATCH LOAD',
    bars: [55, 72, 64, 91],
    line: [48, 65, 70, 85],
  },
  {
    id: 'fleet',
    label: 'Fleet Visibility',
    title: 'Fleet Telemetry',
    description:
      'Planned-vs-actual routes, deviation alerts, and retailer self-serve tracking — one honest picture of where every truck is.',
    href: '/solutions/fleet-visibility',
    chartLabel: 'ON-TIME RATE',
    bars: [68, 74, 79, 86],
    line: [62, 70, 76, 84],
  },
  {
    id: 'payments',
    label: 'Payment Integrity',
    title: 'Checkout to Treasury',
    description:
      'From retailer checkout through driver cash collection to supplier reconciliation — duplicate protection and a clear audit trail.',
    href: '/projects/payment-integrity',
    chartLabel: 'COLLECTION RATE',
    bars: [61, 70, 78, 84],
    line: [58, 66, 74, 80],
  },
  {
    id: 'commerce',
    label: 'Retailer Commerce',
    title: 'Ordering & Tracking',
    description:
      'Catalog, checkout, scheduling, and live order tracking with desktop and mobile parity for retailer teams.',
    href: '/projects/retailer-commerce',
    chartLabel: 'ORDER GROWTH',
    bars: [35, 48, 62, 77],
    line: [32, 44, 58, 72],
  },
  {
    id: 'gate',
    label: 'Gate & Seal Control',
    title: 'Payload Accountability',
    description:
      'Seal, scan, and release at the gate — terminal and mobile parity so nothing leaves without a verified manifest.',
    href: '/projects/payload-gate-control',
    chartLabel: 'SEAL COMPLIANCE',
    bars: [88, 90, 93, 96],
    line: [85, 88, 91, 94],
  },
  {
    id: 'partner',
    label: 'Partner Collaboration',
    title: 'Six-Role Coordination',
    description:
      'Supplier, warehouse, factory, driver, retailer, and gate teams on shared contracts — every role sees the same live state.',
    href: '/roles/role-parity-matrix',
    chartLabel: 'ROLE PARITY',
    bars: [72, 78, 82, 90],
    line: [68, 74, 80, 88],
  },
];

function DispatchChart({ solution }: { solution: Solution }) {
  const prefersReducedMotion = useReducedMotion();
  const chartRef = useRef<HTMLDivElement>(null);
  const barRefs = useRef<(HTMLDivElement | null)[]>([]);
  const lineRef = useRef<SVGPolylineElement>(null);
  const labelRef = useRef<HTMLParagraphElement>(null);
  const max = Math.max(...solution.bars, ...solution.line);

  const linePoints = solution.line
    .map((value, index) => {
      const x = 12.5 + index * 25;
      const y = 100 - (value / max) * 78;
      return `${x},${y}`;
    })
    .join(' ');

  useEffect(() => {
    const bars = barRefs.current.filter(Boolean) as HTMLDivElement[];
    const line = lineRef.current;
    const label = labelRef.current;

    if (prefersReducedMotion) {
      gsap.set(bars, { scaleY: 1 });
      if (line) gsap.set(line, { strokeDashoffset: 0 });
      if (label) gsap.set(label, { opacity: 1, y: 0 });
      return;
    }

    gsap.set(bars, { scaleY: 0, transformOrigin: 'bottom center' });
    if (line) {
      const length = line.getTotalLength();
      gsap.set(line, { strokeDasharray: length, strokeDashoffset: length, opacity: 0.9 });
    }
    if (label) gsap.set(label, { opacity: 0, y: 6 });

    const timeline = gsap.timeline();
    timeline
      .to(label, { opacity: 1, y: 0, duration: 0.35, ease: 'power2.out' })
      .to(
        bars,
        { scaleY: 1, duration: 0.75, stagger: 0.12, ease: 'power3.out' },
        '-=0.1'
      )
      .to(line, { strokeDashoffset: 0, duration: 1, ease: 'power2.inOut' }, '-=0.55');

    return () => {
      timeline.kill();
    };
  }, [solution.id, prefersReducedMotion]);

  return (
    <div
      ref={chartRef}
      className="h-full flex flex-col justify-end p-6 md:p-8 bg-[#141414] border-l border-white/10 min-h-[18rem]"
    >
      <p
        ref={labelRef}
        className="text-[0.65rem] font-mono tracking-[0.18em] text-white/50 mb-4 uppercase"
      >
        {solution.chartLabel}
      </p>

      <div className="relative h-44 md:h-52 mb-3">
        {/* Grid lines */}
        <div className="absolute inset-0 flex flex-col justify-between pointer-events-none" aria-hidden="true">
          {[0, 1, 2, 3].map((line) => (
            <div key={line} className="w-full border-t border-white/[0.07]" />
          ))}
        </div>

        {/* Bars */}
        <div className="absolute inset-0 flex items-end justify-between gap-3 md:gap-5 px-1">
          {solution.bars.map((value, index) => {
            const bodyHeight = (value / max) * 72;
            const capHeight = (value / max) * 18;

            return (
              <div key={`${solution.id}-bar-${index}`} className="flex-1 flex flex-col items-center h-full justify-end">
                <span className="text-[0.6rem] font-mono text-white/40 mb-2 tabular-nums">{value}</span>
                <div
                  ref={(el) => {
                    barRefs.current[index] = el;
                  }}
                  className="w-full max-w-[3.25rem] flex flex-col justify-end h-[calc(100%-1.25rem)]"
                  style={{ transformOrigin: 'bottom center' }}
                >
                  <div
                    className="w-full bg-white/18 transition-colors duration-300"
                    style={{ height: `${bodyHeight}%` }}
                  />
                  <div
                    className="w-full bg-[#FFA500] shadow-[0_0_18px_rgba(255,165,0,0.35)]"
                    style={{ height: `${capHeight}%` }}
                  />
                </div>
                <span className="mt-3 text-[0.6rem] font-mono tracking-wider text-white/35 uppercase">
                  {QUARTERS[index]}
                </span>
              </div>
            );
          })}
        </div>

        {/* Trend line */}
        <svg
          className="absolute inset-0 w-full h-[calc(100%-1.75rem)] pointer-events-none"
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <polyline
            ref={lineRef}
            fill="none"
            stroke="white"
            strokeWidth="1.75"
            strokeLinecap="round"
            strokeLinejoin="round"
            vectorEffect="non-scaling-stroke"
            points={linePoints}
          />
        </svg>
      </div>

      <div className="flex flex-wrap gap-4 pt-2 text-[0.6rem] font-mono tracking-wider text-white/45 uppercase border-t border-white/10">
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 bg-[#FFA500]" /> Forecast
        </span>
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 bg-white/25" /> Actual
        </span>
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 border border-white/40" /> Trend
        </span>
      </div>
    </div>
  );
}

export default function LogisticsSolutions() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [activeId, setActiveId] = useState(SOLUTIONS[0].id);

  const active = SOLUTIONS.find((item) => item.id === activeId) ?? SOLUTIONS[0];

  useEffect(() => {
    if (!sectionRef.current) return;

    if (isMobile) {
      gsap.set([titleRef.current, panelRef.current], { opacity: 1, y: 0 });
      return;
    }

    const timeline = gsap.timeline({
      scrollTrigger: {
        trigger: sectionRef.current,
        start: 'top 80%',
        toggleActions: 'play none none reverse',
      },
    });

    timeline
      .fromTo(titleRef.current, { opacity: 0, y: 24 }, { opacity: 1, y: 0, duration: 0.8 })
      .fromTo(panelRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 }, '-=0.4');
  }, [isMobile]);

  return (
    <section ref={sectionRef} id="solutions" className="py-20 md:py-28 bg-black text-white">
      <div className="container mx-auto px-4 max-w-7xl">
        <div ref={titleRef} className="mb-12 md:mb-16">
          <p className="editorial-eyebrow text-white/50 mb-4">The Pegasus Network</p>
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-light tracking-tight max-w-4xl">
            End-to-End Logistics Solutions
          </h2>
        </div>

        <div ref={panelRef} className="grid grid-cols-1 lg:grid-cols-[minmax(0,17rem)_1fr] gap-0 border border-white/15">
          <nav className="border-b lg:border-b-0 lg:border-r border-white/15" aria-label="Logistics solutions">
            <ul>
              {SOLUTIONS.map((solution) => {
                const isActive = solution.id === activeId;
                return (
                  <li key={solution.id} className="border-b border-white/10 last:border-b-0">
                    <button
                      type="button"
                      onClick={() => setActiveId(solution.id)}
                      className={`w-full text-left px-5 py-4 md:py-5 text-sm md:text-base font-medium transition-colors duration-200 flex items-center gap-3 ${
                        isActive ? 'bg-white text-black' : 'text-white hover:bg-white/5'
                      }`}
                    >
                      {isActive ? (
                        <span className="w-2 h-2 bg-black shrink-0" aria-hidden="true" />
                      ) : (
                        <span className="w-2 h-2 shrink-0" aria-hidden="true" />
                      )}
                      {solution.label}
                    </button>
                  </li>
                );
              })}
            </ul>
            <Link href="/projects" className="editorial-link inline-block m-5 text-white">
              ALL MODULES →
            </Link>
          </nav>

          <div className="grid grid-cols-1 md:grid-cols-2 min-h-[22rem]">
            <div className="p-8 md:p-10 lg:p-12 flex flex-col justify-between border-b md:border-b-0 md:border-r border-white/10">
              <div>
                <h3 className="text-2xl md:text-3xl font-light mb-5">{active.title}</h3>
                <p className="text-white/70 leading-relaxed max-w-md">{active.description}</p>
              </div>
              <Link href={active.href} className="editorial-btn mt-8 w-fit">
                LEARN MORE
              </Link>
            </div>
            {active.id === 'fleet' || active.id === 'dispatch' ? (
              <FleetVisualPanel key={active.id} mode={active.id === 'fleet' ? 'fleet' : 'dispatch'} />
            ) : (
              <DispatchChart key={active.id} solution={active} />
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
