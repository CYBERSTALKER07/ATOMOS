"use client";

import { useEffect, useRef } from "react";
import { api } from "@/lib/api";

function backendWsURL(): string {
  const base = (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080").replace(/\/$/, "");
  if (base.startsWith("https://")) return base.replace(/^https/, "wss") + "/v1/ws";
  if (base.startsWith("http://")) return base.replace(/^http/, "ws") + "/v1/ws";
  return "ws://localhost:8080/v1/ws";
}

/**
 * Opens a PLATFORM_ADMIN `/v1/ws` session and invokes onSignal for governance events.
 * Polling/manual refresh remain valid — this accelerates multi-admin consoles.
 */
export function useAdminWsRefresh(token: string, onSignal: (eventType: string, action: string) => void, enabled = true) {
  const onSignalRef = useRef(onSignal);
  onSignalRef.current = onSignal;

  useEffect(() => {
    if (!enabled || !token) return;

    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let signalDebounceTimer: number | undefined;
    let attempts = 0;
    let lastEventId: string | null = null;

    const jitteredDelay = (attempt: number) => {
      const exp = Math.min(10_000, 1_000 * 2 ** Math.min(attempt, 4));
      return exp + Math.floor(Math.random() * (exp / 2 + 1));
    };

    const closeSocket = () => {
      if (!socket) return;
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      socket.close();
      socket = null;
    };

    const connect = async () => {
      if (cancelled) return;
      try {
        const session = await api.wsSession(token);
        if (cancelled) return;
        const query = new URLSearchParams({ token: session.token });
        if (lastEventId) {
          query.set("last_event_id", lastEventId);
        }
        const url = `${backendWsURL()}?${query.toString()}`;
        socket = new WebSocket(url);
        socket.onmessage = (ev) => {
          try {
            const msg = JSON.parse(String(ev.data)) as { type?: string; action?: string; event_id?: string; id?: string };
            if (msg.event_id) {
              lastEventId = msg.event_id;
            } else if (msg.id) {
              lastEventId = msg.id;
            }
            if (msg.type === "PLATFORM_ADMIN_AUDIT") {
              const eventType = msg.type;
              const action = msg.action || "";
              if (signalDebounceTimer !== undefined) {
                window.clearTimeout(signalDebounceTimer);
              }
              signalDebounceTimer = window.setTimeout(() => {
                if (!cancelled) {
                  onSignalRef.current(eventType, action);
                }
              }, 150);
            }
          } catch {
            /* ignore non-json */
          }
        };
        socket.onopen = () => {
          attempts = 0;
        };
        socket.onclose = () => {
          if (cancelled) return;
          const delay = jitteredDelay(attempts);
          attempts += 1;
          reconnectTimer = window.setTimeout(() => void connect(), delay);
        };
      } catch {
        if (cancelled) return;
        const delay = jitteredDelay(attempts);
        attempts += 1;
        reconnectTimer = window.setTimeout(() => void connect(), delay);
      }
    };

    void connect();
    return () => {
      cancelled = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      if (signalDebounceTimer !== undefined) window.clearTimeout(signalDebounceTimer);
      closeSocket();
    };
  }, [token, enabled]);
}
