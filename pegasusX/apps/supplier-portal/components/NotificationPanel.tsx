'use client';

import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';
import { motion, AnimatePresence } from 'framer-motion';
import Icon from './Icon';
import { HandoffInboxCard } from './HandoffInboxCard';

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
  ORDER_DISPATCHED: 'dispatch',
  ORDER_REASSIGNED: 'dispatch',
  DRIVER_ARRIVED: 'pin',
  ORDER_STATUS_CHANGED: 'refresh',
  PAYLOAD_READY_TO_SEAL: 'inventory',
  PAYLOAD_SEALED: 'verified',
  PAYMENT_SETTLED: 'payment',
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

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          ref={panelRef}
          initial={{ opacity: 0, y: -20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -20, scale: 0.95 }}
          transition={{ type: "spring", stiffness: 300, damping: 25 }}
          className="absolute right-0 top-14 w-95 md:w-105 max-h-140 desk-inspector z-100"
        >
          {/* Header */}
          <div className="desk-inspector-header px-4 py-3">
            <div className="flex items-center gap-3">
              <h3 className="md-typescale-title-medium" style={{ color: 'var(--desk-text-primary)' }}>
                Notifications
              </h3>
              {unreadCount > 0 && (
                <span
                  className="inline-flex min-w-5 h-5 items-center justify-center px-1.5 rounded-full text-[10px] font-semibold"
                  style={{ background: 'var(--desk-danger)', color: 'var(--desk-accent-on)' }}
                >
                  {unreadCount}
                </span>
              )}
            </div>
            {unreadCount > 0 && (
              <button
                className="desk-btn-ghost px-2 py-1 text-xs"
                onClick={onMarkAllRead}
              >
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="overflow-y-auto flex-1">
            {items.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 px-8 text-center">
                <div
                  className="w-12 h-12 rounded-full flex items-center justify-center mb-3"
                  style={{ background: 'var(--desk-surface-subtle)', color: 'var(--desk-text-tertiary)' }}
                >
                  <Icon name="notifications" size={20} />
                </div>
                <p className="md-typescale-body-large" style={{ color: 'var(--desk-text-secondary)' }}>
                  All caught up!
                </p>
                <p className="md-typescale-body-small mt-1" style={{ color: 'var(--desk-text-tertiary)' }}>
                  No new notifications at this time.
                </p>
              </div>
            ) : (
              <div>
                {items.map((n, i) => (
                  <motion.div
                    key={n.id}
                    initial={{ opacity: 0, x: 10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.05 }}
                    style={{ borderBottom: '1px solid var(--desk-border)' }}
                  >
                    <button
                      className="w-full text-left px-4 py-3 flex gap-3 items-start transition-colors relative"
                      style={{ background: !n.read_at ? 'var(--desk-surface-subtle)' : 'var(--desk-surface)' }}
                      onClick={() => {
                        if (!n.read_at) onMarkRead(n.id);
                      }}
                    >
                      {/* Unread indicator */}
                      {!n.read_at && (
                        <div className="absolute left-0 top-1.5 bottom-1.5 w-1 rounded-r" style={{ background: 'var(--desk-accent)' }} />
                      )}

                      <div
                        className="p-2 rounded-lg shrink-0"
                        style={{
                          background: !n.read_at ? 'var(--desk-accent-soft)' : 'var(--desk-surface-subtle)',
                          color: !n.read_at ? 'var(--desk-accent-strong)' : 'var(--desk-text-tertiary)',
                        }}
                      >
                        <Icon name={typeIcons[n.type] || 'notifications'} size={18} />
                      </div>

                      <div className="flex-1 min-w-0 pt-0.5">
                        <div className="flex items-center justify-between gap-3 mb-1.5">
                          <span
                            className={`truncate tracking-tight ${!n.read_at ? 'font-semibold' : 'font-medium'}`}
                            style={{ color: !n.read_at ? 'var(--desk-text-primary)' : 'var(--desk-text-secondary)' }}
                          >
                            {n.title}
                          </span>
                          <span className="text-[10px] font-semibold uppercase tracking-[0.08em] whitespace-nowrap" style={{ color: 'var(--desk-text-tertiary)' }}>
                            {timeAgo(n.created_at)}
                          </span>
                        </div>
                        <p
                          className="text-xs leading-relaxed line-clamp-2"
                          style={{ color: !n.read_at ? 'var(--desk-text-secondary)' : 'var(--desk-text-tertiary)' }}
                        >
                          {n.body}
                        </p>
                        {n.handoff_metadata ? (
                          <div onClick={(e) => e.stopPropagation()} role="presentation">
                            <HandoffInboxCard handoff={n.handoff_metadata} />
                          </div>
                        ) : null}
                      </div>

                      {!n.read_at && (
                        <div className="w-2 h-2 rounded-full mt-1.5" style={{ background: 'var(--desk-accent)' }} />
                      )}
                    </button>
                  </motion.div>
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          {items.length > 0 && (
            <div className="p-3 text-center" style={{ borderTop: '1px solid var(--desk-border)', background: 'var(--desk-surface)' }}>
              <button
                className="desk-btn-secondary w-full justify-center"
                onClick={onClose}
              >
                Close Panel
              </button>
            </div>
          )}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
