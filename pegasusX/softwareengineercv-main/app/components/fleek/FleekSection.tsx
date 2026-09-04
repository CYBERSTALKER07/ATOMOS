'use client';

import type { ReactNode } from 'react';

type FleekSectionProps = {
  id: string;
  number: string;
  title?: string;
  subtitle?: string;
  children: ReactNode;
  showConnector?: boolean;
  className?: string;
};

export default function FleekSection({
  id,
  number,
  title,
  subtitle,
  children,
  showConnector = true,
  className = '',
}: FleekSectionProps) {
  return (
    <section id={id} className={`fleek-section ${className}`} data-section={number}>
      <div className="fleek-section__head">
        <span className="fleek-section__num">{number}</span>
        {title ? (
          <div>
            <h2 className="fleek-section__title">{title}</h2>
            {subtitle ? <p className="fleek-section__subtitle">{subtitle}</p> : null}
          </div>
        ) : null}
      </div>
      <div className="fleek-section__body">{children}</div>
      {showConnector ? <div className="fleek-section__connector" aria-hidden /> : null}
    </section>
  );
}
