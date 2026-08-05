'use client';

import dynamic from 'next/dynamic';
import type { CategoryHub } from '@/app/data/topicPages';
import { HubTopicGrid } from '@/app/components/page-sections';
import type { HubLayoutConfig } from '@/app/lib/explore/hubLayouts';
import { AxionPageLayout } from '@/app/components/fleek/axion';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import FleekDataSection from '@/app/components/fleek/FleekDataSection';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import {
  mapTopicsToSolutions,
  DEFAULT_TECH_FEATURES,
} from '@/app/data/axionSectionContent';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9Related';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), {
  ssr: false,
});

type HubLayoutRendererProps = {
  hub: CategoryHub;
  config: HubLayoutConfig;
};

export default function HubLayoutRenderer({ hub, config }: HubLayoutRendererProps) {
  const fleetBand =
    config.showFleetBand && hub.id === 'apps-deploy' ? (
      <FleetScrollShowcase
        eyebrow="Deploy"
        title="Fleet-ready on every surface"
        subtitle="Warehouse boards, driver missions, and retailer tracking — the same fleet picture whether you deploy portal, mobile, or desktop."
        learnMoreHref="/apps-deploy/dispatch-fleet"
      />
    ) : null;

  const topicLinks = hub.topics.map((t) => ({
    label: t.content.title,
    href: `/${hub.id}/${t.slug}`,
    description: t.content.summary,
  }));

  const industryItems = hub.topics.slice(0, 6).map((t, i) => ({
    title: t.content.title,
    description: t.content.summary,
    icon: (['retail', 'health', 'tech', 'manufacturing', 'fleet', 'warehouse'] as const)[i % 6],
    href: `/${hub.id}/${t.slug}`,
    highlight: i === 3,
  }));

  const techFeatures: typeof DEFAULT_TECH_FEATURES =
    hub.topics.slice(0, 4).map((t) => ({
      title: t.content.title,
      description: t.content.summary,
      href: `/${hub.id}/${t.slug}`,
    }));

  return (
    <FleekPageShell activeHref={`/${hub.id}`}>
      <AxionPageLayout
        hero={{
          title: hub.label,
          summary: config.intro?.body ?? hub.promo?.body ?? `Explore ${hub.label} on Pegasus.`,
          primaryHref: hub.promo?.primaryHref ?? '/join',
          primaryLabel: hub.promo?.primaryLabel ?? 'Learn More',
          imageSrc: EDITORIAL_IMAGES[hub.topics.length % EDITORIAL_IMAGES.length],
        }}
        solutions={{
          title: `${hub.label} solutions`,
          subtitle: config.intro?.body ?? hub.promo?.body,
          items: mapTopicsToSolutions(topicLinks, [...EDITORIAL_IMAGES]),
          seeAllHref: `/${hub.id}`,
        }}
        industries={{
          eyebrow: `/ ${hub.label.toUpperCase()}`,
          title: config.intro?.title ?? `Tailored ${hub.label.toLowerCase()} for your network`,
          description: config.intro?.body ?? hub.promo?.body,
          items: industryItems,
        }}
        technology={{
          eyebrow: '/ TECHNOLOGY',
          title: 'Innovation that moves your business',
          features: techFeatures.length >= 4 ? techFeatures : DEFAULT_TECH_FEATURES,
          extra: (
            <>
              <FleekDataSection hubId={hub.id} extra={fleetBand} />
            </>
          ),
        }}
        details={
          <>
            {config.intro ? (
              <div className="axion-details__intro">
                <p className="axion-eyebrow">{config.intro.eyebrow}</p>
                <h2 className="axion-section__title">{config.intro.title}</h2>
                <p className="axion-section__subtitle">{config.intro.body}</p>
              </div>
            ) : null}
            <HubTopicGrid
              hubId={hub.id}
              hubLabel={hub.label}
              topics={hub.topics}
              layout={config.topicGridLayout}
            />
            <O9TourCTA />
          </>
        }
      />
    </FleekPageShell>
  );
}
