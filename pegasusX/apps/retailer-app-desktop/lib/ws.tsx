'use client';

import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import type { WSEventMessage, WSEventPayloadMap, WSEventTypeValue } from '@pegasusx/types';
import { readToken } from './auth';

export type WsEventType = WSEventTypeValue;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WsMessage = (Record<string, any> & { type?: string });

type WsEventPayload<T extends string> = T extends WsEventType
  ? WSEventMessage<T>
  : WsMessage;

type WsEventHandler<T extends string = string> = (msg: WsEventPayload<T>) => void;

type WebSocketContextType = {
  isConnected: boolean;
  lastMessage: WsMessage | null;
  sendMessage: (msg: WsMessage) => void;
  subscribe: <T extends string>(type: T, handler: WsEventHandler<T>) => () => void;
};

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WsMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Map<string, Set<WsEventHandler>>>(new Map());
  
  const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8180/v1/ws';

  const subscribe = useCallback(<T extends string>(type: T, handler: WsEventHandler<T>) => {
    let set = listenersRef.current.get(type);
    if (!set) { set = new Set(); listenersRef.current.set(type, set); }
    set.add(handler as WsEventHandler);
    return () => { set!.delete(handler as WsEventHandler); };
  }, []);

  useEffect(() => {
    let reconnectTimer: NodeJS.Timeout | null = null;
    let disposed = false;
    
    function connect() {
      if (disposed) return;
      const token = readToken();
      if (!token) return;

      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return;

      const ws = new WebSocket(`${WS_URL}?token=${token}`);
      
      ws.onopen = () => {
        setIsConnected(true);
        console.log('Retailer WS connected');
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as WsMessage;
          const type = typeof msg.type === 'string' ? msg.type : undefined;
          setLastMessage(msg);
          if (type) {
            listenersRef.current.get(type)?.forEach(h => h(msg));
          }
        } catch (e) {
          console.error('WS parse error', e);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        wsRef.current = null;
        if (!disposed) {
          reconnectTimer = setTimeout(connect, 3000);
        }
      };

      ws.onerror = (err) => {
        console.error('WS error', err);
        ws.close();
      };
      
      wsRef.current = ws;
    }

    connect();

    return () => {
      disposed = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
      wsRef.current = null;
    };
  }, [WS_URL]);

  const sendMessage = (msg: WsMessage) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg));
    }
  };

  return (
    <WebSocketContext.Provider value={{ isConnected, lastMessage, sendMessage, subscribe }}>
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

/** Strictly typed helper for generated Pegasus event contracts. */
export function usePegasusEvent<T extends keyof WSEventPayloadMap>(
  type: T,
  callback: (payload: WSEventPayloadMap[T]) => void,
) {
  useWsEvent(type, useCallback((msg: WsEventPayload<T>) => {
    callback(msg as WSEventPayloadMap[T]);
  }, [callback]));
}
