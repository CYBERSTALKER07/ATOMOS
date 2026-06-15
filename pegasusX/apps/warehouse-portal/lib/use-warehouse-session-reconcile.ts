'use client';

import { useEffect, useRef } from 'react';
import { WAREHOUSE_SESSION_RECONCILED } from '@/lib/warehouse-reconnect';

/** Invoke callback when warehouse transport reconnect completes session reconcile. */
export function useWarehouseSessionReconcile(onReconcile: () => void) {
  const handlerRef = useRef(onReconcile);
  handlerRef.current = onReconcile;

  useEffect(() => {
    const onEvent = () => {
      handlerRef.current();
    };
    window.addEventListener(WAREHOUSE_SESSION_RECONCILED, onEvent);
    return () => window.removeEventListener(WAREHOUSE_SESSION_RECONCILED, onEvent);
  }, []);
}
