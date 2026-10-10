'use client';

import React, { type ReactNode } from 'react';
import type { ProofItem, TopicCard } from '@/app/data/topicTypes';
import { useLanguage } from '@/app/context/LanguageContext';
import { getBusinessValueTabs, getTestimonials, type O9ValueTab } from '@/app/data/o9FleekDefaults';
import O9HeroSplit from './O9HeroSplit';
import O9DifferentiatorGrid, { type DifferentiatorCardItem } from './O9DifferentiatorGrid';
import O9BusinessValueSection, { type BusinessValueTabItem } from './O9BusinessValueSection';
import O9CapabilityShowcase, { type O9CapabilityCard } from './O9CapabilityShowcase';
import O9EnterpriseFaq, { type O9FaqItem, type O9EnterpriseFaqProps } from './O9EnterpriseFaq';
import O9SplitTourCTA from './O9SplitTourCTA';
import O9TestimonialRow from './O9TestimonialRow';
import O9InsightCards from './O9InsightCards';

export interface O9DifferentiatorData {
  kicker?: string;
  title?: string;
  description?: string;
  cards?: DifferentiatorCardItem[];
}

export interface O9BusinessValueData {
  kicker?: string;
  title?: string;
  description?: string;
  tabs?: (O9ValueTab | BusinessValueTabItem)[];
}

export interface O9CapabilityData {
  kicker?: string;
  title?: string;
  description?: string;
  items?: O9CapabilityCard[];
}

export interface O9FaqData {
  kicker?: string;
  title?: string;
  description?: string;
  items?: O9FaqItem[];
  allowMultiple?: boolean;
}

export type O9FleekPageLayoutProps = {
  // 1. Hero
  hero?: ReactNode;
  topicSlug?: string;
  categoryLabel?: string;
  categoryHref?: string;
  title?: string;
  summary?: string;
  badge?: string;
  heroImageSrc?: string;
  heroImageAlt?: string;
  heroVisual?: ReactNode;
  proofItems?: ProofItem[];
  showProofStrip?: boolean;
  hubId?: string;

  // 2. Differentiators Bento
  differentiators?: O9DifferentiatorData | (TopicCard | DifferentiatorCardItem)[];
  differentiatorsTitle?: string;
  showDifferentiators?: boolean;

  // 3. Business Value Tabs
  businessValue?: O9BusinessValueData | (O9ValueTab | BusinessValueTabItem)[];
  outcomes?: string[];
  showBusinessValue?: boolean;

  // 4. Capabilities / Use Cases
  capabilities?: O9CapabilityData | O9CapabilityCard[];
  capabilitiesTitle?: string;
  showCapabilities?: boolean;

  // 5. Details
  details?: ReactNode;

  // 6. Enterprise FAQ
  faq?: O9FaqData | boolean;
  showFaq?: boolean;

  // 7. Conversion Tour CTA
  cta?: ReactNode;
  tourCta?: ReactNode;
  showTourCta?: boolean;
  relatedProjectSlug?: string;

  // Optional extensions
  fleetBand?: ReactNode;
  showTestimonials?: boolean;
  showInsightCards?: boolean;
  howItWorks?: { title: string; description: string }[];
  variant?: 'full' | 'secondary';
};

/**
 * Master Enterprise Solution Page Layout (o9 Narrative Flow):
 * 1. Hero (`hero` or `O9HeroSplit`)
 * 2. Differentiators Bento (`differentiators` using `O9DifferentiatorGrid`)
 * 3. Business Value Tabs (`businessValue` using `O9BusinessValueSection`)
 * 4. Capabilities / Use Cases (`capabilities` using `O9CapabilityShowcase`)
 * 5. Topic Details Grid (`details`)
 * 6. Enterprise FAQ (`faq` using `O9EnterpriseFaq`)
 * 7. Conversion Tour CTA (`cta` or `O9SplitTourCTA`)
 */
export default function O9FleekPageLayout({
  // 1. Hero
  hero,
  topicSlug,
  categoryLabel = 'Pegasus',
  categoryHref = '/',
  title = '',
  summary = '',
  badge,
  heroImageSrc,
  heroImageAlt,
  heroVisual,
  proofItems,
  showProofStrip = true,
  hubId,

  // 2. Differentiators Bento
  differentiators,
  differentiatorsTitle,
  showDifferentiators = true,

  // 3. Business Value Tabs
  businessValue,
  outcomes,
  showBusinessValue = true,

  // 4. Capabilities / Use Cases
  capabilities,
  capabilitiesTitle,
  showCapabilities = true,

  // 5. Details
  details,

  // 6. Enterprise FAQ
  faq,
  showFaq = true,

  // 7. Conversion Tour CTA
  cta,
  tourCta,
  showTourCta = true,
  relatedProjectSlug,

  // Optional extensions
  fleetBand,
  showTestimonials = false,
  showInsightCards = false,
}: O9FleekPageLayoutProps) {
  const { language } = useLanguage();

  // Resolve 1. Hero
  const resolvedHero = hero ?? (
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
  );

  // Resolve 2. Differentiators
  let differentiatorComponent: ReactNode = null;
  if (showDifferentiators && differentiators) {
    if (Array.isArray(differentiators)) {
      if (differentiators.length > 0) {
        differentiatorComponent = (
          <O9DifferentiatorGrid
            items={differentiators}
            title={differentiatorsTitle}
          />
        );
      }
    } else if (typeof differentiators === 'object' && differentiators.cards) {
      differentiatorComponent = (
        <O9DifferentiatorGrid
          kicker={differentiators.kicker}
          title={differentiators.title}
          description={differentiators.description}
          cards={differentiators.cards}
        />
      );
    }
  }

  // Resolve 3. Business Value Tabs
  let businessValueComponent: ReactNode = null;
  if (showBusinessValue) {
    if (businessValue) {
      if (Array.isArray(businessValue) && businessValue.length > 0) {
        businessValueComponent = <O9BusinessValueSection tabs={businessValue} />;
      } else if (
        typeof businessValue === 'object' &&
        'tabs' in businessValue &&
        businessValue.tabs &&
        businessValue.tabs.length > 0
      ) {
        businessValueComponent = (
          <O9BusinessValueSection
            kicker={businessValue.kicker}
            title={businessValue.title}
            description={businessValue.description}
            tabs={businessValue.tabs}
          />
        );
      }
    } else if (hubId) {
      const defaultTabs = getBusinessValueTabs(hubId, outcomes, language);
      if (defaultTabs && defaultTabs.length > 0) {
        businessValueComponent = <O9BusinessValueSection tabs={defaultTabs} />;
      }
    }
  }

  // Resolve 4. Capabilities / Use Cases
  let capabilitiesComponent: ReactNode = null;
  if (showCapabilities && capabilities) {
    if (Array.isArray(capabilities)) {
      if (capabilities.length > 0) {
        capabilitiesComponent = (
          <O9CapabilityShowcase
            items={capabilities}
            title={capabilitiesTitle}
          />
        );
      }
    } else if (typeof capabilities === 'object' && capabilities.items) {
      capabilitiesComponent = (
        <O9CapabilityShowcase
          kicker={capabilities.kicker}
          title={capabilities.title}
          description={capabilities.description}
          items={capabilities.items}
        />
      );
    }
  }

  // Resolve 6. Enterprise FAQ
  let faqComponent: ReactNode = null;
  if (showFaq && faq !== false) {
    if (typeof faq === 'object' && faq !== null) {
      faqComponent = <O9EnterpriseFaq {...(faq as O9EnterpriseFaqProps)} />;
    } else {
      faqComponent = <O9EnterpriseFaq />;
    }
  }

  // Resolve 7. Conversion Tour CTA
  const footerCta =
    cta ??
    (showTourCta
      ? (tourCta ?? (
          <O9SplitTourCTA
            relatedProjectSlug={relatedProjectSlug ?? hubId}
          />
        ))
      : null);

  return (
    <div className="w-full flex flex-col min-w-0">
      {/* 1. Hero Section: 100% Full Width Edge-to-Edge */}
      <div className="w-full min-w-0">{resolvedHero}</div>

      {/* Subsequent Sections in Exact o9 Narrative Flow */}
      <div className="o9-page w-full min-w-0">
        {/* 2. Differentiators Bento Grid */}
        {differentiatorComponent}

        {/* 3. Business Value Tabs */}
        {businessValueComponent}

        {/* 4. Capabilities / Use Cases */}
        {capabilitiesComponent}

        {/* Optional Additions (Fleet Band, Testimonials, Insights) */}
        {fleetBand}
        {showTestimonials && <O9TestimonialRow items={getTestimonials(language)} />}
        {showInsightCards && <O9InsightCards />}

        {/* 5. Topic Details Grid */}
        {details && <div className="o9-details w-full min-w-0">{details}</div>}

        {/* 6. Enterprise FAQ Accordion */}
        {faqComponent}

        {/* 7. Conversion Tour CTA */}
        {footerCta}
      </div>
    </div>
  );
}
