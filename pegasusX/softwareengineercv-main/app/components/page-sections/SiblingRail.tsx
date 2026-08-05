'use client';

import ContentCard from '@/app/components/ContentCard';
import PageSectionBlock from './PageSectionBlock';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';

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
  if (siblings.length === 0) return null;

  if (variant === 'rail') {
    return (
      <PageSectionBlock eyebrow="Explore" title={`More in ${categoryLabel}`}>
        <div className="flex gap-4 overflow-x-auto pb-2 snap-x snap-mandatory">
          {siblings.map((s, i) => (
            <div key={s.slug} className="w-[min(85vw,320px)] shrink-0 snap-start">
              <ContentCard
                variant="vertical"
                tone="dark"
                tag={categoryLabel}
                title={s.label}
                description={s.description ?? s.content.summary}
                href={s.href}
                image={topicImageForFlow(flow, i)}
              />
            </div>
          ))}
        </div>
      </PageSectionBlock>
    );
  }

  return (
    <PageSectionBlock eyebrow="Explore" title={`More in ${categoryLabel}`}>
      <div className="grid gap-4 md:grid-cols-2">
        {siblings.map((s, i) => (
          <ContentCard
            key={s.slug}
            variant="split"
            tone="dark"
            tag={categoryLabel}
            title={s.label}
            description={s.description ?? s.content.summary}
            href={s.href}
            image={topicImageForFlow(flow, i)}
          />
        ))}
      </div>
    </PageSectionBlock>
  );
}
