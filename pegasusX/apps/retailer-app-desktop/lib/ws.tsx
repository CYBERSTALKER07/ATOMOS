'use client';

import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import type { WsEvent } from '@pegasusx/types';
import { readToken } from './auth';

function reconnectDelayMs(attempt: number, baseMs = 3_000, maxMs = 60_000): number {
  const capped = Math.min(Math.max(attempt, 0), 10);
  const exp = Math.min(baseMs * 2 ** capped, maxMs);
  return exp + Math.floor(Math.random() * (exp / 2 + 1));
}

export type WsEventType = WsEvent["type"];
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WsMessage = (Record<string, any> & { type?: string });

type WsEventPayload<T extends string> = T extends WsEventType
  ? Extract<WsEvent, { type: T }>
  : WsMessage;

type WsEventHandler<T extends string = string> = (msg: WsEventPayload<T>) => void;

type WebSocketContextType = {
  isConnected: boolean;
  /** Increments after the first successful reconnect (not initial connect) or WS_RESYNC_REQUIRED. */
  reconnectEpoch: number;
  lastMessage: WsMessage | null;
  sendMessage: (msg: WsMessage) => void;
  joinRoom: (room: string) => void;
  leaveRoom: (room: string) => void;
  ackEvent: (eventId: string) => void;
  subscribe: <T extends string>(type: T, handler: WsEventHandler<T>) => () => void;
};

const MAX_OUTBOUND_QUEUE = 64;
const HEARTBEAT_INTERVAL_MS = 25_000;
const CONTROL_FRAME_TYPES = new Set([
  'pong',
  'ack:confirmed',
  'presence:updated',
  'room:joined',
  'room:left',
  'replay:completed',
]);

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [reconnectEpoch, setReconnectEpoch] = useState(0);
  const [lastMessage, setLastMessage] = useState<WsMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const outboundQueueRef = useRef<string[]>([]);
  const listenersRef = useRef<Map<string, Set<WsEventHandler>>>(new Map());
  const hasConnectedOnceRef = useRef(false);
  const lastEventIdRef = useRef<string | null>(null);
  
  const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8180/v1/ws';

  const subscribe = useCallback(<T extends string>(type: T, handler: WsEventHandler<T>) => {
    let set = listenersRef.current.get(type);
    if (!set) { set = new Set(); listenersRef.current.set(type, set); }
    set.add(handler as unknown as WsEventHandler);
    return () => { set!.delete(handler as unknown as WsEventHandler); };
  }, []);

  useEffect(() => {
    let reconnectTimer: NodeJS.Timeout | null = null;
    let heartbeatTimer: NodeJS.Timeout | null = null;
    let disposed = false;
    let reconnectAttempt = 0;

    const clearHeartbeat = () => {
      if (heartbeatTimer) {
        clearInterval(heartbeatTimer);
        heartbeatTimer = null;
      }
    };
    
    function connect() {
      if (disposed) return;
      const token = readToken();
      if (!token) return;

      if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) return;

      const lastIdParam = lastEventIdRef.current ? `&last_event_id=${encodeURIComponent(lastEventIdRef.current)}` : '';
      const ws = new WebSocket(`${WS_URL}?token=${encodeURIComponent(token)}${lastIdParam}`);
      
      ws.onopen = () => {
        reconnectAttempt = 0;
        setIsConnected(true);
        if (hasConnectedOnceRef.current) {
          setReconnectEpoch((epoch) => epoch + 1);
        } else {
          hasConnectedOnceRef.current = true;
        }
        while (outboundQueueRef.current.length > 0 && ws.readyState === WebSocket.OPEN) {
          const frame = outboundQueueRef.current.shift();
          if (frame) {
            ws.send(frame);
          }
        }
        clearHeartbeat();
        heartbeatTimer = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping', id: `hb-${Date.now()}` }));
          }
        }, HEARTBEAT_INTERVAL_MS);
        console.log('Retailer WS connected');
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as WsMessage & { event_id?: string; id?: string };
          const type = typeof msg.type === 'string' ? msg.type : undefined;
          if (type && CONTROL_FRAME_TYPES.has(type.toLowerCase())) {
            return;
          }
          if (type === 'WS_RESYNC_REQUIRED') {
            setReconnectEpoch((epoch) => epoch + 1);
            return;
          }
          const eventId = msg.event_id || msg.id;
          if (typeof eventId === 'string' && eventId.trim() !== '') {
            lastEventIdRef.current = eventId.trim();
          }
          setLastMessage(msg);
          if (type) {
            listenersRef.current.get(type)?.forEach(h => h(msg));
          }
        } catch (e) {
          console.error('WS parse error', e);
        }
      };

      ws.onclose = () => {
        clearHeartbeat();
        setIsConnected(false);
        wsRef.current = null;
        if (!disposed) {
          const delay = reconnectDelayMs(reconnectAttempt);
          reconnectAttempt += 1;
          reconnectTimer = setTimeout(connect, delay);
        }
      };

      ws.onerror = (err) => {
        console.error('WS error', err);
        ws.close();
      };
      
      wsRef.current = ws;
    }

    connect();

    const onOnline = () => {
      reconnectAttempt = 0;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      connect();
    };

    const onOffline = () => {
      clearHeartbeat();
      setIsConnected(false);
    };

    if (typeof window !== 'undefined') {
      window.addEventListener('online', onOnline);
      window.addEventListener('offline', onOffline);
    }

    return () => {
      disposed = true;
      clearHeartbeat();
      if (typeof window !== 'undefined') {
        window.removeEventListener('online', onOnline);
        window.removeEventListener('offline', onOffline);
      }
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
      wsRef.current = null;
    };
  }, [WS_URL]);

  const sendMessage = useCallback((msg: WsMessage) => {
    const serialized = JSON.stringify(msg);
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(serialized);
      return;
    }
    if (outboundQueueRef.current.length >= MAX_OUTBOUND_QUEUE) {
      outboundQueueRef.current.shift();
    }
    outboundQueueRef.current.push(serialized);
  }, []);

  const joinRoom = useCallback((room: string) => {
    sendMessage({ type: 'room:join', room });
  }, [sendMessage]);

  const leaveRoom = useCallback((room: string) => {
    sendMessage({ type: 'room:leave', room });
  }, [sendMessage]);

  const ackEvent = useCallback((eventId: string) => {
    if (!eventId.trim()) return;
    sendMessage({ type: 'ack', event_id: eventId.trim() });
  }, [sendMessage]);

  return (
    <WebSocketContext.Provider value={{ isConnected, reconnectEpoch, lastMessage, sendMessage, joinRoom, leaveRoom, ackEvent, subscribe }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const ctx = useContext(WebSocketContext);
  if (!ctx) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return ctx;
}

export function useOptionalWebSocket() {
  return useContext(WebSocketContext);
}

/** Subscribe to a specific WS message type. Handler is called when msg.type matches. */
export function useWsEvent<T extends string>(type: T, handler: WsEventHandler<T>) {
  const { subscribe } = useWebSocket();
  useEffect(() => subscribe(type, handler), [type, handler, subscribe]);
}


