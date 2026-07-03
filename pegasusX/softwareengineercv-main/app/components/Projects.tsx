'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Link from 'next/link';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import { bentoPlacement } from '../lib/bento';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';

gsap.registerPlugin(ScrollTrigger);

export default function Projects() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sectionRef.current) return;

    if (isMobile) {
      gsap.set([titleRef.current, gridRef.current], { opacity: 1, y: 0 });
      return;
    }

    const timeline = gsap.timeline({
      scrollTrigger: {
        trigger: sectionRef.current,
        start: 'top 80%',
        end: 'bottom 20%',
        toggleActions: 'play none none reverse',
      },
    });

    timeline
      .fromTo(titleRef.current, { opacity: 0, y: 30 }, { opacity: 1, y: 0, duration: 0.8 })
      .fromTo(
        gridRef.current?.children ? Array.from(gridRef.current.children) : [],
        { opacity: 0, y: 40 },
        { opacity: 1, y: 0, duration: 0.55, stagger: 0.08 },
        '-=0.35'
      );
  }, [isMobile]);

  const projects = [
    {
      title: 'Dispatch Engine',
      description:
        'Visual warehouse dispatch with smart truck-and-order matching, gate seals, and live board updates for peak morning loads.',
      tag: 'Operations',
      href: '/projects/dispatch-engine',
      variant: 'featured' as const,
      bento: 'editorial-bento__4-2',
    },
    {
      title: 'Supplier Control Plane',
      description:
        'Network oversight for suppliers — order vetting, dispatch preview, topology, and treasury across warehouses and retailers.',
      tag: 'Platform',
      href: '/projects/supplier-control-plane',
      variant: 'vertical' as const,
      bento: 'editorial-bento__2-1',
    },
    {
      title: 'Driver Execution App',
      description:
        'Native route execution with sealed manifests, stop-by-stop delivery, cash collection, and live progress reporting.',
      tag: 'Mobile',
      href: '/projects/driver-execution-app',
      variant: 'split' as const,
      bento: 'editorial-bento__2-2',
    },
    {
      title: 'Retailer Commerce',
      description:
        'Catalog, checkout, scheduling, and live order tracking — desktop and mobile parity for retailer teams.',
      tag: 'Commerce',
      href: '/projects/retailer-commerce',
      variant: 'split' as const,
      bento: 'editorial-bento__4-1',
    },
    {
      title: 'Fleet Telemetry',
      description:
        'Live fleet map with planned-vs-actual routes, deviation alerts, and retailer self-serve tracking.',
      tag: 'Visibility',
      href: '/projects/fleet-telemetry',
      variant: 'vertical' as const,
      bento: 'editorial-bento__2-1',
    },
    {
      title: 'Payment Integrity',
      description:
        'Checkout through driver collection to supplier treasury — duplicate protection and a clear audit trail.',
      tag: 'Finance',
      href: '/projects/payment-integrity',
      variant: 'vertical' as const,
      tone: 'light' as const,
      bento: 'editorial-bento__2-1',
    },
  ];

  return (
    <PageSection ref={sectionRef} id="projects">
        <div ref={titleRef}>
          <SectionHeader
            align="center"
            title="Platform Modules"
            description="Core modules that power supplier-led logistics from dispatch to delivery"
          />
        </div>

        <div ref={gridRef} className="editorial-bento">
          {projects.map((project, index) => (
            <ContentCard
              key={project.title}
              variant={project.variant}
              tone={project.tone ?? 'dark'}
              tag={project.tag}
              title={project.title}
              description={project.description}
              image={EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length]}
              href={project.href}
              ctaLabel="READ MORE"
              ctaStyle="link"
              className={project.bento}
              imagePriority={index === 0}
            />
          ))}
        </div>

        <div className="text-center mt-12">
          <Link href="/projects" className="editorial-btn">
            VIEW ALL MODULES
          </Link>
        </div>
    </PageSection>
  );
}
