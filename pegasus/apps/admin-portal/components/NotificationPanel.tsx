'use client';

import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';
import { motion, AnimatePresence } from 'framer-motion';
import { Button } from '@heroui/react';

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

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          ref={panelRef}
          initial={{ opacity: 0, y: -20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -20, scale: 0.95 }}
          transition={{ type: "spring", stiffness: 300, damping: 25 }}
          className="absolute right-0 top-14 w-[420px] max-h-[600px] glass-premium rounded-3xl z-[100] flex flex-col overflow-hidden shadow-2xl"
        >
          {/* Header */}
          <div className="px-8 py-6 flex items-center justify-between border-b border-white/10 bg-white/[0.03]">
            <div className="flex items-center gap-3">
              <h3 className="text-xl font-bold text-white tracking-tight">
                Notifications
              </h3>
              {unreadCount > 0 && (
                <span className="bg-desk-accent text-white text-[10px] font-black px-2 py-0.5 rounded-full shadow-lg shadow-desk-accent/40">
                  {unreadCount}
                </span>
              )}
            </div>
            {unreadCount > 0 && (
              <button
                className="text-desk-accent text-sm font-bold hover:underline active-press"
                onClick={onMarkAllRead}
              >
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="overflow-y-auto flex-1 custom-scrollbar">
            {items.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-24 px-10 text-center">
                <div className="w-16 h-16 rounded-full bg-white/5 flex items-center justify-center mb-4 animate-float">
                  <span className="material-symbols-outlined text-3xl text-white/20">
                    notifications_off
                  </span>
                </div>
                <p className="text-lg font-medium text-white/40">
                  All caught up!
                </p>
                <p className="text-sm text-white/20 mt-1">
                  No new notifications at this time.
                </p>
              </div>
            ) : (
              <div className="divide-y divide-white/5">
                {items.map((n, i) => (
                  <motion.div
                    key={n.id}
                    initial={{ opacity: 0, x: 10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.05 }}
                  >
                    <button
                      className={`w-full text-left px-8 py-6 flex gap-5 items-start transition-all hover:bg-white/[0.04] active:bg-white/[0.06] relative group ${
                        !n.read_at ? 'bg-white/[0.02]' : ''
                      } hover-lift`}
                      onClick={() => {
                        if (!n.read_at) onMarkRead(n.id);
                      }}
                    >
                      {/* Unread indicator */}
                      {!n.read_at && (
                        <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1.5 h-12 bg-desk-accent rounded-r-full shadow-[0_0_12px_rgba(var(--desk-accent-rgb),0.5)]" />
                      )}

                      <div className={`p-3 rounded-2xl flex-shrink-0 shadow-inner ${
                        !n.read_at ? 'bg-desk-accent/10 text-desk-accent' : 'bg-white/5 text-white/30'
                      }`}>
                        <span className="material-symbols-outlined text-2xl">
                          {typeIcons[n.type] || 'notifications'}
                        </span>
                      </div>

                      <div className="flex-1 min-w-0 pt-0.5">
                        <div className="flex items-center justify-between gap-3 mb-1.5">
                          <span className={`text-base truncate tracking-tight ${
                            !n.read_at ? 'font-bold text-white' : 'font-medium text-white/50'
                          }`}>
                            {n.title}
                          </span>
                          <span className="text-[11px] font-bold text-white/20 uppercase tracking-widest whitespace-nowrap">
                            {timeAgo(n.created_at)}
                          </span>
                        </div>
                        <p className={`text-sm leading-relaxed line-clamp-2 ${
                          !n.read_at ? 'text-white/70' : 'text-white/30'
                        }`}>
                          {n.body}
                        </p>
                      </div>

                      {!n.read_at && (
                        <div className="w-2.5 h-2.5 rounded-full bg-desk-accent mt-2 animate-glow" />
                      )}
                    </button>
                  </motion.div>
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          {items.length > 0 && (
            <div className="p-4 border-t border-white/10 bg-white/[0.03] text-center">
              <button
                className="w-full py-3 rounded-xl text-sm font-bold text-white/40 hover:text-white hover:bg-white/5 transition-all active-press"
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
