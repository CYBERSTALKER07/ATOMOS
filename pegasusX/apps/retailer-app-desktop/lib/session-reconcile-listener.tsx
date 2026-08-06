"use client";


import { usePortalT } from "@/lib/i18n";
import { useEffect } from "react";
import { useWebSocket } from "./ws";
import { reconcileRetailerSession } from "./session-reconcile";

/** Runs server-authoritative snapshot refetch after each WS reconnect. */
export function SessionReconcileListener() {
  const t = usePortalT();
  const { reconnectEpoch } = useWebSocket();

  useEffect(() => {
    if (reconnectEpoch === 0) return;
    void reconcileRetailerSession().then(() => {
      void import("./pending-checkout").then(({ flushPendingCheckouts }) => flushPendingCheckouts());
      window.dispatchEvent(
        new CustomEvent("retailer:session-reconciled", { detail: { epoch: reconnectEpoch } }),
      );
    });
  }, [reconnectEpoch]);

  return null;
}
