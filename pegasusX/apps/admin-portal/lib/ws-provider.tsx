"use client";

import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import { useAdminToken } from "@/lib/session";
import { api } from "@/lib/api";

export type WSConnectionStatus = "connected" | "connecting" | "disconnected" | "reconnecting";

export interface WSEventEnvelope {
  type: string;
  room?: string;
  event_id?: string;
  last_event_id?: string;
  seq?: number;
  timestamp: string;
  payload?: Record<string, unknown>;
}

interface WSContextType {
  status: WSConnectionStatus;
  lastHeartbeat: Date | null;
  lastEventTime: Date | null;
  eventBuffer: WSEventEnvelope[];
  reconnect: () => void;
  subscribe: (handler: (event: WSEventEnvelope) => void) => () => void;
  sendMessage: (payload: Record<string, unknown>) => void;
  joinRoom: (room: string) => void;
  leaveRoom: (room: string) => void;
  ackEvent: (eventId: string) => void;
  refreshKey: number;
}

const WSContext = createContext<WSContextType>({
  status: "disconnected",
  lastHeartbeat: null,
  lastEventTime: null,
  eventBuffer: [],
  reconnect: () => {},
  subscribe: () => () => {},
  sendMessage: () => {},
  joinRoom: () => {},
  leaveRoom: () => {},
  ackEvent: () => {},
  refreshKey: 0,
});

const MAX_EVENT_BUFFER = 50;
const MAX_OUTBOUND_QUEUE = 64;
const HEARTBEAT_INTERVAL_MS = 25000;
const CONTROL_FRAME_TYPES = new Set([
  "pong",
  "ack:confirmed",
  "presence:updated",
  "room:joined",
  "room:left",
  "replay:completed",
]);

export function AdminWebSocketProvider({ children }: { children: ReactNode }) {
  const { token } = useAdminToken();
  const [status, setStatus] = useState<WSConnectionStatus>("disconnected");
  const [lastHeartbeat, setLastHeartbeat] = useState<Date | null>(null);
  const [lastEventTime, setLastEventTime] = useState<Date | null>(null);
  const [eventBuffer, setEventBuffer] = useState<WSEventEnvelope[]>([]);
  const [refreshKey, setRefreshKey] = useState(0);

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const heartbeatIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const backoffRef = useRef(1000);
  const lastEventIdRef = useRef<string | null>(null);
  const outboundQueueRef = useRef<string[]>([]);
  const handlersRef = useRef<Set<(event: WSEventEnvelope) => void>>(new Set());
  const lastInvalidationRef = useRef<Record<string, number>>({});

  const subscribe = useCallback((handler: (event: WSEventEnvelope) => void) => {
    handlersRef.current.add(handler);
    return () => {
      handlersRef.current.delete(handler);
    };
  }, []);

  const sendMessage = useCallback((payload: Record<string, unknown>) => {
    const serialized = JSON.stringify(payload);
    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      socketRef.current.send(serialized);
      return;
    }
    if (outboundQueueRef.current.length >= MAX_OUTBOUND_QUEUE) {
      outboundQueueRef.current.shift();
    }
    outboundQueueRef.current.push(serialized);
  }, []);

  const joinRoom = useCallback(
    (room: string) => {
      sendMessage({ type: "room:join", room });
    },
    [sendMessage],
  );

  const leaveRoom = useCallback(
    (room: string) => {
      sendMessage({ type: "room:leave", room });
    },
    [sendMessage],
  );

  const ackEvent = useCallback(
    (eventId: string) => {
      if (!eventId.trim()) return;
      sendMessage({ type: "ack", event_id: eventId.trim() });
    },
    [sendMessage],
  );

  const clearHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
  }, []);

  const connect = useCallback(async () => {
    if (!token) {
      clearHeartbeat();
      if (socketRef.current) {
        socketRef.current.close();
        socketRef.current = null;
      }
      setStatus("disconnected");
      return;
    }

    if (socketRef.current && (socketRef.current.readyState === WebSocket.OPEN || socketRef.current.readyState === WebSocket.CONNECTING)) {
      return;
    }

    setStatus("connecting");

    try {
      const session = await api.wsSession(token);
      const wsProto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const host = (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080")
        .replace(/^https?:\/\//, "")
        .replace(/\/$/, "");
      const lastIdParam = lastEventIdRef.current
        ? `&last_event_id=${encodeURIComponent(lastEventIdRef.current)}`
        : "";
      const wsUrl = `${wsProto}//${host}/v1/ws?token=${encodeURIComponent(session.token)}${lastIdParam}`;

      const ws = new WebSocket(wsUrl);
      socketRef.current = ws;

      ws.onopen = () => {
        setStatus("connected");
        backoffRef.current = 1000;
        setLastHeartbeat(new Date());

        while (outboundQueueRef.current.length > 0 && ws.readyState === WebSocket.OPEN) {
          const queued = outboundQueueRef.current.shift();
          if (queued) {
            ws.send(queued);
          }
        }

        clearHeartbeat();
        heartbeatIntervalRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping", id: `hb-${Date.now()}` }));
          }
        }, HEARTBEAT_INTERVAL_MS);
      };

      ws.onmessage = (event) => {
        try {
          const raw = JSON.parse(event.data) as Record<string, unknown>;
          const rawType = String(raw.type || "UNKNOWN");
          const now = new Date();
          setLastHeartbeat(now);

          if (CONTROL_FRAME_TYPES.has(rawType.toLowerCase())) {
            return;
          }

          if (rawType === "WS_RESYNC_REQUIRED") {
            setRefreshKey((k) => k + 1);
            return;
          }

          const eventId = raw.event_id
            ? String(raw.event_id).trim()
            : raw.id
              ? String(raw.id).trim()
              : undefined;
          if (eventId) {
            lastEventIdRef.current = eventId;
          }

          const envelope: WSEventEnvelope = {
            type: rawType,
            room: raw.room ? String(raw.room) : undefined,
            event_id: eventId,
            seq: typeof raw.seq === "number" ? raw.seq : undefined,
            timestamp: String(raw.timestamp || now.toISOString()),
            payload: raw.payload as Record<string, unknown> | undefined,
          };

          setLastEventTime(now);

          // Append to bounded FIFO buffer
          setEventBuffer((prev) => {
            const next = [envelope, ...prev];
            return next.slice(0, MAX_EVENT_BUFFER);
          });

          // Dispatch to specific subscribers
          handlersRef.current.forEach((handler) => {
            try {
              handler(envelope);
            } catch {
              // Ignore subscriber errors
            }
          });

          // Throttled refreshKey invalidation (minimum 5s cooldown per event type) (CG-6)
          const nowMs = Date.now();
          const lastInv = lastInvalidationRef.current[envelope.type] || 0;
          if (nowMs - lastInv >= 5000) {
            lastInvalidationRef.current[envelope.type] = nowMs;
            setRefreshKey((k) => k + 1);
          }
        } catch {
          // Non-JSON ping
          setLastHeartbeat(new Date());
        }
      };

      ws.onerror = () => {
        // Will close and trigger onclose
      };

      ws.onclose = () => {
        clearHeartbeat();
        socketRef.current = null;
        setStatus("disconnected");

        // Exponential backoff with random jitter (base 1s, ceiling 30s) (CG-9)
        const jitter = Math.random() * 500;
        const delay = Math.min(backoffRef.current + jitter, 30000);
        backoffRef.current = Math.min(backoffRef.current * 1.5, 30000);

        if (token) {
          setStatus("reconnecting");
          reconnectTimeoutRef.current = setTimeout(() => {
            void connect();
          }, delay);
        }
      };
    } catch {
      clearHeartbeat();
      setStatus("disconnected");
      reconnectTimeoutRef.current = setTimeout(() => {
        void connect();
      }, 5000);
    }
  }, [token, clearHeartbeat]);

  const reconnect = useCallback(() => {
    clearHeartbeat();
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }
    backoffRef.current = 1000;
    void connect();
  }, [connect, clearHeartbeat]);

  useEffect(() => {
    void connect();
    const onOnline = () => {
      backoffRef.current = 1000;
      void connect();
    };
    const onOffline = () => {
      clearHeartbeat();
      setStatus("disconnected");
    };
    if (typeof window !== "undefined") {
      window.addEventListener("online", onOnline);
      window.addEventListener("offline", onOffline);
    }
    return () => {
      clearHeartbeat();
      if (typeof window !== "undefined") {
        window.removeEventListener("online", onOnline);
        window.removeEventListener("offline", onOffline);
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (socketRef.current) {
        socketRef.current.close();
        socketRef.current = null;
      }
    };
  }, [connect, clearHeartbeat]);

  return (
    <WSContext.Provider
      value={{
        status,
        lastHeartbeat,
        lastEventTime,
        eventBuffer,
        reconnect,
        subscribe,
        sendMessage,
        joinRoom,
        leaveRoom,
        ackEvent,
        refreshKey,
      }}
    >
      {children}
    </WSContext.Provider>
  );
}

export function useAdminWS() {
  return useContext(WSContext);
}
