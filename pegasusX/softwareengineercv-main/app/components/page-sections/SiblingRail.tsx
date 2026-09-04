'use client';

import ContentCard from '@/app/components/ContentCard';
import PageSectionBlock from './PageSectionBlock';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import { useLanguage } from '@/app/context/LanguageContext';

const FLEET_FLOWS: FlowVariant[] = ['fleetMap', 'dispatchBoard'];

function topicImageForFlow(flow: FlowVariant, index: number): string {
  if (FLEET_FLOWS.includes(flow)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

type SiblingRailProps = {
  siblings: TopicPage[];
  categoryLabel: string;
  flow: FlowVariant;
  variant?: 'rail' | 'grid';
};

export default function SiblingRail({
  siblings,
  categoryLabel,
  flow,
  variant = 'grid',
}: SiblingRailProps) {
  const { language, t } = useLanguage();
  if (siblings.length === 0) return null;

  if (variant === 'rail') {
    return (
      <PageSectionBlock eyebrow={t('sec_explore_eyebrow')} title={`${t('sec_more_in')} ${categoryLabel}`}>
        <div className="flex gap-4 overflow-x-auto pb-2 snap-x snap-mandatory">
          {siblings.map((s, i) => {
            const content = s.content[language] || s.content.en;
            return (
              <div key={s.slug} className="w-[min(85vw,320px)] shrink-0 snap-start">
                <ContentCard
                  variant="vertical"
                  tone="dark"
                  tag={categoryLabel}
                  title={content.title}
                  description={content.summary}
                  href={s.href}
                  image={topicImageForFlow(flow, i)}
                />
              </div>
            );
          })}
        </div>
      </PageSectionBlock>
    );
  }

  return (
    <PageSectionBlock eyebrow={t('sec_explore_eyebrow')} title={`${t('sec_more_in')} ${categoryLabel}`}>
      <div className="grid gap-4 md:grid-cols-2">
        {siblings.map((s, i) => {
          const content = s.content[language] || s.content.en;
          return (
            <ContentCard
              key={s.slug}
              variant="split"
              tone="dark"
              tag={categoryLabel}
              title={content.title}
              description={content.summary}
              href={s.href}
              image={topicImageForFlow(flow, i)}
            />
          );
        })}
      </div>
    </PageSectionBlock>
  );
}
