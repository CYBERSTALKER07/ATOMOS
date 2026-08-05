'use client';

import Link from 'next/link';
import Image from 'next/image';
import { EDITORIAL_IMAGES } from '@/app/components/ContentCard';

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
            <p className="o9-split-cta__eyebrow">Discover our platform</p>
            <h3 className="o9-split-cta__heading">Live demo with a Pegasus expert</h3>
            <p className="o9-split-cta__copy">
              Get a personalized walkthrough with a Pegasus specialist and see how to run
              dispatch, fleet tracking, and payments across your network.
            </p>
            <Link href={demoHref} className="o9-btn o9-btn--fill">
              Request demo
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
            <p className="o9-split-cta__eyebrow o9-split-cta__eyebrow--dark">Discover the platform</p>
            <h3 className="o9-split-cta__heading o9-split-cta__heading--dark">Take a tour</h3>
            <p className="o9-split-cta__copy o9-split-cta__copy--dark">
              See how Pegasus unifies planning, dispatch, and execution through one control
              plane for every role in your network.
            </p>
            <Link href={resolvedTourHref} className="o9-btn o9-btn--dark">
              Take platform tour
            </Link>
          </div>
        </article>
      </div>
    </section>
  );
}
