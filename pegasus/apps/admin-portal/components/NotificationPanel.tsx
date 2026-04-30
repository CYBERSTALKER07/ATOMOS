'use client';

import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';

interface NotificationPanelProps {
  open: boolean;
  onClose: () => void;
  items: Notification[];
  unreadCount: number;
  onMarkRead: (id: string) => void;
  onMarkAllRead: () => void;
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const typeIcons: Record<string, string> = {
  ORDER_DISPATCHED: 'local_shipping',
  DRIVER_ARRIVED: 'place',
  ORDER_STATUS_CHANGED: 'sync_alt',
  PAYLOAD_READY_TO_SEAL: 'inventory_2',
  PAYLOAD_SEALED: 'verified',
  PAYMENT_SETTLED: 'payments',
  PAYMENT_FAILED: 'error',
};

export default function NotificationPanel({
  open,
  onClose,
  items,
  unreadCount,
  onMarkRead,
  onMarkAllRead,
}: NotificationPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      ref={panelRef}
      className="md-elevation-3 md-shape-md"
      style={{
        position: 'absolute',
        right: 0,
        top: 44,
        width: 380,
        maxHeight: 480,
        background: 'var(--color-md-surface-container)',
        zIndex: 50,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Header */}
      <div
        className="px-4 py-3 flex items-center justify-between"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <span className="md-typescale-title-small text-foreground">
          Notifications{unreadCount > 0 ? ` (${unreadCount})` : ''}
        </span>
        {unreadCount > 0 && (
          <button
            className="md-typescale-label-small cursor-pointer"
            style={{ color: 'var(--color-md-primary)', background: 'none', border: 'none' }}
            onClick={onMarkAllRead}
          >
            Mark all read
          </button>
        )}
      </div>

      {/* List */}
      <div style={{ overflowY: 'auto', flex: 1 }}>
        {items.length === 0 ? (
          <div className="px-4 py-8 text-center md-typescale-body-small text-muted">
            No notifications yet
          </div>
        ) : (
          items.map((n) => (
            <button
              key={n.id}
              className="w-full text-left px-4 py-3 flex gap-3 items-start cursor-pointer"
              style={{
                background: n.read_at ? 'transparent' : 'var(--color-md-surface-container-high, rgba(0,0,0,0.04))',
                borderBottom: '1px solid var(--border)',
                border: 'none',
                borderBlockEnd: '1px solid var(--border)',
              }}
              onClick={() => {
                if (!n.read_at) onMarkRead(n.id);
              }}
            >
              <span
                className="material-symbols-outlined"
                style={{
                  fontSize: 20,
                  color: n.read_at ? 'var(--color-md-outline)' : 'var(--color-md-primary)',
                  flexShrink: 0,
                  marginTop: 2,
                }}
              >
                {typeIcons[n.type] || 'notifications'}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span
                    className="md-typescale-label-medium truncate"
                    style={{ color: n.read_at ? 'var(--color-md-outline)' : 'var(--color-md-on-surface)' }}
                  >
                    {n.title}
                  </span>
                  <span className="md-typescale-label-small text-muted whitespace-nowrap">
                    {timeAgo(n.created_at)}
                  </span>
                </div>
                <p
                  className="md-typescale-body-small mt-0.5 line-clamp-2"
                  style={{ color: 'var(--color-md-on-surface-variant)' }}
                >
                  {n.body}
                </p>
              </div>
              {!n.read_at && (
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    background: 'var(--color-md-primary)',
                    flexShrink: 0,
                    marginTop: 6,
                  }}
                />
              )}
            </button>
          ))
        )}
      </div>
    </div>
  );
}
