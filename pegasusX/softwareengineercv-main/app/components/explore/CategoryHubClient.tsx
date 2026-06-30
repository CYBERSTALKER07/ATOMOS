'use client';

import dynamic from 'next/dynamic';
import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import type { CategoryHub } from '@/app/data/topicPages';
import SiteNav from './SiteNav';
import ChamferButton from '@/app/components/ChamferButton';
import Footer from '@/app/components/Footer';
import ContentCard, { ContentCardEyebrow } from '@/app/components/ContentCard';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), {
  ssr: false,
});

const FLEET_HUB_IDS = new Set(['solutions', 'apps-deploy']);

function hubTopicImage(hubId: string, index: number): string {
  if (FLEET_HUB_IDS.has(hubId)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

type CategoryHubClientProps = {
  hub: CategoryHub;
};

export default function CategoryHubClient({ hub }: CategoryHubClientProps) {
  const heroRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!heroRef.current) return;
    gsap.fromTo(heroRef.current, { opacity: 0, y: 40 }, { opacity: 1, y: 0, duration: 0.9, ease: 'power3.out' });
  }, []);

  return (
    <main className="min-h-screen bg-black text-white">
      <SiteNav activeHref={`/${hub.id}`} />

      <div className="container mx-auto px-4 pb-20 pt-24 md:pt-28">
        <div ref={heroRef}>
          <ContentCardEyebrow>Explore</ContentCardEyebrow>
          <h1 className="mt-4 max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">{hub.label}</h1>
          {hub.promo ? (
            <p className="mt-6 max-w-3xl text-lg text-white/70">{hub.promo.body}</p>
          ) : null}
        </div>

        {hub.promo ? (
          <div className="mt-12 border border-white/20 p-8 md:p-10">
            <h2 className="text-2xl font-semibold">{hub.promo.title}</h2>
            <div className="mt-6 flex flex-col gap-3 sm:flex-row">
              <ChamferButton href={hub.promo.primaryHref} variant="fill">
                {hub.promo.primaryLabel}
              </ChamferButton>
              {hub.promo.secondaryLabel && hub.promo.secondaryHref ? (
                <ChamferButton href={hub.promo.secondaryHref} variant="ghost">
                  {hub.promo.secondaryLabel}
                </ChamferButton>
              ) : null}
            </div>
          </div>
        ) : null}

        {hub.id === 'solutions' ? (
          <FleetScrollShowcase
            eyebrow="Solutions"
            title="Dispatch and fleet, visualized"
            subtitle="From yard staging to live route tracking — scrub through fleet renders and orbit the rig while you read how Pegasus keeps loads accountable."
            learnMoreHref="/solutions/fleet-visibility"
          />
        ) : null}

        {hub.id === 'apps-deploy' ? (
          <FleetScrollShowcase
            eyebrow="Deploy"
            title="Fleet-ready on every surface"
            subtitle="Warehouse boards, driver missions, and retailer tracking — the same fleet picture whether you deploy portal, mobile, or desktop."
            learnMoreHref="/apps-deploy/dispatch-fleet"
          />
        ) : null}

        <div className="mt-16 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {hub.topics.map((topic, i) => (
            <ContentCard
              key={topic.slug}
              variant="vertical"
              tone={i % 3 === 0 ? 'light' : 'dark'}
              tag={hub.label}
              title={topic.label}
              description={topic.description ?? topic.content.summary}
              href={topic.href}
              image={hubTopicImage(hub.id, i)}
            />
          ))}
        </div>
      </div>

      <Footer />
    </main>
  );
}
