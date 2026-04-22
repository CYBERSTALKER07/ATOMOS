'use client';

import { createContext, useContext, useState, useCallback } from 'react';

type ToastType = 'info' | 'success' | 'error' | 'warning';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
  action?: { label: string; onClick: () => void };
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: Toast['action']) => void;
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

let toastId = 0;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const toast = useCallback((message: string, type: ToastType = 'info', action?: Toast['action']) => {
    const id = ++toastId;
    setToasts(prev => [...prev, { id, message, type, action }]);
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
      {/* Toast stack — bottom center */}
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-300 flex flex-col-reverse gap-2 pointer-events-none">
        {toasts.map(t => {
          const s = typeStyles[t.type];
          return (
            <div
              key={t.id}
              className="md-snackbar pointer-events-auto md-animate-in"
              style={{ background: s.bg, color: s.fg, position: 'relative', transform: 'none', left: 'auto', bottom: 'auto' }}
              role="alert"
            >
              <span className="flex-1">{t.message}</span>
              {t.action && (
                <button
                  onClick={() => { t.action?.onClick(); dismiss(t.id); }}
                  className="md-typescale-label-large font-medium"
                  style={{ color: t.type === 'info' ? 'var(--accent-foreground)' : s.fg, opacity: 0.9 }}
                >
                  {t.action.label}
                </button>
              )}
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
