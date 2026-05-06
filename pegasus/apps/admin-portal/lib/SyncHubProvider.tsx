'use client';

import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useTelemetry } from "../hooks/useTelemetry";

interface SyncContextType {
  registerTopic: (
    topic: string,
    onInvalidate: () => Promise<void>,
  ) => () => void;
}

export const SyncContext = createContext<SyncContextType | null>(null);

/**
 * Root SyncHub Provider that instantiates WebSockets once and propagates
 * updates to components registered via useSyncHub("HYBRID", topic, ...).
 */
export function SyncHubProvider({ children }: { children: React.ReactNode }) {
  const topicsRef = useRef<Record<string, Set<() => Promise<void>>>>({});

  // Tie in standard WebSocket transports
  // 1. Telemetry hub (singleton module scope)
  // Notifications uses multiple connection scopes in older architecture, kept in AdminShell
  useTelemetry();

  const registerTopic = (topic: string, onInvalidate: () => Promise<void>) => {
    if (!topicsRef.current[topic]) {
      topicsRef.current[topic] = new Set();
    }
    topicsRef.current[topic].add(onInvalidate);
    return () => {
      topicsRef.current[topic]?.delete(onInvalidate);
    };
  };

  // Bind to global invalidation events if backend sends 'em over WS
  useEffect(() => {
    const handleInvalidate = (ev: Event) => {
      const topic = (ev as CustomEvent<string>).detail;
      const listeners = topicsRef.current[topic];
      if (listeners) {
        listeners.forEach((fn) => fn());
      }
    };
    window.addEventListener("sync-invalidate", handleInvalidate);
    return () =>
      window.removeEventListener("sync-invalidate", handleInvalidate);
  }, []);

  return (
    <SyncContext.Provider value={{ registerTopic }}>
      {children}
    </SyncContext.Provider>
  );
}
