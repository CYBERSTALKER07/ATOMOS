"use client";

import { useEffect, useRef } from "react";

/** Invoke callback when retailer transport reconnect completes session reconcile. */
export function useRetailerSessionReconcile(onReconcile: () => void) {
  const handlerRef = useRef(onReconcile);
  handlerRef.current = onReconcile;

  useEffect(() => {
    const onEvent = () => {
      handlerRef.current();
    };
    window.addEventListener("retailer:session-reconciled", onEvent);
    return () => window.removeEventListener("retailer:session-reconciled", onEvent);
  }, []);
}
