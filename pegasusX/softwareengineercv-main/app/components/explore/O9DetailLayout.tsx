'use client';

import dynamic from 'next/dynamic';
import type { TopicPage } from '@/app/data/topicTypes';
import { getSiblingTopics } from '@/app/data/topicPages';
import { O9FleekPageLayout } from '@/app/components/fleek/o9';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { SITE_IMAGES } from '@/app/lib/siteAssets';
import { DEFAULT_PROOF, DEFAULT_PROOF_RU } from '@/app/data/topicContent/helpers';
import {
  O9WhyItMatters,
  O9EdgeCaseGrid,
  O9AiDataPanel,
} from '@/app/components/page-sections/o9/O9Sections';
import { O9RelatedUseCases } from '@/app/components/page-sections/o9/O9Related';
import RoleMatrix from '@/app/components/page-sections/RoleMatrix';
import SpecPanel from '@/app/components/page-sections/SpecPanel';
import { useLanguage } from '@/app/context/LanguageContext';

const FleetScrollShowcase = dynamic(() => import('@/app/components/fleet/FleetScrollShowcase'), { ssr: false });

type O9DetailLayoutProps = {
  topic: TopicPage;
  showFleetShowcase?: boolean;
};

export default function O9DetailLayout({ topic, showFleetShowcase }: O9DetailLayoutProps) {
  const { language, t } = useLanguage();
  const rawContent = topic.content as any;
  const content = rawContent?.[language] || rawContent?.en || (rawContent?.title ? rawContent : undefined);

  if (!content) return null;

  const { categoryLabel, categoryId, slug, badge } = topic;
  const siblings = getSiblingTopics(categoryId, slug, 6);
  const showFleet = showFleetShowcase || ['fleetMap', 'dispatchBoard'].includes(content?.flow);
  const isWarehouse = slug === 'warehouse';

  const capabilities = (content.capabilities ?? []).map((cap: any, i: number) => ({
    title: cap.title,
    description: cap.description,
    href: `/${categoryId}/${slug}`,
    image: EDITORIAL_IMAGES[(slug.length + i) % EDITORIAL_IMAGES.length],
    tag: categoryLabel,
  }));

  const fleetBand = showFleet ? (
    <div className="o9-section">
      <FleetScrollShowcase
        eyebrow={categoryLabel}
        title={content.title}
        subtitle={content.summary}
        learnMoreHref="/capabilities/smarter-dispatch"
      />
    </div>
  ) : null;

  const defaultProof = language === 'ru' ? DEFAULT_PROOF_RU : DEFAULT_PROOF;
  const warehouseTitle = language === 'ru' ? 'Минимизируйте затраты. Перевозите товары' : 'Minimize costs. Transport goods';
  const diffTitle = language === 'ru'
    ? `Почему лидеры выбирают Pegasus для: ${content.title.toLowerCase()}`
    : `Why leaders choose Pegasus ${content.title.toLowerCase()}`;

  return (
    <FleekPageShell activeHref={`/${categoryId}`}>
      <O9FleekPageLayout
        categoryLabel={categoryLabel}
        categoryHref={`/${categoryId}`}
        title={isWarehouse ? warehouseTitle : content.title}
        summary={content.summary}
        badge={badge}
        heroImageSrc={
          isWarehouse ? SITE_IMAGES.containerShip : EDITORIAL_IMAGES[slug.length % EDITORIAL_IMAGES.length]
        }
        heroImageAlt={content.title}
        proofItems={content.proofItems ?? defaultProof}
        hubId={categoryId}
        differentiators={content.differentiators ?? []}
        differentiatorsTitle={diffTitle}
        outcomes={content.outcomes}
        capabilities={capabilities}
        capabilitiesTitle={language === 'ru' ? 'Что обеспечивает это решение' : 'What this solution enables'}
        howItWorks={content.howItWorks}
        fleetBand={fleetBand}
        relatedProjectSlug={content.relatedProjectSlug}
        showTourCta
        details={
          <>
            <O9WhyItMatters why={content.whyItMatters} problemFallback={content.problem} />
            {content.crossRole && content.crossRole.length > 0 ? (
              <RoleMatrix crossRole={content.crossRole} variant="tabs" />
            ) : null}
            <O9EdgeCaseGrid items={content.edgeCases} />
            <O9AiDataPanel items={content.aiAndData} />
            {content.specs && content.specs.length > 0 ? (
              <SpecPanel specs={content.specs} variant="terminal" />
            ) : null}
            <O9RelatedUseCases siblings={siblings} categoryLabel={categoryLabel} flow={content.flow} />
          </>
        }
      />
    </FleekPageShell>
  );
}
