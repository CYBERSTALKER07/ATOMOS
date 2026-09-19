'use client';

import { useMemo, useState } from 'react';
import { getLocalizedCities, formatUsd } from '@/app/data/logisticsAnalyticsData';
import { useLanguage } from '@/app/context/LanguageContext';

type PopularShippingDestinationsProps = {
  defaultCityId?: string;
  className?: string;
};

export default function PopularShippingDestinations({
  defaultCityId,
  className = '',
}: PopularShippingDestinationsProps) {
  const { language, t } = useLanguage();
  const cities = useMemo(() => getLocalizedCities(language), [language]);
  const [cityId, setCityId] = useState(defaultCityId ?? cities[0].id);
  const city = cities.find((c) => c.id === cityId) ?? cities[0];

  return (
    <section
      className={`logistics-analytics__section ${className}`}
      aria-labelledby="popular-destinations-heading"
    >
      <div className="logistics-analytics__section-head">
        <h2 id="popular-destinations-heading" className="logistics-analytics__heading">
          {t('logistics_destinations_in', 'Popular shipping destinations in')}{' '}
          <span className="logistics-analytics__city">
            <select
              className="logistics-analytics__city-select"
              value={cityId}
              onChange={(e) => setCityId(e.target.value)}
              aria-label={t('logistics_select_city_dest', 'Select city for popular destinations')}
            >
              {cities.map((c) => (
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
              USD {formatUsd(route.price, language)}{' '}
              <span>{t('logistics_starting_from', 'starting from')}</span>
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
