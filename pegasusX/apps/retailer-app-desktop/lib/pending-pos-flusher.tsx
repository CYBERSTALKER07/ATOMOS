"use client";

import { useEffect } from "react";
import { flushPendingPosSales } from "./pending-pos-sales";

/** Drains offline POS sales when the app comes online or mounts. */
export function PendingPosFlusher() {
  useEffect(() => {
    const run = () => {
      if (typeof navigator !== "undefined" && !navigator.onLine) return;
      void flushPendingPosSales().catch((err) => {
        console.error("pending pos flush failed:", err);
      });
    };

    const handleVisibility = () => {
      if (document.visibilityState === "visible") run();
    };

    run();
    window.addEventListener("online", run);
    document.addEventListener("visibilitychange", handleVisibility);
    return () => {
      window.removeEventListener("online", run);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, []);

  return null;
}
