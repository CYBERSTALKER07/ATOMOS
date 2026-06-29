'use client';

import type { WarehouseYardManifest } from '@pegasusx/types';
import FleetLiveMap from '@/components/FleetLiveMap';
import { useWarehouseFleetLiveMap } from '@/lib/use-warehouse-fleet-live-map';

type FleetLiveMapPanelProps = {
  className?: string;
};

export default function FleetLiveMapPanel({ className }: FleetLiveMapPanelProps) {
  const { routes, yardManifests, loading, error } = useWarehouseFleetLiveMap();
  return (
    <div className={className}>
      {yardManifests && yardManifests.length > 0 ? (
        <div className="mb-3 flex flex-wrap gap-2">
          {yardManifests.map((yard: WarehouseYardManifest) => (
            <span
              key={yard.manifest_id}
              className="text-xs rounded-full px-3 py-1 border border-[var(--border)] bg-[var(--surface)]"
              title={yard.delivery_summary}
            >
              Yard · {yard.manifest_id.slice(0, 8)} · {yard.order_count} orders
              {yard.delivery_summary ? ` · ${yard.delivery_summary}` : ''}
            </span>
          ))}
        </div>
      ) : null}
      <FleetLiveMap routes={routes} loading={loading} error={error} />
    </div>
  );
}
