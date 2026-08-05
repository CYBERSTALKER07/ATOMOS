'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ImpactMetricCard from '@/app/components/fleek/cards/ImpactMetricCard';
import type { AppsFamilyConfig } from './AppsFamilyPage.types';

gsap.registerPlugin(ScrollTrigger);

type AppsFamilyPageProps = {
  config: AppsFamilyConfig;
};

export default function AppsFamilyPage({ config }: AppsFamilyPageProps) {
  const gridRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const ctx = gsap.context(() => {
      if (gridRef.current) {
        gsap.fromTo(
          gridRef.current.children,
          { opacity: 0, y: 24 },
          {
            opacity: 1,
            y: 0,
            duration: 0.6,
            stagger: 0.08,
            ease: 'power3.out',
            scrollTrigger: { trigger: gridRef.current, start: 'top 85%', once: true },
          }
        );
      }
    });
    return () => ctx.revert();
  }, []);

  const navHref =
    config.surface === 'web'
      ? '/web-apps'
      : config.surface === 'mobile'
        ? '/mobile-apps'
        : '/desktop-apps';

  return (
    <FleekSecondaryLayout
      activeHref={navHref}
      sectionTitle={config.laneLabel.toUpperCase()}
      title={config.title}
      summary={config.subtitle}
      secondaryHref="/apps-deploy"
      secondaryLabel="ALL SURFACES"
      hubId="apps-deploy"
      heroVisual={
        <div className="flex h-full min-h-[180px] items-center justify-center">
          {config.deviceVisual}
        </div>
      }
      section06={
        <>
          <section className="docs-section">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Core surfaces on this channel
            </h2>
            <div ref={gridRef} className="mt-10 editorial-grid grid grid-cols-1 lg:grid-cols-2">
              <div className="lg:col-span-2">
                <ContentCard
                  variant="featured"
                  tone="light"
                  tag={config.featured.tag}
                  title={config.featured.title}
                  description={config.featured.description}
                  image={config.featured.image}
                  href={config.featured.href}
                  ctaLabel={config.featured.ctaLabel ?? 'REQUEST DEMO'}
                  ctaStyle="button"
                />
              </div>
              {config.apps.map((app) => (
                <ContentCard
                  key={app.title}
                  variant={app.variant ?? 'split'}
                  tone={app.tone ?? 'dark'}
                  tag={app.tag}
                  title={app.title}
                  description={app.description}
                  image={app.image}
                  href={app.href}
                  ctaLabel={app.ctaLabel ?? 'READ MORE'}
                  className={app.className}
                />
              ))}
            </div>
          </section>

          <section className="docs-section">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Why this surface fits the network
            </h2>
            <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {config.features.map((f) => (
                <ContentCard
                  key={f.title}
                  variant="vertical"
                  tone={f.tone ?? 'dark'}
                  tag={f.tag}
                  title={f.title}
                  description={f.description}
                  image={f.image}
                  href={f.href}
                  ctaLabel="READ MORE"
                />
              ))}
            </div>
          </section>
        </>
      }
    />
  );
}

export type { AppCard, AppsFamilyConfig } from './AppsFamilyPage.types';
export { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
