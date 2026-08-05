'use client';

import type { ReactNode } from 'react';
import type {
  AxionIndustryCard,
  AxionSolutionCard,
  AxionTechFeature,
} from '@/app/data/axionSectionContent';
import {
  DEFAULT_INDUSTRIES,
  DEFAULT_SOLUTIONS,
  DEFAULT_TECH_FEATURES,
} from '@/app/data/axionSectionContent';
import AxionHeroSection from './AxionHeroSection';
import AxionSolutionsGrid from './AxionSolutionsGrid';
import AxionIndustriesSection from './AxionIndustriesSection';
import AxionTechnologySection from './AxionTechnologySection';

export type AxionPageLayoutProps = {
  hero: {
    title: string;
    summary: string;
    primaryHref?: string;
    primaryLabel?: string;
    imageSrc?: string;
    imageAlt?: string;
    visual?: ReactNode;
  };
  solutions?: {
    title?: string;
    subtitle?: string;
    items?: AxionSolutionCard[];
    seeAllHref?: string;
  };
  industries?: {
    eyebrow?: string;
    title?: string;
    description?: string;
    items?: AxionIndustryCard[];
  };
  technology?: {
    eyebrow?: string;
    title?: string;
    imageSrc?: string;
    features?: AxionTechFeature[];
    extra?: ReactNode;
  };
  betweenTechAndDetails?: ReactNode;
  details: ReactNode;
};

export default function AxionPageLayout({
  hero,
  solutions,
  industries,
  technology,
  betweenTechAndDetails,
  details,
}: AxionPageLayoutProps) {
  return (
    <>
      <AxionHeroSection {...hero} />
      <AxionSolutionsGrid
        title={solutions?.title}
        subtitle={solutions?.subtitle}
        items={solutions?.items ?? DEFAULT_SOLUTIONS}
        seeAllHref={solutions?.seeAllHref}
      />
      <AxionIndustriesSection
        eyebrow={industries?.eyebrow}
        title={industries?.title}
        description={industries?.description}
        items={industries?.items ?? DEFAULT_INDUSTRIES}
      />
      <AxionTechnologySection
        eyebrow={technology?.eyebrow}
        title={technology?.title}
        imageSrc={technology?.imageSrc}
        features={technology?.features ?? DEFAULT_TECH_FEATURES}
      >
        {technology?.extra}
      </AxionTechnologySection>
      {betweenTechAndDetails}
      <section className="axion-section axion-details" id="fleek-section-06">
        {details}
      </section>
    </>
  );
}
