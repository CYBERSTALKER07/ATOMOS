'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { warehouseInboundConfirmKey, warehouseInboundScanKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

import { ReturnsList, type InboundRow, isClaimTicket } from '@/components/returns/ReturnsList';
import { ReverseLogisticsPanel } from '@/components/returns/ReverseLogisticsPanel';

export default function ReturnsPage() {
  const t = usePortalT();
  const [rows, setRows] = useState<InboundRow[]>([]);
  const [history, setHistory] = useState<InboundRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'inbound' | 'history'>('inbound');
  const [barcode, setBarcode] = useState('');
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string | null>(null);

  const loadInbound = useCallback(async () => {
    // OPEN = PENDING|ON_TRUCK|ARRIVED|RECEIVING — includes claim reverse-logistics tickets.
    const res = await apiFetch('/v1/returns/inbound?physical_status=OPEN&limit=100');
    if (!res.ok) throw new Error('load_inbound_failed');
    const data = await res.json();
    setRows(data.data ?? []);
  }, []);

  const loadHistory = useCallback(async () => {
    const res = await apiFetch('/v1/returns/history?limit=50');
    if (!res.ok) throw new Error('load_history_failed');
    const data = await res.json();
    setHistory(data.data ?? []);
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (tab === 'inbound') await loadInbound();
      else await loadHistory();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setLoading(false);
    }
  }, [tab, loadInbound, loadHistory]);

  useEffect(() => { void load(); }, [load]);

  useWarehouseSessionReconcile(() => {
    void load();
  });

  async function ensureSession() {
    if (sessionId) return sessionId;
    const res = await apiFetch('/v1/returns/inbound/sessions', { method: 'POST', body: '{}' });
    if (!res.ok) throw new Error('session_failed');
    const data = await res.json();
    setSessionId(data.session_id);
    return data.session_id as string;
  }

  async function handleScan() {
    if (!barcode.trim()) return;
    try {
      const sid = await ensureSession();
      const trimmed = barcode.trim();
      const warehouseId = warehouseHomeNodeId() || 'warehouse';
      const res = await apiFetch('/v1/returns/inbound/scan', {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseInboundScanKey(warehouseId, trimmed, sid),
        },
        body: JSON.stringify({ barcode: trimmed, qty: 1, session_id: sid }),
      });
      const body = await res.json();
      if (!res.ok) throw new Error(body.error || body.message || 'scan_failed');
      setBarcode('');
      await loadInbound();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'scan_failed');
    }
  }

  async function handleConfirm(disposition: 'RESTOCK' | 'WRITE_OFF') {
    const targets = rows.filter(r => selected.has(r.return_id));
    if (targets.length === 0) {
      setError('Select at least one line');
      return;
    }
    try {
      const sid = await ensureSession();
      const warehouseId = warehouseHomeNodeId() || 'warehouse';
      const returnIds = targets.map(r => r.return_id);
      const res = await apiFetch('/v1/returns/inbound/confirm', {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseInboundConfirmKey(warehouseId, returnIds, disposition),
        },
        body: JSON.stringify({
          session_id: sid,
          lines: targets.map(r => ({
            return_id: r.return_id,
            disposition,
            qty: r.received_qty > 0 ? r.received_qty : r.expected_qty,
          })),
        }),
      });
      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.error || 'confirm_failed');
      }
      setSelected(new Set());
      await loadInbound();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'confirm_failed');
    }
  }

  const list = tab === 'inbound' ? rows : history;

  return (
    <PageTransition>
      <PageChrome
        icon="returns"
        title={t("warehouse_portal.returns.text.inbound_returns")}
        description={t("warehouse_portal.residual.text.dock_queue_for_truck_returns_and_claim_reverse_logistics_tickets")}
        actions={
          <button type="button" onClick={() => void load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <div className="flex gap-2 mb-4">
          <button type="button" className={`px-3 py-1.5 rounded-lg text-sm ${tab === 'inbound' ? 'button--primary' : 'button--secondary'}`} onClick={() => setTab('inbound')}>{t("warehouse_portal.returns.text.gate_queue")}</button>
          <button type="button" className={`px-3 py-1.5 rounded-lg text-sm ${tab === 'history' ? 'button--primary' : 'button--secondary'}`} onClick={() => setTab('history')}>{t("warehouse_portal.returns.text.history")}</button>
        </div>

        {tab === 'inbound' && (
          <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-end">
            {/* USB/BT wedge scanners type into the field and send Enter — no camera lib needed. */}
            <label className="flex-1 text-sm">
              <span className="text-[var(--muted)]">{t("warehouse_portal.returns.text.barcode_ean")}</span>
              <input
                autoFocus
                className="mt-1 w-full rounded-lg border border-[var(--border)] bg-[var(--surface)] px-3 py-2"
                value={barcode}
                onChange={e => setBarcode(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    void handleScan();
                  }
                }}
                placeholder={t("warehouse_portal.returns.text.scan_or_type_wedge_scanner_enter")}
              />
            </label>
            <button type="button" className="button--primary px-4 py-2 rounded-lg text-sm" onClick={() => void handleScan()}>{t("warehouse_portal.returns.text.scan")}</button>
            <button type="button" className="button--primary px-4 py-2 rounded-lg text-sm bg-emerald-600" onClick={() => void handleConfirm('RESTOCK')}>{t("warehouse_portal.returns.text.restock_selected")}</button>
            <button type="button" className="button--secondary px-4 py-2 rounded-lg text-sm" onClick={() => void handleConfirm('WRITE_OFF')}>{t("warehouse_portal.returns.text.write_off")}</button>
          </div>
        )}

        {error && <p className="text-sm text-red-500 mb-3">{error}</p>}

        <ReturnsList
          tab={tab}
          loading={loading}
          list={list}
          selected={selected}
          onToggleSelect={(id) => setSelected(prev => {
            const n = new Set(prev);
            if (n.has(id)) n.delete(id);
            else n.add(id);
            return n;
          })}
        />
        <section className="mt-8 p-4 border rounded-lg bg-white">
          <h2 className="text-lg font-semibold mb-2">{t("warehouse_portal.returns.text.credit_note_reverse_logistics")}</h2>
          <ReverseLogisticsPanel />
        </section>
      </PageChrome>
    </PageTransition>
  );
}
