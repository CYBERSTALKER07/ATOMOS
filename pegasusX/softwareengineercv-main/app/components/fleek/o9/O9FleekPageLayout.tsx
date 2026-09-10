'use client';

import type { ReactNode } from 'react';
import type { ProofItem, TopicCard } from '@/app/data/topicTypes';
import { getBusinessValueTabs, getTestimonials } from '@/app/data/o9FleekDefaults';
import { O9HowItWorks } from '@/app/components/page-sections/o9/O9Sections';
import O9HeroSplit from './O9HeroSplit';
import O9DifferentiatorGrid from './O9DifferentiatorGrid';
import O9BusinessValueSection from './O9BusinessValueSection';
import O9TestimonialRow from './O9TestimonialRow';
import O9CapabilityShowcase, { type O9CapabilityCard } from './O9CapabilityShowcase';
import O9InsightCards from './O9InsightCards';
import O9SplitTourCTA from './O9SplitTourCTA';
import { useLanguage } from '@/app/context/LanguageContext';

export type O9FleekPageLayoutProps = {
  variant?: 'full' | 'secondary';
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
  variant = 'full',
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
  hubId,
  differentiators = [],
  differentiatorsTitle,
  outcomes,
  capabilities = [],
  capabilitiesTitle,
  howItWorks = [],
  fleetBand,
  showTestimonials = true,
  showInsightCards = true,
  showTourCta = true,
  details,
  tourCta,
  relatedProjectSlug,
}: O9FleekPageLayoutProps) {
  const { language } = useLanguage();
  const isSecondary = variant === 'secondary';
  const showMarketing =
    !isSecondary ||
    differentiators.length > 0 ||
    capabilities.length > 0 ||
    howItWorks.length > 0;
  const valueTabs = getBusinessValueTabs(hubId, outcomes, language);
  const testimonials = getTestimonials(language);
  const footerCta = showTourCta ? (tourCta ?? <O9SplitTourCTA relatedProjectSlug={relatedProjectSlug} />) : null;

  return (
    <div className="o9-page">
      <O9HeroSplit
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
      {showInsightCards ? <O9InsightCards /> : null}
      {showMarketing ? (
        <>
          <O9DifferentiatorGrid items={differentiators} title={differentiatorsTitle} />
          {!isSecondary ? <O9BusinessValueSection tabs={valueTabs} /> : null}
          <O9CapabilityShowcase items={capabilities} title={capabilitiesTitle} />
          {showTestimonials ? <O9TestimonialRow items={testimonials} /> : null}
          <O9HowItWorks steps={howItWorks} variant="list" />
          {fleetBand}
        </>
      ) : null}
      <div className="o9-details">{details}</div>
      {footerCta}
    </div>
  );
}
