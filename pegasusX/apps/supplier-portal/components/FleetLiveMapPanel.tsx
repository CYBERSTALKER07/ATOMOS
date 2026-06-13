'use client';

import FleetLiveMap from '@/components/FleetLiveMap';
import { useFleetLiveMap } from '@/lib/use-fleet-live-map';

type FleetLiveMapPanelProps = {
  className?: string;
};

export default function FleetLiveMapPanel({ className }: FleetLiveMapPanelProps) {
  const { routes, loading, error } = useFleetLiveMap();
  return <FleetLiveMap routes={routes} loading={loading} error={error} className={className} />;
}
