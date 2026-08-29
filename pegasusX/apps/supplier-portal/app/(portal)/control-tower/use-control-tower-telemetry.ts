"use client";

import { useState, useEffect } from "react";
import { supplierApiBaseUrl } from "@/lib/auth";
import { runSupplierSessionReconcile } from "@/lib/session-reconcile";
import { SSE_SUPPLIER_ENDPOINT } from "@pegasusx/ws-refresh-contract";

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
    if (!sid || typeof window === "undefined") {
      setNetworkNodes([]);
      setNetworkLinks([]);
      setH3Data([]);
      return;
    }

    let cancelled = false;
    let eventSource: EventSource | null = null;
    let hasConnectedOnce = false;

    const closeStream = () => {
      if (!eventSource) return;
      eventSource.onopen = null;
      eventSource.onmessage = null;
      eventSource.onerror = null;
      eventSource.close();
      eventSource = null;
    };

    const handlePayload = (raw: any) => {
      if (!raw || typeof raw !== "object") return;
      if (raw.type === "control_tower_network") {
        setNetworkNodes(raw.nodes || []);
        setNetworkLinks(raw.links || []);
      } else if (raw.type === "control_tower_h3") {
        setH3Data(raw.data || []);
      }
    };

    try {
      const sseUrl = `${supplierApiBaseUrl()}${SSE_SUPPLIER_ENDPOINT}`;
      eventSource = new EventSource(sseUrl, { withCredentials: true });

      eventSource.onopen = () => {
        if (hasConnectedOnce) {
          void runSupplierSessionReconcile();
        }
        hasConnectedOnce = true;
      };

      eventSource.onmessage = (event) => {
        try {
          const raw = JSON.parse(event.data);
          handlePayload(raw);
        } catch (err) {
          console.error("Failed to parse telemetry SSE payload", err);
        }
      };

      eventSource.addEventListener("control_tower_network", (event: MessageEvent) => {
        try {
          const raw = JSON.parse(event.data);
          handlePayload({ type: "control_tower_network", ...raw });
        } catch { /* ignore */ }
      });

      eventSource.addEventListener("control_tower_h3", (event: MessageEvent) => {
        try {
          const raw = JSON.parse(event.data);
          handlePayload({ type: "control_tower_h3", ...raw });
        } catch { /* ignore */ }
      });

      eventSource.onerror = () => {
        // EventSource handles reconnection natively
      };
    } catch (err) {
      console.warn("Failed to initialize telemetry SSE stream", err);
    }

    return () => {
      cancelled = true;
      closeStream();
    };
  }, [supplierId]);

  return { networkNodes, networkLinks, h3Data };
}
