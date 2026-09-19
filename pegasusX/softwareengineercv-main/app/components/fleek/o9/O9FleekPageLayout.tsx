'use client';

import type { ReactNode } from 'react';
import type { ProofItem, TopicCard } from '@/app/data/topicTypes';
import { type O9CapabilityCard } from './O9CapabilityShowcase';
import O9HeroSplit from './O9HeroSplit';
import O9SplitTourCTA from './O9SplitTourCTA';

export type O9FleekPageLayoutProps = {
  variant?: 'full' | 'secondary';
  topicSlug?: string;
  categoryLabel: string;
  categoryHref: string;
  title: string;
  summary: string;
  badge?: string;
  heroImageSrc?: string;
  heroImageAlt?: string;
  heroVisual?: ReactNode;
  proofItems?: ProofItem[];
  showProofStrip?: boolean;
  hubId?: string;
  differentiators?: TopicCard[];
  differentiatorsTitle?: string;
  outcomes?: string[];
  capabilities?: O9CapabilityCard[];
  capabilitiesTitle?: string;
  howItWorks?: { title: string; description: string }[];
  fleetBand?: ReactNode;
  showTestimonials?: boolean;
  showInsightCards?: boolean;
  showTourCta?: boolean;
  details: ReactNode;
  tourCta?: ReactNode;
  relatedProjectSlug?: string;
};

export default function O9FleekPageLayout({
  topicSlug,
  categoryLabel,
  categoryHref,
  title,
  summary,
  badge,
  heroImageSrc,
  heroImageAlt,
  heroVisual,
  proofItems,
  showProofStrip = true,
  showTourCta = true,
  details,
  tourCta,
  relatedProjectSlug,
}: O9FleekPageLayoutProps) {
  const footerCta = showTourCta ? (tourCta ?? <O9SplitTourCTA relatedProjectSlug={relatedProjectSlug} />) : null;

  return (
    <div className="o9-page">
      <O9HeroSplit
        topicSlug={topicSlug}
        categoryLabel={categoryLabel}
        categoryHref={categoryHref}
        title={title}
        summary={summary}
        badge={badge}
        imageSrc={heroImageSrc}
        imageAlt={heroImageAlt}
        visual={heroVisual}
        proofItems={proofItems}
        showProofStrip={showProofStrip}
      />
      <div className="o9-details">{details}</div>
      {footerCta}
    </div>
  );
}
