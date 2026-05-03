'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import type { PaginationState } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import { buildSupplierReturnResolveIdempotencyKey } from '../_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ReturnItem {
  line_item_id: string;
  order_id: string;
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  status: string;
  retailer_name: string;
  created_at: string;
}

type Resolution = 'WRITE_OFF' | 'RETURN_TO_STOCK';

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

export default function ReturnsPage() {
  const [items, setItems] = useState<ReturnItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [hasMore, setHasMore] = useState(false);
  const [resolvingId, setResolvingId] = useState<string | null>(null);
  const [resolution, setResolution] = useState<Resolution>('RETURN_TO_STOCK');
  const [notes, setNotes] = useState('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const { toast } = useToast();

  const token = useToken();
  const serverOffset = (page - 1) * pageSize;

  const fetchReturns = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(pageSize), offset: String(serverOffset) });
      const res = await fetch(`${API}/v1/supplier/returns?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error('Failed to load returns');
      const json = await res.json();
      setItems(json.data || []);
      setHasMore(Boolean(json.has_more));
    } catch (e) {
      toast((e as Error).message , 'error');
    } finally {
      setLoading(false);
    }
  }, [token, toast, pageSize, serverOffset]);

  useEffect(() => {
    if (!token) return;
    fetchReturns();
  }, [token, fetchReturns]);

  const totalItems = hasMore ? serverOffset + items.length + 1 : serverOffset + items.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const pagination: PaginationState<ReturnItem> = {
    page,
    pageSize,
    totalPages,
    totalItems,
    pageItems: items,
    setPage: (p) => setPage(Math.max(1, p)),
    nextPage: () => {
      if (hasMore) setPage((p) => p + 1);
    },
    prevPage: () => setPage((p) => Math.max(1, p - 1)),
    setPageSize: (size) => {
      setPageSize(size);
      setPage(1);
    },
    canNext: hasMore,
    canPrev: page > 1,
  };

  async function handleResolve(lineItemId: string) {
    setActionLoading(lineItemId);
    try {
      const res = await fetch(`${API}/v1/supplier/returns/resolve`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierReturnResolveIdempotencyKey(lineItemId, resolution, notes),
        },
        body: JSON.stringify({ line_item_id: lineItemId, resolution, notes }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({ error: 'Resolution failed' }));
        toast(errJson.error || 'Resolution failed' , 'error');
        return;
      }
      toast(`Item resolved: ${resolution.replace(/_/g, ' ')}`, 'success');
      setResolvingId(null);
      setNotes('');
      fetchReturns();
    } catch (e) {
      toast((e as Error).message , 'error');
    } finally {
      setActionLoading(null);
    }
  }

  const totalDamageValue = items.reduce((sum, i) => sum + i.quantity * i.unit_price, 0);

  if (!token) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
          Unauthorized — supplier credentials required
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-6 pb-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <h1 className="md-typescale-headline-medium">Dispute & Returns</h1>
        <p className="md-typescale-body-small mt-2" style={{ color: 'var(--muted)' }}>
          Rejected/damaged line items — resolve as write-off or return to stock
        </p>
      </header>

      {/* Summary Card */}
      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="md-card md-card-elevated p-5">
          <p className="md-typescale-label-small uppercase mb-1" style={{ color: 'var(--muted)' }}>Open Returns</p>
          <p className="md-typescale-headline-small font-mono" style={{ color: 'var(--danger)' }}>{items.length}</p>
        </div>
        <div className="md-card md-card-elevated p-5">
          <p className="md-typescale-label-small uppercase mb-1" style={{ color: 'var(--muted)' }}>Total Damage Value</p>
          <p className="md-typescale-headline-small font-mono" style={{ color: 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>{formatAmount(totalDamageValue)}</p>
        </div>
      </div>

      <div className="md-card md-card-elevated p-0 overflow-hidden">
        <div className="px-6 py-4 flex items-center justify-between" style={{ borderBottom: '1px solid var(--border)' }}>
          <span className="md-typescale-label-small uppercase" style={{ color: 'var(--muted)' }}>
            Damaged / Rejected Items
          </span>
          <Button variant="outline" onPress={fetchReturns}>
            Refresh
          </Button>
        </div>

        {loading ? (
          <div className="p-8 text-center md-typescale-body-small animate-pulse" style={{ color: 'var(--muted)' }}>Loading returns...</div>
        ) : items.length === 0 ? (
          <EmptyState
            icon="returns"
            headline="No open returns"
            body="All clear — no rejected or damaged items pending resolution"
            action="Refresh"
            onAction={fetchReturns}
          />
        ) : (
          <>
          <div className="divide-y" style={{ borderColor: 'var(--border)' }}>
            {pagination.pageItems.map(item => (
              <div key={item.line_item_id} className="px-6 py-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-1">
                      <span className="md-typescale-body-small">{item.product_name}</span>
                      <span className="md-chip" style={{ cursor: 'default', background: 'var(--danger)', color: 'var(--danger-foreground)', borderColor: 'transparent' }}>
                        REJECTED
                      </span>
                    </div>
                    <div className="flex items-center gap-4 md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      <span>{item.retailer_name}</span>
                      <span className="font-mono">Qty: {item.quantity}</span>
                      <span className="font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{formatAmount(item.quantity * item.unit_price)}</span>
                      <span className="font-mono md-typescale-label-small">Order: {item.order_id.slice(0, 10)}…</span>
                    </div>
                  </div>

                  <div className="shrink-0">
                    {resolvingId === item.line_item_id ? (
                      <div className="flex items-center gap-2">
                        <select
                          value={resolution}
                          onChange={e => setResolution(e.target.value as Resolution)}
                          className="md-input-outlined text-xs"
                          style={{ color: 'var(--foreground)' }}
                        >
                          <option value="RETURN_TO_STOCK">Return to Stock</option>
                          <option value="WRITE_OFF">Write Off</option>
                        </select>
                        <input
                          type="text"
                          value={notes}
                          onChange={e => setNotes(e.target.value)}
                          placeholder="Notes..."
                          className="w-32 md-input-outlined text-xs"
                          style={{ color: 'var(--foreground)' }}
                        />
                        <Button
                          variant="primary"
                          size="sm"
                          onPress={() => handleResolve(item.line_item_id)}
                          isDisabled={actionLoading === item.line_item_id}
                        >
                          Resolve
                        </Button>
                        <button onClick={() => { setResolvingId(null); setNotes(''); }} className="md-typescale-label-small px-2" style={{ color: 'var(--muted)' }}>×</button>
                      </div>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onPress={() => setResolvingId(item.line_item_id)}
                      >
                        Resolve
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
          <PaginationControls pagination={pagination} />
          </>
        )}
      </div>
    </div>
  );
}
