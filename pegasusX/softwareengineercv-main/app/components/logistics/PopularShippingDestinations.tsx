'use client';

import { useState } from 'react';
import {
  LOGISTICS_CITIES,
  formatUsd,
} from '@/app/data/logisticsAnalyticsData';

type PopularShippingDestinationsProps = {
  defaultCityId?: string;
  className?: string;
};

export default function PopularShippingDestinations({
  defaultCityId,
  className = '',
}: PopularShippingDestinationsProps) {
  const [cityId, setCityId] = useState(defaultCityId ?? LOGISTICS_CITIES[0].id);
  const city = LOGISTICS_CITIES.find((c) => c.id === cityId) ?? LOGISTICS_CITIES[0];

  return (
    <section
      className={`logistics-analytics__section ${className}`}
      aria-labelledby="popular-destinations-heading"
    >
      <div className="logistics-analytics__section-head">
        <h2 id="popular-destinations-heading" className="logistics-analytics__heading">
          Popular shipping destinations in{' '}
          <span className="logistics-analytics__city">
            <select
              className="logistics-analytics__city-select"
              value={cityId}
              onChange={(e) => setCityId(e.target.value)}
              aria-label="Select city for popular destinations"
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

      <div className="logistics-analytics__dest-grid">
        {city.destinations.map((route) => (
          <div key={`${route.from}-${route.to}`} className="logistics-analytics__dest-card">
            <p className="logistics-analytics__dest-price">
              USD {formatUsd(route.price)} <span>starting from</span>
            </p>
            <div className="logistics-analytics__dest-route">
              <span>{route.from}</span>
              <span className="logistics-analytics__dest-arrow" aria-hidden>→</span>
              <span>{route.to}</span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
