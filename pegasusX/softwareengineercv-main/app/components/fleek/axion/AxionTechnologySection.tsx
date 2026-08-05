'use client';

import type { ReactNode } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import type { AxionTechFeature } from '@/app/data/axionSectionContent';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

type AxionTechnologySectionProps = {
  eyebrow?: string;
  title?: string;
  imageSrc?: string;
  imageAlt?: string;
  features: AxionTechFeature[];
  children?: ReactNode;
};

export default function AxionTechnologySection({
  eyebrow = '/ TECHNOLOGY',
  title = 'Innovation that moves your business',
  imageSrc = SITE_IMAGES.pegasusContainer,
  imageAlt = 'Pegasus container on site',
  features,
  children,
}: AxionTechnologySectionProps) {
  return (
    <section className="axion-section axion-technology" id="fleek-section-04">
      <p className="axion-eyebrow">{eyebrow}</p>
      <h2 className="axion-section__title axion-technology__title">{title}</h2>
      <div className="axion-technology__split">
        <div className="axion-technology__visual">
          <Image
            src={imageSrc}
            alt={imageAlt}
            width={600}
            height={800}
            className="axion-technology__image"
            sizes="(max-width: 768px) 100vw, 40vw"
          />
        </div>
        <div className="axion-technology__features">
          {features.slice(0, 4).map((feature) => (
            <article key={feature.title} className="axion-tech-feature">
              <h3 className="axion-tech-feature__title">{feature.title}</h3>
              <p className="axion-tech-feature__desc">{feature.description}</p>
              <Link href={feature.href} className="axion-tech-feature__link">
                Learn More
                <span className="axion-tech-feature__arrow" aria-hidden>→</span>
              </Link>
            </article>
          ))}
        </div>
      </div>
      {children ? <div className="axion-technology__extra">{children}</div> : null}
    </section>
  );
}
