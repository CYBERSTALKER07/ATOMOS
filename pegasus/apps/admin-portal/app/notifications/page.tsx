'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useNotifications, type Notification } from '@/lib/useNotifications';
import Icon from '@/components/Icon';
import { Button } from '@heroui/react';

type ChannelFilter = 'ALL' | 'PUSH' | 'WS' | 'EMAIL';
type ReadFilter = 'ALL' | 'UNREAD' | 'READ';

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default function NotificationsInboxPage() {
  const { items, unreadCount, loading, markRead, markAllRead, fetchInbox } = useNotifications();
  const [channelFilter, setChannelFilter] = useState<ChannelFilter>('ALL');
  const [readFilter, setReadFilter] = useState<ReadFilter>('ALL');
  const [typeFilter, setTypeFilter] = useState('ALL');

  const types = useMemo(() => {
    const set = new Set(items.map((n) => n.type).filter(Boolean));
    return ['ALL', ...Array.from(set).sort()];
  }, [items]);

  const filtered = useMemo(() => {
    return items.filter((n) => {
      if (channelFilter !== 'ALL' && (n.channel || 'PUSH').toUpperCase() !== channelFilter) return false;
      if (readFilter === 'UNREAD' && n.read_at) return false;
      if (readFilter === 'READ' && !n.read_at) return false;
      if (typeFilter !== 'ALL' && n.type !== typeFilter) return false;
      return true;
    });
  }, [items, channelFilter, readFilter, typeFilter]);

  return (
    <div className="min-h-full p-6 md:p-8 bg-background text-foreground">
      <header className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-medium">Notifications</h1>
          <p className="md-typescale-body-small mt-1 text-muted">
            Inbox with live updates — {unreadCount} unread
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" onPress={() => void fetchInbox()} isDisabled={loading}>
            <Icon name="refresh" size={16} />
            Refresh
          </Button>
          {unreadCount > 0 && (
            <Button variant="primary" onPress={() => void markAllRead()}>
              Mark all read
            </Button>
          )}
        </div>
      </header>

      <div className="flex flex-wrap gap-2 mb-4">
        {(['ALL', 'UNREAD', 'READ'] as ReadFilter[]).map((value) => (
          <button
            key={value}
            onClick={() => setReadFilter(value)}
            className="px-3 py-1.5 rounded-full text-xs font-medium border"
            style={{
              background: readFilter === value ? 'var(--accent)' : 'transparent',
              color: readFilter === value ? 'var(--accent-foreground)' : 'var(--muted)',
              borderColor: readFilter === value ? 'var(--accent)' : 'var(--border)',
            }}
          >
            {value === 'ALL' ? 'All' : value.charAt(0) + value.slice(1).toLowerCase()}
          </button>
        ))}
        {(['ALL', 'PUSH', 'WS', 'EMAIL'] as ChannelFilter[]).map((value) => (
          <button
            key={value}
            onClick={() => setChannelFilter(value)}
            className="px-3 py-1.5 rounded-full text-xs font-medium border"
            style={{
              background: channelFilter === value ? 'var(--accent-soft)' : 'transparent',
              color: channelFilter === value ? 'var(--accent)' : 'var(--muted)',
              borderColor: 'var(--border)',
            }}
          >
            {value}
          </button>
        ))}
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="md-input-outlined text-xs h-8 px-2"
        >
          {types.map((t) => (
            <option key={t} value={t}>{t === 'ALL' ? 'All types' : t.replace(/_/g, ' ')}</option>
          ))}
        </select>
      </div>

      <div className="rounded-xl border border-border overflow-hidden">
        {loading && items.length === 0 ? (
          <div className="p-8 text-center text-muted md-typescale-body-small">Loading inbox…</div>
        ) : filtered.length === 0 ? (
          <div className="p-12 text-center">
            <Icon name="notifications" size={40} className="mx-auto mb-3 opacity-40" />
            <p className="md-typescale-title-small">No notifications match these filters</p>
          </div>
        ) : (
          <ul>
            {filtered.map((n: Notification) => (
              <li
                key={n.id}
                className="border-b border-border last:border-b-0"
                style={!n.read_at ? { background: 'var(--accent-soft)' } : undefined}
              >
                <button
                  type="button"
                  onClick={() => { if (!n.read_at) void markRead(n.id); }}
                  className="w-full text-left px-4 py-3 hover:bg-surface transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <div className="mt-0.5 w-2 h-2 rounded-full shrink-0" style={{ background: n.read_at ? 'transparent' : 'var(--accent)' }} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="md-typescale-label-large">{n.title}</span>
                        <span className="md-typescale-label-small px-2 py-0.5 rounded-full bg-surface text-muted">
                          {n.type.replace(/_/g, ' ')}
                        </span>
                        <span className="md-typescale-label-small text-muted ml-auto">{timeAgo(n.created_at)}</span>
                      </div>
                      {n.body && (
                        <p className="md-typescale-body-small text-muted mt-1">{n.body}</p>
                      )}
                      <div className="mt-2 flex items-center gap-3 text-xs text-muted">
                        <span>{(n.channel || 'PUSH').toUpperCase()}</span>
                        {!n.read_at && (
                          <span className="text-accent">Unread</span>
                        )}
                      </div>
                    </div>
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <p className="mt-4 md-typescale-label-small text-muted">
        Tip: use the bell in the top bar for quick peek, or{' '}
        <Link href="/supplier/orders" className="text-accent underline">open orders</Link>{' '}
        when dispatch alerts arrive.
      </p>
    </div>
  );
}
