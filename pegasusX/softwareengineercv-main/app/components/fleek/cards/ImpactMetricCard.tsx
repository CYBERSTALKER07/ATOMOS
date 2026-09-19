'use client';

import type { FleekImpactMetric } from '@/app/data/fleekPageContent';
import { getImpactMetric } from '@/app/data/fleekPageContent';
import { useLanguage } from '@/app/context/LanguageContext';

type ImpactMetricCardProps = {
  metric?: FleekImpactMetric;
};

export default function ImpactMetricCard({ metric }: ImpactMetricCardProps) {
  const { language } = useLanguage();
  const resolved = metric ?? getImpactMetric(language);
  const clientLabel = language === 'ru' ? 'клиент' : 'client';

  return (
    <article className="impact-card">
      <div className="impact-card__header">
        <span className="impact-card__brand">
          Impact<span className="impact-card__dot">.</span>
        </span>
        <div className="impact-card__dial" aria-hidden />
      </div>
      <div className="impact-card__body">
        <div className="impact-card__metric-col">
          <div className="impact-card__hatch" aria-hidden />
          <div className="impact-card__metric-block">
            <span className="impact-card__value">{resolved.value}</span>
            <span className="impact-card__unit">{resolved.unit ?? '%'}</span>
          </div>
        </div>
        <div className="impact-card__content">
          <p className="impact-card__nav">
            <span aria-hidden>←</span> {clientLabel}: {resolved.client}{' '}
            <span aria-hidden>→</span>
          </p>
          <h3 className="impact-card__title">{resolved.title}</h3>
          <p className="impact-card__desc">{resolved.description}</p>
        </div>
      </div>
    </article>
  );
}
