'use client';

import type { ReactNode } from 'react';
import type {
  AxionIndustryCard,
  AxionSolutionCard,
  AxionTechFeature,
} from '@/app/data/axionSectionContent';
import {
  getDefaultIndustries,
  getDefaultSolutions,
  getDefaultTechFeatures,
} from '@/app/data/axionSectionContent';
import AxionHeroSection from './AxionHeroSection';
import AxionSolutionsGrid from './AxionSolutionsGrid';
import AxionIndustriesSection from './AxionIndustriesSection';
import AxionTechnologySection from './AxionTechnologySection';
import { useLanguage } from '@/app/context/LanguageContext';

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
  const { language } = useLanguage();

  return (
    <>
      <AxionHeroSection {...hero} />
      <AxionSolutionsGrid
        title={solutions?.title}
        subtitle={solutions?.subtitle}
        items={solutions?.items ?? getDefaultSolutions(language)}
        seeAllHref={solutions?.seeAllHref}
      />
      <AxionIndustriesSection
        eyebrow={industries?.eyebrow}
        title={industries?.title}
        description={industries?.description}
        items={industries?.items ?? getDefaultIndustries(language)}
      />
      <AxionTechnologySection
        eyebrow={technology?.eyebrow}
        title={technology?.title}
        imageSrc={technology?.imageSrc}
        features={technology?.features ?? getDefaultTechFeatures(language)}
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
