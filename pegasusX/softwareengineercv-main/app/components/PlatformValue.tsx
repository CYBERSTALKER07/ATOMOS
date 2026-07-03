'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ContentCard, { EDITORIAL_IMAGES } from './ContentCard';
import { useIsMobile } from '../hooks/useDevice';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';

gsap.registerPlugin(ScrollTrigger);

export default function PlatformValue() {
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
        toggleActions: 'play none none reverse',
      },
    });

    timeline
      .fromTo(titleRef.current, { opacity: 0, y: 24 }, { opacity: 1, y: 0, duration: 0.8 })
      .fromTo(
        gridRef.current?.children ? Array.from(gridRef.current.children) : [],
        { opacity: 0, y: 32 },
        { opacity: 1, y: 0, duration: 0.7, stagger: 0.12 },
        '-=0.35'
      );
  }, [isMobile]);

  return (
    <PageSection ref={sectionRef} id="platform-value">
        <div ref={titleRef}>
          <SectionHeader
            title="Redefining supplier-led logistics for live networks"
          />
        </div>

        <div ref={gridRef} className="editorial-bento">
          <ContentCard
            variant="vertical"
            tone="dark"
            tag="Beyond software"
            title="How Pegasus delivers value"
            description="The only logistics platform built for supplier-led networks — dispatch, tracking, payments, and six-role parity on shared contracts."
            image={EDITORIAL_IMAGES[0]}
            href="/projects"
            ctaLabel="READ MORE"
            ctaStyle="link"
            className="editorial-bento__2-2"
            imagePriority
          />

          <article className="editorial-card editorial-card--featured editorial-card--light editorial-bento__4-2  border-none">
            <div className="editorial-card__media relative min-h-[16rem] border-none bg-black">
            
            </div>
            <div className="editorial-card__body">
              <p className="editorial-tag">Platform overview</p>
              <h3 className="editorial-card__title">
                See Pegasus with your network in mind
              </h3>
              <p className="editorial-card__description">
                Walk through dispatch boards, fleet maps, and payment flows tailored to how your
                supplier-led operation runs today.
              </p>
              <form
                className="mt-6 space-y-4"
                onSubmit={(event) => {
                  event.preventDefault();
                  window.location.href = '/join';
                }}
              >
                <div>
                  <label htmlFor="platform-value-email" className="editorial-eyebrow text-black/60 block mb-2">
                    Work email *
                  </label>
                  <input
                    id="platform-value-email"
                    type="email"
                    required
                    placeholder="you@company.com"
                    className="w-full px-4 py-3 bg-transparent border border-black text-black placeholder:text-black/40 rounded-lg"
                  />
                </div>
                <label className="flex items-start gap-3 text-sm text-black/70">
                  <input type="checkbox" required className="mt-1" />
                  <span>I agree to be contacted about Pegasus.</span>
                </label>
                <button type="submit" className="editorial-btn editorial-btn--on-light w-full sm:w-auto">
                  REQUEST A DEMO
                </button>
              </form>
            </div>
          </article>
        </div>

        <div className="text-center mt-12">
          <Link href="/platform" className="editorial-btn">
            EXPLORE THE PLATFORM
          </Link>
        </div>
    </PageSection>
  );
}
