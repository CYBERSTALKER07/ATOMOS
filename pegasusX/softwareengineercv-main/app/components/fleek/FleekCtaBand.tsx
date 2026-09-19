'use client';

import Link from 'next/link';
import FleekSection from './FleekSection';
import { useLanguage } from '@/app/context/LanguageContext';

type FleekCtaBandProps = {
  title?: string;
  body?: string;
  href?: string;
  label?: string;
};

export default function FleekCtaBand({
  title,
  body,
  href = '/join',
  label,
}: FleekCtaBandProps) {
  const { t } = useLanguage();
  const displayTitle = title || t('cta_band_title');
  const displayBody = body || t('cta_band_body');
  const displayLabel = label || t('btn_request_demo');

  return (
    <FleekSection id="fleek-section-03" number="03" showConnector>
      <div className="fleek-cta-band">
        <div className="fleek-cta-band__rule" aria-hidden />
        <h3 className="fleek-cta-band__title">{displayTitle}</h3>
        <p className="fleek-cta-band__body">{displayBody}</p>
        <Link href={href} className="fleek-btn fleek-btn--accent">{displayLabel}</Link>
        <div className="fleek-cta-band__rule" aria-hidden />
      </div>
    </FleekSection>
  );
}
