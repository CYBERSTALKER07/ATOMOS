"use client";

import { DesktopOfflineTray, useDesktopConnectivity } from "@pegasusx/ui-kit/desktop";
import { isTauri } from "@pegasusx/desktop-bridge";

export function PortalOfflineTray() {
  const { isOffline } = useDesktopConnectivity();

  if (!isTauri() && !isOffline) {
    return null;
  }

  return (
    <DesktopOfflineTray
      isOffline={isOffline}
      offlineMessage="Manifest and loading-bay views may show cached data until you reconnect."
    />
  );
}
