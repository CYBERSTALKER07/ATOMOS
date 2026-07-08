'use client';

import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';

export default function FleetLiveMapPage() {
  return (
    <PageTransition>
      <PageChrome
        icon="map"
        title="Fleet Live Map"
        description="Real-time map tracking for sealed manifests in transit."
        skeletonVariant="default"
      >
        <div className="rounded-xl border border-(--border) overflow-hidden" style={{ height: 'calc(100vh - 200px)' }}>
          <FleetLiveMapPanel className="h-full w-full" show3DViewToggle={true} />
        </div>
      </PageChrome>
    </PageTransition>
  );
}
