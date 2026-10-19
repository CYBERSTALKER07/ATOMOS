'use client';

import { usePortalT } from "@/lib/i18n";
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';

export default function FleetLiveMapPage() {
  const t = usePortalT();
  return (
    <PageTransition>
      <PageChrome
        icon="map"
        title={t("warehouse_portal.fleet_live_map.text.fleet_live_map")}
        description={t("warehouse_portal.residual.text.real_time_map_tracking_for_sealed_manifests_in_transit")}
        skeletonVariant="dashboard"
      >
        <div className="rounded-xl border border-(--border) overflow-hidden" style={{ height: 'calc(100vh - 200px)' }}>
          <FleetLiveMapPanel className="h-full w-full" show3DViewToggle={true} />
        </div>
      </PageChrome>
    </PageTransition>
  );
}
