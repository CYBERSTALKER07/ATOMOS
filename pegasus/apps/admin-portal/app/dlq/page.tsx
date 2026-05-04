"use client";

import { useState, useCallback } from "react";
import { Button } from '@heroui/react';
import { apiFetch, apiFetchNoQueue } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
import Dialog from '@/components/Dialog';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';

// ─── Type Definitions ─────────────────────────────────────────────────────────

interface DLQPayload {
  failed_at?: string;
  reason?: string;
  event_data?: {
    order_id?: string;
    event_type?: string;
    amount?: number;
    retailer_id?: string;
  };
}

interface DLQMessage {
  offset: number;
  key: string;
  timestamp: string;
  payload: DLQPayload;
}

// ─── DLQ Admin Console ────────────────────────────────────────────────────────

export default function DLQPage() {
  const [messages, setMessages] = useState<DLQMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [hasMore, setHasMore] = useState(false);
  const [replayingOffset, setReplayingOffset] = useState<number | null>(null);
  const [pendingReplay, setPendingReplay] = useState<{ offset: number; orderID: string } | null>(null);
  const { toast } = useToast();

  const offset = (page - 1) * pageSize;
  const canPrev = page > 1;
  const canNext = hasMore;

  const fetchDLQ = useCallback(async (signal?: AbortSignal) => {
    try {
      const res = await apiFetch(`/v1/admin/dlq?limit=${pageSize + 1}&offset=${offset}`, { signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: DLQMessage[] = await res.json();
      const rows = data ?? [];
      setHasMore(rows.length > pageSize);
      setMessages(rows.slice(0, pageSize));
      setLastRefreshed(new Date());
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      console.error("DLQ fetch failed:", err);
      setMessages([]);
      setHasMore(false);
    } finally {
      setLoading(false);
    }
  }, [offset, pageSize]);

  // Poll every 5 seconds — DLQ entries are critical assets; check them often
  usePolling((signal) => fetchDLQ(signal), 5000);

  const replayMessage = async (offset: number) => {
    setReplayingOffset(offset);
    try {
      const res = await apiFetchNoQueue('/v1/admin/dlq/replay', {
        method: "POST",
        body: JSON.stringify({ offset }),
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText);
      }
      toast(`OFFSET ${offset} RE-INJECTED → MAIN TOPIC`, 'success');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Unknown error";
      toast(`REPLAY FAULT: ${message}`, 'error');
    } finally {
      setReplayingOffset(null);
    }
  };

  return (
    <div className="min-h-full" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <div className="px-6 md:px-10 py-6 flex items-center justify-between" style={{ borderBottom: '1px solid var(--border)' }}>
        <div>
          <p className="md-typescale-label-small mb-1" style={{ color: 'var(--muted)' }}>
            Admin Console / Financial Safety
          </p>
          <h1 className="md-typescale-title-large">Dead Letter Queue</h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Trapped</p>
            <p className="md-typescale-headline-small tabular-nums">{messages.length}</p>
          </div>          {lastRefreshed && (
            <span className="md-typescale-label-small" style={{ color: 'var(--border)' }}>
              Updated {lastRefreshed.toLocaleTimeString()}
            </span>
          )}          <Button
            variant="outline"
            onPress={() => fetchDLQ()}
          >
            Refresh
          </Button>
        </div>
      </div>

      {/* Info bar */}
      <div className="px-6 md:px-10 py-3" style={{ borderBottom: '1px solid var(--border)', background: 'var(--surface)' }}>
        <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
          These events failed to reach the{" "}
          <code className="px-1 md-shape-xs" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>
            pegasus-logistics-events
          </code>{" "}
          main topic. Each row represents a dropped payment transaction. Use{" "}
          <strong>Replay</strong> to re-inject onto the main topic for reconciliation.
        </p>
      </div>

      {/* Table */}
      <div className="px-6 md:px-10 py-6">
        {loading ? (
          <div className="md-card md-card-outlined p-0 overflow-hidden">
            <table className="md-table">
              <thead>
                <tr>
                  <th className="w-16">Offset</th>
                  <th>Order ID</th>
                  <th>Event Type</th>
                  <th>Amount</th>
                  <th>Failed At</th>
                  <th className="w-48">Reason</th>
                  <th className="text-right w-24">Action</th>
                </tr>
              </thead>
              <tbody>
                {Array.from({ length: 5 }).map((_, i) => (
                  <tr key={`skel-${i}`}>
                    <td><div className="w-10 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-20 h-6 rounded-full animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-32 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-36 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-16 h-8 rounded-full animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : messages.length === 0 ? (
          <EmptyState
            icon="dlq"
            headline="Queue Clear"
            body="No trapped transactions — Financial integrity intact."
          />
        ) : (
          <div className="md-card md-card-outlined p-0 overflow-hidden">
            <table className="md-table">
              <thead>
                <tr>
                  <th className="w-16">Offset</th>
                  <th>Order ID</th>
                  <th>Event Type</th>
                  <th>Amount</th>
                  <th>Failed At</th>
                  <th className="w-48">Reason</th>
                  <th className="text-right w-24">Action</th>
                </tr>
              </thead>
              <tbody>
                {messages.map((msg) => (
                  <tr key={msg.offset} className="transition-colors">
                    <td className="font-mono md-typescale-label-small" style={{ color: 'var(--border)' }}>{msg.offset}</td>
                    <td className="font-mono md-typescale-body-small">
                      {msg.payload?.event_data?.order_id ?? msg.key ?? "—"}
                    </td>
                    <td>
                      <span className="md-chip" style={{ cursor: 'default', background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)', borderColor: 'transparent' }}>
                        {msg.payload?.event_data?.event_type ?? "UNKNOWN"}
                      </span>
                    </td>
                    <td className="font-mono md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                      {msg.payload?.event_data?.amount != null
                        ? `${msg.payload.event_data.amount.toLocaleString()}`
                        : "—"}
                    </td>
                    <td className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      {msg.payload?.failed_at
                        ? new Date(msg.payload.failed_at).toLocaleString()
                        : msg.timestamp
                        ? new Date(msg.timestamp).toLocaleString()
                        : "—"}
                    </td>
                    <td className="md-typescale-label-small max-w-[180px] truncate" style={{ color: 'var(--danger)' }}>
                      {msg.payload?.reason ?? "—"}
                    </td>
                    <td className="text-right">
                      <Button
                        variant="secondary"
                        onPress={() => setPendingReplay({ offset: msg.offset, orderID: msg.payload?.event_data?.order_id ?? msg.key ?? '—' })}
                        isDisabled={replayingOffset === msg.offset}
                      >
                        {replayingOffset === msg.offset ? "Firing..." : "Replay"}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
              <div className="flex items-center gap-2">
                <label className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Rows</label>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value));
                    setPage(1);
                  }}
                  className="md-typescale-label-small px-2 py-1 rounded-md"
                  style={{ border: '1px solid var(--border)', background: 'var(--surface)', color: 'var(--foreground)' }}
                >
                  {[10, 25, 50, 100].map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
              <div className="flex items-center gap-2">
                <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                  Page {page}
                </span>
                <Button variant="ghost" onPress={() => setPage(1)} isDisabled={!canPrev}>First</Button>
                <Button variant="ghost" onPress={() => setPage((p) => Math.max(1, p - 1))} isDisabled={!canPrev}>Prev</Button>
                <Button variant="ghost" onPress={() => setPage((p) => p + 1)} isDisabled={!canNext}>Next</Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Replay Confirmation Dialog */}
      <Dialog
        open={!!pendingReplay}
        onClose={() => setPendingReplay(null)}
        title="Confirm Replay"
        actions={
          <>
            <Button variant="outline" onPress={() => setPendingReplay(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onPress={() => { if (pendingReplay) { replayMessage(pendingReplay.offset); setPendingReplay(null); } }}
            >
              Replay
            </Button>
          </>
        }
      >
        <p className="md-typescale-body-medium mb-2" style={{ color: 'var(--muted)' }}>
          Re-inject offset <strong>{pendingReplay?.offset}</strong> onto the main topic?
        </p>
        <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
          Order: <span className="font-mono">{pendingReplay?.orderID}</span>
        </p>
      </Dialog>

      {/* Footer */}
      <div className="px-6 md:px-10 py-3 mt-8" style={{ borderTop: '1px solid var(--border)' }}>
        <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>
          Topic: <code>pegasus-logistics-events-dlq</code> · Auto-refresh every 5s · Replay re-emits onto <code>pegasus-logistics-events</code>
        </p>
      </div>
    </div>
  );
}
