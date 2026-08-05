'use client';

import { useEffect, useRef } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useIsMobile, useReducedMotion } from '@/app/hooks/useDevice';
import { useInView } from '@/app/hooks/useInView';
import {
  FLEET_SHOWCASE_CAPTIONS,
  FLEET_TRUCK_IMAGES,
} from '@/app/lib/fleetAssets';

gsap.registerPlugin(ScrollTrigger);

type FleetScrollShowcaseProps = {
  eyebrow?: string;
  title?: string;
  subtitle?: string;
  learnMoreHref?: string;
};

export default function FleetScrollShowcase({
  eyebrow = 'Fleet & dispatch',
  title = 'See the fleet before it moves',
  subtitle = 'Scroll through load planning, gate accountability, and live route tracking — one honest picture of every truck in the network.',
  learnMoreHref = '/solutions/fleet-visibility',
}: FleetScrollShowcaseProps) {
  const { isMobile } = useIsMobile();
  const prefersReducedMotion = useReducedMotion();
  const { ref: inViewRef, isInView } = useInView<HTMLElement>({ rootMargin: '400px', exit: true });
  const sectionRef = useRef<HTMLElement>(null);
  const pinRef = useRef<HTMLDivElement>(null);
  const heroWrapRef = useRef<HTMLDivElement>(null);
  const filmstripRef = useRef<HTMLDivElement>(null);
  const captionRef = useRef<HTMLParagraphElement>(null);
  const progressRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!sectionRef.current || !pinRef.current || isMobile || prefersReducedMotion || !isInView) return;

    const ctx = gsap.context(() => {
      const filmstrip = filmstripRef.current;
      const heroWrap = heroWrapRef.current;
      const caption = captionRef.current;
      const progress = progressRef.current;

      if (!filmstrip || !heroWrap) return;

      const slideCount = FLEET_TRUCK_IMAGES.length;
      const stripWidth = filmstrip.scrollWidth - filmstrip.clientWidth;

      if (progress) {
        gsap.set(progress, { scaleX: 0, transformOrigin: 'left center' });
      }

      const tl = gsap.timeline({
        scrollTrigger: {
          trigger: sectionRef.current,
          start: 'top top',
          end: `+=${slideCount * 55}%`,
          pin: pinRef.current,
          scrub: 1,
          anticipatePin: 1,
        },
      });

      tl.to(heroWrap, { scale: 1.03, duration: 1, ease: 'none' }, 0)
        .to(filmstrip, { x: () => -stripWidth, ease: 'none', duration: 1 }, 0)
        .to(progress, { scaleX: 1, ease: 'none', duration: 1 }, 0);

      FLEET_SHOWCASE_CAPTIONS.forEach((text, i) => {
        const at = i / (slideCount - 1);
        tl.call(
          () => {
            if (caption) caption.textContent = text;
          },
          [],
          at,
        );
      });
    }, sectionRef);

    return () => ctx.revert();
  }, [isMobile, prefersReducedMotion, isInView]);

  const showStatic = isMobile || prefersReducedMotion;
  const hero = FLEET_TRUCK_IMAGES[0];

  return (
    <section
      ref={(node) => {
        sectionRef.current = node;
        inViewRef.current = node;
      }}
      className="fleet-scroll-showcase relative bg-black text-white"
      aria-label="Fleet visual showcase"
    >
      <div ref={pinRef} className="relative min-h-screen flex flex-col justify-center overflow-hidden">
        <div className="container mx-auto px-4 py-16 md:py-20">
          <div className="grid gap-10 lg:grid-cols-2 lg:items-center lg:gap-16">
            <div>
              <p className="editorial-eyebrow text-white/50">{eyebrow}</p>
              <h2 className="mt-4 text-3xl font-semibold tracking-tight md:text-5xl">{title}</h2>
              <p className="mt-4 max-w-lg text-white/65 leading-relaxed">{subtitle}</p>
              <p
                ref={captionRef}
                className="mt-6 font-mono text-xs uppercase tracking-[0.2em] text-[#FFA500]"
              >
                {FLEET_SHOWCASE_CAPTIONS[0]}
              </p>
              <div className="mt-4 h-px w-full max-w-xs bg-white/15 origin-left scale-x-0" ref={progressRef} />
              {learnMoreHref ? (
                <Link href={learnMoreHref} className="editorial-btn mt-8 inline-flex">
                  Explore fleet tracking
                </Link>
              ) : null}
            </div>

            <div
              ref={heroWrapRef}
              className="fleet-scroll-showcase__model relative aspect-[4/3] w-full border border-white/15 bg-[#0a0a0a] overflow-hidden"
            >
              <Image
                src={hero.src}
                alt={hero.alt}
                fill
                className="object-cover"
                sizes="(max-width: 1024px) 100vw, 50vw"
                priority
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent" />
              <p className="absolute bottom-3 left-3 right-3 font-mono text-[10px] uppercase tracking-wider text-white/70">
                {hero.caption}
              </p>
            </div>
          </div>

          <div className="mt-12 overflow-hidden">
            <p className="mb-4 font-mono text-[10px] uppercase tracking-[0.2em] text-white/40">
              {showStatic ? 'Fleet renders' : 'Scroll to scrub the yard'}
            </p>
            <div
              ref={filmstripRef}
              className={`flex gap-3 ${showStatic ? 'flex-wrap' : 'w-max'}`}
            >
              {FLEET_TRUCK_IMAGES.map((img, i) => (
                <div
                  key={img.src}
                  className="relative h-36 w-56 shrink-0 overflow-hidden border border-white/15 md:h-44 md:w-72"
                >
                  <Image
                    src={img.src}
                    alt={img.alt}
                    fill
                    className="object-cover"
                    sizes="288px"
                    priority={i < 2}
                  />
                  <span className="absolute bottom-0 left-0 right-0 bg-black/70 px-2 py-1 font-mono text-[9px] uppercase tracking-wider text-white/70">
                    {img.caption}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
