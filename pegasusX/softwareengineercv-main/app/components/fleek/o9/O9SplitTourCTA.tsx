'use client';

import Link from 'next/link';
import Image from 'next/image';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';
import { useLanguage } from '@/app/context/LanguageContext';

type O9SplitTourCTAProps = {
  relatedProjectSlug?: string;
  demoHref?: string;
  tourHref?: string;
};

/**
 * o9-style dual CTA: dark "Request demo" card + light "Take a tour" card.
 * Rendered at the foot of every Fleek page via O9FleekPageLayout.
 */
export default function O9SplitTourCTA({
  relatedProjectSlug,
  demoHref = '/join',
  tourHref,
}: O9SplitTourCTAProps) {
  const { t } = useLanguage();
  const resolvedTourHref = tourHref ?? (relatedProjectSlug ? `/projects/${relatedProjectSlug}` : '/platform');

  return (
    <section className="o9-section o9-split-cta" aria-label="Discover the platform">
      <div className="o9-split-cta__grid">
        <article className="o9-split-cta__card o9-split-cta__card--dark">
          <div className="o9-split-cta__media">
            <Image
              src={EDITORIAL_IMAGES[1]}
              alt=""
              fill
              className="object-cover"
              sizes="(max-width: 900px) 100vw, 50vw"
            />
          </div>
          <div className="o9-split-cta__body">
            <p className="o9-split-cta__eyebrow">{t('licensing_demo_tag')}</p>
            <h3 className="o9-split-cta__heading">{t('licensing_demo_title')}</h3>
            <p className="o9-split-cta__copy">
              {t('licensing_demo_desc')}
            </p>
            <Link href={demoHref} className="o9-btn o9-btn--fill">
              {t('nav_demo')}
            </Link>
          </div>
        </article>

        <article className="o9-split-cta__card o9-split-cta__card--light">
          <div className="o9-split-cta__media">
            <Image
              src={EDITORIAL_IMAGES[2]}
              alt=""
              fill
              className="object-cover"
              sizes="(max-width: 900px) 100vw, 50vw"
            />
          </div>
          <div className="o9-split-cta__body">
            <p className="o9-split-cta__eyebrow o9-split-cta__eyebrow--dark">{t('licensing_tour_tag')}</p>
            <h3 className="o9-split-cta__heading o9-split-cta__heading--dark">{t('licensing_tour_title')}</h3>
            <p className="o9-split-cta__copy o9-split-cta__copy--dark">
              {t('licensing_tour_desc')}
            </p>
            <Link href={resolvedTourHref} className="o9-btn o9-btn--dark">
              {t('nav_tour')}
            </Link>
          </div>
        </article>
      </div>
    </section>
  );
}
