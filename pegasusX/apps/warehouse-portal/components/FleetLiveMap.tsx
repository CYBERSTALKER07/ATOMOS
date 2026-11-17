'use client';

import { GenericFleetLiveMap } from '@/components/ui/GenericFleetLiveMap';
import type { ComponentProps } from 'react';

export default function FleetLiveMap(props: ComponentProps<typeof GenericFleetLiveMap>) {
  return <GenericFleetLiveMap {...props} />;
}
