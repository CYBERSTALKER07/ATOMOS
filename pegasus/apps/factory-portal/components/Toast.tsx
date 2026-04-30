'use client';

import { createContext, useContext, useState, useCallback } from 'react';

type ToastType = 'info' | 'success' | 'error' | 'warning';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType) => void;
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

let toastId = 0;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const toast = useCallback((message: string, type: ToastType = 'info') => {
    const id = ++toastId;
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, 4000);
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  const typeStyles: Record<ToastType, { bg: string; fg: string }> = {
    info:    { bg: 'var(--foreground)', fg: 'var(--background)' },
    success: { bg: 'var(--success)', fg: 'var(--success-foreground)' },
    error:   { bg: 'var(--danger)', fg: 'var(--danger-foreground)' },
    warning: { bg: 'var(--warning)', fg: 'var(--warning-foreground)' },
  };

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-300 flex flex-col-reverse gap-2 pointer-events-none">
        {toasts.map(t => {
          const s = typeStyles[t.type];
          return (
            <div
              key={t.id}
              className="md-snackbar pointer-events-auto md-animate-in"
              style={{ background: s.bg, color: s.fg }}
              role="alert"
            >
              <span className="flex-1">{t.message}</span>
              <button
                onClick={() => dismiss(t.id)}
                className="ml-1 opacity-70 hover:opacity-100"
                style={{ color: s.fg }}
                aria-label="Dismiss"
              >
                ✕
              </button>
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}
