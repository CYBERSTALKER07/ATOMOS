'use client';

import { GenericFleetLiveMap } from '@pegasusx/ui-maps';
import type { ComponentProps } from 'react';

export default function FleetLiveMap(props: ComponentProps<typeof GenericFleetLiveMap>) {
  return <GenericFleetLiveMap {...props} />;
}
