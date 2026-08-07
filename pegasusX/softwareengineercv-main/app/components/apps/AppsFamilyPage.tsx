'use client';

import { useEffect, useRef } from 'react';
import Link from 'next/link';
import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import ContentCard, { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import FleekSecondaryLayout from '@/app/components/fleek/FleekSecondaryLayout';
import ImpactMetricCard from '@/app/components/fleek/cards/ImpactMetricCard';
import type { AppsFamilyConfig } from './AppsFamilyPage.types';
import { useLanguage } from '@/app/context/LanguageContext';

gsap.registerPlugin(ScrollTrigger);

type AppsFamilyPageProps = {
  config: AppsFamilyConfig;
  configRu?: AppsFamilyConfig;
};

export default function AppsFamilyPage({ config, configRu }: AppsFamilyPageProps) {
  const gridRef = useRef<HTMLDivElement>(null);
  const { t, language } = useLanguage();
  const active = language === 'ru' && configRu ? configRu : config;

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
    active.surface === 'web'
      ? '/web-apps'
      : active.surface === 'mobile'
        ? '/mobile-apps'
        : '/desktop-apps';

  return (
    <FleekSecondaryLayout
      activeHref={navHref}
      sectionTitle={active.laneLabel.toUpperCase()}
      title={active.title}
      summary={active.subtitle}
      secondaryHref="/apps-deploy"
      secondaryLabel={t('sec_all_surfaces', 'ALL SURFACES')}
      hubId="apps-deploy"
      heroVisual={
        <div className="flex h-full min-h-[180px] items-center justify-center">
          {active.deviceVisual}
        </div>
      }
      section06={
        <>
          <section className="docs-section">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              {t('sec_core_surfaces', 'Core surfaces on this channel')}
            </h2>
            <div ref={gridRef} className="mt-10 editorial-grid grid grid-cols-1 lg:grid-cols-2">
              <div className="lg:col-span-2">
                <ContentCard
                  variant="featured"
                  tone="light"
                  tag={active.featured.tag}
                  title={active.featured.title}
                  description={active.featured.description}
                  image={active.featured.image}
                  href={active.featured.href}
                  ctaLabel={active.featured.ctaLabel ?? t('nav_demo', 'REQUEST DEMO')}
                  ctaStyle="button"
                />
              </div>
              {active.apps.map((app) => (
                <ContentCard
                  key={app.title}
                  variant={app.variant ?? 'split'}
                  tone={app.tone ?? 'dark'}
                  tag={app.tag}
                  title={app.title}
                  description={app.description}
                  image={app.image}
                  href={app.href}
                  ctaLabel={app.ctaLabel ?? t('btn_read_more', 'READ MORE')}
                  className={app.className}
                />
              ))}
            </div>
          </section>

          <section className="docs-section">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              {t('sec_why_surface_fits', 'Why this surface fits the network')}
            </h2>
            <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {active.features.map((f) => (
                <ContentCard
                  key={f.title}
                  variant="vertical"
                  tone={f.tone ?? 'dark'}
                  tag={f.tag}
                  title={f.title}
                  description={f.description}
                  image={f.image}
                  href={f.href}
                  ctaLabel={t('btn_read_more', 'READ MORE')}
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
