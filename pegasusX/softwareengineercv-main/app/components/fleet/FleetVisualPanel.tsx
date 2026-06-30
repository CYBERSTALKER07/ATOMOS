'use client';

import { useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useReducedMotion } from '@/app/hooks/useDevice';
import { FLEET_TRUCK_IMAGES, SKETCHFAB_SEMI_EMBED } from '@/app/lib/fleetAssets';

gsap.registerPlugin(ScrollTrigger);

type FleetVisualPanelProps = {
  mode: 'fleet' | 'dispatch';
};

export default function FleetVisualPanel({ mode }: FleetVisualPanelProps) {
  const prefersReducedMotion = useReducedMotion();
  const panelRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLDivElement>(null);
  const modelRef = useRef<HTMLDivElement>(null);
  const [embedReady, setEmbedReady] = useState(false);

  const images =
    mode === 'fleet'
      ? [FLEET_TRUCK_IMAGES[2], FLEET_TRUCK_IMAGES[4], FLEET_TRUCK_IMAGES[5]]
      : [FLEET_TRUCK_IMAGES[0], FLEET_TRUCK_IMAGES[1], FLEET_TRUCK_IMAGES[3]];

  useEffect(() => {
    if (!panelRef.current || prefersReducedMotion) return;

    const ctx = gsap.context(() => {
      const imageEls = imageRef.current?.querySelectorAll('.fleet-panel__slide');
      if (!imageEls?.length) return;

      gsap.set(imageEls, { opacity: 0 });
      gsap.set(imageEls[0], { opacity: 1 });

      const tl = gsap.timeline({
        scrollTrigger: {
          trigger: panelRef.current,
          start: 'top 85%',
          end: 'bottom 60%',
          scrub: 1,
        },
      });

      imageEls.forEach((el, i) => {
        if (i === 0) return;
        tl.to(imageEls[i - 1], { opacity: 0, duration: 0.4 }, i * 0.45)
          .to(el, { opacity: 1, duration: 0.4 }, i * 0.45);
      });

      if (modelRef.current) {
        gsap.fromTo(
          modelRef.current,
          { y: 24, opacity: 0.85 },
          {
            y: 0,
            opacity: 1,
            scrollTrigger: {
              trigger: panelRef.current,
              start: 'top 80%',
              end: 'top 40%',
              scrub: 1,
            },
          }
        );
      }
    }, panelRef);

    return () => ctx.revert();
  }, [mode, prefersReducedMotion]);

  return (
    <div
      ref={panelRef}
      className="fleet-visual-panel relative flex h-full min-h-[18rem] flex-col bg-[#141414] border-l border-white/10"
    >
      <div ref={modelRef} className="relative h-40 shrink-0 border-b border-white/10 md:h-48">
        {!embedReady ? (
          <Image
            src={images[0].src}
            alt=""
            fill
            className="object-cover object-center opacity-50"
            sizes="50vw"
          />
        ) : null}
        <iframe
          title="Fleet 3D model preview"
          src={SKETCHFAB_SEMI_EMBED}
          className={`absolute inset-0 h-full w-full border-0 pointer-events-none ${embedReady ? 'opacity-90' : 'opacity-0'}`}
          allow="autoplay"
          loading="lazy"
          onLoad={() => setEmbedReady(true)}
        />
        <p className="absolute left-4 top-4 font-mono text-[0.6rem] uppercase tracking-[0.18em] text-white/50">
          {mode === 'fleet' ? 'Live fleet map' : 'Dispatch board'}
        </p>
      </div>

      <div ref={imageRef} className="relative flex-1">
        {images.map((img) => (
          <div key={img.src} className="fleet-panel__slide absolute inset-0">
            <Image src={img.src} alt={img.alt} fill className="object-cover" sizes="50vw" />
            <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent" />
            <p className="absolute bottom-4 left-4 right-4 font-mono text-[0.65rem] uppercase tracking-wider text-white/70">
              {img.caption}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}
