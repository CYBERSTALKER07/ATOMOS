'use client';

import dynamic from 'next/dynamic';
import type { CategoryHub } from '@/app/data/topicPages';
import { HubTopicGrid } from '@/app/components/page-sections';
import type { HubLayoutConfig } from '@/app/lib/explore/hubLayouts';
import { O9FleekPageLayout } from '@/app/components/fleek/o9';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';
import { useLanguage } from '@/app/context/LanguageContext';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), {
  ssr: false,
});

type HubLayoutRendererProps = {
  hub: CategoryHub;
  config: HubLayoutConfig;
};

export default function HubLayoutRenderer({ hub, config }: HubLayoutRendererProps) {
  const { language, t } = useLanguage();
  const categoryLabel = t(`nav_${hub.id}`, hub.label);
  
  const fleetBand =
    config.showFleetBand && hub.id === 'apps-deploy' ? (
      <FleetScrollShowcase
        eyebrow={t('nav_apps-deploy', 'Deploy')}
        title={t('fleet_showcase_title', 'Fleet-ready on every surface')}
        subtitle={t('fleet_showcase_sub', 'Warehouse boards, driver missions, and retailer tracking — the same fleet picture whether you deploy portal, mobile, or desktop.')}
        learnMoreHref="/apps-deploy/dispatch-fleet"
      />
    ) : null;

  const capabilities = hub.topics.map((item, i) => {
    const content = item.content[language] || item.content.en;
    return {
      title: content.title,
      description: content.summary,
      href: `/${hub.id}/${item.slug}`,
      image: EDITORIAL_IMAGES[i % EDITORIAL_IMAGES.length],
      tag: categoryLabel,
    };
  });

  const differentiators = hub.topics.slice(0, 4).map((item) => {
    const content = item.content[language] || item.content.en;
    return {
      title: content.title,
      description: content.summary,
    };
  });

  return (
    <FleekPageShell activeHref={`/${hub.id}`}>
      <O9FleekPageLayout
        categoryLabel={categoryLabel}
        categoryHref={`/${hub.id}`}
        title={config.intro?.title ? t(`hub_${hub.id}_title`, config.intro.title) : categoryLabel}
        summary={config.intro?.body ? t(`hub_${hub.id}_body`, config.intro.body) : `${categoryLabel}`}
        heroImageSrc={EDITORIAL_IMAGES[hub.topics.length % EDITORIAL_IMAGES.length]}
        proofItems={DEFAULT_PROOF}
        hubId={hub.id}
        differentiators={differentiators}
        differentiatorsTitle={t(`hub_${hub.id}_diff_title`, `Why leaders choose Pegasus ${categoryLabel}`)}
        capabilities={capabilities}
        capabilitiesTitle={`${categoryLabel}`}
        fleetBand={fleetBand ? <div className="o9-section">{fleetBand}</div> : undefined}
        showTourCta
        details={
          <>
            {config.intro ? (
              <section className="docs-section">
                <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">
                  {config.intro.eyebrow}
                </p>
                <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
                  {t(`hub_${hub.id}_title`, config.intro.title)}
                </h2>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-white/70">
                  {t(`hub_${hub.id}_body`, config.intro.body)}
                </p>
              </section>
            ) : null}
            <HubTopicGrid
              hubId={hub.id}
              hubLabel={categoryLabel}
              topics={hub.topics}
              layout={config.topicGridLayout}
            />
          </>
        }
      />
    </FleekPageShell>
  );
}
