'use client';

import { GenericFleetLiveMap } from '@pegasusx/ui-maps';
import type { FactoryFleetLiveRoute } from '@/lib/use-factory-fleet-live-map';

export default function FleetLiveMap(props: any) {
  return <GenericFleetLiveMap {...props} />;
}
