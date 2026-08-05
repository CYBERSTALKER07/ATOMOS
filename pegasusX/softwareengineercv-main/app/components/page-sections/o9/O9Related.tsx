'use client';

import ContentCard from '@/app/components/ContentCard';
import ChamferButton from '@/app/components/ChamferButton';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { FLEET_TRUCK_IMAGES } from '@/app/lib/fleetAssets';
import type { TopicPage, FlowVariant } from '@/app/data/topicTypes';
import { O9SectionLabel } from './O9Sections';

const FLEET_FLOWS: FlowVariant[] = ['fleetMap', 'dispatchBoard'];

function imageFor(flow: FlowVariant, index: number) {
  if (FLEET_FLOWS.includes(flow)) {
    return FLEET_TRUCK_IMAGES[index % FLEET_TRUCK_IMAGES.length].src;
  }
  return EDITORIAL_IMAGES[index % EDITORIAL_IMAGES.length];
}

export function O9RelatedUseCases({
  siblings,
  categoryLabel,
  flow,
}: {
  siblings: TopicPage[];
  categoryLabel: string;
  flow: FlowVariant;
}) {
  if (!siblings.length) return null;
  return (
    <section className="docs-section">
      <O9SectionLabel>use cases</O9SectionLabel>
      <h2 className="mt-3 max-w-3xl text-3xl font-semibold tracking-tight md:text-4xl">
        Related use cases on the Pegasus platform
      </h2>
      <div className="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {siblings.slice(0, 6).map((s, i) => (
          <ContentCard
            key={s.slug}
            variant="vertical"
            tone="dark"
            tag={categoryLabel}
            title={s.label}
            description={s.description ?? s.content.summary}
            href={s.href}
            image={imageFor(flow, i)}
            ctaLabel="Read more"
          />
        ))}
      </div>
    </section>
  );
}

export function O9TourCTA({
  relatedProjectSlug,
}: {
  relatedProjectSlug?: string;
}) {
  return (
    <section className="docs-section">
      <div className="docs-surface-raised docs-grain p-8 text-center md:p-14">
        <O9SectionLabel>Discover the platform</O9SectionLabel>
        <h2 className="mx-auto mt-4 max-w-2xl text-3xl font-semibold tracking-tight md:text-4xl">
          Live walkthrough with a Pegasus specialist
        </h2>
        <p className="docs-body mx-auto mt-4 max-w-xl text-white/65">
          See dispatch, fleet tracking, payments, and role apps on one control plane — grounded in
          the same Spanner truth your teams will run.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <ChamferButton href="/join" variant="fill">
            Request demo
          </ChamferButton>
          {relatedProjectSlug ? (
            <ChamferButton href={`/projects/${relatedProjectSlug}`} variant="ghost">
              View module
            </ChamferButton>
          ) : (
            <ChamferButton href="/platform" variant="ghost">
              Take platform tour
            </ChamferButton>
          )}
        </div>
      </div>
    </section>
  );
}
