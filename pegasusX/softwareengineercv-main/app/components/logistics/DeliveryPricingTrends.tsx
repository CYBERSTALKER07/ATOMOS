'use client';

import { useState } from 'react';
import {
  LOGISTICS_CITIES,
  formatUsd,
} from '@/app/data/logisticsAnalyticsData';

type DeliveryPricingTrendsProps = {
  defaultCityId?: string;
  className?: string;
};

export default function DeliveryPricingTrends({
  defaultCityId,
  className = '',
}: DeliveryPricingTrendsProps) {
  const [cityId, setCityId] = useState(defaultCityId ?? LOGISTICS_CITIES[0].id);
  const city = LOGISTICS_CITIES.find((c) => c.id === cityId) ?? LOGISTICS_CITIES[0];
  const maxValue = Math.max(...city.pricingTrends.map((m) => m.value));

  return (
    <section className={`logistics-analytics__section ${className}`} aria-labelledby="pricing-trends-heading">
      <div className="logistics-analytics__section-head">
        <h2 id="pricing-trends-heading" className="logistics-analytics__heading">
          Delivery pricing trends in{' '}
          <span className="logistics-analytics__city">
            <select
              className="logistics-analytics__city-select"
              value={cityId}
              onChange={(e) => setCityId(e.target.value)}
              aria-label="Select city for pricing trends"
            >
              {LOGISTICS_CITIES.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </span>
        </h2>
      </div>

      <div className="logistics-analytics__chart-panel">
        <div className="logistics-analytics__chart" role="img" aria-label={`Monthly delivery pricing for ${city.name}`}>
          {city.pricingTrends.map((month) => {
            const heightPct = Math.max(8, (month.value / maxValue) * 100);
            return (
              <div key={month.label} className="logistics-analytics__bar-col">
                <div className="logistics-analytics__bar-value">
                  ${formatUsd(month.value).replace(/,/g, ' ')}
                </div>
                <div
                  className={`logistics-analytics__bar ${month.highlight ? 'is-highlight' : ''}`}
                  style={{ height: `${heightPct}%` }}
                />
                <span className="logistics-analytics__bar-label">{month.label}</span>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
