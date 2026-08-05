'use client';

import dynamic from 'next/dynamic';
import type { TopicPage } from '@/app/data/topicTypes';
import { getSiblingTopics } from '@/app/data/topicPages';
import ProcessRGrid from '@/app/components/visuals/ProcessRGrid';
import { AxionPageLayout } from '@/app/components/fleek/axion';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import FleekDataSection from '@/app/components/fleek/FleekDataSection';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import {
  mapTopicsToSolutions,
  DEFAULT_TECH_FEATURES,
} from '@/app/data/axionSectionContent';
import { SITE_IMAGES } from '@/app/lib/siteAssets';
import {
  O9WhyItMatters,
  O9CapabilityGrid,
  O9DifferentiatorList,
  O9EdgeCaseGrid,
  O9AiDataPanel,
} from '@/app/components/page-sections/o9/O9Sections';
import { O9RelatedUseCases, O9TourCTA } from '@/app/components/page-sections/o9/O9Related';
import RoleMatrix from '@/app/components/page-sections/RoleMatrix';
import SpecPanel from '@/app/components/page-sections/SpecPanel';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), { ssr: false });

type O9DetailLayoutProps = {
  topic: TopicPage;
  showFleetShowcase?: boolean;
};

export default function O9DetailLayout({ topic, showFleetShowcase }: O9DetailLayoutProps) {
  const { content, categoryLabel, categoryId, slug, badge } = topic;
  const siblings = getSiblingTopics(categoryId, slug, 6);
  const showFleet = showFleetShowcase || ['fleetMap', 'dispatchBoard'].includes(content.flow);
  const useRGrid = content.howItWorks.length > 0;
  const isWarehouse = slug === 'warehouse';

  const capabilityItems = (content.capabilities ?? []).map((c) => ({
    label: c.title,
    href: `/${categoryId}/${slug}`,
    description: c.description,
  }));

  const industryItems = siblings.slice(0, 4).map((s, i) => ({
    title: s.content.title,
    description: s.content.summary,
    icon: (['retail', 'health', 'tech', 'manufacturing'] as const)[i % 4],
    href: `/${categoryId}/${s.slug}`,
    highlight: i === 3,
  }));

  const techFromContent =
    (content.aiAndData ?? content.differentiators ?? []).slice(0, 4).map((item) => ({
      title: item.title,
      description: item.description,
      href: `/${categoryId}/${slug}`,
    }));

  const fleetExtra = showFleet ? (
    <div className="mt-6">
      <FleetScrollShowcase
        eyebrow={categoryLabel}
        title={content.title}
        subtitle={content.summary}
        learnMoreHref="/capabilities/smarter-dispatch"
      />
    </div>
  ) : null;

  return (
    <FleekPageShell activeHref={`/${categoryId}`}>
      <AxionPageLayout
        hero={{
          title: isWarehouse ? "Minimize costs.\nTransport goods" : content.title,
          summary: content.summary,
          primaryHref: '/join',
          primaryLabel: isWarehouse ? 'Request quote' : 'Learn More',
          imageSrc: isWarehouse ? SITE_IMAGES.containerShip : EDITORIAL_IMAGES[slug.length % EDITORIAL_IMAGES.length],
          imageAlt: content.title,
        }}
        solutions={{
          title: `${categoryLabel} solutions`,
          subtitle: content.whyItMatters?.body ?? content.problem,
          items:
            capabilityItems.length >= 3
              ? mapTopicsToSolutions(capabilityItems, [...EDITORIAL_IMAGES])
              : undefined,
          seeAllHref: `/${categoryId}`,
        }}
        industries={{
          eyebrow: `/ ${categoryLabel.toUpperCase()}`,
          title: content.whyItMatters?.headline ?? `How ${content.title} fits your network`,
          description: content.whyItMatters?.body ?? content.problem,
          items: industryItems.length > 0 ? industryItems : undefined,
        }}
        technology={{
          features: techFromContent.length >= 4 ? techFromContent : DEFAULT_TECH_FEATURES,
          imageSrc: SITE_IMAGES.pegasusContainer,
          extra: <FleekDataSection categoryId={categoryId} slug={slug} extra={fleetExtra} />,
        }}
        betweenTechAndDetails={
          content.howItWorks.length > 0 ? (
            <section className="axion-section axion-process" id="fleek-section-05">
              <p className="axion-eyebrow">/ HOW IT WORKS</p>
              <h2 className="axion-section__title">From signal to settled outcome</h2>
              {useRGrid ? (
                <ProcessRGrid steps={content.howItWorks} />
              ) : (
                <ol className="fleek-process-list">
                  {content.howItWorks.map((step, i) => (
                    <li key={step.title} className="fleek-process-list__item">
                      <span className="fleek-process-list__num">{String(i + 1).padStart(2, '0')}</span>
                      <div>
                        <h3 className="font-semibold">{step.title}</h3>
                        <p className="mt-1 text-sm opacity-70">{step.description}</p>
                      </div>
                    </li>
                  ))}
                </ol>
              )}
            </section>
          ) : null
        }
        details={
          <>
            {badge ? <p className="axion-eyebrow mb-4">[{badge}]</p> : null}
            <O9WhyItMatters why={content.whyItMatters} problemFallback={content.problem} />
            <O9CapabilityGrid capabilities={content.capabilities ?? []} layout="r-grid" />
            <O9DifferentiatorList items={content.differentiators ?? []} />
            {content.crossRole && content.crossRole.length > 0 ? (
              <RoleMatrix crossRole={content.crossRole} variant="tabs" />
            ) : null}
            <O9EdgeCaseGrid items={content.edgeCases} />
            <O9AiDataPanel items={content.aiAndData} />
            {content.specs && content.specs.length > 0 ? (
              <SpecPanel specs={content.specs} variant="terminal" />
            ) : null}
            <O9RelatedUseCases siblings={siblings} categoryLabel={categoryLabel} flow={content.flow} />
            <O9TourCTA relatedProjectSlug={content.relatedProjectSlug} />
          </>
        }
      />
    </FleekPageShell>
  );
}
