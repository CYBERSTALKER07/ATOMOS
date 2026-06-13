'use client';

import FleetLiveMap from '@/components/FleetLiveMap';
import { useWarehouseFleetLiveMap } from '@/lib/use-warehouse-fleet-live-map';

type FleetLiveMapPanelProps = {
  className?: string;
};

export default function FleetLiveMapPanel({ className }: FleetLiveMapPanelProps) {
  const { routes, loading, error } = useWarehouseFleetLiveMap();
  return <FleetLiveMap routes={routes} loading={loading} error={error} className={className} />;
}
