'use client';

import DeliveryPricingTrends from './DeliveryPricingTrends';
import PopularShippingDestinations from './PopularShippingDestinations';

type LogisticsAnalyticsDashboardProps = {
  defaultCityId?: string;
  className?: string;
};

/** Pricing trends bar chart + popular destinations grid — shared across marketing pages. */
export default function LogisticsAnalyticsDashboard({
  defaultCityId,
  className = '',
}: LogisticsAnalyticsDashboardProps) {
  return (
    <div className={`logistics-analytics ${className}`}>
      <DeliveryPricingTrends defaultCityId={defaultCityId} />
      <PopularShippingDestinations defaultCityId={defaultCityId} />
    </div>
  );
}
