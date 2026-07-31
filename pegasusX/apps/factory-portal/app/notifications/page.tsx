'use client';

import { useNotifications } from '@/lib/useNotifications';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import Icon from '@/components/Icon';
import { motion } from 'framer-motion';

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
  ORDER_DISPATCHED: 'fleet',
  ORDER_REASSIGNED: 'fleet',
  DRIVER_ARRIVED: 'loadingBay',
  ORDER_STATUS_CHANGED: 'refresh',
  PAYLOAD_READY_TO_SEAL: 'manifests',
  PAYLOAD_SEALED: 'check_circle',
  MANIFEST_DISPATCHED: 'transfers',
  MANIFEST_EXCEPTION: 'warning',
  PAYMENT_SETTLED: 'analytics',
  PAYMENT_FAILED: 'error',
};

export default function NotificationInboxPage() {
  const { items, unreadCount, loading, markRead, markAllRead, fetchInbox } = useNotifications();

  return (
    <PageTransition>
      <PageChrome
        icon="notifications"
        title="Notifications Inbox"
        description="View and manage all your factory notifications and alerts."
        loading={loading}
        skeletonVariant="table"
        empty={!loading && items.length === 0}
        emptyMessage="No notifications in your inbox at this time."
        actions={
          <div className="flex gap-3">
            {unreadCount > 0 && (
              <button
                type="button"
                className="portal-btn portal-btn--secondary inline-flex items-center gap-1.5"
                onClick={() => void markAllRead()}
              >
                <Icon name="check_circle" size={16} /> Mark All Read
              </button>
            )}
            <button 
              type="button" 
              className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" 
              onClick={() => void fetchInbox()}
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
          </div>
        }
      >
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="desk-card p-0"
        >
          <div className="flex flex-col">
            {items.map((n, i) => (
              <motion.div
                key={n.id}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: i * 0.05 }}
                className={`flex gap-4 p-5 border-b border-[var(--desk-border)] last:border-b-0 transition-colors ${!n.read_at ? 'bg-[var(--desk-surface-subtle)]' : 'bg-transparent'}`}
              >
                <div
                  className="p-3 rounded-xl shrink-0 flex items-center justify-center h-12 w-12"
                  style={{
                    background: !n.read_at ? 'var(--desk-accent-soft)' : 'var(--desk-surface-subtle)',
                    color: !n.read_at ? 'var(--desk-accent-strong)' : 'var(--desk-text-tertiary)',
                  }}
                >
                  <Icon name={typeIcons[n.type] || 'notifications'} size={24} />
                </div>
                
                <div className="flex-1 min-w-0 flex flex-col justify-center">
                  <div className="flex items-center justify-between gap-3 mb-1">
                    <span
                      className={`text-base tracking-tight ${!n.read_at ? 'font-bold' : 'font-semibold'}`}
                      style={{ color: !n.read_at ? 'var(--desk-text-primary)' : 'var(--desk-text-secondary)' }}
                    >
                      {n.title}
                    </span>
                    <span className="text-xs font-semibold uppercase tracking-[0.08em] whitespace-nowrap" style={{ color: 'var(--desk-text-tertiary)' }}>
                      {timeAgo(n.created_at)}
                    </span>
                  </div>
                  <p
                    className="text-sm leading-relaxed max-w-3xl"
                    style={{ color: !n.read_at ? 'var(--desk-text-secondary)' : 'var(--desk-text-tertiary)' }}
                  >
                    {n.body}
                  </p>
                  
                  {n.payload && (
                    <div className="mt-3">
                      <span className="inline-flex items-center justify-center px-2 py-1 rounded text-xs font-mono font-medium tracking-wide bg-[var(--desk-surface-subtle)] text-[var(--desk-text-secondary)] border border-[var(--desk-border)]">
                        Payload: {n.payload}
                      </span>
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-end min-w-[100px] shrink-0 pl-4">
                  {!n.read_at ? (
                    <button
                      type="button"
                      className="portal-btn portal-btn--ghost text-xs"
                      onClick={() => void markRead(n.id)}
                    >
                      Mark read
                    </button>
                  ) : (
                    <div className="flex items-center gap-1.5 text-xs font-medium text-[var(--desk-text-tertiary)]">
                      <Icon name="check_circle" size={14} /> Read
                    </div>
                  )}
                </div>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </PageChrome>
    </PageTransition>
  );
}
