'use client';

import type { FleekImpactMetric } from '@/app/data/fleekPageContent';
import { DEFAULT_IMPACT_METRIC } from '@/app/data/fleekPageContent';

type ImpactMetricCardProps = {
  metric?: FleekImpactMetric;
};

export default function ImpactMetricCard({ metric = DEFAULT_IMPACT_METRIC }: ImpactMetricCardProps) {
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
            <span className="impact-card__value">{metric.value}</span>
            <span className="impact-card__unit">{metric.unit ?? '%'}</span>
          </div>
        </div>
        <div className="impact-card__content">
          <p className="impact-card__nav">
            <span aria-hidden>←</span> client: {metric.client} <span aria-hidden>→</span>
          </p>
          <h3 className="impact-card__title">{metric.title}</h3>
          <p className="impact-card__desc">{metric.description}</p>
        </div>
      </div>
    </article>
  );
}
