'use client';

import { useState, useCallback, useEffect } from 'react';
import { apiFetch, apiFetchNoQueue } from '@/lib/auth';
import { usePagination } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import { Button } from '@heroui/react';

/* ─── Types ───────────────────────────────────────────────── */

interface SettlementRow {
  order_id: string;
  invoice_id: string;
  retailer_id: string;
  amount: number;
  currency?: string;
  fee_amount?: number;
  net_payout_amount?: number;
  payout_owner_type?: string;
  payout_owner_id?: string;
  fee_policy_version?: string;
  settlement_target?: string;
  payment_mode: string;
  invoice_status: string;
  paid_at: string | null;
  created_at: string;
}

interface SettlementSummary {
  total_paid: number;
  total_pending: number;
  paid_count: number;
  pending_count: number;
  period_from: string;
  period_to: string;
}

interface SettlementReport {
  summary: SettlementSummary;
  rows: SettlementRow[];
}

type StatusFilter = '' | 'PAID' | 'PENDING';

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

function shortId(id: string): string {
  return id.length > 12 ? id.slice(0, 12) + '…' : id;
}

function buildBatchSettlementIdempotencyKey(invoiceIds: string[], reference: string): string {
  return ['treasury-batch-settle', [...invoiceIds].map((id) => id.trim()).sort().join(','), reference.trim()].join(':');
}

function buildInvoiceStatusOverrideIdempotencyKey(invoiceId: string, nextStatus: string, reason: string): string {
  return ['treasury-invoice-status', invoiceId.trim(), nextStatus.trim().toUpperCase(), reason.trim()].join(':');
}

function thirtyDaysAgo(): string {
  const d = new Date();
  d.setDate(d.getDate() - 30);
  return d.toISOString().slice(0, 10);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function SettlementPage() {
  const { toast } = useToast();

  const [from, setFrom] = useState(thirtyDaysAgo);
  const [to, setTo] = useState(todayISO);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [selectedInvoiceIds, setSelectedInvoiceIds] = useState<string[]>([]);
  const [report, setReport] = useState<SettlementReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [settling, setSettling] = useState(false);
  const [overridingInvoiceId, setOverridingInvoiceId] = useState<string | null>(null);

  const pagination = usePagination(report?.rows || [], 25);

  /* ─── Fetch ─────────────────────────────────────────────── */

  const fetchReport = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ from, to });
      if (statusFilter) params.set('status', statusFilter);
      const res = await apiFetch(`/v1/supplier/settlement-report?${params}`);
      if (!res.ok) throw new Error('Failed to load settlement report');
      setReport(await res.json());
      setSelectedInvoiceIds([]);
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [from, to, statusFilter, toast]);

  const toggleInvoiceSelection = useCallback((invoiceId: string) => {
    setSelectedInvoiceIds((prev) =>
      prev.includes(invoiceId) ? prev.filter((id) => id !== invoiceId) : [...prev, invoiceId],
    );
  }, []);

  const batchSettleSelected = useCallback(async () => {
    if (selectedInvoiceIds.length === 0) return;
    const reference = window.prompt('Settlement reference (bank batch ID):', 'MANUAL-BATCH');
    if (!reference) return;

    setSettling(true);
    try {
      const res = await apiFetchNoQueue('/v1/treasury/batch-settle', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildBatchSettlementIdempotencyKey(selectedInvoiceIds, reference),
        },
        body: JSON.stringify({ invoice_ids: selectedInvoiceIds, reference }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast(`Settled ${selectedInvoiceIds.length} invoices`, 'success');
      await fetchReport();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setSettling(false);
    }
  }, [fetchReport, selectedInvoiceIds, toast]);

  const overrideInvoiceStatus = useCallback(async (invoiceId: string, nextStatus: string) => {
    const reason = window.prompt(`Reason for setting ${invoiceId} to ${nextStatus}:`, 'Manual override');
    if (!reason) return;

    setOverridingInvoiceId(invoiceId);
    try {
      const res = await apiFetchNoQueue('/v1/treasury/invoice/status', {
        method: 'PATCH',
        headers: {
          'Idempotency-Key': buildInvoiceStatusOverrideIdempotencyKey(invoiceId, nextStatus, reason),
        },
        body: JSON.stringify({ invoice_id: invoiceId, status: nextStatus, reason }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast(`Invoice ${invoiceId} set to ${nextStatus}`, 'success');
      await fetchReport();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setOverridingInvoiceId(null);
    }
  }, [fetchReport, toast]);

  useEffect(() => {
    fetchReport();
  }, [fetchReport]);

  /* ─── Render ────────────────────────────────────────────── */

  return (
    <div className="min-h-full w-full max-w-7xl mx-auto px-4 py-6 flex flex-col gap-6" style={{ background: 'var(--desk-bg)' }}>
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--desk-text-primary)' }}>
            Settlement Report
          </h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--desk-text-secondary)' }}>
            Invoice settlement status and payout reconciliation.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="md-input-outlined px-3 py-2 md-typescale-body-medium md-shape-sm"
            style={{ background: 'var(--desk-surface)', color: 'var(--desk-text-primary)', borderColor: 'var(--desk-border)' }}
          />
          <span className="md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>to</span>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="md-input-outlined px-3 py-2 md-typescale-body-medium md-shape-sm"
            style={{ background: 'var(--desk-surface)', color: 'var(--desk-text-primary)', borderColor: 'var(--desk-border)' }}
          />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="md-input-outlined px-3 py-2 md-typescale-body-medium md-shape-sm"
            style={{ background: 'var(--desk-surface)', color: 'var(--desk-text-primary)', borderColor: 'var(--desk-border)' }}
          >
            <option value="">All statuses</option>
            <option value="PAID">Paid</option>
            <option value="PENDING">Pending</option>
          </select>
          <Button
            variant="outline"
            onPress={fetchReport}
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
          >
            <Icon name="refresh" size={16} className="mr-1.5" />
            Refresh
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <span className="md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>
            Selected: {selectedInvoiceIds.length}
          </span>
          <Button
            variant="primary"
            isDisabled={selectedInvoiceIds.length === 0 || settling}
            onPress={batchSettleSelected}
          >
            {settling ? 'Settling...' : 'Settle Selected'}
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      {!loading && report && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            { label: 'Total Paid', value: formatAmount(report.summary.total_paid), color: 'var(--desk-text-primary)' },
            { label: 'Total Pending', value: formatAmount(report.summary.total_pending), color: 'var(--desk-warning)' },
            { label: 'Paid Invoices', value: report.summary.paid_count.toString(), color: 'var(--desk-text-primary)' },
            { label: 'Pending Invoices', value: report.summary.pending_count.toString(), color: 'var(--desk-warning)' },
          ].map((s) => (
            <div
              key={s.label}
              className="desk-card p-4"
              style={{ background: 'var(--desk-surface)' }}
            >
              <span className="md-typescale-label-small block" style={{ color: 'var(--desk-text-secondary)' }}>
                {s.label}
              </span>
              <span className="md-typescale-headline-small tabular-nums" style={{ color: s.color }}>
                {s.value}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="flex flex-col gap-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Table */}
      {!loading && report && (
        <>
          {report.rows.length === 0 ? (
            <EmptyState
              icon="treasury"
              headline="No settlement records"
              body="No invoices found for the selected period and filter."
            />
          ) : (
            <>
              <div className="desk-card overflow-hidden" style={{ background: 'var(--desk-surface)' }}>
                <table className="md-table">
                  <thead>
                    <tr
                      className="border-b"
                      style={{ borderColor: 'var(--desk-border)', color: 'var(--desk-text-secondary)' }}
                    >
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Order</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Invoice</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Retailer</th>
                      <th className="text-right px-4 py-3 md-typescale-label-medium">Amount</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Gateway</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Status</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Paid At</th>
                      <th className="text-left px-4 py-3 md-typescale-label-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pagination.pageItems.map((r) => (
                      <tr
                        key={r.invoice_id}
                        className="border-b last:border-b-0"
                        style={{ borderColor: 'var(--desk-border)', color: 'var(--desk-text-primary)' }}
                      >
                        <td className="px-4 py-3 font-mono text-xs">{shortId(r.order_id)}</td>
                        <td className="px-4 py-3 font-mono text-xs">{shortId(r.invoice_id)}</td>
                        <td className="px-4 py-3 text-xs">{shortId(r.retailer_id)}</td>
                        <td className="px-4 py-3 text-right tabular-nums font-medium">{formatAmount(r.amount)}</td>
                        <td className="px-4 py-3 text-right text-[11px]" style={{ color: 'var(--desk-text-secondary)' }}>
                          {(r.currency || 'UZS').toUpperCase()}
                          {typeof r.fee_amount === 'number' ? ` · Fee ${formatAmount(r.fee_amount)}` : ''}
                          {typeof r.net_payout_amount === 'number' ? ` · Net ${formatAmount(r.net_payout_amount)}` : ''}
                        </td>
                        <td className="px-4 py-3 text-xs">
                          <div>{r.payment_mode}</div>
                          <div style={{ color: 'var(--desk-text-secondary)' }}>
                            {(r.settlement_target || '—')} · {(r.fee_policy_version || '—')}
                          </div>
                          <div style={{ color: 'var(--desk-text-secondary)' }}>
                            Owner {(r.payout_owner_type || 'SUPPLIER')}:{r.payout_owner_id ? shortId(r.payout_owner_id) : '—'}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className="md-shape-full px-2 py-0.5 text-xs font-medium inline-block"
                            style={{
                              background:
                                r.invoice_status === 'SETTLED'
                                  ? 'var(--desk-success-soft)'
                                  : 'var(--desk-warning-soft)',
                              color:
                                r.invoice_status === 'SETTLED'
                                  ? 'var(--desk-success)'
                                  : 'var(--desk-warning)',
                            }}
                          >
                            {r.invoice_status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-xs tabular-nums" style={{ color: 'var(--desk-text-secondary)' }}>
                          {r.paid_at ? new Date(r.paid_at).toLocaleString() : '—'}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <input
                              type="checkbox"
                              checked={selectedInvoiceIds.includes(r.invoice_id)}
                              onChange={() => toggleInvoiceSelection(r.invoice_id)}
                              aria-label={`Select ${r.invoice_id}`}
                            />
                            <select
                              className="md-input-outlined px-2 py-1 text-xs"
                              disabled={overridingInvoiceId === r.invoice_id}
                              value=""
                              onChange={(e) => {
                                const next = e.target.value;
                                if (!next) return;
                                void overrideInvoiceStatus(r.invoice_id, next);
                                e.currentTarget.value = '';
                              }}
                            >
                              <option value="">Override</option>
                              <option value="SETTLED">SETTLED</option>
                              <option value="PENDING">PENDING</option>
                              <option value="DISPUTED">DISPUTED</option>
                              <option value="WRITTEN_OFF">WRITTEN_OFF</option>
                            </select>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <PaginationControls pagination={pagination} />
            </>
          )}
        </>
      )}
    </div>
  );
}
