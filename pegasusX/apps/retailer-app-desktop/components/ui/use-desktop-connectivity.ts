"use client";

import { useCallback, useSyncExternalStore } from "react";
import { NetworkLifecycleEngine } from "@/lib/enterprise-swr";

export type DesktopConnectivity = {
  isOffline: boolean;
  /** Fires when the browser reports online again. */
  onOnline: (listener: () => void) => () => void;
};

/** Tracks network connectivity via Enterprise NetworkLifecycleEngine with flapping protection. */
export function useDesktopConnectivity(): DesktopConnectivity {
  const isOnline = useSyncExternalStore(
    NetworkLifecycleEngine.subscribe,
    () => NetworkLifecycleEngine.isOnline,
    () => true,
  );

  const onOnline = useCallback((listener: () => void) => {
    let wasOffline = !NetworkLifecycleEngine.isOnline;
    return NetworkLifecycleEngine.subscribe(() => {
      const current = NetworkLifecycleEngine.isOnline;
      if (wasOffline && current) {
        listener();
      }
      wasOffline = !current;
    });
  }, []);

  return { isOffline: !isOnline, onOnline };
}

