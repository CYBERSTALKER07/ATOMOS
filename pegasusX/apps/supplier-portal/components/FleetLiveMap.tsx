'use client';

import React from 'react';
import { GenericFleetLiveMap } from '@/components/ui/GenericFleetLiveMap';

export default function FleetLiveMap(props: React.ComponentProps<typeof GenericFleetLiveMap>) {
  return <GenericFleetLiveMap {...props} />;
}
