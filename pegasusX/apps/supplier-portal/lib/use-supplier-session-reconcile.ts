"use client";

import { useEffect, useRef } from "react";
import { SUPPLIER_SESSION_RECONCILED } from "@/lib/supplier-reconnect";

/** Invoke callback when supplier transport reconnect completes session reconcile. */
export function useSupplierSessionReconcile(onReconcile: () => void) {
  const handlerRef = useRef(onReconcile);
  handlerRef.current = onReconcile;

  useEffect(() => {
    const onEvent = () => {
      handlerRef.current();
    };
    window.addEventListener(SUPPLIER_SESSION_RECONCILED, onEvent);
    return () => window.removeEventListener(SUPPLIER_SESSION_RECONCILED, onEvent);
  }, []);
}
