'use client';

import Image from 'next/image';
import { useEffect, useRef } from 'react';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ChamferButton from './ChamferButton';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import { useReducedMotion } from '@/app/hooks/useDevice';

gsap.registerPlugin(ScrollTrigger);

const HERO_IMAGE = FLEET_TRUCK_IMAGES[1];
const CARD_IMAGE = FLEET_TRUCK_IMAGES[3];

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
          className="bw-visual bw-visual--chamfer relative min-h-[360px] aspect-[16/10] md:min-h-[480px] md:aspect-[21/9]"
          style={reduced ? undefined : { clipPath: 'inset(0 0% 0 0)' }}
        >
          <Image
            src={HERO_IMAGE.src}
            alt={HERO_IMAGE.alt}
            fill
            className="bw-visual__img object-cover"
            sizes="(max-width: 768px) 100vw, 1280px"
            priority
          />
          <span className="bw-visual__label">{HERO_IMAGE.caption}</span>

          <article className="editorial-card editorial-card--featured absolute right-3 top-3 z-10 w-[min(42%,220px)] overflow-hidden border border-white/20 shadow-2xl sm:right-5 sm:top-5 sm:w-[min(38%,280px)] md:right-8 md:top-8">
            <div className="editorial-card__media relative aspect-[4/3] w-full">
              <Image
                src={CARD_IMAGE.src}
                alt={CARD_IMAGE.alt}
                fill
                className="object-cover grayscale contrast-[1.08]"
                sizes="280px"
              />
            </div>
            <div className="editorial-card__body !p-3 sm:!p-4">
              <p className="editorial-tag !text-[0.6rem]">{CARD_IMAGE.caption}</p>
              <p className="mt-1 font-mono text-[0.55rem] uppercase tracking-[0.14em] text-black/55 sm:text-[0.625rem]">
                Gate · seal · depart
              </p>
            </div>
          </article>
        </div>

        <p className="mt-8 max-w-2xl font-mono text-xs uppercase tracking-[0.16em] text-white/40">
          Black & white only · Warehouse confirm · Payload seal · Driver route · Retailer receive
        </p>
      </div>
    </section>
  );
}
