'use client';

import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';

type ToastType = 'info' | 'success' | 'error' | 'warning';

interface ToastAction {
  label: string;
  onClick: () => void;
}

interface Toast {
  id: number;
  message: string;
  type: ToastType;
  action?: ToastAction;
  durationMs: number;
}

interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: ToastAction, durationMs?: number) => void;
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

let toastId = 0;

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function ToastCard({
  toast,
  dismiss,
  typeStyle,
}: {
  toast: Toast;
  dismiss: (id: number) => void;
  typeStyle: { bg: string; fg: string; track: string };
}) {
  const [offsetX, setOffsetX] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [progressActive, setProgressActive] = useState(false);
  const pointerRef = useRef<{ pointerId: number; startX: number; startOffsetX: number } | null>(null);

  useEffect(() => {
    const frame = requestAnimationFrame(() => setProgressActive(true));
    return () => cancelAnimationFrame(frame);
  }, []);

  const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    pointerRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startOffsetX: offsetX,
    };
    setIsDragging(true);
    event.currentTarget.setPointerCapture(event.pointerId);
  };

  const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!pointerRef.current || pointerRef.current.pointerId !== event.pointerId) {
      return;
    }
    const next = pointerRef.current.startOffsetX + (event.clientX - pointerRef.current.startX);
    setOffsetX(clamp(next, -220, 220));
  };

  const finishDrag = () => {
    const threshold = 96;
    if (Math.abs(offsetX) >= threshold) {
      dismiss(toast.id);
      return;
    }
    setOffsetX(0);
  };

  const onPointerUp = (event: React.PointerEvent<HTMLDivElement>) => {
    if (pointerRef.current && pointerRef.current.pointerId === event.pointerId) {
      pointerRef.current = null;
      setIsDragging(false);
      finishDrag();
    }
  };

  const onPointerCancel = (event: React.PointerEvent<HTMLDivElement>) => {
    if (pointerRef.current && pointerRef.current.pointerId === event.pointerId) {
      pointerRef.current = null;
      setIsDragging(false);
      setOffsetX(0);
    }
  };

  const dragOpacity = Math.max(0.3, 1 - Math.abs(offsetX) / 240);

  return (
    <div
      className="md-snackbar pointer-events-auto md-animate-in"
      style={{
        background: typeStyle.bg,
        color: typeStyle.fg,
        position: 'relative',
        left: 'auto',
        bottom: 'auto',
        transform: `translate3d(${offsetX}px, 0, 0) scale(${isDragging ? 0.988 : 1})`,
        opacity: dragOpacity,
        transition: isDragging
          ? 'none'
          : 'transform 220ms cubic-bezier(0.2, 0.8, 0.2, 1), opacity 220ms cubic-bezier(0.2, 0.8, 0.2, 1)',
        border: '1px solid rgba(255, 255, 255, 0.18)',
        cursor: 'grab',
      }}
      role="alert"
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerCancel}
      title="Swipe left or right to dismiss"
    >
      <span className="flex-1" style={{ paddingRight: 8 }}>{toast.message}</span>
      {toast.action && (
        <button
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation();
            toast.action?.onClick();
            dismiss(toast.id);
          }}
          className="md-typescale-label-large font-medium"
          style={{ color: typeStyle.fg, opacity: 0.92 }}
        >
          {toast.action.label}
        </button>
      )}
      <button
        onPointerDown={(event) => event.stopPropagation()}
        onClick={(event) => {
          event.stopPropagation();
          dismiss(toast.id);
        }}
        className="ml-1 opacity-70 hover:opacity-100"
        style={{ color: typeStyle.fg }}
        aria-label="Dismiss"
      >
        x
      </button>
      <div
        style={{
          position: 'absolute',
          left: 12,
          right: 12,
          bottom: 6,
          height: 2,
          borderRadius: 999,
          background: typeStyle.track,
          overflow: 'hidden',
          pointerEvents: 'none',
        }}
      >
        <div
          style={{
            height: '100%',
            borderRadius: 999,
            background: typeStyle.fg,
            opacity: 0.48,
            transformOrigin: 'left center',
            transform: `scaleX(${progressActive ? 0 : 1})`,
            transition: `transform ${toast.durationMs}ms linear`,
          }}
        />
      </div>
    </div>
  );
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timersRef = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  const clearTimer = useCallback((id: number) => {
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
  }, []);

  const dismiss = useCallback((id: number) => {
    clearTimer(id);
    setToasts((prev) => prev.filter((item) => item.id !== id));
  }, [clearTimer]);

  const toast = useCallback((message: string, type: ToastType = 'info', action?: ToastAction, durationMs = 4000) => {
    const id = ++toastId;
    setToasts((prev) => [...prev, { id, message, type, action, durationMs }]);
    const timer = setTimeout(() => dismiss(id), durationMs);
    timersRef.current.set(id, timer);
  }, [dismiss]);

  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();
    };
  }, []);

  const typeStyles: Record<ToastType, { bg: string; fg: string; track: string }> = {
    info: { bg: 'var(--foreground)', fg: 'var(--background)', track: 'rgba(255,255,255,0.22)' },
    success: { bg: 'var(--success)', fg: 'var(--success-foreground)', track: 'rgba(255,255,255,0.24)' },
    error: { bg: 'var(--danger)', fg: 'var(--danger-foreground)', track: 'rgba(255,255,255,0.24)' },
    warning: { bg: 'var(--warning)', fg: 'var(--warning-foreground)', track: 'rgba(0,0,0,0.12)' },
  };

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-300 flex flex-col-reverse gap-2 pointer-events-none">
        {toasts.map((item) => {
          const style = typeStyles[item.type];
          return (
            <ToastCard key={item.id} toast={item} dismiss={dismiss} typeStyle={style} />
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}
