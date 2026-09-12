'use client';

import ChamferButton from '@/app/components/ChamferButton';
import { O9ProofStrip } from '@/app/components/page-sections/o9/O9Hero';
import { O9SectionLabel } from '@/app/components/page-sections/o9/O9Sections';
import { O9TourCTA } from '@/app/components/page-sections/o9/O9Related';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';
import LogisticsAnalyticsDashboard from '@/app/components/logistics/LogisticsAnalyticsDashboard';
import type { ProofItem } from '@/app/data/topicTypes';

type O9PageHeroProps = {
  eyebrow: string;
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
  proofItems?: ProofItem[];
  children?: React.ReactNode;
};

/** Shared o9-style hero + proof for standalone marketing pages. */
export function O9PageHero({
  eyebrow,
  title,
  summary,
  primaryHref = '/join',
  primaryLabel = 'Request demo',
  secondaryHref,
  secondaryLabel,
  proofItems = DEFAULT_PROOF,
  children,
}: O9PageHeroProps) {
  return (
    <>
      <div className="docs-reveal grid gap-10 lg:grid-cols-2 lg:items-center lg:gap-16">
        <div className="v-stack gap-0">
          <p className="editorial-eyebrow">{eyebrow}</p>
          <h1 className="docs-hero-title mt-4 text-4xl font-semibold tracking-tight md:text-6xl">{title}</h1>
          <p className="docs-body mt-6 max-w-xl text-lg text-white/70">{summary}</p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <ChamferButton href={primaryHref} variant="fill">
              {primaryLabel}
            </ChamferButton>
            {secondaryHref && secondaryLabel ? (
              <ChamferButton href={secondaryHref} variant="ghost">
                {secondaryLabel}
              </ChamferButton>
            ) : null}
          </div>
        </div>
        {children ? (
          <div className="docs-surface docs-grain relative min-h-[180px] p-6 md:min-h-[240px]">{children}</div>
        ) : null}
      </div>
      <O9ProofStrip items={proofItems} />
      <LogisticsAnalyticsDashboard />
    </>
  );
}

export { O9SectionLabel, O9TourCTA, O9ProofStrip };
