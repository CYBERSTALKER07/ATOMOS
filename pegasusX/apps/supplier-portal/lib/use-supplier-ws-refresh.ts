"use client";

import { useEffect, useRef } from "react";
import { supplierApiBaseUrl, supplierFetch } from "@/lib/auth";
import { parseSupplierWsEventType } from "@/lib/supplier-ws-events";

interface SupplierWebSocketSessionResponse {
  token: string;
  expires_at: string;
  websocket_url: string;
}

type UseSupplierWsRefreshOptions = {
  eventTypes: ReadonlySet<string>;
  /** Debounce rapid events (e.g. GPS) before calling onSignal. */
  debounceMs?: number;
  enabled?: boolean;
};

/**
 * Opens a supplier-scoped `/v1/ws` session and invokes onSignal when matching events arrive.
 * Polling fallbacks should remain — this is an acceleration path, not sole transport.
 */
export function useSupplierWsRefresh(
  onSignal: (eventType: string) => void,
  { eventTypes, debounceMs = 500, enabled = true }: UseSupplierWsRefreshOptions,
) {
  const onSignalRef = useRef(onSignal);
  onSignalRef.current = onSignal;

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let signalTimer: number | undefined;
    let attempts = 0;

    const clearTimers = () => {
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
        signalTimer = undefined;
      }
    };

    const closeSocket = () => {
      if (!socket) {
        return;
      }
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      socket.close();
      socket = null;
    };

    const scheduleReconnect = () => {
      if (cancelled) {
        return;
      }
      closeSocket();
      attempts += 1;
      const delay = Math.min(1000 * 2 ** Math.min(attempts, 4), 10_000) + Math.floor(Math.random() * 250);
      reconnectTimer = window.setTimeout(() => {
        void connect();
      }, delay);
    };

    const connect = async () => {
      if (cancelled) {
        return;
      }
      try {
        const response = await supplierFetch("/v1/supplier/ws-session", {
          method: "GET",
          cache: "no-store",
        });
        const payload = (await response.json().catch(() => null)) as
          | SupplierWebSocketSessionResponse
          | { error?: string }
          | null;
        if (!response.ok || !payload || typeof (payload as SupplierWebSocketSessionResponse).token !== "string") {
          throw new Error("ws_session_failed");
        }

        const session = payload as SupplierWebSocketSessionResponse;
        const wsBase = supplierApiBaseUrl().replace(/^http/, "ws");
        const wsURL = session.websocket_url || `${wsBase}/v1/ws`;
        const connector = wsURL.includes("?") ? "&" : "?";
        socket = new WebSocket(`${wsURL}${connector}token=${encodeURIComponent(session.token)}`);

        socket.onopen = () => {
          attempts = 0;
        };

        socket.onmessage = (event) => {
          const eventType = parseSupplierWsEventType(event.data);
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
            onSignalRef.current(eventType);
          }, debounceMs);
        };

        socket.onclose = () => {
          scheduleReconnect();
        };

        socket.onerror = () => {
          socket?.close();
        };
      } catch {
        scheduleReconnect();
      }
    };

    void connect();

    return () => {
      cancelled = true;
      clearTimers();
      closeSocket();
    };
  }, [debounceMs, enabled, eventTypes]);
}
