'use client';

import Image from 'next/image';
import type { ReactNode } from 'react';
import { O9Hero, O9ProofStrip } from '@/app/components/page-sections/o9/O9Hero';
import type { ProofItem } from '@/app/data/topicTypes';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';
import DossierHero from '@/app/components/dossier/DossierHero';
import { getDossierConfig } from '@/app/components/dossier/dossierPageConfigs';

type O9HeroSplitProps = {
  categoryLabel: string;
  categoryHref: string;
  title: string;
  summary: string;
  badge?: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
  imageSrc?: string;
  imageAlt?: string;
  visual?: ReactNode;
  proofItems?: ProofItem[];
  showProofStrip?: boolean;
};

export default function O9HeroSplit({
  categoryLabel,
  categoryHref,
  title,
  summary,
  badge,
  imageSrc,
  imageAlt = '',
  visual,
  proofItems = DEFAULT_PROOF,
  showProofStrip = true,
}: O9HeroSplitProps) {
  const dossierConfig = getDossierConfig(categoryHref);
  const isSolutionsPage = categoryHref?.startsWith('/solutions');

  if (dossierConfig && !isSolutionsPage) {
    return (
      <section className="o9-hero-split o9-hero-split--dossier w-full pt-4 pb-2">
        <DossierHero
          {...dossierConfig}
          tabLabel={dossierConfig.tabLabel || categoryLabel}
          className="my-1 sm:my-2"
        />
        {showProofStrip ? (
          <div className="o9-hero-split__proof max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-6">
            <O9ProofStrip items={proofItems} />
          </div>
        ) : null}
      </section>
    );
  }

  const hasVisual = Boolean(visual || imageSrc);

  return (
    <section className="o9-hero-split">
      <div className={`o9-hero-split__grid ${!hasVisual ? 'o9-hero-split__grid--single' : ''}`}>
        <O9Hero
          categoryLabel={categoryLabel}
          categoryHref={categoryHref}
          title={title}
          summary={summary}
          badge={badge}
        />
        {hasVisual ? (
          <div className="o9-hero-split__visual">
            {visual ?? (
              <Image
                src={imageSrc!}
                alt={imageAlt || title}
                width={1200}
                height={800}
                className="o9-hero-split__image"
                priority
                sizes="(max-width: 900px) 100vw, 50vw"
              />
            )}
          </div>
        ) : null}
      </div>
      {showProofStrip ? (
        <div className="o9-hero-split__proof">
          <O9ProofStrip items={proofItems} />
        </div>
      ) : null}
    </section>
  );
}
