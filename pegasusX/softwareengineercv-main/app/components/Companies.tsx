'use client';

import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import {
  motion,
  useScroll,
  useSpring,
  useTransform,
  useMotionValue,
  useVelocity,
  useAnimationFrame
} from 'framer-motion';
import { useLayoutEffect, useState } from 'react';

gsap.registerPlugin(ScrollTrigger);

interface Company {
  name: string;
  logo: string;
  role: string;
  tags: string;
  badge: string;
  remote: string;
  logoStyle?: string;
}

function useElementWidth<T extends HTMLElement>(ref: React.RefObject<T | null>): number {
  const [width, setWidth] = useState(0);

  useLayoutEffect(() => {
    function updateWidth() {
      if (ref.current) {
        setWidth(ref.current.offsetWidth);
      }
    }
    updateWidth();
    window.addEventListener('resize', updateWidth);
    return () => window.removeEventListener('resize', updateWidth);
  }, [ref]);

  return width;
}

const CompanyCard = ({ company }: { company: Company }) => (
  <div className="inline-block mx-4">
    <div className="bg-white text-black border-2 border-black rounded-2xl p-6 min-w-[350px] transition-all duration-300 group company-card hover-orange">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-4">
          {/* Company Logo */}
          <div className="w-12 h-12 bg-black group-hover:bg-white rounded-xl flex items-center justify-center border-2 border-black transition-all duration-300">
            <span className={`font-black text-white group-hover:text-black transition-colors duration-300 ${company.logoStyle || 'text-xl'}`}>
              {company.logo}
            </span>
          </div>
          <div>
            <h3 className="text-2xl font-bold">{company.name}</h3>
            <p className="text-sm text-gray-600 group-hover:text-white transition-colors duration-300">{company.role}</p>
          </div>
        </div>
      </div>
      
      <p className="text-sm text-gray-600 group-hover:text-white transition-colors duration-300 mb-4">
        {company.tags}
      </p>
      
      <div className="flex items-center justify-between gap-2">
        <span className="px-4 py-2 bg-black text-white group-hover:bg-white group-hover:text-black text-xs font-bold rounded-xl border-2 border-black transition-all duration-300">
          {company.badge}
        </span>
        <span className="text-xs text-gray-500 group-hover:text-white transition-colors duration-300">
          • {company.remote}
        </span>
      </div>
    </div>
  </div>
);

interface VelocityScrollProps {
  companies: Company[];
  velocity: number;
  numCopies?: number;
}

function VelocityScroll({ companies, velocity, numCopies = 2 }: VelocityScrollProps) {
  const baseX = useMotionValue(0);
  const { scrollY } = useScroll();
  const scrollVelocity = useVelocity(scrollY);
  const smoothVelocity = useSpring(scrollVelocity, {
    damping: 50,
    stiffness: 400
  });
  const velocityFactor = useTransform(
    smoothVelocity,
    [0, 1000],
    [0, 5],
    { clamp: false }
  );

  const copyRef = useRef<HTMLDivElement>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const copyWidth = useElementWidth(copyRef);
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const node = trackRef.current;
    if (!node) return;

    const observer = new IntersectionObserver(
      ([entry]) => setIsVisible(entry.isIntersecting),
      { rootMargin: '120px' }
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  function wrap(min: number, max: number, v: number): number {
    const range = max - min;
    const mod = (((v - min) % range) + range) % range;
    return mod + min;
  }

  const x = useTransform(baseX, v => {
    if (copyWidth === 0) return '0px';
    return `${wrap(-copyWidth, 0, v)}px`;
  });

  const directionFactor = useRef<number>(1);
  useAnimationFrame((t, delta) => {
    if (!isVisible || copyWidth === 0) return;

    let moveBy = directionFactor.current * velocity * (delta / 1000);

    if (velocityFactor.get() < 0) {
      directionFactor.current = -1;
    } else if (velocityFactor.get() > 0) {
      directionFactor.current = 1;
    }

    moveBy += directionFactor.current * moveBy * velocityFactor.get();
    baseX.set(baseX.get() + moveBy);
  });

  return (
    <div ref={trackRef} className="relative overflow-hidden py-4">
      <motion.div
        className="flex whitespace-nowrap will-change-transform"
        style={{ x }}
      >
        {Array.from({ length: numCopies }).map((_, copyIndex) => (
          <div key={copyIndex} className="flex" ref={copyIndex === 0 ? copyRef : null}>
            {companies.map((company, index) => (
              <CompanyCard key={`${copyIndex}-${index}`} company={company} />
            ))}
          </div>
        ))}
      </motion.div>
    </div>
  );
}

export default function Companies() {
  const sectionRef = useRef<HTMLDivElement>(null);
  const titleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (sectionRef.current && titleRef.current) {
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
        { opacity: 0, y: 50 },
        { opacity: 1, y: 0, duration: 1 }
      );
    }
  }, []);

  const rowOneCompanies: Company[] = [
    {
      name: 'Supplier',
      logo: 'S',
      logoStyle: 'text-3xl font-black',
      role: 'Network Control',
      tags: 'Vetting, Topology, Treasury, Dispatch',
      badge: 'CORE ROLE',
      remote: 'Portal + Mobile'
    },
    {
      name: 'Warehouse',
      logo: 'W',
      logoStyle: 'text-3xl font-black',
      role: 'Dispatch Hub',
      tags: 'Pre-orders, Stock, Fleet Map',
      badge: 'CORE ROLE',
      remote: 'Portal + Android'
    },
    {
      name: 'Factory',
      logo: 'F',
      logoStyle: 'text-3xl font-black',
      role: 'Loading & Seal',
      tags: 'Manifests, Supply, Loading Lanes',
      badge: 'CORE ROLE',
      remote: 'Portal + Mobile'
    },
    {
      name: 'Driver',
      logo: 'D',
      logoStyle: 'text-2xl font-black',
      role: 'Field Execution',
      tags: 'Routes, Delivery, Cash Collection',
      badge: 'CORE ROLE',
      remote: 'Android + iOS'
    },
    {
      name: 'Retailer',
      logo: 'R',
      logoStyle: 'text-3xl font-black',
      role: 'Commerce & Tracking',
      tags: 'Catalog, Checkout, Live Tracking',
      badge: 'CORE ROLE',
      remote: 'Desktop + Mobile'
    }
  ];

  const rowTwoCompanies: Company[] = [
    {
      name: 'Payload',
      logo: 'P',
      logoStyle: 'text-2xl font-black',
      role: 'Gate Control',
      tags: 'Seal, Scan, Terminal, Accountability',
      badge: 'CORE ROLE',
      remote: 'Terminal + Mobile'
    },
    {
      name: 'FMCG Network',
      logo: 'FN',
      logoStyle: 'text-2xl font-black',
      role: 'High-Volume Distribution',
      tags: 'Peak Dispatch, Multi-Site, COD',
      badge: 'SEGMENT',
      remote: 'Multi-Region'
    },
    {
      name: 'Cold Chain',
      logo: 'CC',
      logoStyle: 'text-2xl font-black',
      role: 'Temperature-Sensitive',
      tags: 'Fleet Visibility, SLA Tracking',
      badge: 'SEGMENT',
      remote: 'Regional'
    },
    {
      name: 'Building Materials',
      logo: 'BM',
      logoStyle: 'text-2xl font-black',
      role: 'Heavy Load Logistics',
      tags: 'Capacity Planning, Multi-Stop',
      badge: 'SEGMENT',
      remote: 'Regional'
    },
    {
      name: 'Cash on Delivery',
      logo: 'COD',
      logoStyle: 'text-xl font-black',
      role: 'Payment at Door',
      tags: 'Driver Collection, Reconciliation',
      badge: 'SEGMENT',
      remote: 'Network-Wide'
    }
  ];

  return (
    <section
      ref={sectionRef}
      id="companies"
      className="py-20 bg-black text-white overflow-hidden"
    >
      <div className="container mx-auto px-4 mb-12">
        <div ref={titleRef} className="text-center max-w-3xl mx-auto">
          <h2 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-6 text-white">
            Six Roles, One Network
          </h2>
          <p className="text-lg md:text-xl text-gray-300 leading-relaxed">
            Every team in a supplier-led logistics network — connected on Pegasus
          </p>
        </div>
      </div>

      {/* Row 1 - Scrolling Right */}
      <VelocityScroll companies={rowOneCompanies} velocity={30} numCopies={2} />

      {/* Row 2 - Scrolling Left */}
      <VelocityScroll companies={rowTwoCompanies} velocity={-30} numCopies={2} />
    </section>
  );
}
