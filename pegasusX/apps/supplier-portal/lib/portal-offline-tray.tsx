"use client";

import { DesktopOfflineTray, useDesktopConnectivity } from "@/components/ui";
import { isTauri } from "@pegasusx/desktop-bridge";

export function PortalOfflineTray() {
  const { isOffline } = useDesktopConnectivity();

  if (!isTauri() && !isOffline) {
    return null;
  }

  return (
    <DesktopOfflineTray
      isOffline={isOffline}
      offlineMessage="Dashboard and orders may show cached data until you reconnect."
    />
  );
}
