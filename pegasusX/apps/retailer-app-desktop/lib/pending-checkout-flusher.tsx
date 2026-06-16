"use client";

import { useEffect } from "react";
import { flushPendingCheckouts } from "./pending-checkout";

/** Replays queued unified checkouts after reconnect or on dashboard mount. */
export function PendingCheckoutFlusher() {
  useEffect(() => {
    void flushPendingCheckouts();
  }, []);

  useEffect(() => {
    const onOnline = () => {
      void flushPendingCheckouts();
    };
    window.addEventListener("online", onOnline);
    return () => window.removeEventListener("online", onOnline);
  }, []);

  return null;
}
