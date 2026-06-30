'use client';

import Image from 'next/image';
import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ChamferButton from './ChamferButton';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import { useReducedMotion } from '@/app/hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

const DISPATCH_FRAMES = [
  {
    src: FLEET_TRUCK_IMAGES[1].src,
    alt: 'Studio render — single rig ready for dispatch',
    label: 'Dispatch board',
    span: 'md:col-span-2 md:row-span-2',
    tall: true,
  },
  {
    src: FLEET_TRUCK_IMAGES[3].src,
    alt: 'Fleet lineup — capacity planning',
    label: 'Fleet lineup',
    span: '',
    tall: false,
  },
  {
    src: FLEET_TRUCK_IMAGES[5].src,
    alt: 'Load-ready trailer configuration',
    label: 'Gate & seal',
    span: '',
    tall: false,
  },
] as const;

export default function DispatchVisualSection() {
  const sectionRef = useRef<HTMLElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();

  useEffect(() => {
    if (!sectionRef.current || !panelRef.current || reduced) return;
    gsap.fromTo(
      panelRef.current,
      { clipPath: 'inset(0 100% 0 0)' },
      {
        clipPath: 'inset(0 0% 0 0)',
        duration: 1.1,
        ease: 'power3.inOut',
        scrollTrigger: { trigger: sectionRef.current, start: 'top 70%', end: 'top 35%', scrub: 1 },
      }
    );
  }, [reduced]);

  return (
    <section ref={sectionRef} className="border-t border-white/10 bg-black py-20 md:py-28 text-white">
      <div className="container mx-auto max-w-7xl px-4">
        <div className="mb-10 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="editorial-eyebrow">Dispatch & fleet</p>
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Yard to highway — one monochrome thread
            </h2>
          </div>
          <ChamferButton href="/solutions/visual-dispatch-engine" variant="ghost">
            Explore dispatch
          </ChamferButton>
        </div>

        <div
          ref={panelRef}
          className="grid grid-cols-1 gap-3 md:grid-cols-3 md:grid-rows-2"
          style={reduced ? undefined : { clipPath: 'inset(0 0% 0 0)' }}
        >
          {DISPATCH_FRAMES.map((frame) => (
            <div
              key={frame.label}
              className={`bw-visual bw-visual--chamfer relative ${frame.span} ${
                frame.tall ? 'min-h-[320px] md:min-h-full aspect-[4/5] md:aspect-auto' : 'aspect-[16/10]'
              }`}
            >
              <Image
                src={frame.src}
                alt={frame.alt}
                fill
                className="bw-visual__img object-cover"
                sizes="(max-width: 768px) 100vw, 50vw"
              />
              <span className="bw-visual__label">{frame.label}</span>
            </div>
          ))}
        </div>

        <p className="mt-8 max-w-2xl font-mono text-xs uppercase tracking-[0.16em] text-white/40">
          Black & white only · Warehouse confirm · Payload seal · Driver route · Retailer receive
        </p>
      </div>
    </section>
  );
}
