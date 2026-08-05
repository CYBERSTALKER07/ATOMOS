'use client';

import Link from 'next/link';
import FleekSection from './FleekSection';

type FleekCtaBandProps = {
  title?: string;
  body?: string;
  href?: string;
  label?: string;
};

export default function FleekCtaBand({
  title = 'SUPERCHARGE YOUR LOGISTICS STACK',
  body = 'Dispatch boards, live fleet maps, and treasury on one Spanner truth — portal and native apps for every role row.',
  href = '/join',
  label = 'Request demo',
}: FleekCtaBandProps) {
  return (
    <FleekSection id="fleek-section-03" number="03" showConnector>
      <div className="fleek-cta-band">
        <div className="fleek-cta-band__rule" aria-hidden />
        <h3 className="fleek-cta-band__title">{title}</h3>
        <p className="fleek-cta-band__body">{body}</p>
        <Link href={href} className="fleek-btn fleek-btn--accent">{label}</Link>
        <div className="fleek-cta-band__rule" aria-hidden />
      </div>
    </FleekSection>
  );
}
