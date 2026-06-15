'use client';

import { useEffect, useRef } from 'react';
import { FACTORY_SESSION_RECONCILED } from '@/lib/factory-reconnect';

/** Invoke callback when factory transport reconnect completes session reconcile. */
export function useFactorySessionReconcile(onReconcile: () => void) {
  const handlerRef = useRef(onReconcile);
  handlerRef.current = onReconcile;

  useEffect(() => {
    const onEvent = () => {
      handlerRef.current();
    };
    window.addEventListener(FACTORY_SESSION_RECONCILED, onEvent);
    return () => window.removeEventListener(FACTORY_SESSION_RECONCILED, onEvent);
  }, []);
}
