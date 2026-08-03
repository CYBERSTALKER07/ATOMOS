import { useState, useEffect, useRef } from "react";
import { NetworkNode, NetworkLink } from "./LiveEKGNetworkGraph";

interface H3Density {
  hex: string;
  count: number;
}

interface ControlTowerData {
  networkNodes: NetworkNode[];
  networkLinks: NetworkLink[];
  h3Data: H3Density[];
}

export function useControlTowerWebSocket(supplierId: string): ControlTowerData {
  const [networkNodes, setNetworkNodes] = useState<NetworkNode[]>([]);
  const [networkLinks, setNetworkLinks] = useState<NetworkLink[]>([]);
  const [h3Data, setH3Data] = useState<H3Density[]>([]);

  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    const sid = supplierId.trim();
    if (!sid) {
      setNetworkNodes([]);
      setNetworkLinks([]);
      setH3Data([]);
      return;
    }

    // In a real app, this URL would come from env config
    const wsUrl = `ws://localhost:8080/ws/telemetry?identity=supplier:${encodeURIComponent(sid)}`;

    ws.current = new WebSocket(wsUrl);

    ws.current.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload.type === "control_tower_network") {
          setNetworkNodes(payload.nodes || []);
          setNetworkLinks(payload.links || []);
        } else if (payload.type === "control_tower_h3") {
          setH3Data(payload.data || []);
        }
      } catch (err) {
        console.error("Failed to parse telemetry websocket payload", err);
      }
    };

    return () => {
      if (ws.current) {
        ws.current.close();
        ws.current = null;
      }
    };
  }, [supplierId]);

  return { networkNodes, networkLinks, h3Data };
}
