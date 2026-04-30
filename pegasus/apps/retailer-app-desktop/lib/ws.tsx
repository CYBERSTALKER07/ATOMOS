'use client';

import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import { readToken } from './auth';

export type WsMessage = Record<string, unknown>;
type WsEventHandler = (msg: WsMessage) => void;

type WebSocketContextType = {
  isConnected: boolean;
  lastMessage: WsMessage | null;
  sendMessage: (msg: WsMessage) => void;
  subscribe: (type: string, handler: WsEventHandler) => () => void;
};

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WsMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Map<string, Set<WsEventHandler>>>(new Map());
  
  const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/v1/ws/retailer';

  const subscribe = useCallback((type: string, handler: WsEventHandler) => {
    let set = listenersRef.current.get(type);
    if (!set) { set = new Set(); listenersRef.current.set(type, set); }
    set.add(handler);
    return () => { set!.delete(handler); };
  }, []);

  useEffect(() => {
    let reconnectTimer: NodeJS.Timeout;
    
    function connect() {
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
          setLastMessage(msg);
          const type = msg.type as string | undefined;
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
        reconnectTimer = setTimeout(connect, 3000);
      };

      ws.onerror = (err) => {
        console.error('WS error', err);
        ws.close();
      };
      
      wsRef.current = ws;
    }

    connect();

    return () => {
      clearTimeout(reconnectTimer);
      if (wsRef.current) {
        wsRef.current.close();
      }
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

/** Subscribe to a specific WS message type. Handler is called when msg.type matches. */
export function useWsEvent(type: string, handler: WsEventHandler) {
  const { subscribe } = useWebSocket();
  useEffect(() => subscribe(type, handler), [type, handler, subscribe]);
}