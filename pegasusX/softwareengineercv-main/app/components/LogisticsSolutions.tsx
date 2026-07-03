'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useIsMobile } from '../hooks/useDevice';
import FleetVisualPanel from './fleet/FleetVisualPanel';
import LogisticsSolutionChart from './logistics/LogisticsSolutionChart';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';

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
    href: '/roles',
    chartLabel: 'ROLE PARITY',
    bars: [72, 78, 82, 90],
    line: [68, 74, 80, 88],
  },
];

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
    <PageSection ref={sectionRef} id="solutions">
        <div ref={titleRef}>
          <SectionHeader
            eyebrow="The Pegasus Network"
            title="End-to-End Logistics Solutions"
          />
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
              <LogisticsSolutionChart
                key={active.id}
                chartLabel={active.chartLabel}
                bars={active.bars}
                line={active.line}
              />
            )}
          </div>
        </div>
    </PageSection>
  );
}
