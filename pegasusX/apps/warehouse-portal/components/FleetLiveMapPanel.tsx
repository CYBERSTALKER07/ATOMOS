'use client';

import { useState } from 'react';
import { Map3DViewToggle, useLazyMapMount } from '@pegasusx/ui-kit/desktop';
import FleetLiveMap from '@/components/FleetLiveMap';
import { useWarehouseFleetLiveMap } from '@/lib/use-warehouse-fleet-live-map';

type FleetLiveMapPanelProps = {
  className?: string;
  show3DViewToggle?: boolean;
};

export default function FleetLiveMapPanel({
  className,
  show3DViewToggle = true,
}: FleetLiveMapPanelProps) {
  const { routes, loading, error } = useWarehouseFleetLiveMap();
  const { containerRef, mounted } = useLazyMapMount();
  const [view3D, setView3D] = useState(false);

  return (
    <div ref={containerRef} className={`relative min-h-[200px] ${className ?? ''}`.trim()}>
      {!mounted ? (
        <p className="text-sm text-center px-4 py-8 text-[var(--muted)]">Preparing map…</p>
      ) : (
        <>
          {show3DViewToggle ? (
            <Map3DViewToggle
              className="absolute left-3 top-3 z-10"
              enabled={view3D}
              onChange={setView3D}
            />
          ) : null}
          <FleetLiveMap
            routes={routes}
            loading={loading}
            error={error}
            enable3DView={view3D}
            className="h-full w-full"
          />
        </>
      )}
    </div>
  );
}
