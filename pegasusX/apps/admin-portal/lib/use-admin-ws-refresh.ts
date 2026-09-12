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
    let attempts = 0;

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
        const url = `${backendWsURL()}?token=${encodeURIComponent(session.token)}`;
        socket = new WebSocket(url);
        socket.onmessage = (ev) => {
          try {
            const msg = JSON.parse(String(ev.data)) as { type?: string; action?: string };
            if (msg.type === "PLATFORM_ADMIN_AUDIT") {
              onSignalRef.current(msg.type, msg.action || "");
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
          attempts += 1;
          const delay = Math.min(10_000, 1000 * 2 ** Math.min(attempts, 4));
          reconnectTimer = window.setTimeout(() => void connect(), delay);
        };
      } catch {
        if (cancelled) return;
        attempts += 1;
        const delay = Math.min(10_000, 1000 * 2 ** Math.min(attempts, 4));
        reconnectTimer = window.setTimeout(() => void connect(), delay);
      }
    };

    void connect();
    return () => {
      cancelled = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      closeSocket();
    };
  }, [token, enabled]);
}
