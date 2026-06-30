'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import PixelCard from './PixelCard';
import Link from 'next/link';
import { useIsMobile } from '../hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

export default function Projects() {
  const { isMobile } = useIsMobile();
  const sectionRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);
  const grid1Ref = useRef<HTMLDivElement>(null);
  const grid2Ref = useRef<HTMLDivElement>(null);
  const grid3Ref = useRef<HTMLDivElement>(null);
  const grid4Ref = useRef<HTMLDivElement>(null);
  const grid5Ref = useRef<HTMLDivElement>(null);
  const grid6Ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sectionRef.current) return;

    // Mobile: Simple fade-in only
    if (isMobile) {
      const elements = [titleRef.current, grid1Ref.current, grid2Ref.current, grid3Ref.current, grid4Ref.current, grid5Ref.current, grid6Ref.current];
      gsap.set(elements, { opacity: 1, y: 0 });
      return;
    }

    // Desktop: Full scroll-triggered animations
    const timeline = gsap.timeline({
      scrollTrigger: {
        trigger: sectionRef.current,
        start: 'top 80%',
        end: 'bottom 20%',
        toggleActions: 'play none none reverse'
      }
    });

    timeline.fromTo(
      titleRef.current,
      { opacity: 0, y: 30 },
      { opacity: 1, y: 0, duration: 0.8 }
    );

    // Animate each project card
    [grid1Ref, grid2Ref, grid3Ref, grid4Ref, grid5Ref, grid6Ref].forEach((ref, index) => {
      if (ref.current) {
        timeline.fromTo(
          ref.current,
          { opacity: 0, y: 50 },
          { opacity: 1, y: 0, duration: 0.6 },
          `-=${index === 0 ? 0.3 : 0.4}`
        );
      }
    });
  }, [isMobile]);

  const projects = [
    {
      title: 'Dispatch Engine',
      description: 'Visual warehouse dispatch with smart truck-and-order matching, gate seals, and live board updates for peak morning loads.',
      tech: ['Go', 'Spanner', 'Next.js', 'WebSocket'],
      link: '#',
      variant: 'white' as const,
    },
    {
      title: 'Supplier Control Plane',
      description: 'Network oversight for suppliers — order vetting, dispatch preview, topology, and treasury across warehouses and retailers.',
      tech: ['Next.js', 'TypeScript', 'Go API', 'Treasury'],
      link: '#',
      variant: 'silver' as const,
    },
    {
      title: 'Driver Execution App',
      description: 'Native route execution with sealed manifests, stop-by-stop delivery, cash collection, and live progress reporting.',
      tech: ['Kotlin', 'SwiftUI', 'Telemetry', 'Go API'],
      link: '#',
      variant: 'default' as const,
    },
    {
      title: 'Retailer Commerce',
      description: 'Catalog, checkout, scheduling, and live order tracking — desktop and mobile parity for retailer teams.',
      tech: ['Next.js', 'Tauri', 'React Native', 'Payments'],
      link: '#',
      variant: 'white' as const,
    },
    {
      title: 'Fleet Telemetry',
      description: 'Live fleet map with planned-vs-actual routes, deviation alerts, and retailer self-serve tracking.',
      tech: ['MapLibre', 'Go', 'WebSocket', 'Redis'],
      link: '#',
      variant: 'silver' as const,
    },
    {
      title: 'Payment Integrity',
      description: 'Checkout through driver collection to supplier treasury — duplicate protection and a clear audit trail.',
      tech: ['Go', 'Spanner', 'Webhooks', 'Idempotency'],
      link: '#',
      variant: 'default' as const,
    }
  ];

  const gridRefs = [grid1Ref, grid2Ref, grid3Ref, grid4Ref, grid5Ref, grid6Ref];

  return (
    <section ref={sectionRef} className="py-20 bg-black text-white" id="projects">
      <div className="container mx-auto px-4">
        <div ref={titleRef} className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-4 text-white">
            Platform Modules
          </h2>
          <div className="w-20 h-1 bg-white rounded-full mx-auto mb-6" />
          <p className="text-lg md:text-xl text-white max-w-2xl mx-auto">
            Core modules that power supplier-led logistics from dispatch to delivery
          </p>
        </div>

        {/* Bento Grid */}
        <div className="max-w-7xl mx-auto">
          {/* Desktop: 3 columns, Mobile: 1 column */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-[30px]" style={{ gridAutoRows: '400px' }}>
            {projects.map((project, index) => (
              <div
                key={index}
                ref={gridRefs[index]}
                className={`
                  ${index === 0 ? 'md:col-span-2 lg:col-span-2 md:row-span-2' : ''}
                  ${index === 2 ? 'lg:row-span-2' : ''}
                `}
              >
                <PixelCard
                  variant={project.variant}
                  className="w-full h-full"
                >
                  <div className="absolute inset-0 p-8 md:p-10 flex flex-col justify-between z-10">
                    <div className="flex-1 overflow-hidden">
                      <h3 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 text-white">
                        {project.title}
                      </h3>
                      <p className="text-sm md:text-base text-white/90 mb-6 leading-relaxed line-clamp-3">
                        {project.description}
                      </p>
                    </div>

                    <div className="flex-shrink-0">
                      <div className="flex flex-wrap gap-2 mb-6">
                        {project.tech.map((tech, idx) => (
                          <span
                            key={idx}
                            className="px-3 py-1 text-xs md:text-sm font-medium bg-black/80 text-white border-2 border-white backdrop-blur-sm rounded-2xl"
                          >
                            {tech}
                          </span>
                        ))}
                      </div>

                      <a
                        href={project.link}
                        className="inline-block w-full py-3 px-6 bg-white text-black border-2 border-white hover:bg-black hover:text-white transition-all duration-300 font-bold text-center rounded-2xl"
                      >
                        View Module →
                      </a>
                    </div>
                  </div>
                </PixelCard>
              </div>
            ))}
          </div>
        </div>

        {/* View All Projects Button */}
        <div className="text-center mt-[30px]">
          <Link
            href="/projects"
            className="inline-block px-8 py-4 bg-white text-black border-2 border-white transition-all duration-300 font-bold text-lg rounded-2xl hover:bg-[#FE5934] hover:border-[#FE5934]"
          >
            View All Modules
          </Link>
        </div>
      </div>
    </section>
  );
}
