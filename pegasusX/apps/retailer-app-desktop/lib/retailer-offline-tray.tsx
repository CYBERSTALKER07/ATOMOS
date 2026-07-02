"use client";

import { useCallback, useEffect, useState } from "react";
import {
  DesktopOfflineTray,
  useDesktopConnectivity,
  type DesktopQueuedAction,
} from "@pegasusx/ui-kit/desktop";
import { isTauri } from "@pegasusx/desktop-bridge";
import { flushPendingCheckouts, listPendingCheckouts } from "./pending-checkout";

export function RetailerOfflineTray() {
  const { isOffline } = useDesktopConnectivity();
  const [queuedActions, setQueuedActions] = useState<DesktopQueuedAction[]>([]);
  const [retrying, setRetrying] = useState(false);

  const refreshQueue = useCallback(async () => {
    const queue = await listPendingCheckouts();
    setQueuedActions(
      queue.map((item) => ({
        id: item.id,
        label: "Checkout queued",
        subtitle: item.lastError
          ? `Last error: ${item.lastError}`
          : "Will retry when you are back online",
      })),
    );
  }, []);

  useEffect(() => {
    void refreshQueue();
    const onOnline = () => void refreshQueue();
    window.addEventListener("online", onOnline);
    const timer = window.setInterval(() => void refreshQueue(), 8000);
    return () => {
      window.removeEventListener("online", onOnline);
      window.clearInterval(timer);
    };
  }, [refreshQueue]);

  if (!isTauri() && !isOffline && queuedActions.length === 0) {
    return null;
  }

  const onRetryAll = async () => {
    setRetrying(true);
    try {
      await flushPendingCheckouts();
      await refreshQueue();
    } finally {
      setRetrying(false);
    }
  };

  return (
    <DesktopOfflineTray
      isOffline={isOffline}
      queuedActions={queuedActions}
      onRetryAll={queuedActions.length > 0 ? onRetryAll : undefined}
      retrying={retrying}
      offlineMessage="Cached catalog and orders are shown where available. Checkouts retry when you are back online."
    />
  );
}
