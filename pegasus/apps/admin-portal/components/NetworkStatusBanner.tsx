'use client';

import { useState, useEffect, useRef } from 'react';
import { useNetworkStatus } from '@/lib/offline-queue';

export function NetworkStatusBanner() {
  const { isOnline, pendingCount } = useNetworkStatus();
  const [backpressureMs, setBackpressureMs] = useState(0);
  const backpressureTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Listen for backpressure signals from apiFetch
  useEffect(() => {
    const handler = (e: Event) => {
      const ms = (e as CustomEvent<number>).detail;
      setBackpressureMs(ms);
      if (backpressureTimerRef.current) {
        clearTimeout(backpressureTimerRef.current);
      }
      // Auto-clear after the backpressure window expires
      backpressureTimerRef.current = setTimeout(() => {
        setBackpressureMs(0);
        backpressureTimerRef.current = null;
      }, ms);
    };
    window.addEventListener('backpressure', handler);
    return () => {
      window.removeEventListener('backpressure', handler);
      if (backpressureTimerRef.current) {
        clearTimeout(backpressureTimerRef.current);
        backpressureTimerRef.current = null;
      }
    };
  }, []);

  const totalPending = pendingCount;

  // Nothing to show
  if (isOnline && totalPending === 0 && backpressureMs === 0) return null;

  // Determine banner state — priority: offline > backpressure > syncing
  let bg: string;
  let fg: string;
  let content: React.ReactNode;

  if (!isOnline) {
    bg = 'var(--color-md-error-container, #fecaca)';
    fg = 'var(--color-md-on-error-container, #991b1b)';
    content = (
      <>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <line x1="1" y1="1" x2="23" y2="23" />
          <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55" />
          <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39" />
          <path d="M10.71 5.05A16 16 0 0 1 22.56 9" />
          <path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88" />
          <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
          <line x1="12" y1="20" x2="12.01" y2="20" />
        </svg>
        <span>
          Offline — {totalPending > 0
            ? `${totalPending} change${totalPending !== 1 ? 's' : ''} queued`
            : 'changes will be queued'}
        </span>
      </>
    );
  } else if (backpressureMs > 0) {
    bg = 'var(--color-md-warning-container, #fef3c7)';
    fg = 'var(--color-md-on-warning-container, #92400e)';
    content = (
      <>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <span>Server high load — throttling requests</span>
      </>
    );
  } else {
    bg = 'var(--color-md-warning-container, #fef3c7)';
    fg = 'var(--color-md-on-warning-container, #92400e)';
    content = (
      <>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="animate-spin">
          <path d="M21 12a9 9 0 1 1-6.219-8.56" />
        </svg>
        <span>Syncing {totalPending} queued change{totalPending !== 1 ? 's' : ''}...</span>
      </>
    );
  }

  return (
    <div
      className="fixed top-0 left-0 right-0 z-50 flex items-center justify-center gap-2 px-4 py-2 md-typescale-label-medium"
      style={{ background: bg, color: fg }}
    >
      {content}
    </div>
  );
}
