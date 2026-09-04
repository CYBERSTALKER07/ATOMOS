"use client";

import { useEffect, useRef } from "react";
import { supplierApiBaseUrl } from "@/lib/auth";
import { runSupplierSessionReconcile } from "@/lib/session-reconcile";
import { parseSupplierWsEventType } from "@/lib/supplier-ws-events";
import { SSE_SUPPLIER_ENDPOINT } from "@pegasusx/ws-refresh-contract";

type UseSupplierWsRefreshOptions = {
  eventTypes: ReadonlySet<string>;
  /** Debounce rapid events (e.g. GPS) before calling onSignal. */
  debounceMs?: number;
  enabled?: boolean;
};

/**
 * Opens a supplier-scoped Server-Sent Events (SSE) session against `/v1/supplier/events`
 * and invokes onSignal when matching events arrive.
 * Replaces legacy WebSocket transport with lightweight unidirectional HTTP streaming.
 */
export function useSupplierWsRefresh(
  onSignal: (eventType: string, raw?: unknown) => void,
  { eventTypes, debounceMs = 500, enabled = true }: UseSupplierWsRefreshOptions,
) {
  const onSignalRef = useRef(onSignal);
  onSignalRef.current = onSignal;

  useEffect(() => {
    if (!enabled || typeof window === "undefined") {
      return;
    }

    let cancelled = false;
    let eventSource: EventSource | null = null;
    let signalTimer: number | undefined;
    let hasConnectedOnce = false;

    const clearTimers = () => {
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
        signalTimer = undefined;
      }
    };

    const closeStream = () => {
      if (!eventSource) {
        return;
      }
      eventSource.onopen = null;
      eventSource.onmessage = null;
      eventSource.onerror = null;
      eventSource.close();
      eventSource = null;
    };

    const handleEvent = (eventType: string | null, rawData: unknown) => {
      if (!eventType || !eventTypes.has(eventType) || eventType.startsWith("SYSTEM")) {
        return;
      }
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
      }
      signalTimer = window.setTimeout(() => {
        if (cancelled) {
          return;
        }
        onSignalRef.current(eventType, rawData);
      }, debounceMs);
    };

    try {
      const apiBase = supplierApiBaseUrl();
      const sseUrl = `${apiBase}${SSE_SUPPLIER_ENDPOINT}`;
      eventSource = new EventSource(sseUrl, { withCredentials: true });

      eventSource.onopen = () => {
        if (hasConnectedOnce) {
          void runSupplierSessionReconcile();
        }
        hasConnectedOnce = true;
      };

      eventSource.onmessage = (event) => {
        const eventType = parseSupplierWsEventType(event.data);
        handleEvent(eventType, event.data);
      };

      // Also register named SSE listeners for all requested event types
      for (const type of eventTypes) {
        eventSource.addEventListener(type, (event: MessageEvent) => {
          handleEvent(type, event.data);
        });
      }

      eventSource.onerror = () => {
        // EventSource handles automatic reconnection natively using the server's retry: directive
      };
    } catch (err) {
      console.warn("Failed to initialize supplier SSE stream", err);
    }

    return () => {
      cancelled = true;
      clearTimers();
      closeStream();
    };
  }, [debounceMs, enabled, eventTypes]);
}

/** Alias for useSupplierWsRefresh during migration */
export const useSupplierEvents = useSupplierWsRefresh;
