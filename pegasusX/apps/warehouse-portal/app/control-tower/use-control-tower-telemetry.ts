import { useState, useEffect, useRef } from "react";
import { warehouseApiBaseUrl, apiFetch } from "@/lib/auth";
import { reconnectDelayMs, retryAfterSecondsFromResponse } from '@pegasusx/api-core';
import { runWarehouseSessionReconcile } from "@/lib/session-reconcile";
export interface NetworkNode { id: string; type: "warehouse" | "retailer" | "driver"; label: string; status: "active" | "idle" | "busy"; }
export interface NetworkLink { source: string; target: string; value: number; }

interface H3Density {
  hex: string;
  count: number;
}

interface ControlTowerData {
  networkNodes: NetworkNode[];
  networkLinks: NetworkLink[];
  h3Data: H3Density[];
}

export function useControlTowerTelemetry(supplierId: string): ControlTowerData {
  const [networkNodes, setNetworkNodes] = useState<NetworkNode[]>([]);
  const [networkLinks, setNetworkLinks] = useState<NetworkLink[]>([]);
  const [h3Data, setH3Data] = useState<H3Density[]>([]);

  useEffect(() => {
    const sid = supplierId.trim();
    if (!sid) {
      setNetworkNodes([]);
      setNetworkLinks([]);
      setH3Data([]);
      return;
    }

    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let attempts = 0;
    let pendingRetryAfterSeconds: number | undefined;
    let hasConnectedOnce = false;

    const clearTimers = () => {
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
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

    const scheduleReconnect = () => {
      if (cancelled) return;
      closeSocket();
      attempts += 1;
      const delay = reconnectDelayMs(attempts - 1, {
        baseMs: 1_000,
        maxMs: 10_000,
        retryAfterSeconds: pendingRetryAfterSeconds,
      });
      pendingRetryAfterSeconds = undefined;
      reconnectTimer = window.setTimeout(() => {
        void connect();
      }, delay);
    };

    const connect = async () => {
      if (cancelled) return;
      try {
        const response = await apiFetch("/v1/warehouse/ws-session", {
          method: "GET",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload || typeof payload.token !== "string") {
          pendingRetryAfterSeconds = retryAfterSecondsFromResponse(response);
          throw new Error("ws_session_failed");
        }

        const wsBase = warehouseApiBaseUrl().replace(/^http/, "ws");
        const wsURL = payload.websocket_url || `${wsBase}/v1/ws`;
        const connector = wsURL.includes("?") ? "&" : "?";
        socket = new WebSocket(`${wsURL}${connector}token=${encodeURIComponent(payload.token)}`);

        socket.onopen = () => {
          attempts = 0;
          if (hasConnectedOnce) {
            void runWarehouseSessionReconcile();
          }
          hasConnectedOnce = true;
        };

        socket.onmessage = (event) => {
          try {
            const raw = JSON.parse(event.data);
            if (raw.type === "control_tower_network") {
              setNetworkNodes(raw.nodes || []);
              setNetworkLinks(raw.links || []);
            } else if (raw.type === "control_tower_h3") {
              setH3Data(raw.data || []);
            }
          } catch (err) {
            console.error("Failed to parse telemetry websocket payload", err);
          }
        };

        socket.onclose = () => scheduleReconnect();
        socket.onerror = () => socket?.close();
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
  }, [supplierId]);

  return { networkNodes, networkLinks, h3Data };
}
