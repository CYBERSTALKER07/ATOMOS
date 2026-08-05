'use client';

import type { ReactNode } from 'react';
import { O9FleekPageLayout } from '@/app/components/fleek/o9';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';

type FleekSecondaryLayoutProps = {
  activeHref?: string;
  sectionTitle?: string;
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
  heroVisual?: ReactNode;
  heroImageSrc?: string;
  hubId?: string;
  stackFeatures?: string[];
  section06: ReactNode;
  dataExtra?: ReactNode;
  showStack?: boolean;
  relatedProjectSlug?: string;
};

export default function FleekSecondaryLayout({
  activeHref = '/',
  sectionTitle,
  title,
  summary,
  heroVisual,
  heroImageSrc,
  section06,
  dataExtra,
  relatedProjectSlug,
}: FleekSecondaryLayoutProps) {
  const categoryLabel = sectionTitle ?? 'Pegasus';
  const heroContent = heroVisual ?? dataExtra;

  return (
    <FleekPageShell activeHref={activeHref}>
      <O9FleekPageLayout
        variant="secondary"
        categoryLabel={categoryLabel}
        categoryHref={activeHref}
        title={title}
        summary={summary}
        heroImageSrc={heroImageSrc}
        heroVisual={heroContent}
        proofItems={DEFAULT_PROOF}
        showProofStrip={false}
        showTestimonials={false}
        showTourCta
        details={section06}
        relatedProjectSlug={relatedProjectSlug}
      />
    </FleekPageShell>
  );
}
