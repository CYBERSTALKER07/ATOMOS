'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import type { PaginationState } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import { buildSupplierInventoryAdjustIdempotencyKey } from '../_shared/idempotency';

export const dynamic = 'force-dynamic';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface InventoryItem {
  product_id: string;
  sku_id: string;
  product_name: string;
  quantity_available: number;
  updated_at: string;
}

interface AuditEntry {
  audit_id: string;
  product_id: string;
  product_name: string;
  adjusted_by: string;
  previous_qty: number;
  new_qty: number;
  delta: number;
  reason: string;
  adjusted_at: string;
}

type Tab = 'stock' | 'audit';
type Reason = 'PRODUCTION_RECEIPT' | 'DAMAGE_WRITEOFF' | 'CORRECTION' | 'RETURN_TO_STOCK';

const REASONS: { value: Reason; label: string }[] = [
  { value: 'PRODUCTION_RECEIPT', label: 'Production Receipt' },
  { value: 'DAMAGE_WRITEOFF', label: 'Damage Write-off' },
  { value: 'CORRECTION', label: 'Manual Correction' },
  { value: 'RETURN_TO_STOCK', label: 'Return to Stock' },
];

export default function InventoryPage() {
  const [tab, setTab] = useState<Tab>('stock');
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [audit, setAudit] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [stockPage, setStockPage] = useState(1);
  const [stockPageSize, setStockPageSize] = useState(25);
  const [stockHasMore, setStockHasMore] = useState(false);
  const [auditPage, setAuditPage] = useState(1);
  const [auditPageSize, setAuditPageSize] = useState(25);
  const [auditHasMore, setAuditHasMore] = useState(false);
  const [adjusting, setAdjusting] = useState<string | null>(null);
  const [adjustDelta, setAdjustDelta] = useState('');
  const [adjustReason, setAdjustReason] = useState<Reason>('PRODUCTION_RECEIPT');
  const { toast } = useToast();

  const token = useToken();
  const stockOffset = (stockPage - 1) * stockPageSize;
  const auditOffset = (auditPage - 1) * auditPageSize;

  const fetchInventory = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(stockPageSize), offset: String(stockOffset) });
      const res = await fetch(`${API}/v1/supplier/inventory?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error('Failed to load inventory');
      const json = await res.json();
      setItems(json.data || []);
      setStockHasMore(Boolean(json.has_more));
    } catch (e) {
      toast((e as Error).message , 'error');
    } finally {
      setLoading(false);
    }
  }, [token, stockPageSize, stockOffset, toast]);

  const fetchAudit = useCallback(async () => {
    try {
      const params = new URLSearchParams({ limit: String(auditPageSize), offset: String(auditOffset) });
      const res = await fetch(`${API}/v1/supplier/inventory/audit?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return;
      const json = await res.json();
      setAudit(json.data || []);
      setAuditHasMore(Boolean(json.has_more));
    } catch { /* silent */ }
  }, [token, auditPageSize, auditOffset]);

  useEffect(() => {
    if (!token) return;
    fetchInventory();
    fetchAudit();
  }, [token, fetchInventory, fetchAudit]);

  const stockTotalItems = stockHasMore ? stockOffset + items.length + 1 : stockOffset + items.length;
  const stockTotalPages = Math.max(1, Math.ceil(stockTotalItems / stockPageSize));
  const stockPagination: PaginationState<InventoryItem> = {
    page: stockPage,
    pageSize: stockPageSize,
    totalPages: stockTotalPages,
    totalItems: stockTotalItems,
    pageItems: items,
    setPage: (p) => setStockPage(Math.max(1, p)),
    nextPage: () => {
      if (stockHasMore) setStockPage((p) => p + 1);
    },
    prevPage: () => setStockPage((p) => Math.max(1, p - 1)),
    setPageSize: (size) => {
      setStockPageSize(size);
      setStockPage(1);
    },
    canNext: stockHasMore,
    canPrev: stockPage > 1,
  };

  const auditTotalItems = auditHasMore ? auditOffset + audit.length + 1 : auditOffset + audit.length;
  const auditTotalPages = Math.max(1, Math.ceil(auditTotalItems / auditPageSize));
  const auditPagination: PaginationState<AuditEntry> = {
    page: auditPage,
    pageSize: auditPageSize,
    totalPages: auditTotalPages,
    totalItems: auditTotalItems,
    pageItems: audit,
    setPage: (p) => setAuditPage(Math.max(1, p)),
    nextPage: () => {
      if (auditHasMore) setAuditPage((p) => p + 1);
    },
    prevPage: () => setAuditPage((p) => Math.max(1, p - 1)),
    setPageSize: (size) => {
      setAuditPageSize(size);
      setAuditPage(1);
    },
    canNext: auditHasMore,
    canPrev: auditPage > 1,
  };

  async function handleAdjust(productId: string) {
    const delta = parseInt(adjustDelta, 10);
    if (isNaN(delta) || delta === 0) {
      toast('Adjustment must be a non-zero integer' , 'error');
      return;
    }
    try {
      const res = await fetch(`${API}/v1/supplier/inventory`, {
        method: 'PATCH',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierInventoryAdjustIdempotencyKey(productId, delta, adjustReason),
        },
        body: JSON.stringify({ product_id: productId, adjustment: delta, reason: adjustReason }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({ error: 'Adjustment failed' }));
        toast(errJson.error || 'Adjustment failed' , 'error');
        return;
      }
      toast(`Stock adjusted by ${delta > 0 ? '+' : ''}${delta}`, 'success');
      setAdjusting(null);
      setAdjustDelta('');
      fetchInventory();
      fetchAudit();
    } catch (e) {
      toast((e as Error).message , 'error');
    }
  }

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
        <h1 className="md-typescale-headline-medium">Inventory Management</h1>
        <p className="md-typescale-body-small mt-2" style={{ color: 'var(--muted)' }}>
          Stock levels, replenishment controls, and audit trail
        </p>
      </header>

      {/* Tab Switcher */}
      <div className="flex gap-1 mb-6 p-1 md-shape-lg" style={{ background: 'var(--surface)' }}>
        {(['stock', 'audit'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={tab === t ? "md-chip md-chip-selected flex-1 py-2.5" : "md-chip flex-1 py-2.5"}
          >
            {t === 'stock' ? 'Stock Levels' : 'Audit Log'}
          </button>
        ))}
      </div>

      {tab === 'stock' && (
        <div className="md-card md-card-elevated p-0 overflow-hidden">
          <div className="px-6 py-4 flex items-center justify-between" style={{ borderBottom: '1px solid var(--border)' }}>
            <span className="md-typescale-label-small uppercase" style={{ color: 'var(--muted)' }}>
              {items.length} SKUs
            </span>
            <Button variant="outline" onPress={fetchInventory}>
              Refresh
            </Button>
          </div>
          {loading ? (
            <div className="p-8 text-center md-typescale-body-small animate-pulse" style={{ color: 'var(--muted)' }}>Loading inventory...</div>
          ) : items.length === 0 ? (
            <EmptyState
              icon="inventory"
              headline="No inventory records"
              body="Stock levels will appear here once products are added to the catalog."
              action="Refresh"
              onAction={fetchInventory}
            />
          ) : (
            <>
            <table className="md-table">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>SKU</th>
                  <th className="text-right">Stock</th>
                  <th className="text-right">Action</th>
                </tr>
              </thead>
              <tbody>
                {stockPagination.pageItems.map(item => (
                  <tr key={item.product_id}>
                    <td>{item.product_name}</td>
                    <td className="font-mono md-typescale-body-small" style={{ color: 'var(--muted)' }}>{item.sku_id}</td>
                    <td className="text-right font-mono" style={{
                      color: item.quantity_available <= 10 ? 'var(--danger)' : 'var(--foreground)', fontVariantNumeric: 'tabular-nums',
                    }}>
                      {item.quantity_available}
                    </td>
                    <td className="text-right">
                      {adjusting === item.product_id ? (
                        <div className="flex items-center gap-2 justify-end">
                          <input
                            type="number"
                            value={adjustDelta}
                            onChange={e => setAdjustDelta(e.target.value)}
                            placeholder="+/-"
                            className="w-20 md-input-outlined font-mono text-xs"
                          />
                          <select
                            value={adjustReason}
                            onChange={e => setAdjustReason(e.target.value as Reason)}
                            className="md-input-outlined text-xs"
                            style={{ color: 'var(--foreground)' }}
                          >
                            {REASONS.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
                          </select>
                          <Button
                            variant="primary"
                            size="sm"
                            onPress={() => handleAdjust(item.product_id)}
                          >
                            Apply
                          </Button>
                          <button onClick={() => { setAdjusting(null); setAdjustDelta(''); }} className="md-typescale-label-small px-2 py-1.5" style={{ color: 'var(--muted)' }}>
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onPress={() => setAdjusting(item.product_id)}
                        >
                          Adjust
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <PaginationControls pagination={stockPagination} />
            </>
          )}
        </div>
      )}

      {tab === 'audit' && (
        <div className="md-card md-card-elevated p-0 overflow-hidden">
          <div className="px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
            <span className="md-typescale-label-small uppercase" style={{ color: 'var(--muted)' }}>
              Last 100 adjustments
            </span>
          </div>
          {audit.length === 0 ? (
            <EmptyState icon="ledger" headline="No audit records" body="Adjustments will be logged here." />
          ) : (
            <>
            <table className="md-table">
              <thead>
                <tr>
                  <th>Product</th>
                  <th className="text-right">Prev</th>
                  <th className="text-right">Delta</th>
                  <th className="text-right">New</th>
                  <th>Reason</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {auditPagination.pageItems.map(a => (
                  <tr key={a.audit_id}>
                    <td>{a.product_name}</td>
                    <td className="text-right font-mono md-typescale-body-small" style={{ color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>{a.previous_qty}</td>
                    <td className="text-right font-mono md-typescale-body-small" style={{ color: a.delta > 0 ? 'var(--success)' : 'var(--danger)', fontVariantNumeric: 'tabular-nums' }}>
                      {a.delta > 0 ? '+' : ''}{a.delta}
                    </td>
                    <td className="text-right font-mono md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{a.new_qty}</td>
                    <td>
                      <span className="md-chip" style={{ cursor: 'default', background: 'var(--surface)', color: 'var(--muted)', borderColor: 'transparent' }}>
                        {a.reason.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="font-mono md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      {a.adjusted_at ? new Date(a.adjusted_at).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <PaginationControls pagination={auditPagination} />
            </>
          )}
        </div>
      )}
    </div>
  );
}
