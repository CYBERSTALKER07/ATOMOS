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

    run();
    window.addEventListener("online", run);
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "visible") run();
    });
    return () => {
      window.removeEventListener("online", run);
    };
  }, []);

  return null;
}
