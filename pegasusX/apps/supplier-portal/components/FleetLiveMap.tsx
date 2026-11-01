'use client';

import React from 'react';
import { GenericFleetLiveMap } from '@pegasusx/ui-maps';

export default function FleetLiveMap(props: React.ComponentProps<typeof GenericFleetLiveMap>) {
  return <GenericFleetLiveMap {...props} />;
}
