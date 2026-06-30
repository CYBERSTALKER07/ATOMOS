"use client";

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

type ToastTone = "info" | "success" | "error";

type ToastMessage = {
  id: string;
  message: string;
  tone: ToastTone;
  action?: {
    label: string;
    onClick: () => void;
  };
};

type ToastContextValue = {
  push: (message: string, tone?: ToastTone, action?: ToastMessage["action"]) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [messages, setMessages] = useState<ToastMessage[]>([]);

  const push = useCallback((message: string, tone: ToastTone = "info", action?: ToastMessage["action"]) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    setMessages((current) => [...current, { id, message, tone, action }]);
    window.setTimeout(() => {
      setMessages((current) => current.filter((entry) => entry.id !== id));
    }, 5000);
  }, []);

  const value = useMemo(() => ({ push }), [push]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
        {messages.map((toast) => (
          <div
            key={toast.id}
            className="md-card px-4 py-3 md-typescale-body-medium border flex items-center justify-between gap-4"
            style={{
              borderColor:
                toast.tone === "error"
                  ? "var(--color-md-error)"
                  : toast.tone === "success"
                    ? "var(--color-md-success)"
                    : "var(--color-md-outline-variant)",
            }}
          >
            <div>{toast.message}</div>
            {toast.action ? (
              <button
                type="button"
                className="md-btn md-btn-text text-[var(--color-md-primary)] font-medium whitespace-nowrap"
                onClick={() => {
                  toast.action!.onClick();
                  setMessages((current) => current.filter((entry) => entry.id !== toast.id));
                }}
              >
                {toast.action.label}
              </button>
            ) : null}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
}
