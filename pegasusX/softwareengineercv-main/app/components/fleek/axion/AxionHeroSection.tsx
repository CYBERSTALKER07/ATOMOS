'use client';

import Image from 'next/image';
import Link from 'next/link';
import type { ReactNode } from 'react';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

type AxionHeroSectionProps = {
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  imageSrc?: string;
  imageAlt?: string;
  visual?: ReactNode;
};

export default function AxionHeroSection({
  title,
  summary,
  primaryHref = '/join',
  primaryLabel = 'Learn More',
  imageSrc = SITE_IMAGES.pegasusContainer,
  imageAlt = 'Pegasus logistics containers',
  visual,
}: AxionHeroSectionProps) {
  return (
    <section className="axion-hero" id="fleek-section-01">
      <div className="axion-hero__top">
        <h1 className="axion-hero__title">
          {title.split('\n').map((line, i) => (
            <span key={line}>
              {line}
              {i < title.split('\n').length - 1 ? <br /> : null}
            </span>
          ))}
        </h1>
        <div className="axion-hero__aside">
          <p className="axion-hero__summary">{summary}</p>
          <Link href={primaryHref} className="axion-btn axion-btn--primary">
            {primaryLabel}
          </Link>
        </div>
      </div>
      <div className="axion-hero__visual">
        {visual ?? (
          <Image
            src={imageSrc}
            alt={imageAlt}
            width={1400}
            height={720}
            className="axion-hero__image"
            priority
            sizes="(max-width: 768px) 100vw, 1200px"
          />
        )}
      </div>
    </section>
  );
}
